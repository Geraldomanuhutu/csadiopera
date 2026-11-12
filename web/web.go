// Package web menyematkan tampilan demo OPERA (HTML/JS statis) ke dalam binary.
// Catatan: layar demo masih memakai data contoh tetap, belum memanggil API.
package web

import (
	"embed"
	"io/fs"
)

//go:embed static
var files embed.FS

// Static mengembalikan isi folder static sebagai root file server.
func Static() fs.FS {
	sub, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}
	return sub
}
