package auth

import (
	"context"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// TokenBlacklist menandai token (by jti) sebagai dicabut hingga kedaluwarsa.
// Saat logout, token di-blacklist dengan TTL = sisa masa berlaku — sehingga
// otomatis terhapus saat token memang sudah expired (hemat memori).
//
// PELAJARAN NODEADMIN (kritis): blacklist HARUS diuji terhadap store yang
// BERPERILAKU seperti runtime. Di NodeAdmin, client Redis mode-legacy membuat
// set/get callback-style → kode gaya Promise gagal SENYAP (token tetap valid
// setelah logout) tapi LOLOS test karena mock-nya flat-Promise. Maka:
//   - Interface ini punya SATU kontrak perilaku.
//   - Implementasi Redis & in-memory mengikuti kontrak yang SAMA PERSIS.
//   - Test memakai MemoryBlacklist (fidelity) ATAU redis nyata — bukan mock
//     yang "selalu mulus".
type TokenBlacklist interface {
	// Revoke menandai jti dicabut, kedaluwarsa setelah ttl.
	Revoke(ctx context.Context, jti string, ttl time.Duration) error
	// IsRevoked true bila jti masih dalam masa cabut.
	IsRevoked(ctx context.Context, jti string) (bool, error)
}

const blacklistPrefix = "jwt:blacklist:"

// RedisBlacklist menyimpan token tercabut di Redis (stateless, siap multi-instance).
type RedisBlacklist struct {
	client *redis.Client
}

// NewRedisBlacklist membuat blacklist berbasis Redis.
func NewRedisBlacklist(client *redis.Client) *RedisBlacklist {
	return &RedisBlacklist{client: client}
}

func (b *RedisBlacklist) Revoke(ctx context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Minute // minimal, agar key tetap ada sejenak
	}
	// SET key "1" EX ttl — API yang SAMA dengan yang diuji (fidelity).
	return b.client.Set(ctx, blacklistPrefix+jti, "1", ttl).Err()
}

func (b *RedisBlacklist) IsRevoked(ctx context.Context, jti string) (bool, error) {
	n, err := b.client.Exists(ctx, blacklistPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// MemoryBlacklist adalah implementasi in-memory yang BERPERILAKU IDENTIK dengan
// RedisBlacklist (termasuk kedaluwarsa TTL). Dipakai test & mode tanpa redis.
type MemoryBlacklist struct {
	mu      sync.RWMutex
	entries map[string]time.Time // jti → waktu kedaluwarsa
}

// NewMemoryBlacklist membuat blacklist in-memory.
func NewMemoryBlacklist() *MemoryBlacklist {
	return &MemoryBlacklist{entries: map[string]time.Time{}}
}

func (b *MemoryBlacklist) Revoke(_ context.Context, jti string, ttl time.Duration) error {
	if ttl <= 0 {
		ttl = time.Minute
	}
	b.mu.Lock()
	defer b.mu.Unlock()
	b.entries[jti] = time.Now().Add(ttl)
	return nil
}

func (b *MemoryBlacklist) IsRevoked(_ context.Context, jti string) (bool, error) {
	b.mu.RLock()
	exp, ok := b.entries[jti]
	b.mu.RUnlock()
	if !ok {
		return false, nil
	}
	// Hormati TTL: bila sudah lewat, anggap tak tercabut (dan bersihkan).
	if time.Now().After(exp) {
		b.mu.Lock()
		delete(b.entries, jti)
		b.mu.Unlock()
		return false, nil
	}
	return true, nil
}
