package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"opera/csa/internal/seed"
	"opera/csa/internal/service"
	"opera/csa/internal/store"
)

type client struct {
	t   *testing.T
	srv *httptest.Server
}

func (c client) do(user, method, path string, body any, wantCode int) map[string]any {
	c.t.Helper()
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req, _ := http.NewRequest(method, c.srv.URL+path, rd)
	req.Header.Set("X-User-ID", user)
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		c.t.Fatal(err)
	}
	defer res.Body.Close()
	var out any
	_ = json.NewDecoder(res.Body).Decode(&out)
	if res.StatusCode != wantCode {
		c.t.Fatalf("%s %s sebagai %s: status %d, mau %d, body %v", method, path, user, res.StatusCode, wantCode, out)
	}
	if m, ok := out.(map[string]any); ok {
		return m
	}
	return map[string]any{"list": out}
}

// TestAlurPenuh menjalankan alur yang sama dengan demo web:
// dispatch → Lini 1 isi & submit → Lini 2 kembalikan → Lini 1 submit ulang → Lini 2 setujui → dashboard.
func TestAlurPenuh(t *testing.T) {
	st := store.NewMemory()
	seed.Isi(st)
	clock := time.Date(2026, 11, 12, 8, 30, 0, 0, time.UTC)
	svc := service.New(st, nil, func() time.Time { return clock })
	srv := httptest.NewServer(New(svc))
	defer srv.Close()
	c := client{t, srv}

	// tanpa login
	c.do("", "GET", "/api/me", nil, 401)

	// Lini 1 tidak boleh dispatch
	c.do("sari", "POST", "/api/assessments", map[string]any{"periode": map[string]int{"tahun": 2026, "kuartal": 3}, "kontrolRef": "MKR-31-02", "unitKerja": []string{"Unit Mekaar Cibinong"}, "batasSubmit": "2026-11-14"}, 403)

	// kontrol tahunan tidak masuk lingkup Q3
	r := c.do("rina", "POST", "/api/assessments", map[string]any{"periode": map[string]int{"tahun": 2026, "kuartal": 3}, "kontrolRef": "MKR-73-02", "unitKerja": []string{"Divisi MRE"}, "batasSubmit": "2026-11-14"}, 422)
	if r["pelanggaran"].([]any)[0].(map[string]any)["kode"] != "CSA-02" {
		t.Fatalf("harusnya CSA-02: %v", r)
	}

	// 4. New Assessment: dispatch MKR-31-02 ke dua unit
	r = c.do("rina", "POST", "/api/assessments", map[string]any{
		"periode": map[string]int{"tahun": 2026, "kuartal": 3}, "kontrolRef": "MKR-31-02",
		"unitKerja": []string{"Unit Mekaar Cibinong", "Unit Mekaar Citeureup"}, "perluNasabah": true,
		"batasSubmit": "2026-11-14", "targetValidasi": "2026-11-21",
	}, 201)
	kks := r["kertasKerja"].([]any)
	kkCibinong := kks[0].(map[string]any)["id"].(string)

	// Lini 1 hanya melihat kertas kerja unitnya
	if l := c.do("sari", "GET", "/api/kertas-kerja", nil, 200)["list"].([]any); len(l) != 1 {
		t.Fatalf("Sari harusnya lihat 1 kertas kerja, dapat %d", len(l))
	}
	c.do("andi", "GET", "/api/kertas-kerja/"+kkCibinong, nil, 404)

	// 6. submit tanpa isian ditolak dengan daftar kekurangan
	r = c.do("sari", "POST", "/api/kertas-kerja/"+kkCibinong+"/submit", nil, 422)
	if len(r["pelanggaran"].([]any)) < 4 {
		t.Fatalf("harusnya banyak pelanggaran: %v", r)
	}

	isian := map[string]any{
		"pelaksana":          []map[string]string{{"nama": "Sari Wulandari", "jabatan": "Kepala Unit Mekaar (KUM)"}, {"nama": "Budi Santoso", "jabatan": "FAO"}},
		"adaPerubahanProses": false,
		"jawabanAtribut":     map[string]bool{"A1": true, "A2": true, "A3": true, "A4": true},
		"sampel":             []map[string]any{{"namaNasabah": "Nasabah A", "idTransaksi": "TRX-0931-1102", "tanggal": "2026-10-03"}},
		"bukti": []map[string]string{
			{"dokumen": "Screenshot bukti approval KUM di Mekaar Digi BO", "fileName": "approval_A.png", "fileRef": "csa/kk/approval_A.png"},
			{"dokumen": "Rekap pembayaran PKM", "fileName": "rekap_PKM_okt.pdf", "fileRef": "csa/kk/rekap_PKM_okt.pdf"},
		},
	}
	c.do("sari", "PUT", "/api/kertas-kerja/"+kkCibinong+"/isian", isian, 200)
	r = c.do("sari", "POST", "/api/kertas-kerja/"+kkCibinong+"/submit", nil, 200)
	if r["status"] != "MENUNGGU_VALIDASI" || r["kesimpulan"] != "EFEKTIF" {
		t.Fatalf("setelah submit: %v", r)
	}

	// 8. antrian validasi Lini 2
	if q := c.do("rina", "GET", "/api/antrian-validasi", nil, 200)["list"].([]any); len(q) != 1 {
		t.Fatalf("antrian harusnya 1, dapat %d", len(q))
	}
	c.do("sari", "GET", "/api/antrian-validasi", nil, 403)

	// 10b. kembalikan dengan catatan
	r = c.do("rina", "POST", "/api/kertas-kerja/"+kkCibinong+"/validasi", map[string]any{
		"keputusan": "KEMBALIKAN", "catatan": "Screenshot Nasabah B tanpa tanggal dan user ID KUM.",
	}, 200)
	if r["status"] != "DIKEMBALIKAN" {
		t.Fatalf("status %v", r["status"])
	}

	// 11. Lini 1 submit ulang
	clock = clock.Add(7 * time.Hour)
	r = c.do("sari", "POST", "/api/kertas-kerja/"+kkCibinong+"/submit", nil, 200)
	if r["revisiKe"].(float64) != 1 {
		t.Fatalf("revisi %v", r["revisiKe"])
	}

	// 10a. setujui
	r = c.do("rina", "POST", "/api/kertas-kerja/"+kkCibinong+"/validasi", map[string]any{
		"keputusan": "SETUJUI", "sesuaiAtribut": map[string]bool{"A1": true, "A2": true, "A3": true, "A4": true},
	}, 200)
	if r["status"] != "TERVALIDASI" {
		t.Fatalf("status %v", r["status"])
	}

	// penugasan: 1 dari 2 unit submit
	p := c.do("rina", "GET", "/api/assessments", nil, 200)["list"].([]any)[0].(map[string]any)
	if p["status"] != "IN_PROGRESS" || p["submit"].(float64) != 1 || p["total"].(float64) != 2 {
		t.Fatalf("penugasan %v", p)
	}

	// 12. dashboard
	d := c.do("rina", "GET", "/api/dashboard", nil, 200)
	if d["totalKertasKerja"].(float64) != 2 || d["persenSubmit"].(float64) != 50 {
		t.Fatalf("dashboard %v", d)
	}
}
