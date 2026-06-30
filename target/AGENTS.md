# AGENTS.md — Aturan Pengembangan GoAdmin (untuk AI & developer)

> **Sumber kebenaran tunggal.** Setiap AI (Claude Code, Cursor, Copilot, Codex) dan developer WAJIB mengikuti dokumen ini saat menambah/mengubah kode. File lain (`CLAUDE.md`, dll) hanya mirror tipis yang menunjuk ke sini.

GoAdmin adalah **port Go (Gin)** dari NodeAdmin — bootstrap admin panel yang dikembangkan menjadi aplikasi apa pun, memakai **idiom native Go** (bukan terjemahan TypeScript). Konsistensi dijaga oleh: dokumen ini + **convention checker** (`go run ./cmd/checkconventions`, alias `make check`; sumber: `cmd/checkconventions/main.go`) sebagai CI gate. Penyimpangan **ditolak CI**.

Pola acuan termutakhir: modul **`access`** (User/Role/Permission/Auth) — tiru strukturnya.

---

## Alur Wajib (request lifecycle)

```
Route (named-routes via router.Register)
  → middleware: EnsureAuthenticated{Web|JWT} → Authorize{Web}("subject.action") → binding DTO
  → gin.HandlerFunc (controller method)
  → Controller (inject service.I*Service)         // parsing + response, TANPA logika bisnis
  → Service (implements I*Service, return *errors.AppError bila gagal)
  → GORM model (db di-inject lewat konstruktor)
  → DB
  ↘ error → middleware.ErrorHandler (terpusat; AppError → HTTP status)
```

Urutan middleware RBAC **wajib**: autentikasi dulu, baru otorisasi (`EnsureAuthenticated*` → `Authorize*`). Administrator bypass permission.

## Prinsip Wajib

1. **SOLID / DI (manual constructor injection).** Service & controller menerima dependency lewat **konstruktor** (`NewXxxService(db, ...)`, `NewXxxController(svc)`), bukan global/`new` di tempat pakai. Wiring terpusat di **`module.go`** (`Register`) — itulah composition root; controller/route TIDAK merakit service sendiri.
   - Service mengimplementasi interface `I*Service` dengan **assertion compile**: `var _ IXxxService = (*XxxService)(nil)`.
   - Interface didefinisikan di `service/interfaces.go` (Dependency Inversion — konsumen bergantung ke abstraksi).
2. **DRY.** Pakai helper yang ADA di `internal/helpers/`: `Paginate()`/`Paginated[T]`, `CiLike()`/`CiLikeAny()`, `NewID()`/`NewCode()`, `OK()`/`Created()` (response). Render web lewat `view.RenderView()`. Jangan tulis ulang.
3. **Error handling.** Service mengembalikan **`*errors.AppError`** lewat konstruktor `apperr.NotFound/Conflict/Validation/Unauthorized/Forbidden/BadRequest/Internal`. Controller TIDAK memetakan status manual — cukup `c.Error(err)`; `middleware.ErrorHandler` yang memetakan ke HTTP. **Dilarang di service**: bikin error telanjang `errors.New(...)` / `fmt.Errorf(...)` (idiom Go `return err` atas error yang sudah ada tetap boleh; `errors.Is/As` boleh).
4. **Separation of Concerns.** Controller ≠ Service ≠ Model ≠ View. Logika bisnis hanya di service.
5. **Config terpusat.** Akses konfigurasi HANYA lewat `config.Config` yang **di-inject**. **Dilarang `os.Getenv` di dalam `internal/modules/`** — env dibaca sekali & tervalidasi di `internal/config`.
6. **Portabilitas DB (kode HARUS multi-DB, bukan cuma ORM-nya).** GORM mendukung MySQL/PG/SQLite, tapi aplikasi portabel hanya bila kode dijaga:
   - Model: tag gorm tipe abstrak (`varchar`/`text`/`bigint`/`boolean`); waktu pakai `time.Time` (tanpa `type:`). **Dilarang** `longtext`/`mediumtext`/`tinytext`/`datetime`/`enum`, dan **`collation`/`collate` hardcoded** (beda antar-dialek).
   - Status/kategori: pakai `varchar` + konstanta Go (mis. `StatusActive`), **bukan** `ENUM` native.
   - Query: **dilarang `.Raw()`/`.Exec()`** (raw SQL) & **`LIKE` manual** di modul (case-sensitivity beda MySQL vs PG/SQLite) — pakai **`helpers.CiLike/CiLikeAny`** (`LOWER(..) LIKE LOWER(..)`). Folder `migration/` dikecualikan (boleh DDL).
   - Test jalan di SQLite in-memory → membuktikan portabilitas. Checker menolak pelanggaran di atas.
   - **Migrasi**: dev/test pakai **AutoMigrate** (per-modul `migration/automigrate.go`, dari model). PRODUKSI (mysql/postgres) pakai **golang-migrate** versioned+reversible: SQL `.up/.down` di `internal/migrate/migrations/` (`make migration name=...` untuk file baru). Saat menambah/mengubah kolom model, **tambahkan juga migrasi SQL** agar skema produksi sinkron. CI menguji migrasi di matrix MySQL+Postgres.

## Sebelum Coding: Sajikan Rencana Artefak + Konfirmasi

Saat diminta membuat fitur/modul, AI **wajib** lebih dulu menyimpulkan artefak yang dibutuhkan (Matriks di bawah) lalu **menyajikan rencana** ke user. **Ajukan pertanyaan HANYA bila ambigu**; jika prompt sudah jelas, sajikan rencana lalu lanjut.

Pertanyaan klarifikasi yang umum perlu (bila ambigu): butuh **UI admin** (web) atau **API-only**? **Read-only** atau **CRUD**? Perlu endpoint **API**?

> Contoh: Fitur **Product**: model+migration, IProductService+ProductService, dto+validasi, controller (api+web), route, view CRUD, test (integration+api), update docs/API.md. → *Butuh UI admin atau API-only?*

## Matriks Kebutuhan Artefak

**TEST WAJIB untuk fitur APA PUN.** Tiap modul ber-service yang terjangkau lewat route harus punya ≥1 test yang menyebut modul/service/model-nya.

**Selalu ada** (modul fungsional ber-service): Service + `I*Service` · Controller · Route (≥1) · **Test** · update docs.

**Kondisional** (checker memaksa sesuai sifat modul):
| Artefak | Wajib JIKA |
|---------|------------|
| Model (entity) | menyimpan data |
| Migration | ada model (entity → migration) |
| DTO + binding validasi | ada input tulis (Store/Update) — anti mass-assignment |
| View | ada UI admin → route web wajib |
| Route API | fitur perlu API → api test + entri `docs/API.md` |

**API itu OPSIONAL** untuk modul baru — tawarkan ke user. Modul boleh web-only. Modul `access` sudah lengkap API+test sebagai **referensi pola utuh** — ikuti.

## Checklist Membuat Modul Baru

Ikuti [`docs/MODULE_GUIDE.md`](docs/MODULE_GUIDE.md). Struktur (mirror modul `access`):

```
internal/modules/<m>/
  model/<x>.go             # struct GORM, tag tipe portabel, TableName()
  migration/               # AutoMigrate/seeder (boleh raw SQL — dikecualikan checker)
  dto/<x>_dto.go           # input (binding:"required"...) + ListQuery
  service/interfaces.go    # I<X>Service
  service/<x>_service.go   # struct + `var _ I<X>Service = (*<X>Service)(nil)`, return *AppError
  controller/api/<x>_controller.go   # inject I<X>Service, helpers.OK/Created
  controller/web/<x>_controller.go   # view.RenderView (BUKAN c.HTML)
  middleware/              # bila perlu guard khusus modul
  route_api.go / route_web.go        # router.Register(nama, path) + urutan auth→authorize
  module.go                # init()→router.Add; Register(): wiring service + controller + route
```

## Security Checklist

- Route admin: `EnsureAuthenticated*` SEBELUM `Authorize*` (urutan wajib); Administrator bypass.
- Form web mutasi: token CSRF (saat lapisan CSRF dipasang) — jangan dilewati.
- Endpoint sensitif (login/reset OTP): rate-limit per-IP.
- Validasi semua input via DTO + `binding` tag (whitelist field) — cegah mass-assignment.
- JWT di-pin HS256; blacklist token saat logout (TTL = sisa berlaku). **Uji blacklist nyata** (login→200→logout→401) dengan store ber-perilaku seperti runtime.
- Password bcrypt (rounds dari config); OTP `crypto/rand` + hashed + expiry.
- Jangan bocorkan detail error ke user (ErrorHandler generik di production; `Detail` hanya log).
- Secret hanya dari `config`; fail-fast bila kosong di production. Jangan hardcode.

## DO NOT (akan ditolak CI / `make check`)

- ❌ `service.NewXxx(...)` / `service.XxxService{}` di **controller** → service di-inject lewat konstruktor (DI, wiring di `module.go`).
- ❌ `errors.New(...)` / `fmt.Errorf(...)` di **service** → pakai konstruktor `apperr.*`.
- ❌ Service tanpa assertion `var _ I<X>Service = (*<X>Service)(nil)`.
- ❌ `c.HTML(...)` di web controller → pakai `view.RenderView(c, "modul/view", gin.H{...})`.
- ❌ Tag gorm `longtext`/`datetime`/`enum`/`collation` dll (tak portabel) di model.
- ❌ `.Raw()`/`.Exec()` atau `LIKE` manual di `internal/modules/` → pakai `helpers.CiLike`.
- ❌ `os.Getenv` di `internal/modules/` → pakai `config`.
- ❌ Menambah modul ber-service tanpa test.
- ❌ Hardcode secret/kredensial.

## Definition of Done (modul/fitur)

- [ ] Mengikuti checklist & pola di atas.
- [ ] `make check` (convention checker) → lolos.
- [ ] `go vet ./...` → bersih.
- [ ] `go build ./...` → sukses.
- [ ] `go test ./...` → hijau (+ test baru untuk fitur).
- [ ] Security checklist terpenuhi.
- [ ] Docs diperbarui (README + `docs/API.md` bila ada API).

## Perintah Penting

```
make module ARGS="--name product"   # scaffold modul baru (ikut pola access)
make migrate                        # migrasi DB (sqlite→AutoMigrate, mysql/pg→golang-migrate) + seed
make migration name=add_orders      # buat file migrasi versioned (.up/.down.sql)
make check     # convention checker (WAJIB sebelum selesai)
make verify    # gate lengkap: check + vet + build + test
make test      # go test ./...
make run       # jalankan server (cmd/server)
make build     # go build ./...
make tidy      # go mod tidy
```

> Modul baru sebaiknya dibuat lewat **`make module`** (generator `cmd/make-module`) — output-nya otomatis lolos checker. Lihat [`docs/MODULE_GUIDE.md`](docs/MODULE_GUIDE.md).
