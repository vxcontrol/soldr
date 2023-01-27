package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/jinzhu/gorm"

	"soldr/pkg/crypto"
)

// ModuleConfig is a proprietary structure to contain module config
// E.x. {"property_1": "some property value", "property_2": 42, ...}
type ModuleConfig map[string]interface{}

// Valid is function to control input/output data
func (mc ModuleConfig) Valid() error {
	if err := validate.Var(mc, "required,dive,keys,solid,endkeys,required,valid"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (mc ModuleConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(mc)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (mc *ModuleConfig) Scan(input interface{}) error {
	return scanFromJSON(input, mc)
}

type ModuleSecureConfig map[string]ModuleSecureParameter

type ModuleSecureParameter struct {
	ServerOnly *bool       `form:"server_only" json:"server_only" validate:"required"`
	Value      interface{} `form:"value" json:"value" validate:"required"`
}

// Valid is function to control input/output data
func (mc ModuleSecureConfig) Valid() error {
	if err := validate.Var(mc, "required,dive,keys,solid,endkeys,valid"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (mc ModuleSecureConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(mc)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (mc *ModuleSecureConfig) Scan(input interface{}) error {
	return scanFromJSON(input, mc)
}

// DependencyItem is a proprietary structure to contain a dependency config
type DependencyItem struct {
	ModuleName       string `form:"module_name,omitempty" json:"module_name,omitempty" validate:"required_unless=Type agent_version,solid=omitempty"`
	MinModuleVersion string `form:"min_module_version,omitempty" json:"min_module_version,omitempty" validate:"semver=omitempty"`
	MinAgentVersion  string `form:"min_agent_version,omitempty" json:"min_agent_version,omitempty" validate:"semverex=omitempty"`
	Type             string `form:"type" json:"type" validate:"required,oneof=to_receive_data to_send_data to_make_action agent_version"`
}

// Valid is function to control input/output data
func (di DependencyItem) Valid() error {
	return validate.Struct(di)
}

// Value is interface function to return current value to store to DB
func (di DependencyItem) Value() (driver.Value, error) {
	b, err := json.Marshal(di)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (di *DependencyItem) Scan(input interface{}) error {
	return scanFromJSON(input, di)
}

// Dependencies is a proprietary structure to contains dependency config of module
// E.x. [{"module_name": "some_module", "type": "to_receive_data"}, ...]
type Dependencies []DependencyItem

// Valid is function to control input/output data
func (d Dependencies) Valid() error {
	if err := validate.Var(d, "required,dive,valid"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (d Dependencies) Value() (driver.Value, error) {
	b, err := json.Marshal(d)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (d *Dependencies) Scan(input interface{}) error {
	return scanFromJSON(input, d)
}

// ActionConfigItem is a proprietary structure to contain an action config
type ActionConfigItem struct {
	Priority uint64                 `form:"priority" json:"priority" validate:"min=1,max=100,numeric,required"`
	Fields   []string               `form:"fields" json:"fields" validate:"unique,solid_fld,omitempty"`
	Config   map[string]interface{} `form:"-" json:"-" validate:"omitempty,dive,keys,solid,endkeys,required"`
}

// Valid is function to control input/output data
func (aci ActionConfigItem) Valid() error {
	return validate.Struct(aci)
}

// Value is interface function to return current value to store to DB
func (aci ActionConfigItem) Value() (driver.Value, error) {
	b, err := json.Marshal(aci)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (aci *ActionConfigItem) Scan(input interface{}) error {
	return scanFromJSON(input, aci)
}

// MarshalJSON is a JSON interface function to make JSON data bytes array from the struct object
func (aci ActionConfigItem) MarshalJSON() ([]byte, error) {
	var err error
	var data []byte
	raw := make(map[string]interface{})
	type actionConfigItem ActionConfigItem
	if data, err = json.Marshal((*actionConfigItem)(&aci)); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for k, v := range aci.Config {
		raw[k] = v
	}
	return json.Marshal(raw)
}

// UnmarshalJSON is a JSON interface function to parse JSON data bytes array and to get struct object
func (aci *ActionConfigItem) UnmarshalJSON(input []byte) error {
	var excludeKeys []string
	tp := reflect.TypeOf(ActionConfigItem{})
	for i := 0; i < tp.NumField(); i++ {
		field := tp.Field(i)
		excludeKeys = append(excludeKeys, strings.Split(field.Tag.Get("json"), ",")[0])
	}
	type actionConfigItem ActionConfigItem
	if err := json.Unmarshal(input, (*actionConfigItem)(aci)); err != nil {
		return err
	}
	raw := make(map[string]interface{})
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	aci.Config = make(map[string]interface{})
	for k, v := range raw {
		if !stringInSlice(k, excludeKeys) {
			aci.Config[k] = v
		}
	}
	return nil
}

// ActionConfig is a proprietary structure to contain actions config of module
// E.x. {"action_id": {"priority": 10, "fields": [...], ...}}
type ActionConfig map[string]ActionConfigItem

// Valid is function to control input/output data
func (ac ActionConfig) Valid() error {
	if err := validate.Var(ac, "required,dive,keys,solid_ext,endkeys,required,valid"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (ac ActionConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(ac)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ac *ActionConfig) Scan(input interface{}) error {
	return scanFromJSON(input, ac)
}

// EventConfigAction is a proprietary structure to describe an event config action
// E.x. {"name": "log_to_db", "module": "this", "priority": 10, fields: []}
type EventConfigAction struct {
	Name       string   `form:"name" json:"name" validate:"required,solid_ext"`
	ModuleName string   `form:"module_name" json:"module_name" validate:"required,solid"`
	Priority   uint64   `form:"priority" json:"priority" validate:"min=1,max=100,numeric,required"`
	Fields     []string `form:"fields" json:"fields" validate:"unique,solid_fld,omitempty"`
}

// Valid is function to control input/output data
func (eca EventConfigAction) Valid() error {
	return validate.Struct(eca)
}

// Value is interface function to return current value to store to DB
func (eca EventConfigAction) Value() (driver.Value, error) {
	b, err := json.Marshal(eca)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (eca *EventConfigAction) Scan(input interface{}) error {
	return scanFromJSON(input, eca)
}

// EventConfigSeq is a proprietary structure to describe one of events sequence
// E.x. {"name": "log_to_db", "type": "db"}
type EventConfigSeq struct {
	Name     string `form:"name" json:"name" validate:"required,solid_ext"`
	MinCount uint64 `form:"min_count" json:"min_count" validate:"min=1,numeric,required"`
}

// Valid is function to control input/output data
func (ecs EventConfigSeq) Valid() error {
	return validate.Struct(ecs)
}

// Value is interface function to return current value to store to DB
func (ecs EventConfigSeq) Value() (driver.Value, error) {
	b, err := json.Marshal(ecs)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ecs *EventConfigSeq) Scan(input interface{}) error {
	return scanFromJSON(input, ecs)
}

// EventConfigItem is a proprietary structure to contain an event config
// It has to "type" and "actions" keys for atomic events
type EventConfigItem struct {
	Type     string                 `form:"type" json:"type" validate:"oneof=atomic aggregation correlation,required"`
	Fields   []string               `form:"fields" json:"fields" validate:"unique,solid_fld,omitempty"`
	Actions  []EventConfigAction    `form:"actions" json:"actions" validate:"required,dive,valid"`
	Seq      []EventConfigSeq       `form:"seq,omitempty" json:"seq,omitempty" validate:"required_with=GroupBy,omitempty,unique,dive,required,valid"`
	GroupBy  []string               `form:"group_by,omitempty" json:"group_by,omitempty" validate:"required_with=Seq,unique,solid_fld,omitempty"`
	MaxCount uint64                 `form:"max_count,omitempty" json:"max_count,omitempty" validate:"min=0,max=10000000,numeric,omitempty"`
	MaxTime  uint64                 `form:"max_time,omitempty" json:"max_time,omitempty" validate:"min=0,max=10000000,numeric,omitempty"`
	Config   map[string]interface{} `form:"-" json:"-" validate:"omitempty,dive,keys,solid,endkeys,required"`
}

// Valid is function to control input/output data
func (eci EventConfigItem) Valid() error {
	return validate.Struct(eci)
}

// Value is interface function to return current value to store to DB
func (eci EventConfigItem) Value() (driver.Value, error) {
	b, err := json.Marshal(eci)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (eci *EventConfigItem) Scan(input interface{}) error {
	return scanFromJSON(input, eci)
}

// MarshalJSON is a JSON interface function to make JSON data bytes array from the struct object
func (eci EventConfigItem) MarshalJSON() ([]byte, error) {
	var err error
	var data []byte
	raw := make(map[string]interface{})
	type eventConfigItem EventConfigItem
	if data, err = json.Marshal((*eventConfigItem)(&eci)); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	for k, v := range eci.Config {
		raw[k] = v
	}
	if eci.Type != "atomic" {
		raw["max_count"] = eci.MaxCount
		raw["max_time"] = eci.MaxTime
		raw["group_by"] = eci.GroupBy
		raw["seq"] = eci.Seq
	} else {
		delete(raw, "max_count")
		delete(raw, "max_time")
		delete(raw, "group_by")
		delete(raw, "seq")
	}
	return json.Marshal(raw)
}

// UnmarshalJSON is a JSON interface function to parse JSON data bytes array and to get struct object
func (eci *EventConfigItem) UnmarshalJSON(input []byte) error {
	var excludeKeys []string
	tp := reflect.TypeOf(EventConfigItem{})
	for i := 0; i < tp.NumField(); i++ {
		field := tp.Field(i)
		excludeKeys = append(excludeKeys, strings.Split(field.Tag.Get("json"), ",")[0])
	}
	type eventConfigItem EventConfigItem
	if err := json.Unmarshal(input, (*eventConfigItem)(eci)); err != nil {
		return err
	}
	raw := make(map[string]interface{})
	if err := json.Unmarshal(input, &raw); err != nil {
		return err
	}
	eci.Config = make(map[string]interface{})
	for k, v := range raw {
		if !stringInSlice(k, excludeKeys) {
			eci.Config[k] = v
		}
	}
	return nil
}

// EventConfig is a proprietary structure to contain events config of module
// E.x. {"event_id": {"type": "atomic|aggregation|correlation", "actions": [...], ...}}
type EventConfig map[string]EventConfigItem

// Valid is function to control input/output data
func (ec EventConfig) Valid() error {
	if err := validate.Var(ec, "required,dive,keys,solid_ext,endkeys,required,valid"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (ec EventConfig) Value() (driver.Value, error) {
	b, err := json.Marshal(ec)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ec *EventConfig) Scan(input interface{}) error {
	return scanFromJSON(input, ec)
}

// ChangelogDesc is model to contain description of a version on a language
type ChangelogDesc struct {
	Date        string `form:"date" json:"date" validate:"cldate,required"`
	Title       string `form:"title" json:"title" validate:"max=300,required"`
	Description string `form:"description" json:"description" validate:"max=10000,required"`
}

// Valid is function to control input/output data
func (cld ChangelogDesc) Valid() error {
	return validate.Struct(cld)
}

// Value is interface function to return current value to store to DB
func (cld ChangelogDesc) Value() (driver.Value, error) {
	b, err := json.Marshal(cld)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (cld *ChangelogDesc) Scan(input interface{}) error {
	return scanFromJSON(input, cld)
}

// ChangelogVersion is a proprietary structure to contain changelog of module version
// E.x. {"en": {...}, "ru": {...}}
type ChangelogVersion map[string]ChangelogDesc

// Valid is function to control input/output data
func (clv ChangelogVersion) Valid() error {
	if err := validate.Var(clv, "required,len=2,dive,keys,oneof=ru en,endkeys,valid"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (clv ChangelogVersion) Value() (driver.Value, error) {
	b, err := json.Marshal(clv)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (clv *ChangelogVersion) Scan(input interface{}) error {
	return scanFromJSON(input, clv)
}

// Changelog is a proprietary structure to contain changelog of module
// E.x. {"0.1.0": {"en": {...}, "ru": {...}}}
type Changelog map[string]ChangelogVersion

// Valid is function to control input/output data
func (cl Changelog) Valid() error {
	if err := validate.Var(cl, "required,dive,keys,semver,max=10,endkeys,valid"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (cl Changelog) Value() (driver.Value, error) {
	b, err := json.Marshal(cl)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (cl *Changelog) Scan(input interface{}) error {
	return scanFromJSON(input, cl)
}

// LocaleDesc is model to contain description of something on a language
type LocaleDesc struct {
	Title       string `form:"title" json:"title" validate:"max=300,required"`
	Description string `form:"description" json:"description" validate:"max=10000"`
}

// Valid is function to control input/output data
func (ld LocaleDesc) Valid() error {
	return validate.Struct(ld)
}

// Value is interface function to return current value to store to DB
func (ld LocaleDesc) Value() (driver.Value, error) {
	b, err := json.Marshal(ld)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ld *LocaleDesc) Scan(input interface{}) error {
	return scanFromJSON(input, ld)
}

// ModuleLocaleDesc is a proprietary structure to contain multilanguage of module locale
// E.x. {"en": {...}, "ru": {...}}
type ModuleLocaleDesc map[string]LocaleDesc

// Valid is function to control input/output data
func (mld ModuleLocaleDesc) Valid() error {
	if err := validate.Var(mld, "required,len=2,dive,keys,oneof=ru en,endkeys,valid"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (mld ModuleLocaleDesc) Value() (driver.Value, error) {
	b, err := json.Marshal(mld)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (mld *ModuleLocaleDesc) Scan(input interface{}) error {
	return scanFromJSON(input, mld)
}

// Locale is a proprietary structure to contain locale of module
type Locale struct {
	Module       ModuleLocaleDesc                       `form:"module" json:"module" validate:"required,valid"`
	Config       map[string]ModuleLocaleDesc            `form:"config" json:"config" validate:"required,dive,keys,solid,endkeys,valid"`
	SecureConfig map[string]ModuleLocaleDesc            `form:"secure_config" json:"secure_config" validate:"dive,keys,solid,endkeys,valid"`
	Fields       map[string]ModuleLocaleDesc            `form:"fields" json:"fields" validate:"required,dive,keys,max=100,solid_fld,endkeys,valid"`
	Actions      map[string]ModuleLocaleDesc            `form:"actions" json:"actions" validate:"required,dive,keys,max=100,solid_ext,endkeys,valid"`
	Events       map[string]ModuleLocaleDesc            `form:"events" json:"events" validate:"required,dive,keys,max=100,solid_ext,endkeys,valid"`
	ActionConfig map[string]map[string]ModuleLocaleDesc `form:"action_config" json:"action_config" validate:"required,dive,keys,solid_ext,endkeys,required,dive,keys,solid,endkeys,valid"`
	EventConfig  map[string]map[string]ModuleLocaleDesc `form:"event_config" json:"event_config" validate:"required,dive,keys,solid_ext,endkeys,required,dive,keys,solid,endkeys,valid"`
	Tags         map[string]ModuleLocaleDesc            `form:"tags" json:"tags" validate:"required,dive,keys,max=30,solid_ext,endkeys,valid"`
}

// Valid is function to control input/output data
func (l Locale) Valid() error {
	return validate.Struct(l)
}

// Value is interface function to return current value to store to DB
func (l Locale) Value() (driver.Value, error) {
	b, err := json.Marshal(l)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (l *Locale) Scan(input interface{}) error {
	return scanFromJSON(input, l)
}

// ModuleInfoOS is a proprietary structure to contain module OS list
// E.x. {"windows": ["386", "amd64"], ...}
type ModuleInfoOS map[string][]string

// Valid is function to control input/output data
func (mios ModuleInfoOS) Valid() error {
	if err := validate.Var(mios, "min=1,dive,keys,oneof=windows linux darwin,endkeys,min=1,unique,required,dive,oneof=386 amd64"); err != nil {
		return err
	}
	return nil
}

// Value is interface function to return current value to store to DB
func (mios ModuleInfoOS) Value() (driver.Value, error) {
	b, err := json.Marshal(mios)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (mios *ModuleInfoOS) Scan(input interface{}) error {
	return scanFromJSON(input, mios)
}

// SemVersion is a proprietary structure to contain semantic version as JSON
// E.x. {"major": 1, "minor": 0, "patch": 2}
type SemVersion struct {
	Major uint64 `form:"major" json:"major" validate:"min=0"`
	Minor uint64 `form:"minor" json:"minor" validate:"min=0"`
	Patch uint64 `form:"patch" json:"patch" validate:"min=0"`
}

// String is function for implement generic interface to convert value to string
func (sv *SemVersion) String() string {
	return fmt.Sprintf("%d.%d.%d", sv.Major, sv.Minor, sv.Patch)
}

// Valid is function to control input/output data
func (sv SemVersion) Valid() error {
	if err := validate.Var(sv.String(), "max=10"); err != nil {
		return err
	}
	return validate.Struct(sv)
}

// Value is interface function to return current value to store to DB
func (sv SemVersion) Value() (driver.Value, error) {
	b, err := json.Marshal(sv)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (sv *SemVersion) Scan(input interface{}) error {
	return scanFromJSON(input, sv)
}

// ModuleInfo is model to contain general module information
type ModuleInfo struct {
	Name     string       `form:"name" json:"name" validate:"required,max=255,solid"`
	Template string       `form:"template" json:"template" validate:"required,oneof=generic empty collector detector responder custom"`
	Version  SemVersion   `form:"version" json:"version" validate:"required,valid"`
	OS       ModuleInfoOS `form:"os" json:"os" validate:"required,valid"`
	System   bool         `form:"system" json:"system" validate:""`
	Actions  []string     `form:"actions" json:"actions" validate:"required,solid_ext,max=50,unique,dive,max=100"`
	Events   []string     `form:"events" json:"events" validate:"required,solid_ext,max=500,unique,dive,max=100"`
	Fields   []string     `form:"fields" json:"fields" validate:"required,solid_fld,max=300,unique,dive,max=100"`
	Tags     []string     `form:"tags" json:"tags" validate:"required,solid_ext,max=20,unique,dive,max=30"`
}

// Valid is function to control input/output data
func (mi ModuleInfo) Valid() error {
	return validate.Struct(mi)
}

// Value is interface function to return current value to store to DB
func (mi ModuleInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(mi)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (mi *ModuleInfo) Scan(input interface{}) error {
	return scanFromJSON(input, mi)
}

// ModuleS is model to contain system module information from global DB
type ModuleS struct {
	ID                  uint64             `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	ServiceType         string             `form:"service_type" json:"service_type" validate:"oneof=vxmonitor" gorm:"type:ENUM('vxmonitor');NOT NULL"`
	TenantID            uint64             `form:"tenant_id" json:"tenant_id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
	ConfigSchema        Schema             `form:"config_schema" json:"config_schema" validate:"required" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	DefaultConfig       ModuleConfig       `form:"default_config" json:"default_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	SecureConfigSchema  Schema             `form:"secure_config_schema" json:"secure_config_schema" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	SecureDefaultConfig ModuleSecureConfig `form:"secure_default_config" json:"secure_default_config" validate:"valid" gorm:"type:JSON;NOT NULL"`
	StaticDependencies  Dependencies       `form:"static_dependencies" json:"static_dependencies" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	FieldsSchema        Schema             `form:"fields_schema" json:"fields_schema" validate:"required" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	ActionConfigSchema  Schema             `form:"action_config_schema" json:"action_config_schema" validate:"required" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	EventConfigSchema   Schema             `form:"event_config_schema" json:"event_config_schema" validate:"required" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	DefaultActionConfig ActionConfig       `form:"default_action_config" json:"default_action_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	DefaultEventConfig  EventConfig        `form:"default_event_config" json:"default_event_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Changelog           Changelog          `form:"changelog" json:"changelog" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Locale              Locale             `form:"locale" json:"locale" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Info                ModuleInfo         `form:"info" json:"info" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	State               string             `form:"state" json:"state" validate:"oneof=draft release" gorm:"type:ENUM('draft','release');NOT NULL"`
	LastUpdate          time.Time          `form:"last_update,omitempty" json:"last_update,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name string to guaranty use correct table
func (ms *ModuleS) TableName() string {
	return "modules"
}

// ToModuleA receive all properties from ModuleS object and return ModuleA object
func (ms *ModuleS) ToModuleA() ModuleA {
	secureCurrentConfig := make(ModuleSecureConfig)
	for k, v := range ms.SecureDefaultConfig {
		secureCurrentConfig[k] = v
	}

	return ModuleA{
		Status:              "joined",
		ConfigSchema:        ms.ConfigSchema,
		DefaultConfig:       ms.DefaultConfig,
		CurrentConfig:       ms.DefaultConfig,
		SecureConfigSchema:  ms.SecureConfigSchema,
		SecureDefaultConfig: ms.SecureDefaultConfig,
		SecureCurrentConfig: secureCurrentConfig,
		StaticDependencies:  ms.StaticDependencies,
		DynamicDependencies: Dependencies{},
		FieldsSchema:        ms.FieldsSchema,
		ActionConfigSchema:  ms.ActionConfigSchema,
		EventConfigSchema:   ms.EventConfigSchema,
		DefaultActionConfig: ms.DefaultActionConfig,
		CurrentActionConfig: ms.DefaultActionConfig,
		DefaultEventConfig:  ms.DefaultEventConfig,
		CurrentEventConfig:  ms.DefaultEventConfig,
		Changelog:           ms.Changelog,
		Locale:              ms.Locale,
		Info:                ms.Info,
		State:               ms.State,
		LastModuleUpdate:    ms.LastUpdate,
	}
}

// ToModuleAShort receive all properties from ModuleS object and return ModuleAShort object
func (ms *ModuleS) ToModuleAShort() ModuleAShort {
	return ModuleAShort{
		StaticDependencies:  ms.StaticDependencies,
		DynamicDependencies: Dependencies{},
		Locale:              ms.Locale,
		Info:                ms.Info,
		State:               ms.State,
		LastModuleUpdate:    ms.LastUpdate,
	}
}

// ToModuleSShort receive all properties from ModuleS object and return ModuleSShort object
func (ms *ModuleS) ToModuleSShort() ModuleSShort {
	return ModuleSShort{
		ID:         ms.ID,
		Changelog:  ms.Changelog,
		Locale:     ms.Locale,
		Info:       ms.Info,
		State:      ms.State,
		LastUpdate: ms.LastUpdate,
	}
}

// Valid is function to control input/output data
func (ms ModuleS) Valid() error {
	return validate.Struct(ms)
}

// Validate is function to use callback to control input/output data
func (ms ModuleS) Validate(db *gorm.DB) {
	if err := ms.Valid(); err != nil {
		db.AddError(err)
	}
}

// IsEncrypted validates if module secure parameters are encrypted with db.DBEncryptor.
func (ms ModuleS) IsEncrypted(encryptor crypto.IDBConfigEncryptor) bool {
	return IsConfigEncrypted(encryptor, ms.SecureDefaultConfig)
}

func (ms ModuleS) EncryptSecureParameters(encryptor crypto.IDBConfigEncryptor) error {
	if ms.IsEncrypted(encryptor) {
		return nil
	}

	return EncryptSecureConfig(encryptor, ms.SecureDefaultConfig)
}

func (ms ModuleS) DecryptSecureParameters(encryptor crypto.IDBConfigEncryptor) error {
	if !ms.IsEncrypted(encryptor) {
		return nil
	}

	return DecryptSecureConfig(encryptor, ms.SecureDefaultConfig)
}

func EncryptSecureConfig(encryptor crypto.IDBConfigEncryptor, configs ...ModuleSecureConfig) error {
	for _, cfg := range configs {
		for k, v := range cfg {
			b, err := json.Marshal(v.Value)
			if err != nil {
				return err
			}

			encrypted, err := encryptor.EncryptValue(b)
			if err != nil {
				return err
			}

			cfg[k] = ModuleSecureParameter{
				ServerOnly: cfg[k].ServerOnly,
				Value:      encrypted,
			}
		}
	}

	return nil
}

func DecryptSecureConfig(encryptor crypto.IDBConfigEncryptor, configs ...ModuleSecureConfig) error {
	for _, cfg := range configs {
		for k, v := range cfg {
			if v.Value == nil {
				continue
			}

			value, ok := v.Value.(string)
			if !ok {
				return crypto.NewErrDecryptFailed(fmt.Errorf("wrong parameter value format"))
			}

			decrypted, err := encryptor.DecryptValue(value)
			if err != nil {
				return crypto.NewErrDecryptFailed(err)
			}

			var result interface{}
			err = json.Unmarshal(decrypted, &result)
			if err != nil {
				return crypto.NewErrDecryptFailed(err)
			}

			cfg[k] = ModuleSecureParameter{
				ServerOnly: cfg[k].ServerOnly,
				Value:      result,
			}
		}
	}

	return nil
}

func IsConfigEncrypted(encryptor crypto.IDBConfigEncryptor, c ModuleSecureConfig) bool {
	if c == nil || len(c) == 0 {
		return false
	}

	for _, c := range c {
		val := c.Value
		if val == nil {
			continue
		}

		s, ok := val.(string)
		if !ok || s == "" {
			return false
		}

		if !encryptor.IsFormatMatch(s) {
			return false
		}
	}

	return true
}

// ModuleSTenant is model to contain system module information linked with module tenant
type ModuleSTenant struct {
	Tenant  Tenant `form:"tenant,omitempty" json:"tenant,omitempty" gorm:"association_autoupdate:false;association_autocreate:false"`
	ModuleS `form:"" json:""`
}

// Valid is function to control input/output data
func (mst ModuleSTenant) Valid() error {
	if err := mst.Tenant.Valid(); err != nil {
		return err
	}
	return mst.ModuleS.Valid()
}

// Validate is function to use callback to control input/output data
func (mst ModuleSTenant) Validate(db *gorm.DB) {
	if err := mst.Valid(); err != nil {
		db.AddError(err)
	}
}

// ModuleA is model to contain agent module information from instance DB
type ModuleA struct {
	ID                  uint64             `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	PolicyID            uint64             `form:"policy_id" json:"policy_id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
	Status              string             `form:"status" json:"status" validate:"oneof=joined inactive,required" gorm:"type:ENUM('joined','inactive');NOT NULL"`
	JoinDate            time.Time          `form:"join_date,omitempty" json:"join_date,omitempty" validate:"required_with=LastUpdate" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
	ConfigSchema        Schema             `form:"config_schema" json:"config_schema" validate:"required" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	DefaultConfig       ModuleConfig       `form:"default_config" json:"default_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	CurrentConfig       ModuleConfig       `form:"current_config" json:"current_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	SecureConfigSchema  Schema             `form:"secure_config_schema" json:"secure_config_schema" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	SecureDefaultConfig ModuleSecureConfig `form:"secure_default_config" json:"secure_default_config" validate:"valid" gorm:"type:JSON;NOT NULL"`
	SecureCurrentConfig ModuleSecureConfig `form:"secure_current_config" json:"secure_current_config" validate:"valid" gorm:"type:JSON;NOT NULL"`
	StaticDependencies  Dependencies       `form:"static_dependencies" json:"static_dependencies" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	DynamicDependencies Dependencies       `form:"dynamic_dependencies" json:"dynamic_dependencies" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	FieldsSchema        Schema             `form:"fields_schema" json:"fields_schema" validate:"required" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	ActionConfigSchema  Schema             `form:"action_config_schema" json:"action_config_schema" validate:"required" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	EventConfigSchema   Schema             `form:"event_config_schema" json:"event_config_schema" validate:"required" gorm:"type:JSON;NOT NULL" swaggertype:"object"`
	DefaultActionConfig ActionConfig       `form:"default_action_config" json:"default_action_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	CurrentActionConfig ActionConfig       `form:"current_action_config" json:"current_action_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	DefaultEventConfig  EventConfig        `form:"default_event_config" json:"default_event_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	CurrentEventConfig  EventConfig        `form:"current_event_config" json:"current_event_config" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Changelog           Changelog          `form:"changelog" json:"changelog" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Locale              Locale             `form:"locale" json:"locale" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Info                ModuleInfo         `form:"info" json:"info" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	State               string             `form:"state" json:"state" validate:"oneof=draft release" gorm:"type:ENUM('draft','release');NOT NULL"`
	LastModuleUpdate    time.Time          `form:"last_module_update" json:"last_module_update" validate:"required" gorm:"type:DATETIME;NOT NULL"`
	LastUpdate          time.Time          `form:"last_update,omitempty" json:"last_update,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
	DeletedAt           *time.Time         `form:"deleted_at,omitempty" json:"deleted_at,omitempty" sql:"index"`
	FilesChecksums      FilesChecksumsMap  `form:"files_checksums,omitempty" json:"files_checksums,omitempty" validate:"omitempty" gorm:"type:JSON;NOT NULL"`
}

// TableName returns the table name string to guaranty use correct table
func (ma *ModuleA) TableName() string {
	return "modules"
}

// ToModuleAShort receive all properties from ModuleS object and return ModuleAShort object
func (ma *ModuleA) ToModuleAShort() ModuleAShort {
	return ModuleAShort{
		ID:                  ma.ID,
		PolicyID:            ma.PolicyID,
		StaticDependencies:  ma.StaticDependencies,
		DynamicDependencies: ma.DynamicDependencies,
		Locale:              ma.Locale,
		Info:                ma.Info,
		State:               ma.State,
		LastModuleUpdate:    ma.LastModuleUpdate,
		LastUpdate:          ma.LastUpdate,
		DeletedAt:           ma.DeletedAt,
	}
}

// ToModuleSShort receive all properties from ModuleS object and return ModuleSShort object
func (ma *ModuleA) ToModuleSShort() ModuleSShort {
	return ModuleSShort{
		Changelog:  ma.Changelog,
		Locale:     ma.Locale,
		Info:       ma.Info,
		State:      ma.State,
		LastUpdate: ma.LastModuleUpdate,
	}
}

// FromModuleS receive all properties from ModuleS object to current object
func (ma *ModuleA) FromModuleS(ms *ModuleS) {
	*ma = ms.ToModuleA()
}

// BeforeDelete hook defined for cascade delete
func (ma *ModuleA) BeforeDelete(db *gorm.DB) error {
	return db.Unscoped().Where("module_id = ?", ma.ID).Delete(&Event{}).Error
}

// Valid is function to control input/output data
func (ma ModuleA) Valid() error {
	return validate.Struct(ma)
}

// Validate is function to use callback to control input/output data
func (ma ModuleA) Validate(db *gorm.DB) {
	if err := ma.Valid(); err != nil {
		db.AddError(err)
	}
}

func (ma ModuleA) ValidateEncryption(encryptor crypto.IDBConfigEncryptor) error {
	var (
		msg string
		err error
	)
	if ma.SecureDefaultConfig != nil && len(ma.SecureDefaultConfig) > 0 {
		if !IsConfigEncrypted(encryptor, ma.SecureDefaultConfig) {
			msg += "SecureDefaultConfig not encrypted; "
		}
	}

	if ma.SecureCurrentConfig != nil && len(ma.SecureCurrentConfig) > 0 {
		if !IsConfigEncrypted(encryptor, ma.SecureCurrentConfig) {
			msg += "SecureCurrentConfig not encrypted"
		}
	}

	if msg != "" {
		err = fmt.Errorf(msg)
	}

	return err
}

// IsEncrypted validates if module secure parameters are base64 values, meaning they're encrypted with IDBConfigEncryptor.
func (ma ModuleA) IsEncrypted(encryptor crypto.IDBConfigEncryptor) bool {
	return IsConfigEncrypted(encryptor, ma.SecureDefaultConfig) ||
		IsConfigEncrypted(encryptor, ma.SecureCurrentConfig)
}

func (ma ModuleA) EncryptSecureParameters(encryptor crypto.IDBConfigEncryptor) error {
	if ma.IsEncrypted(encryptor) {
		return nil
	}

	return EncryptSecureConfig(encryptor,
		ma.SecureDefaultConfig,
		ma.SecureCurrentConfig,
	)
}

func (ma ModuleA) DecryptSecureParameters(encryptor crypto.IDBConfigEncryptor) error {
	if !ma.IsEncrypted(encryptor) {
		return nil
	}

	return DecryptSecureConfig(encryptor,
		ma.SecureDefaultConfig,
		ma.SecureCurrentConfig,
	)
}

type FilesChecksumsMap map[string]FileChecksum

type FileChecksum struct {
	Sha256 string `json:"sha256"`
}

// Value is interface function to return current value to store to DB
func (mi FilesChecksumsMap) Value() (driver.Value, error) {
	b, err := json.Marshal(mi)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (mi *FilesChecksumsMap) Scan(input interface{}) error {
	return scanFromJSON(input, mi)
}

// ModuleAShort is model to contain short agent module information to return it in details
type ModuleAShort struct {
	ID                  uint64       `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	PolicyID            uint64       `form:"policy_id" json:"policy_id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
	StaticDependencies  Dependencies `form:"static_dependencies" json:"static_dependencies" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	DynamicDependencies Dependencies `form:"dynamic_dependencies" json:"dynamic_dependencies" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Locale              Locale       `form:"locale" json:"locale" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Info                ModuleInfo   `form:"info" json:"info" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	State               string       `form:"state" json:"state" validate:"oneof=draft release" gorm:"type:ENUM('draft','release');NOT NULL"`
	LastModuleUpdate    time.Time    `form:"last_module_update" json:"last_module_update" validate:"required" gorm:"type:DATETIME;NOT NULL"`
	LastUpdate          time.Time    `form:"last_update,omitempty" json:"last_update,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
	DeletedAt           *time.Time   `form:"deleted_at,omitempty" json:"deleted_at,omitempty" sql:"index"`
}

// TableName returns the table name string to guaranty use correct table
func (mash *ModuleAShort) TableName() string {
	return "modules"
}

// FromModuleS receive all properties from ModuleS object to current object
func (mash *ModuleAShort) FromModuleS(ms *ModuleS) {
	*mash = ms.ToModuleAShort()
}

// FromModuleA receive all properties from ModuleA object to current object
func (mash *ModuleAShort) FromModuleA(ma *ModuleA) {
	*mash = ma.ToModuleAShort()
}

// BeforeDelete hook defined for cascade delete
func (mash *ModuleAShort) BeforeDelete(db *gorm.DB) error {
	return db.Unscoped().Where("module_id = ?", mash.ID).Delete(&Event{}).Error
}

// Valid is function to control input/output data
func (mash ModuleAShort) Valid() error {
	return validate.Struct(mash)
}

// Validate is function to use callback to control input/output data
func (mash ModuleAShort) Validate(db *gorm.DB) {
	if err := mash.Valid(); err != nil {
		db.AddError(err)
	}
}

// ModuleSShort is model to contain short system module information to return it in details
type ModuleSShort struct {
	ID         uint64     `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Changelog  Changelog  `form:"changelog" json:"changelog" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Locale     Locale     `form:"locale" json:"locale" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Info       ModuleInfo `form:"info" json:"info" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	State      string     `form:"state" json:"state" validate:"oneof=draft release" gorm:"type:ENUM('draft','release');NOT NULL"`
	LastUpdate time.Time  `form:"last_update,omitempty" json:"last_update,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name string to guaranty use correct table
func (mssh *ModuleSShort) TableName() string {
	return "modules"
}

// FromModuleS receive all properties from ModuleS object to current object
func (mssh *ModuleSShort) FromModuleS(ms *ModuleS) {
	*mssh = ms.ToModuleSShort()
}

// FromModuleA receive all properties from ModuleA object to current object
func (mssh *ModuleSShort) FromModuleA(ma *ModuleA) {
	*mssh = ma.ToModuleSShort()
}

// BeforeDelete hook defined for cascade delete
func (mssh *ModuleSShort) BeforeDelete(db *gorm.DB) error {
	return db.Unscoped().Where("module_id = ?", mssh.ID).Delete(&Event{}).Error
}

// Valid is function to control input/output data
func (mssh ModuleSShort) Valid() error {
	return validate.Struct(mssh)
}

// Validate is function to use callback to control input/output data
func (mssh ModuleSShort) Validate(db *gorm.DB) {
	if err := mssh.Valid(); err != nil {
		db.AddError(err)
	}
}

// ModuleDependency is a proprietary structure to contain module dependency with status
type ModuleDependency struct {
	Status         bool `form:"status" json:"status" validate:""`
	DependencyItem `form:"" json:""`
}

// Valid is function to control input/output data
func (md ModuleDependency) Valid() error {
	if err := md.DependencyItem.Valid(); err != nil {
		return err
	}
	return validate.Struct(md)
}

// Value is interface function to return current value to store to DB
func (md ModuleDependency) Value() (driver.Value, error) {
	b, err := json.Marshal(md)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (md *ModuleDependency) Scan(input interface{}) error {
	return scanFromJSON(input, md)
}
