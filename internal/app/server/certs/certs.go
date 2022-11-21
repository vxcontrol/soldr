package certs

import (
	"crypto/tls"
	"fmt"

	"soldr/internal/app/server/certs/config"
	"soldr/internal/app/server/certs/provider/static"
)

type Provider interface {
	VXCA() ([]byte, error)
	SC() ([]tls.Certificate, error)
	CreateLTACFromCSR(tlsConnState *tls.ConnectionState, csr []byte) ([]byte, error)
}

func NewProvider(c *config.Config) (Provider, error) {
	if c.StaticProvider != nil {
		return static.NewProvider(c.StaticProvider)
	}
	return nil, fmt.Errorf("no suitable configuration found to initialize the certificate provider")
}
