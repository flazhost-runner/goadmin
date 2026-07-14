// Package database membuka koneksi GORM yang dialect-agnostic (MySQL/Postgres/
// SQLite) berdasar config. Portabilitas dijaga di level kode (model & query),
// paket ini hanya memilih driver + mengatur connection pool.
package database

import (
	"fmt"
	"time"

	"github.com/glebarez/sqlite" // pure-Go sqlite (tanpa cgo) — untuk test in-memory
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"goadmin/internal/config"
)

// defaultMaxOpen dipakai bila DB_CONNECTION_LIMIT tak diisi/0 — 0 pada database/sql
// berarti TAK TERBATAS, yang akan menguras kuota DB terkelola.
const defaultMaxOpen = 10

// Open membuka koneksi DB sesuai cfg.DB.Type dan mengonfigurasi pool.
func Open(cfg *config.Config) (*gorm.DB, error) {
	dialector, err := dialector(cfg.DB)
	if err != nil {
		return nil, err
	}

	logLevel := logger.Silent
	if cfg.DB.Logging {
		logLevel = logger.Info
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger:                 logger.Default.LogMode(logLevel),
		SkipDefaultTransaction: true, // perf: kita kelola transaksi eksplisit bila perlu
	})
	if err != nil {
		return nil, fmt.Errorf("database: gagal open %s: %w", cfg.DB.Type, err)
	}

	if err := configurePool(db, cfg.DB); err != nil {
		return nil, err
	}
	return db, nil
}

// OpenSQLiteMemory membuka SQLite in-memory TERISOLASI (dipakai test — cepat,
// tanpa server). `name` membuat DB unik per pemanggil (cache=shared agar pool
// melihat skema yang sama, tapi nama unik mencegah test saling bocor state).
func OpenSQLiteMemory(name string) (*gorm.DB, error) {
	if name == "" {
		name = "test"
	}
	dsn := "file:" + name + "?mode=memory&cache=shared"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}
	// Batasi 1 koneksi agar DB in-memory tak hilang saat koneksi idle ditutup.
	if sqlDB, derr := db.DB(); derr == nil {
		sqlDB.SetMaxOpenConns(1)
	}
	return db, nil
}

func dialector(db config.DBConfig) (gorm.Dialector, error) {
	switch db.Type {
	case "mysql":
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=UTC",
			db.Username, db.Password, db.Host, db.Port, db.Database)
		return mysql.Open(dsn), nil
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
			db.Host, db.Port, db.Username, db.Password, db.Database)
		return postgres.Open(dsn), nil
	case "sqlite":
		name := db.Database
		if name == "" {
			name = "goadmin.db"
		}
		return sqlite.Open(name), nil
	default:
		return nil, fmt.Errorf("database: dialect '%s' tak didukung", db.Type)
	}
}

// poolLimits menurunkan batas pool PER REPLIKA dari config.
//
// Kuota DB terkelola dibagi ke SEMUA replika, jadi DB_CONNECTION_LIMIT harus kira-kira
// kuota_tier / maks_replika. Dua penjaga di sini:
//   - limit 0 tidak boleh berarti "tak terbatas" (default database/sql) — itu akan
//     menguras kuota DB begitu trafik naik;
//   - idle tak boleh melebihi open (mis. tier kecil limit=2 tapi idle=5), karena
//     database/sql diam-diam memangkasnya dan config jadi menyesatkan saat dibaca.
func poolLimits(cfg config.DBConfig) (open, idle int) {
	open = cfg.ConnMaxOpen
	if open <= 0 {
		open = defaultMaxOpen
	}
	idle = cfg.ConnMaxIdle
	if idle > open {
		idle = open
	}
	if idle < 0 {
		idle = 0
	}
	return open, idle
}

func configurePool(db *gorm.DB, cfg config.DBConfig) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("database: gagal ambil *sql.DB: %w", err)
	}
	open, idle := poolLimits(cfg)
	sqlDB.SetMaxOpenConns(open)
	sqlDB.SetMaxIdleConns(idle)

	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	} else {
		sqlDB.SetConnMaxLifetime(time.Hour)
	}
	// Lepas koneksi nganggur agar kuota bisa berpindah ke replika lain saat auto-scaling.
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}
	return nil
}
