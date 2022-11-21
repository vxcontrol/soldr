package hardening

import (
	"context"
	"fmt"

	vxcommonErrors "soldr/internal/errors"
	"soldr/internal/hardening/luavm/vm"
	"soldr/internal/protoagent"
	"soldr/internal/vxproto"
)

func (v *Validator) OnInitConnect(ctx context.Context, socket vxproto.SyncWS, agentInfo *protoagent.Information) (err error) {
	defer func() {
		if err != nil {
			v.vm.ResetInitConnection()
			err = fmt.Errorf("%w (%v)", vxcommonErrors.ErrConnectionInitializationRequired, err)
			return
		}
	}()
	if err = v.sendInitConnectRequest(ctx, socket, agentInfo); err != nil {
		return err
	}
	if err := v.getInitConnectResponse(ctx, socket); err != nil {
		return err
	}
	return nil
}

func (v *Validator) sendInitConnectRequest(ctx context.Context, ws vxproto.SyncWS, info *protoagent.Information) error {
	req, err := v.vm.PrepareInitConnectionRequest(&vm.InitConnectionAgentInfo{
		ID:      v.agentID,
		Version: v.version,
	}, info)
	if err != nil {
		return fmt.Errorf("failed to prepare the init connection request: %w", err)
	}
	if err := ws.Write(ctx, req); err != nil {
		return fmt.Errorf("failed to send the init connection message: %w", err)
	}
	return nil
}

func (v *Validator) getInitConnectResponse(ctx context.Context, ws vxproto.SyncWS) error {
	msg, err := ws.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read an init connect response: %w", err)
	}
	if err := v.vm.ProcessInitConnectionResponse(msg); err != nil {
		return fmt.Errorf("failed to process the init connection response: %w", err)
	}
	return nil
}
