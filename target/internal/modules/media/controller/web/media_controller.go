// Package web berisi controller AJAX modul media (file manager rich text editor).
// Mengembalikan JSON (bukan view) — dipakai filemanager.js. Checker hanya
// melarang c.HTML( di controller web; c.JSON diizinkan.
package web

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"goadmin/internal/storage"
)

// MediaController menangani upload/list/delete gambar editor ke storage.
type MediaController struct {
	storage storage.Storage
}

// NewMediaController merakit controller (storage di-inject).
func NewMediaController(store storage.Storage) *MediaController {
	return &MediaController{storage: store}
}

// List → GET /admin/v1/media/list (daftar berkas editor).
func (ctl *MediaController) List(c *gin.Context) {
	files, err := ctl.storage.ListMedia(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "message": "Gagal memuat daftar file."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": files})
}

// Upload → POST /admin/v1/media/upload (multipart "file"): validasi magic-byte +
// re-encode → simpan ke folder editor → {name,url,key}.
func (ctl *MediaController) Upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil || fh == nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "File tidak ditemukan."})
		return
	}
	f, oerr := fh.Open()
	if oerr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Gagal membuka file."})
		return
	}
	defer f.Close()

	mf, serr := storage.ValidateAndSaveMedia(c.Request.Context(), ctl.storage, f)
	if serr != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Gagal mengunggah: " + serr.Error()})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"success": true, "message": "File diunggah", "data": mf})
}

// Delete → POST /admin/v1/media/delete (form "key"): hapus berkas editor.
func (ctl *MediaController) Delete(c *gin.Context) {
	if err := ctl.storage.DeleteMedia(c.Request.Context(), c.PostForm("key")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "Gagal menghapus file."})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "File dihapus"})
}
