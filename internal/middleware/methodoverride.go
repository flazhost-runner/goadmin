package middleware

import (
	"net/http"
	"strings"
)

// MethodOverride membungkus handler (engine) agar form HTML (hanya bisa
// GET/POST) dapat memicu PUT/PATCH/DELETE lewat query `?_method=PUT` —
// sejajar NodeAdmin (form action `...?_method=PUT`). Rewrite DILAKUKAN sebelum
// Gin me-routing (Gin memilih route by method), jadi WAJIB membungkus engine
// di level http.Server, bukan sebagai middleware grup.
func MethodOverride(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			if m := strings.ToUpper(r.URL.Query().Get("_method")); m != "" {
				switch m {
				case http.MethodPut, http.MethodPatch, http.MethodDelete:
					// Parse body SEBELUM mengubah method: net/http hanya mem-parse
					// body form untuk POST/PUT/PATCH, bukan DELETE. Jika method
					// diubah ke DELETE sebelum ParseForm dipanggil, r.PostForm
					// tidak akan terisi dan CSRF yang dikirim via body tidak dapat
					// dibaca middleware downstream.
					_ = r.ParseMultipartForm(32 << 20)
					r.Method = m
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}
