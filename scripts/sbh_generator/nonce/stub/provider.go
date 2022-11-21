package stub

import "fmt"

type Config struct {
	Nonce []byte
}

type Provider struct {
	nonce []byte
}

func NewProvider(c *Config) (*Provider, error) {
	if c == nil {
		return nil, fmt.Errorf("passed configuration object is nil")
	}
	if c.Nonce == nil {
		return nil, fmt.Errorf("passed nonce is nil")
	}
	p := &Provider{}
	var err error
	p.nonce, err = copyBytes(c.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to copy the passed nonce: %w", err)
	}
	return p, nil
}

func (p *Provider) GetNonce() ([]byte, error) {
	nonce, err := copyBytes(p.nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to copy the provider's nonce: %w", err)
	}
	return nonce, nil
}

func copyBytes(data []byte) ([]byte, error) {
	dataCopy := make([]byte, len(data))
	if n := copy(dataCopy, data); n != len(data) {
		return nil, fmt.Errorf("expected to copy %d bytes of the nonce, actually copied %d", len(data), n)
	}
	return dataCopy, nil
}
