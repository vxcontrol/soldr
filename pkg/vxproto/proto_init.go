package vxproto

import (
	"context"
	"crypto/tls"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"

	"soldr/pkg/system"
)

func (vxp *vxProto) InitConnection(ctx context.Context, connValidator AgentConnectionValidator, config *ClientInitConfig, logger *logrus.Entry) error {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()
	if vxp.isClosed {
		return fmt.Errorf("failed to connect to the server: vxProto is already closed")
	}
	if ctx.Err() != nil {
		return fmt.Errorf("failed to connect to the server: the operation was cancelled")
	}
	agentInfo, err := system.GetAgentInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the agent info: %w", err)
	}
	conn, err := openAgentSocketToInitConnection(ctx, logger, config)
	if err != nil {
		return err
	}
	if err := connValidator.OnInitConnect(ctx, conn, agentInfo); err != nil {
		_ = conn.Close(ctx)
		return fmt.Errorf("failed to perform the initial connection: %w", err)
	}
	return nil
}

type CommonConfig struct {
	Host      string
	TLSConfig *tls.Config
}

type ServerAPIVersionsConfig map[string]*ServerAPIConfig

type ServerAPIExternalConfig struct {
	Version          string `json:"-"`
	ConnectionPolicy string `json:"connection_policy"`
}

func (c *ServerAPIExternalConfig) GetServerAPIConfig() (*ServerAPIConfig, error) {
	conf := &ServerAPIConfig{
		Version: c.Version,
	}
	if err := conf.ConnectionPolicy.FromString(c.ConnectionPolicy); err != nil {
		return nil, err
	}
	return conf, nil
}

type ServerAPIConfig struct {
	Version          string
	ConnectionPolicy EndpointConnectionPolicy
}

type ServerConfig struct {
	*CommonConfig
	API ServerAPIVersionsConfig
}

type ClientInitConfig struct {
	*CommonConfig
	Type            string
	ProtocolVersion string
}

type SyncWS interface {
	Read(context.Context) ([]byte, error)
	Write(context.Context, []byte) error
	Close(context.Context) error
	Done() <-chan struct{}
}

type SyncWSConfig struct {
	SendPingFrequency time.Duration
	ReadTimeout       time.Duration
}

type syncWS struct {
	done         chan struct{}
	ws           IWSConnection
	buf          chan []byte
	isClosing    bool
	isClosingMux *sync.Mutex
	logger       *logrus.Entry
}

func NewSyncWS(ctx context.Context, logger *logrus.Entry, ws *websocket.Conn, config *SyncWSConfig) (*syncWS, error) {
	if ws == nil {
		return nil, fmt.Errorf("passed websocket is nil")
	}
	if config == nil {
		return nil, fmt.Errorf("passed SyncWS configuration is nil")
	}

	s := &syncWS{
		done:         make(chan struct{}),
		ws:           NewWSConnection(ws, true),
		logger:       logger,
		buf:          make(chan []byte, 1),
		isClosing:    false,
		isClosingMux: &sync.Mutex{},
	}

	if err := ws.SetReadDeadline(time.Now().Add(config.ReadTimeout)); err != nil {
		return nil, fmt.Errorf("failed to set the read timeout: %w", err)
	}

	go s.readMessages()
	go func() {
		select {
		case <-ctx.Done():
			s.Close(context.TODO())
		case <-s.done:
		}
	}()

	configurePing(s.done, logger, s.ws, config.ReadTimeout, config.SendPingFrequency)
	return s, nil
}

func configurePing(done <-chan struct{}, logger *logrus.Entry, ws IWSConnection, readTimeout time.Duration, pingFrequency time.Duration) {
	if pingFrequency == 0 {
		handleWSPing(ws, logger, readTimeout)
		return
	}
	configureWSPing(done, logger, ws, readTimeout, pingFrequency)
}

func configureWSPing(done <-chan struct{}, logger *logrus.Entry, ws IWSConnection, readTimeout time.Duration, pingFrequency time.Duration) {
	go func() {
		ticker := time.NewTicker(pingFrequency)
		for {
			select {
			case <-ticker.C:
				if err := ws.WriteMessage(websocket.PingMessage, nil); err != nil {
					logger.WithError(err).Warn("failed to send a PING message")
				}
			case <-done:
				ticker.Stop()
				return
			}
		}
	}()
	ws.SetPongHandler(func(_ string) error {
		if err := resetReadDeadline(ws, readTimeout); err != nil {
			logger.WithError(err).Error("an error occurred in PONG handler")
		}
		return nil
	})
}

func handleWSPing(ws IWSConnection, logger *logrus.Entry, readTimeout time.Duration) {
	ws.SetPingHandler(func(data string) (err error) {
		defer func() {
			if err != nil {
				logrus.WithError(err).Error("an error occurred in PING handler")
			}
		}()
		if err = resetReadDeadline(ws, readTimeout); err != nil {
			return
		}
		if err = ws.WriteMessage(websocket.PongMessage, []byte(data)); err != nil {
			err = fmt.Errorf("failed to send a PONG message: %w", err)
			return
		}
		return nil
	})
}

func resetReadDeadline(ws IWSConnection, readTimeout time.Duration) error {
	if err := ws.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
		return fmt.Errorf("failed to reset the WS read deadline: %w", err)
	}
	return nil
}

func (s *syncWS) readMessages() {
	defer func() {
		_ = s.Close(context.TODO())
		close(s.buf)
	}()
	for {
		ctx := context.Background()
		msg, err := s.ws.Read(ctx)
		if err != nil {
			closeErr, isCloseErr := err.(*websocket.CloseError)
			if isCloseErr {
				if closeErr.Code == websocket.CloseNormalClosure {
					return
				}
				s.logger.WithContext(ctx).WithError(err).WithField("component", "reader_messages").
					Error("an unexpected close error occurred while reading messages")
				return
			}
			s.logger.WithContext(ctx).WithError(err).WithField("component", "reader_messages").
				Error("an unexpected error occurred while reading messages")
			return
		}
		s.buf <- msg
	}
}

func (s *syncWS) Close(ctx context.Context) (err error) {
	defer func() {
		if err != nil {
			s.logger.WithError(err).Error("error occurred during the syncWS connection closing")
		}
	}()
	if !s.startClosing() {
		return nil
	}
	close(s.done)
	if err := s.ws.Close(ctx); err != nil {
		s.logger.WithError(err).Error("failed to properly close the websocket connection")
		return err
	}
	return nil
}

func (s *syncWS) startClosing() bool {
	s.isClosingMux.Lock()
	defer s.isClosingMux.Unlock()
	if s.isClosing {
		return false
	}
	s.isClosing = true
	return true
}

func (s *syncWS) Read(ctx context.Context) ([]byte, error) {
	select {
	case data, ok := <-s.buf:
		if !ok {
			return nil, fmt.Errorf("read channel is closed")
		}
		return data, nil
	case <-ctx.Done():
		return nil, fmt.Errorf("the read context has signaled cancellation: %w", ctx.Err())
	}
}

func (s *syncWS) Write(ctx context.Context, msg []byte) error {
	if err := s.ws.Write(ctx, msg); err != nil {
		return fmt.Errorf("failed to write the message: %w", err)
	}
	return nil
}

func (s *syncWS) Done() <-chan struct{} {
	return s.done
}
