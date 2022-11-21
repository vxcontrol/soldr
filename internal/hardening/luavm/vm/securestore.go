package vm

import (
	"crypto/rc4"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

const nonceLen = 24

type SecureConfigEncryptor struct {
	key string
}

func NewSecureConfigEncryptor(agentID string) (*SecureConfigEncryptor, error) {
	return &SecureConfigEncryptor{
		key: GetKeyByAgentID(agentID),
	}, nil
}

func (s *SecureConfigEncryptor) DecryptData(data []byte) ([]byte, error) {
	pt, err := Decrypt(data, []byte(s.key))
	if err != nil {
		return nil, err
	}
	return pt, nil
}

func (s *SecureConfigEncryptor) IsStoreKeyEmpty() (bool, error) {
	return false, nil
}

func GetKeyByAgentID(agentID string) string {
	const salt = "59c65a5d35eb14bed3f47d55b431a3aaaae3b48154a6"
	agentID += salt
	hash := sha256.Sum256([]byte(agentID))
	key := hex.EncodeToString(hash[:])

	return key
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
