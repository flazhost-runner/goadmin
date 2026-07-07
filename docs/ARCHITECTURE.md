# Arsitektur GoAdmin

Port idiomatik NodeAdmin ke Go (Gin + GORM) yang mempertahankan **konsep & prinsip**, bukan kode. Lihat [`AGENTS.md`](../AGENTS.md) untuk aturan yang di-enforce.

## Prinsip

- **SOLID / DI** — constructor injection manual; service implement `I*Service`; wiring terpusat di `module.go` (composition root). Tidak ada `new` service di controller.
- **Separation of Concerns** — Controller (HTTP tipis) ≠ Service (bisnis) ≠ Model (data) ≠ View (presentasi).
- **Error terpusat** — service `return *errors.AppError`; controller `c.Error(err)`; `middleware.ErrorHandler` memetakan ke HTTP. Dilarang error telanjang di service.
- **DRY** — helper bersama (`paginate`, `ciLike`, response, id).
- **Config tervalidasi** — env hanya lewat `internal/config`; secret fail-fast di production.
- **Portabilitas DB** — tipe kolom abstrak, `ciLike`, tanpa raw SQL vendor → multi-DB nyata (bukan hanya "ORM multi-DB").
- **Twelve-Factor** — config env, stateless (sesi/blacklist eksternal), graceful shutdown, log ke stdout.

## Lifecycle request

```
Route (named) → middleware (auth → authorize → CSRF[web]/binding) → Controller
   → Service (logika, return AppError) → GORM → DB
                                       ↘ error → middleware.ErrorHandler (terpusat)
```

- **Web (sesi)**: `EnsureAuthenticatedWeb` → `AuthorizeWeb("subjek.aksi")`; CSRF + flash aktif; render `view.RenderView`.
- **API (JWT)**: `Guard.AuthenticatedJWT` (verifikasi tanda tangan + algoritma + **blacklist**) → `Authorize("subjek.aksi")`; respons JSON `{success, message, data}`.

## Lapisan

| Lapisan | Paket | Tanggung jawab |
|---|---|---|
| Entry | `cmd/server` | load config → DB → container → `app.Build` → listen + graceful shutdown |
| Perakitan | `internal/app` | engine Gin, middleware global, cabang full/api, mount web/api |
| DI | `internal/container` | menyimpan service ter-resolve (token = nama interface) |
| Config | `internal/config` | env tervalidasi (viper), tipe terkonversi |
| Error | `internal/errors` | `AppError{Status, Message, Detail, Fields}` |
| Helper | `internal/helpers` | `Paginate`/`Paginated[T]`, `CiLike`, `NewID`, response |
| Middleware | `internal/middleware` | error handler, security headers, gzip/CORS, **CSRF**, **rate-limit**, **flash** |
| Router | `internal/router` | registry modul (`Add`/`RegisterAll`) + named routes |
| View | `internal/view` | `RenderView` (html/template) + inject `currentUser`/`_csrf`/flash |
| Fitur | `internal/modules/*` | tiap modul: model/dto/service/controller/route/(view) |

## Modul & registrasi

Tiap modul mengimplementasi `router.Module` dan mendaftarkan diri lewat `init()` (blank-import di `internal/modules/modules.go`). `RegisterAll` menjalankan tiap `Register(ctx)` **urut nama** (deterministik). Modul UI didaftarkan dengan **guard kehadiran** (`ctx.Web != nil`) → absen otomatis di mode api.

Modul inti: **access** (User/Role/Permission/Auth — referensi pola), **setting**, **profile**, **dashboard**, **components** (web-only), **home** (web-only, landing publik).

### Dependency lintas-modul

- Service di-`Provide` ke container dengan token `"<modul>.I<X>Service"`.
- Modul lain me-`Resolve` token tersebut. Karena registrasi urut-nama, dependency yang terdaftar **belakangan** di-resolve **lazy** (provider-func dipanggil saat request) — mis. `home` → `setting`. Guard JWT bersama dari `access` (token `access.Guard`) menjaga blacklist konsisten lintas-modul.

## Varian Full vs API-only

Dipilih runtime via `APP_MODE`:

- **full** — pasang lapisan web (sesi, static, template, CSRF, flash) + REST API.
- **api** — hanya REST + JWT (stateless); modul web absen lewat guard kehadiran.

Diff antar-varian **purely-additive**: file shared identik, cabang lewat env/guard runtime — bukan dua project terpisah.

## Migrasi DB

Dua jalur, sesuai konteks (selaras `PORTING_GUIDE`):

- **Dev/test** — **AutoMigrate** GORM dari model (`internal/modules/*/migration/automigrate.go`, diagregasi `internal/bootstrap`). Cepat, dipakai `cmd/server` saat dev & `testutil`. SQLite in-memory.
- **Produksi** — **golang-migrate** versioned + **reversible**: file SQL `.up/.down` portabel di `internal/migrate/migrations/` (di-embed), dijalankan `cmd/migrate` untuk **mysql/postgres** (riwayat versi + rollback). Generator: `make migration name=...`.

`cmd/migrate` dialect-aware: sqlite → AutoMigrate; mysql/postgres → golang-migrate `up` (default) / `-down N` / `-force V` / `-version`. CI matrix menguji migrasi nyata di MySQL+Postgres (up→down→up). Portabilitas SQL juga diuji di SQLite (`tests/integration/migrate_test.go`).

## Penyimpanan file (ringkas)

`internal/storage` = abstraksi upload gambar dengan 3 driver (`local` disk /
`s3` untuk AWS S3 · OSS · MinIO), dipilih hanya lewat `STORAGE_DRIVER`. Driver
`local` disajikan via static mount `STORAGE_URL → STORAGE_DIR` di boot
(`internal/app/app.go`); URL prefix dipisah dari path filesystem agar
`STORAGE_DIR` absolut tetap valid. Cara ganti backend + caveat migrasi/volume
persisten: lihat bagian **Penyimpanan & ganti backend** di [`README`](../README.md).

## Keamanan (ringkas)

Security headers, CSRF (form web; API JWT dikecualikan), rate-limit login per-IP, RBAC `auth → authorize`, bcrypt, JWT HS256 + blacklist logout, cookie `HttpOnly`/`Secure`(prod), secret fail-fast, error generik di production. Detail di [`AGENTS.md`](../AGENTS.md) (Security Checklist).
