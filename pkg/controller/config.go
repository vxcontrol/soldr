package controller

import (
	"fmt"

	"soldr/pkg/db"
	"soldr/pkg/filestorage/fs"
	"soldr/pkg/filestorage/s3"
)

type getCallback func() string
type setCallback func(string) bool

// sConfig is universal container for modules configuration loader
type sConfig struct {
	clt tConfigLoaderType
	IConfigLoader
}

// sConfigItem is struct for contains schema, default and current config data
type sConfigItem struct {
	getConfigSchema        getCallback
	getDefaultConfig       getCallback
	getCurrentConfig       getCallback
	setCurrentConfig       setCallback
	getStaticDependencies  getCallback
	getDynamicDependencies getCallback
	setDynamicDependencies setCallback
	getFieldsSchema        getCallback
	getActionConfigSchema  getCallback
	getDefaultActionConfig getCallback
	getCurrentActionConfig getCallback
	setCurrentActionConfig setCallback
	getEventConfigSchema   getCallback
	getDefaultEventConfig  getCallback
	getCurrentEventConfig  getCallback
	setCurrentEventConfig  setCallback
	getSecureConfigSchema  getCallback
	getSecureDefaultConfig getCallback
	getSecureCurrentConfig getCallback
	setSecureCurrentConfig setCallback
}

// GetConfigSchema is function which return JSON schema data of config as string
func (ci *sConfigItem) GetConfigSchema() string {
	if ci.getConfigSchema != nil {
		return ci.getConfigSchema()
	}

	return ""
}

// GetDefaultConfig is function which return default config data as string
func (ci *sConfigItem) GetDefaultConfig() string {
	if ci.getDefaultConfig != nil {
		return ci.getDefaultConfig()
	}

	return ""
}

// GetCurrentConfig is function which return current config data as string
func (ci *sConfigItem) GetCurrentConfig() string {
	if ci.getCurrentConfig != nil {
		return ci.getCurrentConfig()
	}

	return ""
}

// SetCurrentConfig is function which store this string config data to storage
func (ci *sConfigItem) SetCurrentConfig(config string) bool {
	if ci.setCurrentConfig != nil {
		return ci.setCurrentConfig(config)
	}

	return false
}

// GetStaticDependencies is function which return static module dependencies as string
func (ci *sConfigItem) GetStaticDependencies() string {
	if ci.getStaticDependencies != nil {
		return ci.getStaticDependencies()
	}

	return ""
}

// GetDynamicDependencies is function which return dynamic module dependencies as string
func (ci *sConfigItem) GetDynamicDependencies() string {
	if ci.getDynamicDependencies != nil {
		return ci.getDynamicDependencies()
	}

	return ""
}

// SetDynamicDependencies is function which store this string dynamic module dependencies data to storage
func (ci *sConfigItem) SetDynamicDependencies(config string) bool {
	if ci.setDynamicDependencies != nil {
		return ci.setDynamicDependencies(config)
	}

	return false
}

// GetFieldsSchema is function which return JSON schema of actions and events fields as string
func (ci *sConfigItem) GetFieldsSchema() string {
	if ci.getFieldsSchema != nil {
		return ci.getFieldsSchema()
	}

	return ""
}

// GetActionConfigSchema is function which return JSON schema data of action config as string
func (ci *sConfigItem) GetActionConfigSchema() string {
	if ci.getActionConfigSchema != nil {
		return ci.getActionConfigSchema()
	}

	return ""
}

// GetDefaultActionConfig is function which return default action config data as string
func (ci *sConfigItem) GetDefaultActionConfig() string {
	if ci.getDefaultActionConfig != nil {
		return ci.getDefaultActionConfig()
	}

	return ""
}

// GetCurrentActionConfig is function which return current action config data as string
func (ci *sConfigItem) GetCurrentActionConfig() string {
	if ci.getCurrentActionConfig != nil {
		return ci.getCurrentActionConfig()
	}

	return ""
}

// SetCurrentActionConfig is function which store this string action config data to storage
func (ci *sConfigItem) SetCurrentActionConfig(config string) bool {
	if ci.setCurrentActionConfig != nil {
		return ci.setCurrentActionConfig(config)
	}

	return false
}

// GetEventConfigSchema is function which return JSON schema data of event config as string
func (ci *sConfigItem) GetEventConfigSchema() string {
	if ci.getEventConfigSchema != nil {
		return ci.getEventConfigSchema()
	}

	return ""
}

// GetDefaultEventConfig is function which return default event config data as string
func (ci *sConfigItem) GetDefaultEventConfig() string {
	if ci.getDefaultEventConfig != nil {
		return ci.getDefaultEventConfig()
	}

	return ""
}

// GetCurrentEventConfig is function which return current event config data as string
func (ci *sConfigItem) GetCurrentEventConfig() string {
	if ci.getCurrentEventConfig != nil {
		return ci.getCurrentEventConfig()
	}

	return ""
}

// SetCurrentEventConfig is function which store this string event config data to storage
func (ci *sConfigItem) SetCurrentEventConfig(config string) bool {
	if ci.setCurrentEventConfig != nil {
		return ci.setCurrentEventConfig(config)
	}

	return false
}

func (ci *sConfigItem) GetSecureConfigSchema() string {
	if ci.getSecureConfigSchema != nil {
		return ci.getSecureConfigSchema()
	}

	return ""
}

func (ci *sConfigItem) GetSecureDefaultConfig() string {
	if ci.getSecureDefaultConfig != nil {
		return ci.getSecureDefaultConfig()
	}

	return ""
}

func (ci *sConfigItem) GetSecureCurrentConfig() string {
	if ci.getSecureCurrentConfig != nil {
		return ci.getSecureCurrentConfig()
	}

	return ""
}

func (ci *sConfigItem) SetSecureCurrentConfig(config string) bool {
	if ci.setSecureCurrentConfig != nil {
		return ci.setSecureCurrentConfig(config)
	}

	return false
}

// NewConfigFromDB is function which constructed Configuration loader object
func NewConfigFromDB(dsn *db.DSN) (IConfigLoader, error) {
	dbc, err := db.New(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the DB driver: %w", err)
	}
	return &sConfig{
		clt:           eDBConfigLoader,
		IConfigLoader: &configLoaderDB{dbc: dbc},
	}, nil
}

// NewConfigFromS3 is function which constructed Configuration loader object
func NewConfigFromS3(connParams *s3.Config) (IConfigLoader, error) {
	sc, err := s3.New(connParams)
	if err != nil {
		return nil, generateDriverInitErrMsg(driverTypeS3, err)
	}
	return &sConfig{
		clt:           eS3ConfigLoader,
		IConfigLoader: &configLoaderS3{sc: sc},
	}, nil
}

// NewConfigFromFS is function which constructed Configuration loader object
func NewConfigFromFS(path string) (IConfigLoader, error) {
	sc, err := fs.New()
	if err != nil {
		return nil, generateDriverInitErrMsg(driverTypeFS, err)
	}
	return &sConfig{
		clt:           eFSConfigLoader,
		IConfigLoader: &configLoaderFS{path: path, sc: sc},
	}, nil
}

type driverType int

const (
	driverTypeS3 driverType = iota + 1
	driverTypeFS
)

func generateDriverInitErrMsg(t driverType, originalErr error) error {
	var driverName string
	switch t {
	case driverTypeS3:
		driverName = "S3"
	case driverTypeFS:
		driverName = "FS"
	default:
		driverName = fmt.Sprintf("type %d", t)
	}
	return fmt.Errorf("failed to initialize the %s driver: %w", driverName, originalErr)
}
