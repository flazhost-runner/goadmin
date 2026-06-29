// Package migrate menjalankan migrasi SKEMA VERSIONED & REVERSIBLE memakai
// golang-migrate (file SQL .up/.down portabel di migrations/). Ini jalur
// PRODUKSI (punya riwayat + rollback), pelengkap AutoMigrate yang dipakai
// dev/test (lihat internal/bootstrap).
package migrate

import (
	"embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/mysql"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	// CATATAN: driver sqlite golang-migrate (modernc) TIDAK diimpor karena
	// bentrok register driver "sqlite" dengan glebarez/sqlite (GORM). SQLite =
	// jalur dev/test → pakai AutoMigrate (internal/bootstrap). golang-migrate
	// ini untuk PRODUKSI (mysql/postgres) yang butuh versioned+reversible.
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"goadmin/internal/config"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// newMigrate merakit instance golang-migrate (source embed + DB dari DSN).
func newMigrate(db config.DBConfig) (*migrate.Migrate, error) {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return nil, err
	}
	dsn, err := dsnURL(db)
	if err != nil {
		return nil, err
	}
	return migrate.NewWithSourceInstance("iofs", src, dsn)
}

// dsnURL membangun URL koneksi golang-migrate per dialek.
func dsnURL(db config.DBConfig) (string, error) {
	switch db.Type {
	case "mysql":
		// multiStatements: izinkan banyak statement per file migrasi.
		// sql_mode=ANSI_QUOTES: izinkan identifier ber-double-quote ("desc") agar
		// SATU file SQL portabel (sama dgn Postgres) bisa memuat kolom reserved
		// word `desc`. Backtick GORM tetap valid di runtime (koneksi terpisah).
		return fmt.Sprintf("mysql://%s:%s@tcp(%s:%d)/%s?multiStatements=true&sql_mode=ANSI_QUOTES",
			db.Username, db.Password, db.Host, db.Port, db.Database), nil
	case "postgres":
		return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
			db.Username, db.Password, db.Host, db.Port, db.Database), nil
	case "sqlite":
		return "", fmt.Errorf("migrate: sqlite memakai AutoMigrate (dev/test), bukan golang-migrate")
	default:
		return "", fmt.Errorf("migrate: dialect '%s' tak didukung", db.Type)
	}
}

// Up menerapkan semua migrasi yang belum dijalankan (no-op bila sudah terbaru).
func Up(db config.DBConfig) error {
	m, err := newMigrate(db)
	if err != nil {
		return err
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// Down memutar mundur n langkah (rollback). n<=0 → semua.
func Down(db config.DBConfig, n int) error {
	m, err := newMigrate(db)
	if err != nil {
		return err
	}
	defer m.Close()
	if n <= 0 {
		if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		return nil
	}
	if err := m.Steps(-n); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

// Force menyetel versi paksa (memulihkan state "dirty" setelah gagal).
func Force(db config.DBConfig, v int) error {
	m, err := newMigrate(db)
	if err != nil {
		return err
	}
	defer m.Close()
	return m.Force(v)
}

// Version mengembalikan versi terakhir + status dirty.
func Version(db config.DBConfig) (uint, bool, error) {
	m, err := newMigrate(db)
	if err != nil {
		return 0, false, err
	}
	defer m.Close()
	v, dirty, err := m.Version()
	if errors.Is(err, migrate.ErrNilVersion) {
		return 0, false, nil
	}
	return v, dirty, err
}

// Create menghasilkan pasangan file migrasi kosong ber-nomor urut berikutnya
// di internal/migrate/migrations (untuk dijalankan saat dev menambah skema).
func Create(name string) (string, error) {
	dir := filepath.Join("internal", "migrate", "migrations")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}
	next := nextSeq(entries)
	base := fmt.Sprintf("%06d_%s", next, name)
	for _, suf := range []string{".up.sql", ".down.sql"} {
		p := filepath.Join(dir, base+suf)
		if err := os.WriteFile(p, []byte("-- "+base+suf+"\n"), 0o644); err != nil {
			return "", err
		}
	}
	return base, nil
}

// nextSeq mencari nomor urut migrasi tertinggi + 1.
func nextSeq(entries []os.DirEntry) int {
	max := 0
	for _, e := range entries {
		var n int
		if _, err := fmt.Sscanf(e.Name(), "%06d_", &n); err == nil && n > max {
			max = n
		}
	}
	return max + 1
}
