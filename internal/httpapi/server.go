// Package httpapi memetakan use case CSA ke REST API JSON (net/http, Go 1.22+).
package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"opera/csa/internal/domain"
	"opera/csa/internal/service"
)

type Server struct {
	svc *service.CSA
	mux *http.ServeMux
}

func New(svc *service.CSA) *Server {
	s := &Server{svc: svc, mux: http.NewServeMux()}
	s.routes()
	return s
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) { s.mux.ServeHTTP(w, r) }

func (s *Server) routes() {
	s.mux.HandleFunc("GET /{$}", s.beranda)
	s.mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	a := s.auth
	s.mux.Handle("GET /api/me", a(s.me))
	// master
	s.mux.Handle("GET /api/proses-bisnis", a(s.listProses))
	s.mux.Handle("POST /api/proses-bisnis", a(s.simpanProses))
	s.mux.Handle("GET /api/rcm", a(s.listKontrol))
	s.mux.Handle("POST /api/rcm", a(s.simpanKontrol))
	s.mux.Handle("GET /api/rcm/lingkup", a(s.lingkup))
	s.mux.Handle("GET /api/rcm/{ref}", a(s.getKontrol))
	s.mux.Handle("GET /api/rcm/{ref}/template", a(s.getTemplate))
	s.mux.Handle("PUT /api/rcm/{ref}/template", a(s.simpanTemplate))
	// penugasan
	s.mux.Handle("POST /api/assessments", a(s.dispatch))
	s.mux.Handle("GET /api/assessments", a(s.penugasan))
	// kertas kerja
	s.mux.Handle("GET /api/kertas-kerja", a(s.listKK))
	s.mux.Handle("GET /api/kertas-kerja/{id}", a(s.getKK))
	s.mux.Handle("PUT /api/kertas-kerja/{id}/isian", a(s.simpanDraf))
	s.mux.Handle("POST /api/kertas-kerja/{id}/submit", a(s.submit))
	s.mux.Handle("POST /api/kertas-kerja/{id}/validasi", a(s.validasi))
	s.mux.Handle("GET /api/antrian-validasi", a(s.antrian))
	s.mux.Handle("GET /api/dashboard", a(s.dashboard))
}

// ---------- auth ----------

type ctxHandler func(w http.ResponseWriter, r *http.Request, u domain.User)

// auth demo: user dibaca dari header X-User-ID.
// Di OPERA ganti dengan SSO/JWT yang sudah ada, lalu petakan ke domain.User
// (peran Lini 1/Lini 2 dan unit kerja dari Mapping Unit Kerja).
func (s *Server) auth(h ctxHandler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-User-ID")
		if id == "" {
			id = r.URL.Query().Get("as") // DEMO SAJA: supaya bisa dicoba dari browser. Hapus di produksi.
		}
		u, ok := s.svc.User(id)
		if !ok {
			writeJSON(w, http.StatusUnauthorized, errBody("belum login atau user tidak dikenal"))
			return
		}
		h(w, r, u)
	})
}

// ---------- handlers ----------

func (s *Server) me(w http.ResponseWriter, r *http.Request, u domain.User) { writeJSON(w, 200, u) }

func (s *Server) listProses(w http.ResponseWriter, r *http.Request, u domain.User) {
	writeJSON(w, 200, s.svc.ListProses(r.URL.Query().Get("siklus")))
}

func (s *Server) simpanProses(w http.ResponseWriter, r *http.Request, u domain.User) {
	var p domain.ProsesBisnis
	if !decode(w, r, &p) {
		return
	}
	out, err := s.svc.SimpanProses(u, p)
	respond(w, http.StatusCreated, out, err)
}

func (s *Server) listKontrol(w http.ResponseWriter, r *http.Request, u domain.User) {
	writeJSON(w, 200, s.svc.ListKontrol())
}

func (s *Server) simpanKontrol(w http.ResponseWriter, r *http.Request, u domain.User) {
	var k domain.Kontrol
	if !decode(w, r, &k) {
		return
	}
	out, err := s.svc.SimpanKontrol(u, k)
	respond(w, http.StatusCreated, out, err)
}

func (s *Server) lingkup(w http.ResponseWriter, r *http.Request, u domain.User) {
	q, _ := strconv.Atoi(r.URL.Query().Get("kuartal"))
	if q < 1 || q > 4 {
		writeJSON(w, http.StatusBadRequest, errBody("parameter kuartal harus 1-4"))
		return
	}
	writeJSON(w, 200, s.svc.LingkupPeriode(q))
}

func (s *Server) getKontrol(w http.ResponseWriter, r *http.Request, u domain.User) {
	out, err := s.svc.GetKontrol(r.PathValue("ref"))
	respond(w, 200, out, err)
}

func (s *Server) getTemplate(w http.ResponseWriter, r *http.Request, u domain.User) {
	out, err := s.svc.Template(r.PathValue("ref"))
	respond(w, 200, out, err)
}

func (s *Server) simpanTemplate(w http.ResponseWriter, r *http.Request, u domain.User) {
	var t domain.TemplateKontrol
	if !decode(w, r, &t) {
		return
	}
	out, err := s.svc.SimpanTemplate(u, r.PathValue("ref"), t)
	respond(w, 200, out, err)
}

type dispatchReq struct {
	domain.DispatchInput
	BatasSubmit    string `json:"batasSubmit"`    // YYYY-MM-DD
	TargetValidasi string `json:"targetValidasi"` // YYYY-MM-DD
}

func (s *Server) dispatch(w http.ResponseWriter, r *http.Request, u domain.User) {
	var req dispatchReq
	if !decode(w, r, &req) {
		return
	}
	in := req.DispatchInput
	var err error
	if in.BatasSubmit, err = time.Parse("2006-01-02", req.BatasSubmit); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("batasSubmit harus berformat YYYY-MM-DD"))
		return
	}
	if req.TargetValidasi != "" {
		if in.TargetValidasi, err = time.Parse("2006-01-02", req.TargetValidasi); err != nil {
			writeJSON(w, http.StatusBadRequest, errBody("targetValidasi harus berformat YYYY-MM-DD"))
			return
		}
	}
	a, kks, err := s.svc.Dispatch(u, in)
	respond(w, http.StatusCreated, map[string]any{"assessment": a, "kertasKerja": kks}, err)
}

func (s *Server) penugasan(w http.ResponseWriter, r *http.Request, u domain.User) {
	out, err := s.svc.Penugasan(u)
	respond(w, 200, out, err)
}

func (s *Server) listKK(w http.ResponseWriter, r *http.Request, u domain.User) {
	writeJSON(w, 200, s.svc.ListKertasKerja(u, domain.Status(r.URL.Query().Get("status"))))
}

func (s *Server) getKK(w http.ResponseWriter, r *http.Request, u domain.User) {
	kk, a, err := s.svc.GetKertasKerja(u, r.PathValue("id"))
	respond(w, 200, map[string]any{"kertasKerja": kk, "assessment": a}, err)
}

func (s *Server) simpanDraf(w http.ResponseWriter, r *http.Request, u domain.User) {
	var is domain.IsianLini1
	if !decode(w, r, &is) {
		return
	}
	out, err := s.svc.SimpanDraf(u, r.PathValue("id"), is)
	respond(w, 200, out, err)
}

func (s *Server) submit(w http.ResponseWriter, r *http.Request, u domain.User) {
	out, err := s.svc.Submit(u, r.PathValue("id"))
	respond(w, 200, out, err)
}

func (s *Server) validasi(w http.ResponseWriter, r *http.Request, u domain.User) {
	var in domain.ValidasiInput
	if !decode(w, r, &in) {
		return
	}
	out, err := s.svc.Validasi(u, r.PathValue("id"), in)
	respond(w, 200, out, err)
}

func (s *Server) antrian(w http.ResponseWriter, r *http.Request, u domain.User) {
	out, err := s.svc.AntrianValidasi(u)
	respond(w, 200, out, err)
}

func (s *Server) dashboard(w http.ResponseWriter, r *http.Request, u domain.User) {
	out, err := s.svc.Dashboard(u)
	respond(w, 200, out, err)
}

// beranda menampilkan petunjuk singkat saat URL dibuka di browser.
func (s *Server) beranda(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(berandaHTML))
}

const berandaHTML = `<!doctype html><html lang="id"><head><meta charset="utf-8"><meta name="viewport" content="width=device-width, initial-scale=1">
<title>OPERA · CSA ICoFR API</title>
<style>body{font-family:system-ui,sans-serif;max-width:860px;margin:40px auto;padding:0 20px;color:#14202E;line-height:1.55}
h1{font-size:24px;margin:0 0 4px}code{background:#EEF1F5;padding:2px 6px;border-radius:4px}a{color:#0066B2}
table{border-collapse:collapse;width:100%;margin-top:8px}td,th{border-bottom:1px solid #DDE3EA;padding:8px;text-align:left;font-size:14px}</style></head><body>
<h1>OPERA · CSA ICoFR API</h1>
<p>Server referensi modul CSA ICoFR. Ini API JSON tanpa tampilan. Login demo memakai header <code>X-User-ID</code>, atau parameter <code>?as=</code> supaya bisa dicoba dari browser.</p>
<p>User demo: <code>rina</code> (Lini 2), <code>sari</code> (Lini 1, Unit Mekaar Cibinong), <code>andi</code> (Lini 1, Unit Mekaar Citeureup).</p>
<table><tr><th>Coba (GET)</th><th>Isi</th></tr>
<tr><td><a href="/healthz">/healthz</a></td><td>cek server hidup</td></tr>
<tr><td><a href="/api/me?as=rina">/api/me?as=rina</a></td><td>user yang login</td></tr>
<tr><td><a href="/api/proses-bisnis?as=rina">/api/proses-bisnis</a></td><td>master proses bisnis ICoFR</td></tr>
<tr><td><a href="/api/rcm?as=rina">/api/rcm</a></td><td>kontrol RCM</td></tr>
<tr><td><a href="/api/rcm/lingkup?kuartal=3&as=rina">/api/rcm/lingkup?kuartal=3</a></td><td>key control wajib CSA di Q3 (kontrol tahunan tidak ikut)</td></tr>
<tr><td><a href="/api/rcm/MKR-31-02/template?as=rina">/api/rcm/MKR-31-02/template</a></td><td>template kontrol CSA</td></tr>
<tr><td><a href="/api/assessments?as=rina">/api/assessments</a></td><td>penugasan (Lini 2)</td></tr>
<tr><td><a href="/api/kertas-kerja?as=sari">/api/kertas-kerja?as=sari</a></td><td>kertas kerja milik Unit Mekaar Cibinong</td></tr>
<tr><td><a href="/api/antrian-validasi?as=rina">/api/antrian-validasi</a></td><td>antrian validasi Lini 2</td></tr>
<tr><td><a href="/api/dashboard?as=rina">/api/dashboard</a></td><td>ringkasan dashboard</td></tr></table>
<p>Aksi (dispatch, simpan isian, submit, validasi) memakai POST/PUT; contoh lengkap ada di README repository.</p>
</body></html>`

// ---------- helper ----------

func errBody(msg string) map[string]any { return map[string]any{"error": msg} }

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		writeJSON(w, http.StatusBadRequest, errBody("body JSON tidak valid: "+err.Error()))
		return false
	}
	return true
}

// respond memetakan error domain ke status HTTP:
// pelanggaran aturan 422, tidak berwenang 403, tidak ditemukan 404.
func respond(w http.ResponseWriter, okCode int, v any, err error) {
	var res domain.RuleErrors
	switch {
	case err == nil:
		writeJSON(w, okCode, v)
	case errors.As(err, &res):
		writeJSON(w, http.StatusUnprocessableEntity, map[string]any{"error": "aturan bisnis tidak terpenuhi", "pelanggaran": res})
	case errors.Is(err, domain.ErrTidakBerwenang):
		writeJSON(w, http.StatusForbidden, errBody(err.Error()))
	case errors.Is(err, domain.ErrTidakDitemukan):
		writeJSON(w, http.StatusNotFound, errBody(err.Error()))
	default:
		writeJSON(w, http.StatusInternalServerError, errBody("kesalahan server"))
	}
}
