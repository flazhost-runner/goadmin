package dashboard

import (
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	accesssvc "goadmin/internal/modules/access/service"
	webctl "goadmin/internal/modules/dashboard/controller/web"
	"goadmin/internal/modules/dashboard/service"
)

// registerWebRoutes memasang halaman dashboard (HTML, sesi). Hanya dipanggil
// mode full (ctx.Web != nil). Cukup EnsureAuthenticatedWeb (landing admin).
func registerWebRoutes(ctx *router.RegistrationContext) {
	c := ctx.Container

	authSvc, _ := c.Resolve("access.IAuthService").(accesssvc.IAuthService)
	dashSvc, _ := c.Resolve("dashboard.IDashboardService").(service.IDashboardService)

	jwtless := accessmw.NewGuardWebOnly(authSvc)
	ctl := webctl.NewDashboardController(dashSvc)

	admin := ctx.Web.Group("/admin/v1")
	admin.Use(jwtless.EnsureAuthenticatedWeb("/auth/login"))

	admin.GET("/dashboard", ctl.Index)
	router.Register("GET", "admin.v1.dashboard.index", "/admin/v1/dashboard")
}
