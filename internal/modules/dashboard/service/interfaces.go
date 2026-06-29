// Package service berisi logika bisnis modul dashboard (statistik ringkas,
// read-only). Mengembalikan *errors.AppError saat gagal & mengimplementasi
// IDashboardService (Dependency Inversion).
package service

import "context"

// Stats = ringkasan jumlah entity inti untuk kartu statistik dashboard.
type Stats struct {
	Users       int64 `json:"users"`
	Roles       int64 `json:"roles"`
	Permissions int64 `json:"permissions"`
}

// IDashboardService = kontrak statistik dashboard.
type IDashboardService interface {
	Stats(ctx context.Context) (Stats, error)
}
