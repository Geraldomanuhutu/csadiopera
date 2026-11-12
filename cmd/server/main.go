// Server referensi modul CSA ICoFR OPERA.
//
//	go run ./cmd/server            # http://localhost:8080
//	go run ./cmd/server -addr :9090
package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"time"

	"opera/csa/internal/httpapi"
	"opera/csa/internal/seed"
	"opera/csa/internal/service"
	"opera/csa/internal/store"
	"opera/csa/web"
)

type logNotifier struct{}

func (logNotifier) Kirim(ke, judul, isi string) {
	log.Printf("notifikasi → %s | %s: %s", ke, judul, isi)
}

func main() {
	def := ":8080"
	if p := os.Getenv("PORT"); p != "" { // Vercel, Render, Railway, dll. memberi port lewat env PORT
		def = ":" + p
	}
	addr := flag.String("addr", def, "alamat server")
	flag.Parse()

	st := store.NewMemory()
	seed.Isi(st)
	svc := service.New(st, logNotifier{}, time.Now)

	api := httpapi.New(svc)
	root := http.NewServeMux()
	root.Handle("/api", api)
	root.Handle("/api/", api)
	root.Handle("/healthz", api)
	root.Handle("/", http.FileServer(http.FS(web.Static()))) // tampilan demo OPERA

	srv := &http.Server{Addr: *addr, Handler: root, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("CSA ICoFR API jalan di %s (user demo: rina=Lini 2, sari=Lini 1 Cibinong, andi=Lini 1 Citeureup)", *addr)
	log.Fatal(srv.ListenAndServe())
}
