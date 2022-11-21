package sk

import (
	"crypto/ed25519"
	"fmt"

	"soldr/scripts/sbh_generator/sk/static"
)

type Provider interface {
	GetSecretKey() (ed25519.PrivateKey, error)
}

type Config struct {
	Static *static.Config
}

func NewProvider(c *Config) (Provider, error) {
	if c.Static != nil {
		p, err := static.NewProvider(c.Static)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize a static SK provider: %w", err)
		}
		return p, nil
	}
	return nil, fmt.Errorf("no appropriate configuration passed")
}
