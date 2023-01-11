package proto

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"soldr/internal/app/api/models"
	srvcontext "soldr/internal/app/api/server/context"
	"soldr/internal/app/api/server/proto/vm"
	"soldr/internal/app/api/server/response"
	"soldr/internal/app/api/utils"
	"soldr/internal/hardening/luavm/certs"
	vxcommonVM "soldr/internal/hardening/luavm/vm"
	connValidator "soldr/internal/hardening/validator"
	"soldr/internal/protoagent"
	"soldr/internal/system"
	"soldr/internal/version"
	"soldr/internal/vxproto"
	"soldr/internal/vxproto/tunnel"
	tunnelSimple "soldr/internal/vxproto/tunnel/simple"
)

const protoVersion string = "v1"

type ctxVXConnection struct {
	sockID   string
	sockType string
	authReq  *protoagent.AuthenticationRequest
	connType vxproto.AgentType
	ctx      context.Context
	sv       *models.Service
	logger   *logrus.Entry
}

func getRandomID() string {
	raw := make([]byte, 6)
	if _, err := rand.Read(raw); err != nil {
		return ""
	}
	return hex.EncodeToString(raw[:])
}

func getAgentSocketURL(ctxConn *ctxVXConnection) (*url.URL, error) {
	var u url.URL
	u.Scheme = ctxConn.sv.Info.Server.Proto
	u.Host = fmt.Sprintf("%s:%d", ctxConn.sv.Info.Server.Host, ctxConn.sv.Info.Server.Port)
	u.Path = fmt.Sprintf("/api/%s/vxpws/%s/%s/", protoVersion, ctxConn.sockType, ctxConn.sockID)
	return &u, nil
}

func prepareVM(
	ctxConn *ctxVXConnection,
	certsProvider certs.CertProvider,
	ltacGetter vxcommonVM.LTACGetter,
) (*vm.VM, tunnel.PackEncryptor, error) {
	packEncrypter, err := tunnel.NewPackEncrypter(&tunnel.Config{
		Simple: &tunnelSimple.Config{},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize a new pack encrypter: %w", err)
	}
	hardeningVM, err := vm.NewVM(packEncrypter, certsProvider, ltacGetter)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize a hardening VM: %w", err)
	}

	return hardeningVM, packEncrypter, nil
}

func prepareServerWSConn(ctxConn *ctxVXConnection, tlsConfig *tls.Config) (vxproto.IWSConnection, error) {
	u, err := getAgentSocketURL(ctxConn)
	if err != nil {
		return nil, fmt.Errorf("failed to get the URL for the agent socket: %w", err)
	}

	dialer := websocket.Dialer{
		TLSClientConfig: tlsConfig,
	}
	ws, _, err := dialer.DialContext(ctxConn.ctx, u.String(), http.Header{})
	if err != nil {
		return nil, err
	}

	// Use experimental feature
	ws.EnableWriteCompression(true)

	return vxproto.NewWSConnection(ws, false), nil
}

func prepareClientWSConn(w http.ResponseWriter, r *http.Request) (vxproto.IWSConnection, error) {
	// Upgrade connection to Websocket
	upgrader := websocket.Upgrader{
		// Use experimental feature
		EnableCompression: true,
		// TODO: May need to check the Origin but for now so
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	// Use experimental feature
	ws.EnableWriteCompression(true)

	return vxproto.NewWSConnection(ws, false), nil
}

func doVXServerConnection(
	ctxConn *ctxVXConnection,
	agentInfo *protoagent.Information,
	ltacGetter vxcommonVM.LTACGetter,
) (*socket, error) {
	certsDir := filepath.Join("security", "certs", "api")
	if dir, ok := os.LookupEnv("CERTS_PATH"); ok {
		certsDir = dir
	}
	hardeningVM, packEncryptor, err := prepareVM(ctxConn, NewCertProvider(certsDir), ltacGetter)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare VM: %w", err)
	}

	version := version.GetBinaryVersion()
	connValidator := connValidator.NewValidator(ctxConn.sockID, version, hardeningVM)
	tlsConfig, err := hardeningVM.GetTLSConfigForConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to generate TLS config: %w", err)
	}

	ctxConn.logger.Debug("try prepareServerWSConn")
	wsConn, err := prepareServerWSConn(ctxConn, tlsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare WS Connection: %w", err)
	}

	serverConn := &socket{
		id:            ctxConn.sockID,
		ip:            ctxConn.sv.Info.Server.Host,
		src:           ctxConn.authReq.GetAtoken(),
		at:            ctxConn.connType,
		packEncrypter: packEncryptor,
		IWSConnection: wsConn,
		IVaildator:    &validator{},
		IMMInformator: &informator{},
	}

	setPinger := func(p vxproto.Pinger) error {
		serverConn.pinger = p
		return nil
	}
	ctxConn.logger.Debug("try OnConnect")
	if err := connValidator.OnConnect(
		context.WithValue(ctxConn.ctx, vm.CTXSockIDKey, ctxConn.sockID),
		serverConn,
		packEncryptor,
		setPinger,
		agentInfo,
	); err != nil {
		wsConn.Close(ctxConn.ctx)
		return nil, fmt.Errorf("connection callback error: %w", err)
	}
	ctxConn.logger.Debug("try pinger Start")
	if err := serverConn.pinger.Start(ctxConn.ctx, serverConn.ping); err != nil {
		wsConn.Close(ctxConn.ctx)
		return nil, fmt.Errorf("failed to start the pingee: %w", err)
	}
	serverConn.connected = true
	ctxConn.logger.Debug("doVXServerConnection done")
	return serverConn, nil
}

func recvAuthReq(ctx context.Context, conn vxproto.IConnection) (*protoagent.AuthenticationRequest, error) {
	var authReqMessage protoagent.AuthenticationRequest
	authMessageData, err := conn.Read(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to recv auth request from client: %w", err)
	}
	if err := proto.Unmarshal(authMessageData, &authReqMessage); err != nil {
		return nil, fmt.Errorf("failed to parse auth client request: %w", err)
	}
	return &authReqMessage, nil
}

func sendAuthResp(ctx context.Context, conn vxproto.IConnection, authRespMessage *protoagent.AuthenticationResponse) error {
	authMessageData, err := proto.Marshal(authRespMessage)
	if err != nil {
		return fmt.Errorf("failed to build auth client response: %w", err)
	}
	if err = conn.Write(ctx, authMessageData); err != nil {
		return fmt.Errorf("failed to send auth response to client: %w", err)
	}
	return nil
}

func wsConnectToVXServer(c *gin.Context, connType vxproto.AgentType, sockID, sockType string, uaf utils.UserActionFields) {
	var (
		err        error
		serverConn *socket
		sv         *models.Service
		validate   = models.GetValidator()
	)

	logger := utils.FromContext(c).WithFields(logrus.Fields{
		"sock_id":   sockID,
		"sock_type": sockType,
		"conn_id":   getRandomID(),
	})

	if err := validate.Var(sockID, "len=32,hexadecimal,lowercase,required"); err != nil {
		logger.WithError(err).Error("failed to validate sock ID (agent or group)")
		utils.HTTPErrorWithUAFields(c, response.ErrProtoInvalidAgentID, err, uaf)
		return
	}

	if val, ok := c.Get("SV"); !ok {
		logger.Error("error getting vxservice instance from context")
		utils.HTTPErrorWithUAFields(c, response.ErrProtoNoServiceInfo, nil, uaf)
		return
	} else if sv = val.(*models.Service); sv == nil {
		logger.Error("got nil value vxservice instance from context")
		utils.HTTPErrorWithUAFields(c, response.ErrProtoNoServiceInfo, nil, uaf)
		return
	}

	agentInfo, err := system.GetAgentInfo(c)
	if err != nil {
		logger.WithError(err).Error("failed to get the agent info")
		utils.HTTPErrorWithUAFields(c, response.ErrProtoNoServiceInfo, err, uaf)
		return
	}
	logger.Debug("try prepareClientWSConn")
	clientConn, err := prepareClientWSConn(c.Writer, c.Request)
	if err != nil {
		logger.WithError(err).Error("failed to upgrade to websockets")
		utils.HTTPErrorWithUAFields(c, response.ErrProtoUpgradeFail, err, uaf)
		return
	}
	defer clientConn.Close(c.Request.Context())
	defer func(uaFields *utils.UserActionFields) {
		c.Set("uaf", []utils.UserActionFields{*uaFields})
	}(&uaf)

	logger.Debug("try recvAuthReq")
	authReq, err := recvAuthReq(c.Request.Context(), clientConn)
	if err != nil {
		logger.WithError(err).Error("failed to do client handshake")
		uaf.FailReason = "failed to do client handshake"
		uaf.Success = false
		return
	} else {
		logger.WithField("auth_req", authReq).Debug("got authentication request")
	}

	ctxConn := &ctxVXConnection{
		sockID:   sockID,
		authReq:  authReq,
		connType: connType,
		ctx:      c.Request.Context(),
		sockType: sockType,
		sv:       sv,
		logger:   logger,
	}
	certsDir := filepath.Join("security", "certs", "api")
	if dir, ok := os.LookupEnv("CERTS_PATH"); ok {
		certsDir = dir
	}
	logger.WithField("auth_req", authReq).Debug("try doVXServerConnection")
	if serverConn, err = doVXServerConnection(
		ctxConn,
		agentInfo,
		NewStore(filepath.Join(certsDir, sockType)),
	); err != nil {
		clientConn.Close(c.Request.Context())
		logger.WithError(err).Error("failed to initialize connection to server")
		uaf.FailReason = "failed to initialize connection to server"
		uaf.Success = false
		return
	}
	defer serverConn.Close(c.Request.Context())

	logger.Debug("try sendAuthResp")
	if err := sendAuthResp(c.Request.Context(), clientConn, serverConn.authResp); err != nil {
		logger.WithError(err).Error("failed to do client handshake")
		uaf.FailReason = "failed to do client handshake"
		uaf.Success = false
		return
	} else {
		logger.WithField("auth_resp", serverConn.authResp).Debug("sent authentication response")
	}

	logger.WithFields(logrus.Fields{
		"client_conn": clientConn,
		"server_conn": serverConn,
	}).Debug("try linkPairSockets")
	if err := linkPairSockets(ctxConn, clientConn, serverConn); err != nil {
		logger.WithError(err).Error("error on processing data through ws conns")
		uaf.FailReason = "error on processing data through ws conns"
		uaf.Success = false
		return
	} else {
		logger.Debug("WS handler connection done correctly")
	}
	uaf.Success = true
}

func AggregateWSConnect(c *gin.Context) {
	sockID := c.Param("group_id")
	uaf := utils.UserActionFields{
		Domain:            "agent",
		ObjectType:        "agent",
		ObjectId:          sockID,
		ActionCode:        "interactive interaction",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	name, err := utils.GetGroupName(c, sockID)
	if err == nil {
		uaf.ObjectDisplayName = name
	} else {
		utils.FromContext(c).WithError(err).Errorf("error finding group by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, response.ErrAgentsNotFound, nil, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, response.ErrInternal, err, uaf)
		}
		return
	}

	sockType, ok := srvcontext.GetString(c, "cpt")
	if !ok || sockType != "aggregate" {
		utils.FromContext(c).WithError(nil).Errorf("mismatch socket type to incoming token type")
		utils.HTTPErrorWithUAFields(c, response.ErrProtoSockMismatch, nil, uaf)
		return
	}

	wsConnectToVXServer(c, vxproto.Aggregate, sockID, sockType, uaf)
}

func BrowserWSConnect(c *gin.Context) {
	sockID := c.Param("agent_id")
	uaf := utils.UserActionFields{
		Domain:            "agent",
		ObjectType:        "agent",
		ObjectId:          sockID,
		ActionCode:        "interactive interaction",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	name, err := utils.GetAgentName(c, sockID)
	if err == nil {
		uaf.ObjectDisplayName = name
	} else {
		utils.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, response.ErrAgentsNotFound, nil, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, response.ErrInternal, err, uaf)
		}
		return
	}

	sockType, ok := srvcontext.GetString(c, "cpt")
	if !ok || sockType != "browser" {
		utils.FromContext(c).WithError(nil).Errorf("mismatch socket type to incoming token type")
		utils.HTTPErrorWithUAFields(c, response.ErrProtoSockMismatch, nil, uaf)
		return
	}

	wsConnectToVXServer(c, vxproto.Browser, sockID, sockType, uaf)
}

func ExternalWSConnect(c *gin.Context) {
	sockID := c.Param("agent_id")
	uaf := utils.UserActionFields{
		Domain:            "agent",
		ObjectType:        "agent",
		ObjectId:          sockID,
		ActionCode:        "interactive interaction",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	name, err := utils.GetAgentName(c, sockID)
	if err == nil {
		uaf.ObjectDisplayName = name
	} else {
		utils.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, response.ErrAgentsNotFound, nil, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, response.ErrInternal, err, uaf)
		}
		return
	}

	sockType, ok := srvcontext.GetString(c, "cpt")
	if !ok || sockType != "external" {
		utils.FromContext(c).WithError(nil).Errorf("mismatch socket type to incoming token type")
		utils.HTTPErrorWithUAFields(c, response.ErrProtoSockMismatch, nil, uaf)
		return
	}

	wsConnectToVXServer(c, vxproto.External, sockID, sockType, uaf)
}
