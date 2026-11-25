package secret

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"errors"
	"fmt"
)

type Sealed struct {
	KeyID       string
	Nonce       []byte
	Ciphertext  []byte
	ContentHash []byte
}

type Cipher struct {
	aead  cipher.AEAD
	keyID string
}

func NewCipher(key string, keyID string) (*Cipher, error) {
	if key == "" || keyID == "" {
		return nil, errors.New("secret key and key id are required")
	}

	keyBytes := sha256.Sum256([]byte(key))
	block, err := aes.NewCipher(keyBytes[:])
	if err != nil {
		return nil, fmt.Errorf("create AES cipher: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("create AES-GCM: %w", err)
	}
	return &Cipher{aead: aead, keyID: keyID}, nil
}

func (c *Cipher) KeyID() string {
	return c.keyID
}

func (c *Cipher) Seal(plaintext []byte, aad string) (Sealed, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return Sealed{}, fmt.Errorf("generate nonce: %w", err)
	}

	ciphertext := c.aead.Seal(nil, nonce, plaintext, []byte(aad))
	digest := sha256.Sum256(plaintext)
	return Sealed{
		KeyID:       c.keyID,
		Nonce:       nonce,
		Ciphertext:  ciphertext,
		ContentHash: digest[:],
	}, nil
}

func (c *Cipher) Open(sealed Sealed, aad string) ([]byte, error) {
	if sealed.KeyID != c.keyID {
		return nil, fmt.Errorf("secret key id %q is not available", sealed.KeyID)
	}
	plaintext, err := c.aead.Open(nil, sealed.Nonce, sealed.Ciphertext, []byte(aad))
	if err != nil {
		return nil, fmt.Errorf("decrypt secret payload: %w", err)
	}
	digest := sha256.Sum256(plaintext)
	if !equalBytes(digest[:], sealed.ContentHash) {
		return nil, errors.New("secret payload content hash mismatch")
	}
	return plaintext, nil
}

func equalBytes(left []byte, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var result byte
	for index := range left {
		result |= left[index] ^ right[index]
	}
	return result == 0
}
