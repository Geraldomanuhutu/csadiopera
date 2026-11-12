// Package service berisi use case CSA: siapa boleh melakukan apa, dan urutan
// baca, cek aturan domain, lalu simpan.
package service

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"opera/csa/internal/domain"
)

// Store adalah port penyimpanan. Implementasi database OPERA cukup memenuhi interface ini.
// Catatan untuk implementasi database: SaveKertasKerja sebaiknya memakai
// optimistic locking (kolom versi_baris) supaya dua orang tidak saling menimpa.
type Store interface {
	GetUser(id string) (domain.User, bool)
	SaveProses(domain.ProsesBisnis)
	ListProses() []domain.ProsesBisnis
	SaveKontrol(domain.Kontrol)
	GetKontrol(ref string) (domain.Kontrol, bool)
	ListKontrol() []domain.Kontrol
	SaveTemplate(domain.TemplateKontrol)
	LatestTemplate(ref string) (domain.TemplateKontrol, bool)
	SaveAssessment(domain.Assessment)
	GetAssessment(id string) (domain.Assessment, bool)
	ListAssessments() []domain.Assessment
	SaveKertasKerja(domain.KertasKerja)
	GetKertasKerja(id string) (domain.KertasKerja, bool)
	ListKertasKerja() []domain.KertasKerja
}

// Notifier mengirim notifikasi in-app/email. Default tidak melakukan apa-apa.
type Notifier interface {
	Kirim(keUnitAtauPeran, judul, isi string)
}

type nopNotifier struct{}

func (nopNotifier) Kirim(string, string, string) {}

type CSA struct {
	st    Store
	notif Notifier
	now   func() time.Time
	mu    sync.Mutex // menyerialkan perubahan; di database ganti dengan transaksi + optimistic locking
	seq   int
}

func New(st Store, n Notifier, now func() time.Time) *CSA {
	if n == nil {
		n = nopNotifier{}
	}
	if now == nil {
		now = time.Now
	}
	return &CSA{st: st, notif: n, now: now}
}

func (s *CSA) User(id string) (domain.User, bool) { return s.st.GetUser(id) }

func (s *CSA) nextID(prefix string) string {
	s.seq++
	return fmt.Sprintf("%s-%04d", prefix, s.seq)
}

func hanyaLini2(u domain.User) error {
	if u.Peran != domain.PeranLini2 {
		return domain.ErrTidakBerwenang
	}
	return nil
}

// ---------- Master ----------

func (s *CSA) ListProses(siklus string) []domain.ProsesBisnis {
	all := s.st.ListProses()
	if siklus == "" {
		return all
	}
	out := all[:0]
	for _, p := range all {
		if p.Siklus == siklus {
			out = append(out, p)
		}
	}
	return out
}

func (s *CSA) SimpanProses(u domain.User, p domain.ProsesBisnis) (domain.ProsesBisnis, error) {
	if err := hanyaLini2(u); err != nil {
		return p, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if p.ID == "" {
		p.ID = s.nextID("PB")
	}
	if err := domain.CekProsesBisnis(p, s.st.ListProses()); err != nil {
		return p, err
	}
	s.st.SaveProses(p)
	return p, nil
}

func (s *CSA) ListKontrol() []domain.Kontrol { return s.st.ListKontrol() }

func (s *CSA) GetKontrol(ref string) (domain.Kontrol, error) {
	k, ok := s.st.GetKontrol(ref)
	if !ok {
		return k, domain.ErrTidakDitemukan
	}
	return k, nil
}

func (s *CSA) SimpanKontrol(u domain.User, k domain.Kontrol) (domain.Kontrol, error) {
	if err := hanyaLini2(u); err != nil {
		return k, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := domain.CekKontrol(k, s.st.ListProses()); err != nil {
		return k, err
	}
	s.st.SaveKontrol(k)
	return k, nil
}

// LingkupPeriode mengembalikan key control yang wajib di-CSA pada kuartal tertentu (CSA-02).
func (s *CSA) LingkupPeriode(kuartal int) []domain.Kontrol {
	out := []domain.Kontrol{}
	for _, k := range s.st.ListKontrol() {
		if ok, _ := domain.MasukLingkup(k, kuartal, false); ok {
			out = append(out, k)
		}
	}
	return out
}

func (s *CSA) Template(ref string) (domain.TemplateKontrol, error) {
	t, ok := s.st.LatestTemplate(ref)
	if !ok {
		return t, domain.ErrTidakDitemukan
	}
	return t, nil
}

// SimpanTemplate membuat versi baru template. Berlaku mulai diambil dari
// periode efektif RCM, tidak diinput manual.
func (s *CSA) SimpanTemplate(u domain.User, ref string, t domain.TemplateKontrol) (domain.TemplateKontrol, error) {
	if err := hanyaLini2(u); err != nil {
		return t, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.st.GetKontrol(ref)
	if !ok {
		return t, domain.ErrTidakDitemukan
	}
	t.KontrolRef = ref
	t.BerlakuMulai = k.PeriodeEfektif
	t.Versi = 1
	if prev, ok := s.st.LatestTemplate(ref); ok {
		t.Versi = prev.Versi + 1
	}
	if err := domain.CekTemplate(t); err != nil {
		return t, err
	}
	s.st.SaveTemplate(t)
	return t, nil
}

// ---------- Dispatch ----------

func (s *CSA) Dispatch(u domain.User, in domain.DispatchInput) (domain.Assessment, []domain.KertasKerja, error) {
	if err := hanyaLini2(u); err != nil {
		return domain.Assessment{}, nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	k, ok := s.st.GetKontrol(in.KontrolRef)
	if !ok {
		return domain.Assessment{}, nil, domain.ErrTidakDitemukan
	}
	var tplPtr *domain.TemplateKontrol
	tpl, okT := s.st.LatestTemplate(in.KontrolRef)
	if okT {
		tplPtr = &tpl
	}
	now := s.now()
	if err := domain.CekDispatch(in, k, tplPtr, now); err != nil {
		return domain.Assessment{}, nil, err
	}
	a := domain.Assessment{
		ID: s.nextID("AS"), Periode: in.Periode, KontrolRef: k.Ref, Template: tpl,
		PerluNasabah: in.PerluNasabah, BatasSubmit: in.BatasSubmit, TargetValidasi: in.TargetValidasi,
		Catatan: in.Catatan, DibuatOleh: u.ID, DibuatPada: now,
	}
	var kks []domain.KertasKerja
	for _, unit := range in.UnitKerja {
		kk := domain.KertasKerja{
			ID: s.nextID("KK"), AssessmentID: a.ID, KontrolRef: k.Ref, UnitKerja: unit,
			Status:  domain.StatusDitugaskan,
			Riwayat: []domain.Riwayat{{Pada: now, Oleh: u.ID, Aksi: "DISPATCH", Ke: domain.StatusDitugaskan}},
		}
		a.KertasKerjaID = append(a.KertasKerjaID, kk.ID)
		s.st.SaveKertasKerja(kk)
		kks = append(kks, kk)
		s.notif.Kirim(unit, "Kertas kerja CSA baru", fmt.Sprintf("%s harus disubmit paling lambat %s", k.Ref, in.BatasSubmit.Format("2 Jan 2006")))
	}
	s.st.SaveAssessment(a)
	return a, kks, nil
}

// ---------- Kertas kerja ----------

func bolehLihat(u domain.User, kk domain.KertasKerja) bool {
	return u.Peran == domain.PeranLini2 || (u.Peran == domain.PeranLini1 && u.UnitKerja == kk.UnitKerja)
}

func (s *CSA) ListKertasKerja(u domain.User, status domain.Status) []domain.KertasKerja {
	out := []domain.KertasKerja{}
	for _, kk := range s.st.ListKertasKerja() {
		if bolehLihat(u, kk) && (status == "" || kk.Status == status) {
			out = append(out, kk)
		}
	}
	return out
}

func (s *CSA) GetKertasKerja(u domain.User, id string) (domain.KertasKerja, domain.Assessment, error) {
	kk, ok := s.st.GetKertasKerja(id)
	if !ok {
		return kk, domain.Assessment{}, domain.ErrTidakDitemukan
	}
	if !bolehLihat(u, kk) {
		return kk, domain.Assessment{}, domain.ErrTidakDitemukan // jangan bocorkan keberadaan data unit lain
	}
	a, _ := s.st.GetAssessment(kk.AssessmentID)
	return kk, a, nil
}

func (s *CSA) SimpanDraf(u domain.User, id string, is domain.IsianLini1) (domain.KertasKerja, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kk, _, err := s.GetKertasKerja(u, id)
	if err != nil {
		return kk, err
	}
	if err := kk.SimpanDraf(u, is, s.now()); err != nil {
		return kk, err
	}
	s.st.SaveKertasKerja(kk)
	return kk, nil
}

func (s *CSA) Submit(u domain.User, id string) (domain.KertasKerja, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kk, a, err := s.GetKertasKerja(u, id)
	if err != nil {
		return kk, err
	}
	if err := kk.Submit(u, a, s.now()); err != nil {
		return kk, err
	}
	s.st.SaveKertasKerja(kk)
	if kk.Status == domain.StatusTindakLanjut {
		s.notif.Kirim(string(domain.PeranLini2), "Kontrol tidak efektif", kk.KontrolRef+" · "+kk.UnitKerja+" perlu koordinasi perbaikan")
	} else {
		s.notif.Kirim(string(domain.PeranLini2), "Kertas kerja menunggu validasi", kk.KontrolRef+" · "+kk.UnitKerja)
	}
	return kk, nil
}

// ---------- Validasi ----------

func (s *CSA) AntrianValidasi(u domain.User) ([]domain.KertasKerja, error) {
	if err := hanyaLini2(u); err != nil {
		return nil, err
	}
	q := s.ListKertasKerja(u, domain.StatusMenungguValidasi)
	sort.Slice(q, func(i, j int) bool { return q[i].SubmitPada.Before(*q[j].SubmitPada) })
	return q, nil
}

func (s *CSA) Validasi(u domain.User, id string, in domain.ValidasiInput) (domain.KertasKerja, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	kk, a, err := s.GetKertasKerja(u, id)
	if err != nil {
		return kk, err
	}
	if err := kk.Validasi(u, a.Template, in, s.now()); err != nil {
		return kk, err
	}
	s.st.SaveKertasKerja(kk)
	if kk.Status == domain.StatusDikembalikan {
		s.notif.Kirim(kk.UnitKerja, "Kertas kerja dikembalikan", kk.KontrolRef+": "+in.Catatan)
	}
	return kk, nil
}

// ---------- Penugasan dan dashboard ----------

type RingkasanPenugasan struct {
	Assessment domain.Assessment        `json:"assessment"`
	Status     domain.StatusPenugasan   `json:"status"`
	Submit     int                      `json:"submit"`
	Total      int                      `json:"total"`
	PerUnit    map[string]domain.Status `json:"perUnit"`
}

func (s *CSA) Penugasan(u domain.User) ([]RingkasanPenugasan, error) {
	if err := hanyaLini2(u); err != nil {
		return nil, err
	}
	out := []RingkasanPenugasan{}
	for _, a := range s.st.ListAssessments() {
		r := RingkasanPenugasan{Assessment: a, PerUnit: map[string]domain.Status{}}
		var st []domain.Status
		for _, id := range a.KertasKerjaID {
			if kk, ok := s.st.GetKertasKerja(id); ok {
				st = append(st, kk.Status)
				r.PerUnit[kk.UnitKerja] = kk.Status
			}
		}
		r.Status, r.Submit, r.Total = domain.RingkasPenugasan(st)
		out = append(out, r)
	}
	return out, nil
}

type Dashboard struct {
	TotalKertasKerja   int                       `json:"totalKertasKerja"`
	PerStatus          map[domain.Status]int     `json:"perStatus"`
	FinalPerKesimpulan map[domain.Kesimpulan]int `json:"finalPerKesimpulan"`
	PersenSubmit       float64                   `json:"persenSubmit"`
	Terlambat          int                       `json:"terlambat"`
	TidakEfektif       []domain.KertasKerja      `json:"tidakEfektif"`
}

// Dashboard menghitung ringkasan. Kesimpulan final = hasil validasi Lini 2;
// tidak efektif dari Lini 1 langsung final karena tidak divalidasi (CSA-21).
func (s *CSA) Dashboard(u domain.User) (Dashboard, error) {
	if err := hanyaLini2(u); err != nil {
		return Dashboard{}, err
	}
	d := Dashboard{PerStatus: map[domain.Status]int{}, FinalPerKesimpulan: map[domain.Kesimpulan]int{}, TidakEfektif: []domain.KertasKerja{}}
	submit := 0
	for _, kk := range s.st.ListKertasKerja() {
		d.TotalKertasKerja++
		d.PerStatus[kk.Status]++
		if kk.Status.SudahSubmit() {
			submit++
		}
		if kk.Status.Final() {
			d.FinalPerKesimpulan[kk.Kesimpulan]++
		}
		if kk.Terlambat {
			d.Terlambat++
		}
		if kk.Status == domain.StatusTindakLanjut {
			d.TidakEfektif = append(d.TidakEfektif, kk)
		}
	}
	if d.TotalKertasKerja > 0 {
		d.PersenSubmit = float64(submit) * 100 / float64(d.TotalKertasKerja)
	}
	return d, nil
}
