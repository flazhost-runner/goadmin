// Command server adalah entry-point GoAdmin. Satu entry untuk kedua varian
// (Full/API) — cabang lewat APP_MODE di config. Menerapkan:
//   - fail-fast config (secret wajib di production),
//   - listen fail-fast (EADDRINUSE → pesan jelas + exit non-zero),
//   - graceful shutdown (tutup DB/Redis saat SIGTERM/SIGINT).
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"goadmin/internal/app"
	"goadmin/internal/middleware"
	"goadmin/internal/bootstrap"
	"goadmin/internal/config"
	"goadmin/internal/container"
	"goadmin/internal/database"
	_ "goadmin/internal/modules" // registrasi semua modul (side-effect import)
)

func main() {
	// Paksa timezone aplikasi ke UTC (konsisten lintas server).
	time.Local = time.UTC

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL config: %v", err)
	}

	db, err := database.Open(cfg)
	if err != nil {
		log.Fatalf("FATAL database: %v", err)
	}

	// Dev/test: auto-migrate skema + seed admin agar app langsung jalan tanpa
	// `make migrate`. Produksi: dilewati (pakai golang-migrate versioned).
	if !cfg.IsProd {
		if err := bootstrap.MigrateAndSeed(db, "admin@admin.com", "12345678", cfg.Security.BcryptRounds); err != nil {
			log.Fatalf("FATAL dev migrate: %v", err)
		}
	}

	rdb := connectRedis(cfg)

	c := container.New(cfg, db, rdb)
	engine := app.Build(c)

	// Permission route-driven: setelah route terdaftar (app.Build mengisi
	// registry), turunkan permission dari registry agar tersedia untuk
	// assignment role (a la NodeAdmin). Dev saja; prod pakai migrasi versioned.
	if !cfg.IsProd {
		if err := bootstrap.SyncPermissions(db); err != nil {
			log.Printf("WARN sync permission dari route: %v", err)
		}
	}

	addr := fmt.Sprintf(":%d", cfg.App.Port)
	srv := &http.Server{
		Addr: addr,
		// MethodOverride: form `?_method=PUT/DELETE` → ubah method SEBELUM Gin
		// me-routing (sejajar NodeAdmin). Membungkus engine di level server.
		Handler:           middleware.MethodOverride(engine),
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Jalankan listen di goroutine; tangani error bind (mis. port dipakai)
	// secara eksplisit — di Go, ListenAndServe mengembalikan error bind langsung.
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("GoAdmin [%s] mendengarkan di %s", cfg.App.Mode, addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// Tunggu sinyal shutdown atau error server.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErr:
		log.Fatalf("FATAL listen %s: %v", addr, err) // exit non-zero, pesan jelas
	case sig := <-quit:
		log.Printf("Sinyal %s diterima — graceful shutdown...", sig)
	}

	gracefulShutdown(srv, db, rdb)
}

func connectRedis(cfg *config.Config) *redis.Client {
	if cfg.IsTest {
		return nil // test pakai store in-memory; tak butuh redis nyata
	}
	opt, err := redis.ParseURL(cfg.Redis.URL)
	if err != nil {
		log.Printf("WARN redis URL tak valid (%v) — fitur sesi/blacklist nonaktif", err)
		return nil
	}
	return redis.NewClient(opt)
}

func gracefulShutdown(srv *http.Server, db *gorm.DB, rdb *redis.Client) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("WARN shutdown HTTP: %v", err)
	}
	if rdb != nil {
		_ = rdb.Close()
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	log.Println("Shutdown selesai.")
}
