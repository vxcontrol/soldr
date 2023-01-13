package vm

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

type challengeCalculator struct {
	agentID string
	ABHCalculator
}

func NewChallengeCalculator(a ABHCalculator, agentID string) (*challengeCalculator, error) {
	return &challengeCalculator{
		agentID:       agentID,
		ABHCalculator: a,
	}, nil
}

func (c *challengeCalculator) PrepareChallengeResponse(nonce []byte) ([]byte, error) {
	abh, err := c.ABHCalculator.GetABH()
	if err != nil {
		return nil, fmt.Errorf("failed to get ABH for challenge response: %w", err)
	}
	key := GetChallengeKey(c.agentID, abh)
	resp, err := aesEncrypt(key, nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt the received challenge response: %w", err)
	}

	return resp, nil
}

type AESKey []byte

func GetChallengeKey(agentID string, abh []byte) AESKey {
	agentIDData := []byte(agentID)
	key := make([]byte, 0, len(agentIDData)+len(abh))
	key = append(append(key, agentIDData...), abh...)
	return getAESKey(key)
}

func getAESKey(k []byte) AESKey {
	keyHash := sha256.Sum256(k)
	return keyHash[:]
}

func aesGetCipherBlock(key AESKey) (cipher.Block, error) {
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, fmt.Errorf("bad key length (%d)", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new cipher block: %w", err)
	}
	return block, nil
}

func aesEncrypt(key AESKey, pt []byte) ([]byte, error) {
	block, err := aesGetCipherBlock(key)
	if err != nil {
		return nil, err
	}
	ct := make([]byte, aes.BlockSize+len(pt))
	iv := ct[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to initialize an IV: %w", err)
	}
	cipher.NewCFBEncrypter(block, iv).
		XORKeyStream(ct[aes.BlockSize:], pt)
	return ct, nil
}
