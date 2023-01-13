package crypto

import (
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/utils/dbencryptor"
	vxcommon_crypto "soldr/pkg/crypto"
)

// DBEncryptKey is set with go build
var DBEncryptKey string

const keySize = 32

type ConfigDecryptor struct {
	decryptor vxcommon_crypto.IDBConfigEncryptor
}

func NewConfigDecryptor() *ConfigDecryptor {
	return &ConfigDecryptor{
		decryptor: dbencryptor.NewSecureConfigEncryptor(getKey),
	}
}

func (c *ConfigDecryptor) Decrypt(cfg string) (string, error) {
	var moduleConfig models.ModuleSecureConfig

	err := json.Unmarshal([]byte(cfg), &moduleConfig)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal config: %w", err)
	}

	err = models.DecryptSecureConfig(c.decryptor, moduleConfig)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt config: %w", err)
	}

	result, err := json.Marshal(moduleConfig)
	if err != nil {
		return "", fmt.Errorf("failed to marshal decrypted config: %w", err)
	}

	return string(result), nil
}

func getKey() ([]byte, error) {
	if DBEncryptKey == "" {
		return nil, fmt.Errorf("key not set")
	}

	keyBytes, err := base64.StdEncoding.DecodeString(DBEncryptKey)
	if err != nil {
		return nil, fmt.Errorf("error decoding key: %w", err)
	}
	if len(keyBytes) != keySize {
		return nil, fmt.Errorf("expected key length to be %d, got %d", keySize, len(keyBytes))
	}

	return keyBytes, nil
}
