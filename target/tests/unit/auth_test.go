package unit

import (
	"context"
	"testing"
	"time"

	"goadmin/internal/auth"
)

// Unit: JWT pinning algoritma + verifikasi round-trip.
func TestJWT_GenerateVerify(t *testing.T) {
	m := auth.NewJWTManager("secret-x", time.Hour)
	token, exp, err := m.Generate("u1", "u1@example.com", "jti-1")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if exp.Before(time.Now()) {
		t.Fatal("exp di masa lalu")
	}
	claims, err := m.Verify(token)
	if err != nil {
		t.Fatalf("verify: %v", err)
	}
	if claims.UserID != "u1" || claims.ID != "jti-1" {
		t.Fatalf("claims salah: %+v", claims)
	}
}

func TestJWT_RejectWrongSecret(t *testing.T) {
	m1 := auth.NewJWTManager("secret-a", time.Hour)
	m2 := auth.NewJWTManager("secret-b", time.Hour)
	token, _, _ := m1.Generate("u1", "e", "j")
	if _, err := m2.Verify(token); err == nil {
		t.Fatal("token dengan secret beda harus ditolak")
	}
}

// Unit: blacklist in-memory MENGHORMATI TTL (fidelity perilaku runtime).
func TestMemoryBlacklist_TTL(t *testing.T) {
	bl := auth.NewMemoryBlacklist()
	ctx := context.Background()

	_ = bl.Revoke(ctx, "jti-a", 50*time.Millisecond)
	revoked, _ := bl.IsRevoked(ctx, "jti-a")
	if !revoked {
		t.Fatal("token harus tercabut tepat setelah revoke")
	}

	time.Sleep(70 * time.Millisecond)
	revoked, _ = bl.IsRevoked(ctx, "jti-a")
	if revoked {
		t.Fatal("token harus tak-tercabut setelah TTL lewat")
	}
}

func TestPassword_HashCheck(t *testing.T) {
	hash, err := auth.HashPassword("rahasia123", 4)
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "rahasia123" {
		t.Fatal("hash == plaintext")
	}
	if !auth.CheckPassword(hash, "rahasia123") {
		t.Fatal("check password benar harus true")
	}
	if auth.CheckPassword(hash, "salah") {
		t.Fatal("check password salah harus false")
	}
}
