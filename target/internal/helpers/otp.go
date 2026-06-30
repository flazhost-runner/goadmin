package helpers

import (
	"crypto/rand"
	"math/big"
)

// NewNumericOTP menghasilkan kode OTP numerik acak-kriptografis sepanjang
// digits (default 6). Dipakai reset password — disimpan ter-hash, bukan plaintext.
func NewNumericOTP(digits int) string {
	if digits < 1 {
		digits = 6
	}
	const d = "0123456789"
	b := make([]byte, digits)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(d))))
		if err != nil {
			// Fallback sangat jarang; tetap dalam ruang digit.
			b[i] = d[0]
			continue
		}
		b[i] = d[n.Int64()]
	}
	return string(b)
}
