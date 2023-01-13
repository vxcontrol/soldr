package provider

import (
	"crypto/x509"
	"encoding/base64"
	"fmt"
)

func CheckValidity(cert *x509.Certificate, intermidiates [][]byte, root []byte) error {
	intermidiatesPool, err := newCertPool(intermidiates...)
	if err != nil {
		return fmt.Errorf("failed to create an intermidiates cert pool: %w", err)
	}
	rootPool, err := newCertPool(root)
	if err != nil {
		return fmt.Errorf("failed to create a root cert pool: %w", err)
	}
	chains, err := cert.Verify(x509.VerifyOptions{
		Intermediates: intermidiatesPool,
		Roots:         rootPool,
	})
	if err != nil {
		return fmt.Errorf("failed to verify the certificate validity: %w", err)
	}
	if len(chains) != 1 {
		return fmt.Errorf("an unexpected number of validity chains: %d", len(chains))
	}
	if len(intermidiates) == 0 {
		return nil
	}
	expectedChainLen := 2 + len(intermidiates)
	actualChainLen := len(chains[0])
	if actualChainLen != expectedChainLen {
		return fmt.Errorf(
			"a validity chain has been found, but it does not contain all the passed intermidiates: "+
				"expected length %d, got: %d",
			expectedChainLen, actualChainLen)
	}
	return nil
}

func newCertPool(certs ...[]byte) (*x509.CertPool, error) {
	p := x509.NewCertPool()
	for _, c := range certs {
		if !p.AppendCertsFromPEM(c) {
			return nil, fmt.Errorf("failed to append certificate %s to the cert pool", base64.StdEncoding.EncodeToString(c))
		}
	}
	return p, nil
}
