# Testing GoAdmin

Test berjalan di **SQLite in-memory** (pure-Go `glebarez/sqlite`, tanpa cgo/server) → cepat dan **membuktikan portabilitas** kode lintas dialek. CI dapat menjalankan matrix MySQL/Postgres untuk uji kompatibilitas.

## Menjalankan

```bash
make test        # go test ./...
make verify      # konvensi + vet + build + test  (gate "selesai")
go test ./tests/integration/ -run Setting -v   # subset
```

**Manual (Postman):** impor [`docs/postman/GoAdmin.postman_collection.json`](postman/GoAdmin.postman_collection.json) untuk menguji endpoint REST secara manual. Variabel `base_url` default `http://localhost:3000`.

## Lapisan test

| Lapisan | Lokasi | Cakupan |
|---|---|---|
| **Unit** | `tests/unit` | helper murni (mis. auth/JWT) |
| **Integration** | `tests/integration` | service ↔ DB (SQLite in-memory) per modul |
| **API / Security** | `tests/api` | alur HTTP (RBAC, blacklist JWT), CSRF & rate-limit middleware |
| **View** | `tests/integration/views_test.go` | semua template render (chrome + view modul) tanpa error |
| **BDD** | `tests/bdd` (godog) | skenario perilaku Gherkin atas app API nyata (RBAC) |

## Pola

- **Setup bersama** via `tests/testutil`:
  - `NewContainer(t, mode)` → container dengan SQLite in-memory (skema **access** sudah ter-migrate) + store in-memory (tanpa Redis).
  - `SeedAdmin(t, c)` → admin + permission inti + role Administrator.
- **Migrasi per modul**: `testutil` hanya migrasi skema `access`. Test modul lain memanggil migrasi modelnya sendiri, mis. `settingmig.AutoMigrate(c.DB)`.
- **Assert AppError**: `apperr.As(err)` lalu cek `ae.Status` (404/409/422/…).

```go
func TestXxxService_Conflict(t *testing.T) {
    c := testutil.NewContainer(t, config.ModeFull)
    xmig.AutoMigrate(c.DB)
    svc := service.NewXxxService(c.DB)
    _, err := svc.Store(ctx, dupInput, "")
    ae, ok := apperr.As(err)
    if !ok || ae.Status != 409 { t.Fatalf("harus 409, dapat %v", err) }
}
```

## Prinsip penting

- **Mock setia-perilaku (fidelity)** — saat memalsukan dependency eksternal (Redis/cache), tiru API & perilaku runtime PERSIS. Pelajaran NodeAdmin: blacklist JWT lolos test tapi gagal di produksi karena mock Redis "selalu mulus". Untuk jalur kritis (auth/blacklist) pakai store yang **berperilaku seperti runtime** (mis. `MemoryBlacklist` meniru TTL) + uji nyata: login → akses 200 → logout → akses **401**.
- **Test WAJIB tiap fitur** — modul ber-service yang terjangkau lewat route harus punya ≥1 test yang menyebut modul/service/model (di-cek convention checker, level warning).
- **Verifikasi nyata** — jangan klaim tanpa bukti; jalankan `make verify` sampai hijau sebelum menganggap selesai.

## BDD (godog)

Skenario perilaku ditulis Gherkin di `tests/bdd/features/*.feature`; step definitions di `tests/bdd/*_test.go` menjalankan app API nyata (`app.Build` + httptest + SQLite in-memory). Jalankan via `go test ./tests/bdd/` (ikut `make test`). Contoh: `rbac.feature` (tanpa token→401, admin→200, user tanpa izin→403).

## Tambah test untuk modul baru

`make module` sudah menghasilkan `tests/integration/<modul>_service_test.go` (Store/Show/Index + konflik + not-found). Tambah skenario sesuai logika domain. Untuk fitur web, tambah view ke `tests/integration/views_test.go`.
