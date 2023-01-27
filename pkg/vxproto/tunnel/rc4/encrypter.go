package rc4

import (
	"crypto/rc4"
	"fmt"
	"sync"

	"soldr/pkg/app/agent"
	compressor "soldr/pkg/vxproto/tunnel/compressor/simple"
)

func GenerateKey(rand func(buf []byte) error) ([]byte, error) {
	const keyLen = 48
	buf := make([]byte, keyLen)
	if err := rand(buf); err != nil {
		return nil, fmt.Errorf("calling rand failed: %w", err)
	}
	return buf, nil
}

type Config struct {
	Key []byte
}

type Encrypter struct {
	key    []byte
	keyMux *sync.RWMutex

	compressor *compressor.Compressor
}

func New(c *Config) *Encrypter {
	return &Encrypter{
		key:        c.Key,
		keyMux:     &sync.RWMutex{},
		compressor: compressor.NewCompressor(),
	}
}

func (e *Encrypter) Encrypt(data []byte) ([]byte, error) {
	compressedData, err := e.compressor.Compress(data)
	if err != nil {
		return nil, err
	}
	ct, err := e.applyCipher(compressedData)
	if err != nil {
		return nil, err
	}
	return ct, nil
}

func (e *Encrypter) Decrypt(data []byte) ([]byte, error) {
	compressedData, err := e.applyCipher(data)
	if err != nil {
		return nil, err
	}
	data, err = e.compressor.Decompress(compressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress the data: %w", err)
	}
	return data, nil
}

func (e *Encrypter) applyCipher(data []byte) ([]byte, error) {
	xoredData := make([]byte, len(data))

	e.keyMux.RLock()
	defer e.keyMux.RUnlock()
	cipher, err := rc4.NewCipher(e.key)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a new RC4 cipher: %w", err)
	}

	cipher.XORKeyStream(xoredData, data)
	return xoredData, err
}

func (e *Encrypter) Reset(config *agent.TunnelConfig) error {
	c := config.GetLua()
	if c == nil {
		return fmt.Errorf("passed config is not of the type *TunnelConfig_TunnelConfigSimple")
	}
	e.reset(&resetConfig{
		Key: c.Key,
	})
	return nil
}

type resetConfig struct {
	Key []byte
}

func (e *Encrypter) reset(config *resetConfig) {
	e.keyMux.Lock()
	defer e.keyMux.Unlock()

	e.key = config.Key
}
