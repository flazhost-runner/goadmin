// Package middleware berisi middleware lintas-aplikasi (error handler terpusat,
// security headers, dll). Padanan src/middleware/* + core di NodeAdmin.
package middleware

import (
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
)

// ErrorHandler adalah middleware error terpusat. Controller cukup memanggil
// `c.Error(err)` lalu return; middleware ini yang memetakan *AppError → HTTP.
//
//   - *AppError → status & pesan publiknya.
//   - error lain → 500 generik (detail hanya ke log, tak bocor ke user di prod).
//
// Mode render: jalur API (prefix /api) → JSON; selain itu, bila htmlErrors aktif
// (mode full punya template) → halaman HTML "errors/index". isProd menyembunyikan
// detail internal.
func ErrorHandler(isProd, htmlErrors bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}
		// Ambil error terakhir (paling spesifik).
		err := c.Errors.Last().Err

		status := http.StatusInternalServerError
		message := "Internal server error"
		var fields map[string]string

		if ae, ok := apperr.As(err); ok {
			status = ae.Status
			message = ae.Message
			fields = ae.Fields
			if ae.Detail != "" {
				log.Printf("[error] %s %s → %d: %s", c.Request.Method, c.Request.URL.Path, status, ae.Detail)
			}
		} else {
			// Error tak terklasifikasi → log penuh, kirim generik.
			log.Printf("[error] %s %s → 500: %v", c.Request.Method, c.Request.URL.Path, err)
		}

		// Hindari menulis body dua kali bila handler sudah menulis status.
		if c.Writer.Written() {
			return
		}

		// Jalur web (mode full, bukan /api) → halaman HTML. Selain itu JSON.
		if htmlErrors && !strings.HasPrefix(c.Request.URL.Path, "/api") {
			c.HTML(status, "errors/index", gin.H{"status": status, "message": message})
			return
		}

		var errPayload interface{}
		if fields != nil {
			errPayload = fields
		}
		helpers.JSONError(c, status, message, errPayload)
	}
}
