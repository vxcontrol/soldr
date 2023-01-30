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
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"soldr/pkg/app/api/client"
	"soldr/pkg/app/api/logger"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/server/proto/vm"
	"soldr/pkg/app/api/server/response"
	"soldr/pkg/app/api/storage"
	"soldr/pkg/app/api/useraction"
	"soldr/pkg/hardening/luavm/certs"
	vxcommonVM "soldr/pkg/hardening/luavm/vm"
	connValidator "soldr/pkg/hardening/validator"
	"soldr/pkg/protoagent"
	"soldr/pkg/system"
	"soldr/pkg/version"
	"soldr/pkg/vxproto"
	"soldr/pkg/vxproto/tunnel"
	tunnelSimple "soldr/pkg/vxproto/tunnel/simple"
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
	certsPath string,
) (*socket, error) {
	hardeningVM, packEncryptor, err := prepareVM(ctxConn, NewCertProvider(certsPath), ltacGetter)
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

func wsConnectToVXServer(
	c *gin.Context,
	connType vxproto.AgentType,
	sockID string,
	sockType string,
	certsPath string,
	uaf useraction.Fields,
) {
	var (
		serverConn *socket
		sv         *models.Service
		validate   = models.GetValidator()
	)

	logger := logger.FromContext(c).WithFields(logrus.Fields{
		"sock_id":   sockID,
		"sock_type": sockType,
		"conn_id":   getRandomID(),
	})

	if err := validate.Var(sockID, "len=32,hexadecimal,lowercase,required"); err != nil {
		logger.WithError(err).Error("failed to validate agent ID")
		response.Error(c, response.ErrProtoInvalidAgentID, err)
		return
	}

	if val, ok := c.Get("SV"); !ok {
		logger.Error("error getting vxservice instance from context")
		response.Error(c, response.ErrProtoNoServiceInfo, nil)
		return
	} else if sv = val.(*models.Service); sv == nil {
		logger.Error("got nil value vxservice instance from context")
		response.Error(c, response.ErrProtoNoServiceInfo, nil)
		return
	}

	agentInfo, err := system.GetAgentInfo(c)
	if err != nil {
		logger.WithError(err).Error("failed to get the agent info")
		response.Error(c, response.ErrProtoNoServiceInfo, err)
		return
	}
	logger.Debug("try prepareClientWSConn")
	clientConn, err := prepareClientWSConn(c.Writer, c.Request)
	if err != nil {
		logger.WithError(err).Error("failed to upgrade to websockets")
		response.Error(c, response.ErrProtoUpgradeFail, err)
		return
	}
	defer clientConn.Close(c.Request.Context())
	defer func(uaFields *useraction.Fields) {
		c.Set("uaf", []useraction.Fields{*uaFields})
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

	logger.WithField("auth_req", authReq).Debug("try doVXServerConnection")
	serverConn, err = doVXServerConnection(
		ctxConn,
		agentInfo,
		NewStore(filepath.Join(certsPath, sockType)),
		certsPath,
	)
	if err != nil {
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

func getServiceHash(c *gin.Context) (string, error) {
	tid := c.GetUint64("tid")
	if tid == 0 {
		return "", fmt.Errorf("could not get tenant ID from context")
	}
	sid := c.GetUint64("tid")
	if sid == 0 {
		return "", fmt.Errorf("could not get service ID from context")
	}

	gDB := getGormDB(c, "gDB")
	if gDB == nil {
		return "", fmt.Errorf("could not get global DB connection from context")
	}

	var svc models.Service
	if err := gDB.Where("tenant_id = ? AND id = ?", tid, sid).Take(&svc).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return "", fmt.Errorf("could not get service record from DB: record not found")
		}
		return "", fmt.Errorf("could not get service record from DB: internal error: %w", err)
	} else if err = svc.Valid(); err != nil {
		return "", fmt.Errorf("could not validate service record: %w", err)
	}
	return svc.Hash, nil
}

func getGormDB(c *gin.Context, name string) *gorm.DB {
	var db *gorm.DB

	if val, ok := c.Get(name); !ok {
		logger.FromContext(c).WithField("component", "gorm_conn_getter").
			Errorf("error getting '" + name + "' from context")
	} else if db = val.(*gorm.DB); db == nil {
		logger.FromContext(c).WithField("component", "gorm_conn_getter").
			Errorf("got nil value '" + name + "' from context")
	}

	return db
}

type ProtoService struct {
	db               *gorm.DB
	serverConnector  *client.AgentServerClient
	userActionWriter useraction.Writer
	certsPath        string
}

func NewProtoService(
	db *gorm.DB,
	serverConnector *client.AgentServerClient,
	userActionWriter useraction.Writer,
	certsPath string,
) *ProtoService {
	return &ProtoService{
		db:               db,
		serverConnector:  serverConnector,
		userActionWriter: userActionWriter,
		certsPath:        certsPath,
	}
}

func (s *ProtoService) AggregateWSConnect(c *gin.Context) {
	sockID := c.Param("group_id")

	uaf := useraction.Fields{
		Domain:            "agent",
		ObjectType:        "agent",
		ObjectID:          sockID,
		ActionCode:        "interactive interaction",
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
	defer s.userActionWriter.WriteUserAction(c, uaf)

	serviceHash, err := getServiceHash(c)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, err)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		logger.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	name, err := storage.GetGroupName(iDB, sockID)
	if err == nil {
		uaf.ObjectDisplayName = name
	} else {
		logger.FromContext(c).WithError(err).Errorf("error finding group by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrAgentsNotFound, nil)
			return
		}
		response.Error(c, response.ErrInternal, err)
		return
	}

	sockType := c.GetString("cpt")
	if sockType != "aggregate" {
		logger.FromContext(c).Errorf("mismatch socket type to incoming token type")
		response.Error(c, response.ErrProtoSockMismatch, nil)
		return
	}

	wsConnectToVXServer(c, vxproto.Aggregate, sockID, sockType, s.certsPath, uaf)
}

func (s *ProtoService) BrowserWSConnect(c *gin.Context) {
	sockID := c.Param("agent_id")

	uaf := useraction.Fields{
		Domain:            "agent",
		ObjectType:        "agent",
		ObjectID:          sockID,
		ActionCode:        "interactive interaction",
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
	defer s.userActionWriter.WriteUserAction(c, uaf)

	serviceHash, err := getServiceHash(c)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, err)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		logger.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	name, err := storage.GetAgentName(iDB, sockID)
	if err == nil {
		uaf.ObjectDisplayName = name
	} else {
		logger.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrAgentsNotFound, nil)
			return
		}

		response.Error(c, response.ErrInternal, err)
		return
	}

	sockType := c.GetString("cpt")
	if sockType != "browser" {
		logger.FromContext(c).Errorf("mismatch socket type to incoming token type")
		response.Error(c, response.ErrProtoSockMismatch, nil)
		return
	}

	wsConnectToVXServer(c, vxproto.Browser, sockID, sockType, s.certsPath, uaf)
}

func (s *ProtoService) ExternalWSConnect(c *gin.Context) {
	sockID := c.Param("agent_id")
	uaf := useraction.Fields{
		Domain:            "agent",
		ObjectType:        "agent",
		ObjectID:          sockID,
		ActionCode:        "interactive interaction",
		ObjectDisplayName: useraction.UnknownObjectDisplayName,
	}
	defer s.userActionWriter.WriteUserAction(c, uaf)

	serviceHash, err := getServiceHash(c)
	if err != nil {
		logger.FromContext(c).WithError(err).Errorf("could not get service hash")
		response.Error(c, response.ErrInternal, err)
		return
	}
	iDB, err := s.serverConnector.GetDB(c, serviceHash)
	if err != nil {
		logger.FromContext(c).WithError(err).Error()
		response.Error(c, response.ErrInternalDBNotFound, err)
		return
	}

	name, err := storage.GetAgentName(iDB, sockID)
	if err == nil {
		uaf.ObjectDisplayName = name
	} else {
		logger.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, response.ErrAgentsNotFound, nil)
			return
		}
		response.Error(c, response.ErrInternal, err)
		return
	}

	sockType := c.GetString("cpt")
	if sockType != "external" {
		logger.FromContext(c).Errorf("mismatch socket type to incoming token type")
		response.Error(c, response.ErrProtoSockMismatch, nil)
		return
	}

	wsConnectToVXServer(c, vxproto.External, sockID, sockType, s.certsPath, uaf)
}
