package api

import (
	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/service"
)

// RoleController = CRUD role (path verbose persis NodeAdmin, bukan REST).
type RoleController struct {
	roles service.IRoleService
}

// NewRoleController merakit controller.
func NewRoleController(roles service.IRoleService) *RoleController {
	return &RoleController{roles: roles}
}

// Index → GET /api/v1/access/role.
func (ctl *RoleController) Index(c *gin.Context) {
	var q dto.ListQuery
	_ = c.ShouldBindQuery(&q)
	res, err := ctl.roles.Index(c.Request.Context(), q)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "OK", res)
}

// Edit → GET /api/v1/access/role/:id/edit (kembar web; kembalikan entity utk edit).
func (ctl *RoleController) Edit(c *gin.Context) {
	role, err := ctl.roles.Show(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "OK", role)
}

// Store → POST /api/v1/access/role/store.
func (ctl *RoleController) Store(c *gin.Context) {
	var in dto.CreateRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	role, err := ctl.roles.Store(c.Request.Context(), in)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.Created(c, "Role dibuat", role)
}

// Update → PUT /api/v1/access/role/:id/update.
func (ctl *RoleController) Update(c *gin.Context) {
	var in dto.UpdateRoleInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	role, err := ctl.roles.Update(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Role diperbarui", role)
}

// Destroy → DELETE /api/v1/access/role/:id/delete.
func (ctl *RoleController) Destroy(c *gin.Context) {
	if err := ctl.roles.Destroy(c.Request.Context(), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Role dihapus", nil)
}

// DeleteSelected → POST /api/v1/access/role/delete_selected (body `{ selected: [id,...] }`).
func (ctl *RoleController) DeleteSelected(c *gin.Context) {
	var in struct {
		Selected []string `json:"selected"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	if err := ctl.roles.DestroyMany(c.Request.Context(), in.Selected); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Role terpilih dihapus", nil)
}

// --- Kelola permission per-role (kembar web; padanan NodeAdmin api role.permission*) ---

// Permission → GET /api/v1/access/role/:id/permission (daftar permission + status assigned).
func (ctl *RoleController) Permission(c *gin.Context) {
	var q dto.ListQuery
	_ = c.ShouldBindQuery(&q)
	res, role, err := ctl.roles.PermissionList(c.Request.Context(), c.Param("id"), q)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "OK", gin.H{"role": role, "permissions": res.Data, "meta": res.Meta})
}

// PermissionAssign → GET /api/v1/access/role/:id/permission/:permission_id/assign.
func (ctl *RoleController) PermissionAssign(c *gin.Context) {
	if err := ctl.roles.AssignPermission(c.Request.Context(), c.Param("id"), c.Param("permission_id")); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Permission di-assign", nil)
}

// PermissionAssignSelected → POST /api/v1/access/role/:id/permission/assign_selected (body `{selected:[...]}`).
func (ctl *RoleController) PermissionAssignSelected(c *gin.Context) {
	var in struct {
		Selected []string `json:"selected"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	if err := ctl.roles.AssignPermissions(c.Request.Context(), c.Param("id"), in.Selected); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Permission terpilih di-assign", nil)
}

// PermissionUnassign → GET /api/v1/access/role/:id/permission/:permission_id/unassign.
func (ctl *RoleController) PermissionUnassign(c *gin.Context) {
	if err := ctl.roles.UnassignPermission(c.Request.Context(), c.Param("id"), c.Param("permission_id")); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Permission di-unassign", nil)
}

// PermissionUnassignSelected → POST /api/v1/access/role/:id/permission/unassign_selected.
func (ctl *RoleController) PermissionUnassignSelected(c *gin.Context) {
	var in struct {
		Selected []string `json:"selected"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	if err := ctl.roles.UnassignPermissions(c.Request.Context(), c.Param("id"), in.Selected); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Permission terpilih di-unassign", nil)
}
