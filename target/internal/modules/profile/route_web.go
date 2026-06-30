package profile

import (
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	accesssvc "goadmin/internal/modules/access/service"
	webctl "goadmin/internal/modules/profile/controller/web"
	"goadmin/internal/modules/profile/service"
)

// registerWebRoutes memasang halaman profil (HTML, sesi). Hanya dipanggil mode
// full (ctx.Web != nil). Cukup EnsureAuthenticatedWeb — tanpa permission RBAC
// (profil milik-sendiri, bukan resource ber-izin).
func registerWebRoutes(ctx *router.RegistrationContext) {
	c := ctx.Container

	authSvc, _ := c.Resolve("access.IAuthService").(accesssvc.IAuthService)
	profileSvc, _ := c.Resolve("profile.IProfileService").(service.IProfileService)

	jwtless := accessmw.NewGuardWebOnly(authSvc)
	ctl := webctl.NewProfileController(profileSvc, c.Storage)

	admin := ctx.Web.Group("/admin/v1")
	admin.Use(jwtless.EnsureAuthenticatedWeb("/auth/login"))

	admin.GET("/profile", ctl.Index)
	router.Register("GET", "admin.v1.profile.index", "/admin/v1/profile")

	admin.PUT("/profile/update", ctl.Update)
	router.Register("PUT", "admin.v1.profile.update", "/admin/v1/profile/update")
}
