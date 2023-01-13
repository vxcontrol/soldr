package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"testing"

	hardeningConfig "soldr/pkg/app/server/mmodule/hardening/config"
	"soldr/pkg/vxproto"
)

func Test_mergeConfigs(t *testing.T) {
	tests := []struct {
		Src            *Config
		Dst            *Config
		ExpectedResult *Config
	}{
		{
			Dst: &Config{
				Listen: "ws://localhost:8080",
				Base:   "",
				Loader: Loader{
					Config: "fs",
				},
				ConfigFile:         "",
				LogDir:             "some-log-dir",
				Debug:              true,
				Service:            false,
				IsPrintVersionOnly: true,
				IsProfiling:        false,
				S3: S3{
					AccessKey: "some-access-key",
					SecretKey: "some-secret-key",
				},
				Validator: hardeningConfig.Validator{
					Type: "fs",
				},
				Certs: CertsConfig{
					Type: "fs",
				},
			},
			Src: &Config{
				Listen: "ws://localhost:4443",
				Base:   "some-base-dir",
				Loader: Loader{
					Files: "s3",
				},
				DB: DB{
					Host: "some-host",
					Port: "some-port",
				},
				ConfigFile:         "some-config-file",
				LogDir:             "",
				Debug:              false,
				Service:            true,
				IsPrintVersionOnly: true,
				IsProfiling:        false,
				S3: S3{
					AccessKey:  "some-other-access-key",
					BucketName: "some-bucket-name",
					Endpoint:   "some-endpoint",
				},
				Certs: CertsConfig{},
			},
			ExpectedResult: &Config{
				Listen: "ws://localhost:4443",
				Base:   "some-base-dir",
				Loader: Loader{
					Config: "fs",
					Files:  "s3",
				},
				DB: DB{
					Host: "some-host",
					Port: "some-port",
				},
				ConfigFile:         "some-config-file",
				LogDir:             "some-log-dir",
				Debug:              true,
				Service:            true,
				IsPrintVersionOnly: true,
				IsProfiling:        false,
				S3: S3{
					AccessKey:  "some-other-access-key",
					SecretKey:  "some-secret-key",
					BucketName: "some-bucket-name",
					Endpoint:   "some-endpoint",
				},
				Certs: CertsConfig{
					Type: "fs",
				},
				Validator: hardeningConfig.Validator{
					Type: "fs",
				},
			},
		},
	}

	for i, tc := range tests {
		t.Run(fmt.Sprintf("test #%d", i), func(t *testing.T) {
			actualResult, err := mergeConfigs(tc.Dst, tc.Src)
			if err != nil {
				t.Errorf("failed to merge configs: %v", err)
				return
			}
			if err := areStructureFieldsEqual(tc.ExpectedResult, actualResult); err != nil {
				comparisonErr := err
				expected, err := prettyPrintStruct(tc.ExpectedResult)
				if err != nil {
					t.Errorf("internal test error: %v", err)
					return
				}
				actual, err := prettyPrintStruct(actualResult)
				if err != nil {
					t.Errorf("internal test error: %v", err)
					return
				}
				t.Errorf("expected: \n%v\n, got: \n%v, err: %v", expected, actual, comparisonErr)
				return
			}
		})
	}
}

func areStructureFieldsEqual(expected interface{}, actual interface{}) error {
	expectedVal, actualVal := reflect.ValueOf(expected), reflect.ValueOf(actual)
	expectedType, actualType := expectedVal.Type(), actualVal.Type()
	if expectedType != actualType {
		return fmt.Errorf("expected type %v and actual type %v are not equal", expectedType, actualType)
	}
	unequalFields := make([]string, 0)
	switch expectedVal.Kind() {
	case reflect.Ptr:
		expectedVal, actualVal = reflect.Indirect(expectedVal), reflect.Indirect(actualVal)
		for i := 0; i < expectedVal.NumField(); i++ {
			expectedField := expectedVal.Field(i)
			fieldName := expectedVal.Type().Field(i).Name
			actualField := actualVal.FieldByName(fieldName)
			if !reflect.DeepEqual(expectedField.Interface(), actualField.Interface()) {
				unequalFields = append(unequalFields, fmt.Sprintf("%v: %v / %v", fieldName, expectedField, actualField))
			}
		}
	case reflect.Map:
		iter := expectedVal.MapRange()
		for iter.Next() {
			k, v := iter.Key(), iter.Value()
			actualV := actualVal.MapIndex(k)
			if !reflect.DeepEqual(v.Interface(), actualV.Interface()) {
				unequalFields = append(unequalFields, fmt.Sprintf("%v: %v / %v", k.Interface(), v, actualV))
			}
		}
	default:
		return fmt.Errorf("NYI")
	}
	if len(unequalFields) != 0 {
		return fmt.Errorf("the following fields in the configs are not equal: %v", unequalFields)
	}
	return nil
}

func Test_getAPIConfigFromEnvVars(t *testing.T) {
	cases := []struct {
		Name           string
		EnvVars        map[string]string
		ExpectedResult vxproto.ServerAPIVersionsConfig
		ExpectedErr    error
	}{
		{
			Name: "normal",
			EnvVars: map[string]string{
				"VXSERVER_API_VERSION_POLICY_v1": "allow",
				"VXSERVER_API_VERSION_POLICY_v2": "block",
				"VXSERVER_API_VERSION_POLICY_V3": "upgrade",
			},
			ExpectedResult: map[string]*vxproto.ServerAPIConfig{
				"v1": {
					Version:          "v1",
					ConnectionPolicy: vxproto.EndpointConnectionPolicyAllow,
				},
				"v2": {
					Version:          "v2",
					ConnectionPolicy: vxproto.EndpointConnectionPolicyBlock,
				},
				"V3": {
					Version:          "V3",
					ConnectionPolicy: vxproto.EndpointConnectionPolicyUpgrade,
				},
			},
			ExpectedErr: nil,
		},
		{
			Name: "with url encoding",
			EnvVars: map[string]string{
				"VXSERVER_API_VERSION_POLICY_новая_версия": "allow",
				"VXSERVER_API_VERSION_POLICY_//%?":         "block",
			},
			ExpectedResult: map[string]*vxproto.ServerAPIConfig{
				"новая_версия": {
					Version:          "%D0%BD%D0%BE%D0%B2%D0%B0%D1%8F_%D0%B2%D0%B5%D1%80%D1%81%D0%B8%D1%8F",
					ConnectionPolicy: vxproto.EndpointConnectionPolicyAllow,
				},
				"//%?": {
					Version:          "%2F%2F%25%3F",
					ConnectionPolicy: vxproto.EndpointConnectionPolicyBlock,
				},
			},
			ExpectedErr: nil,
		},
		{
			Name:           "no API version defined",
			EnvVars:        map[string]string{},
			ExpectedResult: map[string]*vxproto.ServerAPIConfig{},
		},
		{
			Name: "an invalid connection policy defined",
			EnvVars: map[string]string{
				"VXSERVER_API_VERSION_POLICY_v1":                           "allow",
				"VXSERVER_API_VERSION_POLICY_VXSERVER_API_VERSION_POLICY_": "random_policy",
			},
			ExpectedResult: nil,
			ExpectedErr:    fmt.Errorf("failed to register the API version VXSERVER_API_VERSION_POLICY_ with a connection policy \"random_policy\": unknown endpoint connection policy passed: \"random_policy\""),
		},
	}

	for i, tc := range cases {
		i, tc := i, tc
		t.Run(fmt.Sprintf("test case #%d: %s", i, tc.Name), func(t *testing.T) {
			defer func() {
				for name := range tc.EnvVars {
					if err := os.Unsetenv(name); err != nil {
						t.Errorf("failed to unset the env var %s: %v", name, err)
					}
				}
			}()
			for name, val := range tc.EnvVars {
				if err := os.Setenv(name, val); err != nil {
					t.Errorf("failed to set the env var %s with value %s: %v", name, val, err)
					return
				}
			}
			actualResult, err := getAPIConfigFromEnvVars()
			if err := compareErrors(tc.ExpectedErr, err); err != nil {
				t.Errorf("error comparison failed: %v", err)
				return
			}
			if err := areStructureFieldsEqual(tc.ExpectedResult, actualResult); err != nil {
				comparisonErr := err
				expected, err := prettyPrintStruct(tc.ExpectedResult)
				if err != nil {
					t.Errorf("internal test error: %v", err)
					return
				}
				actual, err := prettyPrintStruct(actualResult)
				if err != nil {
					t.Errorf("internal test error: %v", err)
					return
				}
				t.Errorf("expected: \n%v\n, got: \n%v, err: %v", expected, actual, comparisonErr)
				return
			}
		})
	}
}

func compareErrors(expectedErr error, actualErr error) error {
	if actualErr == nil && expectedErr == nil {
		return nil
	}
	if actualErr == nil || expectedErr == nil || actualErr.Error() != expectedErr.Error() {
		return fmt.Errorf("expected error: %v, got: %v", expectedErr, actualErr)
	}
	return nil
}

func prettyPrintStruct(v interface{}) (string, error) {
	serialized, err := json.MarshalIndent(v, "", "\t")
	if err != nil {
		return "", fmt.Errorf("failed to JSON-marhsal with indent: %v", err)
	}
	return string(serialized), nil
}
