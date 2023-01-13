package models

import "github.com/jinzhu/gorm"

type AgentUpgradeTask struct {
	ID         uint64 `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Batch      string `form:"batch" json:"batch" validate:"len=32,hexadecimal,lowercase,required" gorm:"type:VARCHAR(32);NOT NULL"`
	AgentID    uint64 `form:"agent_id" json:"agent_id" validate:"min=1,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
	Version    string `form:"version" json:"version" validate:"max=20,required" gorm:"type:VARCHAR(20);NOT NULL"`
	Status     string `form:"status" json:"status" validate:"oneof=new running ready failed,required" gorm:"type:ENUM('new','running','ready','failed');NOT NULL;default:'new'"`
	Reason     string `form:"reason,omitempty" json:"reason,omitempty" validate:"required_if=Status failed,lockey=omitempty,omitempty" gorm:"type:VARCHAR(150);NULL"`
	Created    string `form:"created,omitempty" json:"created,omitempty" valdiate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
	LastUpdate string `form:"last_update,omitempty" json:"last_update,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name string to guaranty use correct table
func (aut *AgentUpgradeTask) TableName() string {
	return "upgrade_tasks"
}

// Valid is function to control input/output data
func (aut AgentUpgradeTask) Valid() error {
	return validate.Struct(aut)
}

// Validate is function to use callback to control input/output data
func (aut AgentUpgradeTask) Validate(db *gorm.DB) {
	if err := aut.Valid(); err != nil {
		db.AddError(err)
	}
}
