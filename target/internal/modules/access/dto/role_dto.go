package dto

// CreateRoleInput = payload buat role.
type CreateRoleInput struct {
	Name          string   `json:"name" form:"name" binding:"required,max=50"`
	Status        string   `json:"status" form:"status" binding:"omitempty,oneof=Active Inactive"`
	Description   string   `json:"desc" form:"desc" binding:"omitempty,max=255"`
	PermissionIDs []string `json:"permission_ids" form:"permission_ids" binding:"omitempty,dive,max=36"`
}

// UpdateRoleInput = payload ubah role.
type UpdateRoleInput struct {
	Name          string   `json:"name" form:"name" binding:"required,max=50"`
	Status        string   `json:"status" form:"status" binding:"omitempty,oneof=Active Inactive"`
	Description   string   `json:"desc" form:"desc" binding:"omitempty,max=255"`
	PermissionIDs []string `json:"permission_ids" form:"permission_ids" binding:"omitempty,dive,max=36"`
}

// CreatePermissionInput = payload buat permission.
type CreatePermissionInput struct {
	Name        string `json:"name" form:"name" binding:"required,max=100"`
	GuardName   string `json:"guard_name" form:"guard_name" binding:"omitempty,oneof=web api"`
	Method      string `json:"method" form:"method" binding:"omitempty,oneof=GET POST PATCH PUT DELETE"`
	Status      string `json:"status" form:"status" binding:"omitempty,oneof=Active Inactive"`
	Description string `json:"description" form:"description" binding:"omitempty,max=255"`
}

// UpdatePermissionInput = payload ubah permission.
type UpdatePermissionInput struct {
	Name        string `json:"name" form:"name" binding:"required,max=100"`
	GuardName   string `json:"guard_name" form:"guard_name" binding:"omitempty,oneof=web api"`
	Method      string `json:"method" form:"method" binding:"omitempty,oneof=GET POST PATCH PUT DELETE"`
	Status      string `json:"status" form:"status" binding:"omitempty,oneof=Active Inactive"`
	Description string `json:"description" form:"description" binding:"omitempty,max=255"`
}
