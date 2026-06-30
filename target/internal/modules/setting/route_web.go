package setting

import (
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	accesssvc "goadmin/internal/modules/access/service"
	"goadmin/internal/modules/home/fetemplate"
	webctl "goadmin/internal/modules/setting/controller/web"
	"goadmin/internal/modules/setting/service"
)

// registerWebRoutes memasang halaman admin setting (HTML, sesi). Hanya dipanggil
// mode full (ctx.Web != nil). Urutan: EnsureAuthenticatedWeb → AuthorizeWeb.
func registerWebRoutes(ctx *router.RegistrationContext) {
	c := ctx.Container

	authSvc, _ := c.Resolve("access.IAuthService").(accesssvc.IAuthService)
	settingSvc, _ := c.Resolve("setting.ISettingService").(service.ISettingService)
	// home terdaftar SEBELUM setting (urut nama) → fetemplate.Service sudah ada.
	fe, _ := c.Resolve("home.fetemplate").(*fetemplate.Service)

	jwtless := accessmw.NewGuardWebOnly(authSvc)
	ctl := webctl.NewSettingController(settingSvc, c.Storage, fe)

	admin := ctx.Web.Group("/admin/v1")
	admin.Use(jwtless.EnsureAuthenticatedWeb("/auth/login"))

	admin.GET("/setting", accessmw.AuthorizeWeb(), ctl.Index)
	router.Register("GET", "admin.v1.setting.index", "/admin/v1/setting")

	admin.PUT("/setting/update", accessmw.AuthorizeWeb(), ctl.Update)
	router.Register("PUT", "admin.v1.setting.update", "/admin/v1/setting/update")
	// Catatan: template FE disimpan via form Setting utama (hidden fe_template),
	// diunduh saat Save — sama NodeAdmin (tanpa endpoint apply terpisah).
}
