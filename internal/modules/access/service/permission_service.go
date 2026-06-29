package service

import (
	"context"
	"errors"
	"strings"

	"gorm.io/gorm"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/model"
	"goadmin/internal/router"
)

// PermissionService mengimplementasi IPermissionService.
type PermissionService struct {
	db *gorm.DB
}

var _ IPermissionService = (*PermissionService)(nil)

// NewPermissionService merakit service.
func NewPermissionService(db *gorm.DB) *PermissionService {
	return &PermissionService{db: db}
}

// SyncFromRoutes menurunkan permission dari SELURUH named route terdaftar
// (router.Entries) — padanan NodeAdmin getAllRegisteredRoute. Idempoten:
// buat record {name, method, guard} yang belum ada. guard diturunkan dari prefix
// nama route (`api.` → api, lainnya → web), sejajar NodeAdmin
// (`name.startsWith('api.') ? 'api' : 'web'`).
func (s *PermissionService) SyncFromRoutes(ctx context.Context) error {
	for _, e := range router.Entries() {
		if e.Name == "" || e.Method == "" {
			continue // hanya route bernama + bermethod yang dipersistensi
		}
		guard := "web"
		if strings.HasPrefix(e.Name, "api.") {
			guard = "api"
		}
		var p model.Permission
		err := s.db.WithContext(ctx).
			Where("name = ? AND method = ? AND guard_name = ?", e.Name, e.Method, guard).
			First(&p).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			p = model.Permission{
				ID:        helpers.NewID(),
				Name:      e.Name,
				Method:    e.Method,
				GuardName: guard,
				Status:    model.StatusActive,
			}
			if err := s.db.WithContext(ctx).Create(&p).Error; err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}
	return nil
}

func (s *PermissionService) Index(ctx context.Context, q dto.ListQuery) (helpers.Paginated[model.Permission], error) {
	query := s.db.WithContext(ctx).Model(&model.Permission{})
	if q.Search != "" {
		query = helpers.CiLike(query, "name", q.Search)
	}
	if q.QName != "" {
		query = helpers.CiLike(query, "name", q.QName)
	}
	if q.QMethod != "" {
		query = query.Where("method = ?", q.QMethod)
	}
	if q.QStatus != "" {
		query = query.Where("status = ?", q.QStatus)
	}
	if q.QGuard != "" {
		query = query.Where("guard_name = ?", q.QGuard)
	}
	if q.QDesc != "" {
		query = helpers.CiLike(query, "desc", q.QDesc)
	}
	query = query.Order("name ASC")

	var perms []model.Permission
	meta, err := helpers.Paginate(query, q.Page, q.PerPage, &perms)
	if err != nil {
		return helpers.Paginated[model.Permission]{}, apperr.Internal(err.Error())
	}
	return helpers.Paginated[model.Permission]{Data: perms, Meta: meta}, nil
}

func (s *PermissionService) Show(ctx context.Context, id string) (*model.Permission, error) {
	var perm model.Permission
	if err := s.db.WithContext(ctx).First(&perm, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("Permission tidak ditemukan")
		}
		return nil, apperr.Internal(err.Error())
	}
	return &perm, nil
}

func (s *PermissionService) Store(ctx context.Context, in dto.CreatePermissionInput) (*model.Permission, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Permission{}).Where("name = ?", in.Name).Count(&count).Error; err != nil {
		return nil, apperr.Internal(err.Error())
	}
	if count > 0 {
		return nil, apperr.Conflict("Nama permission sudah terpakai")
	}
	status := in.Status
	if status == "" {
		status = model.StatusActive
	}
	guard := in.GuardName
	if guard == "" {
		guard = "web"
	}
	perm := model.Permission{
		ID:          helpers.NewID(),
		Name:        in.Name,
		GuardName:   guard,
		Method:      in.Method,
		Status:      status,
		Description: in.Description,
	}
	if err := s.db.WithContext(ctx).Create(&perm).Error; err != nil {
		return nil, apperr.Internal(err.Error())
	}
	return &perm, nil
}

func (s *PermissionService) Update(ctx context.Context, id string, in dto.UpdatePermissionInput) (*model.Permission, error) {
	perm, err := s.Show(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != perm.Name {
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.Permission{}).
			Where("name = ? AND id <> ?", in.Name, id).Count(&count).Error; err != nil {
			return nil, apperr.Internal(err.Error())
		}
		if count > 0 {
			return nil, apperr.Conflict("Nama permission sudah terpakai")
		}
	}
	perm.Name = in.Name
	if in.GuardName != "" {
		perm.GuardName = in.GuardName
	}
	perm.Method = in.Method
	if in.Status != "" {
		perm.Status = in.Status
	}
	perm.Description = in.Description
	if err := s.db.WithContext(ctx).Save(perm).Error; err != nil {
		return nil, apperr.Internal(err.Error())
	}
	return perm, nil
}

func (s *PermissionService) Destroy(ctx context.Context, id string) error {
	perm, err := s.Show(ctx, id)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Select("Roles").Delete(perm).Error; err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

// DestroyMany menghapus banyak permission sekaligus ("Delete Selected").
func (s *PermissionService) DestroyMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Select("Roles").Delete(&model.Permission{}, "id IN ?", ids).Error; err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}
