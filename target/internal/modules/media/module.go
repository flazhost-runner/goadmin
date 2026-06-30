// Package media menyediakan endpoint AJAX untuk file manager rich text editor
// (Trumbowyg): list/upload/delete gambar editor ke storage. WEB-ONLY: pada mode
// api modul ini ABSEN total (guard kehadiran ctx.Web) — jaga diff full↔api
// purely-additive. Endpoint mengembalikan JSON (bukan HTML), dipakai plugin
// filemanager.js (web/assets/vendor/trumbowyg/filemanager.js).
package media

import (
	"goadmin/internal/config"
	"goadmin/internal/router"
)

// Module mengimplementasi router.Module.
type Module struct{}

// init mendaftarkan modul saat paket diimpor.
func init() { router.Add(&Module{}) }

// Name → identitas modul.
func (m *Module) Name() string { return "media" }

// Register hanya memasang route web (mode full). Mode api → tak ada apa-apa.
func (m *Module) Register(ctx *router.RegistrationContext) {
	if ctx.Mode != config.ModeFull || ctx.Web == nil {
		return
	}
	registerWebRoutes(ctx)
}
