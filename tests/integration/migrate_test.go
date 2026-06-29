package integration

import (
	"os"
	"strings"
	"testing"

	"gorm.io/gorm"

	"goadmin/internal/database"
)

const migDir = "../../internal/migrate/migrations/"

// File SQL migrasi (yang dijalankan golang-migrate di mysql/postgres) juga
// HARUS valid di SQLite. Test mengeksekusi .up.sql lalu .down.sql terhadap
// SQLite → membuktikan SQL PORTABEL + REVERSIBLE (tabel dibuat lalu terhapus).
func TestMigrate_SQLPortableReversible(t *testing.T) {
	db, err := database.OpenSQLiteMemory(t.Name())
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	// --- up (urut) ---
	execSQLFile(t, db, "000001_create_access_tables.up.sql")
	execSQLFile(t, db, "000002_create_settings_table.up.sql")

	for _, tbl := range []string{"users", "roles", "permissions", "roles_permissions", "users_roles", "settings"} {
		if !db.Migrator().HasTable(tbl) {
			t.Fatalf("tabel %q seharusnya ada setelah up", tbl)
		}
	}

	// --- down (urut terbalik) ---
	execSQLFile(t, db, "000002_create_settings_table.down.sql")
	execSQLFile(t, db, "000001_create_access_tables.down.sql")

	for _, tbl := range []string{"users", "roles", "permissions", "settings"} {
		if db.Migrator().HasTable(tbl) {
			t.Fatalf("tabel %q seharusnya terhapus setelah down", tbl)
		}
	}
}

// execSQLFile membaca file migrasi, memecah per statement (;), lalu mengeksekusi.
func execSQLFile(t *testing.T, db *gorm.DB, name string) {
	t.Helper()
	raw, err := os.ReadFile(migDir + name)
	if err != nil {
		t.Fatalf("baca %s: %v", name, err)
	}
	for _, stmt := range strings.Split(string(raw), ";") {
		if strings.TrimSpace(stmt) == "" {
			continue
		}
		if err := db.Exec(stmt).Error; err != nil {
			t.Fatalf("%s — exec gagal: %v\n--- stmt ---\n%s", name, err, stmt)
		}
	}
}
