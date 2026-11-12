// Package seed mengisi data contoh yang sama dengan demo web OPERA.
package seed

import (
	"opera/csa/internal/domain"
	"opera/csa/internal/store"
)

func Isi(m *store.Memory) {
	m.SaveUser(domain.User{ID: "rina", Nama: "Rina Pratama", Jabatan: "Staf ICoFR · Divisi MRO", Peran: domain.PeranLini2})
	m.SaveUser(domain.User{ID: "sari", Nama: "Sari Wulandari", Jabatan: "Kepala Unit Mekaar Cibinong", Peran: domain.PeranLini1, UnitKerja: "Unit Mekaar Cibinong"})
	m.SaveUser(domain.User{ID: "andi", Nama: "Andi Firmansyah", Jabatan: "Kepala Unit Mekaar Citeureup", Peran: domain.PeranLini1, UnitKerja: "Unit Mekaar Citeureup"})
	m.SaveUser(domain.User{ID: "amk", Nama: "Kepala Divisi AMK", Jabatan: "Kepala Divisi AMK", Peran: domain.PeranLini1, UnitKerja: "Divisi AMK"})

	m.SaveProses(domain.ProsesBisnis{ID: "PB-SEED-3", Siklus: "Kredit Mekaar", Kode: "3", Nama: "Pembayaran Kredit", PemilikProses: "Divisi BUM",
		SubProses: []domain.SubProses{{Kode: "3.1", Nama: "Penerimaan Pembayaran Nasabah melalui PKM"}, {Kode: "3.2", Nama: "Penggunaan Dana UP"}, {Kode: "3.3", Nama: "Pelunasan Dini"}}})
	m.SaveProses(domain.ProsesBisnis{ID: "PB-SEED-7", Siklus: "Kredit Mekaar", Kode: "7", Nama: "Cadangan Kerugian Penurunan Nilai (CKPN)", PemilikProses: "Divisi MRE",
		SubProses: []domain.SubProses{{Kode: "7.1", Nama: "Perhitungan EIR Bulanan"}, {Kode: "7.2", Nama: "Perhitungan CKPN Bulanan"}, {Kode: "7.3", Nama: "Remodelling"}, {Kode: "7.4", Nama: "Segmentasi Produk"}}})

	m.SaveKontrol(domain.Kontrol{
		Ref: "MKR-31-02", Siklus: "Kredit Mekaar", ProsesKode: []string{"3"}, SubProsesKode: []string{"3.1"}, RefRisiko: []string{"R-MKR-31-02"},
		NamaAktivitas: "Kepala Unit Mekaar (KUM) melakukan verifikasi dan approval di sistem Mekaar Digi Back Office (BO) atas pencatatan penerimaan pembayaran yang telah di approve oleh FAO",
		Deskripsi:     "Setelah FAO melakukan approval di sistem Mekaar Digi Back Office (BO), Kepala Unit Mekaar (KUM) melakukan verifikasi dan approval di sistem Mekaar Digi Back Office (BO).",
		KeyControl:    true, Sifat: "Manual", Frekuensi: domain.FrekLebihDariSekaliSehari,
		FungsiPelaksana: "Kantor Unit Mekaar", PelakuPengendalian: "Kepala Unit Mekaar (KUM)", PeriodeEfektif: "2025-10",
		AtributRCM: map[string]any{"ipo": []string{"A"}, "asersi": []string{"A", "R&O"}, "preventifDetektif": "Preventif", "aplikasi": []string{"Mekaar Digi Back Office"},
			"fsli": []string{"Pinjaman Yang Diberikan", "Pendapatan bunga dari pinjaman - cash basis"}, "penilaianRisiko": "Medium", "risikoFraud": true},
	})
	m.SaveKontrol(domain.Kontrol{
		Ref: "MKR-72-04", Siklus: "Kredit Mekaar", ProsesKode: []string{"7"}, SubProsesKode: []string{"7.2"},
		NamaAktivitas: "Kepala Divisi AMK mereviu dan memberikan persetujuan pada jurnal pencatatan CKPN",
		Deskripsi:     "Kepala Divisi AMK mereviu dan memberikan persetujuan pada jurnal pencatatan CKPN berdasarkan hasil perhitungan CKPN bulanan yang telah direviu.",
		KeyControl:    true, Sifat: "Manual", Frekuensi: domain.FrekBulanan, FungsiPelaksana: "Divisi AMK", PelakuPengendalian: "Kepala Divisi AMK", PeriodeEfektif: "2025-10",
	})
	m.SaveKontrol(domain.Kontrol{
		Ref: "MKR-73-02", Siklus: "Kredit Mekaar", ProsesKode: []string{"7"}, SubProsesKode: []string{"7.3"},
		NamaAktivitas: "Pejabat yang berwenang mereviu laporan hasil backtesting dan memberikan persetujuan",
		Deskripsi:     "Pejabat yang berwenang melakukan reviu atas laporan hasil backtesting dan memberikan persetujuan.",
		KeyControl:    true, Sifat: "Manual", Frekuensi: domain.FrekTahunan, FungsiPelaksana: "Divisi MRE dan EVP Manajemen Risiko", PelakuPengendalian: "Pejabat yang berwenang", PeriodeEfektif: "2025-10",
	})

	m.SaveTemplate(domain.TemplateKontrol{KontrolRef: "MKR-31-02", Versi: 1, BerlakuMulai: "2025-10",
		Atribut: []domain.AtributPengujian{
			{Kode: "A1", Deskripsi: "Pencatatan penerimaan pembayaran sudah di-approve FAO di Mekaar Digi BO sebelum diverifikasi KUM."},
			{Kode: "A2", Deskripsi: "KUM memverifikasi kesesuaian nominal penerimaan dengan rekap pembayaran PKM."},
			{Kode: "A3", Deskripsi: "Approval dilakukan oleh user KUM yang berwenang di Mekaar Digi BO, bukan user lain."},
			{Kode: "A4", Deskripsi: "Approval dilakukan pada hari yang sama dengan pencatatan penerimaan."},
		},
		Dokumen: []domain.DokumenBukti{
			{Nama: "Screenshot bukti approval KUM di Mekaar Digi BO", WajibUnggah: true, Field: []domain.FieldDokumen{{Nama: "Tanggal dokumen", Tipe: domain.FieldTanggal}, {Nama: "ID transaksi", Tipe: domain.FieldTeks}, {Nama: "Nama nasabah", Tipe: domain.FieldTeks}}},
			{Nama: "Rekap pembayaran PKM", WajibUnggah: true, Field: []domain.FieldDokumen{{Nama: "Tanggal PKM", Tipe: domain.FieldTanggal}, {Nama: "Total pembayaran", Tipe: domain.FieldNominal}}},
		}})
	m.SaveTemplate(domain.TemplateKontrol{KontrolRef: "MKR-72-04", Versi: 1, BerlakuMulai: "2025-10",
		Atribut: []domain.AtributPengujian{
			{Kode: "A1", Deskripsi: "Jurnal CKPN disusun dari hasil perhitungan CKPN yang sudah direviu Kepala Divisi MRE."},
			{Kode: "A2", Deskripsi: "Nominal jurnal sesuai dengan laporan hasil perhitungan CKPN bulanan."},
			{Kode: "A3", Deskripsi: "Persetujuan Kepala Divisi AMK diberikan sebelum jurnal diposting."},
		},
		Dokumen: []domain.DokumenBukti{{Nama: "Jurnal CKPN beserta bukti persetujuan", WajibUnggah: true}}})
}
