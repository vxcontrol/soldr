package secret_test

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	"soldr/internal/secret"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v2"
)

const secretValue = "admin"

type Foo struct {
	Password secret.String `json:"password" yaml:"password"`
}

// TestStringUnmask that secret is correctly unmasked on demand
func TestStringUnmask(t *testing.T) {
	assert.Equal(t, secretValue, secret.NewString(secretValue).Unmask(), "secret is not unmasked")
}

// TestStringString that secret is hidden from normal access
func TestStringString(t *testing.T) {
	assert.NotEqual(t, "", secret.NewString("").String(), "empty secret is not masked")
	assert.NotEqual(t, secretValue, secret.NewString(secretValue).String(), "secret is not masked")
	//nolint:S1025
	assert.NotContains(t, fmt.Sprintf("%s", secret.NewString(secretValue)), secretValue, "secret is not masked when formatting")
	assert.NotContains(t, fmt.Sprintf("%v", secret.NewString(secretValue)), secretValue, "secret is not masked when formatting")
	assert.NotContains(t, fmt.Sprintf("%+v", secret.NewString(secretValue)), secretValue, "secret is not masked when formatting")
	assert.NotContains(t, fmt.Sprintf("%q", secret.NewString(secretValue)), secretValue, "secret is not masked when formatting")
}

// TestStringReflect that secret is hidden from reflect
func TestStringReflect(t *testing.T) {
	s := secret.NewString(secretValue)

	// Check that we can't get value as string
	v := reflect.ValueOf(s)
	assert.NotContains(t, v.String(), secretValue)

	// Check that all fields are unexported
	typ := reflect.TypeOf(s)
	for i := 0; i < typ.NumField(); i++ {
		assert.NotEmpty(t, typ.Field(i).PkgPath, fmt.Sprintf("field %s is exported", typ.Field(i).Name))
	}
}

// TestStringUnmarshalJSON that secret is correctly unmarshaled
func TestStringUnmarshalJSON(t *testing.T) {
	var foo Foo
	require.NoError(t, json.Unmarshal([]byte(fmt.Sprintf("{\"password\": \"%s\"}", secretValue)), &foo))
	require.NotEqual(t, secretValue, foo.Password.String())
	require.Equal(t, secretValue, foo.Password.Unmask())
}

// TestStringMarshalJSON that secret is hidden when marshaled
func TestStringMarshalJSON(t *testing.T) {
	foo := Foo{Password: secret.NewString(secretValue)}
	marshaled, err := json.Marshal(foo)
	require.NoError(t, err)

	str := string(marshaled)
	assert.NotContains(t, str, secretValue)
	// Secret must be marshaled as string, not object
	assert.Equal(t, "{\"password\":\"xxx\"}", str)
}

// TestStringUnmarshalYAML that secret is correctly unmarshaled
func TestStringUnmarshalYAML(t *testing.T) {
	var foo Foo
	require.NoError(t, yaml.Unmarshal([]byte("password: "+secretValue), &foo))
	require.NotEqual(t, secretValue, foo.Password.String())
	require.Equal(t, secretValue, foo.Password.Unmask())
}

// TestStringMarshalYAML that secret is hidden when marshaled
func TestStringMarshalYAML(t *testing.T) {
	foo := Foo{Password: secret.NewString(secretValue)}
	marshaled, err := yaml.Marshal(foo)
	require.NoError(t, err)

	str := string(marshaled)
	assert.NotContains(t, str, secretValue)
	// Secret must be marshaled as string, not object
	assert.Equal(t, "password: xxx\n", str)
}
