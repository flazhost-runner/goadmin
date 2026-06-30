package home

import (
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	accesssvc "goadmin/internal/modules/access/service"
	webctl "goadmin/internal/modules/home/controller/web"
	"goadmin/internal/modules/home/fetemplate"
	"goadmin/internal/modules/home/service"
)

// registerWebRoutes memasang landing publik + proxy pratinjau template FE.
// CATATAN: switcher template frontend DI-FOLD ke halaman Setting (persis
// NodeAdmin) — TIDAK ada halaman "appearance" terpisah. Pemilihan disimpan via
// form Setting (PUT /admin/v1/setting/update, hidden fe_template). Yang tersisa
// di sini hanya proxy pratinjau, bernama `admin.v1.setting.fe_preview`.
func registerWebRoutes(ctx *router.RegistrationContext, svc service.IHomeService, fe *fetemplate.Service) {
	c := ctx.Container

	// --- Landing publik ('/' render langsung; '/home' alias) ---
	ctl := webctl.NewHomeController(svc, fe)
	ctx.Web.GET("/", ctl.Index)
	router.Register("GET", "web.home.root", "/")
	ctx.Web.GET("/home", ctl.Index)
	router.Register("GET", "web.home.index", "/home")

	// --- Admin: proxy pratinjau template FE (thumbnail/modal di halaman Setting).
	// Namespace `setting` (folded), BUKAN modul appearance terpisah. ---
	authSvc, _ := c.Resolve("access.IAuthService").(accesssvc.IAuthService)
	jwtless := accessmw.NewGuardWebOnly(authSvc)
	admin := ctx.Web.Group("/admin/v1")
	admin.Use(jwtless.EnsureAuthenticatedWeb("/auth/login"))

	admin.GET("/setting/fe-preview/:slug", accessmw.AuthorizeWeb(), ctl.Preview)
	router.Register("GET", "admin.v1.setting.fe_preview", "/admin/v1/setting/fe-preview/:slug")
}
