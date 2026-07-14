// Package container melakukan wiring dependency terpusat (padanan container.ts
// + tsyringe di NodeAdmin). Idiom Go: constructor injection MANUAL — semua
// dependency dirakit di satu tempat, service menerima dependency lewat
// konstruktor (bukan di-`new` di dalam controller/route).
//
// Convention checker menolak controller/route yang meng-instansiasi service
// konkret langsung — service harus diambil dari Container ini.
package container

import (
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"goadmin/internal/config"
	"goadmin/internal/mail"
	"goadmin/internal/storage"
)

// Container menampung seluruh dependency tingkat-aplikasi yang sudah dirakit.
// Field service modul ditambahkan oleh masing-masing modul lewat Wire (lihat
// container.Services) agar paket container tak bergantung ke paket modul
// (menghindari import cycle) — modul mendaftarkan factory-nya sendiri.
type Container struct {
	Config *config.Config
	DB     *gorm.DB
	Redis   *redis.Client   // nil di test/mode tanpa redis
	Mailer  mail.Mailer     // SMTP bila dikonfigurasi, selain itu LogMailer (dev)
	Storage storage.Storage // penyimpanan file upload (lokal/disk)

	// Services adalah bag service ter-resolve, di-keyed oleh token interface.
	// Modul menyimpan implementasinya di sini saat Wire; controller mengambil
	// lewat helper typed Get*. Pendekatan ini menjaga container bebas-siklus.
	services map[string]interface{}
}

// New membuat container dengan dependency inti.
//
// Gagal bila storage object (s3/oss) tak bisa dirakit: lebih baik app menolak start
// daripada boot "sehat" tapi diam-diam menulis upload ke disk container sehingga file
// tak pernah sampai ke bucket.
func New(cfg *config.Config, db *gorm.DB, rdb *redis.Client) (*Container, error) {
	store, err := storage.New(cfg.Storage)
	if err != nil {
		return nil, err
	}
	return &Container{
		Config:   cfg,
		DB:       db,
		Redis:    rdb,
		Mailer:   mail.New(cfg.Mail),
		Storage:  store,
		services: map[string]interface{}{},
	}, nil
}

// MustNew = New yang panic bila storage gagal dirakit. Untuk test/bootstrap yang
// memakai storage lokal, di mana kegagalan berarti bug — bukan salah config user.
func MustNew(cfg *config.Config, db *gorm.DB, rdb *redis.Client) *Container {
	c, err := New(cfg, db, rdb)
	if err != nil {
		panic(err)
	}
	return c
}

// Provide menyimpan implementasi service di bawah token tertentu.
func (c *Container) Provide(token string, impl interface{}) {
	c.services[token] = impl
}

// Resolve mengambil service mentah by token (nil bila belum di-Provide).
func (c *Container) Resolve(token string) interface{} {
	return c.services[token]
}
