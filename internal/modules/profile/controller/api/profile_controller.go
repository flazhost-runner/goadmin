package api

import (
	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
	"goadmin/internal/helpers"
	accessmw "goadmin/internal/modules/access/middleware"
	"goadmin/internal/modules/profile/dto"
	"goadmin/internal/modules/profile/service"
)

// ProfileController = REST profil milik-sendiri (show + update).
type ProfileController struct {
	profiles service.IProfileService
}

// NewProfileController merakit controller (service di-inject).
func NewProfileController(profiles service.IProfileService) *ProfileController {
	return &ProfileController{profiles: profiles}
}

// Show → GET /api/v1/profile (profil user dari token).
func (ctl *ProfileController) Index(c *gin.Context) {
	user := accessmw.UserFrom(c)
	if user == nil {
		c.Error(apperr.Unauthorized("Belum terautentikasi"))
		return
	}
	profile, err := ctl.profiles.Get(c.Request.Context(), user.ID)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "OK", profile)
}

// Update → PUT /api/v1/profile.
func (ctl *ProfileController) Update(c *gin.Context) {
	user := accessmw.UserFrom(c)
	if user == nil {
		c.Error(apperr.Unauthorized("Belum terautentikasi"))
		return
	}
	var in dto.UpdateProfileInput
	if err := c.ShouldBindJSON(&in); err != nil {
		c.Error(apperr.Validation("Input tidak valid", nil))
		return
	}
	updated, err := ctl.profiles.Update(c.Request.Context(), user.ID, in)
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "Profil diperbarui", updated)
}
