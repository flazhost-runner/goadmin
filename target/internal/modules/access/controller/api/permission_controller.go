package api

import (
	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	"goadmin/internal/modules/access/dto"
	"goadmin/internal/modules/access/service"
)

// PermissionController = CRUD permission (path verbose persis NodeAdmin, bukan REST).
type PermissionController struct {
	perms service.IPermissionService
}

// NewPermissionController merakit controller.
func NewPermissionController(perms service.IPermissionService) *PermissionController {
	return &PermissionController{perms: perms}
}

// Index → GET /api/v1/access/permission.
func (ctl *PermissionController) Index(c *gin.Context) {
	// Sinkronkan permission dari named-route registry (route-driven, a la
	// NodeAdmin) — lazy tiap akses; idempoten.
	_ = ctl.perms.SyncFromRoutes(c.Request.Context())
	var q dto.ListQuery
	_ = c.ShouldBindQuery(&q)
	res, err := ctl.perms.Index(c.Request.Context(), q)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "OK", res)
}

// Edit → GET /api/v1/access/permission/:id/edit (kembar web; kembalikan entity utk edit).
func (ctl *PermissionController) Edit(c *gin.Context) {
	perm, err := ctl.perms.Show(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "OK", perm)
}

// Store → POST /api/v1/access/permission/store.
func (ctl *PermissionController) Store(c *gin.Context) {
	var in dto.CreatePermissionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	perm, err := ctl.perms.Store(c.Request.Context(), in)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.Created(c, "Permission dibuat", perm)
}

// Update → PUT /api/v1/access/permission/:id/update.
func (ctl *PermissionController) Update(c *gin.Context) {
	var in dto.UpdatePermissionInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	perm, err := ctl.perms.Update(c.Request.Context(), c.Param("id"), in)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Permission diperbarui", perm)
}

// Destroy → DELETE /api/v1/access/permission/:id/delete.
func (ctl *PermissionController) Destroy(c *gin.Context) {
	if err := ctl.perms.Destroy(c.Request.Context(), c.Param("id")); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Permission dihapus", nil)
}

// DeleteSelected → POST /api/v1/access/permission/delete_selected (body `{ selected: [id,...] }`).
func (ctl *PermissionController) DeleteSelected(c *gin.Context) {
	var in struct {
		Selected []string `json:"selected"`
	}
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	if err := ctl.perms.DestroyMany(c.Request.Context(), in.Selected); err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Permission terpilih dihapus", nil)
}
