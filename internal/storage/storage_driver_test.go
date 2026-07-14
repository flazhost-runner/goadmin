package storage_test

import (
	"strings"
	"testing"

	"goadmin/internal/config"
	"goadmin/internal/storage"
)

func ossConfig() config.StorageConfig {
	return config.StorageConfig{
		Driver: "oss",
		// Endpoint sengaja ditulis lengkap dengan skema — begitulah orang menyalinnya
		// dari konsol Aliyun, dan bentuk inilah yang dulu ditolak minio.New lalu
		// ditelan diam-diam menjadi storage lokal.
		S3Endpoint:  "https://oss-ap-southeast-5.aliyuncs.com",
		S3Bucket:    "my-bucket",
		S3AccessKey: "key",
		S3SecretKey: "secret",
		S3UseSSL:    true,
	}
}

// Driver "oss" WAJIB menghasilkan object storage, bukan Local. Sebelum perbaikan,
// New() hanya mengenali "s3" sehingga STORAGE_DRIVER=oss menulis file ke disk container:
// upload tampak sukses tapi tak pernah sampai ke bucket.
func TestNewOSSDriverIsNotLocal(t *testing.T) {
	store, err := storage.New(ossConfig())
	if err != nil {
		t.Fatalf("driver oss harus bisa dirakit, dapat error: %v", err)
	}
	if _, isLocal := store.(*storage.Local); isLocal {
		t.Fatal("driver oss jatuh ke Local — file tidak akan pernah sampai ke bucket")
	}
}

// Endpoint dengan skema harus diterima (skema di-strip), bukan gagal senyap.
func TestNewS3AcceptsSchemedEndpoint(t *testing.T) {
	cfg := ossConfig()
	cfg.Driver = "s3"
	if _, err := storage.New(cfg); err != nil {
		t.Fatalf("endpoint dengan skema https:// harus diterima, dapat error: %v", err)
	}
}

// Config object storage yang tak lengkap harus GAGAL berisik, bukan diam-diam
// jatuh ke disk lokal.
func TestNewObjectDriverFailsLoudlyOnIncompleteConfig(t *testing.T) {
	cfg := ossConfig()
	cfg.S3AccessKey = ""
	cfg.S3Bucket = ""

	store, err := storage.New(cfg)
	if err == nil {
		t.Fatalf("config tak lengkap harus error, malah dapat storage %T", store)
	}
	for _, want := range []string{"STORAGE_BUCKET", "STORAGE_ACCESS_KEY_ID"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("pesan error harus menyebut %s, dapat: %v", want, err)
		}
	}
}

// Driver lokal tetap lokal.
func TestNewLocalDriver(t *testing.T) {
	store, err := storage.New(config.StorageConfig{Driver: "local", Dir: "web/uploads", URLBase: "/uploads"})
	if err != nil {
		t.Fatalf("driver local tidak boleh error: %v", err)
	}
	if _, isLocal := store.(*storage.Local); !isLocal {
		t.Fatalf("driver local harus menghasilkan *storage.Local, dapat %T", store)
	}
}

// Static mount /uploads hanya untuk disk lokal — "oss" dulu ikut lolos ke sini.
func TestIsObjectDriver(t *testing.T) {
	cases := map[string]bool{"s3": true, "S3": true, "oss": true, "OSS": true, "local": false, "": false}
	for driver, want := range cases {
		if got := storage.IsObjectDriver(driver); got != want {
			t.Errorf("IsObjectDriver(%q) = %v, mau %v", driver, got, want)
		}
	}
}
