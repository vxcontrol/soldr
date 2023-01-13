package models

import (
	"database/sql/driver"
	"encoding/json"
)

// OptionsActions is a proprietary structure to contain modules actions list
type OptionsActions struct {
	Name       string           `form:"name" json:"name" validate:"required,solid_ext"`
	Config     ActionConfigItem `form:"config" json:"config" validate:"required,valid"`
	Locale     ModuleLocaleDesc `form:"locale" json:"locale" validate:"required,valid"`
	ModuleName string           `form:"module_name" json:"module_name" validate:"required,solid"`
	ModuleOS   ModuleInfoOS     `form:"module_os" json:"module_os" validate:"required,valid"`
}

// Valid is function to control input/output data
func (oa OptionsActions) Valid() error {
	return validate.Struct(oa)
}

// Value is interface function to return current value to store to DB
func (oa OptionsActions) Value() (driver.Value, error) {
	b, err := json.Marshal(oa)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (oa *OptionsActions) Scan(input interface{}) error {
	return scanFromJSON(input, oa)
}

// OptionsEvents is a proprietary structure to contain modules events list
type OptionsEvents struct {
	Name       string           `form:"name" json:"name" validate:"required,solid_ext"`
	Config     EventConfigItem  `form:"config" json:"config" validate:"required,valid"`
	Locale     ModuleLocaleDesc `form:"locale" json:"locale" validate:"required,valid"`
	ModuleName string           `form:"module_name" json:"module_name" validate:"required,solid"`
	ModuleOS   ModuleInfoOS     `form:"module_os" json:"module_os" validate:"required,valid"`
}

// Valid is function to control input/output data
func (oe OptionsEvents) Valid() error {
	return validate.Struct(oe)
}

// Value is interface function to return current value to store to DB
func (oe OptionsEvents) Value() (driver.Value, error) {
	b, err := json.Marshal(oe)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (oe *OptionsEvents) Scan(input interface{}) error {
	return scanFromJSON(input, oe)
}

// OptionsFields is a proprietary structure to contain modules fields list
type OptionsFields struct {
	Name       string           `form:"name" json:"name" validate:"required,solid_fld"`
	Locale     ModuleLocaleDesc `form:"locale" json:"locale" validate:"required,valid"`
	ModuleName string           `form:"module_name" json:"module_name" validate:"required,solid"`
	ModuleOS   ModuleInfoOS     `form:"module_os" json:"module_os" validate:"required,valid"`
}

// Valid is function to control input/output data
func (of OptionsFields) Valid() error {
	return validate.Struct(of)
}

// Value is interface function to return current value to store to DB
func (of OptionsFields) Value() (driver.Value, error) {
	b, err := json.Marshal(of)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (of *OptionsFields) Scan(input interface{}) error {
	return scanFromJSON(input, of)
}

// OptionsTags is a proprietary structure to contain modules tags list
type OptionsTags struct {
	Name       string           `form:"name" json:"name" validate:"required,solid_ext"`
	Locale     ModuleLocaleDesc `form:"locale" json:"locale" validate:"required,valid"`
	ModuleName string           `form:"module_name" json:"module_name" validate:"required,solid"`
	ModuleOS   ModuleInfoOS     `form:"module_os" json:"module_os" validate:"required,valid"`
}

// Valid is function to control input/output data
func (ot OptionsTags) Valid() error {
	return validate.Struct(ot)
}

// Value is interface function to return current value to store to DB
func (ot OptionsTags) Value() (driver.Value, error) {
	b, err := json.Marshal(ot)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ot *OptionsTags) Scan(input interface{}) error {
	return scanFromJSON(input, ot)
}

// OptionsVersions is a proprietary structure to contain modules versions list
type OptionsVersions struct {
	Name       string       `form:"name" json:"name" validate:"required,semver"`
	ModuleName string       `form:"module_name" json:"module_name" validate:"required,solid"`
	ModuleOS   ModuleInfoOS `form:"module_os" json:"module_os" validate:"required,valid"`
}

// Valid is function to control input/output data
func (ov OptionsVersions) Valid() error {
	return validate.Struct(ov)
}

// Value is interface function to return current value to store to DB
func (ov OptionsVersions) Value() (driver.Value, error) {
	b, err := json.Marshal(ov)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ov *OptionsVersions) Scan(input interface{}) error {
	return scanFromJSON(input, ov)
}
