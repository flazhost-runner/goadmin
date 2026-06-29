package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/model"
)

// RoleService mengimplementasi IRoleService.
type RoleService struct {
	db *gorm.DB
}

var _ IRoleService = (*RoleService)(nil)

// NewRoleService merakit service.
func NewRoleService(db *gorm.DB) *RoleService {
	return &RoleService{db: db}
}

func (s *RoleService) Index(ctx context.Context, q dto.ListQuery) (helpers.Paginated[model.Role], error) {
	query := s.db.WithContext(ctx).Model(&model.Role{})
	if q.Search != "" {
		query = helpers.CiLike(query, "name", q.Search)
	}
	if q.QName != "" {
		query = helpers.CiLike(query, "name", q.QName)
	}
	if q.QStatus != "" {
		query = query.Where("status = ?", q.QStatus)
	}
	if q.QDesc != "" {
		query = helpers.CiLike(query, "desc", q.QDesc)
	}
	query = query.Order("name ASC").Preload("Permissions")

	var roles []model.Role
	meta, err := helpers.Paginate(query, q.Page, q.PerPage, &roles)
	if err != nil {
		return helpers.Paginated[model.Role]{}, apperr.Internal(err.Error())
	}
	return helpers.Paginated[model.Role]{Data: roles, Meta: meta}, nil
}

func (s *RoleService) Show(ctx context.Context, id string) (*model.Role, error) {
	var role model.Role
	if err := s.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("Role tidak ditemukan")
		}
		return nil, apperr.Internal(err.Error())
	}
	return &role, nil
}

func (s *RoleService) Store(ctx context.Context, in dto.CreateRoleInput) (*model.Role, error) {
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.Role{}).Where("name = ?", in.Name).Count(&count).Error; err != nil {
		return nil, apperr.Internal(err.Error())
	}
	if count > 0 {
		return nil, apperr.Conflict("Nama role sudah terpakai")
	}

	status := in.Status
	if status == "" {
		status = model.StatusActive
	}
	role := model.Role{
		ID:          helpers.NewID(),
		Name:        in.Name,
		Status:      status,
		Description: in.Description,
	}
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&role).Error; err != nil {
			return err
		}
		return syncRolePermissions(tx, &role, in.PermissionIDs)
	})
	if err != nil {
		return nil, apperr.Internal(err.Error())
	}
	return s.Show(ctx, role.ID)
}

func (s *RoleService) Update(ctx context.Context, id string, in dto.UpdateRoleInput) (*model.Role, error) {
	role, err := s.Show(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != role.Name {
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.Role{}).
			Where("name = ? AND id <> ?", in.Name, id).Count(&count).Error; err != nil {
			return nil, apperr.Internal(err.Error())
		}
		if count > 0 {
			return nil, apperr.Conflict("Nama role sudah terpakai")
		}
	}
	role.Name = in.Name
	if in.Status != "" {
		role.Status = in.Status
	}
	role.Description = in.Description

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(role).Error; err != nil {
			return err
		}
		return syncRolePermissions(tx, role, in.PermissionIDs)
	})
	if err != nil {
		return nil, apperr.Internal(err.Error())
	}
	return s.Show(ctx, id)
}

func (s *RoleService) Destroy(ctx context.Context, id string) error {
	role, err := s.Show(ctx, id)
	if err != nil {
		return err
	}
	if role.Name == model.RoleAdministrator {
		return apperr.Forbidden("Role Administrator tak boleh dihapus")
	}
	if err := s.db.WithContext(ctx).Select("Permissions", "Users").Delete(role).Error; err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

// DestroyMany menghapus banyak role sekaligus ("Delete Selected"); role
// Administrator dilewati (proteksi sama seperti Destroy tunggal).
func (s *RoleService) DestroyMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Select("Permissions", "Users").
		Where("name <> ?", model.RoleAdministrator).
		Delete(&model.Role{}, "id IN ?", ids).Error; err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

func syncRolePermissions(tx *gorm.DB, role *model.Role, permIDs []string) error {
	if permIDs == nil {
		return nil
	}
	var perms []model.Permission
	if len(permIDs) > 0 {
		if err := tx.Where("id IN ?", permIDs).Find(&perms).Error; err != nil {
			return err
		}
	}
	return tx.Model(role).Association("Permissions").Replace(perms)
}

// --- Kelola permission per-role (padanan RoleService.permission* NodeAdmin) ---

// PermissionList mengembalikan SELURUH permission (paginated + filter q_*) untuk
// halaman kelola permission sebuah role, beserta role-nya (Permissions ter-preload
// → penanda "assigned"). Filter status: Active = permission yang DIMILIKI role,
// Inactive = yang BELUM (subquery roles_permissions).
func (s *RoleService) PermissionList(ctx context.Context, roleID string, q dto.ListQuery) (helpers.Paginated[model.Permission], *model.Role, error) {
	var role model.Role
	if err := s.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", roleID).Error; err != nil {
		return helpers.Paginated[model.Permission]{}, nil, apperr.NotFound("Role tidak ditemukan")
	}
	query := s.db.WithContext(ctx).Model(&model.Permission{})
	if q.QName != "" {
		query = helpers.CiLike(query, "name", q.QName)
	}
	if q.QMethod != "" {
		query = query.Where("method = ?", q.QMethod)
	}
	if q.QDesc != "" {
		query = helpers.CiLike(query, "desc", q.QDesc)
	}
	assigned := s.db.Table("roles_permissions").Select("permission_id").Where("role_id = ?", roleID)
	switch q.QStatus {
	case "Active":
		query = query.Where("id IN (?)", assigned)
	case "Inactive":
		query = query.Where("id NOT IN (?)", assigned)
	}
	query = query.Order("name ASC")

	var perms []model.Permission
	meta, err := helpers.Paginate(query, q.Page, q.PerPage, &perms)
	if err != nil {
		return helpers.Paginated[model.Permission]{}, nil, apperr.Internal(err.Error())
	}
	return helpers.Paginated[model.Permission]{Data: perms, Meta: meta}, &role, nil
}

// AssignPermission menambah satu permission ke role (idempoten — abai bila sudah ada).
func (s *RoleService) AssignPermission(ctx context.Context, roleID, permID string) error {
	var role model.Role
	if err := s.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", roleID).Error; err != nil {
		return apperr.NotFound("Role tidak ditemukan")
	}
	for _, p := range role.Permissions {
		if p.ID == permID {
			return nil
		}
	}
	var perm model.Permission
	if err := s.db.WithContext(ctx).First(&perm, "id = ?", permID).Error; err != nil {
		return apperr.NotFound("Permission tidak ditemukan")
	}
	if err := s.db.WithContext(ctx).Model(&role).Association("Permissions").Append(&perm); err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

// AssignPermissions menambah banyak permission terpilih (tanpa duplikat).
func (s *RoleService) AssignPermissions(ctx context.Context, roleID string, permIDs []string) error {
	if len(permIDs) == 0 {
		return nil
	}
	var role model.Role
	if err := s.db.WithContext(ctx).Preload("Permissions").First(&role, "id = ?", roleID).Error; err != nil {
		return apperr.NotFound("Role tidak ditemukan")
	}
	existing := make(map[string]bool, len(role.Permissions))
	for _, p := range role.Permissions {
		existing[p.ID] = true
	}
	var found []model.Permission
	if err := s.db.WithContext(ctx).Where("id IN ?", permIDs).Find(&found).Error; err != nil {
		return apperr.Internal(err.Error())
	}
	var toAdd []model.Permission
	for _, p := range found {
		if !existing[p.ID] {
			toAdd = append(toAdd, p)
		}
	}
	if len(toAdd) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Model(&role).Association("Permissions").Append(&toAdd); err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

// UnassignPermission melepas satu permission dari role (abai bila tak ada).
func (s *RoleService) UnassignPermission(ctx context.Context, roleID, permID string) error {
	var role model.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		return apperr.NotFound("Role tidak ditemukan")
	}
	if err := s.db.WithContext(ctx).Model(&role).Association("Permissions").Delete(&model.Permission{ID: permID}); err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

// UnassignPermissions melepas banyak permission terpilih dari role.
func (s *RoleService) UnassignPermissions(ctx context.Context, roleID string, permIDs []string) error {
	if len(permIDs) == 0 {
		return nil
	}
	var role model.Role
	if err := s.db.WithContext(ctx).First(&role, "id = ?", roleID).Error; err != nil {
		return apperr.NotFound("Role tidak ditemukan")
	}
	perms := make([]model.Permission, 0, len(permIDs))
	for _, id := range permIDs {
		perms = append(perms, model.Permission{ID: id})
	}
	if err := s.db.WithContext(ctx).Model(&role).Association("Permissions").Delete(&perms); err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}
