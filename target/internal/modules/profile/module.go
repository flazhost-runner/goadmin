// Package profile adalah modul self-service profil user terautentikasi (lihat/
// edit profil + ganti password). Reuse entity access.User; tak punya tabel
// sendiri. Mendaftarkan diri ke router registry lewat init().
package profile

import (
	"goadmin/internal/config"
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	apictl "goadmin/internal/modules/profile/controller/api"
	"goadmin/internal/modules/profile/service"
)

// Module mengimplementasi router.Module.
type Module struct{}

// init mendaftarkan modul saat paket diimpor.
func init() { router.Add(&Module{}) }

// Name → identitas modul.
func (m *Module) Name() string { return "profile" }

// Register merakit service ke container + memasang route.
func (m *Module) Register(ctx *router.RegistrationContext) {
	c := ctx.Container
	cfg := c.Config

	// --- Wiring service (DI manual terpusat) ---
	svc := service.NewProfileService(c.DB, cfg.Security.BcryptRounds)
	c.Provide("profile.IProfileService", svc)

	// Guard dipinjam dari modul access (blacklist sama → logout konsisten).
	guard, _ := c.Resolve("access.Guard").(*accessmw.Guard)

	// --- Route API (selalu) ---
	registerAPIRoutes(ctx, guard, apictl.NewProfileController(svc))

	// --- Route web (hanya mode full) ---
	if ctx.Mode == config.ModeFull && ctx.Web != nil {
		registerWebRoutes(ctx)
	}
}
