package domain

import (
	"errors"
	"strings"
)

// RuleError adalah pelanggaran satu aturan bisnis. Kode dirujuk di README
// (misal CSA-14) dan Dasar menunjuk pasal pedoman atau ditandai [Usulan].
type RuleError struct {
	Kode  string `json:"kode"`
	Pesan string `json:"pesan"`
	Dasar string `json:"dasar,omitempty"`
}

func (e RuleError) Error() string { return e.Kode + ": " + e.Pesan }

// RuleErrors mengumpulkan beberapa pelanggaran sekaligus supaya UI bisa
// menampilkan semua field yang belum lengkap dalam satu kali submit.
type RuleErrors []RuleError

func (es RuleErrors) Error() string {
	parts := make([]string, len(es))
	for i, e := range es {
		parts[i] = e.Error()
	}
	return strings.Join(parts, "; ")
}

func (es RuleErrors) OrNil() error {
	if len(es) == 0 {
		return nil
	}
	return es
}

// Error umum untuk lapisan service/HTTP.
var (
	ErrTidakDitemukan = errors.New("data tidak ditemukan")
	ErrTidakBerwenang = errors.New("tidak berwenang")
)

func rule(kode, pesan, dasar string) RuleError {
	return RuleError{Kode: kode, Pesan: pesan, Dasar: dasar}
}

// Dasar pedoman yang sering dipakai.
const (
	dasarVB1   = "PK-0038 Rev. 0.1 Bab V.B.1"
	dasarVB2   = "PK-0038 Rev. 0.1 Bab V.B.2"
	dasarVB3a1 = "PK-0038 Rev. 0.1 Bab V.B.3.a.1"
	dasarVB3a2 = "PK-0038 Rev. 0.1 Bab V.B.3.a.2"
	dasarVB3a3 = "PK-0038 Rev. 0.1 Bab V.B.3.a.3"
	usulan     = "[Usulan] desain OPERA"
)
