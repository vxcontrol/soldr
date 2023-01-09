package vxproto

import (
	"bytes"
	"context"
	"crypto/cipher"
	"crypto/des"
	"crypto/md5"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash/crc32"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/sirupsen/logrus"

	"soldr/internal/storage"
	"soldr/internal/system"
	"soldr/internal/vxproto/tunnel"
)

const mainModuleName = "main"

// IVXProto is main interface for vxproto package
type IVXProto interface {
	Close(ctx context.Context) error
	IStats
	IAgents
	IModules
	INetProto
}

// INetProto is network interface for client and server implementation
type INetProto interface {
	Connect(
		ctx context.Context,
		config *ClientConfig,
		connValidator AgentConnectionValidator,
		tunnelEncryptor tunnel.PackEncryptor,
	) error
	InitConnection(
		ctx context.Context,
		connValidator AgentConnectionValidator,
		config *ClientInitConfig,
		logger *logrus.Entry,
	) error
	Listen(
		ctx context.Context,
		config *ServerConfig,
		validatorFactory ConnectionValidatorFactory,
		logger *logrus.Entry,
	) error
}

// IAgents is vxproto interface to get and set agents objects
type IAgents interface {
	GetAgentsCount() int
	GetAgentList() map[string]IAgentSocket
	GetAgentBySrc(src string) IAgentSocket
	GetAgentByDst(dst string) IAgentSocket
	DropAgent(ctx context.Context, aid string)
	MoveAgent(ctx context.Context, aid, gid string)
}

// IModules is vxproto interface to get and set modules objects
type IModules interface {
	NewModule(name, gid string) IModuleSocket
	GetModule(name, gid string) IModuleSocket
	AddModule(iasocket IModuleSocket) bool
	DelModule(iasocket IModuleSocket) bool
}

// IRouter is router interface for shared modules communication
type IRouter interface {
	GetRoutes() map[string]string
	GetRoute(dst string) string
	AddRoute(dst, src string) error
	DelRoute(dst string) error
}

// IProtoIO is internal interface for callbacks which will use as IO in agent and module
type IProtoIO interface {
	recvPacket(ctx context.Context, packet *Packet) error
	sendPacket(ctx context.Context, packet *Packet) error
}

// IProtoStats is internal interface for updating statistics of using proto public interfaces
type IProtoStats interface {
	incStats(m metricType, val int64)
}

// IProtoStats is interface for getting collected statistics of using proto public interfaces
type IStats interface {
	DumpStats() (map[string]float64, error)
}

// IDefaultReceiver is interface for receiver in main module for control data
type IDefaultReceiver interface {
	DefaultRecvPacket(ctx context.Context, packet *Packet) error
}

// IIMC is interface for inter modules communication
type IIMC interface {
	HasIMCTokenFormat(token string) bool
	HasIMCTopicFormat(topic string) bool
	GetIMCModule(token string) IModuleSocket
	GetIMCTopic(topic string) ITopicInfo
	MakeIMCToken(name, gid string) string
	MakeIMCTopic(name, gid string) string
	SubscribeIMCToTopic(name, gid, token string) bool
	UnsubscribeIMCFromTopic(name, gid, token string) bool
	UnsubscribeIMCFromAllTopics(token string) bool
	GetIMCTopics() []string
	GetIMCGroupIDs() []string
	GetIMCModuleIDs() []string
	GetIMCGroupIDsByMID(mid string) []string
	GetIMCModuleIDsByGID(gid string) []string
}

// ITopicInfo is an interface for an object that can give topic registration information
type ITopicInfo interface {
	GetName() string
	GetGroupID() string
	GetSubscriptions() []string
}

// IAgentValidator is interface for validator in main module for agent connection
type IAgentValidator interface {
	HasAgentInfoValid(ctx context.Context, iasocket IAgentSocket) error
}

// IMMInformator is interface for getting environment information about main module
type IMMInformator interface {
	GetVersion() string
}

// IMainModule is interface for implementation common API server or agent
type IMainModule interface {
	IDefaultReceiver
	IAgentValidator
	IMMInformator
}

// ITokenVaildator is interface for validate token information on handshake
type ITokenVaildator interface {
	HasTokenValid(token string, agentID string, agentType AgentType) bool
	HasTokenCRCValid(token string) bool
	NewToken(agentID string, agentType AgentType) (string, error)
}

// IVaildator is interface for validate agnet and token information on handshake
type IVaildator interface {
	IAgentValidator
	ITokenVaildator
}

// applyPacketCb is a callback to deferred delivery packet after global mutex release
type applyPacketCb = func() error

const (
	// DelayHandleDeferredPacket is time period for sleep before handle deferred packet (ms)
	DelayHandleDeferredPacket = 100
	// MaxSizePacketQueue is maximum of size received or sent packets queue
	MaxSizePacketQueue = 5000
	// DeferredPacketTTLSeconds is time to live of deferred packet in seconds
	DeferredPacketTTLSeconds = 60
	// MaxFilePacketChunkSize is a max size data field of pb packet structure (bytes)
	MaxFilePacketChunkSize = 100 * 1024
	// FileStorePrefix is a directory prefix in the temp folder to control file list
	FileStorePrefix = "vx-store"
	// FileStoreRetentionTTL is a default time to live for each file in the vx-store (7 days)
	FileStoreRetentionTTL = time.Hour * 24 * 7
)

// vxProto struct is main object which will be use for Client and Server logics
type vxProto struct {
	modules     map[string]map[string]*moduleSocket
	agents      map[string]*agentSocket
	routes      map[string]string
	topics      map[string]*topicInfo
	mutex       *sync.RWMutex
	isClosed    bool
	isClosedMux *sync.RWMutex
	tokenKey    []byte
	closers     []func()
	rpqueue     []*Packet
	spqueue     []*Packet
	mxqueue     *sync.Mutex
	wgqueue     sync.WaitGroup
	ProtoStats
	IMainModule
}

type topicInfo struct {
	name          string
	groupID       string
	subscriptions []string
}

type ClientConfig struct {
	ClientInitConfig
	ID    string
	Token string
}

// New is function which constructed vxProto object
func New(mmodule IMainModule) (IVXProto, error) {
	var (
		err      error
		tokenKey []byte
	)
	if tokenKey, err = hex.DecodeString(system.MakeAgentID()); err != nil {
		return nil, fmt.Errorf("failed to get a token key: %w", err)
	}
	tokenKey = append(tokenKey, tokenKey[:8]...)

	// VXStore files rotation
	vxstore, err := storage.NewFS()
	if err != nil {
		return nil, fmt.Errorf("failed to create a vxstore handler: %w", err)
	}
	tempDir := filepath.Join(os.TempDir(), "vx-store")
	if err := os.MkdirAll(tempDir, 0700|os.ModeSetgid); err != nil {
		return nil, fmt.Errorf("failed to create a vx-store directory %s: %w", tempDir, err)
	}
	files, err := vxstore.ListDir(tempDir)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the existing files from the vx-store directory %s: %w", tempDir, err)
	}
	for file, info := range files {
		elapsed := time.Since(info.ModTime())
		if elapsed > FileStoreRetentionTTL {
			filePath := filepath.Join(tempDir, file)
			if err := vxstore.RemoveFile(filePath); err != nil {
				logrus.Warnf("failed to remove the file %s from the vx-server directory: %v", filePath, err)
			}
		}
	}

	return &vxProto{
		IMainModule: mmodule,
		modules:     make(map[string]map[string]*moduleSocket),
		agents:      make(map[string]*agentSocket),
		routes:      make(map[string]string),
		topics:      make(map[string]*topicInfo),
		mutex:       &sync.RWMutex{},
		isClosedMux: &sync.RWMutex{},
		isClosed:    false,
		mxqueue:     &sync.Mutex{},
		tokenKey:    tokenKey,
	}, nil
}

func (vxp *vxProto) runPQueue(handle func(ctx context.Context)) func() {
	vxp.mutex.Lock()
	var closed bool
	closeChan := make(chan struct{})
	closeFunc := func() {
		if !closed {
			closed = true
			closeChan <- struct{}{}
		}
	}
	vxp.closers = append(vxp.closers, closeFunc)
	vxp.mutex.Unlock()

	vxp.wgqueue.Add(1)
	go func() {
		defer vxp.wgqueue.Done()

		for {
			select {
			case <-time.NewTimer(time.Millisecond * time.Duration(DelayHandleDeferredPacket)).C:
				handle(context.TODO())
				continue
			case <-closeChan:
				vxp.mutex.Lock()
				for idx, closer := range vxp.closers {
					if reflect.DeepEqual(closer, closeFunc) {
						// It's possible because there used break in the bottom
						vxp.closers = append(vxp.closers[:idx], vxp.closers[idx+1:]...)
						break
					}
				}
				vxp.mutex.Unlock()
			}

			break
		}
	}()

	return closeFunc
}

func (vxp *vxProto) runRecvPQueue() func() {
	return vxp.runPQueue(func(ctx context.Context) {
		vxp.mxqueue.Lock()
		pq := vxp.rpqueue
		vxp.rpqueue = vxp.rpqueue[:0]
		vxp.mxqueue.Unlock()
		for _, p := range pq {
			if time.Since(time.Unix(p.TS, 0)).Seconds() < DeferredPacketTTLSeconds {
				// Here using public method because there need save proc time to recv other packets
				vxp.recvPacket(ctx, p)
			}
		}
	})
}

func (vxp *vxProto) runSendPQueue() func() {
	return vxp.runPQueue(func(ctx context.Context) {
		vxp.mxqueue.Lock()
		pq := vxp.spqueue
		vxp.spqueue = vxp.spqueue[:0]
		vxp.mxqueue.Unlock()
		for _, p := range pq {
			if time.Since(time.Unix(p.TS, 0)).Seconds() < DeferredPacketTTLSeconds {
				// Here using public method because there need save proc time to send other packets
				vxp.sendPacket(ctx, p)
			}
		}
	})
}

// Close is function which will stop all agents and server
func (vxp *vxProto) Close(ctx context.Context) error {
	vxp.indicateClosedState()

	var retErr error
	vxp.mutex.Lock()

	// Close all registred sockets
	for _, close := range vxp.closers {
		close()
	}

	// Remove and stop all registred agents
	for _, asocket := range vxp.agents {
		if err := asocket.Close(ctx); err != nil {
			retErr = err
		}
		if !vxp.delAgentI(ctx, asocket) {
			retErr = fmt.Errorf("failed to delete an agent")
		}
	}

	vxp.mutex.Unlock()

	// TODO: here need delete all added Module Sockets
	vxp.wgqueue.Wait()

	return retErr
}

func (vxp *vxProto) indicateClosedState() {
	vxp.isClosedMux.Lock()
	defer vxp.isClosedMux.Unlock()
	vxp.isClosed = true
}

// Connect is function for implement client network interface for many wrappers
// config argument should be include next fields:
// connection - described base URL to connect to server ("wss://localhost:8443")
// in the connection field are supported next schemes: ws, wss
// id - means Agent ID for authentication on the server side
// token - means Agent Token for authorization each connection on the server side
func (vxp *vxProto) Connect(
	ctx context.Context,
	config *ClientConfig,
	connValidator AgentConnectionValidator,
	tunnelEncrypter tunnel.PackEncryptor,
) error {
	infoGetter := system.GetAgentInfoAsync()
	conn, closeAgentQueues, err := vxp.prepareAgentConnection(
		ctx,
		config,
		connValidator,
		tunnelEncrypter,
		infoGetter,
	)
	if err != nil {
		return err
	}
	defer closeAgentQueues()
	return conn.connect(ctx)
}

func (vxp *vxProto) prepareAgentConnection(
	ctx context.Context,
	config *ClientConfig,
	connValidator AgentConnectionValidator,
	tunnelEncrypter tunnel.PackEncryptor,
	infoGetter system.AgentInfoGetter,
) (agentConn, func(), error) {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()
	if vxp.isClosed {
		return nil, nil, fmt.Errorf("failed to connect to the server: vxProto is already closed")
	}
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("failed to connect to the server: the operation was cancelled")
	}
	conn, closeConn, err := vxp.openAgentConnection(
		ctx,
		config,
		connValidator,
		tunnelEncrypter,
		infoGetter,
	)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err != nil {
			closeConn()
		}
	}()
	closeQueues, err := vxp.prepareAgentQueues(ctx)
	if err != nil {
		return nil, nil, err
	}
	cleanup := func() {
		closeConn()
		closeQueues()
	}
	return conn, cleanup, nil
}

func (vxp *vxProto) openAgentConnection(
	ctx context.Context,
	config *ClientConfig,
	connValidator AgentConnectionValidator,
	tunnelEncrypter tunnel.PackEncryptor,
	infoGetter system.AgentInfoGetter,
) (agentConn, func(), error) {
	if ctx.Err() != nil {
		return nil, nil, fmt.Errorf("failed to open a connection to the server: the operation was cancelled")
	}

	var conn agentConn
	var closeConn func()
	conn, closeConn, err := vxp.openAgentSocket(
		ctx,
		config,
		connValidator,
		tunnelEncrypter,
		infoGetter,
	)
	if err != nil {
		return nil, nil, err
	}
	if ctx.Err() != nil {
		closeConn()
		return nil, nil, fmt.Errorf("failed to open a connection to the server: the operation was cancelled")
	}
	return conn, closeConn, nil
}

func (vxp *vxProto) prepareAgentQueues(ctx context.Context) (closeFn func(), err error) {
	var recvCloseHandle, sendCloseHandle func()
	if ctx.Err() != nil {
		return nil, fmt.Errorf("failed to prepare agent queues: the operation was cancelled")
	}

	defer func() {
		if r := recover(); r != nil {
			if recvCloseHandle != nil {
				recvCloseHandle()
			}
			if sendCloseHandle != nil {
				sendCloseHandle()
			}
			closeFn = nil
			recoverErr, ok := r.(error)
			if !ok {
				err = fmt.Errorf("failed to prepare the agent queues: %v", r)
				return
			}
			err = fmt.Errorf("failed to prepare the agent queues: %w", recoverErr)
		}
	}()

	recvCloseHandle = vxp.runRecvPQueue()
	sendCloseHandle = vxp.runSendPQueue()
	cleanup := func() {
		recvCloseHandle()
		sendCloseHandle()
	}
	if ctx.Err() != nil {
		cleanup()
		return nil, fmt.Errorf("failed to prepare agent queues: the operation was cancelled")
	}
	return cleanup, nil
}

// Listen is function for implement server network interface for many wrappers
// config argument should be include next fields:
// listen - described base URL to listen on the server ("wss://0.0.0.0:8443")
// in the listen field are supported next schemes: ws, wss
// ssl_cert - contains SSL certificate in PEM format (for wss URL scheme)
// ssl_key - contains SSL private key in PEM format (for wss URL scheme)
func (vxp *vxProto) Listen(
	ctx context.Context,
	config *ServerConfig,
	validatorFactory ConnectionValidatorFactory,
	logger *logrus.Entry,
) error {
	if config == nil {
		return fmt.Errorf("no server config provided")
	}

	recvCloseHandle := vxp.runRecvPQueue()
	defer recvCloseHandle()
	sendCloseHandle := vxp.runSendPQueue()
	defer sendCloseHandle()

	return vxp.listenWS(ctx, config, validatorFactory, logger)
}

// NewModule is function which used for new module creation
func (vxp *vxProto) NewModule(name, gid string) IModuleSocket {
	imcToken := vxp.MakeIMCToken(name, gid)
	return &moduleSocket{
		name:             name,
		groupID:          gid,
		imcToken:         imcToken,
		router:           newRouter(),
		IIMC:             vxp,
		IRouter:          vxp,
		IProtoStats:      vxp,
		IProtoIO:         vxp,
		IDefaultReceiver: vxp,
		closer: func(ctx context.Context) {
			vxp.unsubscribeIMCFromAllTopics(imcToken)
		},
	}
}

// GetModule is function which used for get module by name ang group ID
func (vxp *vxProto) GetModule(name, gid string) IModuleSocket {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	if module, ok := vxp.modules[name]; ok {
		return module[gid]
	}

	return nil
}

// AddModule is function which used for new module registration
func (vxp *vxProto) AddModule(imsocket IModuleSocket) bool {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()
	if vxp.isClosed {
		return false
	}

	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	msocket := imsocket.(*moduleSocket)
	if module, ok := vxp.modules[msocket.GetName()]; !ok {
		vxp.modules[msocket.GetName()] = make(map[string]*moduleSocket)
		vxp.modules[msocket.GetName()][msocket.GetGroupID()] = msocket
	} else if _, ok := module[msocket.GetGroupID()]; !ok {
		vxp.modules[msocket.GetName()][msocket.GetGroupID()] = msocket
	} else {
		return false
	}

	return true
}

// DelModule is function which used for delete module object
func (vxp *vxProto) DelModule(imsocket IModuleSocket) bool {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()
	if vxp.isClosed {
		return false
	}

	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	msocket := imsocket.(*moduleSocket)
	if module, ok := vxp.modules[msocket.GetName()]; ok {
		if _, ok := module[msocket.GetGroupID()]; ok {
			msocket.Close(context.TODO())
			delete(vxp.modules[msocket.GetName()], msocket.GetGroupID())
			if len(vxp.modules[msocket.GetName()]) == 0 {
				delete(vxp.modules, msocket.GetName())
			}
			return true
		}
	}

	return false
}

// GetRoutes is function for get routes to existing destination points
func (vxp *vxProto) GetRoutes() map[string]string {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	return vxp.routes
}

// GetRoutes is function for get route to this destination points
func (vxp *vxProto) GetRoute(dst string) string {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	if src, ok := vxp.routes[dst]; ok {
		return src
	}

	return ""
}

// AddRoute is function for add route to new destination point by source token
func (vxp *vxProto) AddRoute(dst, src string) error {
	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	if _, ok := vxp.routes[dst]; ok {
		return fmt.Errorf("route to this destination already exist")
	}

	if vxp.getAgentBySrcI(src) == nil {
		return fmt.Errorf("source token doesn't exist in the proto")
	}

	vxp.routes[dst] = src

	return nil
}

// DelRoute is function for delete route to exists destination point
func (vxp *vxProto) DelRoute(dst string) error {
	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	if _, ok := vxp.routes[dst]; !ok {
		return fmt.Errorf("route to this destination doesen't exist")
	}

	delete(vxp.routes, dst)

	return nil
}

// GetAgentList is function which return agents list with info structure
func (vxp *vxProto) GetAgentList() map[string]IAgentSocket {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	agents := make(map[string]IAgentSocket)
	for _, asocket := range vxp.agents {
		agents[asocket.GetDestination()] = asocket
	}

	return agents
}

// GetAgentList is function which return agents amount
func (vxp *vxProto) GetAgentsCount() int {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	return len(vxp.agents)
}

// GetAgentBySrc is function which used for get agent socket interface by source
func (vxp *vxProto) GetAgentBySrc(src string) IAgentSocket {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	return vxp.getAgentBySrcI(src)
}

// GetAgentByDst is function which used for get agent socket interface by destination
func (vxp *vxProto) GetAgentByDst(dst string) IAgentSocket {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	return vxp.getAgentByDstI(dst)
}

// DropAgent is function to remove all agents sockets from vxproto
func (vxp *vxProto) DropAgent(ctx context.Context, aid string) {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()
	if vxp.isClosed {
		return
	}

	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	agents := make([]*agentSocket, 0)
	for _, asocket := range vxp.agents {
		if asocket.GetAgentID() == aid {
			agents = append(agents, asocket)
		}
	}
	for _, asocket := range agents {
		asocket.Close(ctx)
		vxp.delAgentI(ctx, asocket)
	}
}

// MoveAgent is function to update agent group id on agent moving
func (vxp *vxProto) MoveAgent(ctx context.Context, aid, gid string) {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()
	if vxp.isClosed {
		return
	}

	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	for _, asocket := range vxp.agents {
		if asocket.GetAgentID() == aid && asocket.GetGroupID() != gid {
			vxp.notifyAgentDisconnected(ctx, asocket.GetSource())
			asocket.SetGroupID(gid)
			vxp.notifyAgentConnected(ctx, asocket.GetSource())
		}
	}
}

// getCRC32 is function for making CRC32 bytes from data bytes
func (vxp *vxProto) getCRC32(data []byte) []byte {
	crc32q := crc32.MakeTable(0xD5828281)
	crc32t := crc32.Checksum(data, crc32q)
	crc32r := make([]byte, 4)
	binary.LittleEndian.PutUint32(crc32r, uint32(crc32t))

	return crc32r
}

// HasTokenValid is function which validate agent token
func (vxp *vxProto) HasTokenValid(token string, agentID string, agentType AgentType) bool {
	if len(token) != 40 {
		return false
	}

	tokenBytes, err := hex.DecodeString(token)
	if err != nil {
		return false
	}

	secret := tokenBytes[:16]
	if !bytes.Equal(tokenBytes[16:], vxp.getCRC32(secret)) {
		return false
	}

	block, err := des.NewTripleDESCipher(vxp.tokenKey)
	if err != nil {
		return false
	}
	iv := vxp.tokenKey[:des.BlockSize]
	mode := cipher.NewCBCDecrypter(block, iv)
	plainPayload := make([]byte, len(secret))
	mode.CryptBlocks(plainPayload, secret)

	tokenPayload := plainPayload[:12]
	crc32Payload := plainPayload[12:]
	if !bytes.Equal(crc32Payload, vxp.getCRC32(tokenPayload)) {
		return false
	}

	tokenRand := tokenPayload[:4]
	tokenStateBuffer := append([]byte(agentID+":"+agentType.String()+":"), tokenRand[:]...)

	return bytes.Equal(tokenPayload[8:12], vxp.getCRC32(tokenStateBuffer))
}

// HasTokenCRCValid is function which validate CRC for agent token
func (vxp *vxProto) HasTokenCRCValid(token string) bool {
	if len(token) != 40 {
		return false
	}

	tokenBytes, err := hex.DecodeString(token)
	if err != nil {
		return false
	}

	if !bytes.Equal(tokenBytes[16:], vxp.getCRC32(tokenBytes[:16])) {
		return false
	}

	return true
}

// NewToken is function which generate new token
// Flags: 4 bytes reserved for future features
// Rand: 4 random bytes which generated on server side (const for VXAgent type)
// State: CRC32(agentID + agentType + tokenRand)
// Payload: 12 bytes (Rand + TS)
// IV: initialization vector for 3DES cipher is first bytes from key
// Secret: 3DES(Payload + CRC32(Payload))
// Token: Secret + CRC32(Secret)
func (vxp *vxProto) NewToken(agentID string, agentType AgentType) (string, error) {
	tokenFlags := uint32(0)
	tokenFlagsBuffer := make([]byte, 4)
	binary.LittleEndian.PutUint32(tokenFlagsBuffer, tokenFlags)

	tokenRand := make([]byte, 4)
	if _, err := rand.Read(tokenRand); err != nil {
		return "", fmt.Errorf("failed to make token random part: %w", err)
	}
	if agentType == VXAgent || agentType == VXServer {
		tokenRand = vxp.getCRC32([]byte(agentID))
	}

	tokenStateBuffer := append([]byte(agentID+":"+agentType.String()+":"), tokenRand[:]...)
	tokenState := vxp.getCRC32(tokenStateBuffer)

	tokenPayload := append(tokenRand[:], tokenFlagsBuffer...)
	tokenPayload = append(tokenPayload, tokenState...)
	crc32Payload := vxp.getCRC32(tokenPayload)

	block, err := des.NewTripleDESCipher(vxp.tokenKey)
	if err != nil {
		return "", fmt.Errorf("failed to encrypt payload token part: %w", err)
	}
	iv := vxp.tokenKey[:des.BlockSize]
	mode := cipher.NewCBCEncrypter(block, iv)

	plainPayload := append(tokenPayload[:], crc32Payload[:]...)
	secret := make([]byte, len(plainPayload))
	mode.CryptBlocks(secret, plainPayload)
	tokenBytes := append(secret[:], vxp.getCRC32(secret)...)
	token := hex.EncodeToString(tokenBytes)

	if vxp.HasIMCTokenFormat(token) {
		return vxp.NewToken(agentID, agentType)
	}

	return token, nil
}

// HasIMCTokenFormat is a IMC token format validation function
func (vxp *vxProto) HasIMCTokenFormat(token string) bool {
	return len(token) == 40 && strings.HasPrefix(token, "ffffffff")
}

// HasIMCTopicFormat is a IMC topic format validation function
func (vxp *vxProto) HasIMCTopicFormat(topic string) bool {
	return len(topic) == 40 && strings.HasPrefix(topic, "ffff7777")
}

// GetIMCModule is function which get module socket for imc token
// The function may return nil if the token doesn't exist
func (vxp *vxProto) GetIMCModule(token string) IModuleSocket {
	if !vxp.HasIMCTokenFormat(token) {
		return nil
	}

	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	for _, module := range vxp.modules {
		for _, msocket := range module {
			if msocket.GetIMCToken() == token {
				return msocket
			}
		}
	}

	return nil
}

// GetIMCTopic is function which get topic info for imc topic token
// The function may return nil if the topic token doesn't exist
func (vxp *vxProto) GetIMCTopic(topic string) ITopicInfo {
	if !vxp.HasIMCTopicFormat(topic) {
		return nil
	}

	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	if info, ok := vxp.topics[topic]; ok && info != nil {
		return info
	}

	return nil
}

// MakeIMCToken generates new IMC token
func (vxp *vxProto) MakeIMCToken(name, gid string) string {
	hash := md5.Sum(append([]byte(gid+":"+name+":"), vxp.tokenKey...))
	return "ffffffff" + hex.EncodeToString(hash[:])
}

// MakeIMCTopic generates new IMC topic
func (vxp *vxProto) MakeIMCTopic(name, gid string) string {
	hash := md5.Sum(append([]byte(gid+":"+name+":"), vxp.tokenKey...))
	return "ffff7777" + hex.EncodeToString(hash[:])
}

// SubscribeIMCToTopic appends IMC topic subscription if it is not registered yet
func (vxp *vxProto) SubscribeIMCToTopic(name, gid, token string) bool {
	if !vxp.HasIMCTokenFormat(token) {
		return false
	}

	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	topic := vxp.MakeIMCTopic(name, gid)
	if info, ok := vxp.topics[topic]; ok && info != nil {
		if !stringInSlice(token, info.subscriptions) {
			info.subscriptions = append(info.subscriptions, token)
		}
	} else {
		vxp.topics[topic] = &topicInfo{
			name:          name,
			groupID:       gid,
			subscriptions: []string{token},
		}
	}

	return true
}

// UnsubscribeIMCFromTopic removes IMC topic subscription
func (vxp *vxProto) UnsubscribeIMCFromTopic(name, gid, token string) bool {
	if !vxp.HasIMCTokenFormat(token) {
		return false
	}

	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	topic := vxp.MakeIMCTopic(name, gid)
	if info, ok := vxp.topics[topic]; ok && info != nil {
		subs := info.subscriptions
		for idx, sub := range subs {
			if sub == token {
				// It's possible because there used break in the bottom
				subs = append(subs[:idx], subs[idx+1:]...)
				break
			}
		}
		if len(subs) == 0 {
			delete(vxp.topics, topic)
		} else {
			info.subscriptions = subs
		}
	}

	return true
}

// UnsubscribeIMCFromAllTopics removes all topic records for IMC token
func (vxp *vxProto) UnsubscribeIMCFromAllTopics(token string) bool {
	if !vxp.HasIMCTokenFormat(token) {
		return false
	}

	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	vxp.unsubscribeIMCFromAllTopics(token)

	return true
}

// Internal method is a helper to stop and close module socket before it'll free
func (vxp *vxProto) unsubscribeIMCFromAllTopics(token string) {
	for topic, info := range vxp.topics {
		if info == nil {
			delete(vxp.topics, topic)
			continue
		}
		subs := info.subscriptions
		for idx, sub := range subs {
			if sub == token {
				// It's possible because there used break in the bottom
				subs = append(subs[:idx], subs[idx+1:]...)
				break
			}
		}
		if len(subs) == 0 {
			delete(vxp.topics, topic)
		} else {
			info.subscriptions = subs
		}
	}
}

// GetIMCTopics returns all registered IMC topics identifiers
func (vxp *vxProto) GetIMCTopics() []string {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	topics := make([]string, 0, len(vxp.topics))
	for topic := range vxp.topics {
		topics = append(topics, topic)
	}

	return topics
}

// GetIMCGroupIDs is function which get group list which registered in vxproto
func (vxp *vxProto) GetIMCGroupIDs() []string {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	groups := make([]string, 0)
	cache := make(map[string]struct{})
	for _, module := range vxp.modules {
		for gid := range module {
			if _, ok := cache[gid]; !ok {
				cache[gid] = struct{}{}
				groups = append(groups, gid)
			}
		}
	}

	return groups
}

// GetIMCModuleIDs is function which get module list which registered in vxproto
func (vxp *vxProto) GetIMCModuleIDs() []string {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	modules := make([]string, 0, len(vxp.modules))
	for mname := range vxp.modules {
		modules = append(modules, mname)
	}

	return modules
}

// GetIMCGroupIDsByMID is function which get group list by the module ID
// The function may return empty list while module ID isn't exist
func (vxp *vxProto) GetIMCGroupIDsByMID(mid string) []string {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	groups := make([]string, 0)
	for gid := range vxp.modules[mid] {
		groups = append(groups, gid)
	}

	return groups
}

// GetIMCModuleIDsByGID is function which get module list by the group ID
// The function may return empty list while group ID isn't exist
func (vxp *vxProto) GetIMCModuleIDsByGID(gid string) []string {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	modules := make([]string, 0)
	for mname, module := range vxp.modules {
		if _, ok := module[gid]; ok {
			modules = append(modules, mname)
		}
	}

	return modules
}

// getModule is function which used for get module object
//
//lint:ignore U1000 getModule
func (vxp *vxProto) getModule(name, gid string) *moduleSocket {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	return vxp.getModuleI(name, gid)
}

// getModuleI is internal function which used for get module object
func (vxp *vxProto) getModuleI(name, gid string) *moduleSocket {
	if module, ok := vxp.modules[name]; ok {
		if msocket, ok := module[gid]; ok {
			return msocket
		}
		// This case for shared modules
		if msocket, ok := module[""]; ok {
			return msocket
		}
		// Fallback for agent side when group ID is unknown
		if gid == "" && len(module) == 1 {
			for _, msocket := range module {
				return msocket
			}
		}
	}

	return nil
}

// addAgent is function which used for new agent registration
func (vxp *vxProto) addAgent(ctx context.Context, asocket *agentSocket) bool {
	if vxp.isClosed {
		return false
	}

	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	return vxp.addAgentI(ctx, asocket)
}

// addAgentI is internal function which used for new agent registration
func (vxp *vxProto) addAgentI(ctx context.Context, asocket *agentSocket) bool {
	if vxp.getAgentBySrcI(asocket.GetSource()) != nil {
		return false
	}

	vxp.agents[asocket.GetSource()] = asocket
	vxp.routes[asocket.GetDestination()] = asocket.GetSource()

	// Notify all modules that the agent has been connected
	vxp.notifyAgentConnected(ctx, asocket.GetSource())

	return true
}

// delAgent is function which used for delete agent object
func (vxp *vxProto) delAgent(ctx context.Context, asocket *agentSocket) bool {
	vxp.mutex.Lock()
	defer vxp.mutex.Unlock()

	return vxp.delAgentI(ctx, asocket)
}

// delAgentI is internal function which used for delete agent object
func (vxp *vxProto) delAgentI(ctx context.Context, asocket *agentSocket) bool {
	if vxp.getAgentBySrcI(asocket.GetSource()) != nil {
		var dsts []string
		for dst, src := range vxp.routes {
			if src == asocket.GetSource() {
				dsts = append(dsts, dst)
			}
		}
		for _, dst := range dsts {
			delete(vxp.routes, dst)
		}

		// Notify all modules that the agent has been disconnected
		vxp.notifyAgentDisconnected(ctx, asocket.GetSource())

		delete(vxp.agents, asocket.GetSource())
		asocket.Close(ctx)
		return true
	}

	return false
}

// getAgentBySrc is function which used for get agent object by source
//
//lint:ignore U1000 getAgentBySrc
func (vxp *vxProto) getAgentBySrc(src string) *agentSocket {
	vxp.mutex.RLock()
	defer vxp.mutex.RUnlock()

	return vxp.getAgentBySrcI(src)
}

// getAgentBySrcI is internal function which used for get agent object
func (vxp *vxProto) getAgentBySrcI(src string) *agentSocket {
	if asocket, ok := vxp.agents[src]; ok {
		return asocket
	}

	return nil
}

// getAgentByDstI is internal function which used for get agent object
func (vxp *vxProto) getAgentByDstI(dst string) *agentSocket {
	for rdst, rsrc := range vxp.routes {
		if asocket, ok := vxp.agents[rsrc]; rdst == dst && ok {
			return asocket
		}
	}

	return nil
}

// recvPacket is function for serving packet to target module
// Result is the success of packet processing otherwise will raise error
func (vxp *vxProto) recvPacket(ctx context.Context, packet *Packet) error {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()
	if vxp.isClosed {
		return fmt.Errorf("vxProto is closed")
	}

	vxp.mutex.RLock()
	applyPacket, err := vxp.recvPacketI(ctx, packet)
	vxp.mutex.RUnlock()
	if err != nil {
		return err
	}

	return applyPacket()
}

// recvPacketI is internal function for serving packet to target module
// Result is the success of packet processing otherwise will raise error
func (vxp *vxProto) recvPacketI(ctx context.Context, packet *Packet) (applyPacketCb, error) {
	// Case for direct forward packets
	if _, ok := vxp.routes[packet.Dst]; ok {
		return vxp.sendPacketI(ctx, packet)
	}

	hDeferredPacket := func() error {
		vxp.mxqueue.Lock()
		defer vxp.mxqueue.Unlock()

		if len(vxp.rpqueue) >= MaxSizePacketQueue {
			vxp.rpqueue = vxp.rpqueue[1:]
		}
		vxp.rpqueue = append(vxp.rpqueue, packet)

		return nil
	}

	// Case for local accept packets
	asocket := vxp.getAgentBySrcI(packet.Dst)
	if asocket == nil {
		// Failed getting of agent information
		return hDeferredPacket, nil
	}
	if err := asocket.connectionPolicy.CheckInPacket(context.Background(), packet); err != nil {
		return nil, fmt.Errorf("packet cannot be received due to the agent connection policy: %w", err)
	}
	msocket := vxp.getModuleI(packet.Module, asocket.GetGroupID())
	if msocket == nil {
		// Failed getting of module
		return hDeferredPacket, nil
	}

	return func() error {
		return msocket.recvPacket(ctx, packet)
	}, nil
}

// sendPacket is function for sending packet to target agent
// Result is the success of packet sending otherwise will raise error
func (vxp *vxProto) sendPacket(ctx context.Context, packet *Packet) error {
	vxp.isClosedMux.RLock()
	defer vxp.isClosedMux.RUnlock()
	if vxp.isClosed {
		return fmt.Errorf("vxProto is closed")
	}

	vxp.mutex.RLock()
	applyPacket, err := vxp.sendPacketI(ctx, packet)
	vxp.mutex.RUnlock()
	if err != nil {
		return err
	}

	return applyPacket()
}

// sendPacket is internal function for sending packet to target agent
// Result is the success of packet sending otherwise will raise error
func (vxp *vxProto) sendPacketI(ctx context.Context, packet *Packet) (applyPacketCb, error) {
	hDeferredPacket := func() error {
		vxp.mxqueue.Lock()
		defer vxp.mxqueue.Unlock()

		if len(vxp.spqueue) >= MaxSizePacketQueue {
			vxp.spqueue = vxp.spqueue[1:]
		}
		vxp.spqueue = append(vxp.spqueue, packet)

		return nil
	}

	asocket := vxp.getAgentByDstI(packet.Dst)
	if asocket == nil {
		// Failed getting of agent
		return hDeferredPacket, nil
	}
	if err := asocket.connectionPolicy.CheckOutPacket(context.Background(), packet); err != nil {
		return nil, fmt.Errorf("packet cannot be sent due to the agent connection policy: %w", err)
	}

	if packet.Src == "" {
		packet.Src = asocket.GetSource()
	}

	return func() error {
		return asocket.sendPacket(ctx, packet)
	}, nil
}

// notifyAgentConnected is a function to send control message
// to modules sockets about that new agent was connected to vxproto
func (vxp *vxProto) notifyAgentConnected(ctx context.Context, src string) {
	asocket := vxp.getAgentBySrcI(src)
	if asocket == nil {
		return
	}

	packet := &Packet{
		ctx:   ctx,
		PType: PTControl,
		Payload: &ControlMessage{
			AgentInfo: asocket.GetPublicInfo(),
			MsgType:   AgentConnected,
		},
	}
	for _, module := range vxp.modules {
		for _, msocket := range module {
			if asocket.isOnlyForUpgrade && msocket.name != mainModuleName {
				continue
			}
			mgid := msocket.GetGroupID()
			agid := asocket.GetGroupID()
			if mgid == agid || mgid == "" || agid == "" {
				// use nonblocking sending notification to avoid concurent blocking
				// when msocket wants to send some data via vxproto and blocks reader
				receiver := msocket.GetReceiver()
				select {
				case receiver <- packet:
				default:
					// get more guaranty by deferred send this notification
					go func() {
						select {
						case receiver <- packet:
						case <-ctx.Done():
						}
					}()
				}
			}
		}
	}
}

// notifyAgentDisconnected is a function to send control message
// to modules sockets about that the agent was disconnected from vxproto
func (vxp *vxProto) notifyAgentDisconnected(ctx context.Context, src string) {
	asocket := vxp.getAgentBySrcI(src)
	if asocket == nil {
		return
	}

	packet := &Packet{
		ctx:   ctx,
		PType: PTControl,
		Payload: &ControlMessage{
			AgentInfo: asocket.GetPublicInfo(),
			MsgType:   AgentDisconnected,
		},
	}
	for _, module := range vxp.modules {
		for _, msocket := range module {
			if asocket.isOnlyForUpgrade && msocket.name != mainModuleName {
				continue
			}
			mgid := msocket.GetGroupID()
			agid := asocket.GetGroupID()
			if mgid == agid || mgid == "" || agid == "" {
				go msocket.router.unlock(asocket.GetSource())
				// use nonblocking sending notification to avoid concurent blocking
				// when msocket wants to send some data via vxproto and blocks reader
				receiver := msocket.GetReceiver()
				select {
				case receiver <- packet:
				default:
					// get more guaranty by deferred send this notification
					go func(quit chan struct{}) {
						select {
						case receiver <- packet:
						case <-quit:
						}
					}(msocket.router.control)
				}
			}
		}
	}
}

func (ti *topicInfo) GetName() string {
	return ti.name
}

func (ti *topicInfo) GetGroupID() string {
	return ti.groupID
}

func (ti *topicInfo) GetSubscriptions() []string {
	subs := make([]string, 0, len(ti.subscriptions))
	return append(subs, ti.subscriptions...)
}

func getURLFromHost(host string) (*url.URL, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the host string %s to URL: %w", host, err)
	}
	if len(u.Scheme) == 0 {
		switch u.Port() {
		case "80":
			u.Scheme = "ws"
		case "443":
			fallthrough
		default:
			u.Scheme = "wss"
		}
	}
	hostname := u.Hostname()
	port := u.Port()
	if len(port) == 0 {
		if u.Scheme == "ws" {
			port = "80"
		} else {
			port = "443"
		}
	}
	u.Host = fmt.Sprintf("%s:%s", hostname, port)
	return u, nil
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}
