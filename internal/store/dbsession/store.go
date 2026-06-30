// Package dbsession implements gin-contrib/sessions.Store backed by GORM.
// Tabel `sessions` di-auto-migrate saat New() dipanggil.
package dbsession

import (
	"encoding/base32"
	"net/http"
	"strings"
	"time"

	"github.com/gin-contrib/sessions"
	"github.com/gorilla/securecookie"
	gsessions "github.com/gorilla/sessions"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// record adalah model GORM untuk tabel `sessions`.
type record struct {
	ID        string    `gorm:"primaryKey;type:varchar(128)"`
	Data      string    `gorm:"type:text"`
	ExpiresAt time.Time `gorm:"index"`
}

func (record) TableName() string { return "sessions" }

// Store implements sessions.Store (gin-contrib) via GORM.
type Store struct {
	db      *gorm.DB
	codecs  []securecookie.Codec
	options *gsessions.Options
}

// New membuat Store baru dan auto-migrasi tabel sessions.
// keyPairs: pasangan hashKey[, blockKey] — format sama seperti securecookie.
func New(db *gorm.DB, keyPairs ...[]byte) (*Store, error) {
	if err := db.AutoMigrate(&record{}); err != nil {
		return nil, err
	}
	return &Store{
		db:     db,
		codecs: securecookie.CodecsFromPairs(keyPairs...),
		options: &gsessions.Options{
			Path:   "/",
			MaxAge: 86400 * 7,
		},
	}, nil
}

// Options meng-update opsi cookie default (dipanggil gin-contrib saat setup).
func (s *Store) Options(opts sessions.Options) {
	s.options = &gsessions.Options{
		Path:     opts.Path,
		Domain:   opts.Domain,
		MaxAge:   opts.MaxAge,
		Secure:   opts.Secure,
		HttpOnly: opts.HttpOnly,
		SameSite: opts.SameSite,
	}
}

// Get mengambil sesi yang sudah ada atau membuat yang baru via registry gorilla.
func (s *Store) Get(r *http.Request, name string) (*gsessions.Session, error) {
	return gsessions.GetRegistry(r).Get(s, name)
}

// New mengambil sesi dari DB (atau membuat baru jika tidak ditemukan/kedaluwarsa).
func (s *Store) New(r *http.Request, name string) (*gsessions.Session, error) {
	sess := gsessions.NewSession(s, name)
	opts := *s.options
	sess.Options = &opts
	sess.IsNew = true

	c, err := r.Cookie(name)
	if err != nil {
		return sess, nil
	}
	if err = securecookie.DecodeMulti(name, c.Value, &sess.ID, s.codecs...); err != nil {
		return sess, nil
	}

	var rec record
	if err := s.db.Where("id = ? AND expires_at > ?", sess.ID, time.Now()).
		First(&rec).Error; err != nil {
		return sess, nil
	}
	if err := securecookie.DecodeMulti(sess.ID, rec.Data, &sess.Values, s.codecs...); err != nil {
		return sess, nil
	}
	sess.IsNew = false
	return sess, nil
}

// Save menyimpan sesi ke DB dan mengatur cookie di response.
func (s *Store) Save(r *http.Request, w http.ResponseWriter, sess *gsessions.Session) error {
	if sess.Options.MaxAge <= 0 {
		s.db.Delete(&record{}, "id = ?", sess.ID)
		http.SetCookie(w, gsessions.NewCookie(sess.Name(), "", sess.Options))
		return nil
	}

	if sess.ID == "" {
		sess.ID = strings.TrimRight(
			base32.StdEncoding.EncodeToString(securecookie.GenerateRandomKey(32)), "=")
	}

	encoded, err := securecookie.EncodeMulti(sess.ID, sess.Values, s.codecs...)
	if err != nil {
		return err
	}

	expiresAt := time.Now().Add(time.Duration(sess.Options.MaxAge) * time.Second)
	if err := s.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns([]string{"data", "expires_at"}),
	}).Create(&record{
		ID:        sess.ID,
		Data:      encoded,
		ExpiresAt: expiresAt,
	}).Error; err != nil {
		return err
	}

	cookieVal, err := securecookie.EncodeMulti(sess.Name(), sess.ID, s.codecs...)
	if err != nil {
		return err
	}
	http.SetCookie(w, gsessions.NewCookie(sess.Name(), cookieVal, sess.Options))
	return nil
}
