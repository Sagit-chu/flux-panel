package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

type AESCrypto struct {
	key []byte
}

func NewAESCrypto(secret string) (*AESCrypto, error) {
	if secret == "" {
		return nil, fmt.Errorf("secret is empty")
	}
	hash := sha256.Sum256([]byte(secret))
	return &AESCrypto{key: hash[:]}, nil
}

func (a *AESCrypto) Encrypt(plain []byte) (string, error) {
	if len(plain) == 0 {
		return "", fmt.Errorf("empty plaintext")
	}
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	sealed := gcm.Seal(nil, nonce, plain, nil)
	data := append(nonce, sealed...)
	return base64.StdEncoding.EncodeToString(data), nil
}

func (a *AESCrypto) Decrypt(cipherText string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(cipherText)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(a.key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce := raw[:nonceSize]
	data := raw[nonceSize:]
	return gcm.Open(nil, nonce, data, nil)
}
