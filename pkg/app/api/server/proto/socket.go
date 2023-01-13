package proto

import (
	"bytes"
	"context"
	"fmt"

	"soldr/pkg/protoagent"
	"soldr/pkg/vxproto"
	"soldr/pkg/vxproto/tunnel"
)

// Constants for ping sender functionality
const constPingPacket string = "PING"

type IWSConn interface {
	Close(context.Context) error
	Read(context.Context) ([]byte, error)
	Write(context.Context, []byte) error
}

// socket is struct that used for registration of connection data
type socket struct {
	id            string
	gid           string
	ip            string
	src           string
	ver           string
	connected     bool
	at            vxproto.AgentType
	info          *protoagent.Information
	authReq       *protoagent.AuthenticationRequest
	authResp      *protoagent.AuthenticationResponse
	packEncrypter tunnel.PackEncryptor
	pinger        vxproto.Pinger
	vxproto.IWSConnection
	vxproto.IVaildator
	vxproto.IMMInformator
}

// GetPublicInfo is function which provided only public information about agent
func (s *socket) GetPublicInfo() *vxproto.AgentInfo {
	return &vxproto.AgentInfo{
		ID:   s.id,
		GID:  s.gid,
		IP:   s.ip,
		Src:  s.GetSource(),
		Dst:  s.GetDestination(),
		Ver:  s.ver,
		Type: s.at,
		Info: s.info,
	}
}

// GetAgentID is function to get current agent ID
func (s *socket) GetAgentID() string {
	return s.id
}

// GetGroupID is function to get current agent group ID
func (s *socket) GetGroupID() string {
	return s.gid
}

// SetGroupID is function to update agent group id on agent connecting
func (s *socket) SetGroupID(gid string) {
	s.gid = gid
}

// GetSource is function which return source token
func (s *socket) GetSource() string {
	return s.src
}

// SetSource is function which storing source token
func (s *socket) SetSource(src string) {
	s.src = src
}

// SetVersion is function which storing version of agent/server on other side
func (s *socket) SetVersion(ver string) {
	s.ver = ver
}

// SetInfo is function which storing information about agent
func (s *socket) SetInfo(info *protoagent.Information) {
	s.info = info
}

// SetAuthReq is function which storing authentication request about agent after handshake
func (s *socket) SetAuthReq(req *protoagent.AuthenticationRequest) {
	s.authReq = req
}

// SetAuthResp is function which storing authentication response about agent after handshake
func (s *socket) SetAuthResp(resp *protoagent.AuthenticationResponse) {
	s.authResp = resp
}

// GetDestination is function which return destination token
func (s *socket) GetDestination() string {
	if s.authResp != nil {
		// This code used on agent side
		if s.authResp.GetAtoken() == s.src {
			return s.authResp.GetStoken()
		}
		// This code used on server side
		if s.authResp.GetStoken() == s.src {
			return s.authResp.GetAtoken()
		}
	}

	return ""
}

func (s *socket) Read(ctx context.Context) (p []byte, err error) {
	for {
		if s.IWSConnection == nil {
			return nil, fmt.Errorf("ws connection is not initialized")
		}

		packetData, err := s.IWSConnection.Read(ctx)
		if err != nil {
			return nil, err
		}

		if s.connected {
			packetData, err = s.packEncrypter.Decrypt(packetData)
			if err != nil {
				return nil, fmt.Errorf("failed to decrypt the received packet: %w", err)
			}
		}

		if bytes.HasPrefix(packetData, []byte(constPingPacket)) {
			if err := s.pinger.Process(ctx, packetData[4:]); err != nil {
				return nil, fmt.Errorf("failed to process the received ping: %w", err)
			}
			continue
		}

		return packetData, nil
	}
}

func (s *socket) Write(ctx context.Context, data []byte) error {
	if len(data) == 4 && string(data) == constPingPacket {
		return nil
	}
	if s.IWSConnection == nil {
		return fmt.Errorf("ws connection is not initialized")
	}

	var err error
	if s.connected {
		data, err = s.packEncrypter.Encrypt(data)
		if err != nil {
			return fmt.Errorf("failed to encrypt the packet data: %w", err)
		}
	}

	if err = s.IWSConnection.Write(ctx, data); err != nil {
		return fmt.Errorf("failed sending of packet: %w", err)
	}
	return nil
}

func (s *socket) Close(ctx context.Context) error {
	if err := s.pinger.Stop(ctx); err != nil {
		return err
	}
	if s.IWSConnection != nil {
		return s.IWSConnection.Close(ctx)
	}
	return nil
}

func (s *socket) ping(ctx context.Context, nonce []byte) error {
	nonce = append([]byte(constPingPacket), nonce...)
	if err := s.Write(ctx, nonce); err != nil {
		return fmt.Errorf("failed sending of ping packet: %w", err)
	}
	return nil
}
