package access

import (
	"github.com/gin-gonic/gin"

	"goadmin/internal/router"

	apictl "goadmin/internal/modules/access/controller/api"
	accessmw "goadmin/internal/modules/access/middleware"
)

// apiDeps mengumpulkan controller API modul.
type apiDeps struct {
	auth *apictl.AuthController
	user *apictl.UserController
	role *apictl.RoleController
	perm *apictl.PermissionController
}

// registerAPIRoutes memasang seluruh endpoint REST modul access.
// Pola: route bernama (router.Register) + middleware authenticated→authorize.
func registerAPIRoutes(ctx *router.RegistrationContext, guard *accessmw.Guard, d apiDeps) {
	api := ctx.API.Group("/v1")

	// Auth (publik: login; terproteksi: logout, me).
	authGrp := api.Group("/auth")
	named(authGrp, "POST", "api.v1.auth.login", "/login", d.auth.Login)
	// Publik (rate-limited): registrasi + reset password OTP — paritas penuh web.
	named(authGrp, "POST", "api.v1.auth.register", "/register", d.auth.Register, otpLimiter.Middleware())
	named(authGrp, "POST", "api.v1.auth.reset.request", "/reset/request", d.auth.ResetRequest, otpLimiter.Middleware())
	named(authGrp, "POST", "api.v1.auth.reset.process", "/reset/process", d.auth.ResetProcess, otpLimiter.Middleware())
	// logout = POST (mutasi: blacklist token; GET tak boleh punya efek samping)
	authGrp.POST("/logout", guard.AuthenticatedJWT(), d.auth.Logout)
	router.Register("POST", "api.v1.auth.logout", "/api/v1/auth/logout")
	authGrp.GET("/me", guard.AuthenticatedJWT(), d.auth.Me)
	router.Register("GET", "api.v1.auth.me", "/api/v1/auth/me")

	// Resource RBAC — namespace `api.v1.access.*` + path `/api/v1/access/*`,
	// singular (seragam NodeAdmin). Butuh auth + permission spesifik.
	auth := guard.AuthenticatedJWT()

	users := api.Group("/access/user", auth)
	resource(users, "/api/v1/access/user", "api.v1.access.user", d.user.Index, d.user.Edit, d.user.Store, d.user.Update, d.user.Destroy, d.user.DeleteSelected)

	roles := api.Group("/access/role", auth)
	resource(roles, "/api/v1/access/role", "api.v1.access.role", d.role.Index, d.role.Edit, d.role.Store, d.role.Update, d.role.Destroy, d.role.DeleteSelected)
	// Kelola permission per-role (kembar web, persis NodeAdmin api.ts).
	roles.GET("/:id/permission", accessmw.Authorize(), d.role.Permission)
	router.Register("GET", "api.v1.access.role.permission", "/api/v1/access/role/:id/permission")
	roles.GET("/:id/permission/:permission_id/assign", accessmw.Authorize(), d.role.PermissionAssign)
	router.Register("GET", "api.v1.access.role.permission.assign", "/api/v1/access/role/:id/permission/:permission_id/assign")
	roles.POST("/:id/permission/assign_selected", accessmw.Authorize(), d.role.PermissionAssignSelected)
	router.Register("POST", "api.v1.access.role.permission.assign_selected", "/api/v1/access/role/:id/permission/assign_selected")
	roles.GET("/:id/permission/:permission_id/unassign", accessmw.Authorize(), d.role.PermissionUnassign)
	router.Register("GET", "api.v1.access.role.permission.unassign", "/api/v1/access/role/:id/permission/:permission_id/unassign")
	roles.POST("/:id/permission/unassign_selected", accessmw.Authorize(), d.role.PermissionUnassignSelected)
	router.Register("POST", "api.v1.access.role.permission.unassign_selected", "/api/v1/access/role/:id/permission/unassign_selected")

	perms := api.Group("/access/permission", auth)
	resource(perms, "/api/v1/access/permission", "api.v1.access.permission", d.perm.Index, d.perm.Edit, d.perm.Store, d.perm.Update, d.perm.Destroy, d.perm.DeleteSelected)
}

// named mendaftarkan satu route bernama (registry) + memasangnya ke grup.
func named(g *gin.RouterGroup, method, name, path string, h gin.HandlerFunc, mw ...gin.HandlerFunc) {
	handlers := append(mw, h)
	switch method {
	case "GET":
		g.GET(path, handlers...)
	case "POST":
		g.POST(path, handlers...)
	case "PUT":
		g.PUT(path, handlers...)
	case "DELETE":
		g.DELETE(path, handlers...)
	}
	router.Register(method, name, fullPath(g, path))
}

// resource memasang endpoint CRUD access dengan path & nama VERBOSE PERSIS
// NodeAdmin (BUKAN REST): index, store `/store`, edit `/:id/edit`, update
// `/:id/update`, delete `/:id/delete`, delete_selected `/delete_selected`.
// RBAC route-driven: Authorize() menurunkan permission dari nama-route+method
// (Administrator bypass). API kembar web; klien JWT kirim method asli (tanpa
// ?_method override). Tak ada REST `GET /:id` (show) / `DELETE /:id` (destroy).
func resource(g *gin.RouterGroup, basePath, nameBase string,
	index, edit, store, update, destroy, deleteSelected gin.HandlerFunc) {

	g.GET("", accessmw.Authorize(), index)
	router.Register("GET", nameBase+".index", basePath)

	g.POST("/store", accessmw.Authorize(), store)
	router.Register("POST", nameBase+".store", basePath+"/store")

	g.GET("/:id/edit", accessmw.Authorize(), edit)
	router.Register("GET", nameBase+".edit", basePath+"/:id/edit")

	g.PUT("/:id/update", accessmw.Authorize(), update)
	router.Register("PUT", nameBase+".update", basePath+"/:id/update")

	g.DELETE("/:id/delete", accessmw.Authorize(), destroy)
	router.Register("DELETE", nameBase+".delete", basePath+"/:id/delete")

	g.POST("/delete_selected", accessmw.Authorize(), deleteSelected)
	router.Register("POST", nameBase+".delete_selected", basePath+"/delete_selected")
}

// fullPath menggabungkan base path grup dengan path relatif untuk registry.
func fullPath(g *gin.RouterGroup, path string) string {
	return g.BasePath() + path
}
