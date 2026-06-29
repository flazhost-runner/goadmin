// Package router menyediakan named-routes + registry modul (padanan namedRoutes
// & registrasi modul di NodeAdmin). URL dirujuk lewat NAMA (Route("admin.v1.access.user.index"))
// bukan string hardcode → mudah refactor. Tiap nama menyimpan METHOD + path
// (sejajar NodeAdmin yang mendeklarasikan route get/post/put bernama).
package router

import (
	"regexp"
	"strings"
	"sync"
)

// routeInfo = metode HTTP + pola path untuk sebuah named route.
type routeInfo struct {
	Method string
	Path   string
}

// namedRegistry menyimpan pemetaan nama → {method, path}.
type namedRegistry struct {
	mu     sync.RWMutex
	routes map[string]routeInfo
}

var registry = &namedRegistry{routes: map[string]routeInfo{}}

// Register menyimpan nama→{method, path}. Dipanggil saat modul mendaftarkan
// route — LENGKAP DENGAN METHOD (GET/POST/PUT/DELETE), sejajar NodeAdmin.
func Register(method, name, path string) {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.routes[name] = routeInfo{Method: method, Path: path}
}

// Method mengembalikan metode HTTP untuk named route (kosong bila tak ada).
func Method(name string) string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	return registry.routes[name].Method
}

var paramRe = regexp.MustCompile(`:([A-Za-z0-9_]+)`)

// Route mengembalikan URL untuk nama tertentu, mensubstitusi parameter `:id`
// dengan nilai dari params (urutan map tak relevan — substitusi by-key).
//
// Contoh: Route("admin.v1.access.user.edit", map[string]string{"id": "7"})
//         → "/admin/v1/access/user/7/edit"
// Bila nama tak terdaftar, mengembalikan "#" (gagal jelas di UI, bukan panik).
func Route(name string, params ...map[string]string) string {
	registry.mu.RLock()
	info, ok := registry.routes[name]
	registry.mu.RUnlock()
	if !ok {
		return "#"
	}
	path := info.Path
	if len(params) == 0 || len(params[0]) == 0 {
		return path
	}
	p := params[0]
	return paramRe.ReplaceAllStringFunc(path, func(m string) string {
		key := strings.TrimPrefix(m, ":")
		if v, ok := p[key]; ok {
			return v
		}
		return m
	})
}

// Entry = satu named route lengkap (nama+method+path) untuk enumerasi
// lintas-paket — dipakai SyncPermissions menurunkan permission dari route,
// sejajar NodeAdmin yang memindai stack route Express.
type Entry struct {
	Name   string
	Method string
	Path   string
}

// Entries mengembalikan SELURUH named route (snapshot). Urutan tak dijamin.
func Entries() []Entry {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	out := make([]Entry, 0, len(registry.routes))
	for name, info := range registry.routes {
		out = append(out, Entry{Name: name, Method: info.Method, Path: info.Path})
	}
	return out
}

// NameByMethodPath mencari NAMA route dari (method, pola-path) — reverse lookup
// untuk RBAC middleware: menurunkan nama route dari request berjalan
// (padanan NodeAdmin `getNameByPathAndMethod`). "" bila tak terdaftar.
func NameByMethodPath(method, path string) string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	for name, info := range registry.routes {
		if info.Method == method && info.Path == path {
			return name
		}
	}
	return ""
}

// All mengembalikan salinan seluruh named routes (path) — untuk debug/template helper.
func All() map[string]string {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	out := make(map[string]string, len(registry.routes))
	for k, v := range registry.routes {
		out[k] = v.Path
	}
	return out
}

// reset hanya untuk test.
func reset() {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	registry.routes = map[string]routeInfo{}
}
