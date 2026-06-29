package api

import (
	"github.com/gin-gonic/gin"

	"goadmin/internal/helpers"
	"goadmin/internal/modules/dashboard/service"
)

// DashboardController = REST statistik dashboard (read-only).
type DashboardController struct {
	dashboard service.IDashboardService
}

// NewDashboardController merakit controller (service di-inject).
func NewDashboardController(dashboard service.IDashboardService) *DashboardController {
	return &DashboardController{dashboard: dashboard}
}

// Index → GET /api/v1/dashboard.
func (ctl *DashboardController) Index(c *gin.Context) {
	stats, err := ctl.dashboard.Stats(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	helpers.OK(c, "OK", stats)
}
