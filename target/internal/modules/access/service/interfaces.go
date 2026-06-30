// Package service berisi logika bisnis modul access. Tiap service:
//   - mengimplementasi interface I*Service (Dependency Inversion),
//   - menerima dependency lewat konstruktor (DI manual, bukan global),
//   - mengembalikan *errors.AppError saat gagal (dipetakan middleware ke HTTP),
//     BUKAN error telanjang yang di-handle status-nya di controller.
package service

import (
	"context"

	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/model"
)

// IUserService = kontrak manajemen user.
type IUserService interface {
	Index(ctx context.Context, q dto.ListQuery) (helpers.Paginated[model.User], error)
	Show(ctx context.Context, id string) (*model.User, error)
	Store(ctx context.Context, in dto.CreateUserInput, actorID string) (*model.User, error)
	Update(ctx context.Context, id string, in dto.UpdateUserInput, actorID string) (*model.User, error)
	Destroy(ctx context.Context, id string) error
	DestroyMany(ctx context.Context, ids []string) error
}

// IRoleService = kontrak manajemen role + permission assignment.
type IRoleService interface {
	Index(ctx context.Context, q dto.ListQuery) (helpers.Paginated[model.Role], error)
	Show(ctx context.Context, id string) (*model.Role, error)
	Store(ctx context.Context, in dto.CreateRoleInput) (*model.Role, error)
	Update(ctx context.Context, id string, in dto.UpdateRoleInput) (*model.Role, error)
	Destroy(ctx context.Context, id string) error
	DestroyMany(ctx context.Context, ids []string) error
	// Kelola permission per-role (padanan RoleService.permission* NodeAdmin).
	PermissionList(ctx context.Context, roleID string, q dto.ListQuery) (helpers.Paginated[model.Permission], *model.Role, error)
	AssignPermission(ctx context.Context, roleID, permID string) error
	AssignPermissions(ctx context.Context, roleID string, permIDs []string) error
	UnassignPermission(ctx context.Context, roleID, permID string) error
	UnassignPermissions(ctx context.Context, roleID string, permIDs []string) error
}

// IPasswordResetService = kontrak reset password lewat OTP email.
type IPasswordResetService interface {
	// RequestReset mengirim OTP ke email bila terdaftar (tak membocorkan
	// keberadaan email — selalu "sukses" dari sisi pemanggil web).
	RequestReset(ctx context.Context, email string) error
	// Reset memverifikasi OTP (hash + belum kedaluwarsa) lalu menyetel password baru.
	Reset(ctx context.Context, email, otp, newPassword string) error
}

// IPermissionService = kontrak manajemen permission.
type IPermissionService interface {
	Index(ctx context.Context, q dto.ListQuery) (helpers.Paginated[model.Permission], error)
	Show(ctx context.Context, id string) (*model.Permission, error)
	Store(ctx context.Context, in dto.CreatePermissionInput) (*model.Permission, error)
	Update(ctx context.Context, id string, in dto.UpdatePermissionInput) (*model.Permission, error)
	Destroy(ctx context.Context, id string) error
	DestroyMany(ctx context.Context, ids []string) error
	// SyncFromRoutes menurunkan permission dari named-route registry (route-driven,
	// padanan NodeAdmin getAllRegisteredRoute). Idempoten.
	SyncFromRoutes(ctx context.Context) error
}
