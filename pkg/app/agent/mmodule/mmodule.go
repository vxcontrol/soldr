package mmodule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/url"
	"path"
	"runtime/debug"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/takama/daemon"
	"github.com/vxcontrol/luar"
	"go.opentelemetry.io/otel/attribute"

	"soldr/pkg/app/api/models"
	vxcommonErrors "soldr/pkg/errors"
	"soldr/pkg/hardening/luavm/store/types"
	"soldr/pkg/hardening/luavm/vm"
	connValidator "soldr/pkg/hardening/validator"
	"soldr/pkg/loader"
	"soldr/pkg/lua"
	obs "soldr/pkg/observability"
	"soldr/pkg/protoagent"
	"soldr/pkg/system"
	"soldr/pkg/vxproto"
	"soldr/pkg/vxproto/tunnel"
	tunnelSimple "soldr/pkg/vxproto/tunnel/simple"
)

var protocolVersion = ""

// MainModule is struct which contains full state for agent working
type MainModule struct {
	ctx              context.Context
	cancelCtx        context.CancelFunc
	proto            vxproto.IVXProto
	connectionString string
	agentID          string
	version          string
	modules          map[string]*loader.ModuleConfig
	loader           loader.ILoader
	msocket          vxproto.IModuleSocket
	wgReceiver       sync.WaitGroup
	hasStopped       bool
	mutexResp        *sync.Mutex
	stopConnect      func()

	meterConfigClient  *obs.HookClientConfig
	tracerConfigClient *obs.HookClientConfig

	upgrader   *upgrader
	upgraderWG sync.WaitGroup

	tlsConfigurer         vm.TLSConfigurer
	connValidator         *connValidator.Validator
	tunnelEncrypter       tunnel.PackEncryptor
	secureConfigEncryptor vm.ISecureConfigEncryptor
}

// GetVersion is function that return of agent version
func (mm *MainModule) GetVersion() string {
	return mm.version
}

// DefaultRecvPacket is function that operate packets as default receiver
func (mm *MainModule) DefaultRecvPacket(ctx context.Context, packet *vxproto.Packet) error {
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": packet.Module,
		"type":   packet.PType.String(),
		"src":    packet.Src,
		"dst":    packet.Dst,
	}).Debug("vxagent: default receiver got new packet")

	return nil
}

// HasAgentInfoValid is function that validate Agent Information in Agent list
func (mm *MainModule) HasAgentInfoValid(_ context.Context, _ vxproto.IAgentSocket) error {
	return nil
}

func (mm *MainModule) forceFreeMemory() {
	select {
	case <-time.NewTicker(time.Second).C:
		debug.FreeOSMemory()
	case <-mm.ctx.Done():
	}
}

func (mm *MainModule) recvData(ctx context.Context, src string, data *vxproto.Data) error {
	logger := logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "data",
		"src":    src,
		"len":    len(data.Data),
	})
	logger.Debug("vxagent: received data")
	if err := mm.serveData(ctx, src, data); err != nil {
		logger.WithError(err).Error("vxagent: failed to exec command")
		return err
	} else {
		logger.Debug("vxagent: successful exec server command")
	}

	return nil
}

func (mm *MainModule) recvFile(ctx context.Context, src string, file *vxproto.File) error {
	if file.IsUpgrader() {
		if err := mm.upgrader.recvFile(ctx, file); err != nil {
			return fmt.Errorf("failed to receive the updater file: %w", err)
		}
	}
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "file",
		"name":   file.Name,
		"path":   file.Path,
		"uniq":   file.Uniq,
		"src":    src,
	}).Debug("vxagent: received file")

	return nil
}

//nolint:unparam
func (mm *MainModule) recvText(ctx context.Context, src string, text *vxproto.Text) error {
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "text",
		"name":   text.Name,
		"len":    len(text.Data),
		"src":    src,
	}).Debug("vxagent: received text")

	return nil
}

//nolint:unparam
func (mm *MainModule) recvMsg(ctx context.Context, src string, msg *vxproto.Msg) error {
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "msg",
		"msg":    msg.MType.String(),
		"len":    len(msg.Data),
		"src":    src,
	}).Debug("vxagent: received message")

	return nil
}

//nolint:unparam
func (mm *MainModule) recvAction(ctx context.Context, src string, act *vxproto.Action) error {
	logrus.WithContext(ctx).WithFields(logrus.Fields{
		"module": "main",
		"type":   "action",
		"name":   act.Name,
		"len":    len(act.Data),
		"src":    src,
	}).Debug("vxagent: received action")

	return nil
}

// New is function which constructed MainModule object
func New(
	connectionString,
	agentID,
	version string,
	isService bool,
	logDir string,
	stopAgent context.CancelFunc,
	svc daemon.Daemon,
	meterConfigClient *obs.HookClientConfig,
	tracerConfigClient *obs.HookClientConfig,
) (*MainModule, error) {
	if agentID == "" {
		agentID = system.MakeAgentID()
	}
	ctx, cancelCtx := context.WithCancel(context.Background())
	mm := &MainModule{
		ctx:              ctx,
		cancelCtx:        cancelCtx,
		connectionString: connectionString,
		agentID:          agentID,
		version:          version,
		modules:          make(map[string]*loader.ModuleConfig),
		loader:           loader.New(),
		mutexResp:        &sync.Mutex{},

		meterConfigClient:  meterConfigClient,
		tracerConfigClient: tracerConfigClient,
	}

	var err error
	hardeningVM, err := initializeHardeningVM(logDir, agentID)
	if err != nil {
		return nil, err
	}
	mm.upgrader, err = newUpgrader(mm, isService, logDir, stopAgent, svc)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize an upgrader for the Main module: %w", err)
	}
	mm.tlsConfigurer = hardeningVM
	mm.tunnelEncrypter, err = tunnel.NewPackEncrypter(&tunnel.Config{
		Simple: &tunnelSimple.Config{},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tunnel encryptor for the Main module: %w", err)
	}
	mm.secureConfigEncryptor = hardeningVM
	mm.connValidator = connValidator.NewValidator(mm.agentID, mm.version, hardeningVM)
	return mm, nil
}

func initializeHardeningVM(logDir, agentID string) (vm.VM, error) {
	luaVM, err := vm.NewVM(&vm.VMConfig{
		StoreConfig: &types.Config{
			Dir:     path.Join(logDir, ".artifacts"),
			AgentID: agentID,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a Lua VM: %w", err)
	}
	return luaVM, nil
}

// Start is function which execute main logic of MainModule
func (mm *MainModule) Start() (err error) {
	startCtx, startSpan := obs.Observer.NewSpan(mm.ctx, obs.SpanKindInternal, "start_agent")
	defer startSpan.End()

	defer func() {
		if err != nil {
			logrus.WithContext(startCtx).WithError(err).Error("vxagent: start main module failed")
		}
	}()

	mm.hasStopped = false

	mm.proto, err = vxproto.New(mm)
	if err != nil {
		err = fmt.Errorf("failed to initialize the main module vxproto: %w", err)
		return
	}

	mm.msocket = mm.proto.NewModule("main", "")
	if mm.msocket == nil {
		err = fmt.Errorf("failed to create new main module into vxproto")
		return
	}

	if !mm.proto.AddModule(mm.msocket) {
		err = fmt.Errorf("failed to register socket for main module")
		return
	}

	if err = mm.registerUploadCallbacks(); err != nil {
		err = fmt.Errorf("failed to register upload callbacks: %w", err)
		return
	}

	// initialize system metric collection in current observer instance
	attr := attribute.String("agent_id", mm.agentID)
	const serviceName = "vxagent"
	if err = obs.Observer.StartProcessMetricCollect(serviceName, mm.version, attr); err != nil {
		err = fmt.Errorf("failed to start Process metric collect: %w", err)
		return
	}
	if err = obs.Observer.StartGoRuntimeMetricCollect(serviceName, mm.version, attr); err != nil {
		err = fmt.Errorf("failed to start Go runtime metric collect: %w", err)
		return
	}
	if err = obs.Observer.StartDumperMetricCollect(mm.proto, serviceName, mm.version, attr); err != nil {
		err = fmt.Errorf("failed to start Dumper metric collect: %w", err)
		return
	}

	// run main handler of packets
	mm.wgReceiver.Add(1)
	go func() {
		_ = mm.recvPacket()
	}()
	logrus.WithContext(startCtx).Debug("vxagent: main module was started")
	defer logrus.WithContext(startCtx).Debug("vxagent: main module was stopped")

	startSpan.End()
	config := map[string]string{
		"id":         mm.agentID,
		"token":      "",
		"connection": mm.connectionString,
	}
	return mm.connect(config)
}

const connectCooldownSeconds = 60

func (mm *MainModule) connect(config map[string]string) error {
	connect := func(ctx context.Context) error {
		err := mm.performConnection(ctx, config)
		if errors.Is(err, vxcommonErrors.ErrConnectionInitializationRequired) {
			logrus.WithContext(ctx).WithError(err).
				Info("connection to the server has not been initialized yet, trying to init connection")
			if err := mm.performInitConnection(ctx, config); err != nil {
				return fmt.Errorf("init connection failed: %w", err)
			}
			logrus.WithContext(ctx).
				Debug("the connection has been successfully initialized, connecting to the server")
			return mm.performConnection(ctx, config)
		}
		return err
	}
	for {
		ctx, cancelCtx := context.WithCancel(mm.ctx)
		connectCtx, connectSpan := obs.Observer.NewSpan(ctx, obs.SpanKindClient, "connect_agent")
		connectCtx = context.WithValue(connectCtx, obs.VXProtoAgentConnect, func() {
			connectSpan.End()
		})
		logger := logrus.WithContext(connectCtx)
		if mm.proto == nil {
			connectSpan.End()
			cancelCtx()
			return fmt.Errorf("vxproto is not set")
		}
		logger.Infof("connecting to the server: %s", mm.connectionString)
		err := connect(connectCtx)
		if !mm.hasStopped {
			logger.WithError(err).Warn("vxagent: try reconnect")
			connectSpan.End()
		} else {
			connectSpan.End()
			cancelCtx()
			return nil
		}
		cancelCtx()
		for i := 0; i < connectCooldownSeconds; i++ {
			if mm.hasStopped {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}

func (mm *MainModule) performInitConnection(ctx context.Context, config map[string]string) error {
	logrus.WithContext(ctx).Debug("perform init connection")
	initConfig, err := getInitConfig(config)
	if err != nil {
		return fmt.Errorf("failed to get the config from the config map to initialize connection: %w", err)
	}
	initConnectCtx, cancelInitConnectCtx := context.WithCancel(ctx)
	defer cancelInitConnectCtx()
	if err := mm.initConnect(initConnectCtx, initConfig); err != nil {
		return fmt.Errorf("failed to initialize connection: %w", err)
	}
	return nil
}

func (mm *MainModule) performConnection(ctx context.Context, connConfig map[string]string) error {
	logrus.WithContext(ctx).Debug("perform connection")
	c, err := mm.getConfig(connConfig)
	if err != nil {
		return fmt.Errorf("failed to get the connection config: %w", err)
	}
	isKeyEmpty, err := mm.secureConfigEncryptor.IsStoreKeyEmpty()
	if err != nil {
		return fmt.Errorf("failed to check if cipher key exists: %w", err)
	}
	if isKeyEmpty {
		return fmt.Errorf("cipher key not exists: %w", vxcommonErrors.ErrConnectionInitializationRequired)
	}

	var connectCtx context.Context
	connectCtx, mm.stopConnect = context.WithCancel(ctx)
	if err := mm.proto.Connect(connectCtx, c, mm.connValidator, mm.tunnelEncrypter); err != nil {
		if errors.Is(err, vxcommonErrors.ErrConnectionInitializationRequired) {
			if unloadErr := mm.unloadModules(ctx, "module_remove"); unloadErr != nil {
				return fmt.Errorf(
					"failed to unload modules (%v) while processing the connection error: %w",
					unloadErr,
					err,
				)
			}
		}
		return fmt.Errorf("connection failed: %w", err)
	}
	return nil
}

var errConfigIsNil = fmt.Errorf("passed configuration map is nil")

func getInitConfig(c map[string]string) (*initConnectConfig, error) {
	if c == nil {
		return nil, errConfigIsNil
	}
	host, err := getConfigHost(c)
	if err != nil {
		return nil, err
	}
	return &initConnectConfig{
		Host: host,
	}, nil
}

type connectionConfig struct {
	Host string
	ID   string
}

func connConfigMapToConfig(c map[string]string) (*connectionConfig, error) {
	if c == nil {
		return nil, errConfigIsNil
	}
	host, err := getConfigHost(c)
	if err != nil {
		return nil, fmt.Errorf("failed to get the host to connect to: %w", err)
	}
	id, ok := c["id"]
	if !ok {
		return nil, fmt.Errorf("agent ID is not found in the config")
	}
	return &connectionConfig{
		Host: host,
		ID:   id,
	}, nil
}

func (mm *MainModule) getConfig(c map[string]string) (*vxproto.ClientConfig, error) {
	connConf, err := connConfigMapToConfig(c)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the config map into a connection config: %w", err)
	}
	tlsConfig, err := mm.tlsConfigurer.GetTLSConfigForConnection()
	if err != nil {
		return nil, fmt.Errorf("failed to get the TLS config for the client connection: %w", err)
	}
	return &vxproto.ClientConfig{
		ClientInitConfig: vxproto.ClientInitConfig{
			CommonConfig: &vxproto.CommonConfig{
				Host:      connConf.Host,
				TLSConfig: tlsConfig,
			},
			ProtocolVersion: protocolVersion,
		},
		ID: connConf.ID,
	}, nil
}

func getConfigHost(c map[string]string) (string, error) {
	conn, ok := c["connection"]
	if !ok {
		return "", fmt.Errorf("connection string is not found in the config")
	}
	return conn, nil
}

// Stop finishes MainModule
func (mm *MainModule) Stop(stopReason string) (err error) {
	stopCtx, stopSpan := obs.Observer.NewSpan(mm.ctx, obs.SpanKindInternal, "stop_agent")
	defer stopSpan.End()

	mm.hasStopped = true
	logrus.WithContext(stopCtx).Info("vxagent: trying to stop main module")
	defer func() {
		if err != nil {
			logrus.WithContext(stopCtx).WithError(err).Debug("vxagent: error on stopping")
		}
		logrus.WithContext(stopCtx).Info("vxagent: stopping of main module has done")
	}()

	mm.cancelCtx()

	if mm.stopConnect != nil {
		mm.stopConnect()
	}

	if mm.proto == nil {
		err = fmt.Errorf("VXProto didn't initialized")
		return
	}
	if mm.msocket == nil {
		err = fmt.Errorf("module socket didn't initialized")
		return
	}

	if !mm.proto.DelModule(mm.msocket) {
		err = fmt.Errorf("failed delete module socket")
		return
	}

	if err = mm.unloadModules(stopCtx, stopReason); err != nil {
		err = fmt.Errorf("failed to unload modules: %w", err)
		return
	}

	receiver := mm.msocket.GetReceiver()
	if receiver != nil {
		receiver <- &vxproto.Packet{
			PType: vxproto.PTControl,
			Payload: &vxproto.ControlMessage{
				MsgType: vxproto.StopModule,
			},
		}
	}
	mm.wgReceiver.Wait()

	stopSpan.End()
	obs.Observer.Flush(context.Background())
	if err = mm.unregisterUploadCallbacks(); err != nil {
		return err
	}

	if err = mm.proto.Close(context.Background()); err != nil {
		return err
	}

	mm.modules = make(map[string]*loader.ModuleConfig)
	mm.proto = nil
	mm.msocket = nil

	mm.upgraderWG.Wait()

	return nil
}

func (mm *MainModule) unloadModules(ctx context.Context, stopReason string) error {
	if err := mm.loader.StopAll(stopReason); err != nil {
		return fmt.Errorf("failed to stop the loaded modules: %w", err)
	}

	for _, id := range mm.loader.List() {
		logrus.WithContext(ctx).Debugf("deleting module %s", id)
		if !mm.loader.Del(id, stopReason) {
			return fmt.Errorf("failed to delete module %s from loader", id)
		}
	}
	return nil
}

const luaFuncTableNameAPI = "__api"

func (mm *MainModule) parseURLString(host string) (*url.URL, error) {
	u, err := url.Parse(host)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the URL string %s to URL object: %w", host, err)
	}
	const wsScheme = "ws"
	if len(u.Scheme) == 0 {
		switch u.Port() {
		case "80":
			u.Scheme = wsScheme
		case "443":
			fallthrough
		default:
			u.Scheme = "wss"
		}
	}
	hostname := u.Hostname()
	if hostname == "" {
		hostname = u.Path
		u.Path = ""
	}
	port := u.Port()
	if len(port) == 0 {
		if u.Scheme == wsScheme {
			port = "80"
		} else {
			port = "443"
		}
	}
	u.Host = fmt.Sprintf("%s:%s", hostname, port)
	return u, nil
}

// RegisterLuaAPI is function that registrate extra API function for each of type service
func (mm *MainModule) RegisterLuaAPI(state *lua.State, config *loader.ModuleConfig) error {
	gid := config.GroupID
	pid := config.PolicyID
	mname := config.Name
	secureCurrentConfig := config.IConfigItem.GetSecureCurrentConfig()
	const pushEvent = "push_event"

	luar.Register(state.L, "__sec", luar.Map{
		"get": func(key string) (string, bool) {
			b, err := mm.secureConfigEncryptor.DecryptData([]byte(secureCurrentConfig))
			if err != nil {
				return "", false
			}

			var model models.ModuleSecureConfig
			err = json.Unmarshal(b, &model)
			if err != nil {
				return "", false
			}

			val, ok := model[key]
			if !ok {
				return "", false
			}

			result, err := json.Marshal(val.Value)
			if err != nil {
				return "", false
			}

			return string(result), true
		},
	})

	luar.Register(state.L, luaFuncTableNameAPI, luar.Map{
		pushEvent: func(aid, info string) bool {
			eventCtx, eventSpan := obs.Observer.NewSpan(mm.ctx, obs.SpanKindInternal, pushEvent)
			defer eventSpan.End()

			log := logrus.WithContext(eventCtx).WithFields(logrus.Fields{
				"agent_id":    aid,
				"group_id":    gid,
				"policy_id":   pid,
				"module_name": mname,
				"event_info":  info,
			})
			log.Debug("exec push event action")

			if aid == "" || aid != mm.agentID {
				log.Warn("failed to get agent ID from arguments")
				return false
			}

			err := mm.sendAction(eventCtx, pushEvent, &protoagent.ActionPushEvent{
				ModuleName: &mname,
				GroupId:    &gid,
				PolicyId:   &pid,
				EventInfo:  &info,
			})
			if err != nil {
				// TODO: here need persist event and retry sending it later
				log.WithError(err).Error("failed to send event to server side")
				return false
			}

			return true
		},
	})
	luar.GoToLua(state.L, mm.version)
	state.L.SetGlobal("__version")
	luar.GoToLua(state.L, mm.agentID)
	state.L.SetGlobal("__aid")
	luar.GoToLua(state.L, gid)
	state.L.SetGlobal("__gid")
	luar.GoToLua(state.L, pid)
	state.L.SetGlobal("__pid")

	getBaseFields := func() logrus.Fields {
		return logrus.Fields{
			"group_id":    gid,
			"policy_id":   pid,
			"module_name": mname,
		}
	}
	if err := state.RegisterLogger(logrus.GetLevel(), getBaseFields()); err != nil {
		return fmt.Errorf("failed to register logger for modules: %w", err)
	}

	fields := getBaseFields()
	fields["agent_id"] = mm.agentID
	if err := state.RegisterMeter(fields); err != nil {
		return fmt.Errorf("failed to register the metrics gatherer for modules: %w", err)
	}

	if u, err := mm.parseURLString(mm.connectionString); err != nil {
		logrus.WithContext(state.Context()).WithError(err).Error("failed to prepare sconn")
	} else {
		ips, err := net.LookupIP(u.Hostname())
		if err != nil {
			logrus.WithContext(state.Context()).WithError(err).Error("failed to lookup server ips")
		}
		state.L.CreateTable(0, 4)
		state.L.PushString(u.Scheme)
		state.L.SetField(-2, "scheme")
		state.L.PushString(u.Hostname())
		state.L.SetField(-2, "host")
		state.L.PushString(u.Port())
		state.L.SetField(-2, "port")
		state.L.NewTable()
		for idx, ip := range ips {
			state.L.PushInteger(int64(idx + 1))
			state.L.PushString(ip.String())
			state.L.RawSet(-3)
		}
		state.L.SetField(-2, "ips")
		state.L.SetGlobal("__sconn")
	}

	return nil
}

// UnregisterLuaAPI is function that unregisters extra API function for each of type service
func (mm *MainModule) UnregisterLuaAPI(state *lua.State, _ *loader.ModuleConfig) error {
	luar.Register(state.L, luaFuncTableNameAPI, luar.Map{})

	return nil
}

var (
	errTracerNotInitialized = errors.New("tracer config not initialized")
	errMeterNotInitialized  = errors.New("meter config not initialized")
)

func (mm *MainModule) registerUploadCallbacks() error {
	if mm.tracerConfigClient == nil {
		return errTracerNotInitialized
	}
	const name = "push_obs_packet"
	mm.tracerConfigClient.UploadCallback = func(ctx context.Context, traces [][]byte) error {
		return mm.sendAction(ctx, name, &protoagent.ObsPacket{
			Traces: traces,
		})
	}
	if mm.meterConfigClient == nil {
		return errMeterNotInitialized
	}
	mm.meterConfigClient.UploadCallback = func(ctx context.Context, metrics [][]byte) error {
		return mm.sendAction(ctx, name, &protoagent.ObsPacket{
			Metrics: metrics,
		})
	}
	return nil
}

func (mm *MainModule) unregisterUploadCallbacks() error {
	if mm.tracerConfigClient == nil {
		return errTracerNotInitialized
	}
	mm.tracerConfigClient.UploadCallback = nil
	if mm.meterConfigClient == nil {
		return errMeterNotInitialized
	}
	mm.meterConfigClient.UploadCallback = nil
	return nil
}
