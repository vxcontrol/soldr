package pinger

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	abherTypes "soldr/pkg/app/server/mmodule/hardening/v1/abher/types"
	obs "soldr/pkg/observability"
	"soldr/pkg/vxproto"
)

const nonceSize = 16

type Pinger struct {
	abher     ABHer
	socket    vxproto.IAgentSocket
	agentType vxproto.AgentType

	logger *logrus.Entry

	nonces    map[string]struct{}
	noncesMux *sync.Mutex

	ctx       context.Context
	cancelCtx context.CancelFunc

	gotPing chan struct{}
}

type ABHer interface {
	GetABH(vxproto.AgentType, *abherTypes.AgentBinaryID) ([][]byte, error)
}

func NewPinger(ctx context.Context, socket vxproto.IAgentSocket, agentType vxproto.AgentType, abh ABHer) *Pinger {
	p := &Pinger{
		abher:     abh,
		socket:    socket,
		agentType: agentType,

		logger: logrus.WithFields(logrus.Fields{
			"component":  "conn_pinger",
			"module":     "main",
			"agent_type": agentType.String(),
			"agent_id":   socket.GetAgentID(),
			"group_id":   socket.GetGroupID(),
		}),

		nonces:    make(map[string]struct{}),
		noncesMux: &sync.Mutex{},

		gotPing: make(chan struct{}, 1),
	}
	p.ctx, p.cancelCtx = context.WithCancel(ctx)
	return p
}

func (p *Pinger) Start(ctx context.Context, ping func(ctx context.Context, nonce []byte) error) error {
	go func() {
		timer := time.NewTicker(vxproto.PingerInterval)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				tickerCtx, tickerSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "pinger_ticker")
				if err := p.sendPingMessage(tickerCtx, ping); err != nil {
					p.logger.WithContext(tickerCtx).WithError(err).Warn("failed to send a ping message")
				}
				tickerSpan.End()
			case <-p.ctx.Done():
				return
			}
		}
	}()
	go func() {
		timer := time.NewTimer(vxproto.PingerTimeout)
		defer timer.Stop()
		for {
			select {
			case <-p.gotPing:
				timer.Reset(vxproto.PingerTimeout)
			case <-timer.C:
				waiterCtx, waiterSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "pinger_waiter")
				p.logger.WithContext(waiterCtx).Error("pinger timeout has been exceeded")
				if err := p.socket.Close(waiterCtx); err != nil {
					p.logger.WithContext(waiterCtx).WithError(err).
						Error("the pinger timer has expired, but the pinger failed to close the agent-server connection")
				}
				p.cancelCtx()
				waiterSpan.End()
				return
			case <-p.ctx.Done():
				return
			}
		}
	}()
	return nil
}

func (p *Pinger) sendPingMessage(ctx context.Context, ping func(ctx context.Context, nonce []byte) error) (err error) {
	nonce, err := p.newPushNonce(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate a new nonce: %w", err)
	}
	defer func() {
		if err != nil {
			p.deleteNonce(ctx, nonce)
		}
	}()

	if err := ping(ctx, nonce); err != nil {
		p.logger.WithContext(ctx).WithError(err).Warn("failed to write the ping message")
	}
	return nil
}

func (p *Pinger) Stop(ctx context.Context) error {
	p.cancelCtx()
	return nil
}

func (p *Pinger) Process(ctx context.Context, packetData []byte) error {
	if err := p.ctx.Err(); err != nil {
		p.logger.WithError(err).Debug("context error")
		return nil
	}
	agentID := p.socket.GetAgentID()
	info := p.socket.GetPublicInfo()
	if info == nil || info.Info == nil {
		return fmt.Errorf("failed to get the agent public info")
	}
	agentBinaryID := &abherTypes.AgentBinaryID{
		OS:      info.Info.GetOs().GetType(),
		Arch:    info.Info.GetOs().GetArch(),
		Version: info.Ver,
	}
	abhs, err := p.abher.GetABH(p.agentType, agentBinaryID)
	if err != nil {
		return fmt.Errorf("failed to get an ABH for the agent binary id \"%s\": %w", agentBinaryID.String(), err)
	}
	actualNonce, err := p.getActualNonce(ctx, packetData, agentID, abhs)
	if err != nil {
		return fmt.Errorf("failed to decrypt the ping nonce: %w", err)
	}
	if err := p.checkNonce(ctx, actualNonce); err != nil {
		return fmt.Errorf("failed to check the received nonce: %w", err)
	}
	p.gotPing <- struct{}{}
	return nil
}

func (p *Pinger) getActualNonce(ctx context.Context, packetData []byte, agentID string, abhs [][]byte) ([]byte, error) {
	return packetData, nil
}

func (p *Pinger) newPushNonce(ctx context.Context) ([]byte, error) {
	nonce := make([]byte, nonceSize)
	var nonceKey string
	for {
		if _, err := rand.Read(nonce); err != nil {
			return nil, fmt.Errorf("failed to generate a nonce: %w", err)
		}
		nonceKey = getNonceKey(nonce)
		if !p.isExistNonce(ctx, nonceKey) {
			break
		}
	}

	p.addNonce(ctx, nonceKey)

	go func() {
		t := time.NewTimer(vxproto.PingerTimeout)
		defer t.Stop()
		select {
		case <-t.C:
			p.deleteNonceWithKey(ctx, nonceKey)
		case <-p.ctx.Done():
		}
	}()
	return nonce, nil
}

func (p *Pinger) isExistNonce(_ context.Context, nonceKey string) bool {
	p.noncesMux.Lock()
	defer p.noncesMux.Unlock()

	_, ok := p.nonces[nonceKey]
	return ok
}

func (p *Pinger) addNonce(_ context.Context, nonceKey string) {
	p.noncesMux.Lock()
	defer p.noncesMux.Unlock()

	p.nonces[nonceKey] = struct{}{}
}

func (p *Pinger) checkNonce(ctx context.Context, actualNonce []byte) error {
	getNonces := func() []string {
		p.noncesMux.Lock()
		defer p.noncesMux.Unlock()

		nonces := make([]string, 0, len(p.nonces))
		for n := range p.nonces {
			nonces = append(nonces, n)
		}
		return nonces
	}

	nonces := getNonces()
	for _, n := range nonces {
		expectedNonce, err := hex.DecodeString(n)
		if err != nil {
			p.logger.WithContext(ctx).WithError(err).Error("failed to decode nonce")
			continue
		}
		if bytes.Equal(expectedNonce, actualNonce) {
			p.deleteNonceWithKey(ctx, n)
			return nil
		}
	}
	return fmt.Errorf("got an unknown nonce %s", hex.EncodeToString(actualNonce))
}

func (p *Pinger) deleteNonce(ctx context.Context, nonce []byte) {
	key := getNonceKey(nonce)
	p.deleteNonceWithKey(ctx, key)
}

func (p *Pinger) deleteNonceWithKey(_ context.Context, nonceKey string) {
	p.noncesMux.Lock()
	defer p.noncesMux.Unlock()

	delete(p.nonces, nonceKey)
}

func getNonceKey(nonce []byte) string {
	return hex.EncodeToString(nonce)
}
