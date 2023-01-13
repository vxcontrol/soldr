package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
)

// PolicyItemLocale is model to contain localization of policy item information
type PolicyItemLocale struct {
	Ru string `form:"ru" json:"ru" validate:"max=4096,required"`
	En string `form:"en" json:"en" validate:"max=4096,required"`
}

// Valid is function to control input/output data
func (pil PolicyItemLocale) Valid() error {
	return validate.Struct(pil)
}

// PolicyInfo is model to contain general policy information
type PolicyInfo struct {
	Name   PolicyItemLocale `form:"name" json:"name" validate:"required,valid"`
	Tags   []string         `form:"tags" json:"tags" validate:"solid_ru,max=20,unique,required"`
	System bool             `form:"system" json:"system" validate:"omitempty"`
}

// Valid is function to control input/output data
func (pi PolicyInfo) Valid() error {
	return validate.Struct(pi)
}

// Value is interface function to return current value to store to DB
func (pi PolicyInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(pi)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (pi *PolicyInfo) Scan(input interface{}) error {
	return scanFromJSON(input, pi)
}

// Policy is model to contain policy information from instance DB
type Policy struct {
	ID          uint64     `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Hash        string     `form:"hash" json:"hash" validate:"len=32,hexadecimal,lowercase,required" gorm:"type:VARCHAR(32);NOT NULL"`
	Info        PolicyInfo `form:"info" json:"info" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	CreatedDate time.Time  `form:"created_date,omitempty" json:"created_date,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time  `form:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time `form:"deleted_at,omitempty" json:"deleted_at,omitempty" sql:"index"`
}

// TableName returns the table name string to guaranty use correct table
func (p *Policy) TableName() string {
	return "policies"
}

// BeforeDelete hook defined for cascade delete
func (p *Policy) BeforeDelete(db *gorm.DB) error {
	subQueryModules := db.Unscoped().Model(&ModuleA{}).Select("id").Where("policy_id = ?", p.ID).QueryExpr()
	if err := db.Unscoped().Where("module_id IN (?)", subQueryModules).Delete(&Event{}).Error; err != nil {
		return err
	}
	if err := db.Where("policy_id = ?", p.ID).Delete(&ModuleA{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("policy_id = ?", p.ID).Delete(&GroupToPolicy{}).Error; err != nil {
		return err
	}
	return nil
}

// Valid is function to control input/output data
func (p Policy) Valid() error {
	return validate.Struct(p)
}

// Validate is function to use callback to control input/output data
func (p Policy) Validate(db *gorm.DB) {
	if err := p.Valid(); err != nil {
		db.AddError(err)
	}
}

// PolicyGroups is model to contain policy information linked with groups
type PolicyGroups struct {
	Groups []Group `form:"groups,omitempty" json:"groups,omitempty" gorm:"many2many:groups_to_policies;jointable_foreignkey:policy_id"`
	Policy `form:"" json:""`
}

// TableName returns the table name string to guaranty use correct table
func (pgs *PolicyGroups) TableName() string {
	return "policies"
}

// Valid is function to control input/output data
func (pgs PolicyGroups) Valid() error {
	for i := range pgs.Groups {
		if err := pgs.Groups[i].Valid(); err != nil {
			return err
		}
	}
	return pgs.Policy.Valid()
}

// Validate is function to use callback to control input/output data
func (pgs PolicyGroups) Validate(db *gorm.DB) {
	if err := pgs.Valid(); err != nil {
		db.AddError(err)
	}
}

// PolicyModulesA is model to contain policy information linked with agent modules
type PolicyModules struct {
	Modules []ModuleAShort `form:"modules,omitempty" json:"modules,omitempty" gorm:"foreignkey:PolicyID;association_autoupdate:false;association_autocreate:false"`
	Policy  `form:"" json:""`
}

// TableName returns the table name string to guaranty use correct table
func (pms *PolicyModules) TableName() string {
	return "policies"
}

// Valid is function to control input/output data
func (pms PolicyModules) Valid() error {
	for i := range pms.Modules {
		if err := pms.Modules[i].Valid(); err != nil {
			return err
		}
	}
	return pms.Policy.Valid()
}

// Validate is function to use callback to control input/output data
func (pms PolicyModules) Validate(db *gorm.DB) {
	if err := pms.Valid(); err != nil {
		db.AddError(err)
	}
}

// PolicyDependency is a proprietary structure to contain policy dependency with status
type PolicyDependency struct {
	SourceModuleName string `form:"source_module_name" json:"source_module_name" validate:"required,solid"`
	ModuleDependency `form:"" json:""`
}

// Valid is function to control input/output data
func (pd PolicyDependency) Valid() error {
	if err := pd.ModuleDependency.Valid(); err != nil {
		return err
	}
	return validate.Struct(pd)
}

// Value is interface function to return current value to store to DB
func (pd PolicyDependency) Value() (driver.Value, error) {
	b, err := json.Marshal(pd)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (pd *PolicyDependency) Scan(input interface{}) error {
	return scanFromJSON(input, pd)
}
