package testing

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"time"
)

const (
	pemTypeCertificate = "CERTIFICATE"
	pemTypePrivateKey  = "PRIVATE KEY"
)

var RootTmpl = &x509.Certificate{
	KeyUsage:              x509.KeyUsageCertSign,
	BasicConstraintsValid: true,
	IsCA:                  true,
}

func newKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate an ed25519 keypair: %w", err)
	}
	return pub, priv, nil
}

func GenerateCert(
	certTemplate, parentTemplate, defaultTemplate *x509.Certificate, parentPriv interface{},
) ([]byte, ed25519.PrivateKey, error) {
	pub, priv, err := newKeypair()
	if err != nil {
		return nil, nil, err
	}

	pemCert, err := createCert(certTemplate, parentTemplate, defaultTemplate, pub, parentPriv)
	if err != nil {
		return nil, nil, err
	}
	return pemCert, priv, nil
}

func GenerateSelfSignedCert(tmpl, defaultTemplate *x509.Certificate) ([]byte, ed25519.PrivateKey, error) {
	pub, priv, err := newKeypair()
	if err != nil {
		return nil, nil, err
	}

	pemCert, err := createCert(tmpl, tmpl, defaultTemplate, pub, priv)
	if err != nil {
		return nil, nil, err
	}
	return pemCert, priv, nil
}

func createCert(
	certTemplate *x509.Certificate,
	parentTemplate *x509.Certificate,
	defaultTemplate *x509.Certificate,
	pub interface{},
	parentPriv interface{},
) ([]byte, error) {
	certTemplate, err := fillMissingCertFields(certTemplate, defaultTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to fill certificate template missing fields: %w", err)
	}
	parentTemplate, err = fillMissingCertFields(parentTemplate, defaultTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to fill parent certificate template missing fields: %w", err)
	}
	cert, err := x509.CreateCertificate(rand.Reader, certTemplate, parentTemplate, pub, parentPriv)
	if err != nil {
		return nil, fmt.Errorf("failed to create a certificate: %w", err)
	}
	return DERCertToPEM(cert), nil
}

func fillMissingCertFields(cert *x509.Certificate, defaultCert *x509.Certificate) (*x509.Certificate, error) {
	if defaultCert == nil {
		var err error
		defaultCert, err = getDefaultCertTemplate()
		if err != nil {
			return nil, fmt.Errorf("failed to get a default certificate: %w", err)
		}
	}
	if cert == nil {
		return defaultCert, nil
	}
	if cert.SerialNumber == nil {
		cert.SerialNumber = defaultCert.SerialNumber
	}
	if cert.Subject.Organization == nil {
		cert.Subject.Organization = defaultCert.Subject.Organization
	}
	if cert.NotBefore.IsZero() {
		cert.NotBefore = defaultCert.NotBefore
	}
	if cert.NotAfter.IsZero() {
		cert.NotAfter = defaultCert.NotAfter
	}
	if cert.KeyUsage == 0 {
		cert.KeyUsage = defaultCert.KeyUsage
	}
	return cert, nil
}

const defaultCertExpiresIn = time.Hour * 24 * 180

func getDefaultCertTemplate() (*x509.Certificate, error) {
	serialNum, err := getRandomBigInt()
	if err != nil {
		return nil, fmt.Errorf("failed to generate a serial num: %w", err)
	}
	return &x509.Certificate{
		SerialNumber: serialNum,
		Subject: pkix.Name{
			Organization: []string{"HoHo Co"},
		},
		NotBefore: time.Now(),
		NotAfter:  time.Now().Add(defaultCertExpiresIn),
		KeyUsage:  x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
	}, nil
}

func getRandomBigInt() (*big.Int, error) {
	i, err := rand.Int(rand.Reader, big.NewInt(2147483647))
	if err != nil {
		return nil, fmt.Errorf("failed to generate a random big int: %w", err)
	}
	return i, nil
}

func CompareErrors(expected error, actual error) error {
	if expected == nil && actual == nil {
		return nil
	}
	genErr := func() error {
		return fmt.Errorf("expected err \"%w\", got \"%v\"", expected, actual)
	}
	if (expected != nil && actual == nil) || (expected == nil && actual != nil) {
		return genErr()
	}
	if expected.Error() != actual.Error() {
		return genErr()
	}
	return nil
}

func PrivateKeyToPEM(key interface{}) ([]byte, error) {
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal the private key: %w", err)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  pemTypePrivateKey,
		Bytes: keyDER,
	}), nil
}

func DERCertToPEM(der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  pemTypeCertificate,
		Bytes: der,
	})
}
