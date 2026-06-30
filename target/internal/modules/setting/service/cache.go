package service

import (
	"sync"
	"time"

	"goadmin/internal/modules/setting/model"
)

// settingCache menyimpan setting global di memori dengan TTL + invalidasi saat
// update. Setting dibaca tiap request (layout/landing) — cache menghindari
// query berulang (checklist performa NodeAdmin). Aman untuk akses konkuren.
//
// Catatan stateless/horizontal-scaling: pada deploy multi-instance, ganti
// implementasi ini dengan store bersama (Redis) + pub/sub invalidasi. Kontrak
// (get/set/invalidate) sengaja kecil agar mudah ditukar.
type settingCache struct {
	mu      sync.RWMutex
	value   *model.Setting
	expires time.Time
	ttl     time.Duration
}

func newSettingCache(ttl time.Duration) *settingCache {
	return &settingCache{ttl: ttl}
}

// get mengembalikan setting ter-cache (nil bila kosong/kedaluwarsa).
func (c *settingCache) get() *model.Setting {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.value == nil || time.Now().After(c.expires) {
		return nil
	}
	return c.value
}

// set mengisi cache + memperbarui masa berlaku.
func (c *settingCache) set(s *model.Setting) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = s
	c.expires = time.Now().Add(c.ttl)
}

// invalidate mengosongkan cache (dipanggil setelah update agar perubahan
// langsung tampil di request berikutnya).
func (c *settingCache) invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value = nil
}
