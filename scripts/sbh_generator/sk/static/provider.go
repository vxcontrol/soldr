package static

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

type Config struct {
	VXCAKeyFile string
}

type Provider struct {
	sk ed25519.PrivateKey
}

func NewProvider(c *Config) (*Provider, error) {
	if c == nil {
		return nil, fmt.Errorf("passed configuration is nil")
	}
	contents, err := os.ReadFile(c.VXCAKeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read the VXCA key file %s: %w", c.VXCAKeyFile, err)
	}
	sk, err := extractSK(contents)
	if err != nil {
		return nil, fmt.Errorf("failed to extract the secret key: %w", err)
	}
	return &Provider{
		sk: sk,
	}, nil
}

func extractSK(data []byte) (ed25519.PrivateKey, error) {
	key, err := pemToKey(data)
	if err != nil {
		return nil, fmt.Errorf("failed to extract a key from the passed data: %w", err)
	}
	sk, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("passed data does not represent an ed25519.PrivateKey")
	}
	return sk, nil
}

func pemToKey(keyData []byte) (interface{}, error) {
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode the PEM-encoded key file: no PEM block found")
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the PEM-encoded private key: %w", err)
	}
	return key, nil
}

func (p *Provider) GetSecretKey() (ed25519.PrivateKey, error) {
	skCopy := make([]byte, len(p.sk))
	copy(skCopy, p.sk)
	return skCopy, nil
}
