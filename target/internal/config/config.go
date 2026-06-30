// Package config memuat konfigurasi environment terpusat & tervalidasi.
//
// Prinsip (sejajar NodeAdmin src/config/env.ts):
//   - Akses env HANYA lewat paket ini. Modul TIDAK boleh memanggil os.Getenv
//     langsung (di-enforce convention checker).
//   - Secret wajib (SESSION_SECRET, JWT_SECRET) → fail-fast bila kosong di
//     production, agar app tak pernah jalan dengan secret default yang bisa ditebak.
//   - Tipe sudah dikonversi (int/bool/duration), bukan string mentah.
//   - Sumber: file .env (via viper) + environment OS (override).
package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// AppMode menentukan varian aplikasi yang dijalankan dari satu basis kode.
type AppMode string

const (
	// ModeFull memasang lapisan web (sesi, static, layout, route web) + REST API.
	ModeFull AppMode = "full"
	// ModeAPI hanya REST + JWT (stateless), melewati lapisan web.
	ModeAPI AppMode = "api"
)

// Config adalah konfigurasi tervalidasi seluruh aplikasi.
type Config struct {
	Env    string // development | production | test
	IsProd bool
	IsTest bool
	App    AppConfig
	DB     DBConfig
	Redis  RedisConfig
	Session  SessionConfig
	JWT      JWTConfig
	Security SecurityConfig
	Mail     MailConfig
	FeTemplate FeTemplateConfig
	Storage    StorageConfig
}

type AppConfig struct {
	Host string
	Port int
	Name string
	Mode AppMode // 'full' (UI+API) atau 'api' (API saja)
}

type DBConfig struct {
	// Type: mysql | postgres | sqlite — dialect-agnostic (ganti cukup lewat env).
	Type            string
	Host            string
	Port            int
	Username        string
	Password        string
	Database        string
	Logging         bool
	ConnMaxOpen     int
	ConnMaxIdle     int
	ConnMaxLifetime time.Duration
}

type RedisConfig struct {
	URL string
}

type SessionConfig struct {
	Driver string // cookie | redis (default: cookie)
	Secret string
	TTL    time.Duration
}

type JWTConfig struct {
	Secret    string
	ExpiresIn time.Duration
	Algorithm string // di-pin HS256
}

type SecurityConfig struct {
	BcryptRounds    int
	OTPExpiryMinutes int
	DefaultPageSize  int
	CORSOrigins     []string
}

// MailConfig = pengiriman email. Host kosong → mailer fallback (log saja, dev).
// Nama env mengikuti standar NodeAdmin: MAIL_HOST, MAIL_PORT, MAIL_SECURE,
// MAIL_USERNAME, MAIL_PASSWORD, MAIL_FROM_NAME, MAIL_FROM_ADDRESS.
type MailConfig struct {
	Host        string
	Port        int
	Secure      bool
	Username    string
	Password    string
	FromName    string
	FromAddress string
}

// StorageConfig = penyimpanan file upload (gambar). Driver "local" (disk) atau
// "s3" (S3/OSS/MinIO-compatible). Lokal disajikan static di URLBase.
// Nama env mengikuti standar NodeAdmin: STORAGE_DRIVER, STORAGE_ACCESS_KEY_ID, dll.
type StorageConfig struct {
	Driver  string // local | s3
	Dir     string // (local) folder, mis. web/uploads
	URLBase string // (local) prefix URL publik, mis. /uploads
	// S3 (driver=s3) — env names: STORAGE_ACCESS_KEY_ID, STORAGE_SECRET_ACCESS_KEY, dll.
	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool
	S3PublicURL string // base URL publik objek (mis. https://cdn.example.com)
}

// FeTemplateConfig = frontend template switcher (katalog landing eksternal).
type FeTemplateConfig struct {
	Remote     bool
	TreeURL    string
	RawBaseURL string
	CacheDir   string
}

// Load membaca konfigurasi dari .env + environment dan memvalidasinya.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigName(".env")
	v.SetConfigType("env")
	v.AddConfigPath(".")
	v.AutomaticEnv()
	_ = v.ReadInConfig()

	setDefaults(v)

	env := strings.ToLower(v.GetString("NODE_ENV"))
	if env == "" {
		env = "development"
	}
	isProd := env == "production"
	isTest := env == "test"

	mode := ModeFull
	if strings.ToLower(v.GetString("APP_MODE")) == "api" {
		mode = ModeAPI
	}

	cfg := &Config{
		Env:    env,
		IsProd: isProd,
		IsTest: isTest,
		App: AppConfig{
			Host: v.GetString("APP_HOST"),
			Port: v.GetInt("APP_PORT"),
			Name: v.GetString("APP_NAME"),
			Mode: mode,
		},
		DB: DBConfig{
			Type:            strings.ToLower(v.GetString("DB_TYPE")),
			Host:            v.GetString("DB_HOST"),
			Port:            v.GetInt("DB_PORT"),
			Username:        v.GetString("DB_USERNAME"),
			Password:        v.GetString("DB_PASSWORD"),
			Database:        v.GetString("DB_DATABASE"),
			Logging:         v.GetBool("DB_LOGGING"),
			ConnMaxOpen:     v.GetInt("DB_CONNECTION_LIMIT"),
			ConnMaxIdle:     v.GetInt("DB_CONNECTION_IDLE"),
			ConnMaxLifetime: time.Duration(v.GetInt("DB_CONNECTION_LIFETIME_MIN")) * time.Minute,
		},
		Redis: RedisConfig{
			URL: v.GetString("REDIS_URL"),
		},
		Session: SessionConfig{
			Driver: v.GetString("SESSION_DRIVER"),
			Secret: v.GetString("SESSION_SECRET"),
			TTL:    time.Duration(v.GetInt("SESSION_TTL_HOURS")) * time.Hour,
		},
		JWT: JWTConfig{
			Secret:    v.GetString("JWT_SECRET"),
			ExpiresIn: parseJWTExpiry(v.GetString("JWT_EXPIRES_IN"), 60*time.Minute),
			Algorithm: "HS256",
		},
		Security: SecurityConfig{
			BcryptRounds:    v.GetInt("BCRYPT_ROUNDS"),
			OTPExpiryMinutes: v.GetInt("OTP_EXPIRY_MINUTES"),
			DefaultPageSize:  v.GetInt("DEFAULT_PAGE_SIZE"),
			CORSOrigins:     splitAndTrim(v.GetString("CORS_ORIGINS")),
		},
		Mail: MailConfig{
			Host:        v.GetString("MAIL_HOST"),
			Port:        v.GetInt("MAIL_PORT"),
			Secure:      v.GetBool("MAIL_SECURE"),
			Username:    v.GetString("MAIL_USERNAME"),
			Password:    v.GetString("MAIL_PASSWORD"),
			FromName:    v.GetString("MAIL_FROM_NAME"),
			FromAddress: v.GetString("MAIL_FROM_ADDRESS"),
		},
		FeTemplate: FeTemplateConfig{
			Remote:     v.GetBool("FE_TEMPLATE_REMOTE"),
			TreeURL:    v.GetString("FE_TEMPLATE_TREE_URL"),
			RawBaseURL: v.GetString("FE_TEMPLATE_RAW_URL"),
			CacheDir:   v.GetString("FE_TEMPLATE_CACHE_DIR"),
		},
		Storage: StorageConfig{
			Driver:      v.GetString("STORAGE_DRIVER"),
			Dir:         v.GetString("STORAGE_DIR"),
			URLBase:     v.GetString("STORAGE_URL"),
			S3Endpoint:  v.GetString("STORAGE_ENDPOINT"),
			S3Region:    v.GetString("STORAGE_REGION"),
			S3Bucket:    v.GetString("STORAGE_BUCKET"),
			S3AccessKey: v.GetString("STORAGE_ACCESS_KEY_ID"),
			S3SecretKey: v.GetString("STORAGE_SECRET_ACCESS_KEY"),
			S3UseSSL:    v.GetBool("STORAGE_SSL"),
			S3PublicURL: v.GetString("STORAGE_PUBLIC_URL"),
		},
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("APP_HOST", "http://localhost")
	v.SetDefault("APP_PORT", 3000)
	v.SetDefault("APP_NAME", "Go Admin")
	v.SetDefault("DB_TYPE", "mysql")
	v.SetDefault("DB_PORT", 3306)
	v.SetDefault("DB_LOGGING", false)
	v.SetDefault("DB_CONNECTION_LIMIT", 10)
	v.SetDefault("DB_CONNECTION_IDLE", 5)
	v.SetDefault("DB_CONNECTION_LIFETIME_MIN", 60)
	v.SetDefault("REDIS_URL", "redis://127.0.0.1:6379")
	v.SetDefault("SESSION_TTL_HOURS", 6)
	v.SetDefault("JWT_EXPIRES_IN", "1h")
	v.SetDefault("BCRYPT_ROUNDS", 10)
	v.SetDefault("OTP_EXPIRY_MINUTES", 10)
	v.SetDefault("DEFAULT_PAGE_SIZE", 10)
	v.SetDefault("CORS_ORIGINS", "")
	v.SetDefault("MAIL_PORT", 587)
	v.SetDefault("MAIL_SECURE", false)
	v.SetDefault("MAIL_FROM_ADDRESS", "no-reply@goadmin.local")
	v.SetDefault("MAIL_FROM_NAME", "GoAdmin")
	v.SetDefault("FE_TEMPLATE_REMOTE", true)
	v.SetDefault("FE_TEMPLATE_TREE_URL", "https://api.github.com/repos/lindoai/opentailwind/git/trees/master?recursive=1")
	v.SetDefault("FE_TEMPLATE_RAW_URL", "https://raw.githubusercontent.com/lindoai/opentailwind/master/landings")
	v.SetDefault("FE_TEMPLATE_CACHE_DIR", "web/cache/fetemplates")
	v.SetDefault("SESSION_DRIVER", "database")
	v.SetDefault("STORAGE_DRIVER", "local")
	v.SetDefault("STORAGE_DIR", "web/uploads")
	v.SetDefault("STORAGE_URL", "/uploads")
	v.SetDefault("STORAGE_SSL", true)
}

// validate menerapkan aturan fail-fast: secret wajib di production.
func (c *Config) validate() error {
	if c.IsProd {
		var missing []string
		if c.Session.Secret == "" {
			missing = append(missing, "SESSION_SECRET")
		}
		if c.JWT.Secret == "" {
			missing = append(missing, "JWT_SECRET")
		}
		if len(missing) > 0 {
			return fmt.Errorf("config: secret wajib di production kosong: %s", strings.Join(missing, ", "))
		}
	}
	if c.Session.Secret == "" {
		c.Session.Secret = "dev-session-secret-change-me"
	}
	if c.JWT.Secret == "" {
		c.JWT.Secret = "dev-jwt-secret-change-me"
	}
	switch c.DB.Type {
	case "mysql", "postgres", "sqlite":
	default:
		return fmt.Errorf("config: DB_TYPE '%s' tak didukung (mysql|postgres|sqlite)", c.DB.Type)
	}
	return nil
}

// parseJWTExpiry mem-parse string durasi seperti "1h", "30m", "24h".
// Fallback ke nilai default bila kosong atau tidak valid.
func parseJWTExpiry(s string, fallback time.Duration) time.Duration {
	if s == "" {
		return fallback
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return fallback
	}
	return d
}

func splitAndTrim(s string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimRight(strings.TrimSpace(p), "/")
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
