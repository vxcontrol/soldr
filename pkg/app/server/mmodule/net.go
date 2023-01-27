package mmodule

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"google.golang.org/protobuf/proto"

	"soldr/pkg/app/agent"
	"soldr/pkg/app/api/models"
	"soldr/pkg/app/server/mmodule/hardening/v1/storecryptor"
	"soldr/pkg/controller"
	"soldr/pkg/loader"
	"soldr/pkg/utils"
	"soldr/pkg/vxproto"
)

// List of default values
const (
	defaultRecvResultTimeout = 60000
)

// requestAgent is function which do request and wait any unstructured result
func (mm *MainModule) requestAgent(
	ctx context.Context, dst string, msgType agent.Message_Type, payload []byte,
) (*agent.Message, error) {
	if mm.msocket == nil {
		return nil, fmt.Errorf("module Socket didn't initialize")
	}
	message, err := mm.sendMsgToAgent(ctx, dst, msgType, payload)
	if err != nil {
		return nil, err
	}
	return message, nil
}

func (mm *MainModule) requestAgentWithDestStruct(
	ctx context.Context,
	dst string,
	reqType agent.Message_Type,
	reqPayload []byte,
	respType agent.Message_Type,
	resp proto.Message,
) error {
	msg, err := mm.requestAgent(ctx, dst, reqType, reqPayload)
	if err != nil {
		return err
	}
	actualRespType := msg.GetType()
	if actualRespType != respType {
		return fmt.Errorf("expected to receive a message of a type %d, got: %d", respType, actualRespType)
	}
	respPayload := msg.GetPayload()
	if respPayload == nil {
		return nil
	}
	if err := proto.Unmarshal(respPayload, resp); err != nil {
		return fmt.Errorf("failed to unmarshal the agent response: %w", err)
	}
	return nil
}

func (mm *MainModule) sendMsgToAgent(
	ctx context.Context,
	dst string,
	msgType agent.Message_Type,
	payload []byte,
) (*agent.Message, error) {
	if mm.msocket == nil {
		return nil, errors.New("module Socket didn't initialize")
	}
	messageData, err := proto.Marshal(&agent.Message{
		Type:    msgType.Enum(),
		Payload: payload,
	})
	if err != nil {
		return nil, fmt.Errorf("error marshal request packet: %w", err)
	}

	data := &vxproto.Data{
		Data: messageData,
	}
	if err = mm.msocket.SendDataTo(ctx, dst, data); err != nil {
		return nil, err
	}

	data, err = mm.msocket.RecvDataFrom(ctx, dst, defaultRecvResultTimeout)
	if err != nil {
		return nil, err
	}

	var message agent.Message
	if err = proto.Unmarshal(data.Data, &message); err != nil {
		return nil, fmt.Errorf("error unmarshal response message packet: %w", err)
	}

	return &message, nil
}

// requestAgentAboutModules is function which do request and wait module status structure
func (mm *MainModule) requestAgentAboutModules(
	ctx context.Context,
	dst string,
	ainfo *agentInfo,
	mIDs []string,
	msgType agent.Message_Type,
) (*agent.ModuleStatusList, error) {
	mm.syncModulesSemaphore <- struct{}{}
	defer func() {
		<-mm.syncModulesSemaphore
	}()

	payload, err := mm.getModuleListData(ctx, dst, ainfo, mIDs, msgType)
	if err != nil {
		return nil, err
	}

	message, err := mm.requestAgent(ctx, dst, msgType, payload)
	if err != nil {
		return nil, err
	}

	var statusMessage agent.ModuleStatusList
	if err = proto.Unmarshal(message.Payload, &statusMessage); err != nil {
		return nil, fmt.Errorf("error unmarshal response packet of status modules: %w", err)
	}

	return &statusMessage, nil
}

// getModuleListData is function for generate ModuleList message
func (mm *MainModule) getModuleListData(
	ctx context.Context,
	dst string,
	ainfo *agentInfo,
	mIDs []string,
	msgType agent.Message_Type,
) ([]byte, error) {
	moduleListMessage, err := mm.getModuleList(ctx, dst, ainfo, mIDs, msgType)
	if err != nil {
		return nil, fmt.Errorf("failed to get module list: %w", err)
	}
	moduleListMessageData, err := proto.Marshal(moduleListMessage)
	if err != nil {
		return nil, fmt.Errorf("error marshal modules list packet: %w", err)
	}

	return moduleListMessageData, nil
}

// moduleToPB is function which convert module data to Protobuf structure
func (mm *MainModule) moduleToPB(
	_ string, // dst
	ainfo *agentInfo,
	mc *loader.ModuleConfig,
	mi *loader.ModuleItem,
	msgType agent.Message_Type,
) (*agent.Module, error) {
	var args []*agent.Module_Arg
	var files []*agent.Module_File
	os := ainfo.info.Info.GetOs().GetType()
	arch := ainfo.info.Info.GetOs().GetArch()

	var osList []*agent.Config_OS
	for osType, archList := range mc.OS {
		osList = append(osList, &agent.Config_OS{
			Type: utils.GetRef(osType),
			Arch: archList,
		})
	}
	config := &agent.Config{
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

	defaultSecConfig, currentSecConfig, err := mm.getModuleSecureConfigForAgent(mc)
	if err != nil {
		return nil, fmt.Errorf("failed to get secure config for agent: %w", err)
	}

	iconfig := &agent.ConfigItem{
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
		SecureDefaultConfig: utils.GetRef(defaultSecConfig),
		SecureCurrentConfig: utils.GetRef(currentSecConfig),
	}

	if err := mm.encryptSecureParamsUsingAgentKey(ainfo, iconfig); err != nil {
		return nil, err
	}

	if msgType == agent.Message_START_MODULES || msgType == agent.Message_UPDATE_MODULES {
		for path, data := range mi.GetFilesByFilter(os, arch) {
			files = append(files, &agent.Module_File{
				Path: utils.GetRef(path),
				Data: data,
			})
		}
	}

	for key, value := range mi.GetArgs() {
		args = append(args, &agent.Module_Arg{
			Key:   utils.GetRef(key),
			Value: value,
		})
	}

	return &agent.Module{
		Name:       utils.GetRef(mc.Name),
		Config:     config,
		ConfigItem: iconfig,
		Args:       args,
		Files:      files,
	}, nil
}

func (mm *MainModule) getModuleSecureConfigForAgent(mc *loader.ModuleConfig) (string, string, error) {
	currentConfig := models.ModuleSecureConfig{}
	err := json.Unmarshal([]byte(mc.GetSecureCurrentConfig()), &currentConfig)
	if err != nil {
		return "", "", err
	}

	// remove server_only params
	for k, v := range currentConfig {
		if *v.ServerOnly {
			delete(currentConfig, k)
		}
	}

	b, err := json.Marshal(currentConfig)
	if err != nil {
		return "", "", err
	}

	currentSecConfig, err := mm.moduleConfigDecryptor.Decrypt(string(b))
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt secure current config: %w", err)
	}

	defaultSecConfig, err := mm.moduleConfigDecryptor.Decrypt(mc.GetSecureDefaultConfig())
	if err != nil {
		return "", "", fmt.Errorf("failed to decrypt secure default config: %w", err)
	}

	return defaultSecConfig, currentSecConfig, nil
}

func (mm *MainModule) encryptSecureParamsUsingAgentKey(a *agentInfo, agentCfg *agent.ConfigItem) error {
	cryptor := storecryptor.NewStoreCryptor(a.info.ID)
	secureDefaultConfig, err := cryptor.EncryptData([]byte(*agentCfg.SecureDefaultConfig))
	if err != nil {
		return fmt.Errorf("failed to encrypt module secure default config: %w", err)
	}
	agentCfg.SecureDefaultConfig = utils.GetRef(string(secureDefaultConfig))

	secureCurrentConfig, err := cryptor.EncryptData([]byte(*agentCfg.SecureCurrentConfig))
	if err != nil {
		return fmt.Errorf("failed to encrypt module secure current config: %w", err)
	}
	agentCfg.SecureCurrentConfig = utils.GetRef(string(secureCurrentConfig))

	return nil
}

// getModuleList is API functions which execute remote logic
func (mm *MainModule) getModuleList(
	_ context.Context,
	dst string,
	ainfo *agentInfo,
	mIDs []string,
	msgType agent.Message_Type,
) (*agent.ModuleList, error) {
	var modules agent.ModuleList
	mObjs := mm.cnt.GetModules(mIDs)

	ids, err := mm.checkMismatchModuleChecksums(mObjs)
	if err != nil {
		return nil, fmt.Errorf("error comparing module files checksums: %w", err)
	}
	if len(ids) > 0 {
		return nil, fmt.Errorf("module checksums in DB not match with data in local cache: %v", ids)
	}

	for _, mID := range mIDs {
		var (
			module *agent.Module
			err    error
		)
		if mObj, ok := mObjs[mID]; ok {
			module, err = mm.moduleToPB(dst, ainfo, mObj.GetConfig(), mObj.GetFiles().GetCModule(), msgType)
			if err != nil {
				return nil, err
			}
		} else {
			module = &agent.Module{
				Name: utils.GetRef(strings.Split(mID, ":")[2]),
			}
		}
		modules.List = append(modules.List, module)
	}

	return &modules, nil
}

func (mm *MainModule) checkMismatchModuleChecksums(inMemModules map[string]*controller.Module) ([]string, error) {
	dbModulesInfo, err := mm.getModulesInfoFromDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get modules info from DB: %w", err)
	}

	dbModulesInfoMap := make(map[string]moduleChecksumsInfo)
	for _, m := range dbModulesInfo {
		dbModulesInfoMap[m.GetID()] = m
	}

	var mismatchModuleIDs []string
	for id, m := range inMemModules {
		dbModule, ok := dbModulesInfoMap[id]
		if !ok {
			continue
		}

		if !m.IsValidByFileChecksums(dbModule.FilesChecksums) {
			mismatchModuleIDs = append(mismatchModuleIDs, id)
		}
	}

	return mismatchModuleIDs, nil
}

type moduleChecksumsInfo struct {
	GroupID        string                   `json:"group_id"`
	PolicyID       string                   `json:"policy_id"`
	Name           string                   `json:"name"`
	FilesChecksums models.FilesChecksumsMap `json:"files_checksums"`
}

func (mi *moduleChecksumsInfo) GetID() string {
	mc := loader.ModuleConfig{
		GroupID:  mi.GroupID,
		PolicyID: mi.PolicyID,
		Name:     mi.Name,
	}
	return mc.ID()
}

const queryGetModuleInfoWithChecksums = `select
       		IFNULL(g.hash, '') AS group_id,
       		IFNULL(p.hash, '') AS policy_id,
       		name,
       		files_checksums
		FROM modules m
		    LEFT JOIN (SELECT id, hash, deleted_at FROM policies) p ON m.policy_id = p.id AND p.deleted_at IS NULL
		    LEFT JOIN groups_to_policies gp ON gp.policy_id = p.id
		    LEFT JOIN (SELECT id, hash, deleted_at FROM groups) g ON gp.group_id = g.id AND g.deleted_at IS NULL
		WHERE m.status = 'joined' AND NOT (ISNULL(g.hash) AND p.hash NOT LIKE '') AND m.deleted_at IS NULL;`

func (mm *MainModule) getModulesInfoFromDB() ([]moduleChecksumsInfo, error) {
	var scanResult []moduleChecksumsInfo
	err := mm.gdbc.Raw(queryGetModuleInfoWithChecksums).Scan(&scanResult).Error
	if err != nil {
		return nil, err
	}

	return scanResult, nil
}

// getInformation is API functions which execute remote logic
//
//lint:ignore U1000 it'll be use in the future
func (mm *MainModule) getInformation(ctx context.Context, dst string) (*agent.Information, error) {
	message, err := mm.requestAgent(ctx, dst, agent.Message_GET_INFORMATION, []byte{})
	if err != nil {
		return nil, err
	}

	var infoMessage agent.Information
	if err = proto.Unmarshal(message.Payload, &infoMessage); err != nil {
		return nil, fmt.Errorf("error unmarshal response packet of information: %w", err)
	}

	return &infoMessage, nil
}

// getStatusModules is API functions which execute remote logic
func (mm *MainModule) getStatusModules(
	ctx context.Context,
	dst string,
	_ *agentInfo,
) (*agent.ModuleStatusList, error) {
	message, err := mm.requestAgent(ctx, dst, agent.Message_GET_STATUS_MODULES, []byte{})
	if err != nil {
		return nil, err
	}

	var statusMessage agent.ModuleStatusList
	if err = proto.Unmarshal(message.Payload, &statusMessage); err != nil {
		return nil, fmt.Errorf("error unmarshal response packet of status modules: %w", err)
	}

	return &statusMessage, nil
}

// startModules is API functions which execute remote logic
func (mm *MainModule) startModules(
	ctx context.Context, dst string, ainfo *agentInfo, mIDs []string,
) (*agent.ModuleStatusList, error) {
	return mm.requestAgentAboutModules(ctx, dst, ainfo, mIDs, agent.Message_START_MODULES)
}

// stopModules is API functions which execute remote logic
func (mm *MainModule) stopModules(
	ctx context.Context, dst string, ainfo *agentInfo, mIDs []string,
) (*agent.ModuleStatusList, error) {
	return mm.requestAgentAboutModules(ctx, dst, ainfo, mIDs, agent.Message_STOP_MODULES)
}

// updateModules is API functions which execute remote logic
func (mm *MainModule) updateModules(
	ctx context.Context, dst string, ainfo *agentInfo, mIDs []string,
) (*agent.ModuleStatusList, error) {
	return mm.requestAgentAboutModules(ctx, dst, ainfo, mIDs, agent.Message_UPDATE_MODULES)
}

// updateModulesConfig is API functions which execute remote logic
func (mm *MainModule) updateModulesConfig(
	ctx context.Context, dst string, ainfo *agentInfo, mIDs []string,
) (*agent.ModuleStatusList, error) {
	return mm.requestAgentAboutModules(ctx, dst, ainfo, mIDs, agent.Message_UPDATE_CONFIG_MODULES)
}
