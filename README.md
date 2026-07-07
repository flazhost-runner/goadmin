# GoAdmin

Bootstrap **admin panel** dalam Go (Gin + GORM) — port idiomatik dari [NodeAdmin](../NodeAdmin). Modular per fitur, RBAC, auth ganda (sesi web + JWT API), theme switcher, multi-database, dengan **guardrail** (convention checker + generator modul) yang menjaga konsistensi saat dikembangkan.

> Satu basis kode, dua varian dipilih runtime via `APP_MODE`: **full** (UI + API) atau **api** (REST + JWT saja).

---

## Mulai cepat (SQLite, tanpa server DB)

```bash
cp .env.example .env          # default: APP_MODE=full, DB sqlite
make migrate                  # buat tabel + seed admin (idempoten)
make run                      # http://localhost:3000
```

Login admin default: **admin@admin.com / 12345678**.
Landing publik di `/`, login di `/auth/login`, dashboard di `/admin/v1/dashboard`.

> Ganti DB ke MySQL/Postgres cukup lewat `.env` (`DB_TYPE=...`) — tanpa ubah kode.

## Perintah (Makefile)

```
make migrate   # migrasi DB: sqlite→AutoMigrate (dev), mysql/postgres→golang-migrate
make migration name=add_orders   # buat file migrasi baru (.up/.down.sql)
make run       # jalankan server (cmd/server)
make check     # convention checker (gate pola/prinsip)
make verify    # check + vet + build + test  ← gate "selesai"
make test      # go test ./...
make module ARGS="--name product"   # scaffold modul baru lengkap
```

## Fitur

- **RBAC** — User · Role · Permission (many2many), per-route, Administrator bypass. CRUD penuh via web **dan** REST API.
- **Auth ganda** — sesi web (cookie, Redis-ready) + **JWT** (HS256 di-pin) dengan **blacklist saat logout** (TTL = sisa berlaku). **Reset password via OTP email** (crypto-random, ter-hash, expiry, rate-limited).
- **Keamanan web** — CSRF token (form), rate-limit login per-IP, security headers (helmet setara), bcrypt, secret fail-fast di production, error generik di production (HTML untuk web, JSON untuk API).
- **Upload gambar aman** — `internal/storage` driver **local/disk atau S3/OSS/MinIO** (via `STORAGE_DRIVER`), validasi **magic-byte** (bukan MIME klien) + whitelist + batas ukuran + **re-encode** (buang metadata/payload); dipakai logo (Setting) & avatar (Profil/User).
- **Setting global** — singleton ber-**cache** (TTL + invalidasi saat update) + **theme switcher** (palet di DB, ganti tanpa rebuild).
- **Profile** self-service (least-privilege: tak bisa ubah status/role sendiri).
- **Dashboard** statistik, **Components** showcase UI, **Home** landing publik data-driven ke Setting + **frontend template switcher DI-FOLD ke halaman Setting** (`/admin/v1/setting`, persis NodeAdmin — bukan halaman terpisah; preview proxy `admin.v1.setting.fe_preview` = `/admin/v1/setting/fe-preview/:slug`): builtin (Go view) + katalog **eksternal opentailwind** (opsional `FE_TEMPLATE_REMOTE=true` → fetch daftar + unduh on-demand + cache + anti-SSRF; default off → katalog kurasi).
- **Email/SMTP** — `internal/mail` (SMTP via `net/smtp`, fallback log saat dev) tersedia di container untuk reset OTP/notifikasi.
- **Multi-database** (MySQL/Postgres/SQLite) dialect-agnostic; kode dijaga portabel (tipe abstrak, `ciLike`, tanpa raw SQL vendor) — di-enforce checker.
- **Guardrail** — `cmd/checkconventions` (CI gate) + generator `make module` + `AGENTS.md`.

## Environment

Semua konfigurasi via `.env` (lihat [`.env.example`](.env.example)), dibaca **hanya** lewat `internal/config`. Kunci utama: `APP_MODE`, `DB_TYPE`/`DB_*`, `REDIS_URL`, `SESSION_SECRET`, `JWT_SECRET`, `BCRYPT_ROUNDS`, `CORS_ORIGINS`. Secret **wajib** di production (app berhenti bila kosong).

## Penyimpanan & ganti backend

Upload gambar (logo Setting, avatar Profil/User, media editor) melewati satu
abstraksi `internal/storage` dengan **3 driver**, dipilih **hanya** lewat
`STORAGE_DRIVER` di `.env` — tanpa ubah kode/view:

| `STORAGE_DRIVER` | Simpan ke | URL render |
| --- | --- | --- |
| `local` (default) | disk (`STORAGE_DIR`, mis. `web/uploads`) | **relatif** `STORAGE_URL/<nama>` (mis. `/uploads/xxxx.jpg`) — disajikan oleh app |
| `s3` (endpoint AWS S3) | bucket S3 | **absolut** `STORAGE_PUBLIC_URL/<key>` (mis. `https://cdn.example.com/images/xxxx.jpg`) |
| `s3` (endpoint OSS/MinIO) | bucket OSS/MinIO S3-compatible | **absolut** `STORAGE_PUBLIC_URL/<key>` |

> Aliyun OSS & MinIO diakses lewat driver `s3` yang sama (S3-compatible) —
> bedakan hanya lewat `STORAGE_ENDPOINT` + `STORAGE_PUBLIC_URL`.

**Cara kerja driver `local`.** Prefix URL (`STORAGE_URL`) **sengaja dipisah** dari
path filesystem (`STORAGE_DIR`). Saat `driver=local`, app memasang static mount
`STORAGE_URL → STORAGE_DIR` di boot (lihat `internal/app/app.go`), lalu render
mengembalikan `STORAGE_URL/<nama>`. Karena keduanya terpisah, `STORAGE_DIR`
**absolut** (mis. `/app/storage` di Docker) tetap menghasilkan URL yang valid.
Saat `driver=s3` tak ada mount lokal — URL sudah absolut (public/CDN).

**Ganti backend.** Ubah `STORAGE_DRIVER` (+ isi `STORAGE_ENDPOINT`,
`STORAGE_BUCKET`, `STORAGE_ACCESS_KEY_ID`, `STORAGE_SECRET_ACCESS_KEY`,
`STORAGE_PUBLIC_URL`) lalu **restart** app. Tidak ada perubahan kode.

**Caveat migrasi (penting).** Mengganti driver **tidak** memindahkan berkas lama:
gambar yang sudah ada di disk tidak otomatis pindah ke bucket. Salin isi
`STORAGE_DIR` ke bucket dengan **mempertahankan key/prefix** (mis. `images/`,
`editor/`) sebelum switch:

```bash
# AWS S3
aws s3 sync web/uploads/ s3://<bucket>/
# Aliyun OSS
ossutil cp -r web/uploads/ oss://<bucket>/
```

**Produksi + driver `local` bersifat ephemeral.** Filesystem kontainer hilang saat
redeploy/scale → berkas terunggah lenyap dan tidak terbagi antar-replika. Untuk
`local` di produksi **wajib** pasang **volume persisten** yang di-mount ke
`STORAGE_DIR` (mis. mount ke `/app/storage` lalu set `STORAGE_DIR=/app/storage`),
atau — lebih disarankan untuk multi-replika — gunakan driver `s3`.

> Isi folder upload **di-ignore git** (`/web/uploads/*`); folder dipertahankan via
> `web/uploads/.gitkeep`. Jangan commit berkas yang diunggah user.

## Struktur

```
cmd/
  server/           entry-point (full/api via APP_MODE), graceful shutdown
  migrate/          migrasi: golang-migrate (mysql/pg) / AutoMigrate (sqlite) + seed
internal/migrate/   migrasi VERSIONED golang-migrate (migrations/*.up/.down.sql)
  checkconventions/ convention checker (go/ast)
  make-module/      generator modul (text/template)
internal/
  app/              perakitan engine Gin (middleware global, mount web/api)
  config/           env tervalidasi (viper)
  container/        DI manual terpusat (composition root)
  database/         koneksi GORM dialect-agnostic + SQLite in-memory (test)
  errors/           AppError (status HTTP + pesan publik)
  helpers/          paginate, ciLike, id, response (DRY)
  middleware/       error handler, security headers, CSRF, rate-limit, flash
  router/           registry modul + named routes
  view/             RenderView (html/template) + inject currentUser/_csrf/flash
  modules/          fitur: access, setting, profile, dashboard, components, home
web/templates/      layout/partials (chrome admin) bersama
tests/              unit · integration · api
```

## Dokumentasi

- [`AGENTS.md`](AGENTS.md) — aturan pengembangan (sumber kebenaran) + DO/DON'T.
- [`docs/ARCHITECTURE.md`](docs/ARCHITECTURE.md) — arsitektur, lifecycle, DI, varian.
- [`docs/MODULE_GUIDE.md`](docs/MODULE_GUIDE.md) — cara membuat modul (manual & generator).
- [`docs/TESTING.md`](docs/TESTING.md) — lapisan test & cara menjalankan.
- [`docs/API.md`](docs/API.md) — daftar endpoint REST.
- [`docs/postman/GoAdmin.postman_collection.json`](docs/postman/GoAdmin.postman_collection.json) — koleksi Postman (impor & jalankan). Variabel `base_url` default `http://localhost:3000`.

## Testing

```bash
make verify   # konvensi + vet + build + test (gate)
make test     # hanya test
```

Test berjalan di **SQLite in-memory** (cepat, membuktikan portabilitas). Lihat [`docs/TESTING.md`](docs/TESTING.md).

## Deployment (ringkas)

1. Set `NODE_ENV=production`, `DB_TYPE` + kredensial DB, `REDIS_URL`, dan **`SESSION_SECRET`/`JWT_SECRET`** (wajib).
2. Migrasi: **produksi** pakai migrasi **versioned + reversible** (golang-migrate, SQL `.up/.down` di `internal/migrate/migrations/`) — `make migrate` (mysql/postgres), rollback `make migrate ARGS="-down 1"`, cek versi `make migrate ARGS="-version"`. **Dev** (sqlite) pakai AutoMigrate cepat dari model. CI menguji migrasi nyata di matrix MySQL+Postgres (up→down→up).
3. `go build -o goadmin ./cmd/server` lalu jalankan di belakang reverse proxy (TLS). Cookie otomatis `Secure` di production.
4. Stateless → siap horizontal scaling (sesi/blacklist di Redis).

## Lisensi

ISC.
