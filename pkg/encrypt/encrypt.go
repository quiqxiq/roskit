package encrypt

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

func Encrypt(plaintext, keyBase64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", fmt.Errorf("encrypt: decode key: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("encrypt: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("encrypt: new gcm: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("encrypt: generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func Decrypt(ciphertext, keyBase64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", fmt.Errorf("decrypt: decode key: %w", err)
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("decrypt: decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("decrypt: new cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("decrypt: new gcm: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("decrypt: ciphertext too short")
	}

	nonce, ct := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", fmt.Errorf("decrypt: open: %w", err)
	}

	return string(plain), nil
}

func DecryptLegacyPHPConfig(encrypted string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("decrypt legacy: decode: %w", err)
	}

	result := make([]byte, len(data))
	for i, b := range data {
		result[i] = b ^ 128
	}

	return string(result), nil
}

func DecodeBlah(encoded string) (string, error) {
	xored := make([]byte, len(encoded))
	for i, b := range []byte(encoded) {
		xored[i] = b ^ 10
	}

	stage1, err := base64.StdEncoding.DecodeString(string(xored))
	if err != nil {
		return "", fmt.Errorf("decode blah: stage1 decode: %w", err)
	}

	stage2, err := base64.StdEncoding.DecodeString(string(stage1))
	if err != nil {
		return "", fmt.Errorf("decode blah: stage2 decode: %w", err)
	}

	return string(stage2), nil
}
