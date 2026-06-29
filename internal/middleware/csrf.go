package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
)

const (
	csrfSessionKey = "csrf_token"   // kunci token di sesi
	csrfFieldName  = "_csrf"        // nama field form
	csrfHeaderName = "x-csrf-token" // alternatif via header (AJAX, lowercase sesuai standar)
	csrfCtxKey     = "csrf_token"   // kunci di gin.Context (dibaca RenderView)
)

// CSRF melindungi form web mutasi dengan sinkronisasi token (double-submit lewat
// sesi). API stateless (JWT, tanpa cookie/sesi) DIKECUALIKAN — pasang middleware
// ini hanya pada grup web. Token diekspos ke template via context; RenderView
// menaruhnya sebagai `_csrf` agar form menyertakan <input hidden name="_csrf">.
func CSRF() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)
		token, _ := sess.Get(csrfSessionKey).(string)
		if token == "" {
			token = generateCSRFToken()
			sess.Set(csrfSessionKey, token)
			_ = sess.Save()
		}
		c.Set(csrfCtxKey, token)

		if isUnsafeMethod(c.Request.Method) {
			// Sumber token (urut): body form → query → header. Query WAJIB jadi
			// fallback karena net/http hanya mem-parse body form untuk
			// POST/PUT/PATCH — BUKAN DELETE; jadi form delete (POST + ?_method=
			// DELETE) menaruh _csrf di query. Sejajar NodeAdmin (body||query||header).
			submitted := c.PostForm(csrfFieldName)
			if submitted == "" {
				submitted = c.Query(csrfFieldName)
			}
			if submitted == "" {
				submitted = c.GetHeader(csrfHeaderName)
			}
			// Perbandingan waktu-konstan (cegah timing attack).
			if submitted == "" || subtle.ConstantTimeCompare([]byte(submitted), []byte(token)) != 1 {
				c.Error(apperr.Forbidden("CSRF token tidak valid"))
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func isUnsafeMethod(method string) bool {
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
		return true
	default:
		return false
	}
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
