package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/jinzhu/gorm"
)

// BynaryVersion is model to contain splited semantic version format for a system binary record
type BinaryVersion struct {
	Major uint64 `form:"major" json:"major" validate:"min=0,max=100"`
	Minor uint64 `form:"minor" json:"minor" validate:"min=0,max=100"`
	Patch uint64 `form:"patch" json:"patch" validate:"min=0,max=100"`
	Build uint64 `form:"build" json:"build" validate:"min=0,max=10000"`
	Rev   string `form:"rev" json:"rev" validate:"omitempty"`
}

// Valid is function to control input/output data
func (bv BinaryVersion) Valid() error {
	return validate.Struct(bv)
}

// Value is interface function to return current value to store to DB
func (bv BinaryVersion) Value() (driver.Value, error) {
	b, err := json.Marshal(bv)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (bv *BinaryVersion) Scan(input interface{}) error {
	return scanFromJSON(input, bv)
}

// BynaryChksum is model to contain check sums information by a system binary file
type BinaryChksum struct {
	MD5    string `form:"md5" json:"md5" validate:"required,len=32,hexadecimal,lowercase"`
	SHA256 string `form:"sha256" json:"sha256" validate:"required,len=64,hexadecimal,lowercase"`
}

// Valid is function to control input/output data
func (bcs BinaryChksum) Valid() error {
	return validate.Struct(bcs)
}

// Value is interface function to return current value to store to DB
func (bcs BinaryChksum) Value() (driver.Value, error) {
	b, err := json.Marshal(bcs)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (bcs *BinaryChksum) Scan(input interface{}) error {
	return scanFromJSON(input, bcs)
}

// BinaryInfo is model to contain general information about system binaries (files, check sums and version)
type BinaryInfo struct {
	Files   []string                `form:"files" json:"files" validate:"required,min=1,unique,dive,required"`
	Chksums map[string]BinaryChksum `form:"chksums" json:"chksums" validate:"required,min=1,dive,keys,required,endkeys,valid"`
	Version BinaryVersion           `form:"version" json:"version" validate:"required,valid"`
}

// Valid is function to control input/output data
func (bi BinaryInfo) Valid() error {
	return validate.Struct(bi)
}

// Value is interface function to return current value to store to DB
func (bi BinaryInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(bi)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (bi *BinaryInfo) Scan(input interface{}) error {
	return scanFromJSON(input, bi)
}

// Binary is model to contain information about system binaries from global S3 bucket
type Binary struct {
	ID         uint64     `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	TenantID   uint64     `form:"tenant_id" json:"tenant_id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL"`
	Hash       string     `form:"hash" json:"hash" validate:"len=32,hexadecimal,lowercase,omitempty" gorm:"type:VARCHAR(32);NOT NULL"`
	Type       string     `form:"type" json:"type" validate:"required,oneof=vxagent" gorm:"type:ENUM('vxagent');NOT NULL"`
	Version    string     `form:"version" json:"version" validate:"required,max=25,semverex" gorm:"type:VARCHAR(25);NOT NULL;UNIQUE_INDEX"`
	Info       BinaryInfo `form:"info" json:"info" validate:"required" gorm:"type:JSON;NOT NULL"`
	UploadDate time.Time  `form:"upload_date,omitempty" json:"upload_date,omitempty" validate:"omitempty" gorm:"type:DATETIME;NOT NULL;default:CURRENT_TIMESTAMP"`
}

// TableName returns the table name string to guaranty use correct table
func (bf *Binary) TableName() string {
	return "binaries"
}

// Valid is function to control input/output data
func (bf Binary) Valid() error {
	return validate.Struct(bf)
}

// Value is interface function to return current value to store to DB
func (bf Binary) Value() (driver.Value, error) {
	b, err := json.Marshal(bf)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (bf *Binary) Scan(input interface{}) error {
	return scanFromJSON(input, bf)
}

// Validate is function to use callback to control input/output data
func (bf Binary) Validate(db *gorm.DB) {
	if err := bf.Valid(); err != nil {
		db.AddError(err)
	}
}

// ExtConnInfo is model to contain general information about binaries for external connections (check sums and version)
type ExtConnInfo struct {
	Chksums map[string]BinaryChksum `form:"chksums" json:"chksums" validate:"required,min=1,dive,keys,required,semverex,endkeys,valid"`
	Version BinaryVersion           `form:"version" json:"version" validate:"required,valid"`
}

// Valid is function to control input/output data
func (eci ExtConnInfo) Valid() error {
	return validate.Struct(eci)
}

// Value is interface function to return current value to store to DB
func (eci ExtConnInfo) Value() (driver.Value, error) {
	b, err := json.Marshal(eci)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (eci *ExtConnInfo) Scan(input interface{}) error {
	return scanFromJSON(input, eci)
}

// ExtConnInfo is model to contain general information about external connections
type ExtConn struct {
	ID   uint64      `form:"id" json:"id" validate:"min=0,numeric" gorm:"type:INT(10) UNSIGNED;NOT NULL;PRIMARY_KEY;AUTO_INCREMENT"`
	Hash string      `form:"hash" json:"hash" validate:"len=32,hexadecimal,lowercase,omitempty" gorm:"type:VARCHAR(32);NOT NULL"`
	Desc string      `form:"description" json:"description" validate:"required,max=255" gorm:"column:description;type:VARCHAR(255);NOT NULL"`
	Type string      `form:"type" json:"type" validate:"required,oneof=aggregate browser external" gorm:"type:ENUM('aggregate','browser','external');NOT NULL"`
	Info ExtConnInfo `form:"info" json:"info" validate:"required" gorm:"type:JSON;NOT NULL"`
}

// TableName returns the table name string to guaranty use correct table
func (ec *ExtConn) TableName() string {
	return "external_connections"
}

// Valid is function to control input/output data
func (ec ExtConn) Valid() error {
	return validate.Struct(ec)
}

// Value is interface function to return current value to store to DB
func (ec ExtConn) Value() (driver.Value, error) {
	b, err := json.Marshal(ec)
	return string(b), err
}

// Scan is interface function to parse DB value when getting from DB
func (ec *ExtConn) Scan(input interface{}) error {
	return scanFromJSON(input, ec)
}

// Validate is function to use callback to control input/output data
func (ec ExtConn) Validate(db *gorm.DB) {
	if err := ec.Valid(); err != nil {
		db.AddError(err)
	}
}
