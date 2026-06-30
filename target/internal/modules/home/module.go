// Package home adalah landing publik (frontend) data-driven dari Setting +
// frontend template switcher. WEB-ONLY: pada mode api modul ini ABSEN total
// (guard kehadiran ctx.Web).
package home

import (
	"goadmin/internal/config"
	"goadmin/internal/router"

	"goadmin/internal/modules/home/fetemplate"
	"goadmin/internal/modules/home/service"
	settingsvc "goadmin/internal/modules/setting/service"
)

// Module mengimplementasi router.Module.
type Module struct{}

// init mendaftarkan modul saat paket diimpor.
func init() { router.Add(&Module{}) }

// Name → identitas modul.
func (m *Module) Name() string { return "home" }

// Register memasang landing publik (mode full). Mode api → tak ada apa-apa.
func (m *Module) Register(ctx *router.RegistrationContext) {
	if ctx.Mode != config.ModeFull || ctx.Web == nil {
		return
	}
	c := ctx.Container
	cfg := c.Config

	// Setting di-resolve LAZY: modul setting terdaftar SETELAH home (urut nama),
	// jadi resolusi ditunda sampai request (saat itu semua modul sudah terdaftar).
	svc := service.NewHomeService(func() settingsvc.ISettingService {
		s, _ := c.Resolve("setting.ISettingService").(settingsvc.ISettingService)
		return s
	})

	// Frontend template service. Remote (fetch katalog + unduh) hanya bila
	// diaktifkan via env — default off → katalog kurasi (aman tanpa jaringan).
	var fetcher fetemplate.Fetcher
	if cfg.FeTemplate.Remote {
		fetcher = fetemplate.NewHTTPFetcher(cfg.FeTemplate.TreeURL, cfg.FeTemplate.RawBaseURL)
	}
	fe := fetemplate.New(fetcher, cfg.FeTemplate.CacheDir)
	// Bagikan ke container agar modul setting (terdaftar SETELAH home) bisa
	// menyematkan switcher template frontend di halaman Setting.
	c.Provide("home.fetemplate", fe)

	registerWebRoutes(ctx, svc, fe)
}
