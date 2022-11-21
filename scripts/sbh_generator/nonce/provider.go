package nonce

import (
	"fmt"

	"soldr/scripts/sbh_generator/nonce/random"
	"soldr/scripts/sbh_generator/nonce/stub"
)

type Provider interface {
	GetNonce() ([]byte, error)
}

type Config struct {
	Random *random.Config
	Stub   *stub.Config
}

func NewProvider(c *Config) (Provider, error) {
	if c == nil {
		return nil, fmt.Errorf("passed configuration object is nil")
	}
	if c.Stub != nil {
		p, err := stub.NewProvider(c.Stub)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize a stub provider: %w", err)
		}
		return p, nil
	}
	if c.Random != nil {
		p, err := random.NewProvider(c.Random)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize a random provider: %w", err)
		}
		return p, nil
	}
	return nil, fmt.Errorf("no appropriate configuration to initialize a provider passed")
}
