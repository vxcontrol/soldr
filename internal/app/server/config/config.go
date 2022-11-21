package config

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"reflect"
	"strings"

	"github.com/imdario/mergo"
	"github.com/sirupsen/logrus"

	hardeningConfig "soldr/internal/app/server/mmodule/hardening/config"
	"soldr/internal/vxproto"
)

// config implements struct of vxserver configuration file
type Config struct {
	Loader            Loader                          `json:"loader"`
	DB                DB                              `json:"db"`
	S3                S3                              `json:"s3"`
	Certs             CertsConfig                     `json:"certs"`
	Validator         hardeningConfig.Validator       `json:"validator"`
	APIVersionsConfig vxproto.ServerAPIVersionsConfig `json:"-"`
	Base              string                          `json:"base"`
	LogDir            string                          `json:"log_dir"`
	Listen            string                          `json:"listen"`
	OtelAddr          string                          `json:"otel_addr"`

	// The following fields should not be present in the config file
	IsPrintVersionOnly bool   `json:"version"`
	IsProfiling        bool   `json:"profiling"`
	ConfigFile         string `json:"config_file"`
	Command            string `json:"command"`
	Debug              bool   `json:"debug"`
	Service            bool   `json:"service"`
}

type Loader struct {
	Config string `json:"config"`
	Files  string `json:"files"`
}

type DB struct {
	Host string `json:"host"`
	Port string `json:"port"`
	Name string `json:"name"`
	User string `json:"user"`
	Pass string `json:"pass"`
}

type S3 struct {
	AccessKey  string `json:"access_key"`
	SecretKey  string `json:"secret_key"`
	BucketName string `json:"bucket_name"`
	Endpoint   string `json:"endpoint"`
}

type CertsConfig struct {
	Type string `json:"type"`
	Base string `json:"base"`
}

func ReadConfig() (*Config, error) {
	var result *Config = defaultConfig
	flagsConfig := parseFlags()
	if len(flagsConfig.ConfigFile) != 0 {
		fileConfig, err := parseConfigFile(flagsConfig.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse the config file %s: %w", flagsConfig.ConfigFile, err)
		}
		result, err = mergeConfigs(result, fileConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to merge default and file configs: %w", err)
		}
	}
	envVarsConfig := parseEnvVars()
	var err error
	result, err = mergeConfigs(result, envVarsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to merge file and EnvVars config: %w", err)
	}
	result, err = mergeConfigs(result, flagsConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to merge EnvVars and flags config: %w", err)
	}
	result.APIVersionsConfig, err = getAPIConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get the API versions config: %w", err)
	}
	return result, nil
}

func (c *Config) PrettyPrint() (string, error) {
	confCopy := *c
	const secretPlaceholder = "xxxxx"
	confCopy.S3.SecretKey = secretPlaceholder
	confCopy.DB.Pass = secretPlaceholder
	prettyConf, err := json.MarshalIndent(&confCopy, "", "\t")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config with indent: %w", err)
	}
	return string(prettyConf), err
}

var defaultConfig = &Config{
	Listen:   "wss://localhost:8443",
	Base:     "./modules",
	OtelAddr: "otel.local:8148",
	Loader: Loader{
		Config: "fs",
		Files:  "fs",
	},
	Certs: CertsConfig{
		Type: "fs",
		Base: "./security/certs/server",
	},
	Validator: hardeningConfig.Validator{
		Type: "fs",
		Base: "./security/vconf",
	},
	ConfigFile:         "",
	Command:            "",
	LogDir:             "",
	Debug:              false,
	Service:            false,
	IsPrintVersionOnly: false,
}

func parseConfigFile(path string) (*Config, error) {
	var cfg Config
	cfgData, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(cfgData, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func parseFlags() *Config {
	c := &Config{}
	flag.StringVar(&c.Listen, "listen", "", "Listen IP:Port")
	flag.StringVar(&c.Base, "path", "", "Path to modules directory (default './modules')")
	flag.StringVar(&c.Certs.Base, "cpath", "", "Path to certificates directory (default './certs')")
	flag.StringVar(&c.Validator.Base, "vpath", "", "Path to validator directory (default './vconf')")
	flag.StringVar(&c.Loader.Config, "mcl", "", "Mode of config loader: [fs, s3, db]")
	flag.StringVar(&c.Validator.Type, "vcl", "", "Mode of validator loader: [fs, s3, db]")
	flag.StringVar(&c.Loader.Files, "mfl", "", "Mode of files loader: [fs, s3]")
	flag.StringVar(&c.Certs.Type, "cfl", "", "Mode of certificates loader: [fs, s3]")
	flag.StringVar(&c.ConfigFile, "config", "", "Path to server config file")
	flag.StringVar(&c.Command, "command", "", `Command to service control (not required):
  install - install the service to the system
  uninstall - uninstall the service from the system
  start - start the service
  stop - stop the service
  status - status of the service`)
	flag.StringVar(&c.LogDir, "logdir", "", "System option to define log directory to vxserver")
	flag.StringVar(&c.OtelAddr, "oteladdr", "", "System option to define log opentelemetry address")
	flag.BoolVar(&c.Debug, "debug", false, "System option to run vxserver in debug mode")
	flag.BoolVar(&c.IsProfiling, "profiling", false, "System option to run vxserver in profiling mode")
	flag.BoolVar(&c.Service, "service", false, "System option to run vxserver as a service")
	flag.BoolVar(&c.IsPrintVersionOnly, "version", false, "Print current version of vxserver and exit")
	flag.Parse()
	return c
}

func parseEnvVars() *Config {
	c := &Config{}
	c.Listen = os.Getenv("LISTEN")
	c.Base = os.Getenv("BASE_PATH")
	c.Certs.Base = os.Getenv("CERTS_PATH")
	c.Certs.Type = os.Getenv("CERTS_LOADER")
	c.Validator.Base = os.Getenv("VALID_PATH")
	c.Validator.Type = os.Getenv("VALID_LOADER")
	c.Loader.Config = os.Getenv("CONFIG_LOADER")
	c.Loader.Files = os.Getenv("FILES_LOADER")
	c.LogDir = os.Getenv("LOG_DIR")
	c.OtelAddr = os.Getenv("OTEL_ADDR")
	// bool parameters can only be passed as flags
	c.IsProfiling = false
	// bool parameters can only be passed as flags
	c.Debug = false
	c.DB.Host = os.Getenv("DB_HOST")
	c.DB.Port = os.Getenv("DB_PORT")
	c.DB.User = os.Getenv("DB_USER")
	c.DB.Pass = os.Getenv("DB_PASS")
	c.DB.Name = os.Getenv("DB_NAME")
	c.S3.AccessKey = os.Getenv("MINIO_ACCESS_KEY")
	c.S3.SecretKey = os.Getenv("MINIO_SECRET_KEY")
	c.S3.BucketName = os.Getenv("MINIO_BUCKET_NAME")
	c.S3.Endpoint = os.Getenv("MINIO_ENDPOINT")
	return c
}

const vxserverAPIVersionEnvVarName = "VXSERVER_API_VERSION_POLICY_"

func getAPIVersions(apiVersions map[string]vxproto.EndpointConnectionPolicy) (vxproto.ServerAPIVersionsConfig, error) {
	if len(apiVersions) == 0 {
		return nil, nil
	}
	result := make(vxproto.ServerAPIVersionsConfig)
	for version, policy := range apiVersions {
		if err := createAPIVersionsConfigEntry(
			result,
			version,
			func() (vxproto.EndpointConnectionPolicy, error) { return policy, nil },
		); err != nil {
			return nil, fmt.Errorf("failed to register the API version %s: %w", version, err)
		}
	}
	return result, nil
}

func mergeWithUserDefinedVersions(
	predefinedVersions vxproto.ServerAPIVersionsConfig,
	userDefinedVersions vxproto.ServerAPIVersionsConfig,
) (vxproto.ServerAPIVersionsConfig, error) {
	if err := mergo.Merge(&predefinedVersions, userDefinedVersions, mergo.WithOverride); err != nil {
		return nil, fmt.Errorf("failed to merge the old versions with the user defined versions: %w", err)
	}
	return predefinedVersions, nil
}

var apiVersions = map[string]vxproto.EndpointConnectionPolicy{
	"v1": vxproto.EndpointConnectionPolicyAllow,
}

func getAPIConfig() (vxproto.ServerAPIVersionsConfig, error) {
	envVarsVersions, err := getAPIConfigFromEnvVars()
	if err != nil {
		return nil, fmt.Errorf("failed to get the API config from environment variables: %w", err)
	}
	predefinedVersions, err := getAPIVersions(apiVersions)
	if err != nil {
		return nil, err
	}
	apiConfig, err := mergeWithUserDefinedVersions(predefinedVersions, envVarsVersions)
	if err != nil {
		return nil, fmt.Errorf("failed to merge with the user defined versions: %w", err)
	}
	return apiConfig, nil
}

var errVersionAlreadyExists = errors.New("version already exists")

func createAPIVersionsConfigEntry(
	result vxproto.ServerAPIVersionsConfig,
	version string,
	getConnectionPolicy func() (vxproto.EndpointConnectionPolicy, error),
) error {
	if _, ok := result[version]; ok {
		return errVersionAlreadyExists
	}
	connPolicy, err := getConnectionPolicy()
	if err != nil {
		return err
	}
	result[version] = &vxproto.ServerAPIConfig{
		Version:          url.QueryEscape(version),
		ConnectionPolicy: connPolicy,
	}
	return nil
}

func getAPIConfigFromEnvVars() (vxproto.ServerAPIVersionsConfig, error) {
	result := make(vxproto.ServerAPIVersionsConfig)
	allEnvs := os.Environ()
	for _, e := range allEnvs {
		if !strings.HasPrefix(e, vxserverAPIVersionEnvVarName) {
			continue
		}
		connPolicy, err := parseAPIVersionConnectionPolicy(e, "=")
		if err != nil {
			logrus.WithField("component", "deamon").WithError(err).
				Errorf("failed to parse the API version connection policy from the environment variable \"%s\"", e)
			continue
		}
		if err = createAPIVersionsConfigEntry(
			result,
			connPolicy.Version,
			func() (vxproto.EndpointConnectionPolicy, error) {
				var endpointConnectionPolicy vxproto.EndpointConnectionPolicy
				if err := endpointConnectionPolicy.FromString(connPolicy.ConnectionPolicy); err != nil {
					return 0, err
				}
				return endpointConnectionPolicy, nil
			}); err != nil {
			return nil, fmt.Errorf(
				"failed to register the API version %s with a connection policy \"%s\": %w",
				connPolicy.Version,
				connPolicy.ConnectionPolicy,
				err,
			)
		}
	}
	return result, nil
}

type versionConnectionPolicy struct {
	Version          string
	ConnectionPolicy string
}

func parseAPIVersionConnectionPolicy(envVar string, sep string) (*versionConnectionPolicy, error) {
	splitIdx := strings.Index(envVar, sep)
	if splitIdx == -1 {
		return nil, fmt.Errorf("separator \"%s\" not found in the environment variable \"%s\"", sep, envVar)
	}
	name, val := envVar[:splitIdx], envVar[splitIdx+1:]
	if len(name) < len(vxserverAPIVersionEnvVarName) {
		return nil, fmt.Errorf("name of the environment variable containing the API version connection policy \"%s\" "+
			"is of incorrect format", name)
	}
	name = name[len(vxserverAPIVersionEnvVarName):]
	if len(name) == 0 {
		return nil, fmt.Errorf("version value in the environment variable containing the API version connection "+
			"policy \"%s\" is empty", name)
	}
	if len(val) == 0 {
		return nil, fmt.Errorf("value of the environment variable %s cannot be empty", name)
	}
	return &versionConnectionPolicy{
		Version:          name,
		ConnectionPolicy: val,
	}, nil
}

func mergeConfigs(dst *Config, src *Config) (*Config, error) {
	dstCopy := *dst
	if err := mergo.Merge(
		&dstCopy,
		src,
		mergo.WithOverride,
		mergo.WithTransformers(boolTransformer{}),
		mergo.WithTransformers(serverAPIExternalConfigTransformer{}),
	); err != nil {
		return nil, fmt.Errorf("failed to merge configs: %w", err)
	}
	return &dstCopy, nil
}

type boolTransformer struct{}

func (t boolTransformer) Transformer(typ reflect.Type) func(dst reflect.Value, src reflect.Value) error {
	var boolVar bool
	if typ == reflect.TypeOf(boolVar) {
		return func(dst reflect.Value, src reflect.Value) error {
			if dst.CanSet() {
				dst.Set(src)
			}
			return nil
		}
	}
	return nil
}

type serverAPIExternalConfigTransformer struct{}

func (t serverAPIExternalConfigTransformer) Transformer(typ reflect.Type) func(dst, src reflect.Value) error {
	var serverApiConfigVar map[string]*vxproto.ServerAPIExternalConfig
	if typ != reflect.TypeOf(serverApiConfigVar) {
		return nil
	}
	return func(dst reflect.Value, src reflect.Value) error {
		if src.IsNil() {
			return nil
		}
		if dst.CanSet() {
			dst.Set(src)
		}
		return nil
	}
}
