# Panduan Membuat Modul Baru (GoAdmin)

Panduan langkah-demi-langkah membuat modul yang **otomatis lolos** convention checker (`make check`) dan konsisten dengan pola modul referensi **`access`**. Baca [`../AGENTS.md`](../AGENTS.md) lebih dulu sebagai sumber kebenaran.

Contoh di bawah memakai modul fiktif **`product`** (`internal/modules/product/`).

> ## ⚡ Cara cepat: generator
>
> Tak perlu menyalin manual — pakai generator yang menghasilkan SEMUA artefak di bawah (model→test) + mendaftarkan blank-import:
> ```
> make module ARGS="--name product"             # full (api + web)
> make module ARGS="--name token --web=false"   # api-only
> make module ARGS="--name category --plural categories"
> # atau langsung: go run ./cmd/make-module --name product
> ```
> Lalu `make verify` (harus hijau). Panduan manual di bawah berguna untuk **memahami** tiap lapisan atau menambah field/relasi setelah scaffold.

```
internal/modules/product/
  model/product.go
  migration/automigrate.go
  dto/product_dto.go
  service/interfaces.go
  service/product_service.go
  controller/api/product_controller.go
  controller/web/product_controller.go        # bila ada UI
  route_api.go
  route_web.go                                 # bila ada UI
  module.go
```

> **Test wajib** (mis. `tests/integration/product_service_test.go`) — checker memperingatkan modul ber-service tanpa test.

---

## 1. Model — tipe portabel

`model/product.go`. Tag gorm **abstrak**; waktu pakai `time.Time` (tanpa `type:`); status pakai `varchar` + konstanta (bukan ENUM).

```go
package model

import "time"

const (
	ProductActive   = "Active"
	ProductInactive = "Inactive"
)

type Product struct {
	ID        string    `gorm:"type:varchar(36);primaryKey" json:"id"`
	Code      string    `gorm:"type:varchar(20);uniqueIndex" json:"code"`
	Name      string    `gorm:"type:varchar(100);index" json:"name"`
	Price     int64     `gorm:"type:bigint" json:"price"`
	Status    string    `gorm:"type:varchar(20);default:Active;index" json:"status"`
	CreatedBy string    `gorm:"type:varchar(36)" json:"created_by"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Product) TableName() string { return "products" }
```

❌ JANGAN: `type:longtext`, `type:datetime`, `type:enum(...)`, `collation:...` → ditolak checker.

## 2. Migration

`migration/automigrate.go` — daftarkan model ke AutoMigrate (dev/test; folder `migration/` dikecualikan checker). Ikuti pola `access/migration/`.

> **Produksi**: tambahkan migrasi **versioned + reversible** (golang-migrate) untuk perubahan skema — `make migration name=create_products` membuat `internal/migrate/migrations/NNNNNN_create_products.{up,down}.sql`; isi SQL portabel (VARCHAR/TEXT/BIGINT/BOOLEAN/TIMESTAMP, tanpa `IF NOT EXISTS` pada index). AutoMigrate (dari model) = jalur cepat dev; SQL versioned = jalur produksi (riwayat + rollback). Jaga keduanya sinkron.

```go
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&model.Product{})
}
```

## 3. DTO — input ter-whitelist (anti mass-assignment)

`dto/product_dto.go`. Hanya field di DTO yang diterima. Validasi via tag `binding` (go-playground/validator). Sertakan `ListQuery` bila list.

```go
package dto

type CreateProductInput struct {
	Name   string `json:"name" form:"name" binding:"required,max=100"`
	Price  int64  `json:"price" form:"price" binding:"required,min=0"`
	Status string `json:"status" form:"status" binding:"omitempty,oneof=Active Inactive"`
}

type UpdateProductInput struct {
	Name   string `json:"name" form:"name" binding:"required,max=100"`
	Price  int64  `json:"price" form:"price" binding:"required,min=0"`
	Status string `json:"status" form:"status" binding:"omitempty,oneof=Active Inactive"`
}

type ListQuery struct {
	Page    int    `form:"page"`
	PerPage int    `form:"per_page"`
	Search  string `form:"search"`
}
```

## 4. Interface service

`service/interfaces.go` — kontrak `I*Service` (Dependency Inversion). Pakai `context.Context`, `helpers.Paginated[T]`.

```go
type IProductService interface {
	Index(ctx context.Context, q dto.ListQuery) (helpers.Paginated[model.Product], error)
	Show(ctx context.Context, id string) (*model.Product, error)
	Store(ctx context.Context, in dto.CreateProductInput, actorID string) (*model.Product, error)
	Update(ctx context.Context, id string, in dto.UpdateProductInput, actorID string) (*model.Product, error)
	Destroy(ctx context.Context, id string) error
}
```

## 5. Service — assertion compile, return *AppError, pakai helper

`service/product_service.go`. WAJIB: assertion `var _ I...`; dependency lewat konstruktor; gagal → `apperr.*` (BUKAN `errors.New`/`fmt.Errorf`); search lewat `helpers.CiLikeAny`; list lewat `helpers.Paginate`.

```go
package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/product/dto"
	"goadmin/internal/modules/product/model"
)

type ProductService struct{ db *gorm.DB }

// Wajib — menjamin kontrak terpenuhi saat compile.
var _ IProductService = (*ProductService)(nil)

func NewProductService(db *gorm.DB) *ProductService { return &ProductService{db: db} }

func (s *ProductService) Index(ctx context.Context, q dto.ListQuery) (helpers.Paginated[model.Product], error) {
	query := s.db.WithContext(ctx).Model(&model.Product{})
	if q.Search != "" {
		query = helpers.CiLikeAny(query, []string{"name", "code"}, q.Search) // ❌ jangan LIKE manual
	}
	query = query.Order("created_at DESC")

	var rows []model.Product
	meta, err := helpers.Paginate(query, q.Page, q.PerPage, &rows)
	if err != nil {
		return helpers.Paginated[model.Product]{}, apperr.Internal(err.Error())
	}
	return helpers.Paginated[model.Product]{Data: rows, Meta: meta}, nil
}

func (s *ProductService) Show(ctx context.Context, id string) (*model.Product, error) {
	var p model.Product
	if err := s.db.WithContext(ctx).First(&p, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) { // errors.Is BOLEH
			return nil, apperr.NotFound("Product tidak ditemukan")
		}
		return nil, apperr.Internal(err.Error())
	}
	return &p, nil
}

// Store/Update/Destroy: ikuti pola access/service/user_service.go.
```

## 6. Controller API — inject interface, helpers.OK/Created

`controller/api/product_controller.go`. Terima `service.IProductService` lewat konstruktor (DI). Tanpa logika bisnis, tanpa map status manual — `c.Error(err)`.

```go
type ProductController struct{ products service.IProductService }

func NewProductController(p service.IProductService) *ProductController {
	return &ProductController{products: p}
}

func (ctl *ProductController) Index(c *gin.Context) {
	var q dto.ListQuery
	_ = c.ShouldBindQuery(&q)
	res, err := ctl.products.Index(c.Request.Context(), q)
	if err != nil { c.Error(err); return }
	helpers.OK(c, "OK", res)
}

func (ctl *ProductController) Store(c *gin.Context) {
	var in dto.CreateProductInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil)); return
	}
	p, err := ctl.products.Store(c.Request.Context(), in, "")
	if err != nil { c.Error(err); return }
	helpers.Created(c, "Product dibuat", p)
}
```

❌ JANGAN `service.NewProductService(...)` atau `service.ProductService{}` di controller → ditolak checker (DI only).

## 7. Controller Web (bila ada UI) — view.RenderView

`controller/web/product_controller.go`. WAJIB `view.RenderView` — **bukan** `c.HTML(...)`.

```go
func (ctl *ProductController) Index(c *gin.Context) {
	var q dto.ListQuery
	_ = c.ShouldBindQuery(&q)
	res, err := ctl.products.Index(c.Request.Context(), q)
	if err != nil { c.Error(err); return }
	view.RenderView(c, "product/index", gin.H{
		"title": "Manajemen Product", "products": res.Data, "meta": res.Meta,
	})
}
```

## 8. Route — named routes + urutan middleware

`route_api.go` (selalu) & `route_web.go` (bila UI). Daftarkan nama route lewat `router.Register`; urutan guard **auth → authorize**. Ikuti `access/route_api.go` (`resource(...)`) & `access/route_web.go`.

```go
// route_api.go
func registerAPIRoutes(ctx *router.RegistrationContext, guard *accessmw.Guard, ctl *apictl.ProductController) {
	g := ctx.API.Group("/v1/products", guard.AuthenticatedJWT())
	g.GET("", accessmw.Authorize("product.view"), ctl.Index)
	router.Register("api.v1.products.index", "/api/v1/products")
	// show/store/update/destroy → ikuti pola resource() di access.
}
```

## 9. Module — init() + Register (composition root)

`module.go`. `init()` mendaftarkan modul; `Register` merakit service + controller (DI manual terpusat) lalu memasang route. Inilah **satu-satunya** tempat service di-`New`.

```go
type Module struct{}

func init() { router.Add(&Module{}) }

func (m *Module) Name() string { return "product" }

func (m *Module) Register(ctx *router.RegistrationContext) {
	c := ctx.Container
	svc := service.NewProductService(c.DB)         // wiring DI di sini (boleh)
	c.Provide("product.IProductService", svc)

	registerAPIRoutes(ctx, /* guard */, apictl.NewProductController(svc))

	if ctx.Mode == config.ModeFull && ctx.Web != nil {
		registerWebRoutes(ctx)                     // route web hanya mode full
	}
}
```

Lalu impor modul agar `init()`-nya jalan — tambahkan blank import di `internal/modules/modules.go` (ikuti pola `access`).

## 10. Test (wajib)

`tests/integration/product_service_test.go` — service ↔ SQLite in-memory (lihat `tests/testutil`). Minimal: Store → Show → Index (search) → Update → Destroy. Tambah `tests/api/` bila ada route API. Nama file menyebut `product` agar terdeteksi checker.

## 11. Verifikasi akhir

```
make check     # konvensi (WAJIB lolos)
make verify    # check + vet + build + test
```

Selesai bila keempatnya hijau + docs (README/`docs/API.md`) diperbarui.
