package service

import (
	"context"

	"gorm.io/gorm"

	apperr "goadmin/internal/errors"
	model "goadmin/internal/modules/access/model"
)

// DashboardService mengimplementasi IDashboardService. DB di-inject lewat konstruktor.
type DashboardService struct {
	db *gorm.DB
}

// Pastikan kontrak terpenuhi saat compile.
var _ IDashboardService = (*DashboardService)(nil)

// NewDashboardService merakit service.
func NewDashboardService(db *gorm.DB) *DashboardService {
	return &DashboardService{db: db}
}

// Stats menghitung jumlah user, role, dan permission (read-only).
func (s *DashboardService) Stats(ctx context.Context) (Stats, error) {
	db := s.db.WithContext(ctx)

	var st Stats
	if err := db.Model(&model.User{}).Count(&st.Users).Error; err != nil {
		return Stats{}, apperr.Internal(err.Error())
	}
	if err := db.Model(&model.Role{}).Count(&st.Roles).Error; err != nil {
		return Stats{}, apperr.Internal(err.Error())
	}
	if err := db.Model(&model.Permission{}).Count(&st.Permissions).Error; err != nil {
		return Stats{}, apperr.Internal(err.Error())
	}
	return st, nil
}
