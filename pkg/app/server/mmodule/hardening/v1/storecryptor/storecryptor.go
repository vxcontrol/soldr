package storecryptor

import (
	"crypto/sha256"
	"encoding/hex"

	"soldr/pkg/app/server/mmodule/hardening/v1/crypto"
)

type StoreCryptor struct {
	key string
}

func NewStoreCryptor(agentID string) *StoreCryptor {
	return &StoreCryptor{
		key: GetKeyByAgentID(agentID),
	}
}

func (s *StoreCryptor) EncryptData(data []byte) ([]byte, error) {
	ct, err := crypto.Encrypt(data, []byte(s.key))
	if err != nil {
		return nil, err
	}
	return ct, nil
}

func (s *StoreCryptor) DecryptData(data []byte) ([]byte, error) {
	panic("not implemented")
}

func GetKeyByAgentID(agentID string) string {
	const salt = "59c65a5d35eb14bed3f47d55b431a3aaaae3b48154a6"
	agentID += salt
	hash := sha256.Sum256([]byte(agentID))
	key := hex.EncodeToString(hash[:])

	return key
}
