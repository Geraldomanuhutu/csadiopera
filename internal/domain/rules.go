package domain

import (
	"fmt"
	"strings"
	"time"
)

// MinSampel adalah jumlah minimal sampel transaksi per kertas kerja.
// Pedoman: dokumentasi CSA berdasarkan 1 (satu) sampel transaksi.
const MinSampel = 1

// ---------- Lingkup periode (CSA-01, CSA-02) ----------

// MasukLingkup menentukan apakah key control wajib di-CSA pada kuartal tertentu.
//
// PK-0038 Bab V.B.2: CSA untuk setiap key control dilakukan semesteran pada
// kuartal 1 dan 3, ditambah kuartal 2 dan 4 untuk (a) kontrol semesteran,
// (b) kontrol tahunan (kuartal 4), dan (c) kontrol yang prosesnya berubah.
//
// Interpretasi yang dipakai di sini (perlu dikonfirmasi Lini 2):
// kontrol tahunan hanya di kuartal 4, sehingga tidak ikut kuartal 1 dan 3.
func MasukLingkup(k Kontrol, kuartal int, adaPerubahan bool) (bool, string) {
	if !k.KeyControl {
		return false, "bukan key control"
	}
	if adaPerubahan {
		return true, "ada perubahan proses atau kontrol"
	}
	switch kuartal {
	case 1, 3:
		if k.Frekuensi == FrekTahunan {
			return false, "kontrol tahunan hanya di-CSA pada kuartal 4"
		}
		return true, "CSA semesteran kuartal 1 dan 3"
	case 2:
		if k.Frekuensi == FrekSemesteran {
			return true, "kontrol semesteran"
		}
		return false, "kuartal 2 hanya untuk kontrol semesteran atau yang berubah"
	case 4:
		if k.Frekuensi == FrekSemesteran || k.Frekuensi == FrekTahunan {
			return true, "kontrol semesteran atau tahunan"
		}
		return false, "kuartal 4 hanya untuk kontrol semesteran, tahunan, atau yang berubah"
	}
	return false, "kuartal tidak valid"
}

// ---------- Dispatch (CSA-03 s.d. CSA-05) ----------

// DispatchInput adalah isian layar New Assessment.
type DispatchInput struct {
	Periode        Periode   `json:"periode"`
	KontrolRef     string    `json:"kontrolRef"`
	UnitKerja      []string  `json:"unitKerja"`
	PerluNasabah   bool      `json:"perluNasabah"`
	AdaPerubahan   bool      `json:"adaPerubahan"` // kontrol masuk lingkup karena perubahan
	BatasSubmit    time.Time `json:"batasSubmit"`
	TargetValidasi time.Time `json:"targetValidasi"`
	Catatan        string    `json:"catatan"`
}

// CekDispatch memeriksa aturan sebelum kertas kerja dibuat.
func CekDispatch(in DispatchInput, k Kontrol, tpl *TemplateKontrol, now time.Time) error {
	var es RuleErrors
	if !k.KeyControl {
		es = append(es, rule("CSA-01", "CSA hanya untuk key control (pengendalian utama)", dasarVB2))
	} else if ok, alasan := MasukLingkup(k, in.Periode.Kuartal, in.AdaPerubahan); !ok {
		es = append(es, rule("CSA-02", "kontrol tidak masuk lingkup kuartal ini: "+alasan, dasarVB2))
	}
	if tpl == nil || len(tpl.Atribut) == 0 {
		es = append(es, rule("CSA-03", "template kontrol CSA belum ada atau belum punya atribut pengujian", usulan))
	}
	seen := map[string]bool{}
	for _, u := range in.UnitKerja {
		if strings.TrimSpace(u) == "" || seen[u] {
			es = append(es, rule("CSA-04", "unit kerja kosong atau dobel: "+u, usulan))
		}
		seen[u] = true
	}
	if len(in.UnitKerja) == 0 {
		es = append(es, rule("CSA-04", "pilih minimal satu unit kerja", usulan))
	}
	if in.BatasSubmit.Before(truncDay(now)) {
		es = append(es, rule("CSA-05", "batas submit tidak boleh sebelum hari ini", usulan))
	}
	if !in.TargetValidasi.IsZero() && in.TargetValidasi.Before(in.BatasSubmit) {
		es = append(es, rule("CSA-05", "target validasi tidak boleh sebelum batas submit", usulan))
	}
	return es.OrNil()
}

// ---------- Isian dan submit Lini 1 (CSA-10 s.d. CSA-21) ----------

// TurunkanKesimpulan menghitung kesimpulan dari jawaban atribut (CSA-20):
// semua Ya = efektif, ada Tidak = tidak efektif, centang tidak ada transaksi = tidak ada transaksi.
func TurunkanKesimpulan(tpl TemplateKontrol, is IsianLini1) Kesimpulan {
	if is.TidakAdaTransaksi {
		return KesimpulanTidakAdaTransaksi
	}
	for _, a := range tpl.Atribut {
		if v, ok := is.JawabanAtribut[a.Kode]; ok && !v {
			return KesimpulanTidakEfektif
		}
	}
	return KesimpulanEfektif
}

// CekIsian memeriksa kelengkapan isian sebelum submit.
func CekIsian(tpl TemplateKontrol, perluNasabah bool, is IsianLini1) error {
	var es RuleErrors
	if is.AdaPerubahanProses == nil {
		es = append(es, rule("CSA-12", "jawab apakah ada perubahan proses bisnis atau kontrol", dasarVB2+" huruf a"))
	}
	if len(is.Pelaksana) == 0 {
		es = append(es, rule("CSA-13", "isi minimal satu pelaksana kontrol", usulan))
	}
	for i, p := range is.Pelaksana {
		if strings.TrimSpace(p.Nama) == "" || strings.TrimSpace(p.Jabatan) == "" {
			es = append(es, rule("CSA-13", fmt.Sprintf("nama dan jabatan pelaksana %d wajib diisi", i+1), usulan))
		}
	}
	if is.TidakAdaTransaksi {
		// CSA-19: tidak ada transaksi, atribut dan sampel tidak diisi. Lini 2 memvalidasi populasi.
		return es.OrNil()
	}
	for _, a := range tpl.Atribut {
		if _, ok := is.JawabanAtribut[a.Kode]; !ok {
			es = append(es, rule("CSA-14", "jawab atribut "+a.Kode, dasarVB2+" huruf c"))
		}
	}
	if len(is.Sampel) < MinSampel {
		es = append(es, rule("CSA-15", fmt.Sprintf("isi minimal %d sampel transaksi", MinSampel), dasarVB2+" huruf b dan d"))
	}
	for i, s := range is.Sampel {
		if strings.TrimSpace(s.IDTransaksi) == "" || strings.TrimSpace(s.Tanggal) == "" {
			es = append(es, rule("CSA-15", fmt.Sprintf("ID transaksi dan tanggal sampel %d wajib diisi", i+1), dasarVB2+" huruf b"))
		}
		if perluNasabah && strings.TrimSpace(s.NamaNasabah) == "" {
			es = append(es, rule("CSA-15", fmt.Sprintf("nama nasabah sampel %d wajib diisi", i+1), usulan))
		}
	}
	for _, d := range tpl.Dokumen {
		if !d.WajibUnggah {
			continue
		}
		ada := false
		for _, b := range is.Bukti {
			if b.Dokumen == d.Nama && b.FileRef != "" {
				ada = true
				break
			}
		}
		if !ada {
			es = append(es, rule("CSA-16", "unggah bukti untuk dokumen: "+d.Nama, dasarVB2+" huruf b"))
		}
	}
	if TurunkanKesimpulan(tpl, is) == KesimpulanTidakEfektif {
		if strings.TrimSpace(is.AlasanTidakEfektif) == "" || strings.TrimSpace(is.TidakEfektifSejak) == "" {
			es = append(es, rule("CSA-17", "kesimpulan tidak efektif wajib mencantumkan alasan dan periode mulai tidak efektif", dasarVB1+" huruf b"))
		}
		r := is.Remediasi
		if r == nil || strings.TrimSpace(r.Rencana) == "" || strings.TrimSpace(r.Deadline) == "" || strings.TrimSpace(r.PIC) == "" {
			es = append(es, rule("CSA-18", "kesimpulan tidak efektif wajib berisi rencana remediasi, deadline, dan PIC", dasarVB2+" (koordinasi perbaikan); rincian field [Usulan]"))
		}
	}
	return es.OrNil()
}

// SimpanDraf menyimpan isian tanpa memeriksa kelengkapan.
func (kk *KertasKerja) SimpanDraf(u User, is IsianLini1, now time.Time) error {
	if err := kk.cekAksesLini1(u); err != nil {
		return err
	}
	dari := kk.Status
	kk.Isian = is
	if kk.Status == StatusDitugaskan {
		kk.Status = StatusDraft
	}
	kk.catat(u, "SIMPAN_DRAF", dari, "", now)
	return nil
}

// Submit menyampaikan kesimpulan CSA ke Lini 2.
func (kk *KertasKerja) Submit(u User, a Assessment, now time.Time) error {
	if err := kk.cekAksesLini1(u); err != nil {
		return err
	}
	if err := CekIsian(a.Template, a.PerluNasabah, kk.Isian); err != nil {
		return err
	}
	dari := kk.Status
	if dari == StatusDikembalikan {
		kk.RevisiKe++
	}
	kk.Kesimpulan = TurunkanKesimpulan(a.Template, kk.Isian)
	t := now
	kk.SubmitPada = &t
	kk.Terlambat = now.After(endOfDay(a.BatasSubmit))
	aksi := "SUBMIT"
	if kk.RevisiKe > 0 && dari == StatusDikembalikan {
		aksi = "SUBMIT_ULANG"
	}
	if kk.Kesimpulan == KesimpulanTidakEfektif {
		// CSA-21: tidak efektif tidak divalidasi; Lini 2 berkoordinasi perbaikan dan memantau tindak lanjut.
		kk.Status = StatusTindakLanjut
	} else {
		kk.Status = StatusMenungguValidasi
	}
	kk.catat(u, aksi, dari, string(kk.Kesimpulan), now)
	return nil
}

func (kk *KertasKerja) cekAksesLini1(u User) error {
	if u.Peran != PeranLini1 || u.UnitKerja != kk.UnitKerja {
		return RuleErrors{rule("CSA-10", "hanya Lini 1 di "+kk.UnitKerja+" yang boleh mengisi kertas kerja ini", dasarVB1)}
	}
	if !kk.Status.BisaDiisiLini1() {
		return RuleErrors{rule("CSA-11", "kertas kerja berstatus "+string(kk.Status)+" tidak bisa diubah", usulan)}
	}
	return nil
}

// ---------- Validasi Lini 2 (CSA-30 s.d. CSA-35) ----------

type ValidasiInput struct {
	SesuaiAtribut map[string]bool `json:"sesuaiAtribut"`
	Keputusan     Keputusan       `json:"keputusan"`
	Catatan       string          `json:"catatan"`
}

// Validasi mencatat hasil evaluasi Lini 2 atas kertas kerja yang menunggu validasi.
func (kk *KertasKerja) Validasi(u User, tpl TemplateKontrol, in ValidasiInput, now time.Time) error {
	if u.Peran != PeranLini2 {
		return RuleErrors{rule("CSA-30", "validasi hanya oleh Lini 2 fungsi ICoFR", dasarVB1)}
	}
	if kk.Status != StatusMenungguValidasi {
		return RuleErrors{rule("CSA-31", "hanya kertas kerja yang menunggu validasi yang bisa divalidasi", usulan)}
	}
	catatan := strings.TrimSpace(in.Catatan)
	var es RuleErrors
	var ke Status
	switch kk.Kesimpulan {
	case KesimpulanEfektif:
		switch in.Keputusan {
		case KeputusanSetujui:
			for _, a := range tpl.Atribut {
				v, ok := in.SesuaiAtribut[a.Kode]
				if !ok {
					es = append(es, rule("CSA-32", "nilai kesesuaian atribut "+a.Kode+" dengan bukti", dasarVB3a1+" huruf a dan b"))
				} else if !v {
					es = append(es, rule("CSA-32", "atribut "+a.Kode+" tidak sesuai bukti, tidak bisa disetujui", dasarVB3a1+" huruf b"))
				}
			}
			ke = StatusTervalidasi
		case KeputusanKembalikan:
			if catatan == "" {
				es = append(es, rule("CSA-33", "catatan wajib diisi saat mengembalikan ke Lini 1", usulan))
			}
			ke = StatusDikembalikan
		case KeputusanTidakEfektif:
			if catatan == "" {
				es = append(es, rule("CSA-34", "catatan wajib diisi saat menetapkan tidak efektif", dasarVB3a1+" huruf d"))
			}
			ke = StatusTindakLanjut
		default:
			es = append(es, rule("CSA-32", "keputusan untuk kesimpulan efektif: SETUJUI, KEMBALIKAN, atau TETAPKAN_TIDAK_EFEKTIF", dasarVB3a1))
		}
	case KesimpulanTidakAdaTransaksi:
		switch in.Keputusan {
		case KeputusanKonfirmasiNihil:
			ke = StatusTervalidasi
		case KeputusanAdaTransaksi:
			if catatan == "" {
				es = append(es, rule("CSA-35", "sebutkan transaksi yang ditemukan di populasi", dasarVB3a3))
			}
			ke = StatusDikembalikan
		default:
			es = append(es, rule("CSA-35", "keputusan untuk tidak ada transaksi: KONFIRMASI_TIDAK_ADA_TRANSAKSI atau ADA_TRANSAKSI", dasarVB3a3))
		}
	default:
		es = append(es, rule("CSA-31", "kesimpulan tidak efektif tidak divalidasi", dasarVB3a2))
	}
	if len(es) > 0 {
		return es
	}
	dari := kk.Status
	kk.Status = ke
	if in.Keputusan == KeputusanTidakEfektif {
		kk.Kesimpulan = KesimpulanTidakEfektif
	}
	if in.Keputusan == KeputusanAdaTransaksi {
		// Lini 1 harus mengisi CSA atas transaksi yang ditemukan.
		kk.Isian.TidakAdaTransaksi = false
	}
	kk.HasilValidasi = append(kk.HasilValidasi, HasilValidasi{
		Validator: u.ID, Pada: now, RevisiKe: kk.RevisiKe,
		SesuaiAtribut: in.SesuaiAtribut, Keputusan: in.Keputusan, Catatan: catatan,
	})
	kk.catat(u, string(in.Keputusan), dari, catatan, now)
	return nil
}

// ---------- Riwayat (CSA-36) ----------

func (kk *KertasKerja) catat(u User, aksi string, dari Status, catatan string, now time.Time) {
	kk.Riwayat = append(kk.Riwayat, Riwayat{Pada: now, Oleh: u.ID, Aksi: aksi, Dari: dari, Ke: kk.Status, Catatan: catatan})
}

func truncDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func endOfDay(t time.Time) time.Time { return truncDay(t).Add(24*time.Hour - time.Nanosecond) }
