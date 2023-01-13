package vm

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"

	vxcommonErrors "soldr/pkg/errors"
	"soldr/pkg/hardening/luavm/certs"
	storeTypes "soldr/pkg/hardening/luavm/store/types"
)

type pemType string

const (
	pemTypeCertificate pemType = "CERTIFICATE"
	pemTypeKey         pemType = "PRIVATE KEY"
)

func derToPem(der []byte, t pemType) ([]byte, error) {
	switch t {
	case pemTypeCertificate:
	case pemTypeKey:
	default:
		return nil, fmt.Errorf("unknown pem type %s", t)
	}
	return pem.EncodeToMemory(&pem.Block{
		Type:  string(t),
		Bytes: der,
	}), nil
}

func (v *vm) getLTACCertificate(storeKey []byte) (*tls.Certificate, error) {
	ltac, err := v.GetLTAC(storeKey)
	if err != nil {
		msg := "failed to get the LTAC certificate for connection"
		if errors.Is(err, storeTypes.ErrNotInitialized) {
			return nil, fmt.Errorf(
				"%s (%v): %w",
				msg, err, vxcommonErrors.ErrConnectionInitializationRequired,
			)
		}
		return nil, fmt.Errorf("%s: %w", msg, err)
	}
	ltacPem, err := derToPem(ltac.Cert, pemTypeCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert LTAC to pem: %w", err)
	}
	scaPem, err := derToPem(ltac.CA, pemTypeCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the LTAC CA to pem: %w", err)
	}
	ltacWithChain := append(ltacPem, scaPem...)
	ltacKeyPem, err := derToPem(ltac.Key, pemTypeKey)
	if err != nil {
		return nil, fmt.Errorf("failed to convert LTAC key to pem: %w", err)
	}
	cert, err := tls.X509KeyPair(ltacWithChain, ltacKeyPem)
	if err != nil {
		return nil, fmt.Errorf("failed to compose the LTAC cert and key into a tls.Certificate: %w", err)
	}
	return &cert, nil
}

func getVXCACertsPool(certsProvider certs.CertProvider) (*x509.CertPool, error) {
	vxca, err := certsProvider.VXCA()
	if err != nil {
		return nil, fmt.Errorf("failed to fetch the VXCA certificate from the certs provider: %w", err)
	}
	vxcaPEM, err := derToPem(vxca, pemTypeCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the VXCA certificate to the PEM format: %w", err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(vxcaPEM) {
		return nil, fmt.Errorf("failed to append the VXCA certificate to the certificates pool")
	}
	return pool, nil
}

func getCertPool(der []byte) (*x509.CertPool, error) {
	certPEM, err := derToPem(der, pemTypeCertificate)
	if err != nil {
		return nil, fmt.Errorf("failed to convert the passed certificate into pem format: %w", err)
	}
	certPool := x509.NewCertPool()
	if !certPool.AppendCertsFromPEM(certPEM) {
		return nil, fmt.Errorf("failed to compose the SCA certs pool")
	}
	return certPool, nil
}

func verifyCertsChain(ltac *x509.Certificate, sca []byte, vxcaPool *x509.CertPool) error {
	scaCertPool, err := getCertPool(sca)
	if err != nil {
		return err
	}
	chains, err := ltac.Verify(x509.VerifyOptions{
		Intermediates: scaCertPool,
		Roots:         vxcaPool,
	})
	if err != nil {
		return err
	}
	if len(chains) != 1 {
		return fmt.Errorf("expected 1 chain, got %d", len(chains))
	}
	if len(chains[0]) != 3 {
		return fmt.Errorf("expected the chains of length 3, got %d", len(chains[0]))
	}
	return nil
}

func (v *vm) checkReceivedLTAC(ltacDER []byte, key []byte, sca []byte) error {
	ltac, err := x509.ParseCertificate(ltacDER)
	if err != nil {
		return fmt.Errorf("failed to parse the certificate: %w", err)
	}
	vxcaPool, err := getVXCACertsPool(v.CertProvider)
	if err != nil {
		return fmt.Errorf("failed to get the VXCA pool: %w", err)
	}
	if err := verifyCertsChain(ltac, sca, vxcaPool); err != nil {
		return fmt.Errorf("failed to verify the certificates chain: %w", err)
	}
	if err := checkKeyPairValidity(ltac.PublicKey, key); err != nil {
		return fmt.Errorf("key pair validation failed: %w", err)
	}
	return nil
}

func (vm *vm) getVXCACert() (*x509.Certificate, error) {
	vxca, err := vm.VXCA()
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(vxca)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the VXCA cert: %w", err)
	}
	return cert, nil
}
