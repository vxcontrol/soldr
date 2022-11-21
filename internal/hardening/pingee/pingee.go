package pingee

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"soldr/internal/hardening/luavm/vm"
	obs "soldr/internal/observability"
	"soldr/internal/vxproto"
)

type Pingee struct {
	vm vm.VM

	socket vxproto.IAgentSocket
	pingFn func(ctx context.Context, nonce []byte) error

	ctx       context.Context
	cancelCtx context.CancelFunc

	gotPing chan struct{}
}

func NewPingee(ctx context.Context, vm vm.VM, socket vxproto.IAgentSocket) *Pingee {
	p := &Pingee{
		vm:      vm,
		gotPing: make(chan struct{}, 1),
		socket:  socket,
	}
	p.ctx, p.cancelCtx = context.WithCancel(ctx)
	return p
}

func (p *Pingee) Start(ctx context.Context, ping func(ctx context.Context, nonce []byte) error) error {
	p.pingFn = ping

	go func() {
		timer := time.NewTimer(vxproto.PingerInterval * 3)
		defer func() {
			timer.Stop()
		}()
		for {
			select {
			case <-timer.C:
				waiterCtx, waiterSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "pinger_waiter")
				logrus.WithContext(waiterCtx).Error("the ping timeout has been exceeded, disconnecting from the server")
				if err := p.socket.Close(ctx); err != nil {
					logrus.WithContext(waiterCtx).WithError(err).Errorf("closing socket has failed")
				}
				p.cancelCtx()
				waiterSpan.End()
				return
			case <-p.gotPing:
				timer.Reset(vxproto.PingerInterval * 3)
			case <-p.ctx.Done():
				waiterCtx, waiterSpan := obs.Observer.NewSpan(ctx, obs.SpanKindInternal, "pinger_waiter")
				logrus.WithContext(waiterCtx).Debug("the pingee waiter routine has been finished")
				waiterSpan.End()
				return
			}
		}
	}()
	return nil
}

func (p *Pingee) Process(ctx context.Context, pingData []byte) error {
	resp, err := p.vm.GeneratePingResponse(pingData)
	if err != nil {
		return fmt.Errorf("failed to generate the ping response: %w", err)
	}
	if err := p.pingFn(ctx, resp); err != nil {
		return fmt.Errorf("failed to send the ping response: %w", err)
	}
	select {
	case p.gotPing <- struct{}{}:
	default:
		return fmt.Errorf("ping channel is blocked")
	}
	return nil
}

func (p *Pingee) Stop(context.Context) error {
	p.cancelCtx()
	return nil
}
