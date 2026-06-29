// Package migration menyediakan AutoMigrate modul setting (dev/test).
// Untuk PRODUKSI gunakan golang-migrate (.up/.down.sql portabel).
//
// Baris setting (singleton) TIDAK di-seed di sini — SettingService.Get membuat
// default secara lazy bila tabel kosong (idempoten, aman di runtime & test).
package migration

import (
	"gorm.io/gorm"

	"goadmin/internal/modules/setting/model"
)

// Models mengembalikan entity modul setting (untuk AutoMigrate).
func Models() []interface{} {
	return []interface{}{
		&model.Setting{},
	}
}

// AutoMigrate membuat/menyelaraskan skema modul setting (dev/test).
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(Models()...)
}
