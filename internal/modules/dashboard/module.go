// Package dashboard adalah modul statistik ringkas (read-only). Reuse entity
// access (User/Role/Permission); tak punya tabel sendiri. Mendaftarkan diri ke
// router registry lewat init().
package dashboard

import (
	"goadmin/internal/config"
	"goadmin/internal/router"

	accessmw "goadmin/internal/modules/access/middleware"
	apictl "goadmin/internal/modules/dashboard/controller/api"
	"goadmin/internal/modules/dashboard/service"
)

// Module mengimplementasi router.Module.
type Module struct{}

// init mendaftarkan modul saat paket diimpor.
func init() { router.Add(&Module{}) }

// Name → identitas modul.
func (m *Module) Name() string { return "dashboard" }

// Register merakit service ke container + memasang route.
func (m *Module) Register(ctx *router.RegistrationContext) {
	c := ctx.Container

	// --- Wiring service (DI manual terpusat) ---
	svc := service.NewDashboardService(c.DB)
	c.Provide("dashboard.IDashboardService", svc)

	// Guard dipinjam dari modul access (blacklist sama → logout konsisten).
	guard, _ := c.Resolve("access.Guard").(*accessmw.Guard)

	// --- Route API (selalu) ---
	registerAPIRoutes(ctx, guard, apictl.NewDashboardController(svc))

	// --- Route web (hanya mode full) ---
	if ctx.Mode == config.ModeFull && ctx.Web != nil {
		registerWebRoutes(ctx)
	}
}
