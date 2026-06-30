package middleware

import (
	"encoding/json"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// Kunci flash di sesi (one-shot feedback pasca-redirect, pola PRG).
// FlashKey menyimpan satu objek JSON {key:"success"|"error", message:"..."}
// — format NodeAdmin: satu kunci tunggal, bukan flash_success/flash_error terpisah.
const (
	FlashKey       = "flash"        // unified flash session key
	FieldErrorsKey = "field_errors" // JSON map field→pesan (validasi inline)
	FieldOldKey    = "field_old"    // JSON map field→nilai lama (repopulasi)
)

// flashMsg adalah bentuk JSON yang disimpan di sesi untuk flash one-shot.
type flashMsg struct {
	Key     string `json:"key"`
	Message string `json:"message"`
}

// SetFlashSuccess menyimpan flash sukses ke sesi (pola PRG).
func SetFlashSuccess(sess sessions.Session, msg string) {
	b, _ := json.Marshal(flashMsg{Key: "success", Message: msg})
	sess.Set(FlashKey, string(b))
	_ = sess.Save()
}

// SetFlashError menyimpan flash error ke sesi (pola PRG).
func SetFlashError(sess sessions.Session, msg string) {
	b, _ := json.Marshal(flashMsg{Key: "error", Message: msg})
	sess.Set(FlashKey, string(b))
	_ = sess.Save()
}

// SetFieldErrors menyimpan error per-field + nilai lama (old input) ke sesi untuk
// SATU redirect (pola PRG). Padanan `req.session.errors`/`req.session.old` +
// helper `getError`/`old` NodeAdmin — view menampilkan error inline & mengisi
// ulang form. Disimpan sebagai JSON (cookie-session aman tanpa gob-register).
func SetFieldErrors(sess sessions.Session, errs, old map[string]string) {
	if b, err := json.Marshal(errs); err == nil {
		sess.Set(FieldErrorsKey, string(b))
	}
	if b, err := json.Marshal(old); err == nil {
		sess.Set(FieldOldKey, string(b))
	}
	_ = sess.Save()
}

// Flash memindahkan pesan flash dari sesi ke context (sekali pakai → dihapus
// dari sesi). RenderView lalu menaruhnya ke locals (`flash_success`/`flash_error`
// + `errors`/`old`) agar chrome/halaman menampilkannya. Hanya jalur web (sesi).
func Flash() gin.HandlerFunc {
	return func(c *gin.Context) {
		sess := sessions.Default(c)
		changed := false
		// Unified flash: baca satu kunci, parse JSON {key, message}.
		if v, ok := sess.Get(FlashKey).(string); ok && v != "" {
			var fm flashMsg
			if json.Unmarshal([]byte(v), &fm) == nil {
				if fm.Key == "success" {
					c.Set("flash_success", fm.Message)
				} else if fm.Key == "error" {
					c.Set("flash_error", fm.Message)
				}
			}
			sess.Delete(FlashKey)
			changed = true
		}
		// Error per-field + old input (JSON) → parse ke map → context.
		if v, ok := sess.Get(FieldErrorsKey).(string); ok && v != "" {
			var m map[string]string
			if json.Unmarshal([]byte(v), &m) == nil {
				c.Set(FieldErrorsKey, m)
			}
			sess.Delete(FieldErrorsKey)
			changed = true
		}
		if v, ok := sess.Get(FieldOldKey).(string); ok && v != "" {
			var m map[string]string
			if json.Unmarshal([]byte(v), &m) == nil {
				c.Set(FieldOldKey, m)
			}
			sess.Delete(FieldOldKey)
			changed = true
		}
		if changed {
			_ = sess.Save()
		}
		c.Next()
	}
}
