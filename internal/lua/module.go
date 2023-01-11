package lua

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vxcontrol/luar"
	"google.golang.org/protobuf/proto"

	"soldr/internal/observability"
	"soldr/internal/protoagent"
	"soldr/internal/vxproto"
)

const (
	defSyncTime = 5 * time.Second // time to renew internal logging span
)

type recvCallbacks struct {
	recvData   *luaCallback
	recvText   *luaCallback
	recvFile   *luaCallback
	recvMsg    *luaCallback
	recvAction *luaCallback
	controlMsg *luaCallback
}

type notificationType int32

// Enumerate control message types
const (
	notifUseSyncMode  notificationType = 0
	notifUseAsyncMode notificationType = 1
)

// Module is struct that used for internal communication logic with Lua state
type Module struct {
	state    *State
	logger   *logrus.Entry
	packet   *vxproto.Packet
	packetMX *sync.Mutex
	socket   vxproto.IModuleSocket
	result   string
	cbs      recvCallbacks
	syncTime time.Time
	waitTime int64
	wgRun    sync.WaitGroup
	agents   map[string]*vxproto.AgentInfo
	args     map[string][]string
	quit     chan struct{}
	notifier chan notificationType
	closed   bool
}

// IsClose is nonblocked function which check a state of module
func (m *Module) IsClose() bool {
	return m.closed || m.state == nil
}

// GetResult is nonblocked function which return result from module
func (m *Module) GetResult() string {
	return m.result
}

// Start is function which prepare state for module
func (m *Module) Start() {
	if m.state == nil {
		return
	}
	if m.closed && m.state.closed {
		m.quit = make(chan struct{})
		m.notifier = make(chan notificationType)
		m.closed = false
	} else {
		return
	}

	m.wgRun.Add(1)
	go func() {
		defer m.wgRun.Done()
		m.recvPacket()
	}()
	m.logger.WithContext(m.state.ctx).Info("the module was started")
	defer func(m *Module) {
		m.logger.WithContext(m.state.ctx).Info("the module was stopped")
	}(m)

	var err error
	m.result = ""
	m.wgRun.Add(1)
	defer m.wgRun.Done()
	for {
		if m.result, err = m.state.Exec(); err == nil || m.closed {
			break
		}
		const msg = "error executing the module code on the lua state"
		m.logger.WithContext(m.state.ctx).WithError(err).WithField("result", m.result).Error(msg)
		time.Sleep(time.Second * time.Duration(5))
		m.state.L.SetTop(0)
		observability.Observer.SpanFromContext(m.state.ctx).End()
		m.state.ctx, _ = observability.Observer.NewSpan(context.Background(), observability.SpanKindInternal, "lua_state")
	}
}

// Stop closes module state
func (m *Module) Stop(stopReason string) {
	if m.closed || m.state == nil {
		return
	}

	m.logger.WithContext(m.state.ctx).Infof("the module wants to stop: %s", stopReason)
	defer m.logger.WithContext(m.state.ctx).Info("the module stopping has done")

	m.closed = true
	if _, err := m.controlMsgCb(m.state.ctx, "quit", stopReason); err != nil {
		m.logger.WithContext(m.state.ctx).WithError(err).Error("failed to send control message")
	}
	close(m.quit)
	close(m.notifier)

	m.wgRun.Wait()
	m.delCbs([]interface{}{"data", "text", "file", "msg", "control"})
}

// Close releases Lua state for a module
func (m *Module) Close(stopReason string) {
	if m.state == nil {
		return
	}

	m.Stop(stopReason)
	luar.Register(m.state.L, "__api", luar.Map{})
	luar.Register(m.state.L, "__agents", luar.Map{})
	luar.Register(m.state.L, "__routes", luar.Map{})
	luar.Register(m.state.L, "__imc", luar.Map{})
	m.state = nil
}

// SetAgents is function for storing agent list into module state
func (m *Module) SetAgents(agents map[string]*vxproto.AgentInfo) {
	m.agents = agents
}

// ControlMsg is function for send control message to module state
func (m *Module) ControlMsg(ctx context.Context, mtype, data string) bool {
	if m.closed || m.state == nil {
		return false
	}

	m.logger.WithContext(ctx).Debugf("control message %s received: '%s'", mtype, data)
	res, err := m.controlMsgCb(ctx, mtype, data)
	if err != nil {
		m.logger.WithContext(ctx).WithError(err).
			Errorf("failed to execute the control msg %s callback", mtype)
	}
	return res
}

// Set timeout for all blocked functions
// If used timeout variable in non-zero value, it will wake up after timeout
// Timeout variable uses in milliseconds and -1 value means infinity
func (m *Module) setRecvTimeout(timeout int64) {
	m.waitTime = timeout
}

// await is blocked function which wait a close of module
// If used timeout variable in non-zero value, it will wake up after timeout
// Timeout variable uses in milliseconds and -1 value means infinity
func (m *Module) await(timeout int64) {
	if m.closed {
		return
	}

	renewCtx := func() {
		if time.Since(m.syncTime) >= defSyncTime {
			observability.Observer.SpanFromContext(m.state.ctx).End()
			m.state.ctx, _ = observability.Observer.NewSpan(context.Background(), observability.SpanKindInternal, "lua_state")
			m.syncTime = time.Now()
		}
	}
	renewCtx()

	timer := time.NewTimer(time.Millisecond * time.Duration(timeout))
	defer timer.Stop()

	m.state.L.Unlock()
	defer m.state.L.Lock()
	runtime.Gosched()

	if timeout >= 0 {
		select {
		case <-timer.C:
		case <-m.quit:
		}
	} else {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				renewCtx()
				continue
			case <-m.quit:
			}
			break
		}
	}
}

func (m *Module) getName() string {
	return m.socket.GetName()
}

func (m *Module) getOS() string {
	return runtime.GOOS
}

func (m *Module) getArch() string {
	return runtime.GOARCH
}

func (m *Module) copyAgent(a *vxproto.AgentInfo) vxproto.AgentInfo {
	ca := *a
	if a.Info != nil {
		ca.Info = proto.Clone(a.Info).(*protoagent.Information)
		if a.Info.Os != nil {
			ca.Info.Os = proto.Clone(a.Info.Os).(*protoagent.Information_OS)
		}
		ca.Info.Net = proto.Clone(a.Info.Net).(*protoagent.Information_Net)
		for _, user := range a.Info.Users {
			clone := proto.Clone(user).(*protoagent.Information_User)
			ca.Info.Users = append(ca.Info.Users, clone)
		}
	}
	return ca
}

func (m *Module) getAgents() map[string]vxproto.AgentInfo {
	agents := make(map[string]vxproto.AgentInfo)
	for t, a := range m.agents {
		agents[t] = m.copyAgent(a)
	}
	return agents
}

func (m *Module) getAgentsCount() int {
	return len(m.agents)
}

func (m *Module) getAgentsByID(aid string) map[string]vxproto.AgentInfo {
	filtered := make(map[string]vxproto.AgentInfo)
	for src, ainfo := range m.agents {
		if ainfo.ID == aid {
			filtered[src] = m.copyAgent(ainfo)
		}
	}
	return filtered
}

func (m *Module) getAgentsBySrc(srcToken string) map[string]vxproto.AgentInfo {
	filtered := make(map[string]vxproto.AgentInfo)
	for src, ainfo := range m.agents {
		if ainfo.Src == srcToken {
			filtered[src] = m.copyAgent(ainfo)
		}
	}
	return filtered
}

func (m *Module) getAgentsByDst(dstToken string) map[string]vxproto.AgentInfo {
	filtered := make(map[string]vxproto.AgentInfo)
	for src, ainfo := range m.agents {
		if ainfo.Dst == dstToken {
			filtered[src] = m.copyAgent(ainfo)
		}
	}
	return filtered
}

func (m *Module) getIMCToken() string {
	return m.socket.GetIMCToken()
}

func (m *Module) getIMCTokenInfo(token string) (string, string, bool) {
	ms := m.socket.GetIMCModule(token)
	if ms == nil {
		return "", "", false
	}
	return ms.GetName(), ms.GetGroupID(), true
}

func (m *Module) isIMCTokenExist(token string) bool {
	return m.socket.GetIMCModule(token) != nil
}

func (m *Module) makeIMCToken(name, gid string) string {
	return m.socket.MakeIMCToken(name, gid)
}

func (m *Module) getIMCGroupIDs() []string {
	return m.socket.GetIMCGroupIDs()
}

func (m *Module) getIMCModuleIDs() []string {
	return m.socket.GetIMCModuleIDs()
}

func (m *Module) getIMCGroupIDsByMID(mid string) []string {
	return m.socket.GetIMCGroupIDsByMID(mid)
}

func (m *Module) getIMCModuleIDsByGID(gid string) []string {
	return m.socket.GetIMCModuleIDsByGID(gid)
}

func (m *Module) getRoutes() map[string]string {
	return m.socket.GetRoutes()
}

func (m *Module) getRoutesCount() int {
	return len(m.socket.GetRoutes())
}

func (m *Module) getRoute(dst string) string {
	return m.socket.GetRoute(dst)
}

func (m *Module) addRoute(dst, src string) bool {
	l := m.logger.WithContext(m.state.ctx).WithFields(logrus.Fields{
		"src": src,
		"dst": dst,
	})
	if err := m.socket.AddRoute(dst, src); err != nil {
		l.WithError(err).Error("failed to add a new route")
		return false
	}
	l.Debug("the module has added a new route")
	return true
}

func (m *Module) delRoute(dst string) bool {
	l := m.logger.WithContext(m.state.ctx).WithFields(logrus.Fields{
		"dst": dst,
	})
	if err := m.socket.DelRoute(dst); err != nil {
		l.WithError(err).Error("failed to delete a route")
		return false
	}
	l.Debug("the module has deleted a route")
	return true
}

func (m *Module) tryPacketUnlock(dst string) {
	m.packetMX.Lock()
	defer m.packetMX.Unlock()

	if m.packet != nil && (m.packet.Src == dst || dst == "") {
		m.packet.SetAck()
		m.packet = nil
	}
}

func (m *Module) sendDataTo(dst, data string) bool {
	if len(data) == 0 {
		return false
	}

	sdata := &vxproto.Data{
		Data: []byte(data),
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "data_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len": len(data),
		"dst": dst,
	})

	m.tryPacketUnlock(dst)
	m.state.L.Unlock()
	err := m.socket.SendDataTo(ctx, dst, sdata)
	m.state.L.Lock()

	if err != nil {
		l.WithError(err).Error("failed to send data")
		return false
	}
	l.Debug("the module has sent data")
	return true
}

func (m *Module) sendFileTo(dst, data, name string) bool {
	if len(data) == 0 || name == "" {
		return false
	}

	sfile := &vxproto.File{
		Data: []byte(data),
		Name: name,
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "file_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len":  len(data),
		"name": name,
		"dst":  dst,
	})

	m.tryPacketUnlock(dst)
	m.state.L.Unlock()
	err := m.socket.SendFileTo(ctx, dst, sfile)
	m.state.L.Lock()

	if err != nil {
		l.WithError(err).Error("failed to send data as a file")
		return false
	}
	l.Debug("the module has sent data as a file")
	return true
}

func (m *Module) sendFileFromFSTo(dst, path, name string) bool {
	if path == "" || name == "" {
		return false
	}

	sfile := &vxproto.File{
		Path: path,
		Name: name,
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "file_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"path": path,
		"name": name,
		"dst":  dst,
	})

	m.tryPacketUnlock(dst)
	m.state.L.Unlock()
	err := m.socket.SendFileTo(ctx, dst, sfile)
	m.state.L.Lock()

	if err != nil {
		l.WithError(err).Error("failed to send a file from the file system")
		return false
	}
	l.Debug("the module has sent a file from the file system")
	return true
}

func (m *Module) sendTextTo(dst, data, name string) bool {
	if len(data) == 0 || name == "" {
		return false
	}

	stext := &vxproto.Text{
		Data: []byte(data),
		Name: name,
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "text_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len":  len(data),
		"name": name,
		"dst":  dst,
	})

	m.tryPacketUnlock(dst)
	m.state.L.Unlock()
	err := m.socket.SendTextTo(ctx, dst, stext)
	m.state.L.Lock()

	if err != nil {
		l.WithError(err).Error("failed to send data as a text")
		return false
	}
	l.Debug("the module has sent data as a text")
	return true
}

func (m *Module) sendMsgTo(dst, data string, mtype int32) bool {
	if len(data) == 0 || mtype < 0 || mtype > 3 {
		return false
	}

	msg := &vxproto.Msg{
		Data:  []byte(data),
		MType: vxproto.MsgType(mtype),
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "msg_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len":  len(data),
		"type": vxproto.MsgType(mtype).String(),
		"dst":  dst,
	})

	m.tryPacketUnlock(dst)
	m.state.L.Unlock()
	err := m.socket.SendMsgTo(ctx, dst, msg)
	m.state.L.Lock()

	if err != nil {
		l.WithError(err).Error("failed to send a message")
		return false
	}
	l.Debug("the module has sent a message")
	return true
}

func (m *Module) sendActionTo(dst, data, name string) bool {
	if len(data) == 0 || len(name) == 0 {
		return false
	}

	act := &vxproto.Action{
		Data: []byte(data),
		Name: name,
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "action_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len":  len(data),
		"name": name,
		"dst":  dst,
	})

	m.tryPacketUnlock(dst)
	m.state.L.Unlock()
	err := m.socket.SendActionTo(ctx, dst, act)
	m.state.L.Lock()

	if err != nil {
		l.WithError(err).Error("failed to send an action")
		return false
	}
	l.Debug("the module has sent an action")
	return true
}

func (m *Module) asyncSendDataTo(dst, data string, callback interface{}) bool {
	if len(data) == 0 {
		return false
	}

	var cb *luar.LuaObject
	sdata := &vxproto.Data{
		Data: []byte(data),
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "async_data_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len": len(data),
		"dst": dst,
	})

	if callback != nil {
		var ok bool
		if cb, ok = callback.(*luar.LuaObject); !ok {
			l.Error("failed to convert callback to lua object type")
			return false
		}
	}

	m.wgRun.Add(1)
	go func() {
		defer m.wgRun.Done()
		m.tryPacketUnlock(dst)
		err := m.socket.SendDataTo(ctx, dst, sdata)
		if cb != nil {
			m.state.L.Lock()
			if errCall := cb.Call(nil, err == nil); errCall != nil {
				l.WithError(errCall).Error("failed to exec of async send lua callback")
			}
			cb.Close()
			m.state.L.Unlock()
		}
		if err != nil {
			l.WithError(err).Error("failed to async send data")
			return
		}
		l.Debug("the module has async sent data")
	}()

	return true
}

func (m *Module) asyncSendFileTo(dst, data, name string, callback interface{}) bool {
	if len(data) == 0 || name == "" {
		return false
	}

	var cb *luar.LuaObject
	sfile := &vxproto.File{
		Data: []byte(data),
		Name: name,
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "async_file_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len":  len(data),
		"name": name,
		"dst":  dst,
	})

	if callback != nil {
		var ok bool
		if cb, ok = callback.(*luar.LuaObject); !ok {
			l.Error("failed to convert callback to lua object type")
			return false
		}
	}

	m.wgRun.Add(1)
	go func() {
		defer m.wgRun.Done()
		m.tryPacketUnlock(dst)
		err := m.socket.SendFileTo(ctx, dst, sfile)
		if cb != nil {
			m.state.L.Lock()
			if errCall := cb.Call(nil, err == nil); errCall != nil {
				l.WithError(errCall).Error("failed to exec of async send lua callback")
			}
			cb.Close()
			m.state.L.Unlock()
		}
		if err != nil {
			l.WithError(err).Error("failed to async send data as a file")
			return
		}
		l.Debug("the module has async sent data as a file")
	}()

	return true
}

func (m *Module) asyncSendFileFromFSTo(dst, path, name string, callback interface{}) bool {
	if path == "" || name == "" {
		return false
	}

	var cb *luar.LuaObject
	sfile := &vxproto.File{
		Path: path,
		Name: name,
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "async_file_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"path": path,
		"name": name,
		"dst":  dst,
	})

	if callback != nil {
		var ok bool
		if cb, ok = callback.(*luar.LuaObject); !ok {
			l.Error("failed to convert callback to lua object type")
			return false
		}
	}

	m.wgRun.Add(1)
	go func() {
		defer m.wgRun.Done()
		m.tryPacketUnlock(dst)
		err := m.socket.SendFileTo(ctx, dst, sfile)
		if cb != nil {
			m.state.L.Lock()
			if errCall := cb.Call(nil, err == nil); errCall != nil {
				l.WithError(errCall).Error("failed to exec of async send lua callback")
			}
			cb.Close()
			m.state.L.Unlock()
		}
		if err != nil {
			l.WithError(err).Error("failed to async send a file from the file system")
			return
		}
		l.Debug("the module has async sent a file from the file system")
	}()

	return true
}

func (m *Module) asyncSendTextTo(dst, data, name string, callback interface{}) bool {
	if len(data) == 0 || name == "" {
		return false
	}

	var cb *luar.LuaObject
	stext := &vxproto.Text{
		Data: []byte(data),
		Name: name,
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "async_text_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len":  len(data),
		"name": name,
		"dst":  dst,
	})

	if callback != nil {
		var ok bool
		if cb, ok = callback.(*luar.LuaObject); !ok {
			l.Error("failed to convert callback to lua object type")
			return false
		}
	}

	m.wgRun.Add(1)
	go func() {
		defer m.wgRun.Done()
		m.tryPacketUnlock(dst)
		err := m.socket.SendTextTo(ctx, dst, stext)
		if cb != nil {
			m.state.L.Lock()
			if errCall := cb.Call(nil, err == nil); errCall != nil {
				l.WithError(errCall).Error("failed to exec of async send lua callback")
			}
			cb.Close()
			m.state.L.Unlock()
		}
		if err != nil {
			l.WithError(err).Error("failed to async send data as a text")
			return
		}
		l.Debug("the module has async sent data as a text")
	}()

	return true
}

func (m *Module) asyncSendMsgTo(dst, data string, mtype int32, callback interface{}) bool {
	if len(data) == 0 || mtype < 0 || mtype > 3 {
		return false
	}

	var cb *luar.LuaObject
	msg := &vxproto.Msg{
		Data:  []byte(data),
		MType: vxproto.MsgType(mtype),
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "async_msg_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len":  len(data),
		"type": vxproto.MsgType(mtype).String(),
		"dst":  dst,
	})

	if callback != nil {
		var ok bool
		if cb, ok = callback.(*luar.LuaObject); !ok {
			l.Error("failed to convert callback to lua object type")
			return false
		}
	}

	m.wgRun.Add(1)
	go func() {
		defer m.wgRun.Done()
		m.tryPacketUnlock(dst)
		err := m.socket.SendMsgTo(ctx, dst, msg)
		if cb != nil {
			m.state.L.Lock()
			if errCall := cb.Call(nil, err == nil); errCall != nil {
				l.WithError(errCall).Error("failed to exec of async send lua callback")
			}
			cb.Close()
			m.state.L.Unlock()
		}
		if err != nil {
			l.WithError(err).Error("failed to async send a message")
			return
		}
		l.Debug("the module has async sent a message")
	}()

	return true
}

func (m *Module) asyncSendActionTo(dst, data, name string, callback interface{}) bool {
	if len(data) == 0 || len(name) == 0 {
		return false
	}

	var cb *luar.LuaObject
	act := &vxproto.Action{
		Data: []byte(data),
		Name: name,
	}

	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindProducer, "async_action_packet_sender")
	defer span.End()

	l := m.logger.WithContext(ctx).WithFields(logrus.Fields{
		"len":  len(data),
		"name": name,
		"dst":  dst,
	})

	if callback != nil {
		var ok bool
		if cb, ok = callback.(*luar.LuaObject); !ok {
			l.Error("failed to convert callback to lua object type")
			return false
		}
	}

	m.wgRun.Add(1)
	go func() {
		defer m.wgRun.Done()
		m.tryPacketUnlock(dst)
		err := m.socket.SendActionTo(ctx, dst, act)
		if cb != nil {
			m.state.L.Lock()
			if errCall := cb.Call(nil, err == nil); errCall != nil {
				l.WithError(errCall).Error("failed to exec of async send lua callback")
			}
			cb.Close()
			m.state.L.Unlock()
		}
		if err != nil {
			l.WithError(err).Error("failed to async send an action")
			return
		}
		l.Debug("the module has async sent an action")
	}()

	return true
}

func (m *Module) recvDataCb(ctx context.Context, src string, data *vxproto.Data) (bool, error) {
	if m.cbs.recvData != nil {
		var res bool
		err := m.cbs.recvData.Call(ctx, &res, src, string(data.Data[:]))
		return res, err
	}

	return false, nil
}

func (m *Module) recvFileCb(ctx context.Context, src string, file *vxproto.File) (bool, error) {
	if m.cbs.recvFile != nil {
		var res bool
		err := m.cbs.recvFile.Call(ctx, &res, src, file.Path, file.Name)
		return res, err
	}

	return false, nil
}

func (m *Module) recvTextCb(ctx context.Context, src string, text *vxproto.Text) (bool, error) {
	if m.cbs.recvText != nil {
		var res bool
		err := m.cbs.recvText.Call(ctx, &res, src, string(text.Data[:]), text.Name)
		return res, err
	}

	return false, nil
}

func (m *Module) recvMsgCb(ctx context.Context, src string, msg *vxproto.Msg) (bool, error) {
	if m.cbs.recvMsg != nil {
		var res bool
		err := m.cbs.recvMsg.Call(ctx, &res, src, string(msg.Data[:]), int32(msg.MType))
		return res, err
	}

	return false, nil
}

func (m *Module) recvActionCb(ctx context.Context, src string, act *vxproto.Action) (bool, error) {
	if m.cbs.recvAction != nil {
		var res bool
		err := m.cbs.recvAction.Call(ctx, &res, src, string(act.Data[:]), act.Name)
		return res, err
	}

	return false, nil
}

func (m *Module) controlMsgCb(ctx context.Context, mtype, data string) (bool, error) {
	if m.cbs.controlMsg != nil {
		var res bool
		err := m.cbs.controlMsg.Call(ctx, &res, mtype, data)
		return res, err
	}

	return false, nil
}

const recvDataErrMsg = "failed to receive data"

func (m *Module) recvData() (string, string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "data_packet_receiver")
	defer span.End()

	m.tryPacketUnlock("")
	src, data, err := m.socket.RecvData(ctx, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithError(err).Error(recvDataErrMsg)
		return "", "", false
	}

	return src, string(data.Data[:]), true
}

const recvFileErrMsg = "failed to receive a file"

func (m *Module) recvFile() (string, string, string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "file_packet_receiver")
	defer span.End()

	m.tryPacketUnlock("")
	src, file, err := m.socket.RecvFile(ctx, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithError(err).Error(recvFileErrMsg)
		return "", "", "", false
	}

	return src, file.Path, file.Name, true
}

const recvTextErrMsg = "failed to receive text"

func (m *Module) recvText() (string, string, string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "text_packet_receiver")
	defer span.End()

	m.tryPacketUnlock("")
	src, text, err := m.socket.RecvText(ctx, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithError(err).Error(recvTextErrMsg)
		return "", "", "", false
	}

	return src, string(text.Data[:]), text.Name, true
}

const recvMsgErrMsg = "failed to received a message"

func (m *Module) recvMsg() (string, string, int32, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "msg_packet_receiver")
	defer span.End()

	m.tryPacketUnlock("")
	src, msg, err := m.socket.RecvMsg(ctx, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithError(err).Error(recvMsgErrMsg)
		return "", "", 0, false
	}

	return src, string(msg.Data[:]), int32(msg.MType), true
}

const recvActionErrMsg = "failed to receive an action"

func (m *Module) recvAction() (string, string, string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "action_packet_receiver")
	defer span.End()

	m.tryPacketUnlock("")
	src, act, err := m.socket.RecvAction(ctx, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithError(err).Error(recvActionErrMsg)
		return "", "", "", false
	}

	return src, string(act.Data[:]), act.Name, true
}

func (m *Module) recvDataFrom(src string) (string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "data_packet_receiver")
	defer span.End()

	m.tryPacketUnlock(src)
	data, err := m.socket.RecvDataFrom(ctx, src, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithField("src", src).WithError(err).
			Error(recvDataErrMsg)
		return "", false
	}

	return string(data.Data[:]), true
}

func (m *Module) recvFileFrom(src string) (string, string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "file_packet_receiver")
	defer span.End()

	m.tryPacketUnlock(src)
	file, err := m.socket.RecvFileFrom(ctx, src, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithField("src", src).WithError(err).
			Error(recvFileErrMsg)
		return "", "", false
	}

	return file.Path, file.Name, true
}

func (m *Module) recvTextFrom(src string) (string, string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "text_packet_receiver")
	defer span.End()

	m.tryPacketUnlock(src)
	text, err := m.socket.RecvTextFrom(ctx, src, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithField("src", src).WithError(err).
			Error(recvTextErrMsg)
		return "", "", false
	}

	return string(text.Data[:]), text.Name, true
}

func (m *Module) recvMsgFrom(src string) (string, int32, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "msg_packet_receiver")
	defer span.End()

	m.tryPacketUnlock(src)
	msg, err := m.socket.RecvMsgFrom(ctx, src, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithField("src", src).WithError(err).
			Error(recvMsgErrMsg)
		return "", 0, false
	}

	return string(msg.Data[:]), int32(msg.MType), true
}

func (m *Module) recvActionFrom(src string) (string, string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "action_packet_receiver")
	defer span.End()

	m.tryPacketUnlock(src)
	act, err := m.socket.RecvActionFrom(ctx, src, m.waitTime)
	if err != nil {
		m.logger.WithContext(ctx).WithField("src", src).WithError(err).
			Error(recvActionErrMsg)
		return "", "", false
	}

	return string(act.Data[:]), act.Name, true
}

func (m *Module) addCbs(callbackTable interface{}) bool {
	callbackMap, ok := callbackTable.(map[string]interface{})
	if !ok {
		return false
	}

	for name, callback := range callbackMap {
		switch name {
		case "data":
			if cb, ok := callback.(*luar.LuaObject); ok {
				cb.Push()
				m.cbs.recvData = newLuaCallback(m.state.L)
				cb.Close()
				m.logger.WithContext(m.state.ctx).Debug("the module has added a receive data callback")
			}
		case "text":
			if cb, ok := callback.(*luar.LuaObject); ok {
				cb.Push()
				m.cbs.recvText = newLuaCallback(m.state.L)
				cb.Close()
				m.logger.WithContext(m.state.ctx).Debug("the module has added a receive text callback")
			}
		case "file":
			if cb, ok := callback.(*luar.LuaObject); ok {
				cb.Push()
				m.cbs.recvFile = newLuaCallback(m.state.L)
				cb.Close()
				m.logger.WithContext(m.state.ctx).Debug("the module has added a receive file callback")
			}
		case "msg":
			if cb, ok := callback.(*luar.LuaObject); ok {
				cb.Push()
				m.cbs.recvMsg = newLuaCallback(m.state.L)
				cb.Close()
				m.logger.WithContext(m.state.ctx).Debug("the module has added a receive message callback")
			}
		case "action":
			if cb, ok := callback.(*luar.LuaObject); ok {
				cb.Push()
				m.cbs.recvAction = newLuaCallback(m.state.L)
				cb.Close()
				m.logger.WithContext(m.state.ctx).Debug("the module has added a receive action callback")
			}
		case "control":
			if cb, ok := callback.(*luar.LuaObject); ok {
				cb.Push()
				m.cbs.controlMsg = newLuaCallback(m.state.L)
				cb.Close()
				m.logger.WithContext(m.state.ctx).Debug("the module has added a receive control message callback")
			}
		default:
		}
	}

	return true
}

func (m *Module) delCbs(CallbackTable interface{}) bool {
	callbackMap, ok := CallbackTable.([]interface{})
	if !ok {
		return false
	}

	for _, name := range callbackMap {
		switch name.(string) {
		case "data":
			if m.cbs.recvData != nil {
				m.cbs.recvData.Close()
				m.cbs.recvData = nil
				m.logger.WithContext(m.state.ctx).Debug("the module deleted receive data callback")
			}
		case "text":
			if m.cbs.recvText != nil {
				m.cbs.recvText.Close()
				m.cbs.recvText = nil
				m.logger.WithContext(m.state.ctx).Debug("the module deleted receive text callback")
			}
		case "file":
			if m.cbs.recvFile != nil {
				m.cbs.recvFile.Close()
				m.cbs.recvFile = nil
				m.logger.WithContext(m.state.ctx).Debug("the module deleted receive file callback")
			}
		case "msg":
			if m.cbs.recvMsg != nil {
				m.cbs.recvMsg.Close()
				m.cbs.recvMsg = nil
				m.logger.WithContext(m.state.ctx).Debug("the module deleted receive message callback")
			}
		case "action":
			if m.cbs.recvAction != nil {
				m.cbs.recvAction.Close()
				m.cbs.recvAction = nil
				m.logger.WithContext(m.state.ctx).Debug("the module deleted receive action callback")
			}
		case "control":
			if m.cbs.controlMsg != nil {
				m.cbs.controlMsg.Close()
				m.cbs.controlMsg = nil
				m.logger.WithContext(m.state.ctx).Debug("the module deleted receive control message callback")
			}
		default:
		}
	}

	return true
}

func (m *Module) useSyncMode() bool {
	if m.closed {
		return false
	}
	m.notifier <- notifUseSyncMode
	return true
}

func (m *Module) useAsyncMode() bool {
	if m.closed {
		return false
	}
	m.notifier <- notifUseAsyncMode
	return true
}

func (m *Module) recvPacket() error {
	defer func(m *Module) {
		m.logger.WithContext(m.state.ctx).Info("packet receiver was stopped")
	}(m)

	syncMode := false
	fakeReceiver := make(chan *vxproto.Packet)
	defer close(fakeReceiver)

	m.logger.WithContext(m.state.ctx).Info("packet receiver was started")
	socketReceiver := m.socket.GetReceiver()
	if socketReceiver == nil {
		const errMsg = "failed to initialize a packet receiver"
		m.logger.WithContext(m.state.ctx).Error(errMsg)
		return fmt.Errorf(errMsg)
	}
	wrapError := func(res bool, err error, name string, logger *logrus.Entry) {
		if err != nil {
			logger.WithError(err).Errorf("failed to execute '%s'", name)
		} else if !res {
			logger.Warnf("lua callback '%s' return false as a execution result", name)
		}
	}
	setPacket := func(packet *vxproto.Packet) {
		m.packetMX.Lock()
		defer m.packetMX.Unlock()

		if m.packet != nil {
			// use default acknowledgement callback before reset
			m.packet.SetAck()
		}
		m.packet = packet
	}
	handleNotification := func(ctx context.Context, notice notificationType) {
		switch notice {
		case notifUseSyncMode:
			syncMode = true
			m.logger.WithContext(ctx).Info("packet receiver changed to synchronous mode")
		case notifUseAsyncMode:
			syncMode = false
			m.logger.WithContext(ctx).Info("packet receiver changed to asynchronous mode")
		}
	}
	handleQuit := func(ctx context.Context) {
		m.logger.WithContext(ctx).Info("got signal to quit from channel")
	}
	for !m.closed {
		var (
			packet   *vxproto.Packet
			receiver chan *vxproto.Packet
		)

		if syncMode {
			receiver = fakeReceiver
		} else {
			receiver = socketReceiver
		}

		select {
		case packet = <-receiver:
		case notice := <-m.notifier:
			handleNotification(m.state.ctx, notice)
			continue
		case <-m.quit:
			handleQuit(m.state.ctx)
			return nil
		}

		if packet == nil {
			const errMsg = "failed to receive a packet"
			m.logger.WithContext(m.state.ctx).Error(errMsg)
			return fmt.Errorf(errMsg)
		}

		// store packet to unlock another lua state when this state wants send return packet
		setPacket(packet)

		logger := m.logger.WithFields(logrus.Fields{
			"module": packet.Module,
			"type":   packet.PType.String(),
			"src":    packet.Src,
			"dst":    packet.Dst,
		}).WithContext(packet.Context())
		logger.Debug("packet receiver got new packet")
		switch packet.PType {
		case vxproto.PTData:
			res, err := m.recvDataCb(packet.Context(), packet.Src, packet.GetData())
			wrapError(res, err, "the recvDataCb", logger)
		case vxproto.PTFile:
			res, err := m.recvFileCb(packet.Context(), packet.Src, packet.GetFile())
			wrapError(res, err, "the recvFileCb", logger)
		case vxproto.PTText:
			res, err := m.recvTextCb(packet.Context(), packet.Src, packet.GetText())
			wrapError(res, err, "the recvTextCb", logger)
		case vxproto.PTMsg:
			res, err := m.recvMsgCb(packet.Context(), packet.Src, packet.GetMsg())
			wrapError(res, err, "the recvMsgCb", logger)
		case vxproto.PTAction:
			res, err := m.recvActionCb(packet.Context(), packet.Src, packet.GetAction())
			wrapError(res, err, "the recvActionCb", logger)
		case vxproto.PTControl:
			msg := packet.GetControlMsg()
			switch msg.MsgType {
			case vxproto.AgentConnected:
				m.agents[msg.AgentInfo.Dst] = msg.AgentInfo
				logger.Debug("agent connected to the module")
				res, err := m.controlMsgCb(packet.Context(), "agent_connected", msg.AgentInfo.Dst)
				wrapError(res, err, "agent connect callback", logger)
			case vxproto.AgentDisconnected:
				logger.Debug("agent disconnected from the module")
				res, err := m.controlMsgCb(packet.Context(), "agent_disconnected", msg.AgentInfo.Dst)
				wrapError(res, err, "agent disconnect callback", logger)
				delete(m.agents, msg.AgentInfo.Dst)
			case vxproto.StopModule:
				logger.Info("got packet with signal to stop module")
				setPacket(nil)
				return nil
			}
		}
		setPacket(nil)
	}

	return nil
}

func GetRunningExecutablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate the symbolic link to get the real path: %w", err)
	}
	return path, nil
}

func (m *Module) getRunningExecutablePath() (string, bool) {
	ctx, span := observability.Observer.NewSpan(m.state.ctx, observability.SpanKindConsumer, "file_packet_receiver")
	defer span.End()

	p, err := GetRunningExecutablePath()
	if err != nil {
		m.logger.WithContext(ctx).WithError(err).Error("failed to get path to the running executable file")
		return "", false
	}
	return p, true
}

// NewModule is function which constructed Module object
func NewModule(args map[string][]string, state *State, socket vxproto.IModuleSocket) (*Module, error) {
	if socket == nil {
		logrus.Error("failed to create a new module because the passed socket is nil")
		return nil, fmt.Errorf("passed socket is nil")
	}
	m := &Module{
		socket:   socket,
		state:    state,
		args:     args,
		packetMX: &sync.Mutex{},
		agents:   make(map[string]*vxproto.AgentInfo),
		closed:   true,
		logger: state.logger.WithFields(logrus.Fields{
			"component": "lua_module_" + socket.GetName(),
			"module":    socket.GetName(),
			"group_id":  socket.GetGroupID(),
		}),
		syncTime: time.Now(),
	}
	state.logger = m.logger

	luar.Register(state.L, "__api", luar.Map{
		// Functions
		"await":         m.await,
		"is_close":      m.IsClose,
		"get_name":      m.getName,
		"get_os":        m.getOS,
		"get_arch":      m.getArch,
		"get_exec_path": m.getRunningExecutablePath,
		"unsafe": luar.Map{
			"lock":   func() { m.state.L.Lock() },
			"unlock": func() { m.state.L.Unlock() },
		},

		"add_cbs":          m.addCbs,
		"del_cbs":          m.delCbs,
		"set_recv_timeout": m.setRecvTimeout,

		"use_sync_mode":  m.useSyncMode,
		"use_async_mode": m.useAsyncMode,

		"send_data_to":         m.sendDataTo,
		"send_file_to":         m.sendFileTo,
		"send_text_to":         m.sendTextTo,
		"send_msg_to":          m.sendMsgTo,
		"send_action_to":       m.sendActionTo,
		"send_file_from_fs_to": m.sendFileFromFSTo,

		"async_send_data_to":         m.asyncSendDataTo,
		"async_send_file_to":         m.asyncSendFileTo,
		"async_send_text_to":         m.asyncSendTextTo,
		"async_send_msg_to":          m.asyncSendMsgTo,
		"async_send_action_to":       m.asyncSendActionTo,
		"async_send_file_from_fs_to": m.asyncSendFileFromFSTo,

		"recv_data":   m.recvData,
		"recv_file":   m.recvFile,
		"recv_text":   m.recvText,
		"recv_msg":    m.recvMsg,
		"recv_action": m.recvAction,

		"recv_data_from":   m.recvDataFrom,
		"recv_file_from":   m.recvFileFrom,
		"recv_text_from":   m.recvTextFrom,
		"recv_msg_from":    m.recvMsgFrom,
		"recv_action_from": m.recvActionFrom,
	})

	luar.Register(state.L, "__agents", luar.Map{
		// Functions
		"dump":       m.getAgents,
		"count":      m.getAgentsCount,
		"get_by_id":  m.getAgentsByID,
		"get_by_src": m.getAgentsBySrc,
		"get_by_dst": m.getAgentsByDst,
	})

	luar.Register(state.L, "__routes", luar.Map{
		// Functions
		"dump":  m.getRoutes,
		"count": m.getRoutesCount,
		"get":   m.getRoute,
		"add":   m.addRoute,
		"del":   m.delRoute,
	})

	luar.Register(state.L, "__imc", luar.Map{
		// Functions
		"get_token":          m.getIMCToken,
		"get_info":           m.getIMCTokenInfo,
		"is_exist":           m.isIMCTokenExist,
		"make_token":         m.makeIMCToken,
		"get_groups":         m.getIMCGroupIDs,
		"get_modules":        m.getIMCModuleIDs,
		"get_groups_by_mid":  m.getIMCGroupIDsByMID,
		"get_modules_by_gid": m.getIMCModuleIDsByGID,
	})

	luar.GoToLua(state.L, args)
	state.L.SetGlobal("__args")

	// TODO: change it to native load function
	state.L.DoString(`
	io.stdout:setvbuf('no')

	function __api.async(f, ...)
		local glue = require("glue")
		__api.unsafe.unlock()
		t = glue.pack(f(...))
		__api.unsafe.lock()
		return glue.unpack(t)
	end

	function __api.sync(f, ...)
		local glue = require("glue")
		__api.unsafe.lock()
		t = glue.pack(f(...))
		__api.unsafe.unlock()
		return glue.unpack(t)
	end
	`)

	m.logger.WithContext(state.ctx).Info("the module was created")
	return m, nil
}
