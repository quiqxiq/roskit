package encrypt

import (
	"encoding/base64"
	"testing"
)

const testKeyB64 = "01234567890123456789012345678901"

func TestEncryptDecryptRoundtrip(t *testing.T) {
	plaintext := "my-secret-router-password"
	encrypted, err := Encrypt(plaintext, testKeyB64)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := Decrypt(encrypted, testKeyB64)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("roundtrip mismatch: got %q, want %q", decrypted, plaintext)
	}
}

func TestDecryptLegacyPHPConfig(t *testing.T) {
	plaintext := "router123"
	xored := make([]byte, len(plaintext))
	for i, b := range []byte(plaintext) {
		xored[i] = b ^ 128
	}
	encrypted := base64.StdEncoding.EncodeToString(xored)

	result, err := DecryptLegacyPHPConfig(encrypted)
	if err != nil {
		t.Fatalf("DecryptLegacyPHPConfig failed: %v", err)
	}
	if result != plaintext {
		t.Errorf("got %q, want %q", result, plaintext)
	}
}

func encodeBlah(plaintext string) []byte {
	stage1 := base64.StdEncoding.EncodeToString([]byte(plaintext))
	stage2 := base64.StdEncoding.EncodeToString([]byte(stage1))
	xored := make([]byte, len(stage2))
	for i, b := range []byte(stage2) {
		xored[i] = b ^ 10
	}
	return xored
}

func TestDecodeBlah(t *testing.T) {
	plaintext := "admin"
	encoded := string(encodeBlah(plaintext))

	result, err := DecodeBlah(encoded)
	if err != nil {
		t.Fatalf("DecodeBlah failed: %v", err)
	}
	if result != plaintext {
		t.Errorf("got %q, want %q", result, plaintext)
	}
}

func TestDecryptInvalidKeyB64(t *testing.T) {
	_, err := Encrypt("test", "not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64 key")
	}
}

func TestDecryptLegacyInvalidB64(t *testing.T) {
	_, err := DecryptLegacyPHPConfig("not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64 input")
	}
}

func TestDecodeBlahInvalidB64(t *testing.T) {
	_, err := DecodeBlah("not-base64!!!")
	if err == nil {
		t.Fatal("expected error for invalid base64 input")
	}
}

func TestDecryptWrongKey(t *testing.T) {
	_, err := Encrypt("test", testKeyB64)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	otherKey := base64.StdEncoding.EncodeToString([]byte("abcdefghijklmnopqrstuvwxyz012345"))
	_, err = Decrypt("invalid", otherKey)
	if err == nil {
		t.Fatal("expected error for invalid ciphertext")
	}
}

func TestDecryptCorruptCiphertext(t *testing.T) {
	encrypted, err := Encrypt("test", testKeyB64)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	corrupt := encrypted[:len(encrypted)-2] + "XX"
	_, err = Decrypt(corrupt, testKeyB64)
	if err == nil {
		t.Fatal("expected error for corrupt ciphertext")
	}
}

func TestEncryptDifferentPlaintexts(t *testing.T) {
	plaintexts := []string{"", "a", "router-password-!@#$%^&*()", "mikrotik-admin-pass-2026"}

	for _, pt := range plaintexts {
		encrypted, err := Encrypt(pt, testKeyB64)
		if err != nil {
			t.Errorf("Encrypt(%q) failed: %v", pt, err)
			continue
		}

		decrypted, err := Decrypt(encrypted, testKeyB64)
		if err != nil {
			t.Errorf("Decrypt failed for %q: %v", pt, err)
			continue
		}

		if decrypted != pt {
			t.Errorf("mismatch for %q: got %q", pt, decrypted)
		}
	}
}

func TestDecryptLegacyPHPLongerInput(t *testing.T) {
	plaintext := "This-is-a-complex-router-passw0rd!@#"
	xored := make([]byte, len(plaintext))
	for i, b := range []byte(plaintext) {
		xored[i] = b ^ 128
	}
	encrypted := base64.StdEncoding.EncodeToString(xored)

	result, err := DecryptLegacyPHPConfig(encrypted)
	if err != nil {
		t.Fatalf("DecryptLegacyPHPConfig failed: %v", err)
	}
	if result != plaintext {
		t.Errorf("got %q, want %q", result, plaintext)
	}
}

func TestDecodeBlahLongerInput(t *testing.T) {
	plaintext := "mikhmon-admin-2026"
	encoded := string(encodeBlah(plaintext))

	result, err := DecodeBlah(encoded)
	if err != nil {
		t.Fatalf("DecodeBlah failed: %v", err)
	}
	if result != plaintext {
		t.Errorf("got %q, want %q", result, plaintext)
	}
}
