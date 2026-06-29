package profile

import (
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	apictl "goadmin/internal/modules/profile/controller/api"
)

// registerAPIRoutes memasang endpoint profil milik-sendiri. Cukup terautentikasi
// (JWT) — TANPA permission RBAC: setiap user boleh mengelola profilnya sendiri.
func registerAPIRoutes(ctx *router.RegistrationContext, guard *accessmw.Guard, ctl *apictl.ProfileController) {
	g := ctx.API.Group("/v1/profile", guard.AuthenticatedJWT())

	g.GET("", ctl.Index)
	router.Register("GET", "api.v1.profile.index", "/api/v1/profile")

	g.PUT("/update", ctl.Update)
	router.Register("PUT", "api.v1.profile.update", "/api/v1/profile/update")
}
