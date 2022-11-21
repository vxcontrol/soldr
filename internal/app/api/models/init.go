package models

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"

	"github.com/go-playground/validator/v10"
	"github.com/jinzhu/copier"

	"soldr/internal/app/api/utils/dbencryptor"
	"soldr/internal/crypto"
)

const (
	solidRegexString    = "^[a-z0-9_\\-]+$"
	solidExtRegexString = "^[a-zA-Z0-9_]+$"
	solidFldRegexString = "^[a-zA-Z0-9_\\.]+$"
	solidRuRegexString  = "^[a-zA-Zа-яА-Я0-9_\\-]+$"
	clDateRegexString   = "^[0-9]{2}[.-][0-9]{2}[.-][0-9]{4}$"
	semverRegexString   = "^[0-9]+\\.[0-9]+(\\.[0-9]+)?$"
	semverexRegexString = "^(v)?[0-9]+\\.[0-9]+(\\.[0-9]+)?(\\.[0-9]+)?(-[a-zA-Z0-9]+)?$"
	locKeyRegexString   = "^[A-Z][a-zA-Z0-9]*(\\.[A-Z][a-zA-Z0-9]*)*$"
	actKeyRegexString   = "^[a-z0-9_\\-]+\\.[a-zA-Z0-9_]+$"
)

var (
	validate            *validator.Validate
	validationEncryptor crypto.IDBConfigEncryptor
)

func init() {
	validationEncryptor = dbencryptor.NewSecureConfigEncryptor(dbencryptor.GetKey)
}

func GetValidator() *validator.Validate {
	return validate
}

// IValid is interface to control all models from user code
type IValid interface {
	Valid() error
}

// GetACSDefinitions is function to return Action Config Schema definitions defaults
func GetACSDefinitions(defs Definitions) map[string]*Type {
	if defs == nil {
		defs = make(Definitions)
	}

	defs["base.action"] = &Type{
		Type: "object",
		Properties: map[string]*Type{
			"priority": {
				Type:    "integer",
				Minimum: 1,
				Maximum: 100,
			},
			"fields": {
				Type: "array",
				Items: &Type{
					Type: "string",
				},
				UniqueItems: true,
			},
		},
		AdditionalProperties: []byte("true"),
		Required:             []string{"priority", "fields"},
	}

	return defs
}

// GetECSDefinitions is function to return Event Config Schema definitions defaults
func GetECSDefinitions(defs Definitions) map[string]*Type {
	eventTypes := []string{"atomic", "aggregation", "correlation"}
	if defs == nil {
		defs = make(Definitions)
	}

	defs["fields"] = &Type{
		Type: "array",
		Items: &Type{
			Type: "string",
		},
		UniqueItems: true,
	}
	defs["actions"] = &Type{
		Type: "array",
		Items: &Type{
			Type: "object",
			Properties: map[string]*Type{
				"name": {
					Type: "string",
				},
				"module_name": {
					Type: "string",
				},
				"priority": {
					Type:    "integer",
					Minimum: 1,
					Maximum: 100,
				},
				"fields": {
					Ref: "#/definitions/fields",
				},
			},
			AdditionalProperties: []byte("false"),
			Required:             []string{"name", "module_name", "priority", "fields"},
		},
	}

	for _, eventType := range eventTypes {
		defs["types."+eventType] = &Type{
			Type:    "string",
			Default: eventType,
			Enum:    []interface{}{eventType},
		}
	}
	defs["events.atomic"] = &Type{
		Type: "object",
		Properties: map[string]*Type{
			"type": {
				Ref: "#/definitions/types.atomic",
			},
			"actions": {
				Ref: "#/definitions/actions",
			},
			"fields": {
				Ref: "#/definitions/fields",
			},
		},
		Required: []string{"type", "actions", "fields"},
	}
	defs["events.complex"] = &Type{
		Type: "object",
		Properties: map[string]*Type{
			"type": {
				Type: "string",
			},
			"actions": {
				Ref: "#/definitions/actions",
			},
			"fields": {
				Ref: "#/definitions/fields",
			},
			"seq": {
				Type:     "array",
				MinItems: 1,
				Items: &Type{
					Type: "object",
					Properties: map[string]*Type{
						"name": {
							Type: "string",
						},
						"min_count": {
							Type:    "integer",
							Minimum: 1,
						},
					},
					Required: []string{"name", "min_count"},
				},
			},
			"group_by": {
				Type:        "array",
				MinItems:    1,
				UniqueItems: true,
				Items: &Type{
					Type: "string",
				},
			},
			"max_count": {
				Type:    "integer",
				Minimum: 0,
			},
			"max_time": {
				Type:    "integer",
				Minimum: 0,
			},
		},
		Required: []string{
			"type",
			"actions",
			"fields",
			"seq",
			"group_by",
			"max_count",
			"max_time",
		},
	}
	defs["events.aggregation"] = &Type{
		AllOf: []*Type{
			{
				Ref: "#/definitions/events.complex",
			},
			{
				Type: "object",
				Properties: map[string]*Type{
					"type": {
						Ref: "#/definitions/types.aggregation",
					},
					"seq": {
						Type:     "array",
						MaxItems: 1,
					},
				},
				Required: []string{"type", "seq"},
			},
		},
	}
	defs["events.correlation"] = &Type{
		AllOf: []*Type{
			{
				Ref: "#/definitions/events.complex",
			},
			{
				Type: "object",
				Properties: map[string]*Type{
					"type": {
						Ref: "#/definitions/types.correlation",
					},
					"seq": {
						Type:     "array",
						MaxItems: 20,
					},
				},
				Required: []string{"type", "seq"},
			},
		},
	}

	return defs
}

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

func stringsInSlice(a []string, list []string) bool {
	for _, b := range a {
		if !stringInSlice(b, list) {
			return false
		}
	}
	return true
}

func templateValidatorString(regexpString string) validator.Func {
	regexpValue := regexp.MustCompile(regexpString)
	return func(fl validator.FieldLevel) bool {
		field := fl.Field()
		matchString := func(str string) bool {
			if str == "" && fl.Param() == "omitempty" {
				return true
			}
			return regexpValue.MatchString(str)
		}

		switch field.Kind() {
		case reflect.String:
			return matchString(fl.Field().String())
		case reflect.Slice, reflect.Array:
			for i := 0; i < field.Len(); i++ {
				if !matchString(field.Index(i).String()) {
					return false
				}
			}
			return true
		case reflect.Map:
			for _, k := range field.MapKeys() {
				if !matchString(field.MapIndex(k).String()) {
					return false
				}
			}
			return true
		default:
			return false
		}
	}
}

func strongPasswordValidatorString() validator.Func {
	numberRegex := regexp.MustCompile("[0-9]")
	alphaLRegex := regexp.MustCompile("[a-z]")
	alphaURegex := regexp.MustCompile("[A-Z]")
	specRegex := regexp.MustCompile("[!@#$&*]")
	return func(fl validator.FieldLevel) bool {
		field := fl.Field()

		switch field.Kind() {
		case reflect.String:
			password := fl.Field().String()
			return len(password) > 15 || (len(password) >= 8 &&
				numberRegex.MatchString(password) &&
				alphaLRegex.MatchString(password) &&
				alphaURegex.MatchString(password) &&
				specRegex.MatchString(password))
		default:
			return false
		}
	}
}

func emailValidatorString() validator.Func {
	emailRegex := regexp.MustCompile(`^[a-z0-9._%+\-]+@[a-z0-9.\-]+\.[a-z]{2,4}$`)
	return func(fl validator.FieldLevel) bool {
		field := fl.Field()

		switch field.Kind() {
		case reflect.String:
			email := fl.Field().String()
			if email == "vxadmin" {
				return true
			}
			if err := validate.Var(email, "required,uuid"); err == nil {
				return true
			}
			return len(email) > 4 && emailRegex.MatchString(email)
		default:
			return false
		}
	}
}

func deepValidator() validator.Func {
	return func(fl validator.FieldLevel) bool {
		if iv, ok := fl.Field().Interface().(IValid); ok {
			if err := iv.Valid(); err != nil {
				return false
			}
		}

		return true
	}
}

func binaryInfoStructValidator(sl validator.StructLevel) {
	bi, ok := sl.Current().Interface().(BinaryInfo)
	if !ok {
		return
	}

	if len(bi.Files) != len(bi.Chksums) {
		sl.ReportError(bi.Files, "Files", "files",
			"must_eq_len_chksums_value", "")
		sl.ReportError(bi.Chksums, "Chksums", "chksums",
			"must_eq_len_files_value", "")
	} else {
		for _, path := range bi.Files {
			if _, ok := bi.Chksums[path]; !ok {
				sl.ReportError(bi.Files, "Files", "files",
					"must_val_in_chksums_value", path)
			}
		}
	}
}

func eventConfigItemStructValidator(sl validator.StructLevel) {
	eci, ok := sl.Current().Interface().(EventConfigItem)
	if !ok {
		return
	}

	if eci.Type != "atomic" {
		if eci.MaxCount == 0 && eci.MaxTime == 0 {
			// cannot be used infinite correlations and aggregations
			sl.ReportError(eci.MaxCount, "MaxCount", "max_count", "must_limit_event", "")
			sl.ReportError(eci.MaxTime, "MaxTime", "max_time", "must_limit_event", "")
		}
		if len(eci.Seq) == 0 {
			// cannot be used empty correlations and aggregations
			sl.ReportError(eci.Seq, "Seq", "seq", "must_seq_event", "")
		}
		if len(eci.GroupBy) == 0 || len(eci.Fields) == 0 {
			// cannot be used empty grouping on correlations and aggregations
			sl.ReportError(eci.GroupBy, "GroupBy", "group_by", "must_grouping_event", "")
		}
	}
}

func checkModuleTags(sl validator.StructLevel, mod ModuleA) {
	if len(mod.Info.Tags) != len(mod.Locale.Tags) {
		sl.ReportError(mod.Info.Tags, "Tags", "tags",
			"must_eq_len_locale_tags", "")
		sl.ReportError(mod.Locale.Tags, "Tags", "tags",
			"must_eq_len_locale_tags", "")
	} else {
		for _, tid := range mod.Info.Tags {
			if _, ok := mod.Locale.Tags[tid]; !ok {
				sl.ReportError(mod.Locale.Tags, "Tags", "tags",
					"must_val_in_locale_tags", "")
			}
		}
	}
}

func checkModuleFields(sl validator.StructLevel, mod ModuleA) {
	if err := mod.FieldsSchema.Valid(); err != nil {
		sl.ReportError(mod.FieldsSchema, "FieldsSchema", "fields_schema",
			"must_valid_fields_schema", "")
	} else if len(mod.Info.Fields) != len(mod.FieldsSchema.Properties) {
		sl.ReportError(mod.Info.Fields, "Fields", "fields",
			"must_eq_len_locale_action", "")
		sl.ReportError(mod.FieldsSchema.Properties, "FieldsSchema",
			"fields_schema", "must_eq_len_locale_action", "")
	} else if len(mod.Info.Fields) != len(mod.Locale.Fields) {
		sl.ReportError(mod.Info.Fields, "Fields", "fields",
			"must_eq_len_locale_action", "")
		sl.ReportError(mod.Locale.Fields, "Fields", "fields",
			"must_eq_len_locale_action", "")
	} else {
		for fid := range mod.FieldsSchema.Properties {
			if _, ok := mod.Locale.Fields[fid]; !ok {
				sl.ReportError(mod.Locale.Fields, "Fields", "fields",
					"must_val_in_locale_fields", fid)
			}
			if !stringInSlice(fid, mod.Info.Fields) {
				sl.ReportError(mod.Info.Fields, "Fields", "fields",
					"must_val_in_module_info_fields", fid)
			}
		}
	}

	blacklistFields := []string{"info"}
	checkRequiredFields := func(flds []string, position string) {
		for _, fid := range mod.FieldsSchema.Required {
			if !stringInSlice(fid, flds) {
				sl.ReportError(flds, "Fields", "fields",
					"must_val_in_"+position+"_for_required_fields", fid)
			}
		}
	}
	fields := make(map[string]struct{})
	for _, fid := range mod.FieldsSchema.Required {
		fields[fid] = struct{}{}
	}
	for _, aid := range mod.Info.Actions {
		if act, ok := mod.DefaultActionConfig[aid]; ok {
			for _, fld := range act.Fields {
				fields[fld] = struct{}{}
			}
			checkRequiredFields(act.Fields, "def_action")
		}
		if act, ok := mod.CurrentActionConfig[aid]; ok {
			for _, fld := range act.Fields {
				fields[fld] = struct{}{}
			}
			checkRequiredFields(act.Fields, "cur_action")
		}
	}
	for _, eid := range mod.Info.Events {
		if ev, ok := mod.DefaultEventConfig[eid]; ok {
			for _, fld := range ev.Fields {
				fields[fld] = struct{}{}
			}
			if ev.Type == "atomic" {
				checkRequiredFields(ev.Fields, "def_event")
			}
		}
		if ev, ok := mod.CurrentEventConfig[eid]; ok {
			for _, fld := range ev.Fields {
				fields[fld] = struct{}{}
			}
			if ev.Type == "atomic" {
				checkRequiredFields(ev.Fields, "cur_event")
			}
		}
	}
	if len(mod.FieldsSchema.Properties) > len(fields) {
		sl.ReportError(mod.FieldsSchema, "FieldsSchema", "fields_schema",
			"must_using_all_fields_in_actions_and_events", "")
	} else if len(mod.FieldsSchema.Properties) < len(fields) {
		sl.ReportError(mod.FieldsSchema, "FieldsSchema", "fields_schema",
			"must_contains_all_fields_for_actions_and_events", "")
	} else {
		for fid := range mod.FieldsSchema.Properties {
			if _, ok := fields[fid]; !ok {
				sl.ReportError(mod.Locale.Fields, "Fields", "fields",
					"must_val_in_actions_or_events_fields", fid)
			}
			if stringInSlice(fid, blacklistFields) {
				sl.ReportError(mod.Info.Fields, "Fields", "fields",
					"must_val_not_in_blacklist_fields", fid)
			}
		}
	}
}

func checkModuleActions(sl validator.StructLevel, mod ModuleA) {
	var fields []string
	for fid := range mod.FieldsSchema.Properties {
		fields = append(fields, fid)
	}
	if len(mod.Info.Actions) != len(mod.Locale.Actions) {
		sl.ReportError(mod.Info.Actions, "Actions", "actions",
			"must_eq_len_locale_action", "")
		sl.ReportError(mod.Locale.Actions, "Actions", "actions",
			"must_eq_len_locale_action", "")
	} else if len(mod.Info.Actions) != len(mod.DefaultActionConfig) {
		sl.ReportError(mod.Info.Actions, "Actions", "actions",
			"must_eq_len_default_action_config", "")
		sl.ReportError(mod.DefaultActionConfig, "DefaultActionConfig",
			"default_action_config", "must_eq_len_default_action_config", "")
	} else if len(mod.Info.Actions) != len(mod.CurrentActionConfig) {
		sl.ReportError(mod.Info.Actions, "Actions", "actions",
			"must_eq_len_current_action_config", "")
		sl.ReportError(mod.CurrentActionConfig, "CurrentActionConfig",
			"current_action_config", "must_eq_len_current_action_config", "")
	} else if len(mod.Info.Actions) != len(mod.ActionConfigSchema.Properties) {
		sl.ReportError(mod.Info.Actions, "Actions", "actions",
			"must_eq_len_action_config_schema", "")
		sl.ReportError(mod.ActionConfigSchema.Properties, "Properties",
			"properties", "must_eq_len_action_config_schema", "")
	} else {
		for _, aid := range mod.Info.Actions {
			if _, ok := mod.Locale.Actions[aid]; !ok {
				sl.ReportError(mod.Locale.Actions, "Actions", "actions",
					"must_val_in_locale_action", aid)
			}
			if act, ok := mod.DefaultActionConfig[aid]; !ok {
				sl.ReportError(mod.DefaultActionConfig, "DefaultActionConfig",
					"default_action_config", "must_val_in_default_action_config", aid)
			} else {
				if _, ok := mod.ActionConfigSchema.Properties[aid]; !ok {
					sl.ReportError(mod.ActionConfigSchema.Properties, "Properties",
						"properties", "must_val_in_action_config_schema_from_def", aid)
				}
				if actcl, ok := mod.Locale.ActionConfig[aid]; !ok {
					sl.ReportError(mod.Locale.ActionConfig, "ActionConfig",
						"action_config", "must_val_in_locale_action_config_from_def", aid)
				} else {
					if len(act.Config) != len(actcl) {
						sl.ReportError(mod.Locale.ActionConfig, "ActionConfig",
							"action_config", "must_eq_len_locale_action_config_from_def", aid)
					} else {
						for actcid := range act.Config {
							if _, ok := actcl[actcid]; !ok {
								sl.ReportError(mod.Locale.ActionConfig, "ActionConfig",
									"action_config", "must_opt_val_in_locale_action_config_from_def", actcid)
							}
						}
					}
				}
				for _, fld := range act.Fields {
					if !stringInSlice(fld, fields) {
						sl.ReportError(mod.FieldsSchema, "FieldsSchema",
							"fields_schema", "must_default_action_config_fields_in_fields_schema", fld)
					}
				}
			}
			if act, ok := mod.CurrentActionConfig[aid]; !ok {
				sl.ReportError(mod.CurrentActionConfig, "CurrentActionConfig",
					"current_action_config", "must_val_in_current_action_config", aid)
			} else {
				if _, ok := mod.ActionConfigSchema.Properties[aid]; !ok {
					sl.ReportError(mod.ActionConfigSchema.Properties, "Properties",
						"properties", "must_val_in_action_config_schema_from_cur", aid)
				}
				if actcl, ok := mod.Locale.ActionConfig[aid]; !ok {
					sl.ReportError(mod.Locale.ActionConfig, "ActionConfig",
						"action_config", "must_val_in_locale_action_config_from_cur", aid)
				} else {
					if len(act.Config) != len(actcl) {
						sl.ReportError(mod.Locale.ActionConfig, "ActionConfig",
							"action_config", "must_eq_len_locale_action_config_from_cur", aid)
					} else {
						for actcid := range act.Config {
							if _, ok := actcl[actcid]; !ok {
								sl.ReportError(mod.Locale.ActionConfig, "ActionConfig",
									"action_config", "must_opt_val_in_locale_action_config_from_cur", actcid)
							}
						}
					}
				}
				for _, fld := range act.Fields {
					if !stringInSlice(fld, fields) {
						sl.ReportError(mod.FieldsSchema, "FieldsSchema",
							"fields_schema", "must_current_action_config_fields_in_fields_schema", fld)
					}
				}
			}
		}

		actionConfigSchema := Schema{}
		copier.Copy(&actionConfigSchema, &mod.ActionConfigSchema)
		actionConfigSchema.Definitions = GetACSDefinitions(actionConfigSchema.Definitions)

		if defActionConfig, err := json.Marshal(mod.DefaultActionConfig); err != nil {
			sl.ReportError(mod.DefaultActionConfig, "DefaultActionConfig",
				"default_action_config", "must_json_compile_default_action_config", "")
		} else {
			if res, err := actionConfigSchema.ValidateBytes(defActionConfig); err != nil {
				sl.ReportError(mod.DefaultActionConfig, "DefaultActionConfig",
					"default_action_config", "must_valid_default_action_config_by_check", "")
			} else if !res.Valid() {
				sl.ReportError(mod.DefaultActionConfig, "DefaultActionConfig",
					"default_action_config", "must_valid_default_action_config_by_schema", "")
			}
		}
		if curActionConfig, err := json.Marshal(mod.CurrentActionConfig); err != nil {
			sl.ReportError(mod.CurrentActionConfig, "CurrentActionConfig",
				"current_action_config", "must_json_compile_current_action_config", "")
		} else {
			if res, err := actionConfigSchema.ValidateBytes(curActionConfig); err != nil {
				sl.ReportError(mod.CurrentActionConfig, "CurrentActionConfig",
					"current_action_config", "must_valid_current_action_config_by_check", "")
			} else if !res.Valid() {
				sl.ReportError(mod.CurrentActionConfig, "CurrentActionConfig",
					"current_action_config", "must_valid_current_action_config_by_schema", "")
			}
		}
	}
}

func checkModuleEvents(sl validator.StructLevel, mod ModuleA) {
	checkDependencies := func(modName string) bool {
		if modName == "this" {
			return true
		}
		for _, dep := range mod.DynamicDependencies {
			if dep.Type != "to_make_action" {
				continue
			}
			if dep.ModuleName == modName {
				return true
			}
		}

		return false
	}
	checkActions := func(eid string, ev EventConfigItem) {
		for _, act := range ev.Actions {
			if act.ModuleName == mod.Info.Name {
				sl.ReportError(act.ModuleName, "ModuleName", "module_name",
					"must_not_using_self_action_in_the_module", eid)
			}
			if !checkDependencies(act.ModuleName) {
				sl.ReportError(act.ModuleName, "ModuleName", "module_name",
					"must_val_in_dynamic_dependencies", eid)
			}
			for _, fid := range act.Fields {
				if !stringInSlice(fid, ev.Fields) {
					sl.ReportError(act.Fields, "Fields", "fields",
						"must_val_in_event_fields", eid)
				}
			}
		}
	}
	checkGroupBy := func(eid string, ev EventConfigItem) {
		if len(ev.GroupBy) != len(ev.Fields) {
			sl.ReportError(ev, "GroupBy", "group_by",
				"must_eq_len_group_by_value_to_fields", eid)
			sl.ReportError(ev, "Fields", "fields",
				"must_eq_len_fields_value_to_group_by", "")
		} else {
			for _, fid := range ev.Fields {
				if !stringInSlice(fid, ev.GroupBy) {
					sl.ReportError(ev, "Fields", "fields",
						"must_val_in_group_by", fid)
				}
			}
			for _, fid := range ev.GroupBy {
				if fld, ok := mod.FieldsSchema.Properties[fid]; !ok {
					sl.ReportError(ev, "GroupBy", "group_by",
						"must_val_in_fields_schema", fid)
				} else {
					switch fld.Type {
					case "string":
					case "boolean":
					case "number":
					case "integer":
					default:
						sl.ReportError(ev, "GroupBy", "group_by",
							"must_val_as_a_simple_type_for_grouping", fid)
					}
				}
			}
		}
	}
	checkSeq := func(eid string, ev EventConfigItem, evc EventConfig) {
		for _, sev := range ev.Seq {
			if rev, ok := evc[sev.Name]; !ok {
				sl.ReportError(ev.Seq, "Seq", "seq",
					"must_val_in_event_config_for_event_seq", "")
			} else {
				if !stringsInSlice(ev.Fields, rev.Fields) {
					sl.ReportError(rev.Fields, "Fields", "fields",
						"must_val_in_deps_event_fields_complex_event", "")
				}
			}
		}
	}
	lookupDepInActions := func(modName string) bool {
		for _, ev := range mod.CurrentEventConfig {
			for _, act := range ev.Actions {
				if act.ModuleName == modName {
					return true
				}
			}
		}
		return false
	}
	for _, dep := range mod.DynamicDependencies {
		if dep.Type != "to_make_action" {
			continue
		}
		if !lookupDepInActions(dep.ModuleName) {
			sl.ReportError(dep.ModuleName, "ModuleName", "module_name",
				"must_use_dynamic_dependency_val_in_action_list", dep.Type)
		}
	}
	if len(mod.Info.Events) != len(mod.Locale.Events) {
		sl.ReportError(mod.Info.Events, "Events", "events",
			"must_eq_len_locale_event", "")
		sl.ReportError(mod.Locale.Events, "Events", "events",
			"must_eq_len_locale_event", "")
	} else if len(mod.Info.Events) != len(mod.DefaultEventConfig) {
		sl.ReportError(mod.Info.Events, "Events", "events",
			"must_eq_len_default_event_config", "")
		sl.ReportError(mod.DefaultEventConfig, "DefaultEventConfig",
			"default_event_config", "must_eq_len_default_event_config", "")
	} else if len(mod.Info.Events) != len(mod.CurrentEventConfig) {
		sl.ReportError(mod.Info.Events, "Events", "events",
			"must_eq_len_current_event_config", "")
		sl.ReportError(mod.CurrentEventConfig, "CurrentEventConfig",
			"current_event_config", "must_eq_len_current_event_config", "")
	} else {
		for _, eid := range mod.Info.Events {
			if _, ok := mod.Locale.Events[eid]; !ok {
				sl.ReportError(mod.Locale.Events, "Events", "events",
					"must_val_in_locale_event", eid)
			}
			if ev, ok := mod.DefaultEventConfig[eid]; !ok {
				sl.ReportError(mod.DefaultEventConfig, "DefaultEventConfig",
					"default_event_config", "must_val_in_default_event_config", eid)
			} else {
				checkActions(eid, ev)
				if ev.Type != "atomic" {
					checkGroupBy(eid, ev)
					checkSeq(eid, ev, mod.DefaultEventConfig)
				}
				if _, ok := mod.EventConfigSchema.Properties[eid]; ev.Type == "atomic" && !ok {
					sl.ReportError(mod.EventConfigSchema.Properties, "Properties",
						"properties", "must_val_in_event_config_schema_from_def", eid)
				}
				if evcl, ok := mod.Locale.EventConfig[eid]; ev.Type == "atomic" && !ok {
					sl.ReportError(mod.Locale.EventConfig, "EventConfig",
						"event_config", "must_val_in_locale_event_config_from_def", eid)
				} else {
					if len(ev.Config) != len(evcl) {
						sl.ReportError(mod.Locale.EventConfig, "EventConfig",
							"event_config", "must_eq_len_locale_event_config_from_def", eid)
					} else {
						for evcid := range ev.Config {
							if _, ok := evcl[evcid]; !ok {
								sl.ReportError(mod.Locale.EventConfig, "EventConfig",
									"event_config", "must_opt_val_in_locale_event_config_from_def", evcid)
							}
						}
					}
				}
			}
			if ev, ok := mod.CurrentEventConfig[eid]; !ok {
				sl.ReportError(mod.CurrentEventConfig, "CurrentEventConfig",
					"current_event_config", "must_val_in_current_event_config", eid)
			} else {
				checkActions(eid, ev)
				if ev.Type != "atomic" {
					checkGroupBy(eid, ev)
					checkSeq(eid, ev, mod.CurrentEventConfig)
				}
				if _, ok := mod.EventConfigSchema.Properties[eid]; ev.Type == "atomic" && !ok {
					sl.ReportError(mod.EventConfigSchema.Properties, "Properties",
						"properties", "must_val_in_event_config_schema_from_cur", eid)
				}
				if evcl, ok := mod.Locale.EventConfig[eid]; ev.Type == "atomic" && !ok {
					sl.ReportError(mod.Locale.EventConfig, "EventConfig",
						"event_config", "must_val_in_locale_event_config_from_cur", eid)
				} else {
					if len(ev.Config) != len(evcl) {
						sl.ReportError(mod.Locale.EventConfig, "EventConfig",
							"event_config", "must_eq_len_locale_event_config_from_cur", eid)
					} else {
						for evcid := range ev.Config {
							if _, ok := evcl[evcid]; !ok {
								sl.ReportError(mod.Locale.EventConfig, "EventConfig",
									"event_config", "must_opt_val_in_locale_event_config_from_cur", evcid)
							}
						}
					}
				}
			}
		}

		eventConfigSchema := Schema{}
		copier.Copy(&eventConfigSchema, &mod.EventConfigSchema)
		eventConfigSchema.Definitions = GetECSDefinitions(eventConfigSchema.Definitions)

		if defEventConfig, err := json.Marshal(mod.DefaultEventConfig); err != nil {
			sl.ReportError(mod.DefaultEventConfig, "DefaultEventConfig",
				"default_event_config", "must_json_compile_default_event_config", "")
		} else {
			if res, err := eventConfigSchema.ValidateBytes(defEventConfig); err != nil {
				sl.ReportError(mod.DefaultEventConfig, "DefaultEventConfig",
					"default_event_config", "must_valid_default_event_config_by_check", "")
			} else if !res.Valid() {
				sl.ReportError(mod.DefaultEventConfig, "DefaultEventConfig",
					"default_event_config", "must_valid_default_event_config_by_schema", "")
			}
		}
		if curEventConfig, err := json.Marshal(mod.CurrentEventConfig); err != nil {
			sl.ReportError(mod.CurrentEventConfig, "CurrentEventConfig",
				"current_event_config", "must_json_compile_current_event_config", "")
		} else {
			if res, err := eventConfigSchema.ValidateBytes(curEventConfig); err != nil {
				sl.ReportError(mod.CurrentEventConfig, "CurrentEventConfig",
					"current_event_config", "must_valid_current_event_config_by_check", "")
			} else if !res.Valid() {
				sl.ReportError(mod.CurrentEventConfig, "CurrentEventConfig",
					"current_event_config", "must_valid_current_event_config_by_schema", "")
			}
		}
	}
}

func checkModuleConfig(sl validator.StructLevel, mod ModuleA) {
	if err := mod.ConfigSchema.Valid(); err != nil {
		sl.ReportError(mod.ConfigSchema, "ConfigSchema", "config_schema",
			"must_valid_config_schema", "")
	}
	if len(mod.CurrentConfig) != len(mod.Locale.Config) {
		sl.ReportError(mod.CurrentConfig, "CurrentConfig", "current_config",
			"must_eq_len_locale_event_config_from_cur", "")
		sl.ReportError(mod.Locale.Config, "Config", "config",
			"must_eq_len_locale_event_config_from_cur", "")
	} else {
		for cid := range mod.CurrentConfig {
			if _, ok := mod.Locale.Config[cid]; !ok {
				sl.ReportError(mod.Locale.Config, "Config", "config",
					"must_val_in_locale_config_from_cur", cid)
			}
		}
	}
	if len(mod.DefaultConfig) != len(mod.Locale.Config) {
		sl.ReportError(mod.DefaultConfig, "DefaultConfig", "default_config",
			"must_eq_len_locale_event_config_from_def", "")
		sl.ReportError(mod.Locale.Config, "Config", "config",
			"must_eq_len_locale_event_config_from_def", "")
	} else {
		for cid := range mod.DefaultConfig {
			if _, ok := mod.Locale.Config[cid]; !ok {
				sl.ReportError(mod.Locale.Config, "Config", "config",
					"must_val_in_locale_config_from_def", cid)
			}
		}
	}
	if curModuleConfig, err := json.Marshal(mod.CurrentConfig); err != nil {
		sl.ReportError(mod.CurrentConfig, "CurrentConfig",
			"current_config", "must_json_compile_current_config", "")
	} else {
		if res, err := mod.ConfigSchema.ValidateBytes(curModuleConfig); err != nil {
			sl.ReportError(mod.CurrentConfig, "CurrentConfig",
				"current_config", "must_valid_current_config_by_check", "")
		} else if !res.Valid() {
			sl.ReportError(mod.CurrentConfig, "CurrentConfig",
				"current_config", "must_valid_current_config_by_schema", "")
		}
	}
	if defModuleConfig, err := json.Marshal(mod.DefaultConfig); err != nil {
		sl.ReportError(mod.DefaultConfig, "DefaultConfig",
			"default_config", "must_json_compile_default_config", "")
	} else {
		if res, err := mod.ConfigSchema.ValidateBytes(defModuleConfig); err != nil {
			sl.ReportError(mod.DefaultConfig, "DefaultConfig",
				"default_config", "must_valid_default_config_by_check", "")
		} else if !res.Valid() {
			sl.ReportError(mod.DefaultConfig, "DefaultConfig",
				"default_config", "must_valid_default_config_by_schema", "")
		}
	}
}

func checkModuleSecureConfig(sl validator.StructLevel, mod ModuleA) {
	if err := mod.SecureConfigSchema.Valid(); err != nil {
		sl.ReportError(mod.SecureConfigSchema, "SecureConfigSchema", "secure_config_schema",
			"must_valid_secure_config_schema", "")
	}

	if len(mod.SecureCurrentConfig) != len(mod.Locale.SecureConfig) {
		sl.ReportError(mod.SecureCurrentConfig, "SecureCurrentConfig", "secure_current_config",
			"must_eq_len_locale_secure_config_from_cur", "")
		sl.ReportError(mod.Locale.SecureConfig, "SecureConfig", "secure_config",
			"must_eq_len_locale_secure_config_from_cur", "")
	} else {
		for cid := range mod.SecureCurrentConfig {
			if _, ok := mod.Locale.SecureConfig[cid]; !ok {
				sl.ReportError(mod.Locale.SecureConfig, "SecureConfig", "secure_config",
					"must_val_in_locale_secure_config_from_cur", cid)
			}
		}
	}

	if len(mod.SecureDefaultConfig) != len(mod.Locale.SecureConfig) {
		sl.ReportError(mod.SecureDefaultConfig, "SecureDefaultConfig", "secure_default_config",
			"must_eq_len_locale_sec_config_from_def", "")
		sl.ReportError(mod.Locale.SecureConfig, "SecureConfig", "secure_config",
			"must_eq_len_locale_sec_config_from_def", "")
	} else {
		for cid := range mod.SecureDefaultConfig {
			if _, ok := mod.Locale.SecureConfig[cid]; !ok {
				sl.ReportError(mod.Locale.SecureConfig, "SecureConfig", "secure_config",
					"must_val_in_locale_sec_config_from_def", cid)
			}
		}
	}

	var isEncrypted bool
	if mod.IsEncrypted(validationEncryptor) {
		isEncrypted = true
		err := mod.DecryptSecureParameters(validationEncryptor)
		if err != nil {
			sl.ReportError(mod.SecureCurrentConfig, "SecureConfig",
				"secure_config", "error_decrypting_secure_config", "")
			return
		}
	}

	validateSecureDefaultConfig := func() {
		defaultSecureConfig, err := json.Marshal(mod.SecureDefaultConfig)
		if err != nil {
			sl.ReportError(mod.SecureDefaultConfig, "SecureDefaultConfig",
				"secure_default_config", "must_json_compile_secure_default_config", "")
			return
		}

		res, err := mod.SecureConfigSchema.ValidateBytes(defaultSecureConfig)
		switch {
		case err != nil:
			sl.ReportError(mod.SecureDefaultConfig, "SecureDefaultConfig",
				"secure_default_config", "must_valid_secure_default_config_by_check", "")
		case !res.Valid():
			sl.ReportError(mod.SecureDefaultConfig, "SecureDefaultConfig",
				"secure_default_config", "must_valid_secure_default_config_by_schema", "")
		}
	}

	validateSecureCurrentConfig := func() {
		currentSecureConfig, err := json.Marshal(mod.SecureCurrentConfig)
		if err != nil {
			sl.ReportError(mod.SecureCurrentConfig, "SecureCurrentConfig",
				"secure_current_config", "must_json_compile_secure_current_config", "")
			return
		}

		res, err := mod.SecureConfigSchema.ValidateBytes(currentSecureConfig)
		switch {
		case err != nil:
			sl.ReportError(mod.SecureCurrentConfig, "SecureCurrentConfig",
				"secure_current_config", "must_valid_secure_current_config_by_check", "")
		case !res.Valid():
			sl.ReportError(mod.SecureCurrentConfig, "SecureCurrentConfig",
				"secure_current_config", "must_valid_secure_current_config_by_schema", "")
		}
	}

	validateSecureDefaultConfig()
	validateSecureCurrentConfig()

	if isEncrypted {
		err := mod.EncryptSecureParameters(validationEncryptor)
		if err != nil {
			sl.ReportError(mod.SecureCurrentConfig, "SecureConfig",
				"secure_config", "error_encrypting_secure_config", "")
			return
		}
	}
}

func checkModuleVersion(sl validator.StructLevel, mod ModuleA) {
	if _, ok := mod.Changelog[mod.Info.Version.String()]; !ok {
		sl.ReportError(mod.Info.Version.String(), "Version", "version",
			"must_mod_version_in_changelog", "")
		sl.ReportError(mod.Changelog, "Changelog", "changelog",
			"must_mod_version_in_changelog", "")
	}
}

func systemModuleStructValidator(sl validator.StructLevel) {
	modS, ok := sl.Current().Interface().(ModuleS)
	if !ok {
		return
	}

	modA := modS.ToModuleA()
	checkModuleTags(sl, modA)
	checkModuleFields(sl, modA)
	checkModuleActions(sl, modA)
	checkModuleEvents(sl, modA)
	checkModuleConfig(sl, modA)
	checkModuleSecureConfig(sl, modA)
	checkModuleVersion(sl, modA)
}

func agentModuleStructValidator(sl validator.StructLevel) {
	modA, ok := sl.Current().Interface().(ModuleA)
	if !ok {
		return
	}

	checkModuleTags(sl, modA)
	checkModuleFields(sl, modA)
	checkModuleActions(sl, modA)
	checkModuleEvents(sl, modA)
	checkModuleConfig(sl, modA)
	checkModuleSecureConfig(sl, modA)
	checkModuleVersion(sl, modA)
}

func scanFromJSON(input interface{}, output interface{}) error {
	if v, ok := input.(string); ok {
		return json.Unmarshal([]byte(v), output)
	} else if v, ok := input.([]byte); ok {
		return json.Unmarshal(v, output)
	}
	return fmt.Errorf("unsupported type of input value to scan")
}

func init() {
	validate = validator.New()
	validate.RegisterValidation("solid", templateValidatorString(solidRegexString))
	validate.RegisterValidation("solid_ext", templateValidatorString(solidExtRegexString))
	validate.RegisterValidation("solid_fld", templateValidatorString(solidFldRegexString))
	validate.RegisterValidation("solid_ru", templateValidatorString(solidRuRegexString))
	validate.RegisterValidation("cldate", templateValidatorString(clDateRegexString))
	validate.RegisterValidation("semver", templateValidatorString(semverRegexString))
	validate.RegisterValidation("semverex", templateValidatorString(semverexRegexString))
	validate.RegisterValidation("lockey", templateValidatorString(locKeyRegexString))
	validate.RegisterValidation("actkey", templateValidatorString(actKeyRegexString))
	validate.RegisterValidation("stpass", strongPasswordValidatorString())
	validate.RegisterValidation("vmail", emailValidatorString())
	validate.RegisterValidation("valid", deepValidator())
	validate.RegisterStructValidation(binaryInfoStructValidator, BinaryInfo{})
	validate.RegisterStructValidation(eventConfigItemStructValidator, EventConfigItem{})
	validate.RegisterStructValidation(systemModuleStructValidator, ModuleS{})
	validate.RegisterStructValidation(agentModuleStructValidator, ModuleA{})

	// Check validation interface for all models
	_, _ = reflect.ValueOf(Schema{}).Interface().(IValid)

	_, _ = reflect.ValueOf(Login{}).Interface().(IValid)
	_, _ = reflect.ValueOf(PermissionsService{}).Interface().(IValid)

	_, _ = reflect.ValueOf(User{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Password{}).Interface().(IValid)
	_, _ = reflect.ValueOf(UserRole{}).Interface().(IValid)
	_, _ = reflect.ValueOf(UserTenant{}).Interface().(IValid)
	_, _ = reflect.ValueOf(UserRoleTenant{}).Interface().(IValid)

	_, _ = reflect.ValueOf(Role{}).Interface().(IValid)

	_, _ = reflect.ValueOf(Tenant{}).Interface().(IValid)

	_, _ = reflect.ValueOf(ServiceInfoDB{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ServiceInfoS3{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ServiceInfoServer{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ServiceInfo{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Service{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ServiceTenant{}).Interface().(IValid)

	_, _ = reflect.ValueOf(ModuleConfig{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleSecureParameter{}).Interface().(IValid)
	_, _ = reflect.ValueOf(DependencyItem{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Dependencies{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ActionConfigItem{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ActionConfig{}).Interface().(IValid)
	_, _ = reflect.ValueOf(EventConfigAction{}).Interface().(IValid)
	_, _ = reflect.ValueOf(EventConfigSeq{}).Interface().(IValid)
	_, _ = reflect.ValueOf(EventConfigItem{}).Interface().(IValid)
	_, _ = reflect.ValueOf(EventConfig{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ChangelogDesc{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ChangelogVersion{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Changelog{}).Interface().(IValid)
	_, _ = reflect.ValueOf(LocaleDesc{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleLocaleDesc{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Locale{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleInfoOS{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleInfo{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleS{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleSTenant{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleA{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleAShort{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleSShort{}).Interface().(IValid)
	_, _ = reflect.ValueOf(ModuleDependency{}).Interface().(IValid)

	_, _ = reflect.ValueOf(AgentOS{}).Interface().(IValid)
	_, _ = reflect.ValueOf(AgentNet{}).Interface().(IValid)
	_, _ = reflect.ValueOf(AgentUser{}).Interface().(IValid)
	_, _ = reflect.ValueOf(AgentInfo{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Agent{}).Interface().(IValid)
	_, _ = reflect.ValueOf(AgentGroup{}).Interface().(IValid)
	_, _ = reflect.ValueOf(AgentPolicies{}).Interface().(IValid)
	_, _ = reflect.ValueOf(AgentDependency{}).Interface().(IValid)

	_, _ = reflect.ValueOf(GroupItemLocale{}).Interface().(IValid)
	_, _ = reflect.ValueOf(GroupInfo{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Group{}).Interface().(IValid)
	_, _ = reflect.ValueOf(GroupToPolicy{}).Interface().(IValid)
	_, _ = reflect.ValueOf(GroupPolicies{}).Interface().(IValid)
	_, _ = reflect.ValueOf(GroupDependency{}).Interface().(IValid)

	_, _ = reflect.ValueOf(PolicyItemLocale{}).Interface().(IValid)
	_, _ = reflect.ValueOf(PolicyInfo{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Policy{}).Interface().(IValid)
	_, _ = reflect.ValueOf(PolicyGroups{}).Interface().(IValid)
	_, _ = reflect.ValueOf(PolicyModules{}).Interface().(IValid)
	_, _ = reflect.ValueOf(PolicyDependency{}).Interface().(IValid)

	_, _ = reflect.ValueOf(EventInfo{}).Interface().(IValid)
	_, _ = reflect.ValueOf(Event{}).Interface().(IValid)
	_, _ = reflect.ValueOf(EventModule{}).Interface().(IValid)
	_, _ = reflect.ValueOf(EventAgent{}).Interface().(IValid)
	_, _ = reflect.ValueOf(EventModuleAgent{}).Interface().(IValid)

	_, _ = reflect.ValueOf(OptionsActions{}).Interface().(IValid)
	_, _ = reflect.ValueOf(OptionsEvents{}).Interface().(IValid)
	_, _ = reflect.ValueOf(OptionsFields{}).Interface().(IValid)
	_, _ = reflect.ValueOf(OptionsTags{}).Interface().(IValid)
}
