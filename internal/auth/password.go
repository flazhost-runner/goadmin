// Package auth menyediakan primitif autentikasi: hashing password (bcrypt),
// token JWT (HS256 di-pin), dan blacklist token saat logout.
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword menghasilkan hash bcrypt dengan cost tertentu (dari env).
func HashPassword(plain string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	b, err := bcrypt.GenerateFromPassword([]byte(plain), cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// CheckPassword membandingkan plaintext dengan hash bcrypt.
func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
