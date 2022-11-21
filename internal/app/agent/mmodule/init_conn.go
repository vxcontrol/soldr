package mmodule

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"soldr/internal/vxproto"
)

type initConnectConfig struct {
	Host string
	Type string
}

func (mm *MainModule) initConnect(ctx context.Context, config *initConnectConfig) error {
	if config == nil {
		return fmt.Errorf("passed config is nil")
	}
	clientConfig, err := mm.getInitConnectionConfigForVXProto(config)
	if err != nil {
		return err
	}
	if err := mm.proto.InitConnection(
		ctx,
		mm.connValidator,
		clientConfig,
		logrus.WithField("step", "connection initialization"),
	); err != nil {
		return fmt.Errorf("connection initialization failed: %w", err)
	}
	logrus.Debug("connection initialized!")
	return nil
}

func (mm *MainModule) getInitConnectionConfigForVXProto(c *initConnectConfig) (*vxproto.ClientInitConfig, error) {
	tlsConfig, err := mm.tlsConfigurer.GetTLSConfigForInitConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to configure TLS connection: %w", err)
	}
	return &vxproto.ClientInitConfig{
		CommonConfig: &vxproto.CommonConfig{
			Host:      c.Host,
			TLSConfig: tlsConfig,
		},
		Type:            c.Type,
		ProtocolVersion: protocolVersion,
	}, nil
}
