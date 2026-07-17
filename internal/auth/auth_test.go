package auth

import (
	"strings"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	raw, hash, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(raw, KeyPrefix) {
		t.Errorf("expected prefix %q, got %q", KeyPrefix, raw)
	}
	if len(raw) <= len(KeyPrefix) {
		t.Error("key too short")
	}
	if hash == "" {
		t.Error("hash should not be empty")
	}
}

func TestHashKeyDeterministic(t *testing.T) {
	raw := "lah_testkey123"
	h1 := HashKey(raw)
	h2 := HashKey(raw)
	if h1 != h2 {
		t.Error("hash should be deterministic")
	}
}

func TestVerify(t *testing.T) {
	raw, hash, err := GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	if !Verify(raw, hash) {
		t.Error("Verify should return true for matching key")
	}
	if Verify(raw+"tampered", hash) {
		t.Error("Verify should return false for tampered key")
	}
	if Verify(raw, "0000000000000000000000000000000000000000000000000000000000000000") {
		t.Error("Verify should return false for wrong hash")
	}
}

func TestKeyPrefixDisplay(t *testing.T) {
	prefix := KeyPrefixDisplay("lah_abcd1234extra")
	if !strings.HasPrefix(prefix, "lah_abcd") {
		t.Errorf("unexpected prefix display: %q", prefix)
	}
	if !strings.HasSuffix(prefix, "...") {
		t.Error("expected ellipsis suffix")
	}

	short := KeyPrefixDisplay("short")
	if short != "short" {
		t.Errorf("short key should not be truncated, got %q", short)
	}
}
