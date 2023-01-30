package vxproto

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	obs "soldr/pkg/observability"
	"soldr/pkg/protoagent"
	"soldr/pkg/system"
	utilsErrors "soldr/pkg/utils/errors"
	"soldr/pkg/vxproto/tunnel"
)

type agentConn interface {
	connect(ctx context.Context) error
}

type AgentConnectionValidator interface {
	OnInitConnect(
		ctx context.Context,
		socket SyncWS,
		agentInfo *protoagent.Information,
	) error
	OnConnect(
		ctx context.Context,
		socket IAgentSocket,
		packEncryptor tunnel.PackEncryptor,
		configurePingee func(p Pinger) error,
		agentInfo *protoagent.Information,
	) error
	ProcessError(err error) error
}

func (vxp *vxProto) openAgentSocket(
	ctx context.Context,
	config *ClientConfig,
	connValidator AgentConnectionValidator,
	tunnelEncrypter tunnel.PackEncryptor,
	infoGetter system.AgentInfoGetter,
) (socket *agentSocket, cleanup func(), err error) {
	if config.Type == "aggregate" || config.Type == "browser" || config.Type == "external" {
		return nil, nil, fmt.Errorf("connection initialization for the browser type is NYI")
	}
	dialer := websocket.Dialer{
		TLSClientConfig: config.TLSConfig,
	}
	u, err := getAgentSocketURL(config.Host, config.ID, config.ProtocolVersion)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the URL for the agent socket: %w", err)
	}

	ctx, cancelCtx := context.WithCancel(ctx)
	defer func() {
		if err != nil {
			cancelCtx()
		}
	}()
	agentInfo, err := infoGetter.Get(ctx)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get the agent info: %w", err)
	}
	ws, resp, err := dialer.Dial(u.String()+"/", http.Header{})
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusForbidden {
			return nil, nil, connValidator.ProcessError(utilsErrors.ErrFailedResponseUnauthorized)
		}
		if strings.HasSuffix(err.Error(), "bad certificate") {
			return nil, nil, connValidator.ProcessError(utilsErrors.ErrFailedResponseTunnelError)
		}
		return nil, nil, err
	}

	defer func() {
		if err != nil {
			ws.Close()
		}
	}()

	// Use experimental feature
	ws.EnableWriteCompression(true)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize a new pack encryptor: %w", err)
	}
	socket = &agentSocket{
		id:               config.ID,
		ip:               config.Host,
		src:              config.Token,
		at:               VXAgent,
		auth:             &AuthenticationData{},
		packEncrypter:    tunnelEncrypter,
		IConnection:      NewWSConnection(ws, false),
		IVaildator:       vxp,
		IMMInformator:    vxp,
		IProtoStats:      vxp,
		IProtoIO:         vxp,
		connectionPolicy: newAllowPacketChecker(),
	}
	if vxp.IMainModule != nil && connValidator != nil {
		err := connValidator.OnConnect(ctx, socket, tunnelEncrypter, getConfigurePingeeFn(socket), agentInfo)
		if err != nil {
			return nil, nil, fmt.Errorf("connection callback error: %w", err)
		}

		// Save received token
		config.Token = socket.src

		// Register new agent
		if !vxp.addAgent(ctx, socket) {
			return nil, nil, fmt.Errorf("failed adding of agent")
		}
	} else {
		return nil, nil, fmt.Errorf("disallowed unauthorized connection")
	}
	return socket, func() {
		vxp.delAgent(ctx, socket)
		ws.Close()
	}, nil
}

func getAgentSocketURL(host string, id string, version string) (*url.URL, error) {
	u, err := getURLFromHost(host)
	if err != nil {
		return nil, err
	}
	u.Path = path.Join(fmt.Sprintf("/api/%s/vxpws/agent", version), id)
	return u, nil
}

func getConfigurePingeeFn(socket *agentSocket) func(p Pinger) error {
	return func(p Pinger) error {
		socket.pinger = p
		return nil
	}
}

func (a *agentSocket) connect(ctx context.Context) error {
	if err := a.pinger.Start(ctx, a.ping); err != nil {
		return fmt.Errorf("failed to start the pingee: %w", err)
	}
	defer func() {
		a.pinger.Stop(ctx)
	}()

	if closeRootSpanCb, ok := ctx.Value(obs.VXProtoAgentConnect).(func()); ok {
		closeRootSpanCb()
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			if err := a.recvPacket(ctx); err != nil {
				return err
			}
		}
	}
}

const (
	defaultPingFrequency = time.Second * 5
	defaultReadTimeout   = defaultPingFrequency * 6
)

func openAgentSocketToInitConnection(ctx context.Context, logger *logrus.Entry, config *ClientInitConfig) (SyncWS, error) {
	if config.Type == "aggregate" || config.Type == "browser" || config.Type == "external" {
		return nil, fmt.Errorf("connection initialization for the browser type is NYI")
	}
	dialer := websocket.Dialer{
		TLSClientConfig: config.TLSConfig,
	}

	u, err := getInitConnURL(config.Host, config.ProtocolVersion)
	if err != nil {
		return nil, fmt.Errorf("failed to get an init connection URL: %w", err)
	}
	ws, _, err := dialer.Dial(u.String(), nil)
	if err != nil {
		return nil, err
	}

	// Use experimental feature
	ws.EnableWriteCompression(true)

	return NewSyncWS(ctx, logger, ws, &SyncWSConfig{
		SendPingFrequency: defaultPingFrequency,
		ReadTimeout:       defaultReadTimeout,
	})
}

func getInitConnURL(host string, version string) (*url.URL, error) {
	u, err := getURLFromHost(host)
	if err != nil {
		return nil, err
	}
	u.Path = fmt.Sprintf("/api/%s/vxpws/agent/", version)
	return u, nil
}
