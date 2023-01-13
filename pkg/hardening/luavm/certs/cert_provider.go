package certs

import (
	"crypto/tls"

	"soldr/pkg/hardening/luavm/certs/provider"
)

type CertProvider interface {
	VXCA() ([]byte, error)
	IAC() (*tls.Certificate, error)
}

func NewCertProvider() (CertProvider, error) {
	return provider.NewStatic()
}
