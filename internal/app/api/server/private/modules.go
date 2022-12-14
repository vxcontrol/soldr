package private

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	srverrors "soldr/internal/app/api/server/errors"
	"soldr/internal/app/api/utils"
	"soldr/internal/crypto"
	"soldr/internal/storage"
)

type agentModuleDetails struct {
	Name   string        `json:"name"`
	Today  uint64        `json:"today"`
	Total  uint64        `json:"total"`
	Update bool          `json:"update"`
	Policy models.Policy `json:"policy"`
}

type groupModuleDetails struct {
	Name   string        `json:"name"`
	Today  uint64        `json:"today"`
	Total  uint64        `json:"total"`
	Update bool          `json:"update"`
	Policy models.Policy `json:"policy"`
}

type policyModuleDetails struct {
	Name      string `json:"name"`
	Today     uint64 `json:"today"`
	Total     uint64 `json:"total"`
	Active    bool   `json:"active"`
	Exists    bool   `json:"exists"`
	Update    bool   `json:"update"`
	Duplicate bool   `json:"duplicate"`
}

type agentModules struct {
	Modules []models.ModuleA     `json:"modules"`
	Details []agentModuleDetails `json:"details"`
	Total   uint64               `json:"total"`
}

type groupModules struct {
	Modules []models.ModuleA     `json:"modules"`
	Details []groupModuleDetails `json:"details"`
	Total   uint64               `json:"total"`
}

type policyModules struct {
	Modules []models.ModuleA      `json:"modules"`
	Details []policyModuleDetails `json:"details"`
	Total   uint64                `json:"total"`
}

type policyModulesUpdates struct {
	Modules  []models.ModuleA `json:"modules"`
	Policies []models.Policy  `json:"policies"`
}

type systemModules struct {
	Modules []models.ModuleS `json:"modules"`
	Total   uint64           `json:"total"`
}

type systemShortModules struct {
	Modules []models.ModuleSShort `json:"modules"`
	Total   uint64                `json:"total"`
}

type policyModulePatch struct {
	// Action on group module must be one of activate, deactivate, update, store
	Action  string         `form:"action" json:"action" binding:"oneof=activate deactivate store update,required" default:"update" enums:"activate,deactivate,store,update"`
	Version string         `form:"version,omitempty" json:"version,omitempty" binding:"required_if=Action update,omitempty"`
	Module  models.ModuleA `form:"module,omitempty" json:"module,omitempty" binding:"required_if=Action store,omitempty"`
}

type moduleVersionPatch struct {
	// Action on group module must be one of store, release
	Action string         `form:"action" json:"action" binding:"oneof=store release,required" default:"store" enums:"store,release"`
	Module models.ModuleS `form:"module" json:"module" binding:"required"`
}

type systemModuleFile struct {
	Path string `form:"path" json:"path" binding:"required"`
	Data string `form:"data" json:"data" binding:"required" default:"base64"`
}

type systemModuleFilePatch struct {
	Action  string `form:"action" json:"action" binding:"oneof=move remove save,required" default:"save" enums:"move,remove,save"`
	Path    string `form:"path" json:"path" binding:"required"`
	Data    string `form:"data,omitempty" json:"data,omitempty" default:"base64" binding:"omitempty,required_if=Action save"`
	NewPath string `form:"newpath,omitempty" json:"newpath,omitempty" binding:"omitempty,required_if=Action move"`
}

const sqlAgentModuleDetails = `
	SELECT
		m.name,
		(SELECT COUNT(id)
		 FROM events AS e1
		 WHERE
			e1.module_id = m.id AND
			e1.agent_id = a.id AND
			DATE(e1.date) = CURDATE()
		) AS today,
		(SELECT COUNT(id)
		 FROM events e2
		 WHERE
			e2.module_id = m.id AND
			e2.agent_id = a.id
		) AS total
	FROM modules AS m
		LEFT JOIN policies p ON m.policy_id = p.id AND p.deleted_at IS NULL
		LEFT JOIN groups_to_policies AS gtp ON p.id = gtp.policy_id
		LEFT JOIN groups g ON gtp.group_id = g.id AND g.deleted_at IS NULL
		LEFT JOIN agents AS a ON g.id = a.group_id AND a.deleted_at IS NULL
	WHERE a.id = ? AND m.deleted_at IS NULL`

const sqlGroupModuleDetails = `
	SELECT
		m.name,
		SUM((SELECT COUNT(id)
			FROM events AS e1
			WHERE
			e1.module_id = m.id AND
			e1.agent_id = a.id AND
			DATE(e1.date) = CURDATE()
		)) AS today,
		SUM((SELECT COUNT(id)
			FROM events e2
			WHERE
			e2.module_id = m.id AND
			e2.agent_id = a.id
		)) AS total
	FROM modules AS m
		LEFT JOIN policies p ON m.policy_id = p.id AND p.deleted_at IS NULL
		LEFT JOIN groups_to_policies AS gtp ON p.id = gtp.policy_id
		LEFT JOIN groups g ON gtp.group_id = g.id AND g.deleted_at IS NULL
		LEFT JOIN agents AS a ON g.id = a.group_id AND a.deleted_at IS NULL
	WHERE g.id = ? AND m.deleted_at IS NULL
	GROUP BY m.name`

const sqlPolicyModuleDetails = `
	SELECT
		m.name,
		(1 = 1) AS 'exists',
		(m.status = "joined") AS active,
		(SELECT COUNT(id)
		 FROM events AS e1
		 WHERE
			e1.module_id = m.id AND
			DATE(e1.date) = CURDATE()
		) AS today,
		(SELECT COUNT(id)
		 FROM events e2
		 WHERE e2.module_id = m.id
		) AS total
	FROM modules AS m
		LEFT JOIN policies p ON m.policy_id = p.id AND p.deleted_at IS NULL
	WHERE p.id = ? AND m.deleted_at IS NULL`

var modulesSQLMappers = map[string]interface{}{
	"status":               "`{{table}}`.status",
	"system":               "`{{table}}`.system",
	"actions":              utils.ActionsMapper,
	"events":               utils.EventsMapper,
	"fields":               utils.FieldsMapper,
	"tags":                 utils.TagsMapper,
	"os":                   utils.ModulesOSMapper,
	"os_arch":              utils.ModulesOSArchMapper,
	"os_type":              utils.ModulesOSTypeMapper,
	"version":              "`{{table}}`.version",
	"ver_major":            "`{{table}}`.ver_major",
	"ver_minor":            "`{{table}}`.ver_minor",
	"ver_patch":            "`{{table}}`.ver_patch",
	"state":                "`{{table}}`.state",
	"localizedName":        "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.title'))",
	"localizedDescription": "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.description'))",
	"localizedTagsName":    "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$.tags.*.{{lang}}.title'))",
	"localizedEventsName":  "JSON_UNQUOTE(JSON_EXTRACT(`{{table}}`.locale, '$.events.*.{{lang}}.title'))",
	"data": "CONCAT(`{{table}}`.name, ' | ', `{{table}}`.version, ' | ', " +
		"COALESCE(JSON_KEYS(`{{table}}`.changelog), ''), ' | ', `{{table}}`.state, ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.info, '$.tags'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.events.*.{{lang}}.title'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.title'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.module.{{lang}}.description'), ''), ' | ', " +
		"COALESCE(JSON_EXTRACT(`{{table}}`.locale, '$.tags.*.{{lang}}.title'), ''))",
	"name": "`{{table}}`.name",
}

// Template is container for all module files
type Template map[string]map[string][]byte

func getService(c *gin.Context) *models.Service {
	var sv *models.Service

	if val, ok := c.Get("SV"); !ok {
		utils.FromContext(c).WithError(nil).Errorf("error getting vxservice instance from context")
	} else if sv = val.(*models.Service); sv == nil {
		utils.FromContext(c).WithError(nil).Errorf("got nil value vxservice instance from context")
	}

	return sv
}

func getDBEncryptor(c *gin.Context) crypto.IDBConfigEncryptor {
	var encryptor crypto.IDBConfigEncryptor

	if cr, ok := c.Get("crp"); !ok {
		utils.FromContext(c).WithError(nil).Errorf("error getting secure config encryptor from context")
	} else if encryptor = cr.(crypto.IDBConfigEncryptor); encryptor == nil {
		utils.FromContext(c).WithError(nil).Errorf("got nil value secure config encryptor from context")
	}

	return encryptor
}

func joinPath(args ...string) string {
	tpath := filepath.Join(args...)
	return strings.Replace(tpath, "\\", "/", -1)
}

func removeLeadSlash(files map[string][]byte) map[string][]byte {
	rfiles := make(map[string][]byte)
	for name, data := range files {
		rfiles[name[1:]] = data
	}
	return rfiles
}

func readDir(s storage.IStorage, path string) ([]string, error) {
	var files []string
	list, err := s.ListDir(path)
	if err != nil {
		return files, err
	}
	for _, info := range list {
		if info.IsDir() {
			list, err := readDir(s, path+"/"+info.Name())
			if err != nil {
				return files, err
			}
			files = append(files, list...)
		} else {
			files = append(files, path+"/"+info.Name())
		}
	}
	return files, nil
}

func LoadModuleSConfig(files map[string][]byte) (*models.ModuleS, error) {
	var module models.ModuleS
	targets := map[string]interface{}{
		"action_config_schema":  &module.ActionConfigSchema,
		"changelog":             &module.Changelog,
		"config_schema":         &module.ConfigSchema,
		"default_action_config": &module.DefaultActionConfig,
		"default_config":        &module.DefaultConfig,
		"default_event_config":  &module.DefaultEventConfig,
		"event_config_schema":   &module.EventConfigSchema,
		"fields_schema":         &module.FieldsSchema,
		"locale":                &module.Locale,
		"static_dependencies":   &module.StaticDependencies,
		"secure_config_schema":  &module.SecureConfigSchema,
		"secure_default_config": &module.SecureDefaultConfig,
	}

	fillMissingFileContents("", files)

	for filename, container := range targets {
		if err := json.Unmarshal(files[filename+".json"], container); err != nil {
			return nil, errors.New("failed unmarshal " + filename + ": " + err.Error())
		}
	}

	return &module, nil
}

func PatchModuleSConfig(module *models.ModuleS) error {
	type patchLocaleCfg struct {
		src     map[string]models.ModuleLocaleDesc
		list    []string
		locType string
	}
	type patchLocaleLst struct {
		src     map[string]map[string]models.ModuleLocaleDesc
		list    []string
		locType string
	}

	placeholder := "{{placeholder}}"
	languages := []string{"ru", "en"}
	patchLocaleLstList := []patchLocaleLst{
		{
			src:     module.Locale.ActionConfig,
			list:    module.Info.Actions,
			locType: "actions",
		},
		{
			src:     module.Locale.EventConfig,
			list:    module.Info.Events,
			locType: "events",
		},
	}
	patchLocaleCfgList := []patchLocaleCfg{
		{
			src:     module.Locale.Actions,
			list:    module.Info.Actions,
			locType: "actions",
		},
		{
			src:     module.Locale.Events,
			list:    module.Info.Events,
			locType: "events",
		},
		{
			src:     module.Locale.Fields,
			list:    module.Info.Fields,
			locType: "fields",
		},
		{
			src:     module.Locale.Tags,
			list:    module.Info.Tags,
			locType: "tags",
		},
	}

	updateFieldsInSchema := func(sh *models.Type) {
		if len(sh.AllOf) >= 2 && sh.AllOf[0].Ref != "" {
			enum := make([]interface{}, 0)
			if len(module.Info.Fields) > 0 {
				for _, field := range module.Info.Fields {
					enum = append(enum, field)
				}
			} else {
				enum = append(enum, nil)
			}
			sh.AllOf[1].Properties["fields"] = &models.Type{
				Type: "array",
				Items: &models.Type{
					Type: "string",
					Enum: enum,
				},
				Default:  module.Info.Fields,
				MinItems: len(module.Info.Fields),
				MaxItems: len(module.Info.Fields),
			}
		}
	}

	getReplacesMap := func(keys []string) map[string]string {
		replaces := make(map[string]string)
		for _, key := range keys {
			if strings.Contains(key, "{{module_name}}") {
				replaces[key] = strings.ReplaceAll(key, "{{module_name}}", module.Info.Name)
			}
		}
		return replaces
	}

	currentTime := time.Now()
	patchLocaleCl := map[string]string{
		"ru": currentTime.Format("02.01.2006"),
		"en": currentTime.Format("01-02-2006"),
	}

	module.ActionConfigSchema.Required = module.Info.Actions
	if acsItems := module.ActionConfigSchema.Properties; len(acsItems) >= 1 {
		if acsItem, ok := acsItems[placeholder]; ok {
			delete(acsItems, placeholder)
			updateFieldsInSchema(acsItem)
			for _, actionID := range module.Info.Actions {
				acsItems[actionID] = acsItem
			}
		} else {
			return errors.New("failed to get action_config_schema placeholder")
		}
	} else {
		return errors.New("action_config_schema is invalid format")
	}

	if dacItems := module.DefaultActionConfig; len(dacItems) >= 1 {
		if dacItem, ok := dacItems[placeholder]; ok {
			delete(dacItems, placeholder)
			for _, actionID := range module.Info.Actions {
				dacItem.Fields = module.Info.Fields
				dacItems[actionID] = dacItem
			}
		} else {
			return errors.New("failed to get default_action_config placeholder")
		}
	} else {
		return errors.New("default_action_config is invalid format")
	}

	module.EventConfigSchema.Required = module.Info.Events
	if ecsItems := module.EventConfigSchema.Properties; len(ecsItems) >= 1 {
		keys := make([]string, 0, len(ecsItems))
		for ecsName := range ecsItems {
			keys = append(keys, ecsName)
		}
		replaces := getReplacesMap(keys)
		for old, new := range replaces {
			ecsItems[new] = ecsItems[old]
			delete(ecsItems, old)
			module.EventConfigSchema.Required = append(module.EventConfigSchema.Required, new)
		}
		if ecsItem, ok := ecsItems[placeholder]; ok {
			delete(ecsItems, placeholder)
			updateFieldsInSchema(ecsItem)
			for _, eventID := range module.Info.Events {
				ecsItems[eventID] = ecsItem
			}
		} else {
			return errors.New("failed to get event_config_schema placeholder")
		}
	} else {
		return errors.New("event_config_schema is invalid format")
	}

	if decItems := module.DefaultEventConfig; len(decItems) >= 1 {
		keys := make([]string, 0, len(decItems))
		for decName := range decItems {
			keys = append(keys, decName)
		}
		replaces := getReplacesMap(keys)
		for old, new := range replaces {
			decItems[new] = decItems[old]
			delete(decItems, old)
		}
		if decItem, ok := decItems[placeholder]; ok {
			delete(decItems, placeholder)
			for _, eventID := range module.Info.Events {
				decItem.Fields = module.Info.Fields
				decItems[eventID] = decItem
			}
		} else {
			return errors.New("failed to get default_event_config placeholder")
		}
	} else {
		return errors.New("default_event_config is invalid format")
	}

	if fsItems := module.FieldsSchema.Properties; len(fsItems) >= 1 {
		if fsItem, ok := fsItems[placeholder]; ok {
			delete(fsItems, placeholder)
			for _, fieldID := range module.Info.Fields {
				fsItems[fieldID] = fsItem
			}
		} else {
			return errors.New("failed to get fields_schema placeholder")
		}
	} else {
		return errors.New("fields_schema is invalid format")
	}

	if clItems := module.Changelog; len(clItems) >= 1 {
		if clItem, ok := clItems[placeholder]; ok {
			delete(clItems, placeholder)
			for lng, date := range patchLocaleCl {
				if clDesc, ok := clItem[lng]; ok {
					clDesc.Date = date
					clItem[lng] = clDesc
				}
			}
			clItems[module.Info.Version.String()] = clItem
		} else {
			return errors.New("failed to get changelog placeholder")
		}
	} else {
		return errors.New("changelog is invalid format")
	}

	for _, pLoc := range patchLocaleLstList {
		if locItems := pLoc.src; len(locItems) >= 1 {
			keys := make([]string, 0, len(locItems))
			for locName := range locItems {
				keys = append(keys, locName)
			}
			replaces := getReplacesMap(keys)
			for old, new := range replaces {
				locItems[new] = locItems[old]
				delete(locItems, old)
			}
			if locItem, ok := locItems[placeholder]; ok {
				delete(locItems, placeholder)
				for _, itemID := range pLoc.list {
					locItems[itemID] = locItem
				}
			} else {
				return errors.New("failed to get locale " + pLoc.locType + " config placeholder")
			}
		} else {
			return errors.New("locale " + pLoc.locType + " config is invalid format")
		}
	}

	for _, pLoc := range patchLocaleCfgList {
		if locItems := pLoc.src; len(locItems) >= 1 {
			keys := make([]string, 0, len(locItems))
			for locName := range locItems {
				keys = append(keys, locName)
			}
			replaces := getReplacesMap(keys)
			for old, new := range replaces {
				locItems[new] = locItems[old]
				delete(locItems, old)
			}
			if locItemEtl, ok := locItems[placeholder]; ok {
				delete(locItems, placeholder)
				for _, itemID := range pLoc.list {
					locItem := make(models.ModuleLocaleDesc)
					for _, lng := range languages {
						if itemDescEtl, ok := locItemEtl[lng]; ok {
							locItem[lng] = models.LocaleDesc{
								Title:       itemID,
								Description: itemDescEtl.Description,
							}
						}
					}
					locItems[itemID] = locItem
				}
			} else {
				return errors.New("failed to get locale " + pLoc.locType + " placeholder")
			}
			for locKey := range locItems {
				if !utils.StringInSlice(locKey, pLoc.list) {
					pLoc.list = append(pLoc.list, locKey)
				}
			}
			switch pLoc.locType {
			case "actions":
				module.Info.Actions = pLoc.list
			case "events":
				module.Info.Events = pLoc.list
			case "fields":
				module.Info.Fields = pLoc.list
			case "tags":
				module.Info.Tags = pLoc.list
			default:
				return errors.New("unknown locale type " + pLoc.locType)
			}
		} else {
			return errors.New("locale " + pLoc.locType + " is invalid format")
		}
	}

	for _, lng := range languages {
		if itemDescEtl, ok := module.Locale.Module[lng]; ok {
			module.Locale.Module[lng] = models.LocaleDesc{
				Title:       module.Info.Name + " " + itemDescEtl.Title,
				Description: itemDescEtl.Description,
			}
		}
	}

	return nil
}

func BuildModuleSConfig(module *models.ModuleS) (map[string][]byte, error) {
	files := make(map[string][]byte)
	targets := map[string]interface{}{
		"action_config_schema":  &module.ActionConfigSchema,
		"changelog":             &module.Changelog,
		"config_schema":         &module.ConfigSchema,
		"current_action_config": &module.DefaultActionConfig,
		"current_config":        &module.DefaultConfig,
		"current_event_config":  &module.DefaultEventConfig,
		"default_action_config": &module.DefaultActionConfig,
		"default_config":        &module.DefaultConfig,
		"default_event_config":  &module.DefaultEventConfig,
		"dynamic_dependencies":  &models.Dependencies{},
		"event_config_schema":   &module.EventConfigSchema,
		"fields_schema":         &module.FieldsSchema,
		"info":                  &module.Info,
		"locale":                &module.Locale,
		"static_dependencies":   &module.StaticDependencies,
		"secure_config_schema":  &module.SecureConfigSchema,
		"secure_default_config": &module.SecureDefaultConfig,
		"secure_current_config": &module.SecureDefaultConfig,
	}

	for filename, container := range targets {
		var (
			containerOut  bytes.Buffer
			containerData []byte
			err           error
		)
		if containerData, err = json.Marshal(container); err != nil {
			return nil, errors.New("failed marshal " + filename + ": " + err.Error())
		}

		if err = json.Indent(&containerOut, containerData, "", "    "); err != nil {
			return nil, errors.New("failed json.Indent " + filename + ": " + err.Error())
		}
		files[filename+".json"] = containerOut.Bytes()
	}

	return files, nil
}

func LoadModuleSTemplate(mi *models.ModuleInfo) (Template, *models.ModuleS, error) {
	fs, err := storage.NewFS()
	if err != nil {
		return nil, nil, errors.New("failed initialize FS driver: " + err.Error())
	}

	var module *models.ModuleS
	template := make(Template)
	loadModuleDir := func(dir string) (map[string][]byte, error) {
		templatesDir := "templates"
		if dir, ok := os.LookupEnv("TEMPLATES_DIR"); ok {
			templatesDir = dir
		}
		tpath := joinPath(templatesDir, mi.Template, dir)
		if fs.IsNotExist(tpath) {
			return nil, errors.New("template directory not found")
		}
		files, err := fs.ReadDirRec(tpath)
		if err != nil {
			return nil, errors.New("failed to read template files: " + err.Error())
		}

		return removeLeadSlash(files), nil
	}

	for _, dir := range []string{"bmodule", "cmodule", "smodule"} {
		if files, err := loadModuleDir(dir); err != nil {
			return nil, nil, err
		} else {
			template[dir] = files
		}
	}

	if files, err := loadModuleDir("config"); err != nil {
		return nil, nil, err
	} else {
		if module, err = LoadModuleSConfig(files); err != nil {
			return nil, nil, err
		}
		module.Info = *mi
		if err = PatchModuleSConfig(module); err != nil {
			return nil, nil, err
		}
		if cfiles, err := BuildModuleSConfig(module); err != nil {
			return nil, nil, err
		} else {
			template["config"] = cfiles
		}
	}

	return template, module, nil
}

func LoadModuleSFromGlobalS3(mi *models.ModuleInfo) (Template, error) {
	s3, err := storage.NewS3(nil)
	if err != nil {
		return nil, errors.New("failed to initialize FS driver: " + err.Error())
	}

	template := make(Template)
	loadModuleDir := func(dir string) (map[string][]byte, error) {
		vpath := joinPath(mi.Name, mi.Version.String(), dir)
		if s3.IsNotExist(vpath) {
			return nil, errors.New("module to version directory not found")
		}
		files, err := s3.ReadDirRec(vpath)
		if err != nil {
			return nil, errors.New("failed to read module version files: " + err.Error())
		}

		return removeLeadSlash(files), nil
	}

	for _, dir := range []string{"bmodule", "cmodule", "smodule"} {
		if files, err := loadModuleDir(dir); err != nil {
			return nil, err
		} else {
			template[dir] = files
		}
	}

	return template, nil
}

func StoreModuleSToGlobalS3(mi *models.ModuleInfo, mf Template) error {
	s3, err := storage.NewS3(nil)
	if err != nil {
		return errors.New("failed initialize S3 driver: " + err.Error())
	}

	for _, dir := range []string{"bmodule", "cmodule", "smodule", "config"} {
		for fpath, fdata := range mf[dir] {
			if err := s3.WriteFile(joinPath(mi.Name, mi.Version.String(), dir, fpath), fdata); err != nil {
				return errors.New("failed to write file to S3: " + err.Error())
			}
		}
	}

	return nil
}

func StoreCleanModuleSToGlobalS3(mi *models.ModuleInfo, mf Template) error {
	s3, err := storage.NewS3(nil)
	if err != nil {
		return errors.New("failed initialize S3 driver: " + err.Error())
	}

	for _, dir := range []string{"bmodule", "cmodule", "smodule", "config"} {
		files, _ := s3.ListDirRec(joinPath(mi.Name, mi.Version.String(), dir))
		for fpath, fdata := range mf[dir] {
			if err := s3.WriteFile(joinPath(mi.Name, mi.Version.String(), dir, fpath), fdata); err != nil {
				return errors.New("failed to write file to S3: " + err.Error())
			}
		}
		if files == nil {
			continue
		}
		for fpath, finfo := range files {
			fpath = strings.TrimPrefix(fpath, "/")
			if finfo.IsDir() || fpath == "" {
				continue
			}
			if _, ok := mf[dir][fpath]; ok {
				continue
			}
			if err := s3.RemoveFile(joinPath(mi.Name, mi.Version.String(), dir, fpath)); err != nil {
				return errors.New("failed to remove unused file from S3: " + err.Error())
			}
		}
	}

	return nil
}

func CopyModuleAFilesToInstanceS3(mi *models.ModuleInfo, sv *models.Service) error {
	gS3, err := storage.NewS3(nil)
	if err != nil {
		return errors.New("failed to initialize global S3 driver: " + err.Error())
	}

	mfiles, err := gS3.ReadDirRec(joinPath(mi.Name, mi.Version.String()))
	if err != nil {
		return errors.New("failed to read system module files: " + err.Error())
	}

	fillMissingFileContents("/config", mfiles)

	ufiles, err := gS3.ReadDirRec("utils")
	if err != nil {
		return errors.New("failed to read utils files: " + err.Error())
	}

	iS3, err := storage.NewS3(sv.Info.S3.ToS3ConnParams())
	if err != nil {
		return errors.New("failed to initialize instance S3 driver: " + err.Error())
	}

	if iS3.RemoveDir(joinPath(mi.Name, mi.Version.String())); err != nil {
		return errors.New("failed to remove module directory from instance S3: " + err.Error())
	}

	for fpath, fdata := range mfiles {
		if err := iS3.WriteFile(joinPath(mi.Name, mi.Version.String(), fpath), fdata); err != nil {
			return errors.New("failed to write system module file to S3: " + err.Error())
		}
	}

	for fpath, fdata := range ufiles {
		if err := iS3.WriteFile(joinPath("utils", fpath), fdata); err != nil {
			return errors.New("failed to write utils file to S3: " + err.Error())
		}
	}

	return nil
}

func fillMissingFileContents(path string, files map[string][]byte) {
	const emptyConfigFileContent = "{}"
	lackFiles := []string{
		joinPath(path, "secure_config_schema.json"),
		joinPath(path, "secure_default_config.json"),
		joinPath(path, "secure_current_config.json"),
	}

	for _, file := range lackFiles {
		if _, ok := files[file]; !ok {
			files[file] = []byte(emptyConfigFileContent)
		}
	}
}

func CheckMultipleModulesDuplicate(iDB *gorm.DB, moduleNames []string, policyId uint64) (map[string]struct{}, error) {
	var pgs models.PolicyGroups
	if err := iDB.Take(&pgs, "id = ?", policyId).Error; err != nil {
		return nil, errors.New("failed to get module policy")
	}
	if err := iDB.Model(pgs).Association("groups").Find(&pgs.Groups).Error; err != nil {
		return nil, errors.New("failed to get linked groups to module policy")
	}

	pidsMap := make(map[uint64]struct{})
	for _, group := range pgs.Groups {
		gps := models.GroupPolicies{Group: group}
		if err := iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
			return nil, errors.New("failed to get linked policies to group")
		}
		for _, p := range gps.Policies {
			pidsMap[p.ID] = struct{}{}
		}
	}
	var pids []uint64
	for k := range pidsMap {
		pids = append(pids, k)
	}

	rows, err := iDB.
		Table((&models.ModuleA{}).TableName()).
		Select("name").
		Where("deleted_at IS NULL").
		Where("policy_id IN (?) AND name IN (?) AND status = 'joined'", pids, moduleNames).
		Group("name").Rows()
	if err != nil {
		return nil, errors.New("failed to merge modules")
	}

	moduleNamesMap := make(map[string]struct{})
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, errors.New("failed to scan module name")
		}
		moduleNamesMap[name] = struct{}{}
	}

	if err := rows.Err(); err != nil {
		return nil, errors.New("failed to scan module name")
	}

	return moduleNamesMap, nil
}

func CheckModulesDuplicate(iDB *gorm.DB, ma *models.ModuleA) error {
	var pgs models.PolicyGroups
	if err := iDB.Take(&pgs, "id = ?", ma.PolicyID).Error; err != nil {
		return errors.New("failed to get module policy")
	}
	if err := iDB.Model(pgs).Association("groups").Find(&pgs.Groups).Error; err != nil {
		return errors.New("failed to get linked groups to module policy")
	}

	for _, group := range pgs.Groups {
		gps := models.GroupPolicies{Group: group}
		if err := iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
			return errors.New("failed to get linked policies to group")
		}

		var pids []uint64
		for _, p := range gps.Policies {
			pids = append(pids, p.ID)
		}

		var cnts []int64
		findDupsQuery := iDB.
			Table((&models.ModuleA{}).TableName()).
			Select("count(*) AS cnt").
			Where("deleted_at IS NULL").
			Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, ma.Info.Name).
			Group("name").
			Find(&cnts)
		if err := findDupsQuery.Error; err != nil {
			return errors.New("failed to merge modules")
		}

		if len(cnts) != 0 {
			return errors.New("found duplicate modules")
		}
	}

	return nil
}

func removeUnusedModuleVersion(c *gin.Context, iDB *gorm.DB, name, version string, sv *models.Service) error {
	var count int64
	err := iDB.
		Model(&models.ModuleA{}).
		Where("name LIKE ? AND version LIKE ?", name, version).
		Count(&count).Error
	if err != nil {
		return errors.New("failed to get count modules by version")
	}

	if count == 0 {
		s3, err := storage.NewS3(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
			return err
		}

		if err = s3.RemoveDir(name + "/" + version + "/"); err != nil && err.Error() != "not found" {
			utils.FromContext(c).WithError(err).Errorf("error removing module data from s3")
			return err
		}
	}

	return nil
}

func updateDependenciesWhenModuleRemove(c *gin.Context, iDB *gorm.DB, name string) error {
	var (
		err     error
		modules []models.ModuleA
		incl    = []interface{}{"current_event_config", "dynamic_dependencies"}
	)

	if err = iDB.Find(&modules, "dependencies LIKE ?", `%"`+name+`"%`).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding modules by dependencies")
		return err
	}

	for _, m := range modules {
		curevs := make(models.EventConfig)
		for eid, ev := range m.CurrentEventConfig {
			actions := make([]models.EventConfigAction, 0)
			for _, act := range ev.Actions {
				if act.ModuleName != name {
					actions = append(actions, act)
				}
			}
			ev.Actions = actions
			curevs[eid] = ev
		}
		m.CurrentEventConfig = curevs

		ddeps := make(models.Dependencies, 0)
		for _, dep := range m.DynamicDependencies {
			if dep.ModuleName != name {
				ddeps = append(ddeps, dep)
			}
		}
		m.DynamicDependencies = ddeps

		if err = iDB.Select("", incl...).Save(&m).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error updating config module")
			return err
		}
	}

	return nil
}

func updatePolicyModulesByModuleS(c *gin.Context, moduleS *models.ModuleS, sv *models.Service) error {
	iDB := utils.GetDB(sv.Info.DB.User, sv.Info.DB.Pass, sv.Info.DB.Host,
		strconv.Itoa(int(sv.Info.DB.Port)), sv.Info.DB.Name)
	if iDB == nil {
		utils.FromContext(c).WithError(nil).Errorf("error openning connection to instance DB")
		return errors.New("failed to connect to instance DB")
	}
	defer iDB.Close()

	encryptor := getDBEncryptor(c)
	if encryptor == nil {
		utils.FromContext(c).WithError(nil).Errorf("encryptor not found")
		return errors.New("encryptor not found")
	}

	var modules []models.ModuleA
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name LIKE ? AND version LIKE ? AND last_module_update NOT LIKE ?",
			moduleS.Info.Name, moduleS.Info.Version.String(), moduleS.LastUpdate)
	}
	if err := iDB.Scopes(scope).Find(&modules).Error; err != nil {
		utils.FromContext(c).WithError(err).
			Errorf("error finding policy modules by name and version '%s' '%s'",
				moduleS.Info.Name, moduleS.Info.Version.String())
		return err
	} else if len(modules) == 0 {
		return nil
	}

	if err := CopyModuleAFilesToInstanceS3(&moduleS.Info, sv); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error copying module files to S3")
		return err
	}

	excl := []string{"policy_id", "status", "join_date", "last_update"}
	for _, moduleA := range modules {
		var err error
		moduleA, err = MergeModuleAConfigFromModuleS(&moduleA, moduleS, encryptor)
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("invalid module state")
			return err
		}

		if err := moduleA.Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("invalid module state")
			return err
		}

		err = moduleA.EncryptSecureParameters(encryptor)
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
			return fmt.Errorf("failed to encrypt module secure config: %w", err)
		}
		if err := iDB.Omit(excl...).Save(&moduleA).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error updating module")
			return err
		}
	}

	return nil
}

// helper method to return fully copy of object json schema and following modification
func copySchema(sht *models.Type, defs models.Definitions) models.Schema {
	rsh := models.Schema{}
	copier.Copy(&rsh.Type, sht)
	rsh.Definitions = defs
	return rsh
}

// this method is need to convert object to the golang types such as map, slice, etc.
func convertToRawInterface(iv interface{}) interface{} {
	var ir interface{}
	if b, err := json.Marshal(iv); err != nil {
		return nil
	} else if err = json.Unmarshal(b, &ir); err != nil {
		return nil
	}
	return ir
}

// this method receives current event actions list and event fields and does filter the list by fields
// all received fields for action must return from current event according to its configuration
func filterEventActionsListByFields(acts []models.EventConfigAction, fields []string) []models.EventConfigAction {
	racts := []models.EventConfigAction{}
	for _, act := range acts {
		if utils.StringsInSlice(act.Fields, fields) {
			racts = append(racts, act)
		}
	}
	return racts
}

// this method is removing new keys from current map and adding not existing ones from default
// args:
//
//	curMap is current document of config keys which user modified early and which we tried to keep
//	defMap is default document of config keys which we are using as a reference data structure
func clearMapKeysList(curMap, defMap map[string]interface{}) map[string]interface{} {
	resMap := make(map[string]interface{}, len(defMap))
	for k, v2 := range defMap {
		if v1, ok := curMap[k]; ok {
			resMap[k] = v1
		} else {
			resMap[k] = v2
		}
	}
	return resMap
}

// args:
//
//	cc is Current module Config from old module version which we tried to keep
//	dc is Default module Config from actual module version which we are using as a reference
//	sh is JSON Schema structure from actual module version which wa are using to check result document
func mergeModuleACurrentConfig(cc, dc models.ModuleConfig, sh models.Schema) models.ModuleConfig {
	// add new config values from default
	for cik, civ := range dc {
		if _, ok := cc[cik]; !ok {
			cc[cik] = civ
		}
	}

	mcsh := copySchema(&sh.Type, sh.Definitions)
	icc := utils.MergeTwoInterfacesBySchema(cc, dc, mcsh)
	if res, err := mcsh.ValidateGo(icc); err != nil || !res.Valid() {
		return dc
	} else if rcc, ok := icc.(models.ModuleConfig); !ok {
		return dc
	} else {
		return rcc
	}
}

// args:
//
//	cc is Current module SecureConfig from old module version which we tried to keep
//	dc is Default module SecureConfig from actual module version which we are using as a reference
//	sh is JSON Schema structure from actual module version which wa are using to check result document
func mergeModuleASecureCurrentConfig(cc, dc models.ModuleSecureConfig, sh models.Schema) models.ModuleSecureConfig {
	// add new config values from default
	for k, v := range dc {
		if _, ok := cc[k]; !ok {
			cc[k] = v
		}
	}

	mcsh := copySchema(&sh.Type, sh.Definitions)
	icc := utils.MergeTwoInterfacesBySchema(cc, dc, mcsh)
	if res, err := mcsh.ValidateGo(icc); err != nil || !res.Valid() {
		return dc
	} else if rcc, ok := icc.(models.ModuleSecureConfig); !ok {
		return dc
	} else {
		return rcc
	}
}

// args:
//
//	cac is Current Action Config from old module version which we tried to keep
//	dac is Default Action Config from actual module version which we are using as a reference
//	sh is JSON Schema structure from actual module version which wa are using to check result document
func mergeModuleACurrentActionConfig(cac, dac models.ActionConfig, sh models.Schema) models.ActionConfig {
	for acn, daci := range dac {
		if caci, ok := cac[acn]; ok {
			caci.Fields = daci.Fields
			caci.Priority = daci.Priority
			caci.Config = clearMapKeysList(caci.Config, daci.Config)
			cac[acn] = caci
		} else {
			cac[acn] = daci
		}
	}
	for acn := range cac {
		if _, ok := dac[acn]; !ok {
			delete(cac, acn)
		}
	}

	rcac := models.ActionConfig{}
	acsh := copySchema(&sh.Type, models.GetACSDefinitions(sh.Definitions))
	icac := utils.MergeTwoInterfacesBySchema(convertToRawInterface(cac), convertToRawInterface(dac), acsh)
	if res, err := acsh.ValidateGo(icac); err != nil || !res.Valid() {
		return dac
	} else if b, err := json.Marshal(icac); err != nil {
		return dac
	} else if err = json.Unmarshal(b, &rcac); err != nil {
		return dac
	}
	return rcac
}

// args:
//
//	ceci is Current Event Config Item from old module version which we tried to keep
//	deci is Default Event Config Item from actual module version which we are using as a reference
//	sh is JSON Schema structure from actual module version which wa are using to check result document
func mergeModuleAEventConfigItem(ceci, deci models.EventConfigItem, sh models.Schema) models.EventConfigItem {
	reci := models.EventConfigItem{}
	iceci, ideci := convertToRawInterface(ceci), convertToRawInterface(deci)
	rieci := utils.MergeTwoInterfacesBySchema(iceci, ideci, sh)
	if b, err := json.Marshal(rieci); err != nil {
		return deci
	} else if err = json.Unmarshal(b, &reci); err != nil {
		return deci
	}
	return reci
}

// args:
//
//	cec is Current Event Config from old module version which we tried to keep
//	dec is Default Event Config from actual module version which we are using as a reference
//	sh is JSON Schema structure from actual module version which wa are using to check result document
func mergeModuleACurrentEventConfig(cec, dec models.EventConfig, sh models.Schema) models.EventConfig {
	ecsh := copySchema(&sh.Type, models.GetECSDefinitions(sh.Definitions))
	for ecn, deci := range dec {
		if ceci, ok := cec[ecn]; ok {
			ceci.Fields = deci.Fields
			ceci.Actions = filterEventActionsListByFields(ceci.Actions, ceci.Fields)
			ceci.Config = clearMapKeysList(ceci.Config, deci.Config)
			if sht, ok := ecsh.Properties[ecn]; ok {
				ecish := copySchema(sht, ecsh.Definitions)
				cec[ecn] = mergeModuleAEventConfigItem(ceci, deci, ecish)
			} else {
				cec[ecn] = ceci
			}
		} else {
			cec[ecn] = deci
		}
	}
	for ecn := range cec {
		if _, ok := dec[ecn]; !ok {
			delete(cec, ecn)
		}
	}

	if res, err := ecsh.ValidateGo(cec); err != nil || !res.Valid() {
		return dec
	}
	return cec
}

// args:
//
//	dd is Dynamic Dependencies from old module version which we tried to keep
//	cec is Current Event Config which was got after merging to default (result Current Event Config)
func clearModuleADynamicDependencies(dd models.Dependencies, cec models.EventConfig) models.Dependencies {
	rdd := models.Dependencies{}
	checkDepInActions := func(ec models.EventConfigItem, moduleName string) bool {
		for _, act := range ec.Actions {
			if act.ModuleName == moduleName {
				return true
			}
		}
		return false
	}
	for _, d := range dd {
		for _, ec := range cec {
			if checkDepInActions(ec, d.ModuleName) {
				rdd = append(rdd, d)
				break
			}
		}
	}
	return rdd
}

func MergeModuleAConfigFromModuleS(moduleA *models.ModuleA, moduleS *models.ModuleS, encryptor crypto.IDBConfigEncryptor) (models.ModuleA, error) {
	err := moduleA.DecryptSecureParameters(encryptor)
	if err != nil {
		return models.ModuleA{}, err
	}
	err = moduleS.DecryptSecureParameters(encryptor)
	if err != nil {
		return models.ModuleA{}, err
	}

	moduleR := moduleS.ToModuleA()

	// restore original key properties
	moduleR.ID = moduleA.ID
	moduleR.PolicyID = moduleA.PolicyID
	moduleR.Status = moduleA.Status
	moduleR.JoinDate = moduleA.JoinDate
	moduleR.LastUpdate = moduleA.LastUpdate

	// merge current and default config
	moduleR.CurrentConfig = mergeModuleACurrentConfig(
		moduleA.CurrentConfig, moduleS.DefaultConfig, moduleS.ConfigSchema,
	)
	moduleR.SecureCurrentConfig = mergeModuleASecureCurrentConfig(
		moduleA.SecureCurrentConfig, moduleS.SecureDefaultConfig, moduleS.SecureConfigSchema,
	)
	moduleR.CurrentActionConfig = mergeModuleACurrentActionConfig(
		moduleA.CurrentActionConfig, moduleS.DefaultActionConfig, moduleS.ActionConfigSchema,
	)
	moduleR.CurrentEventConfig = mergeModuleACurrentEventConfig(
		moduleA.CurrentEventConfig, moduleS.DefaultEventConfig, moduleS.EventConfigSchema,
	)
	moduleR.DynamicDependencies = clearModuleADynamicDependencies(
		moduleA.DynamicDependencies, moduleR.CurrentEventConfig,
	)

	return moduleR, nil
}

// cleanupSecureConfig removes serverOnly parameters from the agent module config
func cleanupSecureConfig(module *models.ModuleA) {
	for k, v := range module.SecureDefaultConfig {
		if serverOnly := v.ServerOnly; serverOnly != nil && *serverOnly {
			module.SecureDefaultConfig[k] = models.ModuleSecureParameter{}
			module.SecureCurrentConfig[k] = models.ModuleSecureParameter{}
		}
	}
}

func LatestModulesQuery(db *gorm.DB) *gorm.DB {
	subQueryLatestVersion := db.Table("modules m2").
		Select("m2.version").
		Where("m2.name LIKE mname").
		Order("m2.ver_major DESC, m2.ver_minor DESC, m2.ver_patch DESC").
		Limit(1).
		SubQuery()
	subQueryModulesList := db.Table("modules m1").
		Select("m1.name AS mname, ? AS mversion", subQueryLatestVersion).
		Group("m1.name").
		SubQuery()
	return db.Select("`modules`.*").
		Joins("INNER JOIN ? AS ml ON name LIKE `ml`.mname AND version LIKE `ml`.mversion", subQueryModulesList)
}

func FilterModulesByVersion(version string) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		switch version {
		case "all":
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC")
		case "latest":
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC").Limit(1)
		default:
			return db.Where("version LIKE ?", version).Limit(1)
		}
	}
}

func getModuleName(c *gin.Context, name string, version string) (string, error) {
	gDB := utils.GetGormDB(c, "gDB")
	if gDB == nil {
		return "", errors.New("can't connect to database")
	}

	sv := getService(c)
	if sv == nil {
		return "", errors.New("can't get service")
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", name, tid, sv.Type)
	}

	var module models.ModuleS
	if err := gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		return "", err
	}
	return module.Locale.Module["en"].Title, nil
}

// GetAgentModules is a function to return agent module list view on dashboard
// @Summary Retrieve agent modules by agent hash and by filters
// @Tags Agents,Modules
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=agentModules} "agent modules received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting agent modules not permitted"
// @Failure 404 {object} utils.errorResp "agent or modules not found"
// @Failure 500 {object} utils.errorResp "internal error on getting agent modules"
// @Router /agents/{hash}/modules [get]
func GetAgentModules(c *gin.Context) {
	var (
		err   error
		gDB   *gorm.DB
		hash  = c.Param("hash")
		iDB   *gorm.DB
		pids  []uint64
		query utils.TableQuery
		resp  agentModules
		sv    *models.Service
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrModulesInvalidRequest, err)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	var agentPolicies models.AgentPolicies
	if err = iDB.Take(&agentPolicies, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrGetAgentModulesAgentNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	}
	if err = iDB.Model(agentPolicies).Association("policies").Find(&agentPolicies.Policies).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent policies by agent model")
		utils.HTTPError(c, srverrors.ErrGetAgentModulesAgentPoliciesNotFound, err)
		return
	}
	if err = agentPolicies.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent policies data '%s'", agentPolicies.Hash)
		utils.HTTPError(c, srverrors.ErrGetAgentModulesInvalidAgentPoliciesData, err)
		return
	}

	for _, p := range agentPolicies.Policies {
		pids = append(pids, p.ID)
	}

	query.Init("modules", modulesSQLMappers)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("policy_id IN (?) AND status LIKE 'joined'", pids)
		},
		func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("name asc")
		},
	})
	if resp.Total, err = query.Query(iDB, &resp.Modules); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent modules")
		utils.HTTPError(c, srverrors.ErrGetAgentModulesInvalidQuery, err)
		return
	}

	for i := 0; i < len(resp.Modules); i++ {
		if err = resp.Modules[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).
				Errorf("error validating agent module data '%s'", resp.Modules[i].Info.Name)
			utils.HTTPError(c, srverrors.ErrGetAgentModulesInvalidAgentData, err)
			return
		}
	}

	modNames := []string{""}
	for _, module := range resp.Modules {
		modNames = append(modNames, module.Info.Name)
	}
	modules := make([]models.ModuleS, 0)
	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return LatestModulesQuery(db).Where("name IN (?) AND tenant_id = ? AND service_type = ?", modNames, tid, sv.Type)
	}

	if err = gDB.Scopes(scope).Find(&modules).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system modules list by names")
		utils.HTTPError(c, srverrors.ErrGetAgentsGetSystemModulesFail, err)
		return
	}

	var details []agentModuleDetails
	if err = iDB.Raw(sqlAgentModuleDetails, agentPolicies.ID).Scan(&details).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading agents modules details")
		utils.HTTPError(c, srverrors.ErrGetAgentModulesDetailsNotFound, err)

		return
	}
	for _, ma := range resp.Modules {
		rmd := agentModuleDetails{Name: ma.Info.Name}
		for _, md := range details {
			if md.Name == ma.Info.Name {
				rmd = md
				break
			}
		}
		for _, pd := range agentPolicies.Policies {
			if pd.ID == ma.PolicyID {
				rmd.Policy = pd
			}
		}
		for _, ms := range modules {
			if ms.Info.Name == ma.Info.Name {
				rmd.Update = ma.Info.Version.String() != ms.Info.Version.String()
				break
			}
		}
		resp.Details = append(resp.Details, rmd)
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// GetAgentModule is a function to return agent module by name
// @Summary Retrieve agent module data by agent hash and module name
// @Tags Agents,Modules
// @Produce json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} utils.successResp{data=models.ModuleA} "agent module data received successful"
// @Failure 403 {object} utils.errorResp "getting agent module data not permitted"
// @Failure 404 {object} utils.errorResp "agent or module not found"
// @Failure 500 {object} utils.errorResp "internal error on getting agent module"
// @Router /agents/{hash}/modules/{module_name} [get]
func GetAgentModule(c *gin.Context) {
	var (
		err        error
		hash       = c.Param("hash")
		iDB        *gorm.DB
		module     models.ModuleA
		moduleName = c.Param("module_name")
		pids       []uint64
	)

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	var agentPolicies models.AgentPolicies
	if err = iDB.Take(&agentPolicies, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrGetAgentModuleAgentNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	}
	if err = iDB.Model(agentPolicies).Association("policies").Find(&agentPolicies.Policies).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent policies by agent model")
		utils.HTTPError(c, srverrors.ErrGetAgentModuleAgentPoliceNotFound, err)
		return
	}
	if err = agentPolicies.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent policies data '%s'", agentPolicies.Hash)
		utils.HTTPError(c, srverrors.ErrGetAgentModuleInvalidAgentPoliceData, err)
		return
	}

	for _, p := range agentPolicies.Policies {
		pids = append(pids, p.ID)
	}

	scopeModule := func(db *gorm.DB) *gorm.DB {
		return db.Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, moduleName)
	}
	if err = iDB.Scopes(scopeModule).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy module by module name")
		utils.HTTPError(c, srverrors.ErrModulesNotFound, err)
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidData, err)
		return
	}

	utils.HTTPSuccess(c, http.StatusOK, module)
}

// GetAgentBModule is a function to return bmodule vue code as a file
// @Summary Retrieve browser module vue code by agent hash and module name
// @Tags Agents,Modules
// @Produce text/javascript,application/javascript,json
// @Param hash path string true "agent hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param file query string false "path to the browser module file" default(main.vue)
// @Success 200 {file} file "browser module vue code as a file"
// @Failure 403 {object} utils.errorResp "getting agent module data not permitted"
// @Router /agents/{hash}/modules/{module_name}/bmodule.vue [get]
func GetAgentBModule(c *gin.Context) {
	var (
		err        error
		data       []byte
		filepath   = path.Join("/", c.DefaultQuery("file", "main.vue"))
		hash       = c.Param("hash")
		iDB        *gorm.DB
		module     models.ModuleA
		moduleName = c.Param("module_name")
		pids       []uint64
		s3         storage.IStorage
		sv         *models.Service
	)

	defer func() {
		ctype := mime.TypeByExtension(path.Ext(filepath))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(filepath)))
		c.Data(http.StatusOK, ctype, data)
	}()

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		return
	}

	if sv = getService(c); sv == nil {
		return
	}

	var agentPolicies models.AgentPolicies
	if err = iDB.Take(&agentPolicies, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent by hash")
		return
	}
	if err = iDB.Model(agentPolicies).Association("policies").Find(&agentPolicies.Policies).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding agent policies by agent model")
		return
	}
	if err = agentPolicies.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating agent policies data '%s'", agentPolicies.Hash)
		utils.HTTPError(c, srverrors.ErrGetAgentBModuleInvalidAgentPoliceData, err)
		return
	}

	for _, p := range agentPolicies.Policies {
		pids = append(pids, p.ID)
	}

	scopeModule := func(db *gorm.DB) *gorm.DB {
		return db.Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, moduleName)
	}
	if err = iDB.Scopes(scopeModule).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy module by module name")
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidData, err)
		return
	}

	s3, err = storage.NewS3(sv.Info.S3.ToS3ConnParams())
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		return
	}

	path := path.Join(moduleName, module.Info.Version.String(), "bmodule", filepath)
	if data, err = s3.ReadFile(path); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error reading module file '%s'", path)
		return
	}
}

// GetGroupModules is a function to return group module list view on dashboard
// @Summary Retrieve group modules by group hash and by filters
// @Tags Groups,Modules
// @Produce json
// @Param hash path string true "group hash in hex format (md5)" minlength(32) maxlength(32)
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=groupModules} "group modules received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting group modules not permitted"
// @Failure 404 {object} utils.errorResp "group or modules not found"
// @Failure 500 {object} utils.errorResp "internal error on getting group modules"
// @Router /groups/{hash}/modules [get]
func GetGroupModules(c *gin.Context) {
	var (
		err   error
		gDB   *gorm.DB
		gps   models.GroupPolicies
		hash  = c.Param("hash")
		iDB   *gorm.DB
		pids  []uint64
		query utils.TableQuery
		resp  groupModules
		sv    *models.Service
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrModulesInvalidRequest, err)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	if err = iDB.Take(&gps, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrGetGroupModulesGroupNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	}
	if err = iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group policies by group model")
		utils.HTTPError(c, srverrors.ErrGetGroupModulesGroupPoliciesNotFound, err)
		return
	}
	if err = gps.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating group policies data '%s'", gps.Hash)
		utils.HTTPError(c, srverrors.ErrGetGroupModulesInvalidGroupPoliciesData, err)
		return
	}

	for _, p := range gps.Policies {
		pids = append(pids, p.ID)
	}

	query.Init("modules", modulesSQLMappers)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("policy_id IN (?) AND status LIKE 'joined'", pids)
		},
		func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("name asc")
		},
	})
	if resp.Total, err = query.Query(iDB, &resp.Modules); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group modules")
		utils.HTTPError(c, srverrors.ErrGetGroupModulesInvalidGroupQuery, err)
		return
	}

	for i := 0; i < len(resp.Modules); i++ {
		if err = resp.Modules[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).
				Errorf("error validating group module data '%s'",
					resp.Modules[i].Info.Name)
			utils.HTTPError(c, srverrors.ErrGetGroupModulesInvalidGroupData, err)
			return
		}
	}

	modNames := []string{""}
	for _, module := range resp.Modules {
		modNames = append(modNames, module.Info.Name)
	}
	modules := make([]models.ModuleS, 0)
	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return LatestModulesQuery(db).Where("name IN (?) AND tenant_id = ? AND service_type = ?", modNames, tid, sv.Type)
	}

	if err = gDB.Scopes(scope).Find(&modules).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system modules list by names")
		utils.HTTPError(c, srverrors.ErrGetGroupsGetSystemModulesFail, err)
		return
	}

	var details []groupModuleDetails
	if err = iDB.Raw(sqlGroupModuleDetails, gps.ID).Scan(&details).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading group modules details")
		utils.HTTPError(c, srverrors.ErrGetGroupModulesDetailsNotFound, err)
		return
	}
	for _, ma := range resp.Modules {
		rmd := groupModuleDetails{Name: ma.Info.Name}
		for _, md := range details {
			if md.Name == ma.Info.Name {
				rmd = md
				break
			}
		}
		for _, pd := range gps.Policies {
			if pd.ID == ma.PolicyID {
				rmd.Policy = pd
			}
		}
		for _, ms := range modules {
			if ms.Info.Name == ma.Info.Name {
				rmd.Update = ma.Info.Version.String() != ms.Info.Version.String()
				break
			}
		}
		resp.Details = append(resp.Details, rmd)
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// GetGroupModule is a function to return group module by name
// @Summary Retrieve group module data by group hash and module name
// @Tags Groups,Modules
// @Produce json
// @Param hash path string true "group hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} utils.successResp{data=models.ModuleA} "group module data received successful"
// @Failure 403 {object} utils.errorResp "getting group module data not permitted"
// @Failure 404 {object} utils.errorResp "group or module not found"
// @Failure 500 {object} utils.errorResp "internal error on getting group"
// @Router /groups/{hash}/modules/{module_name} [get]
func GetGroupModule(c *gin.Context) {
	var (
		err        error
		gps        models.GroupPolicies
		hash       = c.Param("hash")
		iDB        *gorm.DB
		module     models.ModuleA
		moduleName = c.Param("module_name")
		pids       []uint64
	)

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if err = iDB.Take(&gps, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrGetGroupModuleGroupNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	}
	if err = iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group policies by group model")
		utils.HTTPError(c, srverrors.ErrGetGroupModuleGroupPoliciesNotFound, err)
		return
	}
	if err = gps.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating group policies data '%s'", gps.Hash)
		utils.HTTPError(c, srverrors.ErrGetGroupModuleInvalidGroupPoliciesData, err)
		return
	}

	for _, p := range gps.Policies {
		pids = append(pids, p.ID)
	}

	scopeModule := func(db *gorm.DB) *gorm.DB {
		return db.Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, moduleName)
	}
	if err = iDB.Scopes(scopeModule).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy module by module name")
		utils.HTTPError(c, srverrors.ErrModulesNotFound, err)

		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidData, err)
		return
	}

	utils.HTTPSuccess(c, http.StatusOK, module)
}

// GetGroupBModule is a function to return bmodule vue code as a file
// @Summary Retrieve browser module vue code by group hash and module name
// @Tags Groups,Modules
// @Produce text/javascript,application/javascript,json
// @Param hash path string true "group hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param file query string false "path to the browser module file" default(main.vue)
// @Success 200 {file} file "browser module vue code as a file"
// @Failure 403 {object} utils.errorResp "getting group module data not permitted"
// @Router /groups/{hash}/modules/{module_name}/bmodule.vue [get]
func GetGroupBModule(c *gin.Context) {
	var (
		err        error
		data       []byte
		filepath   = path.Join("/", c.DefaultQuery("file", "main.vue"))
		gps        models.GroupPolicies
		hash       = c.Param("hash")
		iDB        *gorm.DB
		module     models.ModuleA
		moduleName = c.Param("module_name")
		pids       []uint64
		s3         storage.IStorage
		sv         *models.Service
	)

	defer func() {
		ctype := mime.TypeByExtension(path.Ext(filepath))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(filepath)))
		c.Data(http.StatusOK, ctype, data)
	}()

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		return
	}

	if sv = getService(c); sv == nil {
		return
	}

	if err = iDB.Take(&gps, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group by hash")
		return
	}
	if err = iDB.Model(gps).Association("policies").Find(&gps.Policies).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding group policies by group model")
		return
	}
	if err = gps.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating group policies data '%s'", gps.Hash)
		utils.HTTPError(c, srverrors.ErrGetGroupBModuleInvalidGroupPoliciesData, err)
		return
	}

	for _, p := range gps.Policies {
		pids = append(pids, p.ID)
	}

	scopeModule := func(db *gorm.DB) *gorm.DB {
		return db.Where("policy_id IN (?) AND name = ? AND status = 'joined'", pids, moduleName)
	}
	if err = iDB.Scopes(scopeModule).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy module by module name")
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidData, err)
		return
	}

	s3, err = storage.NewS3(sv.Info.S3.ToS3ConnParams())
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		return
	}

	path := path.Join(moduleName, module.Info.Version.String(), "bmodule", filepath)
	if data, err = s3.ReadFile(path); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error reading module file '%s'", path)
		return
	}
}

// GetPolicyModules is a function to return policy module list view on dashboard
// @Summary Retrieve policy modules by policy hash and by filters
// @Tags Policies,Modules
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=policyModules} "policy modules received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data
// @Failure 403 {object} utils.errorResp "getting policy modules not permitted"
// @Failure 404 {object} utils.errorResp "policy or modules not found"
// @Failure 500 {object} utils.errorResp "internal error on getting policy modules"
// @Router /policies/{hash}/modules [get]
func GetPolicyModules(c *gin.Context) {
	var (
		err      error
		hash     = c.Param("hash")
		gDB      *gorm.DB
		iDB      *gorm.DB
		modulesA []models.ModuleA
		modulesS []models.ModuleS
		policy   models.Policy
		query    utils.TableQuery
		resp     policyModules
		sv       *models.Service
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.HTTPError(c, srverrors.ErrModulesInvalidRequest, err)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrGetPolicyModulesPolicyNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		utils.HTTPError(c, srverrors.ErrGetPolicyModulesInvalidPolicyData, err)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")

	queryA := query
	queryA.Page = 0
	queryA.Size = 0
	queryA.Filters = []utils.TableFilter{}
	queryA.Init("modules", modulesSQLMappers)
	queryA.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("policy_id = ?", policy.ID)
		},
		func(db *gorm.DB) *gorm.DB {
			return db.Where("deleted_at IS NULL")
		},
	})
	if _, err = queryA.Query(iDB, &modulesA); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy modules")
		utils.HTTPError(c, srverrors.ErrGetPolicyModulesInvalidPolicyQuery, err)
		return
	}

	queryS := query
	queryS.Init("modules", modulesSQLMappers)
	queryS.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("tenant_id IN (0, ?) AND service_type = ?", tid, sv.Type)
		},
	})
	queryS.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("status desc").Order("name asc")
		},
	})
	modNames := []string{""}
	for _, module := range modulesA {
		modNames = append(modNames, module.Info.Name)
	}
	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return LatestModulesQuery(db).Omit("id").
				Select("`modules`.*, IF(`name` IN (?), 'joined', 'inactive') AS `status`", modNames)
		},
	}
	if resp.Total, err = queryS.Query(gDB, &modulesS, funcs...); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system modules")
		utils.HTTPError(c, srverrors.ErrGetPolicyModulesInvalidModulesQuery, err)
		return
	}

	var details []policyModuleDetails
	if err = iDB.Raw(sqlPolicyModuleDetails, policy.ID).Scan(&details).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading policies modules details")
		utils.HTTPError(c, srverrors.ErrGetPolicyModulesDetailsNotFound, err)
		return
	}

	getModule := func(name string) *models.ModuleA {
		for _, m := range modulesA {
			if m.Info.Name == name {
				return &m
			}
		}
		return nil
	}
	getDetails := func(name string, ma *models.ModuleA, ms *models.ModuleS) policyModuleDetails {
		rmd := policyModuleDetails{Name: name}
		for _, md := range details {
			if md.Name == name {
				rmd = md
				break
			}
		}
		rmd.Update = ma.Info.Version.String() != ms.Info.Version.String()
		return rmd
	}
	for _, ms := range modulesS {
		ma := getModule(ms.Info.Name)
		if ma == nil {
			mt := ms.ToModuleA()
			mt.Status = "inactive"
			resp.Modules = append(resp.Modules, mt)
			resp.Details = append(resp.Details, getDetails(ms.Info.Name, &mt, &ms))
		} else {
			resp.Modules = append(resp.Modules, *ma)
			resp.Details = append(resp.Details, getDetails(ms.Info.Name, ma, &ms))
		}
	}

	for i := 0; i < len(resp.Modules); i++ {
		if err = resp.Modules[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).
				Errorf("error validating policy module data '%s'", resp.Modules[i].Info.Name)
			utils.HTTPError(c, srverrors.ErrGetPolicyModulesInvalidPolicyData, err)
			return
		}
	}

	var moduleNames []string
	for _, v := range resp.Modules {
		if v.Status == "inactive" {
			moduleNames = append(moduleNames, v.Info.Name)
		}
	}
	duplicateMap, err := CheckMultipleModulesDuplicate(iDB, moduleNames, policy.ID)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error checking duplicate modules")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}
	for i, v := range resp.Details {
		if _, ok := duplicateMap[v.Name]; ok {
			resp.Details[i].Duplicate = true
		}
	}
	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// GetPolicyModule is a function to return policy module by name
// @Summary Retrieve policy module data by policy hash and module name
// @Tags Policies,Modules
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} utils.successResp{data=models.ModuleA} "policy module data received successful"
// @Failure 403 {object} utils.errorResp "getting policy module data not permitted"
// @Failure 404 {object} utils.errorResp "policy or module not found"
// @Failure 500 {object} utils.errorResp "internal error on getting policy module"
// @Router /policies/{hash}/modules/{module_name} [get]
func GetPolicyModule(c *gin.Context) {
	var (
		err        error
		hash       = c.Param("hash")
		iDB        *gorm.DB
		moduleName = c.Param("module_name")
		module     models.ModuleA
		policy     models.Policy
	)

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrGetPolicyModulePolicyNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		utils.HTTPError(c, srverrors.ErrGetPolicyModuleInvalidPolicyData, err)
		return
	}

	if err = iDB.Take(&module, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy module by name")
		utils.HTTPError(c, srverrors.ErrModulesNotFound, err)
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidData, err)
		return
	}

	utils.HTTPSuccess(c, http.StatusOK, module)
}

// GetPolicyBModule is a function to return bmodule vue code as a file
// @Summary Retrieve browser module vue code by policy hash and module name
// @Tags Policies,Modules
// @Produce text/javascript,application/javascript,json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param file query string false "path to the browser module file" default(main.vue)
// @Success 200 {file} file "browser module vue code as a file"
// @Failure 403 {object} utils.errorResp "getting policy module data not permitted"
// @Router /policies/{hash}/modules/{module_name}/bmodule.vue [get]
func GetPolicyBModule(c *gin.Context) {
	var (
		err        error
		data       []byte
		filepath   = path.Join("/", c.DefaultQuery("file", "main.vue"))
		hash       = c.Param("hash")
		iDB        *gorm.DB
		module     models.ModuleA
		moduleName = c.Param("module_name")
		policy     models.Policy
		s3         storage.IStorage
		sv         *models.Service
	)

	defer func() {
		ctype := mime.TypeByExtension(path.Ext(filepath))
		if ctype == "" {
			ctype = "application/octet-stream"
		}
		c.Writer.Header().Add("Content-Disposition", fmt.Sprintf("attachment; filename=%q", path.Base(filepath)))
		c.Data(http.StatusOK, ctype, data)
	}()

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		return
	}

	if sv = getService(c); sv == nil {
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		return
	}

	if err = iDB.Take(&module, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy module by name")
		return
	}

	s3, err = storage.NewS3(sv.Info.S3.ToS3ConnParams())
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		return
	}

	path := path.Join(moduleName, module.Info.Version.String(), "bmodule", filepath)
	if data, err = s3.ReadFile(path); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error reading module file '%s'", path)
		return
	}
}

// PatchPolicyModule is a function to update policy module info and status
// @Summary Update or patch policy module data by policy hash and module name
// @Tags Policies,Modules
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param json body policyModulePatch true "action on policy module as JSON data (activate, deactivate, store, update)"
// @Success 200 {object} utils.successResp "policy module patched successful"
// @Failure 403 {object} utils.errorResp "updating policy module not permitted"
// @Failure 404 {object} utils.errorResp "policy or module not found"
// @Failure 500 {object} utils.errorResp "internal error on updating policy module"
// @Router /policies/{hash}/modules/{module_name} [put]
func PatchPolicyModule(c *gin.Context) {
	var (
		err        error
		form       policyModulePatch
		hash       = c.Param("hash")
		gDB        *gorm.DB
		iDB        *gorm.DB
		moduleA    models.ModuleA
		moduleName = c.Param("module_name")
		moduleS    models.ModuleS
		policy     models.Policy
		sv         *models.Service
		encryptor  crypto.IDBConfigEncryptor
	)
	uaf := utils.UserActionFields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "editing",
		ObjectId:          hash,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if err = c.ShouldBindJSON(&form); err != nil {
		name, nameErr := getPolicyName(c, hash)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidRequest, err, uaf)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if encryptor = getDBEncryptor(c); encryptor == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBEncryptorNotFound, nil, uaf)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModulePolicyNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En

	if err = policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModuleInvalidPolicyData, err, uaf)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, err, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		db = db.Where("name = ? AND tenant_id IN (0, ?) AND service_type = ?", moduleName, tid, sv.Type)
		switch form.Version {
		case "latest", "":
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC")
		default:
			return db.Where("version = ?", form.Version)
		}
	}

	if err = gDB.Scopes(scope).Take(&moduleS).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesSystemModuleNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = moduleS.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", moduleS.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}

	if err = iDB.Take(&moduleA, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		moduleA.FromModuleS(&moduleS)
		moduleA.PolicyID = policy.ID
	} else if err = moduleA.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", moduleA.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidData, err, uaf)
		return
	}

	if moduleA.ID == 0 && form.Action != "activate" {
		utils.FromContext(c).WithError(nil).Errorf("error on %s module, policy module not found", form.Action)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}

	incl := []interface{}{"status", "last_update"}
	excl := []string{"policy_id", "status", "join_date", "last_update"}
	switch form.Action {
	case "activate":
		if err = CheckModulesDuplicate(iDB, &moduleA); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error checking duplicate modules")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModuleDuplicatedModule, err, uaf)
			return
		}

		if moduleA.ID == 0 {
			if err = CopyModuleAFilesToInstanceS3(&moduleA.Info, sv); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error copying module files to S3")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
				return
			}

			if err = moduleA.ValidateEncryption(encryptor); err != nil {
				utils.FromContext(c).WithError(err).Errorf("module config not encrypted")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesDataNotEncryptedOnDBInsert, nil, uaf)
				return
			}

			if err = iDB.Create(&moduleA).Error; err != nil {
				utils.FromContext(c).WithError(err).Errorf("error creating module")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
				return
			}

			if moduleS.State == "draft" {
				if err = updatePolicyModulesByModuleS(c, &moduleS, sv); err != nil {
					utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
					return
				}
			}
		} else {
			moduleA.Status = "joined"
			if err = iDB.Select("", incl...).Save(&moduleA).Error; err != nil {
				utils.FromContext(c).WithError(err).Errorf("error updating module")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
				return
			}
		}

	case "deactivate":
		moduleA.Status = "inactive"
		if err = iDB.Select("", incl...).Save(&moduleA).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error updating module")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

	case "store":
		if err = form.Module.Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error validating module")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModuleNewModuleInvalid, err, uaf)
			return
		}

		changes, err := compareModulesChanges(form.Module, moduleA, encryptor)
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("failed to compare modules changes")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToCompareChanges, err, uaf)
			return
		}

		for _, ch := range changes {
			if ch {
				utils.FromContext(c).WithError(nil).Errorf("error accepting module changes")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModuleAcceptFail, nil, uaf)
				return
			}
		}

		err = form.Module.EncryptSecureParameters(encryptor)
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToEncryptSecureConfig, err, uaf)
			return
		}
		if err = iDB.Omit(excl...).Save(&form.Module).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error saving module")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

	case "update":
		moduleVersion := moduleA.Info.Version.String()
		if moduleVersion == moduleS.Info.Version.String() {
			utils.FromContext(c).WithError(nil).Errorf("error updating module to the same version: %s", moduleVersion)
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

		moduleA, err = MergeModuleAConfigFromModuleS(&moduleA, &moduleS, encryptor)
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("invalid module state")
			utils.HTTPError(c, srverrors.ErrInternal, err)
			return
		}

		if err = moduleA.Valid(); err != nil {
			utils.FromContext(c).WithError(err).Errorf("invalid module state")
			utils.HTTPError(c, srverrors.ErrInternal, err)
			return
		}

		err = moduleA.EncryptSecureParameters(encryptor)
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToEncryptSecureConfig, err, uaf)
			return
		}

		if err = CopyModuleAFilesToInstanceS3(&moduleA.Info, sv); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error copying module files to S3")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

		if err = iDB.Omit(excl...).Save(&moduleA).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error updating module")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

		if err = removeUnusedModuleVersion(c, iDB, moduleName, moduleVersion, sv); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error removing unused module data")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

	default:
		utils.FromContext(c).WithError(nil).Errorf("error making unknown action on module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModuleActionNotFound, nil, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

func compareModulesChanges(moduleIn, moduleDB models.ModuleA, encryptor crypto.IDBConfigEncryptor) ([]bool, error) {
	err := moduleIn.DecryptSecureParameters(encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt moduleIn secure config: %w", err)
	}
	err = moduleDB.DecryptSecureParameters(encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt moduleDB secure config: %w", err)
	}

	secDefConfigCompare := !reflect.DeepEqual(moduleIn.SecureDefaultConfig, moduleDB.SecureDefaultConfig)
	secCurrentConfigCompare := !reflect.DeepEqual(moduleIn.SecureCurrentConfig, moduleDB.SecureCurrentConfig)

	err = moduleIn.EncryptSecureParameters(encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt moduleIn secure config: %w", err)
	}
	err = moduleDB.EncryptSecureParameters(encryptor)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt moduleDB secure config: %w", err)
	}

	return []bool{
		moduleIn.ID != moduleDB.ID,
		moduleIn.Info.Name != moduleDB.Info.Name,
		moduleIn.Info.System != moduleDB.Info.System,
		moduleIn.Info.Template != moduleDB.Info.Template,
		moduleIn.Info.Version.String() != moduleDB.Info.Version.String(),
		moduleIn.PolicyID != moduleDB.PolicyID,
		moduleIn.JoinDate != moduleDB.JoinDate,
		moduleIn.LastModuleUpdate != moduleDB.LastModuleUpdate,
		moduleIn.State != moduleDB.State,
		secDefConfigCompare,
		secCurrentConfigCompare,
	}, nil
}

// DeletePolicyModule is a function to delete policy module instance
// @Summary Delete module instance by policy hash and module name
// @Tags Policies,Modules
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} utils.successResp "policy module deleted successful"
// @Failure 403 {object} utils.errorResp "deleting policy module not permitted"
// @Failure 404 {object} utils.errorResp "policy or module not found"
// @Failure 500 {object} utils.errorResp "internal error on deleting policy module"
// @Router /policies/{hash}/modules/{module_name} [delete]
func DeletePolicyModule(c *gin.Context) {
	var (
		err        error
		hash       = c.Param("hash")
		iDB        *gorm.DB
		module     models.ModuleA
		moduleName = c.Param("module_name")
		policy     models.Policy
		sv         *models.Service
	)
	uaf := utils.UserActionFields{
		Domain:            "policy",
		ObjectType:        "policy",
		ActionCode:        "editing",
		ObjectId:          hash,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrDeletePolicyModulePolicyNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeletePolicyModuleInvalidPolicyData, err, uaf)
		return
	}
	uaf.ObjectDisplayName = policy.Info.Name.En

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	if err = iDB.Take(&module, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", module.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidData, err, uaf)
		return
	}

	if err = iDB.Delete(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error deleting policy module by name '%s'", moduleName)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	moduleVersion := module.Info.Version.String()
	if err = removeUnusedModuleVersion(c, iDB, moduleName, moduleVersion, sv); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error removing unused module data")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// SetPolicyModuleSecureConfigValue is a function to set secured parameter value in policy module
// @Summary Set parameter value in secured current config for a module
// @Tags Policies,Modules
// @Accept json
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param json body models.ModuleConfig true "param name and value to be set"
// @Success 200 {object} utils.successResp "parameter updated successfully"
// @Success 400 {object} utils.errorResp "bad request"
// @Failure 403 {object} utils.errorResp "updating parameter not permitted"
// @Failure 404 {object} utils.errorResp "policy or module not found"
// @Failure 500 {object} utils.errorResp "internal error on updating secured parameter"
// @Router /policies/{hash}/modules/{module_name}/secure_config [post]
func SetPolicyModuleSecureConfigValue(c *gin.Context) {
	var (
		err        error
		payload    models.ModuleConfig
		gDB        *gorm.DB
		iDB        *gorm.DB
		moduleA    models.ModuleA
		moduleS    models.ModuleS
		policy     models.Policy
		sv         *models.Service
		hash       = c.Param("hash")
		moduleName = c.Param("module_name")
		encryptor  crypto.IDBConfigEncryptor
		paramName  string
	)
	uaf := utils.UserActionFields{
		Domain:     "policy",
		ObjectType: "policy",
		ActionCode: "setting value to module secure config",
		ObjectId:   hash,
	}

	err = c.ShouldBindJSON(&payload)
	switch {
	case err != nil:
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidRequest, err, uaf)
		return
	case len(payload) != 1:
		utils.FromContext(c).WithError(err).Errorf("only one key-value pair in body allowed")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidRequest, err, uaf)
		return
	}

	for k := range payload {
		paramName = k
	}
	uaf.ActionCode = fmt.Sprintf("%s, key: %s", uaf.ActionCode, paramName)

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, err, uaf)
		return
	}

	if encryptor = getDBEncryptor(c); encryptor == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBEncryptorNotFound, nil, uaf)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModulePolicyNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModuleInvalidPolicyData, err, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id IN (0, ?) AND service_type = ?", moduleName, tid, sv.Type).
			Order("ver_major DESC, ver_minor DESC, ver_patch DESC") // latest
	}

	if err = gDB.Scopes(scope).Take(&moduleS).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesSystemModuleNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = moduleS.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", moduleS.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}
	uaf.ObjectDisplayName = moduleS.Locale.Module["en"].Title

	if err = iDB.Take(&moduleA, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		moduleA.FromModuleS(&moduleS)
		moduleA.PolicyID = policy.ID
	} else if err = moduleA.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", moduleA.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidData, err, uaf)
		return
	}

	if err = moduleA.DecryptSecureParameters(encryptor); err != nil {
		utils.FromContext(c).WithError(err).
			Errorf("error decrypting module data '%s'", moduleA.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToDecryptSecureConfig, err, uaf)
		return
	}

	param, ok := moduleA.SecureCurrentConfig[paramName]
	if !ok {
		utils.FromContext(c).WithError(err).Errorf("module secure parameter not exists")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidRequest, err, uaf)
		return
	}

	moduleA.SecureCurrentConfig[paramName] = models.ModuleSecureParameter{
		ServerOnly: param.ServerOnly,
		Value:      payload[paramName],
	}
	if err = moduleA.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error saving module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidParameterValue, err, uaf)
		return
	}

	err = moduleA.EncryptSecureParameters(encryptor)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToEncryptSecureConfig, err, uaf)
		return
	}
	if err = iDB.Save(&moduleA).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error saving module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// GetPolicyModuleSecureConfigValue is a function to get secured parameter value in policy module
// @Summary Get parameter value in secured current config for a module
// @Tags Policies,Modules
// @Produce json
// @Param hash path string true "policy hash in hex format (md5)" minlength(32) maxlength(32)
// @Param module_name path string true "module name without spaces"
// @Param param_name path string true "parameter name without spaces"
// @Success 200 {object} models.ModuleConfig "secured param value received successfully"
// @Failure 403 {object} utils.errorResp "get secured parameter not permitted"
// @Failure 404 {object} utils.errorResp "policy, module or parameter not found"
// @Failure 500 {object} utils.errorResp "internal error on getting module secured parameter"
// @Router /policies/{hash}/modules/{module_name}/secure_config/{param_name} [get]
func GetPolicyModuleSecureConfigValue(c *gin.Context) {
	var (
		err        error
		gDB        *gorm.DB
		iDB        *gorm.DB
		moduleA    models.ModuleA
		moduleS    models.ModuleS
		policy     models.Policy
		sv         *models.Service
		hash       = c.Param("hash")
		moduleName = c.Param("module_name")
		paramName  = c.Param("param_name")
		encryptor  crypto.IDBConfigEncryptor
	)
	uaf := utils.UserActionFields{
		Domain:     "policy",
		ObjectType: "policy",
		ActionCode: fmt.Sprintf("retrieving value in module secure config, key: %s", paramName),
		ObjectId:   hash,
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); iDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, err, uaf)
		return
	}

	if encryptor = getDBEncryptor(c); encryptor == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBEncryptorNotFound, nil, uaf)
		return
	}

	if err = iDB.Take(&policy, "hash = ?", hash).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policy by hash")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModulePolicyNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = policy.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchPolicyModuleInvalidPolicyData, err, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id IN (0, ?) AND service_type = ?", moduleName, tid, sv.Type).
			Order("ver_major DESC, ver_minor DESC, ver_patch DESC") // latest
	}

	if err = gDB.Scopes(scope).Take(&moduleS).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesSystemModuleNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = moduleS.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", moduleS.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}
	uaf.ObjectDisplayName = moduleS.Locale.Module["en"].Title

	if err = iDB.Take(&moduleA, "policy_id = ? AND name = ?", policy.ID, moduleName).Error; err != nil {
		moduleA.FromModuleS(&moduleS)
		moduleA.PolicyID = policy.ID
	} else if err = moduleA.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module data '%s'", moduleA.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidData, err, uaf)
		return
	}

	if err = moduleA.DecryptSecureParameters(encryptor); err != nil {
		utils.FromContext(c).WithError(err).
			Errorf("error decrypting module data '%s'", moduleA.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToDecryptSecureConfig, err, uaf)
		return
	}

	val, ok := moduleA.SecureCurrentConfig[paramName]
	if !ok {
		utils.FromContext(c).WithError(err).Errorf("module secure parameter not exists")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidRequest, err, uaf)
		return
	}

	resp := make(models.ModuleConfig)
	resp[paramName] = val.Value
	utils.HTTPSuccessWithUAFields(c, http.StatusOK, resp, uaf)
}

// GetModules is a function to return system module list
// @Summary Retrieve system modules by filters
// @Tags Modules
// @Produce json
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=systemModules} "system modules received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting system modules not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting system modules"
// @Router /modules/ [get]
func GetModules(c *gin.Context) {
	var (
		err        error
		gDB        *gorm.DB
		query      utils.TableQuery
		sv         *models.Service
		resp       systemModules
		useVersion bool
		encryptor  crypto.IDBConfigEncryptor
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrModulesInvalidRequest, err)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	if encryptor = getDBEncryptor(c); encryptor == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")

	query.Init("modules", modulesSQLMappers)

	setUsingTables := func(sfield string) {
		if sfield == "version" {
			useVersion = true
		}
	}
	setUsingTables(query.Sort.Prop)
	for _, filter := range query.Filters {
		setUsingTables(filter.Field)
	}
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("tenant_id IN (0, ?) AND service_type = ?", tid, sv.Type)
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("name asc")
		},
	})
	funcs := []func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			if !useVersion {
				db = LatestModulesQuery(db)
			}
			return db
		},
	}
	if resp.Total, err = query.Query(gDB, &resp.Modules, funcs...); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system modules")
		utils.HTTPError(c, srverrors.ErrGetModulesInvalidModulesQuery, err)
		return
	}

	for _, module := range resp.Modules {
		if err = module.Valid(); err != nil {
			utils.FromContext(c).WithError(err).
				Errorf("error validating system module data '%s'", module.Info.Name)
			utils.HTTPError(c, srverrors.ErrModulesInvalidSystemModuleData, err)
			return
		}

		if err = module.DecryptSecureParameters(encryptor); err != nil {
			utils.FromContext(c).WithError(err).
				Errorf("error decrypting module data '%s'", module.Info.Name)
			utils.HTTPError(c, srverrors.ErrModulesFailedToDecryptSecureConfig, err)
			return
		}
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// CreateModule is a function to create new system module
// @Summary Create new system module from template
// @Tags Modules
// @Accept json
// @Produce json
// @Param json body models.ModuleInfo true "module info to create one"
// @Success 201 {object} utils.successResp{data=models.ModuleS} "system module created successful"
// @Failure 400 {object} utils.errorResp "invalid system module info"
// @Failure 403 {object} utils.errorResp "creating system module not permitted"
// @Failure 500 {object} utils.errorResp "internal error on creating system module"
// @Router /modules/ [post]
func CreateModule(c *gin.Context) {
	var (
		count     int64
		err       error
		gDB       *gorm.DB
		info      models.ModuleInfo
		module    *models.ModuleS
		sv        *models.Service
		template  Template
		encryptor crypto.IDBConfigEncryptor
	)
	uaf := utils.UserActionFields{
		Domain:            "module",
		ObjectType:        "module",
		ActionCode:        "creation",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	if encryptor = getDBEncryptor(c); encryptor == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBEncryptorNotFound, nil, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")

	if err = c.ShouldBindJSON(&info); err != nil || info.Valid() != nil {
		if err == nil {
			err = info.Valid()
		}
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleInvalidInfo, err, uaf)
		return
	}
	uaf.ObjectId = info.Name

	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", info.Name, tid, sv.Type)
	}

	if err = gDB.Scopes(scope).Model(&module).Count(&count).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding number of system module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleGetCountFail, err, uaf)
		return
	} else if count >= 1 {
		utils.FromContext(c).WithError(nil).Errorf("error creating second system module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleSecondSystemModule, err, uaf)
		return
	}

	info.System = false
	if template, module, err = LoadModuleSTemplate(&info); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error loading module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleLoadFail, err, uaf)
		return
	}
	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	module.State = "draft"
	module.TenantID = tid
	module.ServiceType = sv.Type
	if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleValidationFail, err, uaf)
		return
	}

	if err = StoreModuleSToGlobalS3(&info, template); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error storing module to S3")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleStoreS3Fail, err, uaf)
		return
	}

	err = module.DecryptSecureParameters(encryptor)
	if err != nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToEncryptSecureConfig, nil, uaf)
		return
	}
	if err = gDB.Create(module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error creating module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleStoreDBFail, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusCreated, module, uaf)
}

// DeleteModule is a function to cascade delete system module
// @Summary Delete system module from all DBs and S3 storage
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Success 200 {object} utils.successResp "system module deleted successful"
// @Failure 403 {object} utils.errorResp "deleting system module not permitted"
// @Failure 404 {object} utils.errorResp "system module or services not found"
// @Failure 500 {object} utils.errorResp "internal error on deleting system module"
// @Router /modules/{module_name} [delete]
func DeleteModule(c *gin.Context) {
	var (
		err        error
		gDB        *gorm.DB
		modules    []models.ModuleS
		moduleName = c.Param("module_name")
		s3         storage.IStorage
		sv         *models.Service
		services   []models.Service
	)
	uaf := utils.UserActionFields{
		Domain:            "module",
		ObjectType:        "module",
		ActionCode:        "deletion",
		ObjectId:          moduleName,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(scope).Find(&modules).Error; err != nil || len(modules) == 0 {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if err == nil && len(modules) == 0 {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	}
	uaf.ObjectDisplayName = modules[len(modules)-1].Locale.Module["en"].Title

	deletePolicyModule := func(s *models.Service) error {
		var (
			err     error
			iDB     *gorm.DB
			modules []models.ModuleA
			s3      storage.IStorage
		)

		iDB = utils.GetDB(s.Info.DB.User, s.Info.DB.Pass, s.Info.DB.Host,
			strconv.Itoa(int(s.Info.DB.Port)), s.Info.DB.Name)
		if iDB == nil {
			utils.FromContext(c).WithError(nil).Errorf("error openning connection to instance DB")
			return errors.New("failed to connect to instance DB")
		}
		defer iDB.Close()

		if err = iDB.Find(&modules, "name = ?", moduleName).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding modules by name")
			return err
		} else if len(modules) == 0 {
			return updateDependenciesWhenModuleRemove(c, iDB, moduleName)
		} else if err = iDB.Where("name = ?", moduleName).Delete(&modules).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error deleting module by name '%s'", moduleName)
			return err
		}

		s3, err = storage.NewS3(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
			return err
		}

		if err = s3.RemoveDir(moduleName + "/"); err != nil && err.Error() != "not found" {
			utils.FromContext(c).WithError(err).Errorf("error removing modules files")
			return err
		}

		return updateDependenciesWhenModuleRemove(c, iDB, moduleName)
	}

	if err = gDB.Find(&services, "tenant_id = ? AND type = ?", tid, sv.Type).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding services")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeleteModuleServiceNotFound, err, uaf)
		return
	}

	for _, s := range services {
		if err = deletePolicyModule(&s); err != nil {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrDeleteModuleDeleteFail, err, uaf)
			return
		}
	}

	if err = gDB.Where("name = ?", moduleName).Delete(&modules).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error deleting system module by name '%s'", moduleName)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	if s3, err = storage.NewS3(nil); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	if err = s3.RemoveDir(moduleName + "/"); err != nil && err.Error() != "not found" {
		utils.FromContext(c).WithError(err).Errorf("error removing system modules files")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeleteModuleDeleteFilesFail, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// GetModuleVersions is a function to return all versions for system module
// @Summary Retrieve all version for system module by filters and module name
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param request query utils.TableQuery true "query table params"
// @Success 200 {object} utils.successResp{data=systemShortModules} "system modules received successful"
// @Failure 400 {object} utils.errorResp "invalid query request data"
// @Failure 403 {object} utils.errorResp "getting system modules not permitted"
// @Failure 500 {object} utils.errorResp "internal error on getting system modules"
// @Router /modules/{module_name}/versions/ [get]
func GetModuleVersions(c *gin.Context) {
	var (
		err        error
		gDB        *gorm.DB
		moduleName = c.Param("module_name")
		query      utils.TableQuery
		sv         *models.Service
		resp       systemShortModules
	)

	if err = c.ShouldBindQuery(&query); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding query")
		utils.HTTPError(c, srverrors.ErrModulesInvalidRequest, err)
		return
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")

	query.Init("modules", modulesSQLMappers)
	query.SetFilters([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Where("name = ? AND tenant_id IN (0, ?) AND service_type = ?", moduleName, tid, sv.Type)
		},
	})
	query.SetOrders([]func(db *gorm.DB) *gorm.DB{
		func(db *gorm.DB) *gorm.DB {
			return db.Order("ver_major DESC, ver_minor DESC, ver_patch DESC")
		},
	})
	if resp.Total, err = query.Query(gDB, &resp.Modules); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system modules")
		utils.HTTPError(c, srverrors.ErrGetModuleVersionsInvalidModulesQuery, err)

		return
	}

	for i := 0; i < len(resp.Modules); i++ {
		if err = resp.Modules[i].Valid(); err != nil {
			utils.FromContext(c).WithError(err).
				Errorf("error validating system module data '%s'", resp.Modules[i].Info.Name)
			utils.HTTPError(c, srverrors.ErrModulesInvalidSystemModuleData, err)
			return
		}
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// GetModuleVersion is a function to return system module by name and version
// @Summary Retrieve system module data by module name and version
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 200 {object} utils.successResp{data=models.ModuleS} "system module data received successful"
// @Failure 403 {object} utils.errorResp "getting system module data not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on getting system module"
// @Router /modules/{module_name}/versions/{version} [get]
func GetModuleVersion(c *gin.Context) {
	var (
		err        error
		gDB        *gorm.DB
		module     models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		version    = c.Param("version")
		encryptor  crypto.IDBConfigEncryptor
	)

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	if encryptor = getDBEncryptor(c); encryptor == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBEncryptorNotFound, nil)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrModulesNotFound, err)

		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidSystemModuleData, err)
		return
	}

	if err = module.DecryptSecureParameters(encryptor); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error decrypting module data")
		utils.HTTPError(c, srverrors.ErrModulesFailedToDecryptSecureConfig, err)
		return
	}

	utils.HTTPSuccess(c, http.StatusOK, module)
}

// PatchModuleVersion is a function to update system module by name and version
// @Summary Update the version of system module to global DB and global S3 storage
// @Tags Modules
// @Accept json
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Param json body moduleVersionPatch true "module info to create one"
// @Success 200 {object} utils.successResp "system module updated successful"
// @Failure 403 {object} utils.errorResp "updating system module not permitted"
// @Failure 404 {object} utils.errorResp "system module or services not found"
// @Failure 500 {object} utils.errorResp "internal error on updating system module"
// @Router /modules/{module_name}/versions/{version} [put]
func PatchModuleVersion(c *gin.Context) {
	var (
		cfiles     map[string][]byte
		err        error
		gDB        *gorm.DB
		module     models.ModuleS
		moduleName = c.Param("module_name")
		form       moduleVersionPatch
		sv         *models.Service
		services   []models.Service
		template   = make(Template)
		version    = c.Param("version")
		encryptor  crypto.IDBConfigEncryptor
	)
	uaf := utils.UserActionFields{
		Domain:            "module",
		ObjectType:        "module",
		ObjectId:          moduleName,
		ActionCode:        "undefined action",
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if err = c.ShouldBindJSON(&form); err != nil || form.Module.Valid() != nil {
		if err == nil {
			err = form.Module.Valid()
		}
		name, nameErr := getModuleName(c, moduleName, version)
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidRequest, err, uaf)
		return
	}

	if form.Action == "release" {
		uaf.ActionCode = "release version release"
	} else {
		uaf.ActionCode = "module editing"
	}
	uaf.ObjectDisplayName = form.Module.Locale.Module["en"].Title

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	if encryptor = getDBEncryptor(c); encryptor == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBEncryptorNotFound, nil, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesSystemModuleNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}

	if module.State == "release" {
		utils.FromContext(c).WithError(nil).Errorf("error changing released system module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionAcceptReleaseChangesFail, nil, uaf)
		return
	}

	changes := []bool{
		module.ID != form.Module.ID,
		module.Info.Name != form.Module.Info.Name,
		module.Info.System != form.Module.Info.System,
		module.Info.Template != form.Module.Info.Template,
		module.Info.Version.String() != form.Module.Info.Version.String(),
		module.ServiceType != form.Module.ServiceType,
		module.TenantID != form.Module.TenantID,
	}
	for _, ch := range changes {
		if ch {
			utils.FromContext(c).WithError(nil).Errorf("error accepting system module changes")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionAcceptSystemChangesFail, nil, uaf)
			return
		}
	}

	err = form.Module.EncryptSecureParameters(encryptor)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToEncryptSecureConfig, err, uaf)
		return
	}
	if sqlResult := gDB.Omit("last_update").Save(&form.Module); sqlResult.Error != nil {
		utils.FromContext(c).WithError(sqlResult.Error).Errorf("error saving system module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionUpdateFail, err, uaf)
		return
	} else if sqlResult.RowsAffected != 0 {
		if cfiles, err = BuildModuleSConfig(&form.Module); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error building system module files")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionBuildFilesFail, err, uaf)
			return
		}

		template["config"] = cfiles
		if err = StoreModuleSToGlobalS3(&form.Module.Info, template); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error storing system module files to S3")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionUpdateS3Fail, err, uaf)
			return
		}
	}

	if module.State == "draft" && form.Module.State == "release" {
		if err = gDB.Find(&services, "tenant_id = ? AND type = ?", tid, sv.Type).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding services")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionServiceNotFound, err, uaf)
			return
		}

		if err = gDB.Model(&form.Module).Take(&module).Error; err != nil {
			utils.FromContext(c).WithError(err).Errorf("error finding system module by id '%d'", form.Module.ID)
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
			return
		}

		for _, s := range services {
			if err = updatePolicyModulesByModuleS(c, &module, &s); err != nil {
				utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
				return
			}
		}
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// CreateModuleVersion is a function to create new system module version
// @Summary Create new system module version from latest released version
// @Tags Modules
// @Accept json
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format"
// @Param json body models.ChangelogVersion true "module changelog to add to created module"
// @Success 201 {object} utils.successResp{data=models.ModuleS} "system module created successful"
// @Failure 400 {object} utils.errorResp "invalid system module info"
// @Failure 403 {object} utils.errorResp "creating system module not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on creating system module"
// @Router /modules/{module_name}/versions/{version} [post]
func CreateModuleVersion(c *gin.Context) {
	var (
		cfiles     map[string][]byte
		clver      models.ChangelogVersion
		count      int64
		err        error
		gDB        *gorm.DB
		module     models.ModuleS
		moduleName = c.Param("module_name")
		sv         *models.Service
		template   Template
		version    = c.Param("version")
		encryptor  crypto.IDBConfigEncryptor
	)
	uaf := utils.UserActionFields{
		Domain:            "module",
		ObjectType:        "module",
		ActionCode:        "creation of the draft",
		ObjectId:          moduleName,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	if encryptor = getDBEncryptor(c); encryptor == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBEncryptorNotFound, nil, uaf)
		return
	}

	if err = c.ShouldBindJSON(&clver); err != nil || clver.Valid() != nil {
		if err == nil {
			err = clver.Valid()
		}
		name, nameErr := getModuleName(c, moduleName, "latest")
		if nameErr == nil {
			uaf.ObjectDisplayName = name
		}
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidRequest, err, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion("latest"), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesSystemModuleNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	}

	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}

	draft := func(db *gorm.DB) *gorm.DB {
		return db.Where("state LIKE ?", "draft")
	}

	if err = gDB.Scopes(scope, draft).Model(&module).Count(&count).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding number of system module drafts")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionGetDraftNumberFail, err, uaf)
		return
	} else if count >= 1 {
		utils.FromContext(c).WithError(nil).Errorf("error creating system module second draft")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionSecondSystemModuleDraft, err, uaf)
		return
	}

	newModuleVersion, err := semver.NewVersion(version)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("error parsing new version '%s'", version)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionInvalidModuleVersionFormat, err, uaf)
		return
	}

	switch utils.CompareVersions(module.Info.Version.String(), version) {
	case utils.TargetVersionGreat:
	default:
		utils.FromContext(c).WithError(nil).Errorf("error validating new version '%s' -> '%s'",
			module.Info.Version.String(), version)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionInvalidModuleVersion, nil, uaf)
		return
	}

	if template, err = LoadModuleSFromGlobalS3(&module.Info); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error building system module files")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionBuildFilesFail, err, uaf)
		return
	}

	module.ID = 0
	module.State = "draft"
	module.LastUpdate = time.Now()
	module.Info.Version.Major = newModuleVersion.Major()
	module.Info.Version.Minor = newModuleVersion.Minor()
	module.Info.Version.Patch = newModuleVersion.Patch()
	module.Changelog[module.Info.Version.String()] = clver
	if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionValidationFail, err, uaf)
		return
	}

	if cfiles, err = BuildModuleSConfig(&module); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error building system module files")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionBuildFilesFail, err, uaf)
		return
	}

	template["config"] = cfiles
	if err = StoreModuleSToGlobalS3(&module.Info, template); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error storing module to S3")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionStoreS3Fail, err, uaf)
		return
	}

	err = module.EncryptSecureParameters(encryptor)
	if err != nil {
		utils.FromContext(c).WithError(err).Errorf("failed to encrypt module secure config")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesFailedToEncryptSecureConfig, err, uaf)
		return
	}
	if err = gDB.Create(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error creating module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrCreateModuleVersionStoreDBFail, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusCreated, module, uaf)
}

// DeleteModuleVersion is a function to delete the version system module
// @Summary Delete the version system module from global DB and global S3 storage
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 200 {object} utils.successResp "system module deleted successful"
// @Failure 403 {object} utils.errorResp "deleting system module not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on deleting system module"
// @Router /modules/{module_name}/versions/{version} [delete]
func DeleteModuleVersion(c *gin.Context) {
	var (
		count      int64
		err        error
		gDB        *gorm.DB
		module     models.ModuleS
		moduleName = c.Param("module_name")
		s3         storage.IStorage
		sv         *models.Service
		version    = c.Param("version")
	)
	uaf := utils.UserActionFields{
		Domain:            "module",
		ObjectType:        "module",
		ActionCode:        "deletion of the version",
		ObjectId:          moduleName,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(scope).Model(&module).Count(&count).Error; err != nil || count == 0 {
		utils.FromContext(c).WithError(err).Errorf("error finding number of system module versions")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeleteModuleVersionGetVersionNumberFail, err, uaf)
		return
	} else if count == 1 {
		utils.FromContext(c).WithError(nil).Errorf("error deleting last system module version")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeleteModuleVersionDeleteLastVersionFail, nil, uaf)
		return
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesSystemModuleNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}
	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	if err = gDB.Delete(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error deleting system module by name '%s'", moduleName)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	if s3, err = storage.NewS3(nil); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	path := moduleName + "/" + module.Info.Version.String() + "/"
	if err = s3.RemoveDir(path); err != nil && err.Error() != "not found" {
		utils.FromContext(c).WithError(err).Errorf("error removing system modules files")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrDeleteModuleVersionDeleteFilesFail, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// GetModuleVersionUpdates is a function to return policy modules list ready to update
// @Summary Retrieve policy modules list ready to update by system module name and version
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 200 {object} utils.successResp{data=policyModulesUpdates} "policy modules list received successful"
// @Failure 403 {object} utils.errorResp "getting policy modules list not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on getting policy modules list to update"
// @Router /modules/{module_name}/versions/{version}/updates [get]
func GetModuleVersionUpdates(c *gin.Context) {
	var (
		err        error
		gDB        *gorm.DB
		iDB        *gorm.DB
		module     models.ModuleS
		moduleName = c.Param("module_name")
		pids       []uint64
		resp       policyModulesUpdates
		scope      func(db *gorm.DB) *gorm.DB
		sv         *models.Service
		version    = c.Param("version")
	)

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if iDB = utils.GetGormDB(c, "iDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope = func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrModulesSystemModuleNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidSystemModuleData, err)
		return
	}

	scope = func(db *gorm.DB) *gorm.DB {
		return db.Where("name LIKE ? AND version LIKE ? AND last_module_update NOT LIKE ?",
			module.Info.Name, module.Info.Version.String(), module.LastUpdate)
	}
	if err = iDB.Scopes(scope).Find(&resp.Modules).Error; err != nil {
		utils.FromContext(c).WithError(err).
			Errorf("error finding policy modules by name and version '%s' '%s'",
				module.Info.Name, module.Info.Version.String())
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	for _, module := range resp.Modules {
		pids = append(pids, module.PolicyID)
	}

	if err = iDB.Find(&resp.Policies, "id IN (?)", pids).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding policies by IDs")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	} else {
		for _, policy := range resp.Policies {
			if err = policy.Valid(); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error validating policy data '%s'", policy.Hash)
				utils.HTTPError(c, srverrors.ErrGetModuleVersionUpdatesInvalidPolicyData, err)
				return
			}
		}
	}

	utils.HTTPSuccess(c, http.StatusOK, resp)
}

// CreateModuleVersionUpdates is a function to run policy modules update
// @Summary Run policy modules update by system module name and version
// @Tags Modules
// @Accept json
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 201 {object} utils.successResp "policy modules update run successful"
// @Failure 403 {object} utils.errorResp "running policy modules updates not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on running policy modules updates"
// @Router /modules/{module_name}/versions/{version}/updates [post]
func CreateModuleVersionUpdates(c *gin.Context) {
	var (
		err        error
		gDB        *gorm.DB
		moduleName = c.Param("module_name")
		module     models.ModuleS
		sv         *models.Service
		version    = c.Param("version")
	)
	uaf := utils.UserActionFields{
		Domain:            "module",
		ObjectType:        "module",
		ActionCode:        "version update in policies",
		ObjectId:          moduleName,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesSystemModuleNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}
	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	if err = updatePolicyModulesByModuleS(c, &module, sv); err != nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusCreated, struct{}{}, uaf)
}

// GetModuleVersionFiles is a function to return system module file list
// @Summary Retrieve system module files (relative path) by module name and version
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Success 200 {object} utils.successResp{data=[]string} "system module files received successful"
// @Failure 403 {object} utils.errorResp "getting system module files not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on getting system module files"
// @Router /modules/{module_name}/versions/{version}/files [get]
func GetModuleVersionFiles(c *gin.Context) {
	var (
		err        error
		files      []string
		gDB        *gorm.DB
		module     models.ModuleS
		moduleName = c.Param("module_name")
		s3         storage.IStorage
		sv         *models.Service
		version    = c.Param("version")
	)

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrModulesSystemModuleNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidSystemModuleData, err)
		return
	}

	if s3, err = storage.NewS3(nil); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	path := moduleName + "/" + module.Info.Version.String()
	if files, err = readDir(s3, path); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error listening module files from S3")
		utils.HTTPError(c, srverrors.ErrGetModuleVersionFilesListenFail, err)
		return
	}

	utils.HTTPSuccess(c, http.StatusOK, files)
}

// GetModuleVersionFile is a function to return system module file content
// @Summary Retrieve system module file content (in base64) by module name, version and relative path
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Param path query string true "relative path to module file"
// @Success 200 {object} utils.successResp{data=systemModuleFile} "system module file content received successful"
// @Failure 403 {object} utils.errorResp "getting system module file content not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on getting system module file"
// @Router /modules/{module_name}/versions/{version}/files/file [get]
func GetModuleVersionFile(c *gin.Context) {
	var (
		err        error
		data       string
		fileData   []byte
		filePath   = c.Query("path")
		gDB        *gorm.DB
		module     models.ModuleS
		moduleName = c.Param("module_name")
		s3         storage.IStorage
		sv         *models.Service
		version    = c.Param("version")
	)

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrModulesSystemModuleNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidSystemModuleData, err)
		return
	}

	if s3, err = storage.NewS3(nil); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		utils.HTTPError(c, srverrors.ErrInternal, err)
		return
	}

	prefix := moduleName + "/" + module.Info.Version.String() + "/"
	if !strings.HasPrefix(filePath, prefix) || strings.Contains(filePath, "..") {
		utils.FromContext(c).WithError(nil).Errorf("error parsing path to file: mismatch base prefix")
		utils.HTTPError(c, srverrors.ErrGetModuleVersionFileParsePathFail, nil)
		return
	}

	if fileData, err = s3.ReadFile(filePath); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error reading module file '%s'", filePath)
		utils.HTTPError(c, srverrors.ErrGetModuleVersionFileReadFail, err)
		return
	}

	data = base64.StdEncoding.EncodeToString(fileData)
	utils.HTTPSuccess(c, http.StatusOK, systemModuleFile{Path: filePath, Data: data})
}

// PatchModuleVersionFile is a function to save, move, remove of system module file and its content
// @Summary Patch system module file and content (in base64) by module name, version and relative path
// @Tags Modules
// @Accept json
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Param json body systemModuleFilePatch true "action, relative path and file content for module file"
// @Success 200 {object} utils.successResp "action on system module file did successful"
// @Failure 403 {object} utils.errorResp "making action on system module file not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on making action system module file"
// @Router /modules/{module_name}/versions/{version}/files/file [put]
func PatchModuleVersionFile(c *gin.Context) {
	var (
		err        error
		data       []byte
		files      map[string]os.FileInfo
		form       systemModuleFilePatch
		gDB        *gorm.DB
		info       os.FileInfo
		module     models.ModuleS
		moduleName = c.Param("module_name")
		s3         storage.IStorage
		sv         *models.Service
		version    = c.Param("version")
	)
	uaf := utils.UserActionFields{
		Domain:            "module",
		ObjectType:        "module",
		ActionCode:        "module editing",
		ObjectId:          moduleName,
		ObjectDisplayName: utils.UnknownObjectDisplayName,
	}

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalDBNotFound, nil, uaf)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternalServiceNotFound, nil, uaf)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesSystemModuleNotFound, err, uaf)
		} else {
			utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidSystemModuleData, err, uaf)
		return
	}
	uaf.ObjectDisplayName = module.Locale.Module["en"].Title

	if module.State == "release" {
		utils.FromContext(c).WithError(nil).Errorf("error patching released module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileUpdateFail, nil, uaf)
		return
	}

	if s3, err = storage.NewS3(nil); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error openning connection to S3")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrInternal, err, uaf)
		return
	}

	if err = c.ShouldBindJSON(&form); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error binding JSON")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrModulesInvalidRequest, err, uaf)
		return
	}

	prefix := moduleName + "/" + module.Info.Version.String() + "/"
	if !strings.HasPrefix(form.Path, prefix) || strings.Contains(form.Path, "..") {
		utils.FromContext(c).WithError(nil).Errorf("error parsing path to file: mismatch base prefix")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileParsePathFail, nil, uaf)
		return
	}

	switch form.Action {
	case "save":
		if data, err = base64.StdEncoding.DecodeString(form.Data); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error decoding file data")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileParseModuleFileFail, err, uaf)
			return
		}

		if err = s3.WriteFile(form.Path, data); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error writing file data to S3")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileWriteModuleFileFail, err, uaf)
			return
		}

	case "remove":
		if err = s3.Remove(form.Path); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error removing file from S3")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileWriteModuleObjectFail, err, uaf)
			return
		}

	case "move":
		if !strings.HasPrefix(form.NewPath, prefix) || strings.Contains(form.NewPath, "..") {
			utils.FromContext(c).WithError(nil).Errorf("error parsing path to file: mismatch base prefix")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileParseNewpathFail, nil, uaf)
			return
		}

		if info, err = s3.GetInfo(form.Path); err != nil {
			utils.FromContext(c).WithError(err).Errorf("error getting file info from S3")
			utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileObjectNotFound, err, uaf)
			return
		} else if !info.IsDir() {
			if strings.HasSuffix(form.NewPath, "/") {
				form.NewPath += info.Name()
			}

			if form.Path == form.NewPath {
				utils.FromContext(c).WithError(nil).Errorf("error moving file in S3: newpath is identical to path")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFilePathIdentical, nil, uaf)
				return
			}

			if err = s3.Rename(form.Path, form.NewPath); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error renaming file in S3")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileObjectMoveFail, err, uaf)
				return
			}
		} else {
			if !strings.HasSuffix(form.Path, "/") {
				form.Path += "/"
			}
			if !strings.HasSuffix(form.NewPath, "/") {
				form.NewPath += "/"
			}

			if form.Path == form.NewPath {
				utils.FromContext(c).WithError(nil).Errorf("error moving file in S3: newpath is identical to path")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFilePathIdentical, nil, uaf)
				return
			}

			if files, err = s3.ListDirRec(form.Path); err != nil {
				utils.FromContext(c).WithError(err).Errorf("error getting files by path from S3")
				utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileGetFilesFail, err, uaf)
				return
			}

			for obj, info := range files {
				if !info.IsDir() {
					curfile := filepath.Join(form.Path, obj)
					newfile := filepath.Join(form.NewPath, obj)
					if err = s3.Rename(curfile, newfile); err != nil {
						utils.FromContext(c).WithError(err).Errorf("error moving file in S3")
						utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileObjectMoveFail, err, uaf)
						return
					}
				}
			}
		}

	default:
		utils.FromContext(c).WithError(nil).Errorf("error making unknown action on module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileActionNotFound, nil, uaf)
		return
	}

	if err = gDB.Model(&module).UpdateColumn("last_update", gorm.Expr("NOW()")).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error updating system module")
		utils.HTTPErrorWithUAFields(c, srverrors.ErrPatchModuleVersionFileSystemModuleUpdateFail, err, uaf)
		return
	}

	utils.HTTPSuccessWithUAFields(c, http.StatusOK, struct{}{}, uaf)
}

// GetModuleVersionOption is a function to return option of system module rendered on server side
// @Summary Retrieve rendered Event Config Schema of system module data by module name and version
// @Tags Modules
// @Produce json
// @Param module_name path string true "module name without spaces"
// @Param version path string true "module version string according semantic version format" default(latest)
// @Param option_name path string true "module option without spaces" Enums(id, tenant_id, service_type, state, config_schema, default_config, static_dependencies, fields_schema, action_config_schema, default_action_config, event_config_schema, default_event_config, changelog, locale, info, last_update, event_config_schema_definitions, action_config_schema_definitions)
// @Success 200 {object} utils.successResp{data=interface{}} "module option received successful"
// @Failure 403 {object} utils.errorResp "getting module option not permitted"
// @Failure 404 {object} utils.errorResp "system module not found"
// @Failure 500 {object} utils.errorResp "internal error on getting module option"
// @Router /modules/{module_name}/versions/{version}/options/{option_name} [get]
func GetModuleVersionOption(c *gin.Context) {
	var (
		err        error
		data       []byte
		gDB        *gorm.DB
		module     models.ModuleS
		moduleName = c.Param("module_name")
		optionName = c.Param("option_name")
		sv         *models.Service
		version    = c.Param("version")
	)

	if gDB = utils.GetGormDB(c, "gDB"); gDB == nil {
		utils.HTTPError(c, srverrors.ErrInternalDBNotFound, nil)
		return
	}

	if sv = getService(c); sv == nil {
		utils.HTTPError(c, srverrors.ErrInternalServiceNotFound, nil)
		return
	}

	tid, _ := utils.GetUint64(c, "tid")
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", moduleName, tid, sv.Type)
	}

	if err = gDB.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		utils.FromContext(c).WithError(err).Errorf("error finding system module by name")
		if errors.Is(err, gorm.ErrRecordNotFound) {
			utils.HTTPError(c, srverrors.ErrModulesSystemModuleNotFound, err)
		} else {
			utils.HTTPError(c, srverrors.ErrInternal, err)
		}
		return
	} else if err = module.Valid(); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error validating system module data '%s'", module.Info.Name)
		utils.HTTPError(c, srverrors.ErrModulesInvalidSystemModuleData, err)
		return
	}

	if optionName == "event_config_schema_definitions" {
		utils.HTTPSuccess(c, http.StatusOK, models.GetECSDefinitions(nil))
		return
	}

	if optionName == "action_config_schema_definitions" {
		utils.HTTPSuccess(c, http.StatusOK, models.GetACSDefinitions(nil))
		return
	}

	options := make(map[string]json.RawMessage)
	if data, err = json.Marshal(module); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error building system module JSON")
		utils.HTTPError(c, srverrors.ErrGetModuleVersionOptionMakeJsonFail, err)
		return
	} else if err = json.Unmarshal(data, &options); err != nil {
		utils.FromContext(c).WithError(err).Errorf("error parsing system module JSON")
		utils.HTTPError(c, srverrors.ErrGetModuleVersionOptionParseJsonFail, err)
		return
	} else if _, ok := options[optionName]; !ok {
		utils.FromContext(c).WithError(err).Errorf("error finding system module option by name")
		utils.HTTPError(c, srverrors.ErrGetModuleVersionOptionNotFound, err)
		return
	}

	utils.HTTPSuccess(c, http.StatusOK, options[optionName])
}
