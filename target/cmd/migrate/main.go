// Command migrate menjalankan migrasi SKEMA VERSIONED & REVERSIBLE (golang-migrate,
// file SQL .up/.down portabel di internal/migrate/migrations) — jalur PRODUKSI.
//
// Pakai:
//
//	go run ./cmd/migrate                 # up semua + seed admin
//	go run ./cmd/migrate -down 1         # rollback 1 langkah
//	go run ./cmd/migrate -version        # versi saat ini
//	go run ./cmd/migrate -force 2        # set versi paksa (recover dirty)
//	go run ./cmd/migrate -create add_x   # buat file migrasi baru
//
// (Dev/test memakai AutoMigrate cepat dari model — lihat internal/bootstrap;
// dipakai cmd/server saat dev & testutil.)
package main

import (
	"flag"
	"log"

	"goadmin/internal/bootstrap"
	"goadmin/internal/config"
	"goadmin/internal/database"
	migratepkg "goadmin/internal/migrate"
	accessmig "goadmin/internal/modules/access/migration"
)

func main() {
	down := flag.Int("down", 0, "rollback N langkah (0 = tidak)")
	force := flag.Int("force", -1, "set versi paksa untuk memulihkan state dirty")
	showVersion := flag.Bool("version", false, "tampilkan versi migrasi saat ini")
	create := flag.String("create", "", "buat pasangan file migrasi baru bernama X")
	seed := flag.Bool("seed", true, "seed admin/RBAC setelah up")
	email := flag.String("email", "admin@admin.com", "email admin default")
	password := flag.String("password", "12345678", "password admin default")
	flag.Parse()

	// -create tak butuh koneksi DB.
	if *create != "" {
		base, err := migratepkg.Create(*create)
		if err != nil {
			log.Fatalf("FATAL create: %v", err)
		}
		log.Printf("migrasi dibuat: internal/migrate/migrations/%s.{up,down}.sql", base)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("FATAL config: %v", err)
	}

	// SQLite (dev/test): golang-migrate bentrok driver → pakai AutoMigrate cepat.
	// Operasi versioned (down/force/version) hanya untuk mysql/postgres.
	if cfg.DB.Type == "sqlite" {
		if *down > 0 || *force >= 0 || *showVersion {
			log.Fatalf("operasi versioned (down/force/version) tak berlaku untuk sqlite — pakai mysql/postgres")
		}
		db, derr := database.Open(cfg)
		if derr != nil {
			log.Fatalf("FATAL database: %v", derr)
		}
		if err := bootstrap.MigrateAndSeed(db, *email, *password, cfg.Security.BcryptRounds); err != nil {
			log.Fatalf("FATAL automigrate+seed (sqlite): %v", err)
		}
		log.Printf("AutoMigrate + seed selesai (sqlite dev, admin: %s)", *email)
		return
	}

	switch {
	case *showVersion:
		v, dirty, err := migratepkg.Version(cfg.DB)
		if err != nil {
			log.Fatalf("FATAL version: %v", err)
		}
		log.Printf("versi migrasi: %d (dirty=%v)", v, dirty)

	case *force >= 0:
		if err := migratepkg.Force(cfg.DB, *force); err != nil {
			log.Fatalf("FATAL force: %v", err)
		}
		log.Printf("versi dipaksa ke %d", *force)

	case *down > 0:
		if err := migratepkg.Down(cfg.DB, *down); err != nil {
			log.Fatalf("FATAL down: %v", err)
		}
		log.Printf("rollback %d langkah selesai", *down)

	default: // up (+ seed)
		if err := migratepkg.Up(cfg.DB); err != nil {
			log.Fatalf("FATAL migrate up: %v", err)
		}
		log.Println("migrate up selesai")
		if *seed {
			db, derr := database.Open(cfg)
			if derr != nil {
				log.Fatalf("FATAL database: %v", derr)
			}
			if err := accessmig.Seed(db, *email, *password, cfg.Security.BcryptRounds); err != nil {
				log.Fatalf("FATAL seed: %v", err)
			}
			log.Printf("seed selesai (admin: %s)", *email)
		}
	}
}
