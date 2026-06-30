// Package components adalah modul WEB-ONLY: halaman showcase komponen UI.
// Tak punya service/API/model. Pada mode api modul ini ABSEN total (guard
// kehadiran ctx.Web) — menjaga diff full↔api purely-additive.
package components

import (
	"goadmin/internal/config"
	"goadmin/internal/router"
)

// Module mengimplementasi router.Module.
type Module struct{}

// init mendaftarkan modul saat paket diimpor.
func init() { router.Add(&Module{}) }

// Name → identitas modul.
func (m *Module) Name() string { return "components" }

// Register hanya memasang route web (mode full). Mode api → tak ada apa-apa.
func (m *Module) Register(ctx *router.RegistrationContext) {
	if ctx.Mode != config.ModeFull || ctx.Web == nil {
		return
	}
	registerWebRoutes(ctx)
}
