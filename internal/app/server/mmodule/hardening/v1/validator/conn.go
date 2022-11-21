package validator

import (
	"context"
	"crypto/tls"
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"soldr/internal/app/server/mmodule/hardening/v1/pinger"
	"soldr/internal/protoagent"
	"soldr/internal/utils"
	utilsErrors "soldr/internal/utils/errors"
	"soldr/internal/vxproto"
	"soldr/internal/vxproto/tunnel"
)

func (v *ConnectionValidator) OnConnect(
	ctx context.Context,
	tlsConnState *tls.ConnectionState,
	socket vxproto.IAgentSocket,
	agentType vxproto.AgentType,
	configurePackEncryptor func(c *tunnel.Config) error,
	configurePinger func(p vxproto.Pinger),
) error {
	logger := logrus.WithContext(ctx).WithFields(logrus.Fields{
		"component":  "conn_validator",
		"module":     "main",
		"agent_type": agentType.String(),
		"agent_id":   socket.GetAgentID(),
		"group_id":   socket.GetGroupID(),
	})
	logger.Debug("getting old token")
	if err := getOldToken(ctx, socket, logger); err != nil {
		return fmt.Errorf("failed to get old token: %w", err)
	}

	logger.Debug("performing challenge")
	if err := v.performChallenge(ctx, agentType, socket); err != nil {
		logger.WithError(err).Errorf("failed to perform the connection challenge")
		return fmt.Errorf("connection challenge failed: %w", err)
	}

	logger.Debug("requesting connection start")
	if err := v.requestConnectionStart(ctx, tlsConnState, socket, configurePackEncryptor); err != nil {
		return fmt.Errorf("failed to send a request to start connection: %w", err)
	}

	logger.Debug("starting pinger")
	p := pinger.NewPinger(v.ctx, socket, agentType, v.abher)
	configurePinger(p)
	return nil
}

func (v *ConnectionValidator) CheckConnectionTLS(s *tls.ConnectionState) error {
	return nil
}

func (v *ConnectionValidator) exchangeChallengeMessages(
	ctx context.Context,
	agentType vxproto.AgentType,
	socket vxproto.IAgentSocket,
) error {
	challenge, err := v.challenger.GetConnectionChallenge()
	if err != nil {
		return fmt.Errorf("failed to get a connection challenge: %w", err)
	}
	if err := v.sendConnectionChallenge(ctx, socket, challenge); err != nil {
		return fmt.Errorf("failed to send the connection challenge: %w", err)
	}
	if err := v.checkConnectionChallengeResponse(ctx, agentType, socket, challenge); err != nil {
		return fmt.Errorf("failed to check the connection challenge response: %w", err)
	}
	return nil
}

func (v *ConnectionValidator) performChallenge(
	ctx context.Context, agentType vxproto.AgentType, socket vxproto.IAgentSocket,
) error {
	err := v.exchangeChallengeMessages(ctx, agentType, socket)
	if err == nil {
		return nil
	}

	emptyStr := ""
	ver := socket.GetVersion()
	respMsg := utilsErrors.ErrFailedResponseTunnelError.Error()
	resp := &protoagent.AuthenticationResponse{
		Atoken:   &emptyStr,
		Stoken:   &emptyStr,
		Sversion: &ver,
		Status:   &respMsg,
	}
	data, packErr := protoagent.PackProtoMessage(resp, protoagent.Message_AUTHENTICATION_RESPONSE)
	if packErr != nil {
		return fmt.Errorf("failed to pack the proto mesage (%v) while processing the error: %w", packErr, err)
	}
	if writeErr := socket.Write(ctx, data); writeErr != nil {
		return fmt.Errorf("failed to send the authentication response (%v) while processing the error: %w", writeErr, err)
	}
	return err
}

func (v *ConnectionValidator) sendConnectionChallenge(
	ctx context.Context, asocket vxproto.IAgentSocket, nonce []byte,
) error {
	req := &protoagent.ConnectionChallengeRequest{
		Nonce: nonce,
	}
	msg, err := protoagent.PackProtoMessage(req, protoagent.Message_CONNECTION_CHALLENGE_REQUEST)
	if err != nil {
		return fmt.Errorf("failed to pack the message: %w", err)
	}
	if err := asocket.Write(ctx, msg); err != nil {
		return fmt.Errorf("failed to write the message to the socket: %w", err)
	}
	return nil
}

func (v *ConnectionValidator) checkConnectionChallengeResponse(
	ctx context.Context,
	agentType vxproto.AgentType,
	socket vxproto.IAgentSocket,
	expectedChallenge []byte,
) error {
	var resp protoagent.ConnectionChallengeResponse
	respData, err := socket.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read the message from the socket: %w", err)
	}
	if err := protoagent.UnpackProtoMessage(&resp, respData, protoagent.Message_CONNECTION_CHALLENGE_REQUEST); err != nil {
		return fmt.Errorf("failed to unpack the message: %w", err)
	}
	abh, err := v.abher.GetABHWithSocket(agentType, socket)
	if err != nil {
		return fmt.Errorf("failed to get the ABH: %w", err)
	}
	if err := v.challenger.CheckConnectionChallenge(
		resp.Ct,
		expectedChallenge,
		socket.GetAgentID(),
		abh,
	); err != nil {
		return fmt.Errorf("connection challenge check has failed: %w", err)
	}
	return nil
}

func (v *ConnectionValidator) requestConnectionStart(
	ctx context.Context,
	tlsConnState *tls.ConnectionState,
	socket vxproto.IAgentSocket,
	configureTunnel func(c *tunnel.Config) error,
) error {
	tunnelConfig, agentTunnelConfig, err := v.tunnelConfigurer.GetTunnelConfig()
	if err != nil {
		return fmt.Errorf("failed to get a tunnel config: %w", err)
	}
	if err := configureTunnel(tunnelConfig); err != nil {
		return fmt.Errorf("failed to configure the tunnel: %w", err)
	}
	connReq := &protoagent.ConnectionStartRequest{
		TunnelConfig: agentTunnelConfig,
	}
	connReq.Sbh, err = v.sbher.Get(context.TODO(), tlsConnState.ServerName)
	if err != nil {
		return fmt.Errorf("failed to get the SBH: %w", err)
	}
	req, err := protoagent.PackProtoMessage(connReq, protoagent.Message_CONNECTION_REQUEST)
	if err != nil {
		return fmt.Errorf("failed to prepare a request to start connection: %w", err)
	}
	if err := socket.Write(ctx, req); err != nil {
		return fmt.Errorf("failed to send a request to start connection: %w", err)
	}
	data, err := socket.Read(ctx)
	if err != nil {
		return fmt.Errorf("failed to read a response to the start connection request: %w", err)
	}
	var resp protoagent.ConnectionStartResponse
	if err := protoagent.UnpackProtoMessage(&resp, data, protoagent.Message_CONNECTION_REQUEST); err != nil {
		return fmt.Errorf("failed to unpack the response to the start connection request: %w", err)
	}
	return nil
}

func getOldToken(ctx context.Context, socket vxproto.IAgentSocket, logger *logrus.Entry) error {
	pubInfo := socket.GetPublicInfo()
	var err error
	switch pubInfo.Type {
	case vxproto.VXAgent, vxproto.Browser, vxproto.External:
		err = doHandshakeWithAgentOnServer(ctx, socket)
	default:
		err = fmt.Errorf("unknown client type")
	}
	if err != nil {
		logger.WithError(err).Error("failed to agent connect on callback")
	} else {
		logger.WithFields(logrus.Fields{
			"id":   pubInfo.ID,
			"gid":  pubInfo.GID,
			"type": pubInfo.Type.String(),
			"src":  pubInfo.Src,
			"dst":  pubInfo.Dst,
			"ver":  pubInfo.Ver,
		}).Debug("agent connected successful on callback")
	}
	return err
}

func doHandshakeWithAgentOnServer(ctx context.Context, iasocket vxproto.IAgentSocket) error {
	makeFailedResponse := func(err error) *protoagent.AuthenticationResponse {
		return &protoagent.AuthenticationResponse{
			Atoken:   utils.GetRef(""),
			Stoken:   utils.GetRef(""),
			Sversion: utils.GetRef(iasocket.GetVersion()),
			Status:   utils.GetRef(err.Error()),
		}
	}
	makeSuccessResponse := func(atoken, stoken string) *protoagent.AuthenticationResponse {
		return &protoagent.AuthenticationResponse{
			Atoken:   utils.GetRef(atoken),
			Stoken:   utils.GetRef(stoken),
			Sversion: utils.GetRef(iasocket.GetVersion()),
			Status:   utils.GetRef("authorized"),
		}
	}
	sendResponse := func(resp *protoagent.AuthenticationResponse) error {
		respData, err := proto.Marshal(resp)
		if err != nil {
			return err
		}

		if err = iasocket.Write(ctx, respData); err != nil {
			return err
		}

		return nil
	}

	authMessageData, err := iasocket.Read(ctx)
	if err != nil {
		_ = sendResponse(makeFailedResponse(fmt.Errorf("internal error")))
		return fmt.Errorf("failed to read the message from the socket: %w", err)
	}
	var authReqMessage protoagent.AuthenticationRequest
	err = proto.Unmarshal(authMessageData, &authReqMessage)
	if err != nil {
		_ = sendResponse(makeFailedResponse(utilsErrors.ErrFailedResponseCorrupted))
		return err
	}

	if authReqMessage.Ainfo == nil {
		_ = sendResponse(makeFailedResponse(utilsErrors.ErrFailedResponseCorrupted))
		return fmt.Errorf("agent info is empty")
	}

	iasocket.SetInfo(authReqMessage.Ainfo)
	iasocket.SetVersion(authReqMessage.GetAversion())
	if err = iasocket.HasAgentInfoValid(ctx, iasocket); err != nil {
		_ = sendResponse(makeFailedResponse(err))
		return fmt.Errorf("agent info is invalid: %w", err)
	}

	iasocket.SetAuthReq(&authReqMessage)
	token := authReqMessage.GetAtoken()
	pubInfo := iasocket.GetPublicInfo()
	if !iasocket.HasTokenValid(token, pubInfo.ID, pubInfo.Type) {
		if token, err = iasocket.NewToken(pubInfo.ID, pubInfo.Type); err != nil {
			return fmt.Errorf("failed to make agent token: %w", err)
		}
	}

	authRespMessage := makeSuccessResponse(token, iasocket.GetSource())
	iasocket.SetAuthResp(authRespMessage)

	return sendResponse(authRespMessage)
}
