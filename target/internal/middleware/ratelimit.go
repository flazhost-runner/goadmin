package middleware

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	apperr "goadmin/internal/errors"
)

// RateLimiter membatasi jumlah request per-IP dalam jendela waktu (sliding
// window, in-memory). Dipakai pada endpoint sensitif (login/reset OTP) untuk
// meredam brute-force. Untuk deploy multi-instance, ganti dengan store bersama
// (mis. Redis) — kontraknya kecil sehingga mudah ditukar.
type RateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

// NewRateLimiter membuat limiter: maksimum max request per window per-IP.
func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{hits: make(map[string][]time.Time), max: max, window: window}
}

// Middleware menolak dengan 429 bila IP melebihi kuota.
func (rl *RateLimiter) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !rl.Allow(c.ClientIP()) {
			c.Error(apperr.New(http.StatusTooManyRequests, "Terlalu banyak percobaan. Coba lagi nanti."))
			c.Abort()
			return
		}
		c.Next()
	}
}

// Allow mencatat satu hit untuk key & mengembalikan false bila kuota terlampaui.
func (rl *RateLimiter) Allow(key string) bool {
	now := time.Now()
	cutoff := now.Add(-rl.window)

	rl.mu.Lock()
	defer rl.mu.Unlock()

	// Sisakan hanya hit yang masih dalam jendela.
	kept := rl.hits[key][:0]
	for _, t := range rl.hits[key] {
		if t.After(cutoff) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= rl.max {
		rl.hits[key] = kept
		return false
	}
	rl.hits[key] = append(kept, now)
	return true
}
