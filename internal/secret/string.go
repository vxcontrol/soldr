package secret

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"os"
)

const maskedString = "xxx"

// String protects value from accidental exposure. It is a struct with private attribute because
// that way the value is hidden from reflection.
type String struct {
	value string
}

// Scan implements the sql Scanner interface.
func (s *String) Scan(src interface{}) error {
	v, ok := src.(string)
	if !ok {
		return errors.New("type assertion for secret failed")
	}
	s.value = v
	return nil
}

// Value implements the sql driver Valuer interface.
func (s String) Value() (driver.Value, error) {
	return s.value, nil
}

// NewString constructs secret value of type string
func NewString(v string) String {
	return String{value: v}
}

// String returns masked string
func (s String) String() string {
	return maskedString
}

// Unmask returns real value
func (s String) Unmask() string {
	return s.value
}

// UnmarshalJSON allows parsing secret from JSON
func (s *String) UnmarshalJSON(b []byte) error {
	return json.Unmarshal(b, &s.value)
}

// MarshalJSON hides secret as pure string
func (s String) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

// UnmarshalYAML allows parsing secret from YAML
func (s *String) UnmarshalYAML(unmarshal func(interface{}) error) error {
	return unmarshal(&s.value)
}

// MarshalYAML hides secret as pure string
func (s String) MarshalYAML() (interface{}, error) {
	return s.String(), nil
}

// FromEnv allows loading secret strings from environment variables.
func (s *String) FromEnv(name string) bool {
	if v, ok := os.LookupEnv(name); ok {
		s.value = v
	}
	return false
}
