package access

import (
	"time"

	"goadmin/internal/middleware"
	"goadmin/internal/router"

	webctl "goadmin/internal/modules/access/controller/web"
	accessmw "goadmin/internal/modules/access/middleware"
	"goadmin/internal/modules/access/service"
)

// loginLimiter meredam brute-force login: maks 10 percobaan / 15 menit / IP.
var loginLimiter = middleware.NewRateLimiter(10, 15*time.Minute)

// authLimiter membatasi aksi auth (register, reset request): maks 10 / 15 menit / IP.
var authLimiter = middleware.NewRateLimiter(10, 15*time.Minute)

// otpLimiter membatasi verifikasi OTP reset password: maks 5 / 15 menit / IP.
var otpLimiter = middleware.NewRateLimiter(5, 15*time.Minute)

// registerWebRoutes memasang route web modul access:
//   - Auth sesi PUBLIK: /auth/login (GET/POST), /auth/logout.
//   - Admin (butuh sesi): /admin/v1/users. Urutan middleware WAJIB:
//     EnsureAuthenticatedWeb → AuthorizeWeb.
func registerWebRoutes(ctx *router.RegistrationContext) {
	c := ctx.Container

	userSvc, _ := c.Resolve("access.IUserService").(service.IUserService)
	authSvc, _ := c.Resolve("access.IAuthService").(service.IAuthService)
	roleSvc, _ := c.Resolve("access.IRoleService").(service.IRoleService)
	permSvc, _ := c.Resolve("access.IPermissionService").(service.IPermissionService)
	resetSvc, _ := c.Resolve("access.IPasswordResetService").(service.IPasswordResetService)

	// --- Auth sesi (publik, tanpa guard) — penamaan + method seragam NodeAdmin ---
	authCtl := webctl.NewAuthController(authSvc, resetSvc, userSvc)
	ctx.Web.GET("/auth/login", authCtl.ShowLogin)
	router.Register("GET", "web.auth.login", "/auth/login")
	ctx.Web.POST("/auth/login", loginLimiter.Middleware(), authCtl.Login)
	router.Register("POST", "web.auth.login.post", "/auth/login")
	ctx.Web.POST("/auth/logout", authCtl.Logout)
	router.Register("POST", "web.auth.logout", "/auth/logout")

	// Register (publik; di-rate-limit).
	ctx.Web.GET("/auth/register", authCtl.ShowRegister)
	router.Register("GET", "web.auth.register", "/auth/register")
	ctx.Web.POST("/auth/register", authLimiter.Middleware(), authCtl.Register)
	router.Register("POST", "web.auth.register.post", "/auth/register")

	// Reset password via OTP email (publik; nama/path PERSIS NodeAdmin:
	// req/proc = form view, request/process = submit; OTP di-rate-limit).
	ctx.Web.GET("/admin/v1/auth/reset/req", authCtl.ShowForgot)
	router.Register("GET", "admin.v1.auth.reset.req", "/admin/v1/auth/reset/req")
	ctx.Web.POST("/admin/v1/auth/reset/request", authLimiter.Middleware(), authCtl.Forgot)
	router.Register("POST", "admin.v1.auth.reset.request", "/admin/v1/auth/reset/request")
	ctx.Web.GET("/admin/v1/auth/reset/proc", authCtl.ShowReset)
	router.Register("GET", "admin.v1.auth.reset.proc", "/admin/v1/auth/reset/proc")
	ctx.Web.POST("/admin/v1/auth/reset/process", otpLimiter.Middleware(), authCtl.Reset)
	router.Register("POST", "admin.v1.auth.reset.process", "/admin/v1/auth/reset/process")

	// --- Admin (butuh sesi). Path /admin/v1/access/{user,role,permission},
	// nama admin.v1.access.*, method PERSIS NodeAdmin (DELETE delete & PUT update
	// lewat ?_method= override). ---
	jwtless := accessmw.NewGuardWebOnly(authSvc)
	userCtl := webctl.NewUserController(userSvc, roleSvc, c.Storage)
	roleCtl := webctl.NewRoleController(roleSvc, permSvc)
	permCtl := webctl.NewPermissionController(permSvc)

	admin := ctx.Web.Group("/admin/v1/access")
	admin.Use(jwtless.EnsureAuthenticatedWeb("/auth/login"))

	// --- Users (CRUD web) ---
	admin.GET("/user", accessmw.AuthorizeWeb(), userCtl.Index)
	router.Register("GET", "admin.v1.access.user.index", "/admin/v1/access/user")
	admin.GET("/user/create", accessmw.AuthorizeWeb(), userCtl.Create)
	router.Register("GET", "admin.v1.access.user.create", "/admin/v1/access/user/create")
	admin.POST("/user/store", accessmw.AuthorizeWeb(), userCtl.Store)
	router.Register("POST", "admin.v1.access.user.store", "/admin/v1/access/user/store")
	admin.GET("/user/:id/edit", accessmw.AuthorizeWeb(), userCtl.Edit)
	router.Register("GET", "admin.v1.access.user.edit", "/admin/v1/access/user/:id/edit")
	admin.PUT("/user/:id/update", accessmw.AuthorizeWeb(), userCtl.Update)
	router.Register("PUT", "admin.v1.access.user.update", "/admin/v1/access/user/:id/update")
	admin.DELETE("/user/:id/delete", accessmw.AuthorizeWeb(), userCtl.Destroy)
	router.Register("DELETE", "admin.v1.access.user.delete", "/admin/v1/access/user/:id/delete")
	admin.POST("/user/delete_selected", accessmw.AuthorizeWeb(), userCtl.DeleteSelected)
	router.Register("POST", "admin.v1.access.user.delete_selected", "/admin/v1/access/user/delete_selected")

	// --- Roles (CRUD web) ---
	admin.GET("/role", accessmw.AuthorizeWeb(), roleCtl.Index)
	router.Register("GET", "admin.v1.access.role.index", "/admin/v1/access/role")
	admin.GET("/role/create", accessmw.AuthorizeWeb(), roleCtl.Create)
	router.Register("GET", "admin.v1.access.role.create", "/admin/v1/access/role/create")
	admin.POST("/role/store", accessmw.AuthorizeWeb(), roleCtl.Store)
	router.Register("POST", "admin.v1.access.role.store", "/admin/v1/access/role/store")
	admin.GET("/role/:id/edit", accessmw.AuthorizeWeb(), roleCtl.Edit)
	router.Register("GET", "admin.v1.access.role.edit", "/admin/v1/access/role/:id/edit")
	admin.PUT("/role/:id/update", accessmw.AuthorizeWeb(), roleCtl.Update)
	router.Register("PUT", "admin.v1.access.role.update", "/admin/v1/access/role/:id/update")
	admin.DELETE("/role/:id/delete", accessmw.AuthorizeWeb(), roleCtl.Destroy)
	router.Register("DELETE", "admin.v1.access.role.delete", "/admin/v1/access/role/:id/delete")
	admin.POST("/role/delete_selected", accessmw.AuthorizeWeb(), roleCtl.DeleteSelected)
	router.Register("POST", "admin.v1.access.role.delete_selected", "/admin/v1/access/role/delete_selected")
	// Kelola permission per-role (halaman terpisah, persis NodeAdmin): list +
	// assign/unassign single (GET) + assign/unassign bulk (POST).
	admin.GET("/role/:id/permission", accessmw.AuthorizeWeb(), roleCtl.Permission)
	router.Register("GET", "admin.v1.access.role.permission", "/admin/v1/access/role/:id/permission")
	admin.GET("/role/:id/permission/:permission_id/assign", accessmw.AuthorizeWeb(), roleCtl.PermissionAssign)
	router.Register("GET", "admin.v1.access.role.permission.assign", "/admin/v1/access/role/:id/permission/:permission_id/assign")
	admin.POST("/role/:id/permission/assign_selected", accessmw.AuthorizeWeb(), roleCtl.PermissionAssignSelected)
	router.Register("POST", "admin.v1.access.role.permission.assign_selected", "/admin/v1/access/role/:id/permission/assign_selected")
	admin.GET("/role/:id/permission/:permission_id/unassign", accessmw.AuthorizeWeb(), roleCtl.PermissionUnassign)
	router.Register("GET", "admin.v1.access.role.permission.unassign", "/admin/v1/access/role/:id/permission/:permission_id/unassign")
	admin.POST("/role/:id/permission/unassign_selected", accessmw.AuthorizeWeb(), roleCtl.PermissionUnassignSelected)
	router.Register("POST", "admin.v1.access.role.permission.unassign_selected", "/admin/v1/access/role/:id/permission/unassign_selected")

	// --- Permissions (CRUD web) ---
	admin.GET("/permission", accessmw.AuthorizeWeb(), permCtl.Index)
	router.Register("GET", "admin.v1.access.permission.index", "/admin/v1/access/permission")
	admin.GET("/permission/create", accessmw.AuthorizeWeb(), permCtl.Create)
	router.Register("GET", "admin.v1.access.permission.create", "/admin/v1/access/permission/create")
	admin.POST("/permission/store", accessmw.AuthorizeWeb(), permCtl.Store)
	router.Register("POST", "admin.v1.access.permission.store", "/admin/v1/access/permission/store")
	admin.GET("/permission/:id/edit", accessmw.AuthorizeWeb(), permCtl.Edit)
	router.Register("GET", "admin.v1.access.permission.edit", "/admin/v1/access/permission/:id/edit")
	admin.PUT("/permission/:id/update", accessmw.AuthorizeWeb(), permCtl.Update)
	router.Register("PUT", "admin.v1.access.permission.update", "/admin/v1/access/permission/:id/update")
	admin.DELETE("/permission/:id/delete", accessmw.AuthorizeWeb(), permCtl.Destroy)
	router.Register("DELETE", "admin.v1.access.permission.delete", "/admin/v1/access/permission/:id/delete")
	admin.POST("/permission/delete_selected", accessmw.AuthorizeWeb(), permCtl.DeleteSelected)
	router.Register("POST", "admin.v1.access.permission.delete_selected", "/admin/v1/access/permission/delete_selected")
}
