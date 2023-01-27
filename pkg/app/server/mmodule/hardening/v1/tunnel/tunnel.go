package tunnel

import (
	"crypto/rand"
	"fmt"

	"soldr/pkg/app/agent"
	"soldr/pkg/vxproto/tunnel"
	tunnelSimple "soldr/pkg/vxproto/tunnel/simple"
)

type Configurer struct{}

func NewConfigurer() *Configurer {
	return &Configurer{}
}

func (c *Configurer) GetTunnelConfig() (*tunnel.Config, *agent.TunnelConfig, error) {
	b := make([]byte, 1)
	if _, err := rand.Read(b); err != nil {
		return nil, nil, fmt.Errorf("failed to get a random byte: %w", err)
	}
	key := b[0]
	keyAgentConfig := uint32(key)
	return &tunnel.Config{
			Simple: &tunnelSimple.Config{
				Key: key,
			},
		}, &agent.TunnelConfig{
			Config: &agent.TunnelConfig_Simple{
				Simple: &agent.TunnelConfig_TunnelConfigSimple{
					Key: &keyAgentConfig,
				},
			},
		}, nil
}
