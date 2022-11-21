package crypto

import (
	"encoding/base64"
	"fmt"
	"strings"
)

type IDBConfigEncryptor interface {
	EncryptValue([]byte) (string, error)
	DecryptValue(string) ([]byte, error)
	IsFormatMatch(string) bool
}

type DBConfigEncryptor struct {
	encryptor IEncryptor
	prefix    string
}

func NewDBConfigEncryptor(encryptor IEncryptor, prefix string) *DBConfigEncryptor {
	if prefix != "" {
		prefix = prefix + "."
	}
	return &DBConfigEncryptor{
		encryptor: encryptor,
		prefix:    prefix,
	}
}

func (dbe *DBConfigEncryptor) EncryptValue(b []byte) (string, error) {
	encrypted, err := dbe.encryptor.Encrypt(b)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt data: %w", err)
	}

	b64 := base64.StdEncoding.EncodeToString(encrypted)
	value := fmt.Sprintf("%s%s", dbe.prefix, b64)

	return value, nil
}

func (dbe *DBConfigEncryptor) DecryptValue(s string) ([]byte, error) {
	ciphered, err := dbe.getCiphered(s)
	if err != nil {
		return nil, err
	}

	decrypted, err := dbe.encryptor.Decrypt(ciphered)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return decrypted, nil
}

func (dbe *DBConfigEncryptor) IsFormatMatch(s string) bool {
	if _, err := dbe.getCiphered(s); err != nil {
		return false
	}

	return true
}

func (dbe *DBConfigEncryptor) getCiphered(s string) ([]byte, error) {
	if dbe.prefix != "" && !strings.HasPrefix(s, dbe.prefix) {
		return nil, fmt.Errorf("invalid value format")
	}

	ciphered, err := base64.StdEncoding.DecodeString(s[len(dbe.prefix):])
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64 string: %w", err)
	}

	return ciphered, nil
}
