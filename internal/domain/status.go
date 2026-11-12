package domain

// Status kertas kerja CSA per unit kerja.
//
//	DITUGASKAN ──simpan──▶ DRAFT ──submit──▶ MENUNGGU_VALIDASI ──setujui──▶ TERVALIDASI
//	     │                   │                  │   │
//	     └──────submit───────┘                  │   └─kembalikan / ada transaksi──▶ DIKEMBALIKAN ──submit──▶ MENUNGGU_VALIDASI
//	                                            └─tetapkan tidak efektif──▶ TINDAK_LANJUT
//	submit dengan kesimpulan tidak efektif ─────────────────────────────────▶ TINDAK_LANJUT
type Status string

const (
	StatusDitugaskan       Status = "DITUGASKAN"
	StatusDraft            Status = "DRAFT"
	StatusMenungguValidasi Status = "MENUNGGU_VALIDASI"
	StatusDikembalikan     Status = "DIKEMBALIKAN"
	StatusTervalidasi      Status = "TERVALIDASI"
	StatusTindakLanjut     Status = "TINDAK_LANJUT" // tidak efektif, Lini 2 memantau remediasi
)

// BisaDiisiLini1 menyatakan status yang masih boleh diubah isiannya oleh Lini 1.
func (s Status) BisaDiisiLini1() bool {
	return s == StatusDitugaskan || s == StatusDraft || s == StatusDikembalikan
}

// Final menyatakan kertas kerja sudah selesai untuk periode ini.
func (s Status) Final() bool { return s == StatusTervalidasi || s == StatusTindakLanjut }

// SudahSubmit menyatakan Lini 1 sudah menyampaikan kesimpulan (termasuk yang final).
func (s Status) SudahSubmit() bool {
	return s == StatusMenungguValidasi || s.Final()
}

// StatusPenugasan adalah status agregat satu assessment (satu kontrol, banyak unit kerja),
// dipakai di tabel "Kontrol yang sudah di-assign".
type StatusPenugasan string

const (
	PenugasanAssigned   StatusPenugasan = "ASSIGNED"    // belum ada unit yang submit
	PenugasanInProgress StatusPenugasan = "IN_PROGRESS" // sebagian unit sudah submit
	PenugasanSubmitted  StatusPenugasan = "SUBMITTED"   // semua submit, ada yang menunggu validasi
	PenugasanValidated  StatusPenugasan = "VALIDATED"   // semua unit final
)

// RingkasPenugasan menghitung status agregat dari status tiap kertas kerja.
func RingkasPenugasan(statuses []Status) (StatusPenugasan, int, int) {
	submit, final := 0, 0
	for _, s := range statuses {
		if s.SudahSubmit() {
			submit++
		}
		if s.Final() {
			final++
		}
	}
	n := len(statuses)
	switch {
	case n > 0 && final == n:
		return PenugasanValidated, submit, n
	case n > 0 && submit == n:
		return PenugasanSubmitted, submit, n
	case submit > 0:
		return PenugasanInProgress, submit, n
	default:
		return PenugasanAssigned, submit, n
	}
}
