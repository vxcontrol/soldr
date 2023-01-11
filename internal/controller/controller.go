package controller

import (
	"context"
	"fmt"
	"sync"
	"time"

	"soldr/internal/loader"
	"soldr/internal/lua"
	"soldr/internal/vxproto"
)

// IController is interface for loading and control all modules
type IController interface {
	Load() error
	Close() error
	Lock()
	Unlock()
	GetModuleState(id string) *loader.ModuleState
	GetModuleStates(ids []string) map[string]*loader.ModuleState
	GetModule(id string) *Module
	GetModules(ids []string) map[string]*Module
	GetModuleIds() []string
	GetSharedModuleIds() []string
	GetModuleIdsForGroup(groupID string) []string
	StartAllModules() ([]string, error)
	StartSharedModules() ([]string, error)
	StartModulesForGroup(groupID string) ([]string, error)
	StopSharedModules(shutdown bool) ([]string, error)
	StopModulesForGroup(groupID string, shutdown bool) ([]string, error)
	StopAllModules(shutdown bool) ([]string, error)
	SetUpdateChan(update chan struct{})
	SetUpdateChanForGroup(groupID string, update chan struct{})
	UnsetUpdateChan(update chan struct{})
	UnsetUpdateChanForGroup(groupID string, update chan struct{})
}

// IRegAPI is interface of Main Module for registrate extra Lua API
type IRegAPI interface {
	RegisterLuaAPI(L *lua.State, config *loader.ModuleConfig) error
	UnregisterLuaAPI(L *lua.State, config *loader.ModuleConfig) error
}

// syncModulesInterval is a time period to retrieve modules changes
const syncModulesInterval = 10 * time.Second

// sController is universal container for modules
type sController struct {
	regAPI  IRegAPI
	config  IConfigLoader
	files   IFilesLoader
	loader  loader.ILoader
	modules []*Module
	mutex   *sync.Mutex
	proto   vxproto.IVXProto
	wg      sync.WaitGroup
	quit    chan struct{}
	updates map[string][]chan struct{}
	closed  bool
}

// Module is struct for contains module struct from store
type Module struct {
	id     string
	config *loader.ModuleConfig
	files  *loader.ModuleFiles
}

// GetID is function that return module id
func (m *Module) GetID() string {
	return m.id
}

// GetConfig is function that return module config object
func (m *Module) GetConfig() *loader.ModuleConfig {
	return m.config
}

// GetFiles is function that return module files object
func (m *Module) GetFiles() *loader.ModuleFiles {
	return m.files
}

// NewController is function which constructed Controller object
func NewController(r IRegAPI, c IConfigLoader, f IFilesLoader, p vxproto.IVXProto) IController {
	return &sController{
		regAPI:  r,
		config:  c,
		files:   f,
		proto:   p,
		mutex:   &sync.Mutex{},
		loader:  loader.New(),
		quit:    make(chan struct{}),
		updates: make(map[string][]chan struct{}),
	}
}

// Lock is unsafe function that locked controller state
func (s *sController) Lock() {
	s.mutex.Lock()
}

// Unlock is unsafe function that unlocked controller state
func (s *sController) Unlock() {
	s.mutex.Unlock()
}

// Load is function that retrieve modules list
func (s *sController) Load() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.config == nil {
		return fmt.Errorf("config loader object not found")
	}
	if s.files == nil {
		return fmt.Errorf("files loader object not found")
	}

	config, err := s.config.load()
	if err != nil {
		return fmt.Errorf("failed to load the module configuration: %w", err)
	}

	files, err := s.files.load(config)
	if err != nil {
		return fmt.Errorf("failed to load module files: %w", err)
	}
	if len(config) != len(files) {
		return fmt.Errorf("expected to load %d module files, actually loaded %d", len(config), len(files))
	}

	s.modules = make([]*Module, len(config), (cap(config)+1)*2)
	for idx, mc := range config {
		s.modules[idx] = s.newModule(mc, files[idx])
	}

	s.wg.Add(1)
	go s.updaterModules()

	return nil
}

// Close is function that stop update watcher and all modules
func (s *sController) Close() error {
	if !s.closed {
		s.quit <- struct{}{}
		s.wg.Wait()
	}

	_, err := s.StopAllModules(true)

	return err
}

// GetModuleState is function that return a module state by id
func (s *sController) GetModuleState(id string) *loader.ModuleState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	return s.loader.Get(id)
}

// GetModuleStates is function that return module states by id list
func (s *sController) GetModuleStates(ids []string) map[string]*loader.ModuleState {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	states := make(map[string]*loader.ModuleState)
	for _, id := range ids {
		states[id] = s.loader.Get(id)
	}

	return states
}

// GetModule is function that return a module object by id
func (s *sController) GetModule(id string) *Module {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for idx := range s.modules {
		module := s.modules[idx]
		if module.id == id {
			return module
		}
	}

	return nil
}

// GetModules is function that return module objects by id list
func (s *sController) GetModules(ids []string) map[string]*Module {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	modules := make(map[string]*Module)
	for idx := range s.modules {
		module := s.modules[idx]
		for _, id := range ids {
			if module.id == id {
				modules[id] = module
				break
			}
		}
	}

	return modules
}

// GetModuleIds is function that return module id list
func (s *sController) GetModuleIds() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var ids []string
	for idx := range s.modules {
		ids = append(ids, s.modules[idx].id)
	}

	return ids
}

// GetSharedModuleIds is function that return shared module id list
func (s *sController) GetSharedModuleIds() []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var ids []string
	for idx := range s.modules {
		module := s.modules[idx]
		if module.config.GroupID == "" {
			ids = append(ids, module.id)
		}
	}

	return ids
}

// GetModuleIdsForGroup is function that return module id list for group id
func (s *sController) GetModuleIdsForGroup(groupID string) []string {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var ids []string
	for idx := range s.modules {
		module := s.modules[idx]
		if module.config.GroupID == groupID {
			ids = append(ids, module.id)
		}
	}

	return ids
}

// StartAllModules is function for start all modules
func (s *sController) StartAllModules() ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer s.notifyModules("")

	var ids []string
	for idx := range s.modules {
		module := s.modules[idx]
		if err := s.startModule(module); err != nil {
			return nil, err
		}
		ids = append(ids, module.id)
	}

	return ids, nil
}

// StartSharedModules is function for start shared modules
func (s *sController) StartSharedModules() ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer s.notifyModules("")

	var ids []string
	for idx := range s.modules {
		module := s.modules[idx]
		if module.config.GroupID == "" {
			if err := s.startModule(module); err != nil {
				return nil, err
			}
		}
		ids = append(ids, module.id)
	}

	return ids, nil
}

// StartModulesForGroup is function for start of an group modules
func (s *sController) StartModulesForGroup(groupID string) ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer s.notifyModules(groupID)

	var ids []string
	for idx := range s.modules {
		module := s.modules[idx]
		if module.config.GroupID == groupID {
			if err := s.startModule(module); err != nil {
				return nil, err
			}
		}
		ids = append(ids, module.id)
	}

	return ids, nil
}

// StopSharedModules is function for stop shared modules
// shutdown argument means the controller is staying in closing state now
//
//	and consumers will not receive any updates about stopping modules
func (s *sController) StopSharedModules(shutdown bool) ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer func() {
		if !shutdown {
			s.notifyModules("")
		}
	}()

	var ids []string
	for idx := range s.modules {
		module := s.modules[idx]
		if module.config.GroupID == "" {
			if err := s.stopModule(module, "agent_stop"); err != nil {
				return nil, err
			}
		}
		ids = append(ids, module.id)
	}

	return ids, nil
}

// StopModulesForGroup is function for stop of an group modules
// shutdown argument means the controller is staying in closing state now
//
//	and consumers will not receive any updates about stopping modules
func (s *sController) StopModulesForGroup(groupID string, shutdown bool) ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer func() {
		if !shutdown {
			s.notifyModules(groupID)
		}
	}()

	var ids []string
	for mdx := range s.modules {
		module := s.modules[mdx]
		if module.config.GroupID == groupID {
			if err := s.stopModule(module, "agent_stop"); err != nil {
				return nil, err
			}
		}
		ids = append(ids, module.id)
	}

	return ids, nil
}

// StopAllModules is function for stop all modules
// shutdown argument means the controller is staying in closing state now
//
//	and consumers will not receive any updates about stopping modules
func (s *sController) StopAllModules(shutdown bool) ([]string, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	defer func() {
		if !shutdown {
			s.notifyModules("")
		}
	}()

	var ids []string
	for idx := range s.modules {
		module := s.modules[idx]
		if err := s.stopModule(module, "agent_stop"); err != nil {
			return nil, err
		}
		ids = append(ids, module.id)
	}

	return ids, nil
}

func (s *sController) SetUpdateChan(update chan struct{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.updates[""] = append(s.updates[""], update)
}

func (s *sController) SetUpdateChanForGroup(groupID string, update chan struct{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	s.updates[groupID] = append(s.updates[groupID], update)
}

func (s *sController) UnsetUpdateChan(update chan struct{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for idx, u := range s.updates[""] {
		if u == update {
			// It's possible because there used return in the bottom
			s.updates[""] = append(s.updates[""][:idx], s.updates[""][idx+1:]...)
			break
		}
	}

	if len(s.updates[""]) == 0 {
		delete(s.updates, "")
	}
}

func (s *sController) UnsetUpdateChanForGroup(groupID string, update chan struct{}) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for idx, u := range s.updates[groupID] {
		if u == update {
			// It's possible because there used return in the bottom
			s.updates[groupID] = append(s.updates[groupID][:idx], s.updates[groupID][idx+1:]...)
			break
		}
	}

	if len(s.updates[groupID]) == 0 {
		delete(s.updates, groupID)
	}
}

func (s *sController) newModule(mc *loader.ModuleConfig, mf *loader.ModuleFiles) *Module {
	// This proxy methods need to protect data in the future
	if ci, ok := mc.IConfigItem.(*sConfigItem); ok {
		ci.setCurrentConfig = func(c string) bool { return ci.setCurrentConfig(c) }
		ci.setCurrentActionConfig = func(c string) bool { return ci.setCurrentActionConfig(c) }
		ci.setCurrentEventConfig = func(c string) bool { return ci.setCurrentEventConfig(c) }
		ci.setDynamicDependencies = func(c string) bool { return ci.setDynamicDependencies(c) }
		ci.setSecureCurrentConfig = func(c string) bool { return ci.setSecureCurrentConfig(c) }
	}

	return &Module{
		id:     mc.GroupID + ":" + mc.PolicyID + ":" + mc.Name,
		config: mc,
		files:  mf,
	}
}

// startModule is internal function for start the module
func (s *sController) startModule(module *Module) error {
	name := module.config.Name
	if s.loader.Get(module.id) == nil {
		state, err := loader.NewState(module.config, module.files.GetSModule(), s.proto)
		if err != nil {
			return fmt.Errorf("failed to initialize '%s' Module State: %w", name, err)
		}

		if !s.loader.Add(module.id, state) {
			return fmt.Errorf("failed to add '%s' Module State to loader", name)
		}

		if err = s.regAPI.RegisterLuaAPI(state.GetState(), module.config); err != nil {
			return fmt.Errorf("failed to register extra API for '%s': %w", name, err)
		}
	}

	return s.loader.Start(module.id)
}

// stopModule stops the module
func (s *sController) stopModule(module *Module, stopReason string) error {
	var state *loader.ModuleState
	name := module.config.Name
	if state = s.loader.Get(module.id); state == nil {
		return nil
	}

	if err := s.loader.Stop(module.id, stopReason); err != nil {
		return fmt.Errorf("failed to stop '%s' Module State: %w", name, err)
	}

	if err := s.regAPI.UnregisterLuaAPI(state.GetState(), module.config); err != nil {
		return fmt.Errorf("failed to unregister extra API for '%s': %w", name, err)
	}

	if !s.loader.Del(module.id, stopReason) {
		return fmt.Errorf("failed to delete '%s' Module State from loader", name)
	}

	return nil
}

// updateModule performs module state update by doing start/stop for a module
func (s *sController) updateModule(module *Module, stopReason string) error {
	if err := s.stopModule(module, stopReason); err != nil {
		return err
	}

	if err := s.startModule(module); err != nil {
		return err
	}

	return nil
}

// updateModuleConfig is internal function for update the module config
func (s *sController) updateModuleConfig(module *Module, mc *loader.ModuleConfig) error {
	var state *loader.ModuleState
	if state = s.loader.Get(module.id); state == nil {
		return nil
	}

	if err := module.config.Update(mc); err != nil {
		return err
	}

	if err := state.UpdateCtx(mc); err != nil {
		return err
	}

	if luaState := state.GetModule(); luaState != nil {
		luaState.ControlMsg(context.Background(), "update_config", mc.LastUpdate)
	}

	return nil
}

func (s *sController) notifyModules(GroupID string) {
	for gID, updateList := range s.updates {
		if GroupID == gID || GroupID == "" || gID == "" {
			for _, update := range updateList {
				update <- struct{}{}
			}
		}
	}
}

func (s *sController) checkStartModules(config []*loader.ModuleConfig) {
	var wantStartModules []*loader.ModuleConfig

	for _, mc := range config {
		var mct *loader.ModuleConfig
		id := mc.GroupID + ":" + mc.PolicyID + ":" + mc.Name
		for _, module := range s.modules {
			if module.id == id {
				mct = mc
				break
			}
		}
		if mct == nil {
			wantStartModules = append(wantStartModules, mc)
		}
	}

	if len(wantStartModules) == 0 {
		return
	}

	files, err := s.files.load(wantStartModules)
	if err != nil {
		return
	}
	if len(wantStartModules) != len(files) {
		return
	}

	mupdate := make(map[string]struct{})
	for idx, mc := range wantStartModules {
		id := mc.GroupID + ":" + mc.PolicyID + ":" + mc.Name
		module := s.newModule(mc, files[idx])
		if s.loader.Get(id) != nil {
			continue
		}

		if err := s.startModule(module); err != nil {
			continue
		}

		s.modules = append(s.modules, module)
		mupdate[mc.GroupID] = struct{}{}
	}

	if len(mupdate) > 0 {
		for groupID := range mupdate {
			s.notifyModules(groupID)
		}
	}
}

func (s *sController) checkStopModules(config []*loader.ModuleConfig) {
	var wantStopModules []*loader.ModuleConfig

	for _, module := range s.modules {
		var mct *loader.ModuleConfig
		for _, mc := range config {
			id := mc.GroupID + ":" + mc.PolicyID + ":" + mc.Name
			if module.id == id {
				mct = mc
				break
			}
		}
		if mct == nil {
			wantStopModules = append(wantStopModules, module.GetConfig())
		}
	}

	if len(wantStopModules) == 0 {
		return
	}

	mupdate := make(map[string]struct{})
	for _, mc := range wantStopModules {
		id := mc.GroupID + ":" + mc.PolicyID + ":" + mc.Name
		for mdx, module := range s.modules {
			if module.id == id {
				if s.loader.Get(id) == nil {
					continue
				}

				if err := s.stopModule(module, "module_remove"); err != nil {
					break
				}

				mupdate[mc.GroupID] = struct{}{}
				// It's possible because there used break in the bottom
				s.modules = append(s.modules[:mdx], s.modules[mdx+1:]...)
				break
			}
		}
	}

	if len(mupdate) > 0 {
		for groupID := range mupdate {
			s.notifyModules(groupID)
		}
	}
}

func (s *sController) checkUpdateModules(config []*loader.ModuleConfig) {
	var (
		wantUpdateModules       []*loader.ModuleConfig
		wantUpdateConfigModules []*loader.ModuleConfig
		mupdate                 = make(map[string]struct{})
	)
	isEqualModuleConfig := func(mc1, mc2 *loader.ModuleConfig) bool {
		return mc1.LastUpdate == mc2.LastUpdate
	}
	isEqualModuleVersion := func(mc1, mc2 *loader.ModuleConfig) bool {
		isSameState := mc1.State == mc2.State
		isSameTemplate := mc1.Template == mc2.Template
		isSameVersion := mc1.Version.String() == mc2.Version.String()
		isSameLastModuleUpdate := mc1.LastModuleUpdate == mc2.LastModuleUpdate
		return isSameState && isSameTemplate && isSameVersion && isSameLastModuleUpdate
	}

	for _, mc := range config {
		id := mc.GroupID + ":" + mc.PolicyID + ":" + mc.Name
		for _, module := range s.modules {
			if module.id == id && !isEqualModuleVersion(module.config, mc) {
				wantUpdateModules = append(wantUpdateModules, mc)
				break
			}
			if module.id == id && !isEqualModuleConfig(module.config, mc) {
				wantUpdateConfigModules = append(wantUpdateConfigModules, mc)
				break
			}
		}
	}

	files, err := s.files.load(wantUpdateModules)
	if err != nil {
		return
	}
	if len(wantUpdateModules) != len(files) {
		return
	}

	for idx, mc := range wantUpdateModules {
		id := mc.GroupID + ":" + mc.PolicyID + ":" + mc.Name
		for mdx, module := range s.modules {
			if module.id == id {
				s.modules[mdx] = s.newModule(mc, files[idx])
				if err := s.updateModule(s.modules[mdx], "module_update"); err == nil {
					mupdate[mc.GroupID] = struct{}{}
				}
				break
			}
		}
	}

	for _, mc := range wantUpdateConfigModules {
		id := mc.GroupID + ":" + mc.PolicyID + ":" + mc.Name
		for mdx, module := range s.modules {
			if module.id == id {
				if err := s.updateModuleConfig(s.modules[mdx], mc); err == nil {
					mupdate[mc.GroupID] = struct{}{}
				}
				break
			}
		}
	}

	if len(mupdate) > 0 {
		for groupID := range mupdate {
			s.notifyModules(groupID)
		}
	}
}

func (s *sController) updaterModules() {
	defer s.wg.Done()
	defer func() { s.closed = true }()

	for {
		s.mutex.Lock()
		config, err := s.config.load()
		if err == nil {
			s.checkUpdateModules(config)
			s.checkStopModules(config)
			s.checkStartModules(config)
		}
		s.mutex.Unlock()

		select {
		case <-time.NewTimer(syncModulesInterval).C:
			continue
		case <-s.quit:
		}

		break
	}
}
