package common

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"io"
)

func createHash(key string) (string, error) {
	hasher := md5.New()
	_, err := hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil)), err
}

// Encrypt the input data with passphrase
func Encrypt(data []byte, passphrase string) ([]byte, error) {
	hash, err := createHash(passphrase)
	if err != nil {
		return nil, err
	}

	block, _ := aes.NewCipher([]byte(hash))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

// Decrypt the input data with passphrase
func Decrypt(data []byte, passphrase string) ([]byte, error) {
	hash, err := createHash(passphrase)
	if err != nil {
		return nil, err
	}

	key := []byte(hash)
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}
