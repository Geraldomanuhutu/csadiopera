// Package store berisi implementasi penyimpanan in-memory untuk demo dan test.
// Di OPERA, ganti dengan implementasi database yang memenuhi service.Store
// (skema acuan: migrations/001_init.sql).
package store

import (
	"encoding/json"
	"sort"
	"sync"

	"opera/csa/internal/domain"
)

type Memory struct {
	mu          sync.RWMutex
	users       map[string]domain.User
	proses      map[string]domain.ProsesBisnis
	kontrol     map[string]domain.Kontrol
	template    map[string][]domain.TemplateKontrol // per kontrol, urut versi
	assessment  map[string]domain.Assessment
	kertasKerja map[string]domain.KertasKerja
}

func NewMemory() *Memory {
	return &Memory{
		users:       map[string]domain.User{},
		proses:      map[string]domain.ProsesBisnis{},
		kontrol:     map[string]domain.Kontrol{},
		template:    map[string][]domain.TemplateKontrol{},
		assessment:  map[string]domain.Assessment{},
		kertasKerja: map[string]domain.KertasKerja{},
	}
}

// clone menyalin dalam supaya pemanggil tidak mengubah data tersimpan lewat slice/map bersama.
func clone[T any](v T) T {
	b, _ := json.Marshal(v)
	var out T
	_ = json.Unmarshal(b, &out)
	return out
}

func (m *Memory) SaveUser(u domain.User) { m.mu.Lock(); m.users[u.ID] = u; m.mu.Unlock() }
func (m *Memory) GetUser(id string) (domain.User, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	u, ok := m.users[id]
	return u, ok
}

func (m *Memory) SaveProses(p domain.ProsesBisnis) {
	m.mu.Lock()
	m.proses[p.ID] = clone(p)
	m.mu.Unlock()
}
func (m *Memory) ListProses() []domain.ProsesBisnis {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.ProsesBisnis, 0, len(m.proses))
	for _, p := range m.proses {
		out = append(out, clone(p))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Siklus != out[j].Siklus {
			return out[i].Siklus < out[j].Siklus
		}
		return out[i].Kode < out[j].Kode
	})
	return out
}

func (m *Memory) SaveKontrol(k domain.Kontrol) {
	m.mu.Lock()
	m.kontrol[k.Ref] = clone(k)
	m.mu.Unlock()
}
func (m *Memory) GetKontrol(ref string) (domain.Kontrol, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.kontrol[ref]
	return clone(k), ok
}
func (m *Memory) ListKontrol() []domain.Kontrol {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Kontrol, 0, len(m.kontrol))
	for _, k := range m.kontrol {
		out = append(out, clone(k))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Ref < out[j].Ref })
	return out
}

func (m *Memory) SaveTemplate(t domain.TemplateKontrol) {
	m.mu.Lock()
	m.template[t.KontrolRef] = append(m.template[t.KontrolRef], clone(t))
	m.mu.Unlock()
}
func (m *Memory) LatestTemplate(ref string) (domain.TemplateKontrol, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ts := m.template[ref]
	if len(ts) == 0 {
		return domain.TemplateKontrol{}, false
	}
	return clone(ts[len(ts)-1]), true
}

func (m *Memory) SaveAssessment(a domain.Assessment) {
	m.mu.Lock()
	m.assessment[a.ID] = clone(a)
	m.mu.Unlock()
}
func (m *Memory) GetAssessment(id string) (domain.Assessment, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	a, ok := m.assessment[id]
	return clone(a), ok
}
func (m *Memory) ListAssessments() []domain.Assessment {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.Assessment, 0, len(m.assessment))
	for _, a := range m.assessment {
		out = append(out, clone(a))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].DibuatPada.After(out[j].DibuatPada) })
	return out
}

func (m *Memory) SaveKertasKerja(k domain.KertasKerja) {
	m.mu.Lock()
	m.kertasKerja[k.ID] = clone(k)
	m.mu.Unlock()
}
func (m *Memory) GetKertasKerja(id string) (domain.KertasKerja, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	k, ok := m.kertasKerja[id]
	return clone(k), ok
}
func (m *Memory) ListKertasKerja() []domain.KertasKerja {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]domain.KertasKerja, 0, len(m.kertasKerja))
	for _, k := range m.kertasKerja {
		out = append(out, clone(k))
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
