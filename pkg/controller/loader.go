package controller

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"soldr/pkg/db"
	"soldr/pkg/loader"
	"soldr/pkg/storage"
)

// tConfigLoaderType is type for loading config
type tConfigLoaderType int32

// Enum config loading types
const (
	eDBConfigLoader tConfigLoaderType = 0
	eS3ConfigLoader tConfigLoaderType = 1
	eFSConfigLoader tConfigLoaderType = 2
)

// List of SQL queries string
const (
	sLoadModulesSQL    string = "SELECT m.`id`, IFNULL(g.`hash`, '') AS `group_id`, IFNULL(p.`hash`, '') AS `policy_id`, m.`info`, m.`last_update`, m.`last_module_update`, m.`state`, m.`template` FROM `modules` m LEFT JOIN (SELECT * FROM `policies` UNION SELECT 0, '', '{}', NOW(), NOW(), NULL) p ON m.`policy_id` = p.`id` AND p.deleted_at IS NULL LEFT JOIN `groups_to_policies` gp ON gp.`policy_id` = p.`id` LEFT JOIN (SELECT * FROM `groups` UNION SELECT 0, '', '{}', NOW(), NOW(), NULL) g ON gp.`group_id` = g.`id` AND g.deleted_at IS NULL WHERE m.`status` = 'joined' AND NOT (ISNULL(g.`hash`) AND p.`hash` NOT LIKE '') AND m.deleted_at IS NULL"
	sGetModuleFieldSQL string = "SELECT `%s` FROM `modules` WHERE `id` = ? LIMIT 1"
	sSetModuleFieldSQL string = "UPDATE `modules` SET `%s` = ? WHERE `id` = ?"
)

// tFilesLoaderType is type for loading module
type tFilesLoaderType int32

// Enum files loading types
const (
	eS3FilesLoader tFilesLoaderType = 0
	eFSFilesLoader tFilesLoaderType = 1
)

// IConfigLoader is internal interface for loading config from external storage
type IConfigLoader interface {
	load() ([]*loader.ModuleConfig, error)
}

// IFilesLoader is internal interface for loading files from external storage
type IFilesLoader interface {
	load(mcl []*loader.ModuleConfig) ([]*loader.ModuleFiles, error)
}

// configLoaderDB is container for config which loaded from DB
type configLoaderDB struct {
	dbc *db.DB
}

func (cl *configLoaderDB) getCb(id, col string) getCallback {
	sql := fmt.Sprintf(sGetModuleFieldSQL, col)
	return func() string {
		if cl.dbc == nil {
			return ""
		}

		rows, err := cl.dbc.Query(sql, id)
		if err != nil {
			return ""
		}
		if len(rows) != 1 {
			return ""
		}
		if data, ok := rows[0][col]; ok {
			return data
		}

		return ""
	}
}

func (cl *configLoaderDB) setCb(id, col string) setCallback {
	sql := fmt.Sprintf(sSetModuleFieldSQL, col)
	return func(val string) bool {
		if cl.dbc == nil {
			return false
		}

		if _, err := cl.dbc.Exec(sql, val, id); err != nil {
			return false
		}

		return true
	}
}

func joinPath(args ...string) string {
	tpath := filepath.Join(args...)
	return strings.Replace(tpath, "\\", "/", -1)
}

// load is function what retrieve modules config list from DB
func (cl *configLoaderDB) load() ([]*loader.ModuleConfig, error) {
	if cl.dbc == nil {
		return nil, fmt.Errorf("db connection is not initialized")
	}

	rows, err := cl.dbc.Query(sLoadModulesSQL)
	if err != nil {
		return nil, err
	}

	var ml []*loader.ModuleConfig
	for _, m := range rows {
		var mc loader.ModuleConfig
		const moduleInfoField = "info"
		if info, ok := m[moduleInfoField]; ok {
			if err = json.Unmarshal([]byte(info), &mc); err != nil {
				return nil, fmt.Errorf("failed to parse the module config: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load the module config: returned rows do not contain the field '%s'", moduleInfoField)
		}
		if groupID, ok := m["group_id"]; ok {
			mc.GroupID = groupID
		}
		if policyID, ok := m["policy_id"]; ok {
			mc.PolicyID = policyID
		}
		if lastUpdate, ok := m["last_update"]; ok {
			mc.LastUpdate = lastUpdate
		}
		if lastModuleUpdate, ok := m["last_module_update"]; ok {
			mc.LastModuleUpdate = lastModuleUpdate
		}
		if state, ok := m["state"]; ok {
			mc.State = state
		}
		if template, ok := m["template"]; ok {
			mc.Template = template
		}
		ci := &sConfigItem{}
		if id, ok := m["id"]; ok {
			ci.getConfigSchema = cl.getCb(id, "config_schema")
			ci.getDefaultConfig = cl.getCb(id, "default_config")
			ci.getCurrentConfig = cl.getCb(id, "current_config")
			ci.setCurrentConfig = cl.setCb(id, "current_config")
			ci.getStaticDependencies = cl.getCb(id, "static_dependencies")
			ci.getDynamicDependencies = cl.getCb(id, "dynamic_dependencies")
			ci.setDynamicDependencies = cl.setCb(id, "dynamic_dependencies")
			ci.getFieldsSchema = cl.getCb(id, "fields_schema")
			ci.getActionConfigSchema = cl.getCb(id, "action_config_schema")
			ci.getDefaultActionConfig = cl.getCb(id, "default_action_config")
			ci.getCurrentActionConfig = cl.getCb(id, "current_action_config")
			ci.setCurrentActionConfig = cl.setCb(id, "current_action_config")
			ci.getEventConfigSchema = cl.getCb(id, "event_config_schema")
			ci.getDefaultEventConfig = cl.getCb(id, "default_event_config")
			ci.getCurrentEventConfig = cl.getCb(id, "current_event_config")
			ci.setCurrentEventConfig = cl.setCb(id, "current_event_config")
			ci.getSecureConfigSchema = cl.getCb(id, "secure_config_schema")
			ci.getSecureDefaultConfig = cl.getCb(id, "secure_default_config")
			ci.getSecureCurrentConfig = cl.getCb(id, "secure_current_config")
			ci.setSecureCurrentConfig = cl.setCb(id, "secure_current_config")
		}
		mc.IConfigItem = ci
		ml = append(ml, &mc)
	}

	return ml, nil
}

func readConfig(s storage.IStorage, path string) ([]*loader.ModuleConfig, error) {
	var mcl []*loader.ModuleConfig
	if s.IsNotExist(path) {
		return nil, fmt.Errorf("the config directory '%s' not found", path)
	}
	cpath := strings.Replace(filepath.Join(path, "config.json"), "\\", "/", -1)
	if s.IsNotExist(cpath) {
		return nil, fmt.Errorf("the config file '%s' not found", cpath)
	}
	cdata, err := s.ReadFile(cpath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the config file '%s': %w", cpath, err)
	}
	if err = json.Unmarshal(cdata, &mcl); err != nil {
		return nil, fmt.Errorf("failed to parse the modules config file '%s': %w", cpath, err)
	}
	return mcl, nil
}

func writeConfig(s storage.IStorage, path string, mcl []*loader.ModuleConfig) error {
	if s.IsNotExist(path) {
		return fmt.Errorf("config directory '%s' not found", path)
	}
	cdata, err := json.MarshalIndent(mcl, "", "    ")
	if err != nil {
		return fmt.Errorf("failed to serialize the modules config: %w", err)
	}
	cpath := strings.Replace(filepath.Join(path, "config.json"), "\\", "/", -1)
	if err := s.WriteFile(cpath, cdata); err != nil {
		return fmt.Errorf("failed to write the config file '%s': %w", cpath, err)
	}
	return nil
}

func parsePathToFile(mpath string) (string, string, error) {
	pparts := []string{}
	for _, ppart := range strings.Split(mpath, "/") {
		if ppart == "" {
			pparts = append(pparts, "/")
		} else {
			pparts = append(pparts, ppart)
		}
	}
	if len(pparts) < 3 {
		return "", "", fmt.Errorf("invalid path format: expected 3 parts, actually got %d", len(pparts))
	}
	mname := pparts[len(pparts)-3]
	bpath := joinPath(pparts[:len(pparts)-3]...)
	return mname, bpath, nil
}

func getStorageCb(s storage.IStorage, mpath, file string) getCallback {
	return func() string {
		data, err := s.ReadFile(joinPath(mpath, file))
		if err == nil {
			return string(data)
		}

		return ""
	}
}

func setStorageCb(s storage.IStorage, mpath, file string) setCallback {
	return func(val string) bool {
		mname, bpath, err := parsePathToFile(mpath)
		if err != nil {
			return false
		}
		mcl, err := readConfig(s, bpath)
		if err != nil {
			return false
		}
		for _, mc := range mcl {
			if mc.Name == mname {
				mc.LastUpdate = time.Now().Format("2006-01-02 15:04:05")
				break
			}
		}
		if err = writeConfig(s, bpath, mcl); err != nil {
			return false
		}
		return s.WriteFile(joinPath(mpath, file), []byte(val)) == nil
	}
}

func loadConfig(s storage.IStorage, path string) ([]*loader.ModuleConfig, error) {
	mcl, err := readConfig(s, path)
	if err != nil {
		return nil, err
	}
	for _, mc := range mcl {
		mpath := joinPath(path, mc.Name, mc.Version.String(), "config")
		ci := &sConfigItem{
			getConfigSchema:        getStorageCb(s, mpath, "config_schema.json"),
			getDefaultConfig:       getStorageCb(s, mpath, "default_config.json"),
			getCurrentConfig:       getStorageCb(s, mpath, "current_config.json"),
			setCurrentConfig:       setStorageCb(s, mpath, "current_config.json"),
			getStaticDependencies:  getStorageCb(s, mpath, "static_dependencies.json"),
			getDynamicDependencies: getStorageCb(s, mpath, "dynamic_dependencies.json"),
			setDynamicDependencies: setStorageCb(s, mpath, "dynamic_dependencies.json"),
			getFieldsSchema:        getStorageCb(s, mpath, "fields_schema.json"),
			getActionConfigSchema:  getStorageCb(s, mpath, "action_config_schema.json"),
			getDefaultActionConfig: getStorageCb(s, mpath, "default_action_config.json"),
			getCurrentActionConfig: getStorageCb(s, mpath, "current_action_config.json"),
			setCurrentActionConfig: setStorageCb(s, mpath, "current_action_config.json"),
			getEventConfigSchema:   getStorageCb(s, mpath, "event_config_schema.json"),
			getDefaultEventConfig:  getStorageCb(s, mpath, "default_event_config.json"),
			getCurrentEventConfig:  getStorageCb(s, mpath, "current_event_config.json"),
			setCurrentEventConfig:  setStorageCb(s, mpath, "current_event_config.json"),
			getSecureConfigSchema:  getStorageCb(s, mpath, "secure_config_schema.json"),
			getSecureDefaultConfig: getStorageCb(s, mpath, "secure_default_config.json"),
			getSecureCurrentConfig: getStorageCb(s, mpath, "secure_current_config.json"),
			setSecureCurrentConfig: setStorageCb(s, mpath, "secure_current_config.json"),
		}
		mc.IConfigItem = ci
	}

	return mcl, nil
}

// configLoaderS3 is container for config which loaded from D3
type configLoaderS3 struct {
	sc storage.IStorage
}

// load is function what retrieve modules config list from S3
func (cl *configLoaderS3) load() ([]*loader.ModuleConfig, error) {
	return loadConfig(cl.sc, "/")
}

// configLoaderFS is container for config which loaded from FS
type configLoaderFS struct {
	path string
	sc   storage.IStorage
}

// load is function what retrieve modules config list from FS
func (cl *configLoaderFS) load() ([]*loader.ModuleConfig, error) {
	return loadConfig(cl.sc, cl.path)
}

func removeLeadSlash(files map[string][]byte) map[string][]byte {
	rfiles := make(map[string][]byte)
	for name, data := range files {
		rfiles[name[1:]] = data
	}
	return rfiles
}

func loadUtils(s storage.IStorage, path string) (map[string][]byte, error) {
	var err error
	upath := joinPath(path, "utils")
	if s.IsNotExist(upath) {
		return nil, fmt.Errorf("utils directory '%s' not found", upath)
	}

	files, err := s.ReadDirRec(upath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the files from the utils directory '%s': %w", upath, err)
	}
	files = removeLeadSlash(files)

	return files, nil
}

func loadFiles(s storage.IStorage, path string, mcl []*loader.ModuleConfig) ([]*loader.ModuleFiles, error) {
	var mfl []*loader.ModuleFiles
	if s.IsNotExist(path) {
		return nil, fmt.Errorf("modules directory '%s' not found", path)
	}

	utils, err := loadUtils(s, path)
	if err != nil {
		return nil, err
	}

	for _, mc := range mcl {
		var mf loader.ModuleFiles
		loadModuleDir := func(dir string) (*loader.ModuleItem, error) {
			mpath := joinPath(path, mc.Name, mc.Version.String(), dir)
			if s.IsNotExist(mpath) {
				return nil, fmt.Errorf("module directory '%s' not found", mpath)
			}
			var mi loader.ModuleItem
			files, err := s.ReadDirRec(mpath)
			if err != nil {
				return nil, fmt.Errorf("failed to read the files from the modules directory '%s': %w", mpath, err)
			}
			files = removeLeadSlash(files)
			for p, d := range utils {
				if _, ok := files[p]; !ok {
					files[p] = d
				}
			}
			args := make(map[string][]string)
			if data, ok := files["args.json"]; ok {
				if err = json.Unmarshal(data, &args); err != nil {
					return nil, fmt.Errorf("failed to parse the module args: %w", err)
				}
			}
			mi.SetArgs(args)
			mi.SetFiles(files)

			return &mi, nil
		}

		var smi, cmi *loader.ModuleItem
		if cmi, err = loadModuleDir("cmodule"); err != nil {
			return nil, err
		}
		if smi, err = loadModuleDir("smodule"); err != nil {
			return nil, err
		}
		mf.SetCModule(cmi)
		mf.SetSModule(smi)
		mfl = append(mfl, &mf)
	}

	return mfl, nil
}

// filesLoaderS3 is container for files structure which loaded from S3
type filesLoaderS3 struct {
	sc storage.IStorage
}

// load is function what retrieve modules files data from S3
func (fl *filesLoaderS3) load(mcl []*loader.ModuleConfig) ([]*loader.ModuleFiles, error) {
	if len(mcl) == 0 {
		return []*loader.ModuleFiles{}, nil
	}

	return loadFiles(fl.sc, "/", mcl)
}

// filesLoaderFS is container for files structure which loaded from FS
type filesLoaderFS struct {
	path string
	sc   storage.IStorage
}

// load is function what retrieve modules files data from FS
func (fl *filesLoaderFS) load(mcl []*loader.ModuleConfig) ([]*loader.ModuleFiles, error) {
	if len(mcl) == 0 {
		return []*loader.ModuleFiles{}, nil
	}

	return loadFiles(fl.sc, fl.path, mcl)
}
