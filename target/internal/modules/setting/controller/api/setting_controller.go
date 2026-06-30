package api

import (
	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	accessmw "goadmin/internal/modules/access/middleware"
	"goadmin/internal/modules/setting/dto"
	"goadmin/internal/modules/setting/service"
)

// SettingController = REST setting global (singleton: show + update).
type SettingController struct {
	settings service.ISettingService
}

// NewSettingController merakit controller (service di-inject).
func NewSettingController(settings service.ISettingService) *SettingController {
	return &SettingController{settings: settings}
}

// Show → GET /api/v1/setting (setting tunggal + katalog tema).
func (ctl *SettingController) Index(c *gin.Context) {
	setting, err := ctl.settings.Get(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "OK", gin.H{"setting": setting, "themes": ctl.settings.Themes()})
}

// Update → PUT /api/v1/setting (update parsial).
func (ctl *SettingController) Update(c *gin.Context) {
	var in dto.UpdateSettingInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	setting, err := ctl.settings.Update(c.Request.Context(), in, actorID(c))
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Setting diperbarui", setting)
}

// actorID mengambil id user terotentikasi dari context (kosong bila tak ada).
func actorID(c *gin.Context) string {
	if u := accessmw.UserFrom(c); u != nil {
		return u.ID
	}
	return ""
}
