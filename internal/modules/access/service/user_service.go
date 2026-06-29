package service

import (
	"context"
	"errors"

	"gorm.io/gorm"

	"goadmin/internal/auth"
	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/model"
)

// UserService mengimplementasi IUserService. Dependency (DB, bcrypt cost)
// di-inject lewat konstruktor — tak ada state global.
type UserService struct {
	db           *gorm.DB
	bcryptRounds int
}

// Pastikan kontrak terpenuhi saat compile.
var _ IUserService = (*UserService)(nil)

// NewUserService merakit service (dipanggil di wiring container modul).
func NewUserService(db *gorm.DB, bcryptRounds int) *UserService {
	return &UserService{db: db, bcryptRounds: bcryptRounds}
}

func (s *UserService) Index(ctx context.Context, q dto.ListQuery) (helpers.Paginated[model.User], error) {
	query := s.db.WithContext(ctx).Model(&model.User{})
	if q.Search != "" {
		query = helpers.CiLikeAny(query, []string{"name", "email", "code"}, q.Search)
	}
	// Filter per-kolom (replika tabel index NodeAdmin).
	if q.QCode != "" {
		query = helpers.CiLike(query, "code", q.QCode)
	}
	if q.QName != "" {
		query = helpers.CiLike(query, "name", q.QName)
	}
	if q.QPhone != "" {
		query = helpers.CiLike(query, "phone", q.QPhone)
	}
	if q.QEmail != "" {
		query = helpers.CiLike(query, "email", q.QEmail)
	}
	if q.QStatus != "" {
		query = query.Where("status = ?", q.QStatus)
	}
	if q.QRole != "" {
		query = query.Where("id IN (SELECT user_id FROM users_roles WHERE role_id = ?)", q.QRole)
	}
	query = query.Order("created_at DESC").Preload("Roles")

	var users []model.User
	meta, err := helpers.Paginate(query, q.Page, q.PerPage, &users)
	if err != nil {
		return helpers.Paginated[model.User]{}, apperr.Internal(err.Error())
	}
	return helpers.Paginated[model.User]{Data: users, Meta: meta}, nil
}

func (s *UserService) Show(ctx context.Context, id string) (*model.User, error) {
	var user model.User
	if err := s.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apperr.NotFound("User tidak ditemukan")
		}
		return nil, apperr.Internal(err.Error())
	}
	return &user, nil
}

func (s *UserService) Store(ctx context.Context, in dto.CreateUserInput, actorID string) (*model.User, error) {
	// Cegah email duplikat (pesan domain → 409).
	var count int64
	if err := s.db.WithContext(ctx).Model(&model.User{}).Where("email = ?", in.Email).Count(&count).Error; err != nil {
		return nil, apperr.Internal(err.Error())
	}
	if count > 0 {
		return nil, apperr.Conflict("Email sudah terpakai")
	}

	hash, err := auth.HashPassword(in.Password, s.bcryptRounds)
	if err != nil {
		return nil, apperr.Internal("gagal hash password: " + err.Error())
	}

	status := in.Status
	if status == "" {
		status = model.StatusActive
	}
	tz := in.Timezone
	if tz == "" {
		tz = "UTC"
	}

	user := model.User{
		ID:            helpers.NewID(),
		Code:          helpers.NewCode("U"),
		Name:          in.Name,
		Email:         in.Email,
		Phone:         in.Phone,
		Password:      hash,
		Status:        status,
		Timezone:      tz,
		Picture:       in.Picture,
		Blocked:       in.Blocked,
		BlockedReason: in.BlockedReason,
		CreatedBy:     actorID,
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		return syncUserRoles(tx, &user, in.RoleIDs)
	})
	if err != nil {
		return nil, apperr.Internal(err.Error())
	}
	return s.Show(ctx, user.ID)
}

func (s *UserService) Update(ctx context.Context, id string, in dto.UpdateUserInput, actorID string) (*model.User, error) {
	user, err := s.Show(ctx, id)
	if err != nil {
		return nil, err
	}

	// Email unik (selain milik sendiri).
	if in.Email != user.Email {
		var count int64
		if err := s.db.WithContext(ctx).Model(&model.User{}).
			Where("email = ? AND id <> ?", in.Email, id).Count(&count).Error; err != nil {
			return nil, apperr.Internal(err.Error())
		}
		if count > 0 {
			return nil, apperr.Conflict("Email sudah terpakai")
		}
	}

	user.Name = in.Name
	user.Email = in.Email
	user.Phone = in.Phone
	if in.Status != "" {
		user.Status = in.Status
	}
	if in.Timezone != "" {
		user.Timezone = in.Timezone
	}
	if in.Picture != "" {
		user.Picture = in.Picture
	}
	// Blocked = checkbox: set langsung (centang→true, kosong→false).
	user.Blocked = in.Blocked
	user.BlockedReason = in.BlockedReason
	user.UpdatedBy = actorID

	// Password hanya diubah bila diisi.
	if in.Password != "" {
		hash, herr := auth.HashPassword(in.Password, s.bcryptRounds)
		if herr != nil {
			return nil, apperr.Internal("gagal hash password: " + herr.Error())
		}
		user.Password = hash
	}

	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Save(user).Error; err != nil {
			return err
		}
		return syncUserRoles(tx, user, in.RoleIDs)
	})
	if err != nil {
		return nil, apperr.Internal(err.Error())
	}
	return s.Show(ctx, id)
}

func (s *UserService) Destroy(ctx context.Context, id string) error {
	user, err := s.Show(ctx, id)
	if err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Select("Roles").Delete(user).Error; err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

// DestroyMany menghapus banyak user sekaligus (aksi "Delete Selected" tabel).
// Relasi role ikut dibersihkan via Select("Roles").
func (s *UserService) DestroyMany(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.db.WithContext(ctx).Select("Roles").Delete(&model.User{}, "id IN ?", ids).Error; err != nil {
		return apperr.Internal(err.Error())
	}
	return nil
}

// syncUserRoles mengganti relasi role user dengan daftar id baru (bila disuplai).
func syncUserRoles(tx *gorm.DB, user *model.User, roleIDs []string) error {
	if roleIDs == nil {
		return nil
	}
	var roles []model.Role
	if len(roleIDs) > 0 {
		if err := tx.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
			return err
		}
	}
	return tx.Model(user).Association("Roles").Replace(roles)
}
