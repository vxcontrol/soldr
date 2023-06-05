package validator

import (
	"context"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/pkg/app/server/certs"
	"soldr/pkg/app/server/mmodule/hardening/v1/abher"
	"soldr/pkg/app/server/mmodule/hardening/v1/abher/types"
	"soldr/pkg/app/server/mmodule/hardening/v1/approver"
	"soldr/pkg/app/server/mmodule/hardening/v1/challenger"
	"soldr/pkg/app/server/mmodule/hardening/v1/sbher"
	"soldr/pkg/app/server/mmodule/hardening/v1/ssa"
	tunnelConfigurer "soldr/pkg/app/server/mmodule/hardening/v1/tunnel"
	"soldr/pkg/app/server/mmodule/hardening/v1/validator/ainfo"
	"soldr/pkg/filestorage"
	"soldr/pkg/protoagent"
	"soldr/pkg/vxproto"
	"soldr/pkg/vxproto/tunnel"
)

type ABHer interface {
	GetABH(t vxproto.AgentType, id *types.AgentBinaryID) ([][]byte, error)
	GetABHWithSocket(t vxproto.AgentType, socket vxproto.IAgentSocket) ([][]byte, error)
}

type SBHer interface {
	Get(ctx context.Context, version string) ([]byte, error)
}

type SSAGenerator interface {
	GenerateSSAScript(scriptEncodingKey []byte) ([]byte, error)
}

type Challenger interface {
	GetConnectionChallenge() ([]byte, error)
	CheckConnectionChallenge(challengeCT []byte, expectedChallenge []byte, agentID string, abh [][]byte) error
}

type TunnelConfigurer interface {
	GetTunnelConfig() (*tunnel.Config, *protoagent.TunnelConfig, error)
}

type ConnectionValidator struct {
	vxproto.AgentIDFetcher
	abher            ABHer
	sbher            SBHer
	approver         approver.Approver
	ssaGenerator     SSAGenerator
	certsProvider    certs.Provider
	challenger       Challenger
	tunnelConfigurer TunnelConfigurer
	gdbc             *gorm.DB

	ctx context.Context
}

func NewConnectionValidator(
	ctx context.Context,
	gdbc *gorm.DB,
	fs filestorage.Reader,
	store interface{},
	basePath string,
	certsProvider certs.Provider,
	approver approver.Approver,
	logger *logrus.Entry,
) (*ConnectionValidator, error) {
	abher, err := abher.NewABH(ctx, store, basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize an ABHer: %w", err)
	}
	sbher, err := sbher.NewSBH(ctx, fs, basePath, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize an SBHer: %w", err)
	}
	ssaGenerator := ssa.NewNOPSSAGenerator()
	agentInfoFetcher, err := ainfo.NewAgentInfoFetcher(gdbc)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize an agent info fetcher: %w", err)
	}
	return NewConnectionValidatorWithComponents(
		ctx,
		gdbc,
		certsProvider,
		abher,
		sbher,
		ssaGenerator,
		challenger.NewChallenger(),
		tunnelConfigurer.NewConfigurer(),
		approver,
		agentInfoFetcher,
	), nil
}

func NewConnectionValidatorWithComponents(
	ctx context.Context,
	gdbc *gorm.DB,
	certsProvider certs.Provider,
	abher ABHer,
	sbher SBHer,
	ssaGenerator SSAGenerator,
	challenger Challenger,
	tunnelConfigurer TunnelConfigurer,
	approver approver.Approver,
	agentInfoFetcher vxproto.AgentIDFetcher,
) *ConnectionValidator {
	return &ConnectionValidator{
		AgentIDFetcher:   agentInfoFetcher,
		certsProvider:    certsProvider,
		ctx:              ctx,
		gdbc:             gdbc,
		abher:            abher,
		sbher:            sbher,
		ssaGenerator:     ssaGenerator,
		approver:         approver,
		challenger:       challenger,
		tunnelConfigurer: tunnelConfigurer,
	}
}
