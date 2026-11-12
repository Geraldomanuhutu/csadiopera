// Package domain berisi entitas dan aturan bisnis CSA ICoFR.
// Tidak bergantung pada database, framework HTTP, atau library lain,
// sehingga bisa dipindah utuh ke codebase OPERA.
package domain

import "time"

// Peran pengguna.
type Peran string

const (
	PeranLini1 Peran = "LINI_1" // control owner di unit kerja
	PeranLini2 Peran = "LINI_2" // fungsi penerapan ICoFR (Divisi MRO)
)

// User adalah pengguna yang sudah terautentikasi.
type User struct {
	ID        string `json:"id"`
	Nama      string `json:"nama"`
	Jabatan   string `json:"jabatan"`
	Peran     Peran  `json:"peran"`
	UnitKerja string `json:"unitKerja,omitempty"` // wajib untuk Lini 1
}

// ---------- Master ----------

// ProsesBisnis adalah master proses bisnis ICoFR per siklus.
type ProsesBisnis struct {
	ID            string      `json:"id"`
	Siklus        string      `json:"siklus"`
	Kode          string      `json:"kode"` // contoh "3"
	Nama          string      `json:"nama"`
	PemilikProses string      `json:"pemilikProses"`
	SubProses     []SubProses `json:"subProses"`
}

// SubProses adalah turunan proses bisnis. Kode harus diawali kode proses ("3.1").
type SubProses struct {
	Kode string `json:"kode"`
	Nama string `json:"nama"`
}

// Frekuensi pelaksanaan pengendalian, mengikuti kolom RCM.
type Frekuensi string

const (
	FrekLebihDariSekaliSehari Frekuensi = "LEBIH_DARI_SEKALI_SEHARI"
	FrekHarian                Frekuensi = "HARIAN"
	FrekMingguan              Frekuensi = "MINGGUAN"
	FrekBulanan               Frekuensi = "BULANAN"
	FrekTriwulanan            Frekuensi = "TRIWULANAN"
	FrekSemesteran            Frekuensi = "SEMESTERAN"
	FrekTahunan               Frekuensi = "TAHUNAN"
	FrekAdHoc                 Frekuensi = "AD_HOC"
)

// Kontrol adalah satu baris RCM ICoFR. Kolom yang tidak dipakai logika CSA
// (IPO, asersi, aplikasi, FSLI, dsb.) disimpan apa adanya di Atribut RCM.
type Kontrol struct {
	Ref                string         `json:"ref"` // Referensi pengendalian, contoh MKR-31-02 (kunci unik)
	Siklus             string         `json:"siklus"`
	ProsesKode         []string       `json:"prosesKode"`
	SubProsesKode      []string       `json:"subProsesKode"`
	RefRisiko          []string       `json:"refRisiko"`
	NamaAktivitas      string         `json:"namaAktivitas"`
	Deskripsi          string         `json:"deskripsi"`
	KeyControl         bool           `json:"keyControl"`
	Sifat              string         `json:"sifat"` // Manual, Otomatis, ITDM - EUC, ITDM - IPE, MRC
	Frekuensi          Frekuensi      `json:"frekuensi"`
	FungsiPelaksana    string         `json:"fungsiPelaksana"`
	PelakuPengendalian string         `json:"pelakuPengendalian"`
	PeriodeEfektif     string         `json:"periodeEfektif"` // YYYY-MM, dasar versi template
	AtributRCM         map[string]any `json:"atributRCM,omitempty"`
}

// TipeField menentukan format isian field dokumen di kertas kerja Lini 1.
type TipeField string

const (
	FieldTeks    TipeField = "TEKS"
	FieldTanggal TipeField = "TANGGAL"
	FieldAngka   TipeField = "ANGKA"
	FieldNominal TipeField = "NOMINAL"
)

// TemplateKontrol berisi atribut pengujian dan dokumen bukti untuk satu key control.
// Template berversi: perubahan membuat versi baru, kertas kerja yang sudah
// di-dispatch tetap memakai versi lama (snapshot).
type TemplateKontrol struct {
	KontrolRef   string             `json:"kontrolRef"`
	Versi        int                `json:"versi"`
	BerlakuMulai string             `json:"berlakuMulai"` // YYYY-MM, mengikuti periode efektif RCM
	Atribut      []AtributPengujian `json:"atribut"`
	Dokumen      []DokumenBukti     `json:"dokumen"`
}

type AtributPengujian struct {
	Kode      string `json:"kode"` // A1, A2, ...
	Deskripsi string `json:"deskripsi"`
}

type DokumenBukti struct {
	Nama        string         `json:"nama"`
	Field       []FieldDokumen `json:"field"`
	WajibUnggah bool           `json:"wajibUnggah"`
}

type FieldDokumen struct {
	Nama string    `json:"nama"`
	Tipe TipeField `json:"tipe"`
}

// ---------- Penugasan ----------

// Periode CSA, contoh {Tahun: 2026, Kuartal: 3}.
type Periode struct {
	Tahun   int `json:"tahun"`
	Kuartal int `json:"kuartal"` // 1..4
}

// Assessment adalah satu penugasan CSA atas satu key control ke beberapa unit kerja.
type Assessment struct {
	ID             string          `json:"id"`
	Periode        Periode         `json:"periode"`
	KontrolRef     string          `json:"kontrolRef"`
	Template       TemplateKontrol `json:"template"` // snapshot saat dispatch
	PerluNasabah   bool            `json:"perluNasabah"`
	BatasSubmit    time.Time       `json:"batasSubmit"`
	TargetValidasi time.Time       `json:"targetValidasi"`
	Catatan        string          `json:"catatan,omitempty"`
	DibuatOleh     string          `json:"dibuatOleh"`
	DibuatPada     time.Time       `json:"dibuatPada"`
	KertasKerjaID  []string        `json:"kertasKerjaId"`
}

// ---------- Kertas kerja ----------

type Kesimpulan string

const (
	KesimpulanEfektif           Kesimpulan = "EFEKTIF"
	KesimpulanTidakEfektif      Kesimpulan = "TIDAK_EFEKTIF"
	KesimpulanTidakAdaTransaksi Kesimpulan = "TIDAK_ADA_TRANSAKSI"
)

type Pelaksana struct {
	Nama    string `json:"nama"`
	Jabatan string `json:"jabatan"`
}

// Sampel transaksi yang diuji Lini 1. Pedoman: 1 sampel per key control.
type Sampel struct {
	NamaNasabah  string            `json:"namaNasabah,omitempty"`
	IDTransaksi  string            `json:"idTransaksi"`
	Tanggal      string            `json:"tanggal"` // YYYY-MM-DD
	FieldDokumen map[string]string `json:"fieldDokumen,omitempty"`
}

// Bukti adalah file yang diunggah untuk satu dokumen di template.
type Bukti struct {
	Dokumen  string `json:"dokumen"` // nama dokumen di template
	FileName string `json:"fileName"`
	FileRef  string `json:"fileRef"` // kunci object storage, diisi layanan upload
}

type Remediasi struct {
	Rencana  string `json:"rencana"`
	Deadline string `json:"deadline"` // YYYY-MM-DD
	PIC      string `json:"pic"`
}

// IsianLini1 adalah isian kertas kerja yang bisa disimpan sebagai draf.
type IsianLini1 struct {
	Pelaksana          []Pelaksana     `json:"pelaksana"`
	AdaPerubahanProses *bool           `json:"adaPerubahanProses"`
	JawabanAtribut     map[string]bool `json:"jawabanAtribut"` // kode atribut -> Ya(true)/Tidak(false)
	TidakAdaTransaksi  bool            `json:"tidakAdaTransaksi"`
	Sampel             []Sampel        `json:"sampel"`
	Bukti              []Bukti         `json:"bukti"`
	AlasanTidakEfektif string          `json:"alasanTidakEfektif,omitempty"`
	TidakEfektifSejak  string          `json:"tidakEfektifSejak,omitempty"` // YYYY-MM
	Remediasi          *Remediasi      `json:"remediasi,omitempty"`
}

// KertasKerja adalah kertas kerja CSA satu unit kerja dalam satu assessment.
type KertasKerja struct {
	ID            string          `json:"id"`
	AssessmentID  string          `json:"assessmentId"`
	KontrolRef    string          `json:"kontrolRef"`
	UnitKerja     string          `json:"unitKerja"`
	Status        Status          `json:"status"`
	Isian         IsianLini1      `json:"isian"`
	Kesimpulan    Kesimpulan      `json:"kesimpulan,omitempty"`
	RevisiKe      int             `json:"revisiKe"`
	SubmitPada    *time.Time      `json:"submitPada,omitempty"`
	Terlambat     bool            `json:"terlambat"`
	HasilValidasi []HasilValidasi `json:"hasilValidasi"`
	Riwayat       []Riwayat       `json:"riwayat"`
}

// ---------- Validasi ----------

type Keputusan string

const (
	KeputusanSetujui         Keputusan = "SETUJUI"                // kesimpulan efektif diterima
	KeputusanKembalikan      Keputusan = "KEMBALIKAN"             // [Usulan] dikembalikan ke Lini 1 untuk revisi
	KeputusanTidakEfektif    Keputusan = "TETAPKAN_TIDAK_EFEKTIF" // Lini 2 menyimpulkan tidak efektif
	KeputusanKonfirmasiNihil Keputusan = "KONFIRMASI_TIDAK_ADA_TRANSAKSI"
	KeputusanAdaTransaksi    Keputusan = "ADA_TRANSAKSI" // populasi menunjukkan ada transaksi
)

type HasilValidasi struct {
	Validator     string          `json:"validator"`
	Pada          time.Time       `json:"pada"`
	RevisiKe      int             `json:"revisiKe"`
	SesuaiAtribut map[string]bool `json:"sesuaiAtribut,omitempty"`
	Keputusan     Keputusan       `json:"keputusan"`
	Catatan       string          `json:"catatan,omitempty"`
}

type Riwayat struct {
	Pada    time.Time `json:"pada"`
	Oleh    string    `json:"oleh"`
	Aksi    string    `json:"aksi"`
	Dari    Status    `json:"dari,omitempty"`
	Ke      Status    `json:"ke"`
	Catatan string    `json:"catatan,omitempty"`
}
