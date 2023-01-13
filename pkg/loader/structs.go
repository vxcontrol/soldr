package loader

import (
	"encoding/json"
	"fmt"
	"strings"
)

// ModuleConfig is struct for contains module configuration
type ModuleConfig struct {
	GroupID           string              `json:"group_id"`
	PolicyID          string              `json:"policy_id"`
	State             string              `json:"state"`
	Template          string              `json:"template"`
	OS                map[string][]string `json:"os"`
	Name              string              `json:"name"`
	Version           ModuleVersion       `json:"version"`
	Actions           []string            `json:"actions"`
	Events            []string            `json:"events"`
	Fields            []string            `json:"fields"`
	LastModuleUpdate  string              `json:"last_module_update"`
	LastUpdate        string              `json:"last_update"`
	IConfigItem       `json:"-"`
	IConfigItemUpdate `json:"-"`
}

// Update is function for implement an interface to update current module config
func (mc *ModuleConfig) Update(mcn *ModuleConfig) error {
	if mcd, err := json.Marshal(mcn); err != nil {
		return fmt.Errorf("failed to serialize the module config: %w", err)
	} else if err = json.Unmarshal(mcd, mc); err != nil {
		return fmt.Errorf("failed to parse the modules config: %w", err)
	}
	if mc.IConfigItemUpdate != nil {
		return mc.IConfigItemUpdate.Update(mcn.IConfigItem)
	}
	return nil
}

// IConfigItem is common interface for manage module configuration
type IConfigItem interface {
	GetConfigSchema() string
	GetDefaultConfig() string
	GetCurrentConfig() string
	SetCurrentConfig(string) bool
	GetStaticDependencies() string
	GetDynamicDependencies() string
	SetDynamicDependencies(string) bool
	GetFieldsSchema() string
	GetActionConfigSchema() string
	GetDefaultActionConfig() string
	GetCurrentActionConfig() string
	SetCurrentActionConfig(string) bool
	GetEventConfigSchema() string
	GetDefaultEventConfig() string
	GetCurrentEventConfig() string
	SetCurrentEventConfig(string) bool
	GetSecureConfigSchema() string
	GetSecureDefaultConfig() string
	GetSecureCurrentConfig() string
	SetSecureCurrentConfig(string) bool
}

// IConfigItemUpdate is common interface for manage module configuration
type IConfigItemUpdate interface {
	Update(IConfigItem) error
}

// ModuleConfig is struct for contains module semantic version format
type ModuleVersion struct {
	Major uint64 `json:"major"`
	Minor uint64 `json:"minor"`
	Patch uint64 `json:"patch"`
}

// String is function for implement generic interface to convert value to string
func (mv *ModuleVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", mv.Major, mv.Minor, mv.Patch)
}

// ModuleConfigItem is a static configuration which loaded from protobuf
type ModuleConfigItem struct {
	ConfigSchema        string
	DefaultConfig       string
	CurrentConfig       string
	StaticDependencies  string
	DynamicDependencies string
	FieldsSchema        string
	ActionConfigSchema  string
	DefaultActionConfig string
	CurrentActionConfig string
	EventConfigSchema   string
	DefaultEventConfig  string
	CurrentEventConfig  string
	SecureConfigSchema  string
	SecureDefaultConfig string
	SecureCurrentConfig string
}

// ModuleItem is struct that contains files and args for each module
type ModuleItem struct {
	args  map[string][]string
	files map[string][]byte
}

// ModuleFiles is struct that contains cmodule and smodule files
type ModuleFiles struct {
	smodule *ModuleItem
	cmodule *ModuleItem
}

// NewFiles is function that construct module files
func NewFiles() *ModuleFiles {
	return &ModuleFiles{
		smodule: NewItem(),
		cmodule: NewItem(),
	}
}

// NewItem is function that construct module item
func NewItem() *ModuleItem {
	var mi ModuleItem
	mi.args = make(map[string][]string)
	mi.files = make(map[string][]byte)

	return &mi
}

// Update is function which rewrite all internal fields
func (mci *ModuleConfigItem) Update(nmci IConfigItem) error {
	mci.ConfigSchema = nmci.GetConfigSchema()
	mci.DefaultConfig = nmci.GetDefaultConfig()
	mci.CurrentConfig = nmci.GetCurrentConfig()
	mci.StaticDependencies = nmci.GetStaticDependencies()
	mci.DynamicDependencies = nmci.GetDynamicDependencies()
	mci.FieldsSchema = nmci.GetFieldsSchema()
	mci.ActionConfigSchema = nmci.GetActionConfigSchema()
	mci.DefaultActionConfig = nmci.GetDefaultActionConfig()
	mci.CurrentActionConfig = nmci.GetCurrentActionConfig()
	mci.EventConfigSchema = nmci.GetEventConfigSchema()
	mci.DefaultEventConfig = nmci.GetDefaultEventConfig()
	mci.CurrentEventConfig = nmci.GetCurrentEventConfig()
	mci.SecureConfigSchema = nmci.GetSecureConfigSchema()
	mci.SecureDefaultConfig = nmci.GetSecureDefaultConfig()
	mci.SecureCurrentConfig = nmci.GetSecureCurrentConfig()

	return nil
}

// GetConfigSchema is function which return schema module config
func (mci *ModuleConfigItem) GetConfigSchema() string {
	return mci.ConfigSchema
}

// GetDefaultConfig is function which return default module config
func (mci *ModuleConfigItem) GetDefaultConfig() string {
	return mci.DefaultConfig
}

// GetCurrentConfig is function which return current module config
func (mci *ModuleConfigItem) GetCurrentConfig() string {
	return mci.CurrentConfig
}

// SetCurrentConfig is function which store new module config to item
func (mci *ModuleConfigItem) SetCurrentConfig(config string) bool {
	mci.CurrentConfig = config
	return true
}

// GetStaticDependencies is function which return static module dependencies
func (mci *ModuleConfigItem) GetStaticDependencies() string {
	return mci.StaticDependencies
}

// GetDynamicDependencies is function which return dynamic module dependencies
func (mci *ModuleConfigItem) GetDynamicDependencies() string {
	return mci.DynamicDependencies
}

// SetDynamicDependencies is function which store new dynamic module dependencies to item
func (mci *ModuleConfigItem) SetDynamicDependencies(config string) bool {
	mci.DynamicDependencies = config
	return true
}

// GetFieldsSchema is function which return schema fields for actions and events
func (mci *ModuleConfigItem) GetFieldsSchema() string {
	return mci.FieldsSchema
}

// GetActionConfigSchema is function which return schema action config
func (mci *ModuleConfigItem) GetActionConfigSchema() string {
	return mci.ActionConfigSchema
}

// GetDefaultActionConfig is function which return default action config
func (mci *ModuleConfigItem) GetDefaultActionConfig() string {
	return mci.DefaultActionConfig
}

// GetCurrentActionConfig is function which return current action config
func (mci *ModuleConfigItem) GetCurrentActionConfig() string {
	return mci.CurrentActionConfig
}

// SetCurrentActionConfig is function which store new action config to item
func (mci *ModuleConfigItem) SetCurrentActionConfig(config string) bool {
	mci.CurrentActionConfig = config
	return true
}

// GetEventConfigSchema is function which return schema event config
func (mci *ModuleConfigItem) GetEventConfigSchema() string {
	return mci.EventConfigSchema
}

// GetDefaultEventConfig is function which return default event config
func (mci *ModuleConfigItem) GetDefaultEventConfig() string {
	return mci.DefaultEventConfig
}

// GetCurrentEventConfig is function which return current event config
func (mci *ModuleConfigItem) GetCurrentEventConfig() string {
	return mci.CurrentEventConfig
}

// SetCurrentEventConfig is function which store new event config to item
func (mci *ModuleConfigItem) SetCurrentEventConfig(config string) bool {
	mci.CurrentEventConfig = config
	return true
}

func (mci *ModuleConfigItem) GetSecureConfigSchema() string {
	return mci.SecureConfigSchema
}

func (mci *ModuleConfigItem) GetSecureDefaultConfig() string {
	return mci.SecureDefaultConfig
}

func (mci *ModuleConfigItem) GetSecureCurrentConfig() string {
	return mci.SecureCurrentConfig
}

func (mci *ModuleConfigItem) SetSecureCurrentConfig(config string) bool {
	mci.SecureCurrentConfig = config
	return true
}

// GetFiles is function which return files structure
func (mi *ModuleItem) GetFiles() map[string][]byte {
	return mi.files
}

// GetFilesByFilter is function which return files structure
func (mi *ModuleItem) GetFilesByFilter(os, arch string) map[string][]byte {
	var strictPrefix string
	clibsPrefix := "clibs/"
	mfiles := make(map[string][]byte)

	if os == "" {
		strictPrefix = clibsPrefix
	} else if arch == "" {
		strictPrefix = clibsPrefix + os + "/"
	} else {
		strictPrefix = clibsPrefix + os + "/" + arch + "/"
	}

	for path, data := range mi.files {
		if strings.HasPrefix(path, clibsPrefix) && !strings.HasPrefix(path, strictPrefix) {
			continue
		}
		mfiles[path] = data
	}

	return mfiles
}

// GetArgs is function which return arguments structure
func (mi *ModuleItem) GetArgs() map[string][]string {
	return mi.args
}

// SetFiles is function which store files structure to item
func (mi *ModuleItem) SetFiles(files map[string][]byte) {
	mi.files = files
}

// SetArgs is function which store arguments structure to item
func (mi *ModuleItem) SetArgs(args map[string][]string) {
	mi.args = args
}

// GetSModule is function which return server modules structure
func (mf *ModuleFiles) GetSModule() *ModuleItem {
	return mf.smodule
}

// GetCModule is function which return client modules structure
func (mf *ModuleFiles) GetCModule() *ModuleItem {
	return mf.cmodule
}

// SetSModule is function which store server modules structure
func (mf *ModuleFiles) SetSModule(mi *ModuleItem) {
	mf.smodule = mi
}

// SetCModule is function which store client modules structure
func (mf *ModuleFiles) SetCModule(mi *ModuleItem) {
	mf.cmodule = mi
}
