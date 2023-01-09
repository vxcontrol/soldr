package vxproto

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"net/http"
	"reflect"
	"strings"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.7.0"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/tomasen/realip"

	vxcommonErrors "soldr/internal/errors"
	obs "soldr/internal/observability"
	"soldr/internal/vxproto/tunnel"
)

type InitConnectionInfo struct {
	IP string
}

type AgentConnectionInfo struct {
	ID         string
	GroupID    uint64
	AuthStatus string
}

type AgentInfoForIDFetcher struct {
	ID   string
	Type string
}

type AgentIDFetcher interface {
	GetAgentConnectionInfo(ctx context.Context, info *AgentInfoForIDFetcher) (*AgentConnectionInfo, error)
}

type ServerConnectionValidator interface {
	AgentIDFetcher

	CheckInitConnectionTLS(s *tls.ConnectionState) error
	CheckConnectionTLS(s *tls.ConnectionState) error

	OnInitConnect(
		ctx context.Context,
		tlsConnState *tls.ConnectionState,
		socket SyncWS,
		connInfo *InitConnectionInfo,
	) error
	OnConnect(
		ctx context.Context,
		tlsConnState *tls.ConnectionState,
		socket IAgentSocket,
		agentType AgentType,
		configurePackEncryptor func(c *tunnel.Config) error,
		configurePinger func(p Pinger),
	) error
}

type ConnectionValidatorFactory interface {
	NewValidator(version string) (ServerConnectionValidator, error)
}

// listenWS is only Server function
// TODO: here need to write of config argument description
func (vxp *vxProto) listenWS(
	ctx context.Context,
	config *ServerConfig,
	validatorFactory ConnectionValidatorFactory,
	logger *logrus.Entry,
) error {
	server, err := configureServer(config)
	if err != nil {
		return fmt.Errorf("failed to configure the server: %w", err)
	}
	server.Handler, err = vxp.configureRouter(config.API, validatorFactory, logger)
	if err != nil {
		return fmt.Errorf("failed to configure the server router: %w", err)
	}

	vxp.mutex.Lock()
	var closed bool
	var wg sync.WaitGroup
	closeChan := make(chan struct{})
	closeFunc := func() {
		if !closed {
			closed = true
			closeChan <- struct{}{}
		}
	}
	vxp.closers = append(vxp.closers, closeFunc)
	vxp.mutex.Unlock()

	wg.Add(1)
	go func() {
		defer wg.Done()

		<-closeChan

		vxp.mutex.Lock()
		defer vxp.mutex.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		server.Shutdown(ctx)
		cancel()

		for idx, closer := range vxp.closers {
			if reflect.DeepEqual(closer, closeFunc) {
				// It's possible because there used break in the bottom
				vxp.closers = append(vxp.closers[:idx], vxp.closers[idx+1:]...)
				break
			}
		}
	}()

	defer func() {
		closeFunc()
		wg.Wait()
	}()

	if server.TLSConfig == nil {
		return server.ListenAndServe()
	}
	return server.ListenAndServeTLS("", "")
}

func (vxp *vxProto) configureRouter(c ServerAPIVersionsConfig, validatorFactory ConnectionValidatorFactory, logger *logrus.Entry) (http.Handler, error) {
	policyManagerIterator, err := NewConnectionPolicyManagerIterator(c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connection policy manager factory: %w", err)
	}
	r := mux.NewRouter()
	for policyManagerIterator.Next() {
		managerWithVer, err := policyManagerIterator.GetCurrentManager()
		if err != nil {
			return nil, fmt.Errorf("failed to get a connection policy manager: %w", err)
		}
		if err := vxp.configureVersionHandlers(r, validatorFactory, managerWithVer.Manager, managerWithVer.Version, logger); err != nil {
			return nil, fmt.Errorf("failed to configurate the version handlers for version \"%s\": %w", managerWithVer.Version, err)
		}
	}
	return r, nil
}

type VXProtoCtxKey int

const (
	serverCTXKeyAgentConnInfo VXProtoCtxKey = iota + 1
	serverCTXKeyAgentType
	serverCTXKeyConnectionPolicy
)

func (vxp *vxProto) configureVersionHandlers(
	r *mux.Router,
	validatorFactory ConnectionValidatorFactory,
	connectionPolicyManager ConnectionPolicyManager,
	version string,
	logger *logrus.Entry,
) error {
	basePath := fmt.Sprintf("/api/%s/vxpws/", version)
	connValidator, err := validatorFactory.NewValidator(version)
	if err != nil {
		return fmt.Errorf("failed to get a connection validator: %w", err)
	}

	useTracer := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := obs.Observer.NewSpan(r.Context(), obs.SpanKindConsumer, "proto.handler")
			span.SetAttributes(semconv.NetAttributesFromHTTPRequest("tcp", r)...)
			span.SetAttributes(semconv.EndUserAttributesFromHTTPRequest(r)...)
			span.SetAttributes(attribute.Key("http.method").String(r.Method))
			span.SetAttributes(attribute.Key("http.path").String(r.URL.Path))
			r = r.WithContext(ctx)
			defer span.End()

			next.ServeHTTP(w, r)
		})
	}

	simple := r.PathPrefix(basePath).Subrouter()
	simple.Use(useTracer)
	simple.HandleFunc(
		"/agent/",
		func(w http.ResponseWriter, r *http.Request) {
			loggerCtx := logger.WithContext(r.Context())

			connectionPolicyType, err := connectionPolicyManager.GetConnectionPolicyType()
			if err != nil {
				loggerCtx.WithError(err).Warn("getting connection policy type failed")
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			if connectionPolicyType == EndpointConnectionPolicyBlock {
				loggerCtx.Warn("connection blocked")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			handleAgentInitConn(w, r, vxp, connValidator, loggerCtx)
		},
	)

	withConnInfo := r.PathPrefix(basePath).Subrouter()
	withConnInfo.Use(useTracer)
	withConnInfo.Use(
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				span := obs.Observer.SpanFromContext(r.Context())
				id, ok := mux.Vars(r)["id"]
				if !ok || len(id) == 0 {
					span.AddEvent("")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				span.SetAttributes(attribute.Key("span.conn_id").String(id))

				result, err := getConnectionPolicy(r.Context(), r.URL.Path, id, connValidator, connectionPolicyManager)
				if err != nil {
					loggerCtx := logger.WithContext(r.Context()).WithField("conn_id", id).WithError(err)
					if errors.Is(err, ErrEndpointBlocked) {
						loggerCtx.Warn("connection blocked")
						w.WriteHeader(http.StatusForbidden)
						return
					}
					if errors.Is(err, vxcommonErrors.ErrRecordNotFound) {
						loggerCtx.Warn("agent has not been authorized yet")
						w.WriteHeader(http.StatusForbidden)
						return
					}
					loggerCtx.Warn("getting connection policy failed")
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				r = r.WithContext(context.WithValue(r.Context(), serverCTXKeyAgentConnInfo, result.AgentConnectionInfo))
				r = r.WithContext(context.WithValue(r.Context(), serverCTXKeyAgentType, result.AgentType))
				r = r.WithContext(context.WithValue(r.Context(), serverCTXKeyConnectionPolicy, result.ConnectionPolicy))
				next.ServeHTTP(w, r)
			})
		},
	)
	handleConn := func(w http.ResponseWriter, r *http.Request) {
		if err := handleAgentConnection(w, r, vxp, connValidator, connectionPolicyManager); err != nil {
			logger.WithContext(r.Context()).WithError(err).Warn("failed to handle the agent connection")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}
	withConnInfo.HandleFunc("/agent/{id:[0-9a-z]+}/",
		func(w http.ResponseWriter, r *http.Request) {
			handleConn(w, r)
		})
	withConnInfo.HandleFunc("/aggregate/{id:[0-9a-z]+}/",
		func(w http.ResponseWriter, r *http.Request) {
			handleConn(w, r)
		})
	withConnInfo.HandleFunc("/browser/{id:[0-9a-z]+}/",
		func(w http.ResponseWriter, r *http.Request) {
			handleConn(w, r)
		})
	withConnInfo.HandleFunc("/external/{id:[0-9a-z]+}/",
		func(w http.ResponseWriter, r *http.Request) {
			handleConn(w, r)
		})
	return nil
}

func handleAgentConnection(
	w http.ResponseWriter,
	r *http.Request, vxp *vxProto,
	connValidator ServerConnectionValidator,
	connPolicyTypeGetter ConnectionPolicyTypeGetter,
) error {
	agentType, err := extractAgentType(r)
	if err != nil {
		return fmt.Errorf("failed to extract the agent type from the request context: %w", err)
	}
	connectionPolicy, err := extractConnectionPolicy(r)
	if err != nil {
		return fmt.Errorf("failed to extract the connection policy from the request context: %w", err)
	}
	connPolicyType, err := connPolicyTypeGetter.GetConnectionPolicyType()
	if err != nil {
		return fmt.Errorf("failed to get the connection policy type: %w", err)
	}
	isOnlyForUpgrade := false
	if connPolicyType == EndpointConnectionPolicyUpgrade {
		isOnlyForUpgrade = true
	}
	handleAgentWS(vxp, connValidator, connectionPolicy, isOnlyForUpgrade, agentType, w, r)
	return nil
}

type getConnectionPolicyMiddlewareResult struct {
	ConnectionPolicy    ConnectionPolicy
	AgentType           AgentType
	AgentConnectionInfo *AgentConnectionInfo
}

func getConnectionPolicy(
	ctx context.Context,
	urlPath string,
	id string,
	idFetcher AgentIDFetcher,
	policyManager ConnectionPolicyManager,
) (*getConnectionPolicyMiddlewareResult, error) {
	connType, err := getConnectionTypeFromURL(urlPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get the connection type from the URL path: %w", err)
	}
	if !policyManager.IsConnectionAllowed(connType) {
		return nil, fmt.Errorf("connection for the given type (%d) is forbidden: %w", connType, ErrEndpointBlocked)
	}
	fetchType := "agent"
	if connType == Aggregate {
		fetchType = "group"
	}
	ctxFetcher := &AgentInfoForIDFetcher{
		ID:   id,
		Type: fetchType,
	}
	connInfo, err := fetchConnectionInfo(ctx, idFetcher, ctxFetcher)
	if err != nil {
		return nil, fmt.Errorf("failed to get the info on the connecting: %w", err)
	}
	connPolicy, err := policyManager.GetConnectionPolicy(&ConnectionInfo{
		Agent:     connInfo,
		AgentType: connType,
		URLPath:   urlPath,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get connection policy: %w", err)
	}
	return &getConnectionPolicyMiddlewareResult{
		ConnectionPolicy:    connPolicy,
		AgentType:           connType,
		AgentConnectionInfo: connInfo,
	}, nil
}

func getConnectionTypeFromURL(url string) (AgentType, error) {
	urlParts := strings.Split(url, "/")
	if len(urlParts) != 7 {
		return 0, fmt.Errorf("the connection type cannot be extracted from the URL %s: too few url parts", url)
	}
	agentType := urlParts[4]
	switch agentType {
	case "agent":
		return VXAgent, nil
	case "aggregate":
		return Aggregate, nil
	case "browser":
		return Browser, nil
	case "external":
		return External, nil
	default:
		return 0, fmt.Errorf("an unknown connection type \"%s\" is passed", agentType)
	}
}

func extractAgentType(r *http.Request) (AgentType, error) {
	agentTypeIface := r.Context().Value(serverCTXKeyAgentType)
	if agentTypeIface == nil {
		return 0, fmt.Errorf("agent type not found in the request context")
	}
	agentType, ok := agentTypeIface.(AgentType)
	if !ok {
		return 0, fmt.Errorf("connection info object stored in the request is not of type *AgentConnectionInfo")
	}
	return agentType, nil
}

func fetchConnectionInfo(
	reqCtx context.Context,
	idFetcher AgentIDFetcher,
	ctxFetcher *AgentInfoForIDFetcher,
) (*AgentConnectionInfo, error) {
	connInfo, err := idFetcher.GetAgentConnectionInfo(reqCtx, ctxFetcher)
	if err != nil {
		return nil, fmt.Errorf("failed to get the connection info: %w", err)
	}
	return connInfo, nil
}

func extractConnectionPolicy(r *http.Request) (ConnectionPolicy, error) {
	connPolicyIface := r.Context().Value(serverCTXKeyConnectionPolicy)
	if connPolicyIface == nil {
		return nil, fmt.Errorf("connection policy info not found in the request context")
	}
	connPolicy, ok := connPolicyIface.(ConnectionPolicy)
	if !ok {
		return nil, fmt.Errorf("connection info object stored in the request is not of type ConnectionPolicy")
	}
	return connPolicy, nil
}

func handleAgentWS(
	vxp *vxProto,
	connValidator ServerConnectionValidator,
	connectionPolicy ConnectionPolicy,
	isOnlyForUpgrade bool,
	agentType AgentType,
	w http.ResponseWriter,
	r *http.Request,
) {
	log := logrus.WithContext(r.Context())
	socket, err := prepareAgentConnection(vxp, connValidator, connectionPolicy, isOnlyForUpgrade, agentType, w, r)
	if err != nil {
		vars := mux.Vars(r)
		log.Errorf("failed to prepare agent connection: %s: %s", vars["id"], err.Error())
		return
	}
	defer func() {
		vxp.isClosedMux.RLock()
		defer vxp.isClosedMux.RUnlock()
		if vxp.isClosed {
			return
		}
		vxp.delAgent(r.Context(), socket)
	}()

	// Run ping sender
	socket.pinger.Start(r.Context(), socket.ping)
	defer func() {
		socket.pinger.Stop(r.Context())
	}()

	// Read messages before connection will be closed
	for {
		select {
		case <-r.Context().Done():
			return
		default:
			if err := socket.recvPacket(r.Context()); err != nil {
				// TODO(SSH): need to add logging here
				return
			}
		}
	}
}

func prepareAgentConnection(
	vxp *vxProto,
	connValidator ServerConnectionValidator,
	connectionPolicy ConnectionPolicy,
	isOnlyForUpgrade bool,
	agentType AgentType,
	w http.ResponseWriter,
	r *http.Request,
) (*agentSocket, error) {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()

	if err := connValidator.CheckConnectionTLS(r.TLS); err != nil {
		return nil, fmt.Errorf("connection TLS check has failed: %w", err)
	}
	// Deny all but HTTP GET
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return nil, fmt.Errorf("method %s is not allowed", r.Method)
	}

	// Upgrade connection to Websocket
	upgrader := websocket.Upgrader{
		// Use experimental feature
		EnableCompression: true,
		// TODO: May need to check the Origin but for now so
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "Error Upgrading to websockets", 400)
		return nil, fmt.Errorf("failed to upgrade to websockets: %w", err)
	}

	// Execute callback Connect if it was avalibale
	vars := mux.Vars(r)
	tokenType := agentType
	if tokenType == VXAgent {
		tokenType = VXServer
	}
	token, err := vxp.NewToken(vars["id"], tokenType)
	if err != nil {
		http.Error(w, "Internal error", 500)
		return nil, fmt.Errorf("failed to make source token: %w", err)
	}
	socket := &agentSocket{
		isOnlyForUpgrade: isOnlyForUpgrade,
		id:               vars["id"],
		ip:               getRealAddr(r),
		src:              token,
		at:               agentType,
		auth:             &AuthenticationData{},
		connectionPolicy: connectionPolicy,
		IConnection:      NewWSConnection(ws, false),
		IVaildator:       vxp,
		IMMInformator:    vxp,
		IProtoStats:      vxp,
		IProtoIO:         vxp,
	}
	if _, ok := vars["id"]; ok && vxp.IMainModule != nil {
		if err := connValidator.OnConnect(
			r.Context(),
			r.TLS,
			socket,
			agentType,
			getInitTunnelFn(socket),
			getConfigurePingerFn(socket),
		); err != nil {
			return nil, fmt.Errorf("setup connection error: %w", err)
		}

		// Register new agent
		if !vxp.addAgent(r.Context(), socket) {
			return nil, fmt.Errorf("setup connection error: %w", err)
		}
	} else {
		http.Error(w, "Error Connection setup", http.StatusForbidden)
		return nil, fmt.Errorf("setup connection error: %w", err)
	}
	return socket, nil
}

func getInitTunnelFn(socket *agentSocket) func(c *tunnel.Config) error {
	return func(c *tunnel.Config) error {
		var err error
		socket.packEncrypter, err = tunnel.NewPackEncrypter(c)
		if err != nil {
			return fmt.Errorf("failed to initialize a pack encryptor: %w", err)
		}
		return nil
	}
}

func getConfigurePingerFn(socket *agentSocket) func(p Pinger) {
	return func(p Pinger) {
		socket.pinger = p
	}
}

func isRealAddr(saddr string) bool {
	if strings.HasPrefix(saddr, "10.") || strings.HasPrefix(saddr, "192.168.") ||
		strings.HasPrefix(saddr, "172.") || strings.HasPrefix(saddr, "127.") ||
		strings.HasPrefix(saddr, "169.254.") || strings.HasPrefix(saddr, "::1") {
		return false
	}

	return true
}

func getRealAddr(r *http.Request) string {
	remoteAddrXFF := realip.FromRequest(r)
	remoteAddr := r.RemoteAddr
	if remoteAddrXFF != "" && !isRealAddr(remoteAddr) && isRealAddr(remoteAddrXFF) {
		_, port, _ := net.SplitHostPort(remoteAddr)
		remoteAddr = net.JoinHostPort(remoteAddrXFF, port)
	}

	return remoteAddr
}

func handleAgentInitConn(
	w http.ResponseWriter,
	r *http.Request,
	vxp *vxProto,
	connValidator ServerConnectionValidator,
	logger *logrus.Entry,
) {
	if err := connValidator.CheckInitConnectionTLS(r.TLS); err != nil {
		logger.WithError(err).Error("agent init connection: failed to upgrade to a WebSocket connection")
		http.Error(w, "Error Upgrading to websockets", http.StatusForbidden)
		return
	}
	// Upgrade connection to Websocket
	upgrader := websocket.Upgrader{
		// Use experimental feature
		EnableCompression: true,
		// TODO: May need to check the Origin but for now so
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	wsConn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logger.WithError(err).Error("agent init connection: failed to upgrade to a WebSocket connection")
		http.Error(w, "Error Upgrading to websockets", 400)
		return
	}

	const errMsgToSend = "setup connection error"
	if vxp.IMainModule == nil {
		logger.Error("vxproto field IMainModule is nil")
		http.Error(w, errMsgToSend, http.StatusForbidden)
		return
	}
	ws, err := wrapWSConnection(r.Context(), logger, wsConn)
	if err != nil {
		logger.WithError(err).Error("failed to wrap the WS connection")
		http.Error(w, errMsgToSend, http.StatusInternalServerError)
		return
	}
	agentIP := getRealAddr(r)
	if err := connValidator.OnInitConnect(r.Context(), r.TLS, ws, &InitConnectionInfo{
		IP: agentIP,
	}); err != nil {
		_ = ws.Close(r.Context())
		logger.WithError(err).Error("on init connect failed")
		return
	}
	_ = ws.Close(r.Context())
}

func configureServer(config *ServerConfig) (*http.Server, error) {
	var tlsConfig *tls.Config
	if config == nil {
		return nil, fmt.Errorf("passed server configuration is nil")
	}
	scheme, addr, err := getServerAddr(config.Host)
	if err != nil {
		return nil, fmt.Errorf("failed to get a server address: %w", err)
	}
	if scheme == "wss" && config.TLSConfig != nil {
		tlsConfig = config.TLSConfig
		oldVerifyPeerCert := tlsConfig.VerifyPeerCertificate
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			if oldVerifyPeerCert != nil {
				return oldVerifyPeerCert(rawCerts, verifiedChains)
			}
			return nil
		}
	}
	return &http.Server{
		Addr:              addr,
		TLSConfig:         tlsConfig,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      5 * time.Second,
		IdleTimeout:       60 * time.Second,
		ReadHeaderTimeout: 2 * time.Second,
	}, nil
}

func getServerAddr(host string) (string, string, error) {
	u, err := getURLFromHost(host)
	if err != nil {
		return "", "", err
	}
	return u.Scheme, u.Host, nil
}

func wrapWSConnection(ctx context.Context, logger *logrus.Entry, ws *websocket.Conn) (*syncWS, error) {
	return NewSyncWS(
		ctx,
		logger,
		ws,
		&SyncWSConfig{
			ReadTimeout: defaultReadTimeout,
		},
	)
}
