package encryptor

import (
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"hash"

	"golang.org/x/crypto/salsa20"
)

type Encryptor interface {
	Encrypt(pt []byte, key []byte) ([]byte, error)
	Decrypt(t []byte, key []byte) ([]byte, error)
}

type encryptor struct {
	hasher hash.Hash
}

func New() (Encryptor, error) {
	return &encryptor{
		hasher: sha256.New(),
	}, nil
}

const nonceLen = 24

func (e *encryptor) Encrypt(pt []byte, key []byte) ([]byte, error) {
	ptLen := len(pt)
	ct := make([]byte, ptLen)
	nonce := make([]byte, nonceLen, nonceLen+ptLen)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate an encryption nonce: %w", err)
	}
	salsa20.XORKeyStream(ct, pt, nonce, e.getKey(key))
	ct = append(nonce, ct...)
	return ct, nil
}

func (e *encryptor) Decrypt(ct []byte, key []byte) ([]byte, error) {
	nonce := ct[:nonceLen]
	ct = ct[nonceLen:]
	pt := make([]byte, len(ct))
	salsa20.XORKeyStream(pt, ct, nonce, e.getKey(key))
	return pt, nil
}

func (e *encryptor) getKey(key []byte) *[32]byte {
	k := sha256.Sum256(key)
	return &k
}
