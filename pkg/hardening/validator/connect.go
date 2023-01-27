package hardening

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"soldr/pkg/app/agent"
	vxcommonErrors "soldr/pkg/errors"
	"soldr/pkg/hardening/pingee"
	"soldr/pkg/utils"
	utilsErrors "soldr/pkg/utils/errors"
	"soldr/pkg/vxproto"
	"soldr/pkg/vxproto/tunnel"
)

func (v *Validator) OnConnect(
	ctx context.Context,
	socket vxproto.IAgentSocket,
	encrypter tunnel.PackEncryptor,
	configurePingee func(p vxproto.Pinger) error,
	agentInfo *agent.Information,
) error {
	if err := v.doHandshakeWithServer(ctx, socket, encrypter, agentInfo); err != nil {
		return fmt.Errorf("handshake with server has failed: %w", err)
	}
	p := pingee.NewPingee(ctx, v.vm, socket)
	if err := configurePingee(p); err != nil {
		return fmt.Errorf("failed to configure a pingee: %w", err)
	}
	pubInfo := socket.GetPublicInfo()
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"id":     pubInfo.ID,
		"type":   pubInfo.Type.String(),
		"src":    pubInfo.Src,
		"dst":    pubInfo.Dst,
		"ver":    pubInfo.Ver,
	}).Info("vxserver: connect success")
	return nil
}

func (v *Validator) ProcessError(err error) error {
	if IsAgentUnauthorizedErr(err) {
		err = fmt.Errorf("%w: %v", vxcommonErrors.ErrConnectionInitializationRequired, err)
		if vmErr := v.vm.ResetConnection(); vmErr != nil {
			return fmt.Errorf("failed to reset connection (%v), after the server handshake has failed: %w", vmErr, err)
		}
	}
	return err
}

func (v *Validator) doHandshakeWithServer(
	ctx context.Context,
	socket vxproto.IAgentSocket,
	encrypter tunnel.PackEncryptor,
	agentInfo *agent.Information,
) error {
	logrus.WithContext(ctx).Debug("doing handshake with server")
	if err := doHandshakeWithServerOnAgent(ctx, socket, agentInfo); err != nil {
		return fmt.Errorf("failed to perform the second part of the hadshake with server: %w", v.ProcessError(err))
	}

	logrus.WithContext(ctx).Debug("performing connection challenge")
	msg, err := socket.Read(ctx)
	if err != nil {
		return fmt.Errorf("reading the handshake message from the server has failed")
	}
	connChallengeResp, err := v.vm.ProcessConnectionChallengeRequest(ctx, msg)
	if err != nil {
		return fmt.Errorf("failed to process the connection challenge request: %w", err)
	}
	if err := socket.Write(ctx, connChallengeResp); err != nil {
		return fmt.Errorf("failed to send the connection challenge response to the server: %w", err)
	}

	logrus.WithContext(ctx).Debug("processing connection request")
	msg, err = socket.Read(ctx)
	if err != nil {
		return fmt.Errorf("reading the handshake message to start the connection from the server has failed: %w", err)
	}
	resp, err := v.vm.ProcessConnectionRequest(msg, encrypter)
	if err != nil {
		return fmt.Errorf("failed to process the start connection request: %w", err)
	}
	if err := socket.Write(ctx, resp); err != nil {
		return fmt.Errorf("failed to send the connection response to the server: %w", err)
	}
	return nil
}

func doHandshakeWithServerOnAgent(
	ctx context.Context,
	iasocket vxproto.IAgentSocket,
	infoMessage *agent.Information,
) error {
	logrus.WithContext(ctx).Debug("doing handshake with server or agent")
	seconds := time.Now().Unix()
	authReqMessage := &agent.AuthenticationRequest{
		Timestamp: &seconds,
		Atoken:    utils.GetRef(iasocket.GetSource()),
		Aversion:  utils.GetRef(iasocket.GetVersion()),
		Ainfo:     infoMessage,
	}
	authMessageData, err := proto.Marshal(authReqMessage)
	if err != nil {
		return err
	}
	iasocket.SetAuthReq(authReqMessage)
	iasocket.SetInfo(infoMessage)
	logrus.WithContext(ctx).Debug("sending auth message")
	if err = iasocket.Write(ctx, authMessageData); err != nil {
		return err
	}

	var authRespMessage agent.AuthenticationResponse
	authMessageData, err = iasocket.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the authentication response: %w", err)
	}
	if err = proto.Unmarshal(authMessageData, &authRespMessage); err != nil {
		return err
	}
	emptyTokens := authRespMessage.GetAtoken() == "" && authRespMessage.GetStoken() == ""
	if emptyTokens || authRespMessage.GetStatus() != "authorized" {
		var serverErr error
		switch status := authRespMessage.GetStatus(); status {
		case utilsErrors.ErrFailedResponseCorrupted.Error():
			serverErr = utilsErrors.ErrFailedResponseCorrupted
		case utilsErrors.ErrFailedResponseUnauthorized.Error():
			serverErr = utilsErrors.ErrFailedResponseUnauthorized
		case utilsErrors.ErrFailedResponseBlocked.Error():
			serverErr = utilsErrors.ErrFailedResponseBlocked
		default:
			serverErr = errors.New(status)
		}
		return fmt.Errorf("failed auth on server side: %w", serverErr)
	}
	if !iasocket.HasTokenCRCValid(authRespMessage.GetAtoken()) {
		return fmt.Errorf("agent token is corrupted")
	}
	if !iasocket.HasTokenCRCValid(authRespMessage.GetStoken()) {
		return fmt.Errorf("server token is corrupted")
	}
	iasocket.SetAuthResp(&authRespMessage)
	iasocket.SetSource(authRespMessage.GetAtoken())
	iasocket.SetVersion(authRespMessage.GetSversion())
	if err = iasocket.HasAgentInfoValid(ctx, iasocket); err != nil {
		return fmt.Errorf("agent info is invalid: %w", err)
	}

	return nil
}
