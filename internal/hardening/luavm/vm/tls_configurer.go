package vm

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"

	vxcommonErrors "soldr/internal/errors"
	"soldr/internal/hardening/luavm/certs"
	storeTypes "soldr/internal/hardening/luavm/store/types"
)

type LTACGetter interface {
	GetLTAC(key []byte) (*storeTypes.LTAC, error)
}

type scaStore interface {
	scaPopper
	PushSCA(sca []byte) error
}

type SimpleTLSConfigurer struct {
	certsProvider certs.CertProvider
	ltacGetter    LTACGetter
	scaStore      scaStore
}

func NewSimpleTLSConfigurer(
	certsProvider certs.CertProvider,
	ltacGetter LTACGetter,
	scaStore scaStore,
) *SimpleTLSConfigurer {
	return &SimpleTLSConfigurer{
		certsProvider: certsProvider,
		ltacGetter:    ltacGetter,
		scaStore:      scaStore,
	}
}

func (c *SimpleTLSConfigurer) GetTLSConfigForInitConnection() (*tls.Config, error) {
	iac, err := c.certsProvider.IAC()
	if err != nil {
		return nil, fmt.Errorf("failed to get IAC: %w", err)
	}
	vxcaPool, err := getVXCACertsPool(c.certsProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to get the VXCA pool: %w", err)
	}
	//TODO: TLS version is too low

	// #nosec G402
	tlsConfig := &tls.Config{
		Certificates:          []tls.Certificate{*iac},
		RootCAs:               vxcaPool,
		VerifyPeerCertificate: c.initConnectionVerifyPeerCertificate,
		ServerName:            certServerName,
	}
	return tlsConfig, nil
}

func (c *SimpleTLSConfigurer) GetTLSConfigForConnection() (*tls.Config, error) {
	vxcaPool, err := getVXCACertsPool(c.certsProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create the VXCA certificates pool: %w", err)
	}
	ltacCert, err := c.getLTACCertificate()
	if err != nil {
		return nil, fmt.Errorf("failed to get the LTAC certificate: %w", err)
	}
	//TODO: TLS version is too low

	// #nosec G402
	return &tls.Config{
		Certificates: []tls.Certificate{*ltacCert},
		RootCAs:      vxcaPool,
		ServerName:   certServerName,
	}, nil
}

func (c *SimpleTLSConfigurer) getLTACCertificate() (*tls.Certificate, error) {
	ltac, err := c.ltacGetter.GetLTAC(nil)
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

func (c *SimpleTLSConfigurer) initConnectionVerifyPeerCertificate(
	rawCerts [][]byte,
	verifiedChains [][]*x509.Certificate,
) error {
	_, err := c.scaStore.PopSCA()
	if err != nil {
		return fmt.Errorf("popSCA is failed: %w", err)
	}
	if len(rawCerts) != 2 {
		return fmt.Errorf("expected to get two raw certs, actually got %d", len(rawCerts))
	}
	if len(verifiedChains) != 1 {
		return fmt.Errorf("expected to get one verified chain, actually got %d", len(verifiedChains))
	}
	if len(verifiedChains[0]) != 3 {
		return fmt.Errorf("expected to get three certificates in the verified chain, actually got %d", len(verifiedChains[0]))
	}
	if err = c.scaStore.PushSCA(rawCerts[1]); err != nil {
		return fmt.Errorf("failed to save the passed SCA certificate: %w", err)
	}
	return nil
}
