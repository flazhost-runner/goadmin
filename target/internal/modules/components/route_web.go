package components

import (
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	accesssvc "goadmin/internal/modules/access/service"
	webctl "goadmin/internal/modules/components/controller/web"
)

// registerWebRoutes memasang halaman showcase komponen (HTML, sesi). Modul ini
// WEB-ONLY: tak punya route API. Cukup EnsureAuthenticatedWeb.
func registerWebRoutes(ctx *router.RegistrationContext) {
	c := ctx.Container

	authSvc, _ := c.Resolve("access.IAuthService").(accesssvc.IAuthService)
	jwtless := accessmw.NewGuardWebOnly(authSvc)
	ctl := webctl.NewComponentController()

	admin := ctx.Web.Group("/admin/v1")
	admin.Use(jwtless.EnsureAuthenticatedWeb("/auth/login"))

	admin.GET("/components", ctl.Index)
	router.Register("GET", "admin.v1.components.index", "/admin/v1/components")
}
