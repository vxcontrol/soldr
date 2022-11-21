package random

import (
	"crypto/rand"
	"fmt"
)

type Config struct{}

type Provider struct{}

func NewProvider(c *Config) (*Provider, error) {
	if c == nil {
		return nil, fmt.Errorf("passed configuration object is nil")
	}
	return &Provider{}, nil
}

func (p *Provider) GetNonce() ([]byte, error) {
	const nonceSize = 24
	nonce := make([]byte, nonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate a random nonce: %w", err)
	}
	return nonce, nil
}
