package tunnel

import (
	"fmt"

	"soldr/pkg/protoagent"
	tunnelRC4 "soldr/pkg/vxproto/tunnel/rc4"
	tunnelSimple "soldr/pkg/vxproto/tunnel/simple"
)

type PackEncryptor interface {
	Encrypt(data []byte) ([]byte, error)
	Decrypt(data []byte) ([]byte, error)
	Reset(config *protoagent.TunnelConfig) error
}

type Config struct {
	RC4    *tunnelRC4.Config
	Simple *tunnelSimple.Config
}

func NewPackEncrypter(c *Config) (PackEncryptor, error) {
	if c.RC4 != nil {
		return tunnelRC4.New(c.RC4), nil
	}
	if c.Simple != nil {
		return tunnelSimple.New(c.Simple), nil
	}
	return nil, fmt.Errorf("no appropriate configuration found to initialize the pack encryptor")
}
