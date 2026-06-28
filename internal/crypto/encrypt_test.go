package crypto

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestNewEncryptor_HexKey(t *testing.T) {
	key := hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef"))
	e, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("hex key: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil encryptor")
	}
}

func TestNewEncryptor_Base64Key(t *testing.T) {
	key := "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="
	e, err := NewEncryptor(key)
	if err != nil {
		t.Fatalf("base64 key: %v", err)
	}
	if e == nil {
		t.Fatal("expected non-nil encryptor")
	}
}

func TestNewEncryptor_InvalidKey(t *testing.T) {
	_, err := NewEncryptor("too-short")
	if err == nil {
		t.Fatal("expected error for short key")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	e, err := NewEncryptor(hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))
	if err != nil {
		t.Fatal(err)
	}

	plaintext := "user@example.com"
	encrypted, err := e.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if encrypted == "" {
		t.Fatal("expected non-empty ciphertext")
	}
	if encrypted == plaintext {
		t.Fatal("ciphertext should differ from plaintext")
	}

	decrypted, err := e.Decrypt(encrypted)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if decrypted != plaintext {
		t.Fatalf("expected %q, got %q", plaintext, decrypted)
	}
}

func TestEncrypt_DifferentOutputs(t *testing.T) {
	e, _ := NewEncryptor(hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

	c1, _ := e.Encrypt("same@email.com")
	c2, _ := e.Encrypt("same@email.com")

	if c1 == c2 {
		t.Fatal("encrypting same plaintext should produce different outputs (nonce)")
	}
}

func TestDecrypt_Tampered(t *testing.T) {
	e, _ := NewEncryptor(hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

	enc, _ := e.Encrypt("test@email.com")
	tampered := enc[:len(enc)-1] + "A"

	_, err := e.Decrypt(tampered)
	if err == nil {
		t.Fatal("expected error for tampered ciphertext")
	}
}

func TestDecrypt_TooShort(t *testing.T) {
	e, _ := NewEncryptor(hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

	_, err := e.Decrypt("short")
	if err == nil {
		t.Fatal("expected error for short ciphertext")
	}
}

func TestDecrypt_InvalidBase64(t *testing.T) {
	e, _ := NewEncryptor(hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

	_, err := e.Decrypt("!!!not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestHashEmail(t *testing.T) {
	email := "user@example.com"
	hash := HashEmail(email)

	if len(hash) != 64 { // SHA-256 hex = 64 chars
		t.Fatalf("expected 64-char hex hash, got %d chars", len(hash))
	}

	hash2 := HashEmail(email)
	if hash != hash2 {
		t.Fatal("hash should be deterministic")
	}

	if HashEmail("other@example.com") == hash {
		t.Fatal("different emails should produce different hashes")
	}
}

func TestEncryptDecrypt_EmptyString(t *testing.T) {
	e, _ := NewEncryptor(hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

	enc, err := e.Encrypt("")
	if err != nil {
		t.Fatalf("encrypt empty: %v", err)
	}

	dec, err := e.Decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt empty: %v", err)
	}
	if dec != "" {
		t.Fatalf("expected empty string, got %q", dec)
	}
}

func TestEncryptDecrypt_LongString(t *testing.T) {
	e, _ := NewEncryptor(hex.EncodeToString([]byte("0123456789abcdef0123456789abcdef")))

	longStr := strings.Repeat("a", 10000)
	enc, err := e.Encrypt(longStr)
	if err != nil {
		t.Fatalf("encrypt long: %v", err)
	}

	dec, err := e.Decrypt(enc)
	if err != nil {
		t.Fatalf("decrypt long: %v", err)
	}
	if dec != longStr {
		t.Fatal("long string round-trip failed")
	}
}
