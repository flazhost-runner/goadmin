package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	coreMW "goadmin/internal/middleware"
	"goadmin/internal/router"
)

// Authorize (RBAC route-driven, a la NodeAdmin AccessMiddleware): menurunkan
// NAMA route + METHOD dari request berjalan, lalu memeriksa user memiliki
// permission (name+method) tsb. TANPA argumen subjek — granularitas per-route.
// HARUS dipasang SETELAH AuthenticatedJWT. Administrator bypass.
// Bila akses ditolak: flash error + redirect ke Referer (fallback /admin/v1/access/user).
func Authorize() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := UserFrom(c)
		if user == nil {
			setFlashAndRedirect(c, "Unauthorized.", "/admin/v1/access/user")
			return
		}
		// Nama route diturunkan dari (method, pola-path); "" bila route tak
		// bernama → HasAccess false utk non-admin (admin tetap bypass).
		name := router.NameByMethodPath(c.Request.Method, c.FullPath())
		if !user.HasAccess(name, c.Request.Method) {
			setFlashAndRedirect(c, "Unauthorized.", "/admin/v1/access/user")
			return
		}
		c.Next()
	}
}

// setFlashAndRedirect menyimpan flash error ke sesi lalu redirect ke Referer
// (fallback ke fallbackURL bila Referer kosong). Pola PRG standar NodeAdmin.
func setFlashAndRedirect(c *gin.Context, msg, fallbackURL string) {
	sess := sessions.Default(c)
	coreMW.SetFlashError(sess, msg)
	ref := c.Request.Referer()
	if ref == "" {
		ref = fallbackURL
	}
	c.Redirect(302, ref)
	c.Abort()
}
