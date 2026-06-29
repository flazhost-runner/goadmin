package media

import (
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	accesssvc "goadmin/internal/modules/access/service"
	webctl "goadmin/internal/modules/media/controller/web"
)

// registerWebRoutes memasang endpoint media (AJAX JSON, sesi). Dipakai file
// manager Trumbowyg. CSRF global (webGroup) memvalidasi POST via X-CSRF-Token.
func registerWebRoutes(ctx *router.RegistrationContext) {
	c := ctx.Container

	authSvc, _ := c.Resolve("access.IAuthService").(accesssvc.IAuthService)
	jwtless := accessmw.NewGuardWebOnly(authSvc)
	ctl := webctl.NewMediaController(c.Storage)

	admin := ctx.Web.Group("/admin/v1")
	admin.Use(jwtless.EnsureAuthenticatedWeb("/auth/login"))

	admin.GET("/media/list", ctl.List)
	router.Register("GET", "admin.v1.media.list", "/admin/v1/media/list")
	admin.POST("/media/upload", ctl.Upload)
	router.Register("POST", "admin.v1.media.upload", "/admin/v1/media/upload")
	admin.POST("/media/delete", ctl.Delete)
	router.Register("POST", "admin.v1.media.delete", "/admin/v1/media/delete")
}
