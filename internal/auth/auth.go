package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"time"
)

type ApiKeyEntry struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Hash       string    `json:"hash"`
	CreatedAt  time.Time `json:"createdAt"`
	LastUsedAt time.Time `json:"lastUsedAt"`
	Enabled    bool      `json:"enabled"`
}

const KeyPrefix = "lah_"
const KeyBytes = 32

func GenerateKey() (rawKey string, hash string, err error) {
	buf := make([]byte, KeyBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	rawKey = KeyPrefix + base64.RawURLEncoding.EncodeToString(buf)
	hash = HashKey(rawKey)
	return rawKey, hash, nil
}

func HashKey(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}

func Verify(rawKey, hash string) bool {
	expected := HashKey(rawKey)
	return subtle.ConstantTimeCompare([]byte(expected), []byte(hash)) == 1
}

func KeyPrefixDisplay(rawKey string) string {
	if len(rawKey) <= 8 {
		return rawKey
	}
	return rawKey[:8] + "..."
}
