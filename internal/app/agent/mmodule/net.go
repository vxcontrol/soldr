package mmodule

import (
	"context"
	"fmt"

	"github.com/Masterminds/semver/v3"
	"github.com/sirupsen/logrus"
	"google.golang.org/protobuf/proto"

	"soldr/internal/loader"
	obs "soldr/internal/observability"
	"soldr/internal/protoagent"
	"soldr/internal/system"
	"soldr/internal/utils"
	"soldr/internal/vxproto"
)

// responseAgent is function which send response to server
func (mm *MainModule) getStatusModules() *protoagent.ModuleStatusList {
	var modulesList protoagent.ModuleStatusList
	for _, id := range mm.loader.List() {
		ms := mm.loader.Get(id)
		if ms == nil {
			continue
		}

		mc, ok := mm.modules[id]
		if !ok {
			continue
		}

		var osList []*protoagent.Config_OS
		for osType, archList := range mc.OS {
			osList = append(osList, &protoagent.Config_OS{
				Type: utils.GetRef(osType),
				Arch: archList,
			})
		}
		config := &protoagent.Config{
			Os:               osList,
			GroupId:          utils.GetRef(mc.GroupID),
			PolicyId:         utils.GetRef(mc.PolicyID),
			Name:             utils.GetRef(mc.Name),
			Version:          utils.GetRef(mc.Version.String()),
			Actions:          mc.Actions,
			Events:           mc.Events,
			Fields:           mc.Fields,
			State:            utils.GetRef(mc.State),
			Template:         utils.GetRef(mc.Template),
			LastModuleUpdate: utils.GetRef(mc.LastModuleUpdate),
			LastUpdate:       utils.GetRef(mc.LastUpdate),
		}

		iconfig := &protoagent.ConfigItem{
			ConfigSchema:        utils.GetRef(mc.GetConfigSchema()),
			DefaultConfig:       utils.GetRef(mc.GetDefaultConfig()),
			CurrentConfig:       utils.GetRef(mc.GetCurrentConfig()),
			StaticDependencies:  utils.GetRef(mc.GetStaticDependencies()),
			DynamicDependencies: utils.GetRef(mc.GetDynamicDependencies()),
			FieldsSchema:        utils.GetRef(mc.GetFieldsSchema()),
			ActionConfigSchema:  utils.GetRef(mc.GetActionConfigSchema()),
			DefaultActionConfig: utils.GetRef(mc.GetDefaultActionConfig()),
			CurrentActionConfig: utils.GetRef(mc.GetCurrentActionConfig()),
			EventConfigSchema:   utils.GetRef(mc.GetEventConfigSchema()),
			DefaultEventConfig:  utils.GetRef(mc.GetDefaultEventConfig()),
			CurrentEventConfig:  utils.GetRef(mc.GetCurrentEventConfig()),
			SecureConfigSchema:  utils.GetRef(mc.GetSecureConfigSchema()),
			SecureDefaultConfig: utils.GetRef(mc.GetSecureDefaultConfig()),
			SecureCurrentConfig: utils.GetRef(mc.GetSecureCurrentConfig()),
		}

		moduleStatus := &protoagent.ModuleStatus{
			Name:       utils.GetRef(mc.Name),
			Config:     config,
			ConfigItem: iconfig,
			Status:     ms.GetStatus().Enum(),
		}
		modulesList.List = append(modulesList.List, moduleStatus)
	}

	return &modulesList
}

func (mm *MainModule) getModuleConfig(m *protoagent.Module) *loader.ModuleConfig {
	mci := &loader.ModuleConfigItem{
		ConfigSchema:        m.GetConfigItem().GetConfigSchema(),
		CurrentConfig:       m.GetConfigItem().GetCurrentConfig(),
		DefaultConfig:       m.GetConfigItem().GetDefaultConfig(),
		StaticDependencies:  m.GetConfigItem().GetStaticDependencies(),
		DynamicDependencies: m.GetConfigItem().GetDynamicDependencies(),
		FieldsSchema:        m.GetConfigItem().GetFieldsSchema(),
		ActionConfigSchema:  m.GetConfigItem().GetActionConfigSchema(),
		DefaultActionConfig: m.GetConfigItem().GetDefaultActionConfig(),
		CurrentActionConfig: m.GetConfigItem().GetCurrentActionConfig(),
		EventConfigSchema:   m.GetConfigItem().GetEventConfigSchema(),
		DefaultEventConfig:  m.GetConfigItem().GetDefaultEventConfig(),
		CurrentEventConfig:  m.GetConfigItem().GetCurrentEventConfig(),
		SecureConfigSchema:  m.GetConfigItem().GetSecureConfigSchema(),
		SecureDefaultConfig: m.GetConfigItem().GetSecureDefaultConfig(),
		SecureCurrentConfig: m.GetConfigItem().GetSecureCurrentConfig(),
	}

	osMap := make(map[string][]string)
	for _, os := range m.GetConfig().GetOs() {
		osMap[os.GetType()] = os.GetArch()
	}

	var version loader.ModuleVersion
	sver, err := semver.NewVersion(m.GetConfig().GetVersion())
	if err == nil {
		version.Major = sver.Major()
		version.Minor = sver.Minor()
		version.Patch = sver.Patch()
	}
	mc := &loader.ModuleConfig{
		OS:                osMap,
		GroupID:           m.GetConfig().GetGroupId(),
		PolicyID:          m.GetConfig().GetPolicyId(),
		Name:              m.GetConfig().GetName(),
		Version:           version,
		Actions:           m.GetConfig().GetActions(),
		Events:            m.GetConfig().GetEvents(),
		Fields:            m.GetConfig().GetFields(),
		State:             m.GetConfig().GetState(),
		Template:          m.GetConfig().GetTemplate(),
		LastModuleUpdate:  m.GetConfig().GetLastModuleUpdate(),
		LastUpdate:        m.GetConfig().GetLastUpdate(),
		IConfigItem:       mci,
		IConfigItemUpdate: mci,
	}

	return mc
}

func (mm *MainModule) getModuleItem(m *protoagent.Module) *loader.ModuleItem {
	mi := loader.NewItem()

	mf := make(map[string][]byte)
	for _, f := range m.GetFiles() {
		mf[f.GetPath()] = f.GetData()
	}
	mi.SetFiles(mf)

	ma := make(map[string][]string)
	for _, a := range m.GetArgs() {
		ma[a.GetKey()] = a.GetValue()
	}
	mi.SetArgs(ma)

	return mi
}

const (
	failedToAddModuleToLoaderMsg      = "failed to add module %s to loader"
	failedToDeleteModuleFromLoaderMsg = "failed to delete module %s from loader"
	failedToUnmarshalModulesInfoMsg   = "failed to unmarshal modules information: %w"
	failedToUpdateModuleConfigMsg     = "failed to update module config in internal structure: %w"
	failedToUpdateModuleCtxMsg        = "failed to update module ctx in the lua state global variable: %w"
	moduleNotFoundMsg                 = "module %s not found"
	moduleSocketNotInitializedMsg     = "module socket is not initialized"
)

// responseAgent is function which send response to server
func (mm *MainModule) responseAgent(ctx context.Context, dst string, msgType protoagent.Message_Type, payload []byte) error {
	mm.mutexResp.Lock()
	defer mm.mutexResp.Unlock()

	if mm.msocket == nil {
		return fmt.Errorf(moduleSocketNotInitializedMsg)
	}

	messageData, err := protoagent.PackMessage(msgType, payload)
	if err != nil {
		return err
	}

	data := &vxproto.Data{
		Data: messageData,
	}
	if err = mm.msocket.SendDataTo(ctx, dst, data); err != nil {
		return err
	}

	return nil
}

func (mm *MainModule) sendInformation(ctx context.Context, dst string) error {
	agentInfo, err := system.GetAgentInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get the agent info: %w", err)
	}
	infoMessageData, err := proto.Marshal(agentInfo)
	if err != nil {
		return err
	}

	return mm.responseAgent(ctx, dst, protoagent.Message_INFORMATION_RESULT, infoMessageData)
}

func (mm *MainModule) sendStatusModules(ctx context.Context, dst string) error {
	modules := mm.getStatusModules()
	statusModulesData, err := proto.Marshal(modules)
	if err != nil {
		return fmt.Errorf("failed to marshal modules status list: %w", err)
	}

	return mm.responseAgent(ctx, dst, protoagent.Message_STATUS_MODULES_RESULT, statusModulesData)
}

func (mm *MainModule) sendAction(ctx context.Context, name string, data proto.Message) error {
	mm.mutexResp.Lock()
	defer mm.mutexResp.Unlock()

	if mm.msocket == nil {
		return fmt.Errorf(moduleSocketNotInitializedMsg)
	}

	var dst string
	for _, agent := range mm.proto.GetAgentList() {
		if agent.GetAgentID() == mm.agentID {
			dst = agent.GetDestination()
		}
	}
	if dst == "" {
		return fmt.Errorf("agent isn't connected")
	}

	actionData, err := proto.Marshal(data)
	if err != nil {
		return fmt.Errorf("error marshal action data: %w", err)
	}

	actionPacket := &vxproto.Action{
		Name: name,
		Data: actionData,
	}
	if err := mm.msocket.SendActionTo(ctx, dst, actionPacket); err != nil {
		return err
	}

	return nil
}

func (mm *MainModule) serveStartModules(ctx context.Context, dst string, data []byte) (err error) {
	defer func() {
		if errSend := mm.sendStatusModules(ctx, dst); errSend != nil {
			if err == nil {
				err = errSend
			} else {
				err = getFailedToSendErrorErr(err, errSend)
			}
		}
	}()

	var moduleList protoagent.ModuleList
	if err = proto.Unmarshal(data, &moduleList); err != nil {
		err = fmt.Errorf(failedToUnmarshalModulesInfoMsg, err)
		return
	}

	for _, m := range moduleList.GetList() {
		var s *loader.ModuleState
		id := m.GetName()
		mc := mm.getModuleConfig(m)
		mi := mm.getModuleItem(m)
		s, err = loader.NewState(mc, mi, mm.proto)
		if err != nil {
			return
		}

		if mm.loader.Get(id) != nil {
			err = fmt.Errorf("module %s already exists", id)
			return
		}

		if !mm.loader.Add(id, s) {
			err = fmt.Errorf(failedToAddModuleToLoaderMsg, id)
			return
		}

		if err = mm.RegisterLuaAPI(s.GetState(), mc); err != nil {
			err = getFailedToRegisterExtraAPIErr(id, err)
			return
		}

		if err = mm.loader.Start(id); err != nil {
			return
		}

		mm.modules[id] = mc

		// Flush files list because there will not restart module from previous state
		mi.SetFiles(make(map[string][]byte))
	}

	// it's need because module can is a large size there need forcing release used
	go mm.forceFreeMemory()

	return
}

func (mm *MainModule) serveStopModules(ctx context.Context, dst string, data []byte) (err error) {
	defer func() {
		if errSend := mm.sendStatusModules(ctx, dst); errSend != nil {
			if err == nil {
				err = errSend
			} else {
				err = getFailedToSendErrorErr(err, errSend)
			}
		}
	}()

	var moduleList protoagent.ModuleList
	if err = proto.Unmarshal(data, &moduleList); err != nil {
		err = fmt.Errorf(failedToUnmarshalModulesInfoMsg, err)
		return
	}

	for _, m := range moduleList.GetList() {
		var s *loader.ModuleState
		id := m.GetName()
		mc := mm.getModuleConfig(m)
		if s = mm.loader.Get(id); s == nil {
			err = fmt.Errorf(moduleNotFoundMsg, id)
			return
		}

		if err = mm.UnregisterLuaAPI(s.GetState(), mc); err != nil {
			err = getFailedToUnregisterExtraAPIErr(id, err)
			return
		}

		if err = mm.loader.Stop(id, "module_remove"); err != nil {
			return
		}

		if !mm.loader.Del(id, "module_remove") {
			err = fmt.Errorf(failedToDeleteModuleFromLoaderMsg, id)
			return
		}

		delete(mm.modules, id)
	}

	return
}

func (mm *MainModule) serveUpdateModules(ctx context.Context, dst string, data []byte) (err error) {
	defer func() {
		if errSend := mm.sendStatusModules(ctx, dst); errSend != nil {
			if err == nil {
				err = errSend
			} else {
				err = getFailedToSendErrorErr(err, errSend)
			}
		}
	}()

	var moduleList protoagent.ModuleList
	if err = proto.Unmarshal(data, &moduleList); err != nil {
		err = fmt.Errorf(failedToUnmarshalModulesInfoMsg, err)
		return
	}

	for _, m := range moduleList.GetList() {
		var s *loader.ModuleState
		id := m.GetName()
		mc := mm.getModuleConfig(m)
		mi := mm.getModuleItem(m)
		if s = mm.loader.Get(id); s == nil {
			err = fmt.Errorf(moduleNotFoundMsg, id)
			return
		}

		if err = mm.UnregisterLuaAPI(s.GetState(), mc); err != nil {
			err = getFailedToUnregisterExtraAPIErr(id, err)
			return
		}

		if err = mm.loader.Stop(id, "module_update"); err != nil {
			return
		}

		if !mm.loader.Del(id, "module_update") {
			err = fmt.Errorf(failedToDeleteModuleFromLoaderMsg, id)
			return
		}

		s, err = loader.NewState(mc, mi, mm.proto)
		if err != nil {
			return
		}

		if !mm.loader.Add(id, s) {
			err = fmt.Errorf(failedToAddModuleToLoaderMsg, id)
			return
		}

		if err = mm.RegisterLuaAPI(s.GetState(), mc); err != nil {
			err = getFailedToRegisterExtraAPIErr(id, err)
			return
		}

		if err = mm.loader.Start(id); err != nil {
			return
		}

		mm.modules[id] = mc

		// Flush files list because there will not restart module from previous state
		mi.SetFiles(make(map[string][]byte))
	}

	// it's need because module can is a large size there need forcing release used
	go mm.forceFreeMemory()

	return
}

func (mm *MainModule) serveUpdateConfigModules(ctx context.Context, dst string, data []byte) (err error) {
	defer func() {
		if errSend := mm.sendStatusModules(ctx, dst); errSend != nil {
			if err == nil {
				err = errSend
			} else {
				err = getFailedToSendErrorErr(err, errSend)
			}
		}
	}()

	var moduleList protoagent.ModuleList
	if err = proto.Unmarshal(data, &moduleList); err != nil {
		err = fmt.Errorf(failedToUnmarshalModulesInfoMsg, err)
		return
	}

	for _, m := range moduleList.GetList() {
		id := m.GetName()
		mc := mm.getModuleConfig(m)
		ms := mm.loader.Get(id)
		if ms == nil {
			err = fmt.Errorf(moduleNotFoundMsg, id)
			return
		}

		if romc, ok := mm.modules[id]; ok {
			if err = romc.Update(mc); err != nil {
				err = fmt.Errorf(failedToUpdateModuleConfigMsg, err)
				return
			}

			if err = ms.UpdateCtx(mc); err != nil {
				err = fmt.Errorf(failedToUpdateModuleCtxMsg, err)
				return
			}

			ms.GetModule().ControlMsg(ctx, "update_config", mc.LastUpdate)
		}
	}

	return
}

func (mm *MainModule) serveUpdateExecPushMsg(ctx context.Context, src string, payload []byte) error {
	var msg protoagent.AgentUpgradeExecPush
	if err := proto.Unmarshal(payload, &msg); err != nil {
		return fmt.Errorf("failed to unmarshal the update exec push message: %w", err)
	}
	mm.upgraderWG.Add(1)
	go func(msg *protoagent.AgentUpgradeExecPush) {
		defer mm.upgraderWG.Done()
		if err := mm.upgrader.startUpgrade(ctx, src, msg); err != nil {
			logrus.WithContext(ctx).Error(fmt.Errorf("failed to start the agent update: %w", err))
		}
	}(&msg)
	return nil
}

func (mm *MainModule) serveData(ctx context.Context, src string, data *vxproto.Data) error {
	var message protoagent.Message
	if err := proto.Unmarshal(data.Data, &message); err != nil {
		return err
	}

	switch message.GetType() {
	case protoagent.Message_GET_INFORMATION:
		return mm.sendInformation(ctx, src)
	case protoagent.Message_GET_STATUS_MODULES:
		return mm.sendStatusModules(ctx, src)
	case protoagent.Message_START_MODULES:
		return mm.serveStartModules(ctx, src, message.Payload)
	case protoagent.Message_STOP_MODULES:
		return mm.serveStopModules(ctx, src, message.Payload)
	case protoagent.Message_UPDATE_MODULES:
		return mm.serveUpdateModules(ctx, src, message.Payload)
	case protoagent.Message_UPDATE_CONFIG_MODULES:
		return mm.serveUpdateConfigModules(ctx, src, message.Payload)
	case protoagent.Message_AGENT_UPGRADE_EXEC_PUSH:
		return mm.serveUpdateExecPushMsg(ctx, src, message.Payload)
	default:
		return fmt.Errorf("received unknown message type")
	}
}

func (mm *MainModule) recvPacket() error {
	const componentPacketReceiver = "main_packet_receiver"

	defer mm.wgReceiver.Done()
	logger := logrus.WithField("component", componentPacketReceiver)

	receiver := mm.msocket.GetReceiver()
	if receiver == nil {
		logger.Error("vxagent: failed to initialize packet receiver")
		return fmt.Errorf("failed to initialize packet receiver")
	}
	getAgentEntry := func(ctx context.Context, agentInfo *vxproto.AgentInfo) *logrus.Entry {
		return logrus.WithContext(ctx).WithFields(logrus.Fields{
			"id":   agentInfo.ID,
			"type": agentInfo.Type.String(),
			"ip":   agentInfo.IP,
			"src":  agentInfo.Src,
			"dst":  agentInfo.Dst,
		})
	}
	for {
		packet := <-receiver
		if packet == nil {
			logger.Error("vxagent: failed receive packet")
			return fmt.Errorf("failed receive packet")
		}
		packetCtx, packetSpan := obs.Observer.NewSpan(packet.Context(), obs.SpanKindConsumer, componentPacketReceiver)

		switch packet.PType {
		case vxproto.PTData:
			_ = mm.recvData(packetCtx, packet.Src, packet.GetData())
		case vxproto.PTFile:
			if err := mm.recvFile(packetCtx, packet.Src, packet.GetFile()); err != nil {
				logrus.WithContext(packetCtx).WithError(err).Error("failed to receive a file")
			}
		case vxproto.PTText:
			if err := mm.recvText(packetCtx, packet.Src, packet.GetText()); err != nil {
				logrus.WithContext(packetCtx).WithError(err).Error("failed to receive text")
			}
		case vxproto.PTMsg:
			if err := mm.recvMsg(packetCtx, packet.Src, packet.GetMsg()); err != nil {
				logrus.WithContext(packetCtx).WithError(err).Error("failed to receive a message")
			}
		case vxproto.PTAction:
			if err := mm.recvAction(packetCtx, packet.Src, packet.GetAction()); err != nil {
				logrus.WithContext(packetCtx).WithError(err).Error("failed to receive an action")
			}
		case vxproto.PTControl:
			msg := packet.GetControlMsg()
			switch msg.MsgType {
			case vxproto.AgentConnected:
				getAgentEntry(packetCtx, msg.AgentInfo).Info("vxagent: agent connected")
			case vxproto.AgentDisconnected:
				getAgentEntry(packetCtx, msg.AgentInfo).Info("vxagent: agent disconnected")
			case vxproto.StopModule:
				logrus.WithContext(packetCtx).Info("vxagent: got signal to stop main module")
				packet.SetAck()
				packetSpan.End()
				return nil
			}
		default:
			logrus.Error("vxagent: got packet has unexpected packet type")
			packet.SetAck()
			packetSpan.End()
			return fmt.Errorf("unexpected packet type")
		}
		// use default acknowledgement callback
		packet.SetAck()
		packetSpan.End()
	}
}

func getFailedToRegisterExtraAPIErr(id string, e error) error {
	return fmt.Errorf("failed to register extra API for %s: %w", id, e)
}

func getFailedToUnregisterExtraAPIErr(id string, e error) error {
	return fmt.Errorf("failed to unregister extra API for %s: %w", id, e)
}

func getFailedToSendErrorErr(e error, errSend error) error {
	return fmt.Errorf("%s | %w", e.Error(), errSend)
}
