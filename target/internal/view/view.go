// Package view menyediakan helper render template HTML terpusat (padanan
// renderView() di NodeAdmin). Controller web WAJIB lewat RenderView — bukan
// memanggil c.HTML dengan path mentah (di-enforce checker).
//
// Template di-load sekali saat start (ParseGlob), memetakan layout + partial
// (head/sidebar/topbar/foot) + view modul. Helper Route() & hasAccess di-inject
// sebagai FuncMap agar named-routes & sidebar dinamis tersedia di template.
package view

import (
	"html/template"
	"net/http"
	"reflect"
	"strconv"

	"github.com/gin-gonic/gin"

	"goadmin/internal/router"
)

// Engine membungkus template ter-parse + dipasang ke gin.
type Engine struct {
	tmpl *template.Template
}

// FuncMap = fungsi template default (dipakai Load + test view).
func FuncMap() template.FuncMap {
	return template.FuncMap{
		// route("nama", "id", "7") → URL bernama (named-routes di template).
		"route": func(name string, pairs ...string) string {
			params := map[string]string{}
			for i := 0; i+1 < len(pairs); i += 2 {
				params[pairs[i]] = pairs[i+1]
			}
			return router.Route(name, params)
		},
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		// safeHTML menandai string sbg HTML tepercaya (TIDAK di-escape). Pakai
		// HANYA untuk konten yang sudah disanitasi (mis. deskripsi Setting via
		// helpers.CleanRichText) — agar rich-text editor terender di landing.
		"safeHTML": func(s string) template.HTML { return template.HTML(s) }, //nolint:gosec // input sudah disanitasi server

		// pageURL("q_name=x&", 2) → ?q_name=x&q_page=2 sebagai template.URL
		// (tepercaya) agar html/template TIDAK meng-escape '='/'&' di qbase.
		"pageURL": func(qbase string, page int) template.URL {
			return template.URL("?" + qbase + "q_page=" + strconv.Itoa(page)) //nolint:gosec // qbase dirakit server dari url.Values.Encode()
		},
		// seq 1 n → []int{1..n} (untuk loop pagination).
		"seq": func(from, to int) []int {
			if to < from {
				return nil
			}
			out := make([]int, 0, to-from+1)
			for i := from; i <= to; i++ {
				out = append(out, i)
			}
			return out
		},
		// hasAccess(currentUser, "nama.route", "GET") → bool. Gating UI berbasis
		// permission route-driven (padanan hasAccess(name, method) NodeAdmin di
		// sidebar.ejs). nil-aman (user belum login → false). Administrator bypass
		// (User.HasAccess true). Memakai interface agar view tak meng-import model.
		"hasAccess": func(user any, name, method string) bool {
			if user == nil {
				return false
			}
			// Guard typed-nil pointer (mis. (*User)(nil)) agar tak panik.
			if rv := reflect.ValueOf(user); rv.Kind() == reflect.Ptr && rv.IsNil() {
				return false
			}
			h, ok := user.(interface{ HasAccess(name, method string) bool })
			return ok && h.HasAccess(name, method)
		},
		// hasRole(currentUser, "Administrator") → bool. Cek apakah user memiliki
		// role dengan nama tertentu. Padanan hasRole(name) NodeAdmin. Nil-aman.
		"hasRole": func(user any, roleName string) bool {
			if user == nil {
				return false
			}
			if rv := reflect.ValueOf(user); rv.Kind() == reflect.Ptr && rv.IsNil() {
				return false
			}
			h, ok := user.(interface{ HasRole(name string) bool })
			return ok && h.HasRole(roleName)
		},
		// getOld(data, "field") → nilai lama form (untuk repopulasi setelah validasi
		// gagal). Membaca dari data["old"] yang di-set RenderView dari sesi flash.
		"getOld": func(data interface{}, key string) string {
			m, ok := data.(gin.H)
			if !ok {
				return ""
			}
			old, ok := m["old"].(map[string]string)
			if !ok {
				return ""
			}
			return old[key]
		},
		// getError(data, "field") → pesan error validasi per-field (untuk inline
		// error di form). Membaca dari data["errors"] yang di-set RenderView.
		"getError": func(data interface{}, key string) string {
			m, ok := data.(gin.H)
			if !ok {
				return ""
			}
			errs, ok := m["errors"].(map[string]string)
			if !ok {
				return ""
			}
			return errs[key]
		},
		// getFile(path) → URL file (untuk menampilkan gambar/dokumen yang diupload).
		// Mengembalikan path apa adanya; tambahkan prefix storage bila diperlukan.
		"getFile": func(path string) string {
			return path
		},
	}
}

// Load mem-parse seluruh template (layout + modul) dari glob pattern.
// Mengembalikan nil-aman bila tak ada file (mode api / belum ada view).
func Load(patterns ...string) (*Engine, error) {
	tmpl := template.New("").Funcs(FuncMap())
	var loaded bool
	for _, p := range patterns {
		t, err := tmpl.ParseGlob(p)
		if err != nil {
			// Glob tanpa match bukan error fatal (modul mungkin tak punya view).
			continue
		}
		tmpl = t
		loaded = true
	}
	if !loaded {
		return &Engine{tmpl: template.New("").Funcs(FuncMap())}, nil
	}
	return &Engine{tmpl: tmpl}, nil
}

// Attach memasang template engine ke gin.
func (e *Engine) Attach(r *gin.Engine) {
	r.SetHTMLTemplate(e.tmpl)
}

// RenderView merender satu view dengan locals (data) + status 200.
// Locals otomatis diperkaya dengan user terautentikasi bila ada di context.
func RenderView(c *gin.Context, name string, locals gin.H) {
	if locals == nil {
		locals = gin.H{}
	}
	if u, ok := c.Get("auth_user"); ok {
		locals["currentUser"] = u
	}
	// Token CSRF (diset middleware.CSRF) → form menyertakan <input name="_csrf">.
	if tok, ok := c.Get("csrf_token"); ok {
		locals["_csrf"] = tok
	}
	// Flash one-shot (diset middleware.Flash) → banner sukses/error.
	if v, ok := c.Get("flash_success"); ok {
		locals["flash_success"] = v
	}
	if v, ok := c.Get("flash_error"); ok {
		locals["flash_error"] = v
	}
	// Error per-field + old input (validasi inline, padanan getError/old NodeAdmin).
	// Selalu di-set (default map kosong) agar `index .errors "x"` nil-aman di template.
	locals["errors"] = map[string]string{}
	if v, ok := c.Get("field_errors"); ok {
		locals["errors"] = v
	}
	locals["old"] = map[string]string{}
	if v, ok := c.Get("field_old"); ok {
		locals["old"] = v
	}
	// Tema aktif + setting (diset di app via ThemeContext) → chrome themeable.
	for _, k := range []string{"theme", "themeName", "themes", "setting"} {
		if v, ok := c.Get(k); ok {
			if _, exists := locals[k]; !exists {
				locals[k] = v
			}
		}
	}
	c.HTML(http.StatusOK, name, locals)
}
