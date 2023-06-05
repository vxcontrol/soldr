package vxproto

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"soldr/pkg/protoagent"
	"soldr/pkg/vxproto/tunnel"
)

// IAgentSocket is main interface for Agent Socket integration
type IAgentSocket interface {
	GetPublicInfo() *AgentInfo
	GetAgentID() string
	GetGroupID() string
	SetGroupID(gid string)
	GetSource() string
	SetSource(src string)
	GetDestination() string
	SetVersion(string)
	SetInfo(info *protoagent.Information)
	SetAuthReq(req *protoagent.AuthenticationRequest)
	SetAuthResp(resp *protoagent.AuthenticationResponse)
	IConnection
	IVaildator
	IMMInformator
}

// IConnection is interface that using in socket communication
type IConnection interface {
	Read(ctx context.Context) (data []byte, err error)
	Write(ctx context.Context, data []byte) error
	Close(ctx context.Context) error
}

// AgentType is client type which will be VXAgent or Browser
type AgentType int32

// Enumerate agent types
const (
	VXAgent   AgentType = 0
	Browser   AgentType = 1
	External  AgentType = 2
	VXServer  AgentType = 3
	Aggregate AgentType = 4
)

// Constants for ping sender functionality
const (
	delayPingSender int    = 5000
	constPingPacket string = "PING"
)

var agentTypeName = map[int32]string{
	0: "VXAgent",
	1: "Browser",
	2: "External",
	3: "VXServer",
	4: "Aggregate",
}

var agentTypeValue = map[string]int32{
	"VXAgent":   0,
	"Browser":   1,
	"External":  2,
	"VXServer":  3,
	"Aggregate": 4,
}

func (at AgentType) String() string {
	if str, ok := agentTypeName[int32(at)]; ok {
		return str
	}

	return "unknown"
}

// MarshalJSON using for convert from AgentType to JSON
func (at AgentType) MarshalJSON() ([]byte, error) {
	if str, ok := agentTypeName[int32(at)]; ok {
		return []byte(`"` + str + `"`), nil
	}

	return nil, fmt.Errorf("cannot marshal AgentType")
}

// UnmarshalJSON using for convert from JSON to AgentType
func (at *AgentType) UnmarshalJSON(data []byte) error {
	str := strings.Trim(string(data), `"`)

	if name, ok := agentTypeValue[str]; ok {
		*at = AgentType(name)
		return nil
	}

	return fmt.Errorf("cannot unmarshal AgentType")
}

// AuthenticationData is struct which contains information about authentication
type AuthenticationData struct {
	req  *protoagent.AuthenticationRequest
	resp *protoagent.AuthenticationResponse
}

// AgentInfo is struct which contains only public information about agent
type AgentInfo struct {
	IsOnlyForUpgrade bool                    `json:"-"`
	ID               string                  `json:"id"`
	GID              string                  `json:"gid"`
	IP               string                  `json:"ip"`
	Src              string                  `json:"src"`
	Dst              string                  `json:"dst"`
	Ver              string                  `json:"ver"`
	Type             AgentType               `json:"type"`
	Info             *protoagent.Information `json:"-"`
}

const (
	PingerInterval time.Duration = time.Second * 20
	PingerTimeout  time.Duration = PingerInterval * 6
)

type Pinger interface {
	Start(ctx context.Context, ping func(ctx context.Context, nonce []byte) error) error
	Process(ctx context.Context, pingData []byte) error
	Stop(ctx context.Context) error
}

// agentSocket is struct that used for registration of connection data
type agentSocket struct {
	isOnlyForUpgrade bool
	id               string
	gid              string
	ip               string
	src              string
	ver              string
	at               AgentType
	info             *protoagent.Information
	auth             *AuthenticationData
	packEncrypter    tunnel.PackEncryptor
	pinger           Pinger
	connectionPolicy ConnectionPolicy
	IConnection
	IVaildator
	IMMInformator
	IProtoStats
	IProtoIO
}

// GetPublicInfo is function which provided only public information about agent
func (as *agentSocket) GetPublicInfo() *AgentInfo {
	return &AgentInfo{
		IsOnlyForUpgrade: as.isOnlyForUpgrade,
		ID:               as.id,
		GID:              as.gid,
		IP:               as.ip,
		Src:              as.GetSource(),
		Dst:              as.GetDestination(),
		Ver:              as.ver,
		Type:             as.at,
		Info:             as.info,
	}
}

// GetAgentID is function to get current agent ID
func (as *agentSocket) GetAgentID() string {
	return as.id
}

// GetGroupID is function to get current agent group ID
func (as *agentSocket) GetGroupID() string {
	return as.gid
}

// SetGroupID is function to update agent group id on agent connecting
func (as *agentSocket) SetGroupID(gid string) {
	as.gid = gid
}

// GetSource is function which return source token
func (as *agentSocket) GetSource() string {
	return as.src
}

// SetSource is function which storing source token
func (as *agentSocket) SetSource(src string) {
	as.src = src
}

// SetVersion is function which storing version of agent/server on other side
func (as *agentSocket) SetVersion(ver string) {
	as.ver = ver
}

// SetInfo is function which storing information about agent
func (as *agentSocket) SetInfo(info *protoagent.Information) {
	as.info = info
}

// SetAuthReq is function which storing authentication request about agent after handshake
func (as *agentSocket) SetAuthReq(req *protoagent.AuthenticationRequest) {
	as.auth.req = req
}

// SetAuthResp is function which storing authentication response about agent after handshake
func (as *agentSocket) SetAuthResp(resp *protoagent.AuthenticationResponse) {
	as.auth.resp = resp
}

// GetDestination is function which return destination token
func (as *agentSocket) GetDestination() string {
	if as.auth != nil && as.auth.resp != nil {
		// This code used on agent side
		if as.auth.resp.GetAtoken() == as.src {
			return as.auth.resp.GetStoken()
		}
		// This code used on server side
		if as.auth.resp.GetStoken() == as.src {
			return as.auth.resp.GetAtoken()
		}
	}

	return ""
}

// recvPacket is function for serving packet to vxproto (hub) logic through callback
// Result is the success of packet processing otherwise will raise error
func (as *agentSocket) recvPacket(ctx context.Context) error {
	packetData, err := as.Read(ctx)
	if err != nil {
		return err
	}
	as.incStats(recvNetBytes, int64(len(packetData)))

	packetData, err = as.packEncrypter.Decrypt(packetData)
	if err != nil {
		return fmt.Errorf("failed to decrypt the received packet: %w", err)
	}
	as.incStats(recvPayloadBytes, int64(len(packetData)))
	as.incStats(recvNumPackets, 1)

	if bytes.HasPrefix(packetData, []byte(constPingPacket)) {
		if err := as.pinger.Process(ctx, packetData[4:]); err != nil {
			return fmt.Errorf("failed to process the received ping: %w", err)
		}
		return nil
	}

	packet, err := (&Packet{}).fromBytesPB(packetData)
	if err != nil {
		return err
	}

	err = as.IProtoIO.recvPacket(ctx, packet)
	if err != nil {
		return err
	}

	return nil
}

// sendPacket is function for sending packet to other side
// Result is the success of packet sending otherwise will raise error
func (as *agentSocket) sendPacket(ctx context.Context, packet *Packet) error {
	packetData, err := packet.toBytesPB()
	if err != nil {
		return err
	}
	as.incStats(sendPayloadBytes, int64(len(packetData)))

	packetData, err = as.packEncrypter.Encrypt(packetData)
	if err != nil {
		return fmt.Errorf("failed to encrypt the packet data: %w", err)
	}
	as.incStats(sendNetBytes, int64(len(packetData)))

	if err := as.Write(ctx, packetData); err != nil {
		return fmt.Errorf("failed sending of packet: %w", err)
	}
	as.incStats(sendNumPackets, 1)

	return nil
}

func (as *agentSocket) ping(ctx context.Context, nonce []byte) error {
	nonce = append([]byte(constPingPacket), nonce...)
	as.incStats(sendPayloadBytes, int64(len(nonce)))

	packetData, err := as.packEncrypter.Encrypt(nonce)
	if err != nil {
		return fmt.Errorf("failed to encrypt the packet data: %w", err)
	}
	as.incStats(sendNetBytes, int64(len(packetData)))

	if err := as.Write(ctx, packetData); err != nil {
		return fmt.Errorf("failed sending of ping packet: %w", err)
	}
	as.incStats(sendNumPackets, 1)

	return nil
}
