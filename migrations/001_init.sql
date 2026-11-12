-- Skema acuan modul CSA ICoFR OPERA (PostgreSQL 14+).
-- Sesuaikan nama tabel, tipe ID, dan relasi ke tabel user / unit kerja OPERA yang sudah ada.

CREATE TYPE peran_user        AS ENUM ('LINI_1', 'LINI_2');
CREATE TYPE frekuensi_kontrol AS ENUM ('LEBIH_DARI_SEKALI_SEHARI','HARIAN','MINGGUAN','BULANAN','TRIWULANAN','SEMESTERAN','TAHUNAN','AD_HOC');
CREATE TYPE tipe_field        AS ENUM ('TEKS','TANGGAL','ANGKA','NOMINAL');
CREATE TYPE status_kk         AS ENUM ('DITUGASKAN','DRAFT','MENUNGGU_VALIDASI','DIKEMBALIKAN','TERVALIDASI','TINDAK_LANJUT');
CREATE TYPE kesimpulan_csa    AS ENUM ('EFEKTIF','TIDAK_EFEKTIF','TIDAK_ADA_TRANSAKSI');
CREATE TYPE keputusan_l2      AS ENUM ('SETUJUI','KEMBALIKAN','TETAPKAN_TIDAK_EFEKTIF','KONFIRMASI_TIDAK_ADA_TRANSAKSI','ADA_TRANSAKSI');

-- ---------- Master ----------

CREATE TABLE proses_bisnis (
    id             BIGSERIAL PRIMARY KEY,
    siklus         TEXT NOT NULL,          -- FK ke master_siklus bila ada
    kode           TEXT NOT NULL,
    nama           TEXT NOT NULL,
    pemilik_proses TEXT NOT NULL,          -- FK ke unit kerja (divisi)
    UNIQUE (siklus, kode)                  -- MST-02
);

CREATE TABLE sub_proses (
    id        BIGSERIAL PRIMARY KEY,
    proses_id BIGINT NOT NULL REFERENCES proses_bisnis(id) ON DELETE CASCADE,
    kode      TEXT NOT NULL,               -- harus diawali kode proses + '.' (MST-03, dicek di aplikasi)
    nama      TEXT NOT NULL,
    UNIQUE (proses_id, kode)
);

CREATE TABLE kontrol (                     -- satu baris RCM ICoFR
    ref                 TEXT PRIMARY KEY,  -- referensi pengendalian, contoh MKR-31-02
    siklus              TEXT NOT NULL,
    nama_aktivitas      TEXT NOT NULL,
    deskripsi           TEXT NOT NULL,
    key_control         BOOLEAN NOT NULL,
    sifat               TEXT NOT NULL,     -- Manual / Otomatis / ITDM - EUC / ITDM - IPE / MRC
    frekuensi           frekuensi_kontrol NOT NULL,
    fungsi_pelaksana    TEXT NOT NULL,
    pelaku_pengendalian TEXT NOT NULL,
    periode_efektif     CHAR(7) NOT NULL,  -- YYYY-MM, dasar versi template
    atribut_rcm         JSONB NOT NULL DEFAULT '{}'  -- IPO, asersi, aplikasi, FSLI, penilaian risiko, dll.
);

-- Relasi banyak-ke-banyak (sel RCM yang berisi lebih dari satu nilai disimpan sebagai baris, bukan teks gabungan)
CREATE TABLE kontrol_proses     (kontrol_ref TEXT REFERENCES kontrol(ref) ON DELETE CASCADE, proses_id BIGINT REFERENCES proses_bisnis(id), PRIMARY KEY (kontrol_ref, proses_id));
CREATE TABLE kontrol_sub_proses (kontrol_ref TEXT REFERENCES kontrol(ref) ON DELETE CASCADE, sub_proses_id BIGINT REFERENCES sub_proses(id), PRIMARY KEY (kontrol_ref, sub_proses_id));
CREATE TABLE kontrol_risiko     (kontrol_ref TEXT REFERENCES kontrol(ref) ON DELETE CASCADE, ref_risiko TEXT NOT NULL, PRIMARY KEY (kontrol_ref, ref_risiko));

CREATE TABLE template_kontrol (
    id            BIGSERIAL PRIMARY KEY,
    kontrol_ref   TEXT NOT NULL REFERENCES kontrol(ref),
    versi         INT  NOT NULL,
    berlaku_mulai CHAR(7) NOT NULL,        -- diambil dari kontrol.periode_efektif
    dibuat_oleh   TEXT NOT NULL,
    dibuat_pada   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (kontrol_ref, versi)
);

CREATE TABLE template_atribut (
    template_id BIGINT NOT NULL REFERENCES template_kontrol(id) ON DELETE CASCADE,
    kode        TEXT NOT NULL,             -- A1, A2, ...
    deskripsi   TEXT NOT NULL,
    urutan      INT  NOT NULL,
    PRIMARY KEY (template_id, kode)
);

CREATE TABLE template_dokumen (
    id           BIGSERIAL PRIMARY KEY,
    template_id  BIGINT NOT NULL REFERENCES template_kontrol(id) ON DELETE CASCADE,
    nama         TEXT NOT NULL,
    wajib_unggah BOOLEAN NOT NULL DEFAULT TRUE,
    urutan       INT NOT NULL,
    UNIQUE (template_id, nama)
);

CREATE TABLE template_dokumen_field (
    dokumen_id BIGINT NOT NULL REFERENCES template_dokumen(id) ON DELETE CASCADE,
    nama       TEXT NOT NULL,
    tipe       tipe_field NOT NULL,
    urutan     INT NOT NULL,
    PRIMARY KEY (dokumen_id, nama)
);

-- ---------- Penugasan ----------

CREATE TABLE assessment (
    id              BIGSERIAL PRIMARY KEY,
    tahun           INT NOT NULL,
    kuartal         SMALLINT NOT NULL CHECK (kuartal BETWEEN 1 AND 4),
    kontrol_ref     TEXT NOT NULL REFERENCES kontrol(ref),
    template_id     BIGINT NOT NULL REFERENCES template_kontrol(id),  -- versi dikunci saat dispatch
    perlu_nasabah   BOOLEAN NOT NULL DEFAULT FALSE,
    ada_perubahan   BOOLEAN NOT NULL DEFAULT FALSE,
    batas_submit    DATE NOT NULL,
    target_validasi DATE,
    catatan         TEXT,
    dibuat_oleh     TEXT NOT NULL,
    dibuat_pada     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE kertas_kerja (
    id                   BIGSERIAL PRIMARY KEY,
    assessment_id        BIGINT NOT NULL REFERENCES assessment(id),
    unit_kerja           TEXT NOT NULL,     -- FK ke unit kerja OPERA
    status               status_kk NOT NULL DEFAULT 'DITUGASKAN',
    kesimpulan           kesimpulan_csa,
    ada_perubahan_proses BOOLEAN,
    tidak_ada_transaksi  BOOLEAN NOT NULL DEFAULT FALSE,
    alasan_tidak_efektif TEXT,
    tidak_efektif_sejak  CHAR(7),
    remediasi_rencana    TEXT,
    remediasi_deadline   DATE,
    remediasi_pic        TEXT,
    revisi_ke            INT NOT NULL DEFAULT 0,
    submit_pada          TIMESTAMPTZ,
    terlambat            BOOLEAN NOT NULL DEFAULT FALSE,
    versi_baris          INT NOT NULL DEFAULT 1,  -- optimistic locking: UPDATE ... WHERE versi_baris = $n
    UNIQUE (assessment_id, unit_kerja)
);
CREATE INDEX kertas_kerja_status_idx ON kertas_kerja (status);
CREATE INDEX kertas_kerja_unit_idx   ON kertas_kerja (unit_kerja);

CREATE TABLE kk_pelaksana (
    kertas_kerja_id BIGINT NOT NULL REFERENCES kertas_kerja(id) ON DELETE CASCADE,
    nama            TEXT NOT NULL,
    jabatan         TEXT NOT NULL,
    urutan          INT NOT NULL
);

CREATE TABLE kk_jawaban_atribut (
    kertas_kerja_id BIGINT NOT NULL REFERENCES kertas_kerja(id) ON DELETE CASCADE,
    kode_atribut    TEXT NOT NULL,
    jawaban         BOOLEAN NOT NULL,      -- TRUE = Ya, FALSE = Tidak
    PRIMARY KEY (kertas_kerja_id, kode_atribut)
);

CREATE TABLE kk_sampel (
    id              BIGSERIAL PRIMARY KEY,
    kertas_kerja_id BIGINT NOT NULL REFERENCES kertas_kerja(id) ON DELETE CASCADE,
    nama_nasabah    TEXT,
    id_transaksi    TEXT NOT NULL,
    tanggal         DATE NOT NULL,
    field_dokumen   JSONB NOT NULL DEFAULT '{}'
);

CREATE TABLE kk_bukti (
    id              BIGSERIAL PRIMARY KEY,
    kertas_kerja_id BIGINT NOT NULL REFERENCES kertas_kerja(id) ON DELETE CASCADE,
    dokumen         TEXT NOT NULL,         -- nama dokumen di template
    file_name       TEXT NOT NULL,
    file_ref        TEXT NOT NULL,         -- kunci object storage
    diunggah_pada   TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ---------- Validasi dan jejak audit ----------

CREATE TABLE kk_validasi (
    id              BIGSERIAL PRIMARY KEY,
    kertas_kerja_id BIGINT NOT NULL REFERENCES kertas_kerja(id),
    validator       TEXT NOT NULL,
    pada            TIMESTAMPTZ NOT NULL DEFAULT now(),
    revisi_ke       INT NOT NULL,
    sesuai_atribut  JSONB,                 -- {"A1": true, "A2": false}
    keputusan       keputusan_l2 NOT NULL,
    catatan         TEXT
);

CREATE TABLE kk_riwayat (                  -- append-only, jangan di-UPDATE/DELETE
    id              BIGSERIAL PRIMARY KEY,
    kertas_kerja_id BIGINT NOT NULL REFERENCES kertas_kerja(id),
    pada            TIMESTAMPTZ NOT NULL DEFAULT now(),
    oleh            TEXT NOT NULL,
    aksi            TEXT NOT NULL,
    dari            status_kk,
    ke              status_kk NOT NULL,
    catatan         TEXT
);
