package challenger

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"
)

func AESEncrypt(key AESKey, pt []byte) ([]byte, error) {
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

type Challenger struct{}

func NewChallenger() *Challenger {
	return &Challenger{}
}

func (c *Challenger) GetConnectionChallenge() ([]byte, error) {
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate a random nonce: %w", err)
	}
	return nonce, nil
}

func (c *Challenger) CheckConnectionChallenge(
	challengeCT []byte,
	expectedChallenge []byte,
	agentID string,
	abhs [][]byte,
) error {
	actualChallenges := make([][]byte, 0, len(abhs))
	for _, abh := range abhs {
		key := GetChallengeKey(agentID, abh)
		actualChallenge, err := aesDescrypt(key, challengeCT)
		if err != nil {
			return fmt.Errorf("failed to decrypt the received challenge response: %w", err)
		}
		actualChallenges = append(actualChallenges, actualChallenge)
	}
	for _, actualChallenge := range actualChallenges {
		if bytes.Equal(actualChallenge, expectedChallenge) {
			return nil
		}
	}
	return fmt.Errorf("an unexpected challenge response received")

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

func aesDescrypt(key AESKey, ct []byte) ([]byte, error) {
	block, err := aesGetCipherBlock(key)
	if err != nil {
		return nil, err
	}
	if len(ct) <= aes.BlockSize {
		return nil, fmt.Errorf("passed CT is too short")
	}
	pt := make([]byte, len(ct)-aes.BlockSize)
	iv := ct[:aes.BlockSize]
	cipher.NewCFBDecrypter(block, iv).
		XORKeyStream(pt, ct[aes.BlockSize:])
	return pt, nil
}
