package models

import "github.com/jinzhu/gorm"

const (
	// RoleSAdmin is an SAdmin role
	RoleSAdmin = iota
	// RoleAdmin is an Admin role
	RoleAdmin
	// RoleUser is a User role
	RoleUser
	// RoleExternal is a External role
	RoleExternal = 100
)

// Role is model to contain user role information
type Role struct {
	ID   uint64 `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Name string `form:"name" json:"name" validate:"max=50,required" gorm:"type:VARCHAR(50);NOT NULL;UNIQUE_INDEX"`
}

// TableName returns the table name string to guaranty use correct table
func (r *Role) TableName() string {
	return "roles"
}

// Valid is function to control input/output data
func (r Role) Valid() error {
	return validate.Struct(r)
}

// Validate is function to use callback to control input/output data
func (r Role) Validate(db *gorm.DB) {
	if err := r.Valid(); err != nil {
		db.AddError(err)
	}
}
