package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
)

// GroupItemLocale is model to contain localization of group item information
type GroupItemLocale struct {
	Ru string `form:"ru" json:"ru" validate:"max=4096,required"`
	En string `form:"en" json:"en" validate:"max=4096,required"`
}

// Valid is function to control input/output data
func (gil GroupItemLocale) Valid() error {
	return validate.Struct(gil)
}

// GroupInfo is model to contain general group information
type GroupInfo struct {
	Name   GroupItemLocale `form:"name" json:"name" validate:"required,valid"`
	Tags   []string        `form:"tags" json:"tags" validate:"solid_ru,max=20,unique,required"`
	System bool            `form:"system" json:"system" validate:"omitempty"`
}

// Valid is function to control input/output data
func (gi GroupInfo) Valid() error {
	return validate.Struct(gi)
}

// Value is interface function to return current value to store to DB
func (gi GroupInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(gi)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (gi *GroupInfo) Scan(input interface{}) error {
	return scanFromJSON(input, gi)
}

// Group is model to contain group information from instance DB
type Group struct {
	ID          uint64     `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Hash        string     `form:"hash" json:"hash" validate:"len=32,hexadecimal,lowercase,required" gorm:"type:VARCHAR(32);NOT NULL"`
	Info        GroupInfo  `form:"info" json:"info" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	CreatedDate time.Time  `form:"created_date,omitempty" json:"created_date,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time  `form:"updated_at" json:"updated_at"`
	DeletedAt   *time.Time `form:"deleted_at,omitempty" json:"deleted_at,omitempty" sql:"index"`
}

// TableName returns the table name string to guaranty use correct table
func (g *Group) TableName() string {
	return "groups"
}

// BeforeDelete hook defined for cascade delete
func (g *Group) BeforeDelete(db *gorm.DB) error {
	subQueryAgents := db.Unscoped().Model(&Agent{}).Select("id").Where("group_id = ?", g.ID).QueryExpr()
	if err := db.Unscoped().Where("agent_id IN (?)", subQueryAgents).Delete(&Event{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("group_id = ?", g.ID).Delete(&GroupToPolicy{}).Error; err != nil {
		return err
	}
	err := db.Model(&Agent{}).Where("group_id = ?", g.ID).
		UpdateColumns(map[string]interface{}{
			"group_id":   0,
			"updated_at": gorm.Expr("NOW()"),
		}).Error
	return err
}

// Valid is function to control input/output data
func (g Group) Valid() error {
	return validate.Struct(g)
}

// Validate is function to use callback to control input/output data
func (g Group) Validate(db *gorm.DB) {
	if err := g.Valid(); err != nil {
		db.AddError(err)
	}
}

// GroupToPolicy is model to link group to policy in instance DB
type GroupToPolicy struct {
	ID       uint64 `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	GroupID  uint64 `form:"group_id" json:"group_id" validate:"min=1,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
	PolicyID uint64 `form:"policy_id" json:"policy_id" validate:"min=1,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
}

// TableName returns the table name string to guaranty use correct table
func (gtp *GroupToPolicy) TableName() string {
	return "groups_to_policies"
}

// Valid is function to control input/output data
func (gtp GroupToPolicy) Valid() error {
	return validate.Struct(gtp)
}

// Validate is function to use callback to control input/output data
func (gtp GroupToPolicy) Validate(db *gorm.DB) {
	if err := gtp.Valid(); err != nil {
		db.AddError(err)
	}
}

// GroupPolicies is model to contain group information linked with policies
type GroupPolicies struct {
	Policies []Policy `form:"policies,omitempty" json:"policies,omitempty" gorm:"many2many:groups_to_policies;jointable_foreignkey:group_id"`
	Group    `form:"" json:""`
}

// TableName returns the table name string to guaranty use correct table
func (gps *GroupPolicies) TableName() string {
	return "groups"
}

// Valid is function to control input/output data
func (gps GroupPolicies) Valid() error {
	for i := range gps.Policies {
		if err := gps.Policies[i].Valid(); err != nil {
			return err
		}
	}
	return gps.Group.Valid()
}

// Validate is function to use callback to control input/output data
func (gps GroupPolicies) Validate(db *gorm.DB) {
	if err := gps.Valid(); err != nil {
		db.AddError(err)
	}
}

// GroupDependency is a proprietary structure to contain group dependency with status
type GroupDependency struct {
	PolicyID         uint64 `form:"policy_id" json:"policy_id" validate:"min=0,numeric"`
	PolicyDependency `form:"" json:""`
}

// Valid is function to control input/output data
func (gd GroupDependency) Valid() error {
	if err := gd.PolicyDependency.Valid(); err != nil {
		return err
	}
	return validate.Struct(gd)
}

// Value is interface function to return current value to store to DB
func (gd GroupDependency) Value() (driver.Value, error) {
	b, err := json.Marshal(gd)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (gd *GroupDependency) Scan(input interface{}) error {
	return scanFromJSON(input, gd)
}
