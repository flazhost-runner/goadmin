package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims adalah payload JWT GoAdmin.
type Claims struct {
	UserID string `json:"id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

// JWTManager menerbitkan & memverifikasi token dengan algoritma DI-PIN (HS256).
// Pinning algoritma mencegah serangan "alg=none" / downgrade.
type JWTManager struct {
	secret    []byte
	expiresIn time.Duration
}

// NewJWTManager membuat manager. expiresIn default 1 jam bila <= 0.
func NewJWTManager(secret string, expiresIn time.Duration) *JWTManager {
	if expiresIn <= 0 {
		expiresIn = time.Hour
	}
	return &JWTManager{secret: []byte(secret), expiresIn: expiresIn}
}

// Generate menerbitkan token + jti (id token, untuk blacklist) + waktu kedaluwarsa.
func (m *JWTManager) Generate(userID, email, jti string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(m.expiresIn)
	claims := Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	return signed, exp, err
}

// Verify memvalidasi tanda tangan + algoritma + masa berlaku, mengembalikan claims.
func (m *JWTManager) Verify(tokenStr string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		// Pin algoritma: hanya HS256 yang diterima.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("algoritma token tak valid")
		}
		return m.secret, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("token tidak valid")
	}
	return claims, nil
}
