// Package web berisi controller HTML modul dashboard. Render lewat
// view.RenderView (bukan c.HTML path mentah) — di-enforce checker.
package web

import (
	"github.com/gin-gonic/gin"

	"goadmin/internal/modules/dashboard/service"
	"goadmin/internal/view"
)

// DashboardController menyajikan halaman dashboard admin (kartu statistik).
type DashboardController struct {
	dashboard service.IDashboardService
}

// NewDashboardController merakit controller (service di-inject).
func NewDashboardController(dashboard service.IDashboardService) *DashboardController {
	return &DashboardController{dashboard: dashboard}
}

// Index → GET /admin/v1/dashboard.
func (ctl *DashboardController) Index(c *gin.Context) {
	stats, err := ctl.dashboard.Stats(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}
	view.RenderView(c, "dashboard/index", gin.H{
		"title":  "Dashboard",
		"active": "dashboard",
		"stats":  stats,
	})
}
