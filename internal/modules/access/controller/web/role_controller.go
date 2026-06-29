package web

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/model"
	"goadmin/internal/modules/access/service"
	"goadmin/internal/view"
)

// RoleController menyajikan CRUD role (web). Memakai PermissionService untuk
// menyediakan daftar permission pada form (checkbox).
type RoleController struct {
	roles service.IRoleService
	perms service.IPermissionService
}

// NewRoleController merakit controller (service di-inject).
func NewRoleController(roles service.IRoleService, perms service.IPermissionService) *RoleController {
	return &RoleController{roles: roles, perms: perms}
}

// Index → GET /admin/v1/access/role.
func (ctl *RoleController) Index(c *gin.Context) {
	q, qbase := bindListQuery(c)
	res, err := ctl.roles.Index(c.Request.Context(), q)
	if err != nil {
		c.Error(err)
		return
	}
	view.RenderView(c, "roles/index", gin.H{
		"title": "Manajemen Role", "active": "role",
		"roles": res.Data, "meta": res.Meta,
		"filter": filterMap(q), "qbase": qbase,
	})
}

// DeleteSelected → POST /admin/v1/access/role/delete_selected (bulk delete tabel).
func (ctl *RoleController) DeleteSelected(c *gin.Context) {
	if err := ctl.roles.DestroyMany(c.Request.Context(), selectedIDs(c)); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Delete Role Success.")
	}
	c.Redirect(http.StatusFound, "/admin/v1/access/role")
}

// Create → GET /admin/v1/access/role/create.
func (ctl *RoleController) Create(c *gin.Context) {
	perms, err := ctl.allPermissions(c)
	if err != nil {
		c.Error(err)
		return
	}
	view.RenderView(c, "roles/form", gin.H{
		"title": "Tambah Role", "active": "role",
		"action": "/admin/v1/access/role/store", "role": nil,
		"permissions": perms, "selected": map[string]bool{},
	})
}

// Store → POST /admin/v1/access/role.
func (ctl *RoleController) Store(c *gin.Context) {
	var in dto.CreateRoleInput
	_ = c.ShouldBind(&in)
	if _, err := ctl.roles.Store(c.Request.Context(), in); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
		c.Redirect(http.StatusFound, "/admin/v1/access/role/create")
		return
	}
	setFlashSuccess(sessions.Default(c), "Create Role Success.")
	c.Redirect(http.StatusFound, "/admin/v1/access/role")
}

// Edit → GET /admin/v1/access/role/:id/edit.
func (ctl *RoleController) Edit(c *gin.Context) {
	role, err := ctl.roles.Show(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	perms, err := ctl.allPermissions(c)
	if err != nil {
		c.Error(err)
		return
	}
	selected := make(map[string]bool, len(role.Permissions))
	for _, p := range role.Permissions {
		selected[p.ID] = true
	}
	view.RenderView(c, "roles/form", gin.H{
		"title": "Ubah Role", "active": "role",
		"action": "/admin/v1/access/role/" + role.ID + "/update?_method=PUT", "role": role,
		"permissions": perms, "selected": selected,
	})
}

// Update → POST /admin/v1/access/role/:id.
func (ctl *RoleController) Update(c *gin.Context) {
	id := c.Param("id")
	var in dto.UpdateRoleInput
	_ = c.ShouldBind(&in)
	if _, err := ctl.roles.Update(c.Request.Context(), id, in); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
		c.Redirect(http.StatusFound, "/admin/v1/access/role/"+id+"/edit")
		return
	}
	setFlashSuccess(sessions.Default(c), "Update Role Success.")
	c.Redirect(http.StatusFound, "/admin/v1/access/role")
}

// Destroy → DELETE /admin/v1/access/role/:id/delete (form POST + ?_method=DELETE).
func (ctl *RoleController) Destroy(c *gin.Context) {
	if err := ctl.roles.Destroy(c.Request.Context(), c.Param("id")); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Delete Role Success.")
	}
	c.Redirect(http.StatusFound, "/admin/v1/access/role")
}

// allPermissions mengambil seluruh permission (untuk pilihan di form).
func (ctl *RoleController) allPermissions(c *gin.Context) ([]model.Permission, error) {
	res, err := ctl.perms.Index(c.Request.Context(), dto.ListQuery{Page: 1, PerPage: 100})
	if err != nil {
		return nil, err
	}
	return res.Data, nil
}

// --- Kelola permission per-role (padanan RoleController.permission* NodeAdmin) ---

// Permission → GET /admin/v1/access/role/:id/permission. Halaman assign/unassign
// permission untuk satu role (tabel SEMUA permission + penanda "assigned").
func (ctl *RoleController) Permission(c *gin.Context) {
	q, qbase := bindListQuery(c)
	res, role, err := ctl.roles.PermissionList(c.Request.Context(), c.Param("id"), q)
	if err != nil {
		c.Error(err)
		return
	}
	assigned := make(map[string]bool, len(role.Permissions))
	for _, p := range role.Permissions {
		assigned[p.ID] = true
	}
	view.RenderView(c, "roles/permission", gin.H{
		"title": "Kelola Permission Role", "active": "role",
		"role": role, "assigned": assigned,
		"permissions": res.Data, "meta": res.Meta,
		"filter": filterMap(q), "qbase": qbase,
	})
}

// PermissionAssign → GET /admin/v1/access/role/:id/permission/:permission_id/assign.
func (ctl *RoleController) PermissionAssign(c *gin.Context) {
	if err := ctl.roles.AssignPermission(c.Request.Context(), c.Param("id"), c.Param("permission_id")); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Assign Permission Success.")
	}
	redirectBack(c, "/admin/v1/access/role/"+c.Param("id")+"/permission")
}

// PermissionAssignSelected → POST /admin/v1/access/role/:id/permission/assign_selected.
func (ctl *RoleController) PermissionAssignSelected(c *gin.Context) {
	if err := ctl.roles.AssignPermissions(c.Request.Context(), c.Param("id"), selectedIDs(c)); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Assign Permission Success.")
	}
	redirectBack(c, "/admin/v1/access/role/"+c.Param("id")+"/permission")
}

// PermissionUnassign → GET /admin/v1/access/role/:id/permission/:permission_id/unassign.
func (ctl *RoleController) PermissionUnassign(c *gin.Context) {
	if err := ctl.roles.UnassignPermission(c.Request.Context(), c.Param("id"), c.Param("permission_id")); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Unassign Permission Success.")
	}
	redirectBack(c, "/admin/v1/access/role/"+c.Param("id")+"/permission")
}

// PermissionUnassignSelected → POST /admin/v1/access/role/:id/permission/unassign_selected.
func (ctl *RoleController) PermissionUnassignSelected(c *gin.Context) {
	if err := ctl.roles.UnassignPermissions(c.Request.Context(), c.Param("id"), selectedIDs(c)); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Unassign Permission Success.")
	}
	redirectBack(c, "/admin/v1/access/role/"+c.Param("id")+"/permission")
}
