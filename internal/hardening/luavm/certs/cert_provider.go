package certs

import (
	"crypto/tls"

	"soldr/internal/hardening/luavm/certs/provider"
)

type CertProvider interface {
	VXCA() ([]byte, error)
	IAC() (*tls.Certificate, error)
}

func NewCertProvider() (CertProvider, error) {
	return provider.NewStatic()
}
