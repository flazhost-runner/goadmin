// Package setting adalah modul konfigurasi global (singleton ber-cache) +
// theme switcher. Mendaftarkan diri ke router registry lewat init().
package setting

import (
	"goadmin/internal/config"
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	apictl "goadmin/internal/modules/setting/controller/api"
	"goadmin/internal/modules/setting/service"
)

// Module mengimplementasi router.Module.
type Module struct{}

// init mendaftarkan modul saat paket diimpor.
func init() { router.Add(&Module{}) }

// Name → identitas modul.
func (m *Module) Name() string { return "setting" }

// Register merakit service ke container + memasang route.
func (m *Module) Register(ctx *router.RegistrationContext) {
	c := ctx.Container

	// --- Wiring service (DI manual terpusat) ---
	svc := service.NewSettingService(c.DB)
	// Disediakan agar modul lain (mis. layout/home) bisa membaca setting global.
	c.Provide("setting.ISettingService", svc)

	// Guard dipinjam dari modul access (blacklist sama → logout konsisten).
	guard, _ := c.Resolve("access.Guard").(*accessmw.Guard)

	// --- Route API (selalu) ---
	registerAPIRoutes(ctx, guard, apictl.NewSettingController(svc))

	// --- Route web (hanya mode full) ---
	if ctx.Mode == config.ModeFull && ctx.Web != nil {
		registerWebRoutes(ctx)
	}
}
