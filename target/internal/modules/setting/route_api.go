package setting

import (
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	apictl "goadmin/internal/modules/setting/controller/api"
)

// registerAPIRoutes memasang endpoint REST setting (singleton: index + update),
// path VERBOSE persis NodeAdmin (GET ``, PUT `/update`), terproteksi JWT + RBAC
// (route-driven; Administrator bypass).
func registerAPIRoutes(ctx *router.RegistrationContext, guard *accessmw.Guard, ctl *apictl.SettingController) {
	g := ctx.API.Group("/v1/setting", guard.AuthenticatedJWT())

	g.GET("", accessmw.Authorize(), ctl.Index)
	router.Register("GET", "api.v1.setting.index", "/api/v1/setting")

	g.PUT("/update", accessmw.Authorize(), ctl.Update)
	router.Register("PUT", "api.v1.setting.update", "/api/v1/setting/update")
}
