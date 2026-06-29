// Package migration menyediakan AutoMigrate (dev/test) + seeder modul access.
// Untuk PRODUKSI gunakan golang-migrate (file .up/.down.sql portabel) via
// cmd/migrate — AutoMigrate hanya untuk iterasi cepat & test SQLite in-memory.
package migration

import (
	"gorm.io/gorm"

	"goadmin/internal/modules/access/model"
)

// Models mengembalikan seluruh entity modul access (untuk AutoMigrate).
func Models() []interface{} {
	return []interface{}{
		&model.Permission{},
		&model.Role{},
		&model.User{},
	}
}

// AutoMigrate membuat/menyelaraskan skema modul access (dev/test).
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}
