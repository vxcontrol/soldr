package crypto

import (
	"crypto/rand"
	"crypto/rc4"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const nonceLen = 24

func GenerateNonce() (string, error) {
	nonce := make([]byte, nonceLen)
	if err := generateNonce(nonce); err != nil {
		return "", err
	}
	return hex.EncodeToString(nonce), nil
}

func generateNonce(dst []byte) error {
	if _, err := rand.Read(dst); err != nil {
		return fmt.Errorf("failed to generate random data: %w", err)
	}
	return nil
}

func Encrypt(data, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("key is empty")
	}

	result := make([]byte, len(data)+nonceLen)
	nonce := result[:nonceLen]
	if err := generateNonce(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate a nonce: %w", err)
	}

	messageKey, err := getMessageKey(key, nonce)
	if err != nil {
		return nil, err
	}

	if err = xorKeyStream(data, messageKey, result[nonceLen:]); err != nil {
		return nil, fmt.Errorf("failed to encrypt data: %w", err)
	}

	return result, nil
}

func Decrypt(ct, key []byte) ([]byte, error) {
	switch {
	case len(ct) < nonceLen:
		return nil, fmt.Errorf("passed cipher text is malformed")
	case len(key) == 0:
		return nil, fmt.Errorf("key is empty")
	}

	nonce := ct[:nonceLen]
	messageKey, err := getMessageKey(key, nonce)
	if err != nil {
		return nil, err
	}

	result := make([]byte, len(ct)-nonceLen)
	if err = xorKeyStream(ct[nonceLen:], messageKey, result); err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return result, nil
}

func xorKeyStream(data, messageKey, result []byte) error {
	c, err := rc4.NewCipher(messageKey)
	if err != nil {
		return fmt.Errorf("failed to create a new cipher: %w", err)
	}
	c.XORKeyStream(result, data)

	return nil
}

func getMessageKey(key, nonce []byte) ([]byte, error) {
	getErr := func(err error) error {
		return fmt.Errorf("failed to get message key: %w", err)
	}

	messageKey := make([]byte, len(key)+len(nonce))
	if n := copy(messageKey, key); n != len(key) {
		return nil, getErr(fmt.Errorf("failed to copy key, expected %d, got %d", len(key), n))
	}
	if n := copy(messageKey[len(key):], nonce); n != len(nonce) {
		return nil, getErr(fmt.Errorf("failed to copy nonce, expected %d, got %d", len(key), n))
	}
	messageKeyHash := sha256.Sum256(messageKey)

	return messageKeyHash[:], nil
}
