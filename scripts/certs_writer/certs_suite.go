package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/sirupsen/logrus"
)

var certsSuiteDir = ""

const (
	vxcaCertFile     = "vxca.cert"
	vxcaKeyFile      = "vxca.key"
	caCertFile       = "ca.cert"
	caKeyFile        = "ca.key"
	scCertFile       = "sc.cert"
	scKeyFile        = "sc.key"
	iacCertFile      = "iac.cert"
	iacKeyFile       = "iac.key"
	ltacCertFileTmpl = "ltac_%s.cert"
	ltacKeyFileTmpl  = "ltac_%s.key"
)

var (
	notBefore = setToMidnight(time.Now().UTC().AddDate(0, 0, -1))
	vxcaTmpl  = x509.Certificate{
		SerialNumber:          big.NewInt(42),
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		NotBefore:             notBefore,
		Subject: pkix.Name{
			CommonName: "*",
		},
	}
	caTmpl = x509.Certificate{
		SerialNumber:          big.NewInt(43),
		BasicConstraintsValid: true,
		IsCA:                  true,
		DNSNames:              []string{"VX CA"},
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		NotBefore:             notBefore,
		Subject: pkix.Name{
			CommonName: "*",
		},
	}
	scTmpl = x509.Certificate{
		SerialNumber:          big.NewInt(44),
		BasicConstraintsValid: true,
		IsCA:                  false,
		DNSNames:              []string{},
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		NotBefore:             notBefore,
		Subject:               pkix.Name{},
	}
	iacTmpl = x509.Certificate{
		SerialNumber:          big.NewInt(45),
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		NotBefore:             notBefore,
		Subject: pkix.Name{
			CommonName: "IAC Cert",
		},
	}
	ltacTmpl = x509.Certificate{
		BasicConstraintsValid: true,
		IsCA:                  false,
		KeyUsage:              x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
		NotBefore:             notBefore,
		SignatureAlgorithm:    x509.PureEd25519,
	}
)

func getNotAfterParam(u ExpirationTimeUnit) time.Time {
	t := time.Now().UTC().AddDate(u.Years, u.Months, u.Days+1)
	return setToMidnight(t)
}

func setToMidnight(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.UTC().Location())
}

func setNotAfterParams(expTimeConfig ExpirationTimeConfig) {
	vxcaTmpl.NotAfter = getNotAfterParam(expTimeConfig.VXCA)
	caTmpl.NotAfter = getNotAfterParam(expTimeConfig.SCA)
	scTmpl.NotAfter = getNotAfterParam(expTimeConfig.SC)
	iacTmpl.NotAfter = getNotAfterParam(expTimeConfig.IAC)
	ltacTmpl.NotAfter = getNotAfterParam(expTimeConfig.LTAC)
}

func generateCertificatesSuite(c *CertsSuiteConfig, expTimeConfig ExpirationTimeConfig) error {
	setNotAfterParams(expTimeConfig)

	var err error
	certsSuiteDir, err = createCertsSuiteDir(c.OutDir, c.SCServerName)
	if err != nil {
		return fmt.Errorf("failed to create the certificates out directory: %w", err)
	}
	setServerName(c.SCServerName)
	vxcaKey, isVXCANew, err := generateVXCA()
	if err != nil {
		return fmt.Errorf("failed to generate the VXCA certificate: %w", err)
	}
	if !isVXCANew {
		logrus.Info("reusing an existing VXCA")
	}
	if err := createSBHFile(vxcaKey); err != nil {
		return fmt.Errorf("failed to create an SBH file: %w", err)
	}
	caKey, isCANew, err := generateCA(vxcaKey, isVXCANew)
	if err != nil {
		return fmt.Errorf("failed to generate the CA certificate: %w", err)
	}
	if !isCANew {
		logrus.Info("reusing an existing CA")
	}
	isSCNew, err := generateSC(caKey, isCANew)
	if err != nil {
		return fmt.Errorf("failed to generate the SC certificate: %w", err)
	}
	if !isSCNew {
		logrus.Info("reusing an existing SC")
	}
	isIACNew, err := generateIAC(vxcaKey, isVXCANew)
	if err != nil {
		return fmt.Errorf("failed to generate the IAC certificate: %w", err)
	}
	if !isIACNew {
		logrus.Info("reusing an existing IAC")
	}
	for _, certType := range []string{"aggregate", "external", "browser"} {
		isLTACNew, err := generateLTAC(caKey, isCANew, certType)
		if err != nil {
			return fmt.Errorf("failed to generate the %s LTAC certificate: %w", certType, err)
		}
		if !isLTACNew {
			logrus.Infof("reusing an existing %s LTAC", certType)
		}
	}

	return nil
}

func createCertsSuiteDir(baseDir string, serverName string) (string, error) {
	var err error
	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		return "", fmt.Errorf("failed to get the absolute path to the directory %s: %w", baseDir, err)
	}
	const dirPerm = 0o755
	if err := os.MkdirAll(baseDir, dirPerm); err != nil {
		return "", fmt.Errorf("failed to create the base directory %s: %w", baseDir, err)
	}
	if err := os.Chmod(baseDir, dirPerm); err != nil {
		return "", fmt.Errorf("failed to set the base directory permissions to %o: %w", dirPerm, err)
	}
	certsSuiteDir = filepath.Join(baseDir, serverName)
	if err := os.MkdirAll(certsSuiteDir, dirPerm); err != nil {
		return "", fmt.Errorf("failed to create a directory %s: %w", certsSuiteDir, err)
	}
	if err := os.Chmod(certsSuiteDir, dirPerm); err != nil {
		return "", fmt.Errorf("failed to set the created directory permissions to %o: %w", dirPerm, err)
	}
	return certsSuiteDir, nil
}

func setServerName(name string) {
	scTmpl.DNSNames = append(scTmpl.DNSNames, name)
	scTmpl.Subject.CommonName = name
}

func generateVXCA() (interface{}, bool, error) {
	const certFile, keyFile = vxcaCertFile, vxcaKeyFile
	key, err := checkIfAlreadyExists(certsSuiteDir, certFile, keyFile)
	if err != nil {
		return nil, false, err
	}
	if key != nil {
		return key, false, nil
	}
	key, err = generate(certsSuiteDir, &vxcaTmpl, &vxcaTmpl, nil, certFile, keyFile)
	return key, true, err
}

func generateCA(vxcaKey interface{}, isVXCANew bool) (interface{}, bool, error) {
	const certFile, keyFile = caCertFile, caKeyFile
	if !isVXCANew {
		key, err := checkIfAlreadyExists(certsSuiteDir, certFile, keyFile)
		if err != nil {
			return nil, false, err
		}
		if key != nil {
			return key, false, nil
		}
	}
	key, err := generate(certsSuiteDir, &caTmpl, &vxcaTmpl, vxcaKey, certFile, keyFile)
	return key, true, err
}

func generateSC(caKey interface{}, isCANew bool) (bool, error) {
	const certFile, keyFile = scCertFile, scKeyFile
	if !isCANew {
		key, err := checkIfAlreadyExists(certsSuiteDir, certFile, keyFile)
		if err != nil {
			return false, err
		}
		if key != nil {
			return false, nil
		}
	}
	_, err := generate(certsSuiteDir, &scTmpl, &caTmpl, caKey, certFile, keyFile)
	return true, err
}

func generateIAC(vxcaKey interface{}, isVXCANew bool) (bool, error) {
	const certFile, keyFile = iacCertFile, iacKeyFile
	if !isVXCANew {
		key, err := checkIfAlreadyExists(certsSuiteDir, certFile, keyFile)
		if err != nil {
			return false, err
		}
		if key != nil {
			return false, nil
		}
	}
	_, err := generate(certsSuiteDir, &iacTmpl, &vxcaTmpl, vxcaKey, iacCertFile, iacKeyFile)
	return true, err
}

func generateLTAC(caKey interface{}, isCANew bool, certType string) (bool, error) {
	var (
		err      error
		certFile = fmt.Sprintf(ltacCertFileTmpl, certType)
		keyFile  = fmt.Sprintf(ltacKeyFileTmpl, certType)
	)
	ltacTmplCopy := ltacTmpl
	ltacTmplCopy.SerialNumber, err = generateCertSerialNumber()
	if err != nil {
		return false, fmt.Errorf("failed to generate a certificate serial number: %w", err)
	}

	if !isCANew {
		key, err := checkIfAlreadyExists(certsSuiteDir, certFile, keyFile)
		if err != nil {
			return false, err
		}
		if key != nil {
			return false, nil
		}
	}
	_, err = generate(certsSuiteDir, &ltacTmplCopy, &caTmpl, caKey, certFile, keyFile)
	return true, err
}

const maxInt64 int64 = int64((^uint64(0)) >> 1)

func generateCertSerialNumber() (*big.Int, error) {
	unixNano := strconv.FormatInt(time.Now().UnixNano(), 10)
	nonce, err := rand.Int(rand.Reader, big.NewInt(maxInt64))
	if err != nil {
		return nil, fmt.Errorf("failed to get a random nonce: %w", err)
	}
	serialNumStr := unixNano + nonce.String()
	serialNum, ok := new(big.Int).SetString(serialNumStr, 10)
	if !ok {
		return nil, fmt.Errorf("failed to convert the string %s to a big int", serialNumStr)
	}
	return serialNum, nil
}

func checkIfAlreadyExists(outDir string, certFile string, keyFile string) (interface{}, error) {
	isCertExist, err := checkIfCertExists(outDir, certFile)
	if err != nil {
		return nil, err
	}
	if !isCertExist {
		return nil, nil
	}
	key, err := checkIfKeyExists(outDir, keyFile)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func checkIfCertExists(outDir string, certFile string) (bool, error) {
	certData, err := getFileContents(outDir, certFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return false, nil
		}
		return false, err
	}
	if _, err := decodePEMData(certData, pemTypeCertificate); err != nil {
		return false, fmt.Errorf("failed to decode the read PEM data: %w", err)
	}
	return true, nil
}

func decodePEMData(data []byte, expectedPEMType string) ([]byte, error) {
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, fmt.Errorf("passed data does not contain a valid certificate")
	}
	if block.Type != expectedPEMType {
		return nil, fmt.Errorf("the decoded pem block has a wrong type \"%s\" (expected \"%s\")", block.Type, expectedPEMType)
	}
	return block.Bytes, nil
}

func checkIfKeyExists(outDir string, keyFile string) (interface{}, error) {
	keyData, err := getFileContents(outDir, keyFile)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	keyBlock, err := decodePEMData(keyData, pemTypePrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the read PEM data: %w", err)
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the key as a PKCS8 private key: %w", err)
	}
	return key, nil
}

func getFileContents(outDir string, certFile string) ([]byte, error) {
	path, err := getFilePath(outDir, certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get the cert file path: %w", err)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read the file %s: %w", path, err)
	}
	return contents, nil
}

func generate(
	outDir string,
	tmpl *x509.Certificate,
	parentTmpl *x509.Certificate,
	parentKey interface{},
	outCertFile string,
	outKeyFile string,
) (interface{}, error) {
	cert, key, err := createCert(tmpl, parentTmpl, parentKey)
	if err != nil {
		return nil, err
	}
	priv := key.(ed25519.PrivateKey)
	if err := writeCert(outDir, cert, priv, outCertFile, outKeyFile); err != nil {
		return nil, err
	}
	return key, err
}

func getFilePath(outDir string, name string) (string, error) {
	joinedPath := filepath.Join(outDir, name)
	path, err := filepath.Abs(joinedPath)
	if err != nil {
		return "", fmt.Errorf("failed to get the absolute path to %s: %w", joinedPath, err)
	}
	return path, nil
}

func createCert(tmpl *x509.Certificate, parentTmpl *x509.Certificate, parentKey interface{}) ([]byte, interface{}, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate a key pair: %w", err)
	}
	if parentKey == nil {
		parentKey = priv
	}
	cert, err := x509.CreateCertificate(rand.Reader, tmpl, parentTmpl, pub, parentKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create a certificate: %w", err)
	}
	return cert, priv, err
}

const (
	pemTypeCertificate = "CERTIFICATE"
	pemTypePrivateKey  = "PRIVATE KEY"
)

func writeCert(outDir string, cert []byte, key interface{}, certFile string, keyFile string) error {
	certFilePath, err := getFilePath(outDir, certFile)
	if err != nil {
		return fmt.Errorf("failed to get the path of the cert file %s: %w", certFile, err)
	}
	if err := writePEM(certFilePath, cert, pemTypeCertificate); err != nil {
		return fmt.Errorf("failed to write the certificate file %v: %w", certFilePath, err)
	}
	keyFilePath, err := getFilePath(outDir, keyFile)
	if err != nil {
		return fmt.Errorf("failed to get the path of the key file %s: %w", keyFile, err)
	}
	marshalledKey, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal the key: %w", err)
	}
	if err := writePEM(keyFilePath, marshalledKey, pemTypePrivateKey); err != nil {
		return fmt.Errorf("failed to write the key file %s: %w", keyFilePath, err)
	}
	return nil
}

func writePEM(dst string, data []byte, pemType string) error {
	dstDir := filepath.Dir(dst)
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return fmt.Errorf("failed to create the file directory %s: %w", dstDir, err)
	}
	f, err := os.Create(dst)
	if err != nil {
		return fmt.Errorf("failed to write a PEM-encoded file %s: %w", dst, err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{
		Type:  pemType,
		Bytes: data,
	}); err != nil {
		return fmt.Errorf("failed to encode a PEM block of the type %s: %w", pemType, err)
	}
	return nil
}

func createSBHFile(vxcaKey interface{}) error {
	sbh, err := generateSBH(vxcaKey)
	if err != nil {
		return fmt.Errorf("SBH generation failed: %w", err)
	}
	sbhFilePath, err := getFilePath(certsSuiteDir, "sbh.json")
	if err != nil {
		return fmt.Errorf("failed to get the SBH file path: %w", err)
	}
	dstDir := filepath.Dir(sbhFilePath)
	if err := os.MkdirAll(dstDir, 0o700); err != nil {
		return fmt.Errorf("failed to create the file directory %s: %w", dstDir, err)
	}
	f, err := os.Create(sbhFilePath)
	if err != nil {
		return fmt.Errorf("failed to open the file %s for writing: %w", sbhFilePath, err)
	}
	defer f.Close()
	if _, err := f.Write(sbh); err != nil {
		return fmt.Errorf("failed to write the SBH to file %s: %w", sbhFilePath, err)
	}
	return nil
}

func generateSBH(vxcaKey interface{}) ([]byte, error) {
	key := vxcaKey.(ed25519.PrivateKey)
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return nil, fmt.Errorf("failed to generate the SBH token: %w", err)
	}
	sig := ed25519.Sign(key, token)
	sbh := append(sig, token...)
	return sbh, nil
}
