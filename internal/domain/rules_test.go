package domain

import (
	"errors"
	"testing"
	"time"
)

var (
	now  = time.Date(2026, 11, 12, 9, 0, 0, 0, time.UTC)
	l1   = User{ID: "sari", Peran: PeranLini1, UnitKerja: "Unit Mekaar Cibinong"}
	l1b  = User{ID: "andi", Peran: PeranLini1, UnitKerja: "Unit Mekaar Citeureup"}
	l2   = User{ID: "rina", Peran: PeranLini2}
	tpl  = TemplateKontrol{KontrolRef: "MKR-31-02", Versi: 1, Atribut: []AtributPengujian{{Kode: "A1", Deskripsi: "a"}, {Kode: "A2", Deskripsi: "b"}}, Dokumen: []DokumenBukti{{Nama: "Screenshot", WajibUnggah: true}}}
	asmt = Assessment{ID: "AS-1", Template: tpl, PerluNasabah: true, BatasSubmit: time.Date(2026, 11, 14, 0, 0, 0, 0, time.UTC)}
)

func tidak() *bool { b := false; return &b }

func isianLengkap() IsianLini1 {
	return IsianLini1{
		Pelaksana:          []Pelaksana{{Nama: "Sari Wulandari", Jabatan: "KUM"}},
		AdaPerubahanProses: tidak(),
		JawabanAtribut:     map[string]bool{"A1": true, "A2": true},
		Sampel:             []Sampel{{NamaNasabah: "Nasabah A", IDTransaksi: "TRX-1", Tanggal: "2026-10-03"}},
		Bukti:              []Bukti{{Dokumen: "Screenshot", FileName: "a.png", FileRef: "obj/a.png"}},
	}
}

func kodeErr(err error) map[string]bool {
	out := map[string]bool{}
	var es RuleErrors
	if errors.As(err, &es) {
		for _, e := range es {
			out[e.Kode] = true
		}
	}
	return out
}

func TestMasukLingkup(t *testing.T) {
	harian := Kontrol{KeyControl: true, Frekuensi: FrekHarian}
	tahunan := Kontrol{KeyControl: true, Frekuensi: FrekTahunan}
	semester := Kontrol{KeyControl: true, Frekuensi: FrekSemesteran}
	cases := []struct {
		k       Kontrol
		q       int
		berubah bool
		want    bool
	}{
		{harian, 1, false, true}, {harian, 3, false, true}, {harian, 2, false, false}, {harian, 4, false, false},
		{tahunan, 3, false, false}, {tahunan, 4, false, true},
		{semester, 2, false, true}, {semester, 4, false, true},
		{harian, 2, true, true}, // berubah: wajib di kuartal mana pun
		{Kontrol{KeyControl: false, Frekuensi: FrekHarian}, 1, false, false},
	}
	for i, c := range cases {
		if got, _ := MasukLingkup(c.k, c.q, c.berubah); got != c.want {
			t.Errorf("kasus %d: dapat %v, mau %v", i, got, c.want)
		}
	}
}

func TestCekIsianMenyebutSemuaYangKurang(t *testing.T) {
	err := CekIsian(tpl, true, IsianLini1{})
	k := kodeErr(err)
	for _, want := range []string{"CSA-12", "CSA-13", "CSA-14", "CSA-15", "CSA-16"} {
		if !k[want] {
			t.Errorf("harusnya ada %s, dapat %v", want, k)
		}
	}
}

func TestTidakEfektifWajibAlasanDanRemediasi(t *testing.T) {
	is := isianLengkap()
	is.JawabanAtribut["A2"] = false
	k := kodeErr(CekIsian(tpl, true, is))
	if !k["CSA-17"] || !k["CSA-18"] {
		t.Fatalf("harusnya CSA-17 dan CSA-18, dapat %v", k)
	}
	is.AlasanTidakEfektif, is.TidakEfektifSejak = "rekap PKM tidak diverifikasi", "2026-08"
	is.Remediasi = &Remediasi{Rencana: "sosialisasi ulang", Deadline: "2026-12-15", PIC: "KUM"}
	if err := CekIsian(tpl, true, is); err != nil {
		t.Fatalf("harusnya lolos: %v", err)
	}
}

func TestTidakAdaTransaksiTidakWajibAtributDanSampel(t *testing.T) {
	is := IsianLini1{Pelaksana: []Pelaksana{{Nama: "Sari", Jabatan: "KUM"}}, AdaPerubahanProses: tidak(), TidakAdaTransaksi: true}
	if err := CekIsian(tpl, true, is); err != nil {
		t.Fatalf("harusnya lolos: %v", err)
	}
	if TurunkanKesimpulan(tpl, is) != KesimpulanTidakAdaTransaksi {
		t.Fatal("kesimpulan harusnya tidak ada transaksi")
	}
}

func TestUnitLainTidakBolehMengisi(t *testing.T) {
	kk := KertasKerja{UnitKerja: "Unit Mekaar Cibinong", Status: StatusDitugaskan}
	if !kodeErr(kk.SimpanDraf(l1b, isianLengkap(), now))["CSA-10"] {
		t.Fatal("Lini 1 unit lain harusnya ditolak")
	}
	if !kodeErr(kk.SimpanDraf(l2, isianLengkap(), now))["CSA-10"] {
		t.Fatal("Lini 2 tidak mengisi kertas kerja")
	}
}

func TestSubmitTidakEfektifLangsungTindakLanjut(t *testing.T) {
	kk := KertasKerja{UnitKerja: "Unit Mekaar Cibinong", Status: StatusDitugaskan}
	is := isianLengkap()
	is.JawabanAtribut["A1"] = false
	is.AlasanTidakEfektif, is.TidakEfektifSejak = "x", "2026-08"
	is.Remediasi = &Remediasi{Rencana: "r", Deadline: "2026-12-01", PIC: "p"}
	_ = kk.SimpanDraf(l1, is, now)
	if err := kk.Submit(l1, asmt, now); err != nil {
		t.Fatal(err)
	}
	if kk.Status != StatusTindakLanjut || kk.Kesimpulan != KesimpulanTidakEfektif {
		t.Fatalf("dapat %s / %s", kk.Status, kk.Kesimpulan)
	}
	if !kodeErr(kk.Validasi(l2, tpl, ValidasiInput{Keputusan: KeputusanSetujui}, now))["CSA-31"] {
		t.Fatal("tidak efektif tidak boleh divalidasi")
	}
}

func TestValidasiSetujuiButuhSemuaAtributSesuai(t *testing.T) {
	kk := KertasKerja{UnitKerja: "Unit Mekaar Cibinong", Status: StatusDitugaskan}
	_ = kk.SimpanDraf(l1, isianLengkap(), now)
	if err := kk.Submit(l1, asmt, now); err != nil {
		t.Fatal(err)
	}
	if kk.Status != StatusMenungguValidasi {
		t.Fatalf("status %s", kk.Status)
	}
	if !kodeErr(kk.Validasi(l1, tpl, ValidasiInput{Keputusan: KeputusanSetujui}, now))["CSA-30"] {
		t.Fatal("Lini 1 tidak boleh memvalidasi")
	}
	err := kk.Validasi(l2, tpl, ValidasiInput{Keputusan: KeputusanSetujui, SesuaiAtribut: map[string]bool{"A1": true, "A2": false}}, now)
	if !kodeErr(err)["CSA-32"] {
		t.Fatal("ada atribut tidak sesuai, harusnya ditolak")
	}
	if err := kk.Validasi(l2, tpl, ValidasiInput{Keputusan: KeputusanSetujui, SesuaiAtribut: map[string]bool{"A1": true, "A2": true}}, now); err != nil {
		t.Fatal(err)
	}
	if kk.Status != StatusTervalidasi || len(kk.HasilValidasi) != 1 {
		t.Fatalf("status %s, validasi %d", kk.Status, len(kk.HasilValidasi))
	}
}

func TestKembalikanLaluSubmitUlangMenaikkanRevisi(t *testing.T) {
	kk := KertasKerja{UnitKerja: "Unit Mekaar Cibinong", Status: StatusDitugaskan}
	_ = kk.SimpanDraf(l1, isianLengkap(), now)
	_ = kk.Submit(l1, asmt, now)
	if !kodeErr(kk.Validasi(l2, tpl, ValidasiInput{Keputusan: KeputusanKembalikan}, now))["CSA-33"] {
		t.Fatal("kembalikan tanpa catatan harusnya ditolak")
	}
	if err := kk.Validasi(l2, tpl, ValidasiInput{Keputusan: KeputusanKembalikan, Catatan: "screenshot tanpa tanggal"}, now); err != nil {
		t.Fatal(err)
	}
	if kk.Status != StatusDikembalikan {
		t.Fatalf("status %s", kk.Status)
	}
	if err := kk.Submit(l1, asmt, now.Add(2*time.Hour)); err != nil {
		t.Fatal(err)
	}
	if kk.RevisiKe != 1 || kk.Status != StatusMenungguValidasi {
		t.Fatalf("revisi %d status %s", kk.RevisiKe, kk.Status)
	}
	if kk.Riwayat[len(kk.Riwayat)-1].Aksi != "SUBMIT_ULANG" {
		t.Fatalf("aksi terakhir %s", kk.Riwayat[len(kk.Riwayat)-1].Aksi)
	}
}

func TestSubmitTerlambatDitandai(t *testing.T) {
	kk := KertasKerja{UnitKerja: "Unit Mekaar Cibinong", Status: StatusDitugaskan}
	_ = kk.SimpanDraf(l1, isianLengkap(), now)
	_ = kk.Submit(l1, asmt, time.Date(2026, 11, 15, 8, 0, 0, 0, time.UTC))
	if !kk.Terlambat {
		t.Fatal("submit sesudah batas harusnya ditandai terlambat")
	}
}

func TestRingkasPenugasan(t *testing.T) {
	cases := map[StatusPenugasan][]Status{
		PenugasanAssigned:   {StatusDitugaskan, StatusDraft},
		PenugasanInProgress: {StatusMenungguValidasi, StatusDraft},
		PenugasanSubmitted:  {StatusMenungguValidasi, StatusTervalidasi},
		PenugasanValidated:  {StatusTervalidasi, StatusTindakLanjut},
	}
	for want, st := range cases {
		if got, _, _ := RingkasPenugasan(st); got != want {
			t.Errorf("%v: dapat %s mau %s", st, got, want)
		}
	}
}

func TestKodeSubProsesHarusDiawaliKodeProses(t *testing.T) {
	p := ProsesBisnis{Siklus: "Kredit Mekaar", Kode: "9", Nama: "X", PemilikProses: "Divisi BUM", SubProses: []SubProses{{Kode: "8.1", Nama: "salah"}}}
	if !kodeErr(CekProsesBisnis(p, nil))["MST-03"] {
		t.Fatal("harusnya MST-03")
	}
	existing := []ProsesBisnis{{ID: "x", Siklus: "Kredit Mekaar", Kode: "9"}}
	p.SubProses = nil
	if !kodeErr(CekProsesBisnis(p, existing))["MST-02"] {
		t.Fatal("kode proses dobel harusnya MST-02")
	}
}
