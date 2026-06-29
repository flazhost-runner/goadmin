package web

import (
	"net/http"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"

	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/service"
	"goadmin/internal/view"
)

// PermissionController menyajikan CRUD permission (web) — hanya nama.
type PermissionController struct {
	perms service.IPermissionService
}

// NewPermissionController merakit controller (service di-inject).
func NewPermissionController(perms service.IPermissionService) *PermissionController {
	return &PermissionController{perms: perms}
}

// Index → GET /admin/v1/access/permission.
func (ctl *PermissionController) Index(c *gin.Context) {
	// Sinkronkan permission dari named-route registry (route-driven, a la
	// NodeAdmin getAllRegisteredRoute) — lazy tiap buka halaman; idempoten.
	_ = ctl.perms.SyncFromRoutes(c.Request.Context())
	q, qbase := bindListQuery(c)
	res, err := ctl.perms.Index(c.Request.Context(), q)
	if err != nil {
		c.Error(err)
		return
	}
	view.RenderView(c, "permissions/index", gin.H{
		"title": "Manajemen Permission", "active": "permission",
		"permissions": res.Data, "meta": res.Meta,
		"filter": filterMap(q), "qbase": qbase,
	})
}

// DeleteSelected → POST /admin/v1/access/permission/delete_selected (bulk delete).
func (ctl *PermissionController) DeleteSelected(c *gin.Context) {
	if err := ctl.perms.DestroyMany(c.Request.Context(), selectedIDs(c)); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Delete Permission Success.")
	}
	c.Redirect(http.StatusFound, "/admin/v1/access/permission")
}

// Create → GET /admin/v1/access/permission/create.
func (ctl *PermissionController) Create(c *gin.Context) {
	view.RenderView(c, "permissions/create", gin.H{
		"title": "Tambah Permission", "active": "permission",
	})
}

// Store → POST /admin/v1/access/permission.
func (ctl *PermissionController) Store(c *gin.Context) {
	var in dto.CreatePermissionInput
	_ = c.ShouldBind(&in)
	if _, err := ctl.perms.Store(c.Request.Context(), in); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
		c.Redirect(http.StatusFound, "/admin/v1/access/permission/create")
		return
	}
	setFlashSuccess(sessions.Default(c), "Create Permission Success.")
	c.Redirect(http.StatusFound, "/admin/v1/access/permission")
}

// Edit → GET /admin/v1/access/permission/:id/edit.
func (ctl *PermissionController) Edit(c *gin.Context) {
	perm, err := ctl.perms.Show(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	view.RenderView(c, "permissions/edit", gin.H{
		"title": "Ubah Permission", "active": "permission",
		"action": "/admin/v1/access/permission/" + perm.ID + "/update?_method=PUT", "permission": perm,
	})
}

// Update → POST /admin/v1/access/permission/:id.
func (ctl *PermissionController) Update(c *gin.Context) {
	id := c.Param("id")
	var in dto.UpdatePermissionInput
	_ = c.ShouldBind(&in)
	if _, err := ctl.perms.Update(c.Request.Context(), id, in); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
		c.Redirect(http.StatusFound, "/admin/v1/access/permission/"+id+"/edit")
		return
	}
	setFlashSuccess(sessions.Default(c), "Update Permission Success.")
	c.Redirect(http.StatusFound, "/admin/v1/access/permission")
}

// Destroy → DELETE /admin/v1/access/permission/:id/delete (form POST + ?_method=DELETE).
func (ctl *PermissionController) Destroy(c *gin.Context) {
	if err := ctl.perms.Destroy(c.Request.Context(), c.Param("id")); err != nil {
		setFlashError(sessions.Default(c), errMessage(err))
	} else {
		setFlashSuccess(sessions.Default(c), "Permission berhasil dihapus.")
	}
	c.Redirect(http.StatusFound, "/admin/v1/access/permission")
}
