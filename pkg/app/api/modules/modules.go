package modules

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"

	"github.com/jinzhu/copier"
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/models"
	"soldr/pkg/app/api/utils"
	"soldr/pkg/crypto"
	"soldr/pkg/storage"
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

// Template is container for all module files
type Template map[string]map[string][]byte

func joinPath(args ...string) string {
	tpath := filepath.Join(args...)
	return strings.Replace(tpath, "\\", "/", -1)
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
