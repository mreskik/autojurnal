package token

import (
	"crypto/rand"
	"math/big"
)

const (
	// Length: panjang default token runner / jurnal.
	Length   = 40
	alphabet = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// Generate membuat string acak alfanumerik dari crypto/rand.
func Generate(length int) (string, error) {
	b := make([]byte, length)
	max := big.NewInt(int64(len(alphabet)))
	for i := range b {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			return "", err
		}
		b[i] = alphabet[n.Int64()]
	}
	return string(b), nil
}
