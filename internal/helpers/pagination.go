package helpers

import (
	"math"

	"gorm.io/gorm"
)

// PageMeta adalah metadata paginasi yang dikirim ke view/JSON (format NodeAdmin).
type PageMeta struct {
	Total       int64 `json:"total_data"`
	PerPage     int   `json:"page_size"`
	CurrentPage int   `json:"current_page"`
	LastPage    int   `json:"total_page"`
	From        int   `json:"from"`
	To          int   `json:"to"`
}

// Paginated membungkus hasil list + meta (padanan paginate() NodeAdmin).
// Bentuk JSON: { "datas": [...], "paginate_data": { ... } }.
type Paginated[T any] struct {
	Data []T      `json:"datas"`
	Meta PageMeta `json:"paginate_data"`
}

const (
	defaultPerPage = 10
	maxPerPage     = 100
)

// NormalizePage membatasi page/perPage ke rentang aman (hindari ambil seluruh tabel).
func NormalizePage(page, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = defaultPerPage
	}
	if perPage > maxPerPage {
		perPage = maxPerPage
	}
	return page, perPage
}

// Paginate menjalankan COUNT + LIMIT/OFFSET pada query GORM yang sudah difilter,
// lalu mengisi out (pointer ke slice) dan mengembalikan meta.
//
// query harus sudah memuat Model + Where (tanpa Limit/Offset). Fungsi ini meng-clone
// session untuk count agar tidak mengganggu query utama.
func Paginate[T any](query *gorm.DB, page, perPage int, out *[]T) (PageMeta, error) {
	page, perPage = NormalizePage(page, perPage)

	var total int64
	if err := query.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return PageMeta{}, err
	}

	offset := (page - 1) * perPage
	if err := query.Limit(perPage).Offset(offset).Find(out).Error; err != nil {
		return PageMeta{}, err
	}

	lastPage := int(math.Ceil(float64(total) / float64(perPage)))
	if lastPage < 1 {
		lastPage = 1
	}
	from := 0
	to := 0
	if total > 0 {
		from = offset + 1
		to = offset + len(*out)
	}

	return PageMeta{
		Total:       total,
		PerPage:     perPage,
		CurrentPage: page,
		LastPage:    lastPage,
		From:        from,
		To:          to,
	}, nil
}
