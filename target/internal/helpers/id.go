package helpers

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// NewID menghasilkan UUID v4 sebagai string (primary key portabel lintas dialek).
func NewID() string {
	return uuid.NewString()
}

// NewCode menghasilkan kode unik singkat berbasis waktu + sufiks UUID,
// dipakai untuk kolom 'code' user (mis. "U-20260622-AB12").
func NewCode(prefix string) string {
	now := time.Now().UTC()
	suffix := strings.ToUpper(uuid.NewString()[0:4])
	return fmt.Sprintf("%s-%s-%s", prefix, now.Format("20060102"), suffix)
}
