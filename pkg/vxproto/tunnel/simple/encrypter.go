package simple

import (
	"fmt"
	"sync"

	"soldr/pkg/protoagent"
	compressor "soldr/pkg/vxproto/tunnel/compressor/simple"
)

type Config struct {
	Key byte
}

type Encryptor struct {
	key    byte
	keyMux *sync.RWMutex

	compressor *compressor.Compressor
}

func New(c *Config) *Encryptor {
	return &Encryptor{
		key:    c.Key,
		keyMux: &sync.RWMutex{},

		compressor: compressor.NewCompressor(),
	}
}

func (e *Encryptor) Encrypt(data []byte) ([]byte, error) {
	compressedData, err := e.compressor.Compress(data)
	if err != nil {
		return nil, err
	}
	return xor(compressedData, e.getKey()), err
}

func (e *Encryptor) Decrypt(data []byte) ([]byte, error) {
	compressedData := xor(data, e.getKey())
	data, err := e.compressor.Decompress(compressedData)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress the data: %w", err)
	}
	return data, nil
}

func (e *Encryptor) Reset(config *protoagent.TunnelConfig) error {
	c := config.GetSimple()
	if c == nil {
		return fmt.Errorf("passed config is not of the type *TunnelConfig_Simple")
	}
	e.reset(&resetConfig{
		Key: byte(*c.Key),
	})
	return nil
}

func xor(src []byte, key byte) []byte {
	result := make([]byte, len(src))
	for i, d := range src {
		result[i] = d ^ key
	}
	return result
}

func (e *Encryptor) getKey() byte {
	e.keyMux.RLock()
	defer e.keyMux.RUnlock()
	return e.key
}

type resetConfig struct {
	Key byte
}

func (e *Encryptor) reset(config *resetConfig) {
	e.keyMux.Lock()
	defer e.keyMux.Unlock()

	e.key = config.Key
}
