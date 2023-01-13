package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
)

// AgentOS is model to contain agent OS information
type AgentOS struct {
	Type string `form:"type" json:"type" validate:"oneof=windows linux darwin,required"`
	Arch string `form:"arch" json:"arch" validate:"oneof=386 amd64,required"`
	Name string `form:"name" json:"name" validate:"max=255,required"`
}

// Valid is function to control input/output data
func (aos AgentOS) Valid() error {
	return validate.Struct(aos)
}

// AgentUser is model to contain agent User information
type AgentUser struct {
	Name   string   `form:"name" json:"name" validate:"required"`
	Groups []string `form:"groups" json:"groups" validate:"required"`
}

// Valid is function to control input/output data
func (au AgentUser) Valid() error {
	return validate.Struct(au)
}

// AgentNet is model to contain agent network information
type AgentNet struct {
	Hostname string   `form:"hostname" json:"hostname" validate:"required"`
	IPs      []string `form:"ips" json:"ips" validate:"required"`
}

// Valid is function to control input/output data
func (an AgentNet) Valid() error {
	return validate.Struct(an)
}

// AgentInfo is model to contain general agent information
type AgentInfo struct {
	OS    AgentOS     `form:"os" json:"os" validate:"required,valid"`
	Net   AgentNet    `form:"net" json:"net" validate:"required,valid"`
	Users []AgentUser `form:"users" json:"users" validate:"required"`
	Tags  []string    `form:"tags" json:"tags" validate:"solid_ru,max=20,unique,required"`
}

// Valid is function to control input/output data
func (ai AgentInfo) Valid() error {
	for i := range ai.Users {
		if err := ai.Users[i].Valid(); err != nil {
			return err
		}
	}
	return validate.Struct(ai)
}

// Value is interface function to return current value to store to DB
func (ai AgentInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(ai)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ai *AgentInfo) Scan(input interface{}) error {
	return scanFromJSON(input, ai)
}

// Agent is model to contain agent information from instance DB
type Agent struct {
	ID            uint64     `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Hash          string     `form:"hash" json:"hash" validate:"len=32,hexadecimal,lowercase,required" gorm:"type:VARCHAR(32);NOT NULL"`
	GroupID       uint64     `form:"group_id" json:"group_id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
	IP            string     `form:"ip" json:"ip" validate:"max=50,ip,required" gorm:"type:VARCHAR(50);NOT NULL"`
	Description   string     `form:"description" json:"description" validate:"max=255,required" gorm:"type:VARCHAR(255);NOT NULL"`
	Version       string     `form:"version" json:"version" validate:"max=20,required" gorm:"type:VARCHAR(20);NOT NULL"`
	Info          AgentInfo  `form:"info" json:"info" validate:"required,valid" gorm:"type:JSON;NOT NULL"`
	Status        string     `form:"status" json:"status" validate:"oneof=connected disconnected,required" gorm:"type:ENUM('connected','disconnected');NOT NULL"`
	AuthStatus    string     `form:"auth_status" json:"auth_status" validate:"oneof=authorized unauthorized blocked,required" gorm:"type:ENUM('authorized','unauthorized','blocked');NOT NULL"`
	ConnectedDate time.Time  `form:"connected_date,omitempty" json:"connected_date,omitempty" validate:"omitempty" gorm:"type:DATETIME;default:NULL"`
	CreatedDate   time.Time  `form:"created_date,omitempty" json:"created_date,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
	UpdatedAt     time.Time  `form:"updated_at" json:"updated_at"`
	DeletedAt     *time.Time `form:"deleted_at,omitempty" json:"deleted_at,omitempty" sql:"index"`
}

// TableName returns the table name string to guaranty use correct table
func (a *Agent) TableName() string {
	return "agents"
}

// BeforeDelete hook defined for cascade delete
func (a *Agent) BeforeDelete(db *gorm.DB) error {
	if err := db.Unscoped().Where("agent_id = ?", a.ID).Delete(&Event{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("agent_id = ?", a.ID).Delete(&AgentUpgradeTask{}).Error; err != nil {
		return err
	}
	return nil
}

// Valid is function to control input/output data
func (a Agent) Valid() error {
	return validate.Struct(a)
}

// Validate is function to use callback to control input/output data
func (a Agent) Validate(db *gorm.DB) {
	if err := a.Valid(); err != nil {
		db.AddError(err)
	}
}

// AgentGroup is model to contain agent information linked with agent group
type AgentGroup struct {
	Group Group `form:"group,omitempty" json:"group,omitempty" gorm:"association_autoupdate:false;association_autocreate:false"`
	Agent `form:"" json:""`
}

// Valid is function to control input/output data
func (ag AgentGroup) Valid() error {
	if err := ag.Group.Valid(); err != nil {
		return err
	}
	return ag.Agent.Valid()
}

// Validate is function to use callback to control input/output data
func (ag AgentGroup) Validate(db *gorm.DB) {
	if err := ag.Valid(); err != nil {
		db.AddError(err)
	}
}

// AgentPolicies is model to contain agent information linked with policies
type AgentPolicies struct {
	Policies []Policy `form:"policies,omitempty" json:"policies,omitempty" gorm:"many2many:groups_to_policies;foreignkey:GroupID;jointable_foreignkey:group_id"`
	Agent    `form:"" json:""`
}

// TableName returns the table name string to guaranty use correct table
func (aps *AgentPolicies) TableName() string {
	return "agents"
}

// Valid is function to control input/output data
func (aps AgentPolicies) Valid() error {
	for i := range aps.Policies {
		if err := aps.Policies[i].Valid(); err != nil {
			return err
		}
	}
	return aps.Agent.Valid()
}

// Validate is function to use callback to control input/output data
func (aps AgentPolicies) Validate(db *gorm.DB) {
	if err := aps.Valid(); err != nil {
		db.AddError(err)
	}
}

// AgentDependency is a proprietary structure to contain agent dependency with status
type AgentDependency struct {
	GroupDependency `form:"" json:""`
}

// Valid is function to control input/output data
func (ad AgentDependency) Valid() error {
	if err := ad.GroupDependency.Valid(); err != nil {
		return err
	}
	return validate.Struct(ad)
}

// Value is interface function to return current value to store to DB
func (ad AgentDependency) Value() (driver.Value, error) {
	b, err := json.Marshal(ad)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ad *AgentDependency) Scan(input interface{}) error {
	return scanFromJSON(input, ad)
}
