// Package app merakit engine Gin lengkap dari container + modul terdaftar.
// Inilah titik tunggal yang mencabangkan varian Full vs API-only via APP_MODE
// (diff purely-additive: file ini identik di kedua mode, cabang lewat runtime).
package app

import (
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	redisstore "github.com/gin-contrib/sessions/redis"
	"github.com/gin-gonic/gin"

	"goadmin/internal/config"
	"goadmin/internal/container"
	"goadmin/internal/middleware"
	"goadmin/internal/store/dbsession"
	settingmodel "goadmin/internal/modules/setting/model"
	settingsvc "goadmin/internal/modules/setting/service"
	"goadmin/internal/modules/setting/theme"
	"goadmin/internal/router"
	"goadmin/internal/view"
)

// Build menghasilkan *gin.Engine siap-listen.
func Build(c *container.Container) *gin.Engine {
	cfg := c.Config
	if cfg.IsProd {
		gin.SetMode(gin.ReleaseMode)
	} else if cfg.IsTest {
		gin.SetMode(gin.TestMode)
	}

	engine := gin.New()

	// --- Middleware global (urutan penting) ---
	engine.Use(gin.Recovery())                                                   // pulih dari panic → 500
	engine.Use(middleware.ErrorHandler(cfg.IsProd, cfg.App.Mode == config.ModeFull)) // error terpusat (HTML di mode full)
	engine.Use(middleware.SecurityHeaders(cfg))   // header keamanan (helmet setara)
	engine.Use(gzip.Gzip(gzip.DefaultCompression)) // kompresi response

	if len(cfg.Security.CORSOrigins) > 0 {
		engine.Use(cors.New(cors.Config{
			AllowOrigins:     cfg.Security.CORSOrigins,
			AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			AllowCredentials: true,
		}))
	}

	// --- Grup route per varian ---
	apiGroup := engine.Group("/api")

	var webGroup *gin.RouterGroup
	if cfg.App.Mode == config.ModeFull {
		// Mode full: pasang lapisan web (sesi, static, template, route web).
		// webGroup non-nil → modul UI mendaftarkan route web-nya.
		mountWebLayer(engine, cfg)

		// Sesi web: pilih store berdasarkan SESSION_DRIVER (cookie | redis | database).
		// - database : sesi di tabel `sessions` DB utama (default, persist lintas restart).
		// - redis    : sesi di Redis (persist lintas restart, butuh REDIS_URL).
		// - cookie   : sesi di signed cookie client-side (stateless, tanpa server storage).
		sessOpts := sessions.Options{
			MaxAge:   int(cfg.Session.TTL.Seconds()),
			HttpOnly: true,
			Secure:   cfg.IsProd,
			Path:     "/",
		}
		var store sessions.Store
		switch cfg.Session.Driver {
		case "redis":
			rs, err := redisstore.NewStore(10, "tcp", cfg.Redis.URL, "", "", []byte(cfg.Session.Secret))
			if err != nil {
				panic("session: gagal konek Redis store: " + err.Error())
			}
			rs.Options(sessOpts)
			store = rs
		case "database":
			ds, err := dbsession.New(c.DB, []byte(cfg.Session.Secret))
			if err != nil {
				panic("session: gagal inisialisasi database store: " + err.Error())
			}
			ds.Options(sessOpts)
			store = ds
		default:
			cs := cookie.NewStore([]byte(cfg.Session.Secret))
			cs.Options(sessOpts)
			store = cs
		}
		webGroup = engine.Group("/")
		webGroup.Use(sessions.Sessions("goadmin_session", store))
		// CSRF setelah sesi (butuh sesi). Hanya jalur web — API (JWT) dikecualikan.
		webGroup.Use(middleware.CSRF())
		// Flash one-shot (feedback pasca-redirect) → context → RenderView locals.
		webGroup.Use(middleware.Flash())
		// Tema aktif + setting global → context (chrome admin themeable + switchable).
		webGroup.Use(themeContext(c))
	}
	// Mode api: webGroup tetap nil → modul UI guard & lewati registrasi web.

	router.RegisterAll(&router.RegistrationContext{
		Mode:      cfg.App.Mode,
		Container: c,
		Web:       webGroup,
		API:       apiGroup,
	})

	registerHealthcheck(engine)
	return engine
}

// mountWebLayer memasang aset statis + template HTML (mode full).
func mountWebLayer(engine *gin.Engine, cfg *config.Config) {
	// Static assets (posisi awal). Folder boleh kosong saat dev.
	engine.Static("/assets", "./web/assets")
	// File upload (avatar/logo) disajikan dari folder storage lokal.
	if cfg.Storage.Dir != "" && cfg.Storage.URLBase != "" {
		_ = os.MkdirAll(cfg.Storage.Dir, 0o755)
		engine.Static(cfg.Storage.URLBase, cfg.Storage.Dir)
	}

	// Template: layout/partial + view tiap modul.
	if eng, err := view.Load(
		"web/templates/layouts/*.html",
		"web/templates/partials/*.html",
		// View per-modul: tiap file `{{define "<modul>/<view>"}}…{{end}}`.
		// Pola satu-tingkat (Go ParseGlob TIDAK mendukung `**`).
		"internal/modules/*/view/*.html",
	); err == nil {
		eng.Attach(engine)
	}
}

// themeContext meng-inject setting global + palet tema aktif ke gin.Context tiap
// request web (RenderView lalu menaruhnya ke locals chrome). Setting di-resolve
// LAZY dari container (ter-cache di service).
func themeContext(c *container.Container) gin.HandlerFunc {
	return func(gc *gin.Context) {
		name := theme.Default
		var setting *settingmodel.Setting
		if svc, ok := c.Resolve("setting.ISettingService").(settingsvc.ISettingService); ok && svc != nil {
			if s, err := svc.Get(gc.Request.Context()); err == nil {
				setting = s
				if s.Theme != "" && theme.IsValid(s.Theme) {
					name = s.Theme
				}
			}
		}
		gc.Set("setting", setting)
		gc.Set("themeName", name)
		gc.Set("theme", theme.ByName(name))
		gc.Set("themes", theme.All())
		gc.Next()
	}
}

func registerHealthcheck(engine *gin.Engine) {
	engine.GET("/healthz", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
}
