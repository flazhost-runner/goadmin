// Package testutil menyediakan setup test bersama: DB SQLite in-memory ter-migrate
// + container + app engine. Memakai store dengan FIDELITY perilaku runtime
// (mis. MemoryBlacklist yang meniru TTL Redis), bukan mock "selalu mulus".
package testutil

import (
	"testing"

	"gorm.io/gorm"

	"goadmin/internal/config"
	"goadmin/internal/container"
	"goadmin/internal/database"
	accessmig "goadmin/internal/modules/access/migration"
)

// TestConfig membuat config minimal untuk test (mode penuh, bcrypt cepat).
func TestConfig(mode config.AppMode) *config.Config {
	return &config.Config{
		Env:    "test",
		IsTest: true,
		App:    config.AppConfig{Name: "GoAdmin Test", Port: 0, Mode: mode},
		DB:     config.DBConfig{Type: "sqlite"},
		Session: config.SessionConfig{Secret: "test-session-secret"},
		JWT:    config.JWTConfig{Secret: "test-jwt-secret", Algorithm: "HS256"},
		Security: config.SecurityConfig{BcryptRounds: 4}, // rounds rendah → test cepat
	}
}

// NewDB membuat SQLite in-memory ter-migrate (skema modul access), terisolasi
// per-test (nama unik dari t.Name()).
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := database.OpenSQLiteMemory(t.Name())
	if err != nil {
		t.Fatalf("buka sqlite: %v", err)
	}
	if err := accessmig.AutoMigrate(db); err != nil {
		t.Fatalf("automigrate: %v", err)
	}
	return db
}

// NewContainer merakit container test (tanpa redis → store in-memory fidelity).
func NewContainer(t *testing.T, mode config.AppMode) *container.Container {
	t.Helper()
	cfg := TestConfig(mode)
	db := NewDB(t)
	return container.MustNew(cfg, db, nil)
}

// SeedAdmin menyemai admin default + RBAC, mengembalikan kredensial login.
func SeedAdmin(t *testing.T, c *container.Container) (email, password string) {
	t.Helper()
	email, password = "admin@goadmin.test", "secret123"
	if err := accessmig.Seed(c.DB, email, password, c.Config.Security.BcryptRounds); err != nil {
		t.Fatalf("seed admin: %v", err)
	}
	return email, password
}
