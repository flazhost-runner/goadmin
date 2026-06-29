package middleware

import (
	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	coreMW "goadmin/internal/middleware"
	"goadmin/internal/router"
)

// SessionUserKey = kunci penyimpanan user id di sesi web.
const SessionUserKey = "user_id"

// EnsureAuthenticatedWeb memastikan ada sesi login (web). Bila tidak, redirect
// ke halaman login. Padanan ensureAuthenticated di NodeAdmin (jalur sesi).
func (g *Guard) EnsureAuthenticatedWeb(loginPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)
		uid, _ := sess.Get(SessionUserKey).(string)
		if uid == "" {
			c.Redirect(302, loginPath)
			c.Abort()
			return
		}
		user, err := g.auths.FindByID(c.Request.Context(), uid)
		if err != nil {
			sess.Clear()
			_ = sess.Save()
			c.Redirect(302, loginPath)
			c.Abort()
			return
		}
		c.Set(ctxUserKey, user)
		c.Next()
	}
}

// AuthorizeWeb = RBAC route-driven untuk jalur web (a la NodeAdmin
// AccessMiddleware). Nama route + method diturunkan dari request berjalan —
// TANPA argumen subjek. Administrator bypass.
// Bila akses ditolak: flash error 'Unauthorized.' + redirect ke Referer
// (fallback /admin/v1/access/user). Pola PRG standar NodeAdmin.
func AuthorizeWeb() gin.HandlerFunc {
	return func(c *gin.Context) {
		user := UserFrom(c)
		name := router.NameByMethodPath(c.Request.Method, c.FullPath())
		if user == nil || !user.HasAccess(name, c.Request.Method) {
			sess := sessions.Default(c)
			coreMW.SetFlashError(sess, "Unauthorized.")
			ref := c.Request.Referer()
			if ref == "" {
				ref = "/admin/v1/access/user"
			}
			c.Redirect(302, ref)
			c.Abort()
			return
		}
		c.Next()
	}
}
