// Package bootstrap menyatukan migrasi skema + seed data awal seluruh modul
// (dev/test) di satu tempat, dipakai cmd/migrate & cmd/server (dev). Untuk
// PRODUKSI gunakan golang-migrate (.up/.down.sql portabel, versioned).
package bootstrap

import (
	"context"

	"gorm.io/gorm"

	accessmig "goadmin/internal/modules/access/migration"
	accesssvc "goadmin/internal/modules/access/service"
	settingmig "goadmin/internal/modules/setting/migration"
)

// Migrate menyelaraskan skema seluruh modul ber-tabel (modul lain reuse entity
// access). Idempoten.
func Migrate(db *gorm.DB) error {
	if err := accessmig.AutoMigrate(db); err != nil {
		return err
	}
	return settingmig.AutoMigrate(db)
}

// MigrateAndSeed = Migrate + seed admin/RBAC inti (idempoten).
func MigrateAndSeed(db *gorm.DB, adminEmail, adminPassword string, bcryptRounds int) error {
	if err := Migrate(db); err != nil {
		return err
	}
	return accessmig.Seed(db, adminEmail, adminPassword, bcryptRounds)
}

// SyncPermissions menurunkan permission dari named-route registry ke DB
// (route-driven, padanan NodeAdmin getAllRegisteredRoute). WAJIB dipanggil
// SETELAH route terdaftar (app.Build) — registry diisi saat registrasi route.
// Idempoten. Permission juga di-sync lazy tiap buka halaman Permission.
func SyncPermissions(db *gorm.DB) error {
	return accesssvc.NewPermissionService(db).SyncFromRoutes(context.Background())
}
