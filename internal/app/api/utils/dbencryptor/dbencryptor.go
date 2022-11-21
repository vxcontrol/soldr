package dbencryptor

import (
	_ "embed"
	"encoding/base64"
	"fmt"

	"soldr/internal/crypto"
)

var (
	//go:embed sec-store-key.txt
	keyFile []byte

	// DBEncryptKey is set with go build
	DBEncryptKey string
)

const (
	keySize              = 32
	EncryptedValuePrefix = "soldr"
)

func NewSecureConfigEncryptor(key crypto.KeyGetter) *crypto.DBConfigEncryptor {
	encryptor, err := crypto.NewAESEncryptor(key)
	if err != nil {
		panic("error creating AESEncryptor: " + err.Error())
	}

	dbencryptor := crypto.NewDBConfigEncryptor(encryptor, EncryptedValuePrefix)

	return dbencryptor
}

func GetKey() ([]byte, error) {
	var key string
	if DBEncryptKey != "" {
		key = DBEncryptKey
	} else {
		key = string(keyFile)
	}

	keyBytes, err := base64.StdEncoding.DecodeString(key)
	if err != nil {
		return nil, fmt.Errorf("error decoding key")
	}
	if len(keyBytes) != keySize {
		return nil, fmt.Errorf("wrong key len")
	}

	return keyBytes, nil
}
