package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/vxcontrol/luar"

	"soldr/pkg/lua"
	"soldr/pkg/protoagent"
	"soldr/pkg/vxproto"
)

type deferredModuleCallback func() bool

// ModuleState is struct for contains information about module
type ModuleState struct {
	name      string
	item      *ModuleItem
	status    protoagent.ModuleStatus_Status
	wg        sync.WaitGroup
	luaModule *lua.Module
	luaState  *lua.State
	cbStart   deferredModuleCallback
	cbStop    deferredModuleCallback
}

// NewState is function which constructed ModuleState object
func NewState(mc *ModuleConfig, mi *ModuleItem, p vxproto.IVXProto) (*ModuleState, error) {
	ms := &ModuleState{
		name:   mc.Name,
		item:   mi,
		status: protoagent.ModuleStatus_UNKNOWN,
	}

	socket := p.NewModule(mc.Name, mc.GroupID)
	if socket == nil {
		return nil, fmt.Errorf("socket for the module '%s' is not initialized", mc.Name)
	}
	ms.cbStart = func() bool {
		agents := make(map[string]*vxproto.AgentInfo)
		for dst, agent := range p.GetAgentList() {
			mgid := mc.GroupID
			agid := agent.GetGroupID()
			if mgid == agid || mgid == "" || agid == "" {
				agents[dst] = agent.GetPublicInfo()
			}
		}
		ms.luaModule.SetAgents(agents)

		if !p.AddModule(socket) {
			socket.Close(context.TODO())
			return false
		}
		return true
	}
	ms.cbStop = func() bool {
		ms.luaModule.SetAgents(make(map[string]*vxproto.AgentInfo))
		return p.DelModule(socket)
	}

	var err error
	if ms.luaState, err = lua.NewState(mi.files); err != nil {
		return nil, fmt.Errorf("failed to create a new lua state: %w", err)
	}

	if ms.luaModule, err = lua.NewModule(mi.args, ms.luaState, socket); err != nil {
		return nil, fmt.Errorf("failed to create a new module: %w", err)
	}

	// TODO: here need to register API for communication to logging subsystem

	luar.Register(ms.luaState.L, "__config", luar.Map{
		"get_config_schema":         mc.GetConfigSchema,
		"get_default_config":        mc.GetDefaultConfig,
		"get_current_config":        mc.GetCurrentConfig,
		"get_static_dependencies":   mc.GetStaticDependencies,
		"get_dynamic_dependencies":  mc.GetDynamicDependencies,
		"get_fields_schema":         mc.GetFieldsSchema,
		"get_action_config_schema":  mc.GetActionConfigSchema,
		"get_default_action_config": mc.GetDefaultActionConfig,
		"get_current_action_config": mc.GetCurrentActionConfig,
		"get_event_config_schema":   mc.GetEventConfigSchema,
		"get_default_event_config":  mc.GetDefaultEventConfig,
		"get_current_event_config":  mc.GetCurrentEventConfig,
		"set_current_config":        mc.SetCurrentConfig,
		"set_current_action_config": mc.SetCurrentActionConfig,
		"set_current_event_config":  mc.SetCurrentEventConfig,
		"set_dynamic_dependencies":  mc.SetDynamicDependencies,
		"get_secure_config_schema":  mc.GetSecureConfigSchema,
		"get_secure_default_config": mc.GetSecureDefaultConfig,
		"get_secure_current_config": mc.GetSecureCurrentConfig,
		"get_module_info": func() string {
			if dinfo, err := json.Marshal(mc); err == nil {
				return string(dinfo)
			}
			return ""
		},
		"ctx": luar.Map{
			"group_id":           mc.GroupID,
			"policy_id":          mc.PolicyID,
			"os":                 mc.OS,
			"name":               mc.Name,
			"version":            mc.Version.String(),
			"actions":            mc.Actions,
			"events":             mc.Events,
			"fields":             mc.Fields,
			"state":              mc.State,
			"template":           mc.Template,
			"last_module_update": mc.LastModuleUpdate,
			"last_update":        mc.LastUpdate,
		},
	})

	ms.status = protoagent.ModuleStatus_LOADED
	return ms, nil
}

// Start is function for running server module
func (ms *ModuleState) Start() error {
	switch ms.status {
	case protoagent.ModuleStatus_LOADED:
		fallthrough
	case protoagent.ModuleStatus_STOPPED:
		if !ms.cbStart() {
			return fmt.Errorf("failed to start the module '%s'", ms.name)
		}
		ms.wg.Add(1)
		ms.status = protoagent.ModuleStatus_RUNNING
		go func(ms *ModuleState) {
			defer ms.wg.Done()
			ms.luaModule.Start()
			ms.status = protoagent.ModuleStatus_STOPPED
		}(ms)
	case protoagent.ModuleStatus_UNKNOWN, protoagent.ModuleStatus_RUNNING, protoagent.ModuleStatus_FREED:
		fallthrough
	default:
		return genModuleError(ms.status, ms.name)
	}

	return nil
}

// Stop stops server module
func (ms *ModuleState) Stop(stopReason string) error {
	switch ms.status {
	case protoagent.ModuleStatus_LOADED:
		ms.status = protoagent.ModuleStatus_STOPPED
	case protoagent.ModuleStatus_RUNNING:
		ms.luaModule.Stop(stopReason)
		if !ms.cbStop() {
			return genFailedToStopModuleErr(ms.name)
		}
		ms.status = protoagent.ModuleStatus_STOPPED
	case protoagent.ModuleStatus_UNKNOWN, protoagent.ModuleStatus_STOPPED, protoagent.ModuleStatus_FREED:
		fallthrough
	default:
		return genModuleError(ms.status, ms.name)
	}

	return nil
}

// Close releases module object, stop the module beforehand if it is in a RUNNING state
func (ms *ModuleState) Close(stopReason string) error {
	switch ms.status {
	case protoagent.ModuleStatus_RUNNING:
		ms.luaModule.Stop(stopReason)
		if !ms.cbStop() {
			return genFailedToStopModuleErr(ms.name)
		}
		ms.status = protoagent.ModuleStatus_STOPPED
		fallthrough
	case protoagent.ModuleStatus_STOPPED:
		ms.wg.Wait()
		luar.Register(ms.luaState.L, "__config", luar.Map{})
		ms.luaModule.Close(stopReason)
		ms.luaState = nil
		ms.status = protoagent.ModuleStatus_FREED
	case protoagent.ModuleStatus_UNKNOWN, protoagent.ModuleStatus_LOADED, protoagent.ModuleStatus_FREED:
		fallthrough
	default:
		return genModuleError(ms.status, ms.name)
	}

	return nil
}

func genFailedToStopModuleErr(moduleName string) error {
	return fmt.Errorf("failed to stop the module '%s'", moduleName)
}

func genModuleError(status protoagent.ModuleStatus_Status, moduleName string) error {
	switch status {
	case protoagent.ModuleStatus_UNKNOWN:
		return fmt.Errorf("failed to load the module '%s'", moduleName)
	case protoagent.ModuleStatus_RUNNING:
		return fmt.Errorf("module '%s' is already running", moduleName)
	case protoagent.ModuleStatus_FREED:
		return fmt.Errorf("module '%s' is already closed", moduleName)
	default:
		return fmt.Errorf("the module '%s' has an unknown status %d", moduleName, status)
	}
}

// GetName is function that return module name
func (ms *ModuleState) GetName() string {
	return ms.name
}

// GetStatus is function that return module status
func (ms *ModuleState) GetStatus() protoagent.ModuleStatus_Status {
	return ms.status
}

// GetResult is function that return module re3sult after close
func (ms *ModuleState) GetResult() string {
	return ms.luaModule.GetResult()
}

// GetState is function that return current lua state
func (ms *ModuleState) GetState() *lua.State {
	return ms.luaState
}

// GetModule is function that return current lua module object
func (ms *ModuleState) GetModule() *lua.Module {
	return ms.luaModule
}

// GetItem is function that return module item (args, files and config)
func (ms *ModuleState) GetItem() *ModuleItem {
	return ms.item
}

func (ms *ModuleState) UpdateCtx(mc *ModuleConfig) error {
	if ms.luaState == nil || ms.luaState.L == nil {
		return fmt.Errorf("module's lua state does not exist")
	}

	luar.Register(ms.luaState.L, "__config", luar.Map{
		"ctx": luar.Map{
			"group_id":           mc.GroupID,
			"policy_id":          mc.PolicyID,
			"os":                 mc.OS,
			"name":               mc.Name,
			"version":            mc.Version.String(),
			"actions":            mc.Actions,
			"events":             mc.Events,
			"fields":             mc.Fields,
			"state":              mc.State,
			"template":           mc.Template,
			"last_module_update": mc.LastModuleUpdate,
			"last_update":        mc.LastUpdate,
		},
	})

	return nil
}
