package models

import "github.com/jinzhu/gorm"

// Tenant is model to contain tenant information
type Tenant struct {
	ID          uint64 `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Status      string `form:"status" json:"status" validate:"oneof=active blocked,required" gorm:"type:ENUM('active','blocked');NOT NULL"`
	Hash        string `form:"hash" json:"hash" validate:"len=32,hexadecimal,lowercase,required" gorm:"type:VARCHAR(32);NOT NULL"`
	UUID        string `form:"uuid,omitempty" json:"uuid,omitempty" validate:"len=36,uuid,omitempty" gorm:"type:VARCHAR(100)"`
	Description string `form:"description" json:"description" validate:"max=255" gorm:"type:VARCHAR(255)"`
}

// TableName returns the table name string to guaranty use correct table
func (t *Tenant) TableName() string {
	return "tenants"
}

// BeforeDelete hook defined for cascade delete
func (t *Tenant) BeforeDelete(db *gorm.DB) error {
	if err := db.Unscoped().Where("tenant_id = ?", t.ID).Delete(&User{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("tenant_id = ?", t.ID).Delete(&Service{}).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Where("tenant_id = ?", t.ID).Delete(&ModuleS{}).Error; err != nil {
		return err
	}

	return db.Error
}

// Valid is function to control input/output data
func (t Tenant) Valid() error {
	return validate.Struct(t)
}

// Validate is function to use callback to control input/output data
func (t Tenant) Validate(db *gorm.DB) {
	if err := t.Valid(); err != nil {
		db.AddError(err)
	}
}
