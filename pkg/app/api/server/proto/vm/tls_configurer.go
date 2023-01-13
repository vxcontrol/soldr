package vm

import (
	"crypto/tls"
	"fmt"

	"soldr/pkg/hardening/luavm/certs"
	"soldr/pkg/hardening/luavm/vm"
)

type tlsConfigurer struct {
	vm.SimpleTLSConfigurer
}

func newTLSConfigurer(certsProvider certs.CertProvider, ltacGetter vm.LTACGetter) *tlsConfigurer {
	return &tlsConfigurer{
		SimpleTLSConfigurer: *vm.NewSimpleTLSConfigurer(certsProvider, ltacGetter, nil),
	}
}

func (c *tlsConfigurer) GetTLSConfigForInitConnection() (*tls.Config, error) {
	return nil, fmt.Errorf("GetTLSConfigForInitConnection is not implemented")
}
