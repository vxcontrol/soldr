package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"io"
)

type IEncryptor interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

type AESEncryptor struct {
	key []byte
}

type KeyGetter func() ([]byte, error)

func NewAESEncryptor(f KeyGetter) (*AESEncryptor, error) {
	key, err := f()
	if err != nil {
		return nil, fmt.Errorf("failed to get the key: %w", err)
	}

	return &AESEncryptor{
		key: key,
	}, nil
}

func (e *AESEncryptor) Encrypt(data []byte) ([]byte, error) {
	c, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	seal := gcm.Seal(nonce, nonce, data, nil)

	return seal, err
}

func (e *AESEncryptor) Decrypt(data []byte) ([]byte, error) {
	c, err := aes.NewCipher(e.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, err
	}

	nonce, data := data[:nonceSize], data[nonceSize:]
	result, err := gcm.Open(nil, nonce, data, nil)
	if err != nil {
		return nil, err
	}

	return result, nil
}
