package domain

import (
	"fmt"
	"strings"
)

// CekProsesBisnis memeriksa master proses bisnis ICoFR (MST-01 s.d. MST-03).
// existing berisi proses lain di siklus yang sama.
func CekProsesBisnis(p ProsesBisnis, existing []ProsesBisnis) error {
	var es RuleErrors
	if strings.TrimSpace(p.Siklus) == "" || strings.TrimSpace(p.Kode) == "" || strings.TrimSpace(p.Nama) == "" || strings.TrimSpace(p.PemilikProses) == "" {
		es = append(es, rule("MST-01", "siklus, kode, nama, dan pemilik proses wajib diisi", usulan))
	}
	used := map[string]bool{}
	for _, e := range existing {
		if e.ID == p.ID || e.Siklus != p.Siklus {
			continue
		}
		used[e.Kode] = true
		for _, s := range e.SubProses {
			used[s.Kode] = true
		}
	}
	if used[p.Kode] {
		es = append(es, rule("MST-02", "kode proses "+p.Kode+" sudah dipakai di siklus "+p.Siklus, usulan))
	}
	for i, s := range p.SubProses {
		switch {
		case strings.TrimSpace(s.Kode) == "" || strings.TrimSpace(s.Nama) == "":
			es = append(es, rule("MST-03", fmt.Sprintf("kode dan nama sub-proses %d wajib diisi", i+1), usulan))
		case !strings.HasPrefix(s.Kode, p.Kode+"."):
			es = append(es, rule("MST-03", "kode sub-proses "+s.Kode+" harus diawali "+p.Kode+".", usulan))
		case used[s.Kode]:
			es = append(es, rule("MST-03", "kode sub-proses "+s.Kode+" dobel", usulan))
		}
		used[s.Kode] = true
	}
	return es.OrNil()
}

// CekKontrol memeriksa baris RCM (MST-10 s.d. MST-12).
func CekKontrol(k Kontrol, prosesSiklus []ProsesBisnis) error {
	var es RuleErrors
	if strings.TrimSpace(k.Ref) == "" || strings.TrimSpace(k.NamaAktivitas) == "" || strings.TrimSpace(k.Deskripsi) == "" {
		es = append(es, rule("MST-10", "referensi pengendalian, nama aktivitas, dan deskripsi wajib diisi", usulan))
	}
	if len(k.ProsesKode) == 0 {
		es = append(es, rule("MST-11", "pilih minimal satu proses bisnis", usulan))
	}
	// Proses dan sub-proses harus ada di master (bukan teks bebas), supaya data RCM konsisten.
	valid := map[string]bool{}
	for _, p := range prosesSiklus {
		if p.Siklus != k.Siklus {
			continue
		}
		valid[p.Kode] = true
		for _, s := range p.SubProses {
			valid[s.Kode] = true
		}
	}
	for _, c := range append(append([]string{}, k.ProsesKode...), k.SubProsesKode...) {
		if !valid[c] {
			es = append(es, rule("MST-11", "proses/sub-proses "+c+" tidak ada di master siklus "+k.Siklus, usulan))
		}
	}
	if k.Frekuensi == "" || k.FungsiPelaksana == "" || k.PelakuPengendalian == "" {
		es = append(es, rule("MST-12", "frekuensi, fungsi pelaksana, dan pelaku pengendalian wajib diisi", usulan))
	}
	return es.OrNil()
}

// CekTemplate memeriksa template kontrol CSA (MST-20 s.d. MST-21).
func CekTemplate(t TemplateKontrol) error {
	var es RuleErrors
	if len(t.Atribut) == 0 {
		es = append(es, rule("MST-20", "template wajib punya minimal satu atribut pengujian", usulan))
	}
	kode := map[string]bool{}
	for _, a := range t.Atribut {
		if strings.TrimSpace(a.Kode) == "" || strings.TrimSpace(a.Deskripsi) == "" || kode[a.Kode] {
			es = append(es, rule("MST-20", "kode atribut kosong atau dobel: "+a.Kode, usulan))
		}
		kode[a.Kode] = true
	}
	nama := map[string]bool{}
	for i, d := range t.Dokumen {
		if strings.TrimSpace(d.Nama) == "" || nama[d.Nama] {
			es = append(es, rule("MST-21", fmt.Sprintf("nama dokumen %d kosong atau dobel", i+1), usulan))
		}
		nama[d.Nama] = true
	}
	return es.OrNil()
}
