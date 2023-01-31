package modules

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"
	"github.com/sirupsen/logrus"

	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/storage"
	"soldr/pkg/app/api/utils"
	"soldr/pkg/crypto"
	"soldr/pkg/filestorage"
	"soldr/pkg/filestorage/fs"
	"soldr/pkg/filestorage/s3"
)

// Template is container for all module files
type Template map[string]map[string][]byte

func joinPath(args ...string) string {
	tpath := filepath.Join(args...)
	return strings.Replace(tpath, "\\", "/", -1)
}

func CopyModuleAFilesToInstanceS3(mi *models.ModuleInfo, sv *models.Service) (models.FilesChecksumsMap, error) {
	gS3, err := s3.New(nil)
	if err != nil {
		return nil, errors.New("failed to initialize global S3 driver: " + err.Error())
	}

	mfiles, err := gS3.ReadDirRec(joinPath(mi.Name, mi.Version.String()))
	if err != nil {
		return nil, errors.New("failed to read system module files: " + err.Error())
	}

	fillMissingFileContents("/config", mfiles)

	ufiles, err := gS3.ReadDirRec("utils")
	if err != nil {
		return nil, errors.New("failed to read utils files: " + err.Error())
	}

	iS3, err := s3.New(sv.Info.S3.ToS3ConnParams())
	if err != nil {
		return nil, errors.New("failed to initialize instance S3 driver: " + err.Error())
	}

	if iS3.RemoveDir(joinPath(mi.Name, mi.Version.String())); err != nil {
		return nil, errors.New("failed to remove module directory from instance S3: " + err.Error())
	}

	for fpath, fdata := range mfiles {
		if err := iS3.WriteFile(joinPath(mi.Name, mi.Version.String(), fpath), fdata); err != nil {
			return nil, errors.New("failed to write system module file to S3: " + err.Error())
		}
	}

	for fpath, fdata := range ufiles {
		if err := iS3.WriteFile(joinPath("utils", fpath), fdata); err != nil {
			return nil, errors.New("failed to write utils file to S3: " + err.Error())
		}
	}

	filesChecksum := make(models.FilesChecksumsMap)
	for path, data := range mfiles {
		if strings.HasPrefix(path, "/smodule/") || strings.HasPrefix(path, "/cmodule/") {
			filesChecksum[path] = models.FileChecksum{
				Sha256: calcChecksum(data),
			}
		}
	}

	return filesChecksum, nil
}

func CalcFilesChecksums(files map[string][]byte) models.FilesChecksumsMap {
	chsms := make(models.FilesChecksumsMap, len(files))
	for path, data := range files {
		chsms[path] = models.FileChecksum{
			Sha256: calcChecksum(data),
		}
	}

	return chsms
}

func calcChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
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
	icc := MergeTwoInterfacesBySchema(cc, dc, mcsh)
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
	icc := MergeTwoInterfacesBySchema(cc, dc, mcsh)
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
	icac := MergeTwoInterfacesBySchema(convertToRawInterface(cac), convertToRawInterface(dac), acsh)
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
	rieci := MergeTwoInterfacesBySchema(iceci, ideci, sh)
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

func removeLeadSlash(files map[string][]byte) map[string][]byte {
	rfiles := make(map[string][]byte)
	for name, data := range files {
		rfiles[name[1:]] = data
	}
	return rfiles
}

func ReadDir(s filestorage.Storage, path string) ([]string, error) {
	var files []string
	list, err := s.ListDir(path)
	if err != nil {
		return files, err
	}
	for _, info := range list {
		if info.IsDir() {
			list, err := ReadDir(s, path+"/"+info.Name())
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

func LoadModuleSTemplate(mi *models.ModuleInfo, templatesDir string) (Template, *models.ModuleS, error) {
	fs, err := fs.New()
	if err != nil {
		return nil, nil, errors.New("failed initialize LocalStorage driver: " + err.Error())
	}

	var module *models.ModuleS
	template := make(Template)
	loadModuleDir := func(dir string) (map[string][]byte, error) {
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
	s3, err := s3.New(nil)
	if err != nil {
		return nil, errors.New("failed to initialize LocalStorage driver: " + err.Error())
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
	s3, err := s3.New(nil)
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
	s3, err := s3.New(nil)
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

func RemoveUnusedModuleVersion(iDB *gorm.DB, name, version string, sv *models.Service) error {
	var count int64
	err := iDB.
		Model(&models.ModuleA{}).
		Where("name LIKE ? AND version LIKE ?", name, version).
		Count(&count).Error
	if err != nil {
		return errors.New("failed to get count modules by version")
	}

	if count == 0 {
		s3, err := s3.New(sv.Info.S3.ToS3ConnParams())
		if err != nil {
			logrus.WithError(err).Errorf("error openning connection to S3")
			return err
		}

		if err = s3.RemoveDir(name + "/" + version + "/"); err != nil && err.Error() != "not found" {
			logrus.WithError(err).Errorf("error removing module data from s3")
			return err
		}
	}

	return nil
}

func UpdateDependenciesWhenModuleRemove(iDB *gorm.DB, name string) error {
	var (
		err     error
		modules []models.ModuleA
		incl    = []interface{}{"current_event_config", "dynamic_dependencies"}
	)

	if err = iDB.Find(&modules, "dependencies LIKE ?", `%"`+name+`"%`).Error; err != nil {
		logrus.WithError(err).Errorf("error finding modules by dependencies")
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
			logrus.WithError(err).Errorf("error updating config module")
			return err
		}
	}

	return nil
}

func UpdatePolicyModulesByModuleS(moduleS *models.ModuleS, sv *models.Service, encryptor crypto.IDBConfigEncryptor) error {
	iDB := storage.GetDB(sv.Info.DB.User, sv.Info.DB.Pass, sv.Info.DB.Host,
		strconv.Itoa(int(sv.Info.DB.Port)), sv.Info.DB.Name)
	if iDB == nil {
		logrus.Errorf("error openning connection to instance DB")
		return errors.New("failed to connect to instance DB")
	}
	defer iDB.Close()

	var modules []models.ModuleA
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name LIKE ? AND version LIKE ? AND last_module_update NOT LIKE ?",
			moduleS.Info.Name, moduleS.Info.Version.String(), moduleS.LastUpdate)
	}
	if err := iDB.Scopes(scope).Find(&modules).Error; err != nil {
		return fmt.Errorf("error finding policy modules by name and version '%s' '%s': %w",
			moduleS.Info.Name, moduleS.Info.Version.String(), err)
	} else if len(modules) == 0 {
		return nil
	}

	checksums, err := CopyModuleAFilesToInstanceS3(&moduleS.Info, sv)
	if err != nil {
		return fmt.Errorf("error copying module files to S3: %w", err)
	}

	excl := []string{"policy_id", "status", "join_date", "last_update"}
	for _, moduleA := range modules {
		var err error
		moduleA, err = MergeModuleAConfigFromModuleS(&moduleA, moduleS, encryptor)
		if err != nil {
			return fmt.Errorf("error merging agent module to system module config: %w", err)
		}

		moduleA.FilesChecksums = checksums

		if err := moduleA.Valid(); err != nil {
			return fmt.Errorf("invalid agent module state: %w", err)
		}

		err = moduleA.EncryptSecureParameters(encryptor)
		if err != nil {
			return fmt.Errorf("failed to encrypt module secure config: %w", err)
		}
		if err := iDB.Omit(excl...).Save(&moduleA).Error; err != nil {
			return fmt.Errorf("error updating module into DB: %w", err)
		}
	}

	return nil
}

func ValidateFileOnSave(path string, data []byte) error {
	fileName := filepath.Base(path)
	fileDir := filepath.Base(filepath.Dir(path))
	if utils.StringInSlice(fileDir, []string{"cmodule", "smodule"}) && fileName == "args.json" {
		args := make(map[string][]string)
		if err := json.Unmarshal(data, &args); err != nil {
			return fmt.Errorf("failed to parse the module '%s' args: %w", path, err)
		}
	}
	return nil
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

func GetModuleName(db *gorm.DB, sv *models.Service, tid uint64, name string, version string) (string, error) {
	scope := func(db *gorm.DB) *gorm.DB {
		return db.Where("name = ? AND tenant_id = ? AND service_type = ?", name, tid, sv.Type)
	}

	var module models.ModuleS
	if err := db.Scopes(FilterModulesByVersion(version), scope).Take(&module).Error; err != nil {
		return "", err
	}
	return module.Locale.Module["en"].Title, nil
}

func CompareModulesChanges(moduleIn, moduleDB models.ModuleA, encryptor crypto.IDBConfigEncryptor) ([]bool, error) {
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
