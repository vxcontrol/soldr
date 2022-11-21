package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"

	"soldr/scripts/sbh_generator/nonce"
	"soldr/scripts/sbh_generator/nonce/random"
	"soldr/scripts/sbh_generator/nonce/stub"
	"soldr/scripts/sbh_generator/printer"
	jsonPrinter "soldr/scripts/sbh_generator/printer/file/json"
	"soldr/scripts/sbh_generator/printer/stdout"
	"soldr/scripts/sbh_generator/sk"
	"soldr/scripts/sbh_generator/sk/static"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

type Config struct {
	VXCAKeyFile string
	ExpiresAt   string
	SBHFile     string
	Nonce       string
	Version     string
	Force       bool
}

var errFlagsMisused = errors.New("flags misused")

func parseFlags() (*Config, error) {
	c := &Config{}
	flag.StringVar(&c.VXCAKeyFile, "key", "", "required: path to the file that contains a key to sign SBH")
	flag.StringVar(&c.ExpiresAt, "expires", "", "required: SBH expiration date in the UTC format (see RFC3339)")
	flag.StringVar(&c.SBHFile, "file", "", "path to the SBH JSON file into which the SBH nonce will be injected")
	flag.StringVar(&c.Version, "version", "", "version of the SBH token (key in the SBH data file)")
	flag.StringVar(&c.Nonce, "nonce", "", "base64-encoded SBH nonce (if empty, a random nonce is used)")
	flag.BoolVar(&c.Force, "force", false, "rewrite the SBH for the provided version")
	flag.Parse()

	if len(c.VXCAKeyFile) == 0 {
		return nil, fmt.Errorf("VXCA key file path must be defined: (%w)", errFlagsMisused)
	}
	if len(c.ExpiresAt) == 0 {
		return nil, fmt.Errorf("ExpiresAt value must be defined: (%w)", errFlagsMisused)
	}
	if len(c.Version) == 0 && len(c.SBHFile) != 0 {
		return nil, fmt.Errorf("version value must be defined: (%w)", errFlagsMisused)
	}
	return c, nil
}

func run() error {
	conf, err := parseFlags()
	if err != nil {
		if errors.Is(err, errFlagsMisused) {
			flag.Usage()
		}
		return fmt.Errorf("failed to extract the configuration from the passed flags: %w", err)
	}
	err = touchSBHFile(conf.SBHFile)
	if err != nil {
		return fmt.Errorf("failed to write empty SBH file: %w", err)
	}
	nonceProvider, err := getNonceProvider(conf.Nonce)
	if err != nil {
		return fmt.Errorf("failed to get a nonce provider: %w", err)
	}
	skProvider, err := sk.NewProvider(&sk.Config{
		Static: &static.Config{
			VXCAKeyFile: conf.VXCAKeyFile,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize an SK provider: %w", err)
	}
	tok, err := generateJWT(skProvider, nonceProvider, conf.ExpiresAt)
	if err != nil {
		return fmt.Errorf("failed to generate a JWT token: %w", err)
	}
	printer, err := getPrinter(conf.SBHFile, conf.Force)
	if err != nil {
		return fmt.Errorf("failed to get the SBH token printer: %w", err)
	}
	if err := printer.Print(conf.Version, tok); err != nil {
		return fmt.Errorf("failed to print the SBH token: %w", err)
	}
	return nil
}

func isFileExists(filePath string) bool {
	info, err := os.Stat(filePath)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func touchSBHFile(sbhFilePath string) error {
	if !isFileExists(sbhFilePath) {
		return ioutil.WriteFile(sbhFilePath, []byte(`{"v1":{}}`), 0640)
	}
	return nil
}

func getNonceProvider(configNonce string) (nonce.Provider, error) {
	if len(configNonce) == 0 {
		return nonce.NewProvider(&nonce.Config{
			Random: &random.Config{},
		})
	}
	nonceBytes, err := base64.StdEncoding.DecodeString(configNonce)
	if err != nil {
		return nil, fmt.Errorf("failed to base64-decode the passed nonce value %s: %w", configNonce, err)
	}
	return nonce.NewProvider(&nonce.Config{
		Stub: &stub.Config{
			Nonce: nonceBytes,
		},
	})
}

func getPrinter(jsonConfigPath string, force bool) (printer.Printer, error) {
	if len(jsonConfigPath) == 0 {
		return printer.NewPrinter(&printer.Config{
			Stdout: &stdout.Config{},
		})
	}
	return printer.NewPrinter(&printer.Config{
		JSON: &jsonPrinter.Config{
			File:  jsonConfigPath,
			Force: force,
		},
	})
}

type Claims struct {
	Nonce []byte `json:"nonce"`
	jwt.StandardClaims
}

func generateJWT(p sk.Provider, n nonce.Provider, expiresAt string) (string, error) {
	expClaim, err := getExpClaim(expiresAt)
	if err != nil {
		return "", fmt.Errorf("failed to get the exp field for the JWT-token: %w", err)
	}
	sbhNonce, err := n.GetNonce()
	if err != nil {
		return "", fmt.Errorf("failed to generate a nonce: %w", err)
	}
	claims := &Claims{
		Nonce: sbhNonce,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: expClaim,
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodEdDSA, claims)
	sk, err := p.GetSecretKey()
	if err != nil {
		return "", fmt.Errorf("failed to get an SK from the SK provider: %w", err)
	}
	signedTok, err := tok.SignedString(sk)
	if err != nil {
		return "", fmt.Errorf("failed to sign the token: %w", err)
	}
	return signedTok, nil
}

func getExpClaim(expiresAt string) (int64, error) {
	t, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the passed expiresAt value %s: %w", expiresAt, err)
	}
	return t.Unix(), nil
}
