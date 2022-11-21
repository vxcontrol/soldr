package main

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	CertsSuite           *CertsSuiteConfig
	ExpirationTimeConfig *ExpirationTimeConfig
}

type ExpirationTimeConfig struct {
	VXCA ExpirationTimeUnit
	SCA  ExpirationTimeUnit
	IAC  ExpirationTimeUnit
	SC   ExpirationTimeUnit
	LTAC ExpirationTimeUnit
}

type ExpirationTimeUnit struct {
	Years  int
	Months int
	Days   int
}

var expTimeRE = regexp.MustCompile(`(?P<years>\d+Y)?(?P<months>\d+M)?(?P<days>\d+D)?`)

func expirationTimeFromString(s string) (ExpirationTimeUnit, error) {
	u := ExpirationTimeUnit{}
	parts := expTimeRE.FindStringSubmatch(s)
	if len(parts) != 4 || (len(parts[1]) == 0 && len(parts[2]) == 0 && len(parts[3]) == 0) {
		return u, fmt.Errorf("failed to extract expiration time from the passed string %s", s)
	}

	getParseErr := func(units string, origErr error) error {
		return fmt.Errorf("failed to parse the %s part of the expiration time string: %w", units, origErr)
	}
	var err error
	u.Years, err = parseExpirationTimePart(parts[1])
	if err != nil {
		return u, getParseErr("years", err)
	}
	u.Months, err = parseExpirationTimePart(parts[2])
	if err != nil {
		return u, getParseErr("months", err)
	}
	u.Days, err = parseExpirationTimePart(parts[3])
	if err != nil {
		return u, getParseErr("days", err)
	}
	return u, nil
}

func parseExpirationTimePart(p string) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	pRunes := []rune(p)
	lastPRune := pRunes[len(pRunes)-1]
	if lastPRune < 48 || lastPRune > 57 {
		p = string(pRunes[:len(pRunes)-1])
	}
	pInt, err := strconv.Atoi(p)
	if err != nil {
		return -1, fmt.Errorf("failed to parse the passed part %s as an int", p)
	}
	return pInt, nil
}

type CertsSuiteConfig struct {
	Active       bool
	OutDir       string
	SCServerName string
}

func NewConfig() (*Config, error) {
	c, err := getConfigFromFlags()
	if err != nil {
		return nil, fmt.Errorf("failed to get configuration from flags: %w", err)
	}
	return c, nil
}

func getConfigFromFlags() (*Config, error) {
	c := &Config{
		CertsSuite: &CertsSuiteConfig{},
	}
	flag.BoolVar(&c.CertsSuite.Active, "cert_gen", true, "generate certificates suite")
	flag.StringVar(&c.CertsSuite.OutDir, "dst", "", "certificates destination directory")
	flag.StringVar(&c.CertsSuite.SCServerName, "server_name", "", "SC server name")
	var expirationTimeConfigFile string
	flag.StringVar(&expirationTimeConfigFile, "expiration_time_config", "./expiration_time.yaml", "certificates expiration time config file")
	flag.Parse()
	var err error
	c.ExpirationTimeConfig, err = getExpirationTimeConfig(expirationTimeConfigFile)
	if err != nil {
		return nil, err
	}
	if err := checkFlagsConfig(c); err != nil {
		flag.Usage()
		return nil, err
	}
	return c, nil
}

func checkFlagsConfig(c *Config) error {
	if err := checkCertsSuiteConfig(c.CertsSuite); err != nil {
		return fmt.Errorf("bad configuration for the certificates suite generation: %w", err)
	}
	return nil
}

func checkCertsSuiteConfig(c *CertsSuiteConfig) error {
	if !c.Active {
		return nil
	}
	if len(c.OutDir) == 0 {
		return fmt.Errorf("no certificates destination directory specified")
	}
	if len(c.SCServerName) == 0 {
		var err error
		c.SCServerName, err = generateRandomServerName()
		if err != nil {
			return fmt.Errorf("failed to generate a random server name: %w", err)
		}
	}
	return nil
}

func generateRandomServerName() (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	randSuffix := hex.EncodeToString(buf)
	return fmt.Sprintf("vx_%s", randSuffix), nil
}

func getExpirationTimeConfig(expirationTimeConfigFile string) (*ExpirationTimeConfig, error) {
	absPath, err := filepath.Abs(filepath.Join("./", expirationTimeConfigFile))
	if err != nil {
		return nil, fmt.Errorf("failed to get the absolute path to the configuration file %s: %w", expirationTimeConfigFile, err)
	}
	contents, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read the expiration time configuration file %s: %w", absPath, err)
	}
	var expirationTimes map[string]string
	if err := yaml.Unmarshal(contents, &expirationTimes); err != nil {
		return nil, fmt.Errorf("failed to read the expiration time configuration file: %w", err)
	}
	return parseExpirationTimeConfig(expirationTimes)
}

func parseExpirationTimeConfig(expirationTimes map[string]string) (*ExpirationTimeConfig, error) {
	times := &ExpirationTimeConfig{}
	defaultExpirationTimes := map[string]ExpirationTimeUnit{
		"vxca": {
			Years: 20,
		},
		"sca": {
			Years: 5,
		},
		"iac": {
			Years: 5,
		},
		"sc": {
			Years: 1,
		},
		"ltac": {
			Years: 1,
		},
	}

	getFieldValueToSet := func(name string, defaultExpirationTime ExpirationTimeUnit) reflect.Value {
		expTimeStr, ok := expirationTimes[name]
		if !ok {
			logrus.Warnf("expiration time for %s not found in the passed config, using a default", name)
			return reflect.ValueOf(defaultExpirationTime)
		}
		u, err := expirationTimeFromString(expTimeStr)
		if err != nil {
			logrus.
				WithError(err).
				Warnf("failed to parse the expiration time for %s, default is used", name)
			return reflect.ValueOf(defaultExpirationTime)
		}
		return reflect.ValueOf(u)
	}

	v := reflect.ValueOf(times).Elem()
	for i := 0; i < v.NumField(); i++ {
		fieldName := v.Type().Field(i).Name
		fieldNameLower := strings.ToLower(fieldName)
		defaultExpirationTime, ok := defaultExpirationTimes[fieldNameLower]
		if !ok {
			return nil, fmt.Errorf("default expiration time not found for the field %s", fieldNameLower)
		}
		v.FieldByName(fieldName).Set(getFieldValueToSet(fieldName, defaultExpirationTime))
	}
	return times, nil
}
