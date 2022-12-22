package models

import (
	"github.com/jinzhu/gorm"
)

// User is model to contain user information
type User struct {
	ID       uint64 `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Mail     string `form:"mail" json:"mail" validate:"max=50,vmail,required" gorm:"type:VARCHAR(50);NOT NULL;UNIQUE_INDEX"`
	Name     string `form:"name" json:"name" validate:"max=70,required" gorm:"type:VARCHAR(70);NOT NULL;default:''"`
	Status   string `form:"status" json:"status" validate:"oneof=created active blocked,required" gorm:"type:ENUM('created','active','blocked');NOT NULL"`
	Type     string `form:"type" json:"type" validate:"oneof=local oauth,required" gorm:"type:ENUM('local','oauth');NOT NULL"`
	RoleID   uint64 `form:"role_id" json:"role_id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;default:2"`
	TenantID uint64 `form:"tenant_id" json:"tenant_id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
	Hash     string `form:"hash" json:"hash" validate:"len=32,hexadecimal,lowercase,omitempty" gorm:"type:VARCHAR(32);NOT NULL"`
}

// TableName returns the table name string to guaranty use correct table
func (u *User) TableName() string {
	return "users"
}

// Valid is function to control input/output data
func (u User) Valid() error {
	return validate.Struct(u)
}

// Validate is function to use callback to control input/output data
func (u User) Validate(db *gorm.DB) {
	if err := u.Valid(); err != nil {
		db.AddError(err)
	}
}

// UserPassword is model to contain user information
type UserPassword struct {
	Password string `form:"password" json:"password" validate:"max=100,required" gorm:"type:VARCHAR(100)"`
	User     `form:"" json:""`
}

// TableName returns the table name string to guaranty use correct table
func (up *UserPassword) TableName() string {
	return "users"
}

// Valid is function to control input/output data
func (up UserPassword) Valid() error {
	if err := up.User.Valid(); err != nil {
		return err
	}
	return validate.Struct(up)
}

// Validate is function to use callback to control input/output data
func (up UserPassword) Validate(db *gorm.DB) {
	if err := up.Valid(); err != nil {
		db.AddError(err)
	}
}

// Login is model to contain user information on Login procedure
type Login struct {
	Mail     string `form:"mail" json:"mail" validate:"max=50,required" gorm:"type:VARCHAR(50);NOT NULL;UNIQUE_INDEX"`
	Password string `form:"password" json:"password" validate:"min=4,max=100,required" gorm:"type:VARCHAR(100)"`
	Service  string `form:"service" json:"service" validate:"omitempty,len=32,hexadecimal,lowercase" gorm:"-" default:""`
}

// TableName returns the table name string to guaranty use correct table
func (sin *Login) TableName() string {
	return "users"
}

// Valid is function to control input/output data
func (sin Login) Valid() error {
	return validate.Struct(sin)
}

// Validate is function to use callback to control input/output data
func (sin Login) Validate(db *gorm.DB) {
	if err := sin.Valid(); err != nil {
		db.AddError(err)
	}
}

// PermissionsService is model to contain service permissions for current user
type PermissionsService struct {
	Roles      []string `json:"roles" validate:"required,dive,printascii,required"`
	RoleIDs    []string `json:"roleIds" validate:"required,dive,uuid,required"`
	Privileges []string `json:"privileges" validate:"required,dive,printascii,required"`
}

// Valid is function to control input/output data
func (ps PermissionsService) Valid() error {
	return validate.Struct(ps)
}

// Password is model to contain user password to change it
type Password struct {
	CurrentPassword string `form:"current_password" json:"current_password" validate:"nefield=Password,min=8,max=100,required" gorm:"-"`
	Password        string `form:"password" json:"password" validate:"stpass,max=100,required" gorm:"type:VARCHAR(100)"`
	ConfirmPassword string `form:"confirm_password" json:"confirm_password" validate:"eqfield=Password" gorm:"-"`
}

// TableName returns the table name string to guaranty use correct table
func (p *Password) TableName() string {
	return "users"
}

// Valid is function to control input/output data
func (p Password) Valid() error {
	return validate.Struct(p)
}

// Validate is function to use callback to control input/output data
func (p Password) Validate(db *gorm.DB) {
	if err := p.Valid(); err != nil {
		db.AddError(err)
	}
}

// UserRole is model to contain user information linked with user role
type UserRole struct {
	Role Role `form:"role,omitempty" json:"role,omitempty" gorm:"association_autoupdate:false;association_autocreate:false"`
	User `form:"" json:""`
}

// Valid is function to control input/output data
func (ur UserRole) Valid() error {
	if err := ur.Role.Valid(); err != nil {
		return err
	}
	return ur.User.Valid()
}

// Validate is function to use callback to control input/output data
func (ur UserRole) Validate(db *gorm.DB) {
	if err := ur.Valid(); err != nil {
		db.AddError(err)
	}
}

// UserTenant is model to contain user information linked with user tenant
type UserTenant struct {
	Tenant Tenant `form:"tenant,omitempty" json:"tenant,omitempty" gorm:""`
	User   `form:"" json:""`
}

// Valid is function to control input/output data
func (ut UserTenant) Valid() error {
	if err := ut.Tenant.Valid(); err != nil {
		return err
	}
	return ut.User.Valid()
}

// Validate is function to use callback to control input/output data
func (ut UserTenant) Validate(db *gorm.DB) {
	if err := ut.Valid(); err != nil {
		db.AddError(err)
	}
}

// UserRoleTenant is model to contain user information linked with user role and tenant
type UserRoleTenant struct {
	Role   Role   `form:"role,omitempty" json:"role,omitempty" gorm:"association_autoupdate:false;association_autocreate:false"`
	Tenant Tenant `form:"tenant,omitempty" json:"tenant,omitempty" gorm:"association_autoupdate:false;association_autocreate:false"`
	User   `form:"" json:""`
}

// Valid is function to control input/output data
func (urt UserRoleTenant) Valid() error {
	if err := urt.Role.Valid(); err != nil {
		return err
	}
	if err := urt.Tenant.Valid(); err != nil {
		return err
	}
	return urt.User.Valid()
}

// Validate is function to use callback to control input/output data
func (urt UserRoleTenant) Validate(db *gorm.DB) {
	if err := urt.Valid(); err != nil {
		db.AddError(err)
	}
}
