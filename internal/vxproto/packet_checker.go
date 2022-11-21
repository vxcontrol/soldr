package vxproto

import (
	"context"
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"

	"soldr/internal/protoagent"
)

func NewConnectionPolicyManagerIterator(versionsConfig ServerAPIVersionsConfig) (*ConnectionPolicyManagerIterator, error) {
	if versionsConfig == nil {
		return nil, fmt.Errorf("a nil configuration object passed")
	}
	if len(versionsConfig) == 0 {
		return nil, fmt.Errorf("no policies defined in the ConnectionPolicyManagerIterator configuration")
	}
	f := &ConnectionPolicyManagerIterator{
		versionPolicies: make([]*ConnectionPolicyManagerForVersion, 0, len(versionsConfig)),
	}
	managers := map[EndpointConnectionPolicy]ConnectionPolicyManager{}
	for ver, verConfig := range versionsConfig {
		var (
			m  ConnectionPolicyManager
			ok bool
		)
		switch p := verConfig.ConnectionPolicy; p {
		case EndpointConnectionPolicyAllow:
			if m, ok = managers[p]; !ok {
				m = newAllowConnectionPolicyManager(p)
				managers[p] = m
			}
		case EndpointConnectionPolicyBlock:
			if m, ok = managers[p]; !ok {
				m = newBlockConnectionPolicyManager(p)
				managers[p] = m
			}
		case EndpointConnectionPolicyUpgrade:
			if m, ok = managers[p]; !ok {
				m = newUpgradeConnectionPolicyManager(p)
				managers[p] = m
			}
		default:
			return nil, fmt.Errorf("an unknown policy %d found", verConfig.ConnectionPolicy)
		}
		f.versionPolicies = append(f.versionPolicies, &ConnectionPolicyManagerForVersion{
			Version: ver,
			Manager: m,
		})
	}
	return f, nil
}

type ConnectionPolicyManagerIterator struct {
	versionPolicies []*ConnectionPolicyManagerForVersion
	currPolicyIdx   int
}

type ConnectionPolicyManagerForVersion struct {
	Version string
	Manager ConnectionPolicyManager
}

func (f *ConnectionPolicyManagerIterator) Next() bool {
	return f.currPolicyIdx < len(f.versionPolicies)
}

func (f *ConnectionPolicyManagerIterator) GetCurrentManager() (*ConnectionPolicyManagerForVersion, error) {
	if f.currPolicyIdx >= len(f.versionPolicies) {
		return nil, fmt.Errorf("no more managers configured, rewind the iterator")
	}
	m := f.versionPolicies[f.currPolicyIdx]
	f.currPolicyIdx++
	return &ConnectionPolicyManagerForVersion{
		Version: m.Version,
		Manager: m.Manager,
	}, nil
}

func (f *ConnectionPolicyManagerIterator) Rewind() {
	f.currPolicyIdx = 0
}

var ErrEndpointBlocked = errors.New("the endpoint is blocked")

type ConnectionPolicyManager interface {
	ConnectionPolicyTypeGetter
	IsConnectionAllowed(agentType AgentType) bool
	GetConnectionPolicy(info *ConnectionInfo) (ConnectionPolicy, error)
}

type ConnectionPolicyTypeGetter interface {
	GetConnectionPolicyType() (EndpointConnectionPolicy, error)
}

type EndpointConnectionPolicy int

const (
	EndpointConnectionPolicyAllow EndpointConnectionPolicy = iota + 1
	EndpointConnectionPolicyUpgrade
	EndpointConnectionPolicyBlock
)

func (p *EndpointConnectionPolicy) FromString(policy string) error {
	switch policy {
	case EndpointConnectionPolicyAllow.String():
		*p = EndpointConnectionPolicyAllow
	case EndpointConnectionPolicyUpgrade.String():
		*p = EndpointConnectionPolicyUpgrade
	case EndpointConnectionPolicyBlock.String():
		*p = EndpointConnectionPolicyBlock
	default:
		return fmt.Errorf("unknown endpoint connection policy passed: \"%s\"", policy)
	}
	return nil
}

func (p EndpointConnectionPolicy) String() string {
	switch p {
	case EndpointConnectionPolicyAllow:
		return "allow"
	case EndpointConnectionPolicyBlock:
		return "block"
	case EndpointConnectionPolicyUpgrade:
		return "upgrade"
	default:
		return "unknown"
	}
}

type staticConnectionPolicyManager struct {
	policy     ConnectionPolicy
	policyType EndpointConnectionPolicy
}

func newStaticConnectionPolicyManager(policy ConnectionPolicy, policyType EndpointConnectionPolicy) *staticConnectionPolicyManager {
	return &staticConnectionPolicyManager{
		policy:     policy,
		policyType: policyType,
	}
}

func (m *staticConnectionPolicyManager) GetConnectionPolicy(info *ConnectionInfo) (ConnectionPolicy, error) {
	return m.policy, nil
}

func (m *staticConnectionPolicyManager) GetConnectionPolicyType() (EndpointConnectionPolicy, error) {
	return m.policyType, nil
}

type allowConnectionPolicyManager struct {
	*staticConnectionPolicyManager
}

func newAllowConnectionPolicyManager(policyType EndpointConnectionPolicy) *allowConnectionPolicyManager {
	return &allowConnectionPolicyManager{
		staticConnectionPolicyManager: newStaticConnectionPolicyManager(newAllowPacketChecker(), policyType),
	}
}

func (m *allowConnectionPolicyManager) IsConnectionAllowed(_ AgentType) bool {
	return true
}

type upgradeConnectionPolicyManager struct {
	*staticConnectionPolicyManager
}

func newUpgradeConnectionPolicyManager(policyType EndpointConnectionPolicy) *upgradeConnectionPolicyManager {
	return &upgradeConnectionPolicyManager{
		staticConnectionPolicyManager: newStaticConnectionPolicyManager(newUpgradePacketChecker(), policyType),
	}
}

func (m *upgradeConnectionPolicyManager) GetConnectionPolicy(info *ConnectionInfo) (ConnectionPolicy, error) {
	if !m.IsConnectionAllowed(info.AgentType) {
		return nil, ErrEndpointBlocked
	}
	return m.policy, nil
}

func (m *upgradeConnectionPolicyManager) IsConnectionAllowed(agentType AgentType) bool {
	return agentType == VXAgent
}

type blockConnectionPolicyManager struct {
	*staticConnectionPolicyManager
}

func newBlockConnectionPolicyManager(policyType EndpointConnectionPolicy) *blockConnectionPolicyManager {
	return &blockConnectionPolicyManager{
		staticConnectionPolicyManager: newStaticConnectionPolicyManager(newBlockPacketChecker(), policyType),
	}
}

func (m *blockConnectionPolicyManager) GetConnectionPolicy(_ *ConnectionInfo) (ConnectionPolicy, error) {
	return nil, ErrEndpointBlocked
}

func (m *blockConnectionPolicyManager) IsConnectionAllowed(_ AgentType) bool {
	return false
}

type ConnectionInfo struct {
	Agent     *AgentConnectionInfo
	AgentType AgentType
	URLPath   string
}

type ConnectionPolicy interface {
	InPacketChecker
	OutPacketChecker
}

type InPacketChecker interface {
	CheckInPacket(ctx context.Context, packet *Packet) error
}

type OutPacketChecker interface {
	CheckOutPacket(ctx context.Context, packet *Packet) error
}

type customPacketChecker struct {
	*policyInPacketChecker
	*policyOutPacketChecker
}

type allowPacketChecker customPacketChecker

func newAllowPacketChecker() *allowPacketChecker {
	return &allowPacketChecker{
		policyInPacketChecker:  newPolicyInPacketChecker(),
		policyOutPacketChecker: newPolicyOutPacketChecker(),
	}
}

type blockPacketChecker customPacketChecker

func newBlockPacketChecker() *blockPacketChecker {
	blockPolicy := func(_ context.Context, _ *Packet) error {
		return ErrEndpointBlocked
	}
	return &blockPacketChecker{
		policyInPacketChecker:  newPolicyInPacketChecker(blockPolicy),
		policyOutPacketChecker: newPolicyOutPacketChecker(blockPolicy),
	}
}

type upgradePacketChecker customPacketChecker

func newUpgradePacketChecker() *upgradePacketChecker {
	checkPacket := func(p *Packet, in bool) error {
		destMsg := "sending"
		if in {
			destMsg = "got"
		}
		if p.Module != mainModuleName {
			return fmt.Errorf(
				"%s packet with destination \"%s\", but only packets to module \"%s\" are allowed for this connection",
				destMsg,
				p.Module,
				mainModuleName,
			)
		}
		return nil
	}
	inCheck := func(ctx context.Context, p *Packet) error {
		if err := checkPacket(p, true); err != nil {
			return err
		}
		if p.PType != PTData {
			return fmt.Errorf(
				"got packet of type %d, but only data packets (of type %d) are allowed for this connection",
				p.PType,
				PTData,
			)
		}
		var message protoagent.Message
		if err := proto.Unmarshal(p.GetData().Data, &message); err != nil {
			return fmt.Errorf("failed to unmarshal the received message: %w", err)
		}
		msgType := message.GetType()
		if msgType != protoagent.Message_AGENT_UPGRADE_EXEC_PUSH_RESULT {
			return fmt.Errorf("message of type %s cannot be received", msgType.String())
		}
		return nil
	}
	outCheck := func(ctx context.Context, p *Packet) error {
		if err := checkPacket(p, true); err != nil {
			return err
		}
		switch p.PType {
		case PTData:
			var message protoagent.Message
			if err := proto.Unmarshal(p.GetData().Data, &message); err != nil {
				return fmt.Errorf("failed to unmarshal the received message: %w", err)
			}
			msgType := message.GetType()
			if msgType != protoagent.Message_AGENT_UPGRADE_EXEC_PUSH {
				return fmt.Errorf("message of type %s cannot be sent", msgType.String())
			}
			return nil
		case PTFile:
			if p.GetFile().IsUpgrader() {
				return nil
			}
			return fmt.Errorf("only upgrade files can be sent via this endpoint")
		default:
			return fmt.Errorf(
				"sending packet of type %d, but only data packets (of type %d) or file packets (of type %d) are allowed for this connection",
				p.PType,
				PTData,
				PTFile,
			)
		}
	}
	return &upgradePacketChecker{
		policyInPacketChecker:  newPolicyInPacketChecker(inCheck),
		policyOutPacketChecker: newPolicyOutPacketChecker(outCheck),
	}
}

type inPolicy func(ctx context.Context, packet *Packet) error
type outPolicy func(ctx context.Context, packet *Packet) error

func chainInPolicies(policies []inPolicy) inPolicy {
	return func(ctx context.Context, packet *Packet) error {
		for _, p := range policies {
			if err := p(ctx, packet); err != nil {
				return err
			}
		}
		return nil
	}
}

func chainOutPolicies(policies []outPolicy) outPolicy {
	return func(ctx context.Context, packet *Packet) error {
		for _, p := range policies {
			if err := p(ctx, packet); err != nil {
				return err
			}
		}
		return nil
	}
}

type policyInPacketChecker struct {
	policy inPolicy
}

func newPolicyInPacketChecker(policies ...inPolicy) *policyInPacketChecker {
	return &policyInPacketChecker{
		policy: chainInPolicies(policies),
	}
}

func (p *policyInPacketChecker) CheckInPacket(ctx context.Context, packet *Packet) error {
	return p.policy(ctx, packet)
}

type policyOutPacketChecker struct {
	policy outPolicy
}

func newPolicyOutPacketChecker(policies ...outPolicy) *policyOutPacketChecker {
	return &policyOutPacketChecker{
		policy: chainOutPolicies(policies),
	}
}

func (p *policyOutPacketChecker) CheckOutPacket(ctx context.Context, packet *Packet) error {
	return p.policy(ctx, packet)
}
