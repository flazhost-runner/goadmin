// Package access adalah modul referensi RBAC (User, Role, Permission) + auth.
// Mendaftarkan diri ke router registry lewat init() (side-effect import).
package access

import (
	"goadmin/internal/auth"
	"goadmin/internal/config"
	"goadmin/internal/container"
	"goadmin/internal/router"

	apictl "goadmin/internal/modules/access/controller/api"
	accessmw "goadmin/internal/modules/access/middleware"
	"goadmin/internal/modules/access/service"
)

// Module mengimplementasi router.Module.
type Module struct{}

// init mendaftarkan modul saat paket diimpor.
func init() {
	router.Add(&Module{})
}

// Name → identitas modul.
func (m *Module) Name() string { return "access" }

// Register merakit service modul ke container + memasang route.
// Route API selalu dipasang (kedua mode). Route web hanya bila ctx.Web != nil
// (mode full) — guard kehadiran menjaga diff full↔api purely-additive.
func (m *Module) Register(ctx *router.RegistrationContext) {
	c := ctx.Container
	cfg := c.Config

	// --- Wiring service (DI manual terpusat) ---
	userSvc := service.NewUserService(c.DB, cfg.Security.BcryptRounds)
	roleSvc := service.NewRoleService(c.DB)
	permSvc := service.NewPermissionService(c.DB)

	jwtMgr := auth.NewJWTManager(cfg.JWT.Secret, cfg.JWT.ExpiresIn)
	blacklist := resolveBlacklist(c)
	authSvc := service.NewAuthService(c.DB, jwtMgr, blacklist)
	resetSvc := service.NewPasswordResetService(c.DB, c.Mailer, cfg.Security.BcryptRounds, cfg.App.Name, cfg.Security.OTPExpiryMinutes)

	// Simpan ke container (token = nama interface) agar bisa diintrospeksi.
	c.Provide("access.IUserService", userSvc)
	c.Provide("access.IRoleService", roleSvc)
	c.Provide("access.IPermissionService", permSvc)
	c.Provide("access.IAuthService", authSvc)
	c.Provide("access.IPasswordResetService", resetSvc)

	guard := accessmw.NewGuard(jwtMgr, blacklist, authSvc)
	// Sediakan guard ke container agar modul lain bisa memproteksi route API-nya
	// dengan blacklist yang SAMA (fidelity logout lintas-modul). Diresolve di
	// module.go modul lain (mereka terdaftar setelah 'access' — urut nama).
	c.Provide("access.Guard", guard)

	// --- Route API (selalu) ---
	registerAPIRoutes(ctx, guard, apiDeps{
		auth: apictl.NewAuthController(authSvc, userSvc, resetSvc),
		user: apictl.NewUserController(userSvc),
		role: apictl.NewRoleController(roleSvc),
		perm: apictl.NewPermissionController(permSvc),
	})

	// --- Route web (hanya mode full) ---
	if ctx.Mode == config.ModeFull && ctx.Web != nil {
		registerWebRoutes(ctx)
	}
}

// resolveBlacklist memilih store blacklist sesuai ketersediaan Redis.
// Tanpa redis (test/dev) → MemoryBlacklist yang BERPERILAKU IDENTIK (fidelity).
func resolveBlacklist(c *container.Container) auth.TokenBlacklist {
	if c.Redis != nil {
		return auth.NewRedisBlacklist(c.Redis)
	}
	return auth.NewMemoryBlacklist()
}
