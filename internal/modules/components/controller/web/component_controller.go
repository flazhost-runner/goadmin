// Package web berisi controller halaman showcase komponen UI — acuan visual +
// markup untuk membuat elemen serupa (stat card, badge, tabel, form, alert,
// button). Statis, tanpa service. Render lewat view.RenderView (di-enforce checker).
package web

import (
	"github.com/gin-gonic/gin"

	"goadmin/internal/view"
)

// ComponentController menyajikan katalog komponen UI.
type ComponentController struct{}

// NewComponentController merakit controller (tanpa dependency).
func NewComponentController() *ComponentController {
	return &ComponentController{}
}

// Index → GET /admin/v1/components (showcase 9 seksi; konten statis + bertema).
func (ctl *ComponentController) Index(c *gin.Context) {
	view.RenderView(c, "components/index", gin.H{
		"title":  "UI Components",
		"active": "components",
	})
}
