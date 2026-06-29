// Package web berisi controller landing publik (home). Render lewat
// view.RenderView (bukan c.HTML path mentah) — di-enforce checker.
package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
	"goadmin/internal/modules/home/fetemplate"
	"goadmin/internal/modules/home/service"
	"goadmin/internal/view"
)

// HomeController menyajikan landing publik (root + /home). Template builtin
// dirender lewat Go view; template eksternal disajikan dari HTML ter-cache.
type HomeController struct {
	home service.IHomeService
	fe   *fetemplate.Service
}

// NewHomeController merakit controller (service di-inject).
func NewHomeController(home service.IHomeService, fe *fetemplate.Service) *HomeController {
	return &HomeController{home: home, fe: fe}
}

// Index → GET / dan GET /home (publik). Data terikat ke Setting; template aktif
// menentukan render: builtin (Go view) vs eksternal (HTML ter-cache).
func (ctl *HomeController) Index(c *gin.Context) {
	landing, err := ctl.home.Landing(c.Request.Context())
	if err != nil {
		c.Error(err)
		return
	}

	if fetemplate.IsBuiltin(landing.Template) {
		view.RenderView(c, "home/"+fetemplate.BuiltinView(landing.Template), gin.H{
			"title":   landing.AppName,
			"landing": landing,
		})
		return
	}

	// Eksternal: sajikan HTML mentah ter-cache (fallback ke builtin default).
	html, herr := ctl.fe.ActiveHTML(c.Request.Context(), landing.Template)
	if herr != nil {
		view.RenderView(c, "home/"+fetemplate.BuiltinView(fetemplate.DefaultSlug), gin.H{
			"title":   landing.AppName,
			"landing": landing,
		})
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// Preview → GET /admin/v1/setting/fe-preview/:slug (admin; thumbnail/modal Setting). Mengembalikan HTML
// satu template untuk dirender di iframe (thumbnail/modal). Builtin → Go view;
// eksternal → HTML proxy (cache lokal → upstream); gagal → placeholder ramah.
func (ctl *HomeController) Preview(c *gin.Context) {
	slug := c.Param("slug")
	if !fetemplate.IsValidSlug(slug) {
		c.Error(apperr.BadRequest("Template tidak dikenali"))
		return
	}

	if fetemplate.IsBuiltin(slug) {
		landing, err := ctl.home.Landing(c.Request.Context())
		if err != nil {
			c.Error(err)
			return
		}
		view.RenderView(c, "home/"+fetemplate.BuiltinView(slug), gin.H{"title": landing.AppName, "landing": landing})
		return
	}

	html, err := ctl.fe.PreviewHTML(c.Request.Context(), slug)
	if err != nil {
		c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(previewPlaceholder(slug)))
		return
	}
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(html))
}

// previewPlaceholder = halaman kecil saat pratinjau tak tersedia (mis. template
// eksternal belum diunduh & mode remote nonaktif).
func previewPlaceholder(slug string) string {
	return `<!doctype html><meta charset="utf-8"><body style="margin:0;font-family:sans-serif;` +
		`display:flex;align-items:center;justify-content:center;height:100vh;background:#f1f5f9;color:#64748b">` +
		`<div style="text-align:center;padding:16px"><div style="font-size:13px">Pratinjau belum tersedia</div>` +
		`<div style="font-size:11px;margin-top:4px;opacity:.7">` + slug + `</div></div></body>`
}
