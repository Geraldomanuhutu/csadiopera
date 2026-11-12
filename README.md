# OPERA · Modul CSA ICoFR (reference implementation, Go)

Kode acuan untuk alur Control Self Assessment (CSA) ICoFR di OPERA, sama dengan demo web dan slide alur pengerjaan. Tujuannya supaya logika bisnisnya jelas dan bisa diuji, lalu disesuaikan vendor ke codebase dan stack OPERA.

- Go 1.22+, **hanya standard library** (net/http, encoding/json). Tidak ada dependensi luar.
- Penyimpanan in-memory supaya bisa langsung dijalankan. Skema database acuan ada di `migrations/001_init.sql` (PostgreSQL).
- Login masih demo (header `X-User-ID`). Di OPERA diganti SSO/JWT yang sudah ada.

## Jalankan

```bash
go test ./...          # aturan bisnis + alur end-to-end
go run ./cmd/server    # API di http://localhost:8080
```

User demo: `rina` (Lini 2, Divisi MRO), `sari` (Lini 1, Unit Mekaar Cibinong), `andi` (Lini 1, Unit Mekaar Citeureup), `amk` (Lini 1, Divisi AMK).

## Deploy

- **GitHub**: push folder ini apa adanya (sebaiknya repo private).
- **Vercel**: Import repo → Framework Preset **Go** (terdeteksi dari `go.mod` dan `cmd/server/main.go`). Server membaca env `PORT`.
  Catatan: data disimpan di memori, jadi bisa hilang setiap kali instance baru jalan atau tidur. Cukup untuk coba API, bukan untuk dipakai bersama. Untuk data tetap, sambungkan database (implementasi `service.Store` dengan skema `migrations/001_init.sql`).

## Struktur

```
cmd/server/          main, pasang store + service + HTTP
internal/domain/     entitas, status, aturan bisnis (tanpa dependensi, bisa dipindah utuh)
internal/service/    use case + otorisasi per peran, interface Store dan Notifier
internal/store/      Store in-memory (ganti dengan repository database)
internal/httpapi/    REST API JSON + test alur end-to-end
internal/seed/       data contoh (MKR-31-02, MKR-72-04, MKR-73-02 dari RCM Kredit Mekaar)
migrations/          skema PostgreSQL acuan
```

Yang paling penting dibaca vendor: `internal/domain/rules.go` (semua aturan CSA) dan `internal/domain/status.go` (alur status).

## Alur status kertas kerja

```
DITUGASKAN ─simpan─▶ DRAFT ─submit─▶ MENUNGGU_VALIDASI ─setujui──────────────▶ TERVALIDASI
                                      │      │
                                      │      └─kembalikan / ada transaksi─▶ DIKEMBALIKAN ─submit ulang─▶ MENUNGGU_VALIDASI
                                      └─tetapkan tidak efektif─▶ TINDAK_LANJUT
submit dengan kesimpulan tidak efektif ─────────────────────────▶ TINDAK_LANJUT
```

Status agregat per kontrol (tabel "Kontrol yang sudah di-assign"): `ASSIGNED` (belum ada unit submit), `IN_PROGRESS` (sebagian), `SUBMITTED` (semua submit, ada yang menunggu validasi), `VALIDATED` (semua final).

## Aturan bisnis

Setiap pelanggaran dikembalikan API sebagai HTTP 422 dengan kode di bawah, supaya UI bisa menampilkan semua kekurangan sekaligus. Dasar "PK-0038" = Pedoman ICoFR PNM/PK-0038 Rev. 0.1. Aturan bertanda **[Usulan]** adalah keputusan desain OPERA, bukan bunyi pedoman.

| Kode | Aturan | Dasar |
|---|---|---|
| CSA-01 | CSA hanya untuk key control | PK-0038 Bab V.B.2 |
| CSA-02 | Lingkup per kuartal: Q1 dan Q3 semua key control kecuali tahunan; Q2 kontrol semesteran; Q4 semesteran dan tahunan; kontrol yang berubah wajib di kuartal mana pun | PK-0038 Bab V.B.2 (lihat catatan interpretasi) |
| CSA-03 | Dispatch butuh template kontrol dengan minimal satu atribut | [Usulan] |
| CSA-04 | Minimal satu unit kerja, tidak dobel | [Usulan] |
| CSA-05 | Batas submit tidak sebelum hari ini; target validasi tidak sebelum batas submit | [Usulan] |
| CSA-10 | Hanya Lini 1 di unit kerja terkait yang boleh mengisi dan submit | PK-0038 Bab V.B.1 |
| CSA-11 | Isian hanya bisa diubah saat DITUGASKAN, DRAFT, atau DIKEMBALIKAN | [Usulan] |
| CSA-12 | Pertanyaan "ada perubahan proses/kontrol?" wajib dijawab | PK-0038 Bab V.B.2 huruf a |
| CSA-13 | Minimal satu pelaksana kontrol (nama dan jabatan) | [Usulan] |
| CSA-14 | Semua atribut pengujian dijawab Ya/Tidak | PK-0038 Bab V.B.2 huruf c |
| CSA-15 | Minimal 1 sampel transaksi (ID dan tanggal); nama nasabah wajib bila assessment "perlu nama nasabah" | PK-0038 Bab V.B.2 huruf b dan d |
| CSA-16 | Bukti wajib diunggah untuk dokumen yang ditandai wajib di template | PK-0038 Bab V.B.2 huruf b |
| CSA-17 | Tidak efektif wajib mencantumkan alasan dan periode mulai tidak efektif | PK-0038 Bab V.B.1 huruf b |
| CSA-18 | Tidak efektif wajib berisi rencana remediasi, deadline, dan PIC | PK-0038 Bab V.B.2 (koordinasi perbaikan); rincian field [Usulan] |
| CSA-19 | Tidak ada transaksi: atribut dan sampel tidak diisi | PK-0038 Bab V.B.1 huruf c |
| CSA-20 | Kesimpulan diturunkan dari jawaban: semua Ya = efektif, ada Tidak = tidak efektif | [Usulan] |
| CSA-21 | Tidak efektif tidak divalidasi, langsung TINDAK_LANJUT untuk dipantau Lini 2 | PK-0038 Bab V.B.3.a.2 |
| CSA-30 | Validasi hanya oleh Lini 2 | PK-0038 Bab V.B.1 |
| CSA-31 | Hanya kertas kerja MENUNGGU_VALIDASI yang bisa divalidasi | [Usulan] |
| CSA-32 | Kesimpulan efektif bisa disetujui hanya jika semua atribut dinilai sesuai dengan bukti | PK-0038 Bab V.B.3.a.1 huruf a dan b |
| CSA-33 | Mengembalikan ke Lini 1 wajib dengan catatan | [Usulan] |
| CSA-34 | Lini 2 bisa menetapkan tidak efektif (wajib catatan), status jadi TINDAK_LANJUT | PK-0038 Bab V.B.3.a.1 huruf d |
| CSA-35 | Tidak ada transaksi: Lini 2 memvalidasi populasi; bila ternyata ada transaksi, dikembalikan untuk diisi | PK-0038 Bab V.B.3.a.3 |
| CSA-36 | Setiap perubahan status tercatat di riwayat (append-only) | [Usulan] |
| MST-01..03 | Proses bisnis: field wajib, kode unik per siklus, kode sub-proses diawali kode proses | [Usulan] |
| MST-10..12 | RCM: field wajib, proses/sub-proses harus dari master (bukan teks bebas) | [Usulan] |
| MST-20..21 | Template: minimal satu atribut, kode atribut dan nama dokumen tidak dobel | [Usulan] |

Aturan lain yang sudah tertanam:
- Template berversi. Saat dispatch, versi template dikunci di assessment, jadi perubahan template tidak mengubah kertas kerja yang sedang berjalan. "Berlaku mulai" diambil dari periode efektif RCM.
- Submit lewat batas tetap diterima tapi ditandai `terlambat`.
- Lini 1 tidak bisa melihat kertas kerja unit lain (API mengembalikan 404, bukan 403, supaya keberadaan data tidak bocor).

**Catatan interpretasi CSA-02.** Pedoman menyebut CSA semesteran di kuartal 1 dan 3 untuk setiap key control, ditambah kuartal 2 dan 4 untuk kontrol semesteran, tahunan (kuartal 4), dan yang berubah. Kode ini menganggap kontrol tahunan hanya di kuartal 4, sehingga 3 kontrol tahunan RCM Kredit Mekaar tidak masuk Q3 (51 dari 54 key control). Perlu dikonfirmasi fungsi ICoFR.

## API

Semua request memakai header `X-User-ID`. Body dan respons JSON.

| Method | Path | Peran | Keterangan |
|---|---|---|---|
| GET | `/api/me` | semua | user yang login |
| GET | `/api/proses-bisnis?siklus=` | semua | master proses bisnis |
| POST | `/api/proses-bisnis` | Lini 2 | tambah/ubah proses (+ sub-proses) |
| GET | `/api/rcm` | semua | daftar kontrol RCM |
| POST | `/api/rcm` | Lini 2 | tambah/ubah kontrol |
| GET | `/api/rcm/lingkup?kuartal=3` | semua | key control yang wajib di-CSA kuartal itu |
| GET | `/api/rcm/{ref}` | semua | detail kontrol |
| GET/PUT | `/api/rcm/{ref}/template` | Lini 2 untuk PUT | template kontrol CSA (PUT = versi baru) |
| POST | `/api/assessments` | Lini 2 | New Assessment, buat kertas kerja per unit |
| GET | `/api/assessments` | Lini 2 | penugasan + status agregat per unit |
| GET | `/api/kertas-kerja?status=` | semua | Lini 1 hanya unitnya |
| GET | `/api/kertas-kerja/{id}` | semua | kertas kerja + assessment (template) |
| PUT | `/api/kertas-kerja/{id}/isian` | Lini 1 | simpan draf |
| POST | `/api/kertas-kerja/{id}/submit` | Lini 1 | submit / submit ulang |
| GET | `/api/antrian-validasi` | Lini 2 | urut dari submit paling lama |
| POST | `/api/kertas-kerja/{id}/validasi` | Lini 2 | keputusan validasi |
| GET | `/api/dashboard` | Lini 2 | ringkasan status, % submit, terlambat, tidak efektif |

Contoh alur (sama dengan `internal/httpapi/flow_test.go`):

```bash
# Lini 2: New Assessment
curl -s -X POST localhost:8080/api/assessments -H 'X-User-ID: rina' -d '{
  "periode": {"tahun": 2026, "kuartal": 3}, "kontrolRef": "MKR-31-02",
  "unitKerja": ["Unit Mekaar Cibinong", "Unit Mekaar Citeureup"], "perluNasabah": true,
  "batasSubmit": "2026-12-14", "targetValidasi": "2026-12-21"}'   # batas submit tidak boleh sebelum hari ini (CSA-05)

# Lini 1: simpan isian lalu submit (ganti KK-0002 dengan id dari respons di atas)
curl -s -X PUT localhost:8080/api/kertas-kerja/KK-0002/isian -H 'X-User-ID: sari' -d '{
  "pelaksana": [{"nama": "Sari Wulandari", "jabatan": "Kepala Unit Mekaar (KUM)"}],
  "adaPerubahanProses": false,
  "jawabanAtribut": {"A1": true, "A2": true, "A3": true, "A4": true},
  "sampel": [{"namaNasabah": "Nasabah A", "idTransaksi": "TRX-0931-1102", "tanggal": "2026-10-03"}],
  "bukti": [{"dokumen": "Screenshot bukti approval KUM di Mekaar Digi BO", "fileName": "a.png", "fileRef": "csa/a.png"},
            {"dokumen": "Rekap pembayaran PKM", "fileName": "rekap.pdf", "fileRef": "csa/rekap.pdf"}]}'
curl -s -X POST localhost:8080/api/kertas-kerja/KK-0002/submit -H 'X-User-ID: sari'

# Lini 2: kembalikan atau setujui
curl -s -X POST localhost:8080/api/kertas-kerja/KK-0002/validasi -H 'X-User-ID: rina' -d '{
  "keputusan": "SETUJUI", "sesuaiAtribut": {"A1": true, "A2": true, "A3": true, "A4": true}}'
```

## Yang perlu disesuaikan vendor

1. **Auth**: ganti middleware `auth` di `internal/httpapi/server.go` dengan SSO/JWT OPERA; peran dan unit kerja diambil dari Mapping Unit Kerja.
2. **Database**: implementasikan `service.Store` dengan skema `migrations/001_init.sql`. Bungkus tiap use case dalam transaksi dan pakai `versi_baris` untuk optimistic locking.
3. **Upload bukti**: `Bukti.FileRef` diasumsikan kunci object storage dari layanan upload OPERA; validasi tipe dan ukuran file di sana.
4. **Notifikasi**: implementasikan `service.Notifier` (in-app/email). Titik kirimnya sudah ada: dispatch, submit, tidak efektif, dikembalikan.
5. **Framework HTTP**: handler hanya memanggil service, jadi mudah dipindah ke Gin/Echo/Fiber.
6. **Laporan evaluasi CSA kuartalan Lini 2** (PK-0038 Bab V.B.3.b) belum dibuat; datanya sudah tersedia dari status, kesimpulan, validasi, dan riwayat.
