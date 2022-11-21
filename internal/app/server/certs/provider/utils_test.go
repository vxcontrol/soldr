package provider

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"testing"

	internalTesting "soldr/internal/app/server/certs/provider/internal/testing"
)

func TestCheckValidity(t *testing.T) {
	root, rootPriv, x509Root, err := generateSelfSignedCertWithX509Repr(internalTesting.RootTmpl)
	if err != nil {
		t.Fatalf("failed to generate the root certificate: %v", err)
	}
	caFirst, caFirstPriv, x509CAFirst, err := generateCertWithX509Repr(internalTesting.RootTmpl, internalTesting.RootTmpl, rootPriv)
	if err != nil {
		t.Fatalf("failed to generate the first CA cert: %v", err)
	}
	caSecond, caSecondPriv, _, err := generateCertWithX509Repr(internalTesting.RootTmpl, internalTesting.RootTmpl, caFirstPriv)
	if err != nil {
		t.Fatalf("failed to generate the second CA cert: %v", err)
	}
	firstCert, firstCertPriv, x509FirstCert, err := generateCertWithX509Repr(nil, internalTesting.RootTmpl, caFirstPriv)
	if err != nil {
		t.Fatalf("failed to generate the cert signed by the first CA: %v", err)
	}
	_, _, x509FirstAndSecondCert, err := generateCertWithX509Repr(nil, internalTesting.RootTmpl, caSecondPriv)
	if err != nil {
		t.Fatalf("faild to generate the cert signed by the second CA: %v", err)
	}
	cases := []struct {
		Name string

		Certs         func() (cert *x509.Certificate, intermidiates [][]byte, root []byte, err error)
		ExpectedError error
	}{
		{
			Name: "normal",
			Certs: func() (*x509.Certificate, [][]byte, []byte, error) {
				return x509FirstCert, [][]byte{caFirst}, root, nil
			},
		},
		{
			Name: "normal: no intermidiates",
			Certs: func() (*x509.Certificate, [][]byte, []byte, error) {
				return x509CAFirst, nil, root, nil
			},
		},
		{
			Name: "normal: self-signed",
			Certs: func() (*x509.Certificate, [][]byte, []byte, error) {
				return x509Root, nil, root, nil
			},
		},
		{
			Name: "normal: multiple intermidiates",
			Certs: func() (*x509.Certificate, [][]byte, []byte, error) {
				return x509FirstAndSecondCert, [][]byte{caSecond, caFirst}, root, nil
			},
		},
		{
			Name: "normal: multiple intermidiates in different order",
			Certs: func() (*x509.Certificate, [][]byte, []byte, error) {
				return x509FirstAndSecondCert, [][]byte{caFirst, caSecond}, root, nil
			},
		},
		{
			Name: "error: missing intermidiate",
			Certs: func() (*x509.Certificate, [][]byte, []byte, error) {
				return x509FirstCert, nil, root, nil
			},
			ExpectedError: fmt.Errorf("failed to verify the certificate validity: " +
				"x509: certificate signed by unknown authority " +
				"(possibly because of \"x509: Ed25519 verification failure\" while trying to verify candidate authority certificate \"HoHo Co\")",
			),
		},
		{
			Name: "error: missing root",
			Certs: func() (*x509.Certificate, [][]byte, []byte, error) {
				return x509FirstCert, [][]byte{caFirst}, nil, nil
			},
			ExpectedError: fmt.Errorf("failed to create a root cert pool: failed to append certificate  to the cert pool"),
		},
		{
			Name: "error: an extra intermidiate",
			Certs: func() (*x509.Certificate, [][]byte, []byte, error) {
				return x509FirstCert, [][]byte{caFirst, caSecond}, root, nil
			},
			ExpectedError: fmt.Errorf("a validity chain has been found, but it does not contain all the passed intermidiates: expected length 4, got: 3"),
		},
	}

	for i, tc := range cases {
		i, tc := i, tc
		t.Run(fmt.Sprintf("test case #%d, \"%s\"", i, tc.Name), func(t *testing.T) {
			t.Parallel()

			cert, intermidiates, root, err := tc.Certs()
			if err != nil {
				t.Error(err)
				return
			}
			actualErr := CheckValidity(cert, intermidiates, root)
			if err := internalTesting.CompareErrors(tc.ExpectedError, actualErr); err != nil {
				t.Error(err)
				return
			}
		})
	}

	t.Run("test: \"normal: intermidiates in the TLS certificate chain\"", func(t *testing.T) {
		t.Parallel()

		certWithChainPEM := make([]byte, len(firstCert))
		copy(certWithChainPEM, firstCert)
		certWithChainPEM = append(certWithChainPEM, caFirst...)
		certKey, err := internalTesting.PrivateKeyToPEM(firstCertPriv)
		if err != nil {
			t.Errorf("failed to convert the first certificate private key to the PEM format: %v", err)
			return
		}
		tlsCert, err := tls.X509KeyPair(certWithChainPEM, certKey)
		if err != nil {
			t.Errorf("failed to get a tlsCert from the keypair: %v", err)
			return
		}
		if len(tlsCert.Certificate) != 2 {
			t.Errorf("expected to have 2 certificates in the validity chain, got %d", len(tlsCert.Certificate))
			return
		}
		leafCert, err := x509.ParseCertificate(tlsCert.Certificate[0])
		if err != nil {
			t.Errorf("failed to convert the leaf cert to x509 cert: %v", err)
			return
		}
		interm := internalTesting.DERCertToPEM(tlsCert.Certificate[1])
		if err := CheckValidity(leafCert, [][]byte{interm}, root); err != nil {
			t.Errorf("failed to check the certificate validity: %v", err)
			return
		}
	})
}

func Test_newCertPool(t *testing.T) {
	_, rootPriv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("root key generation failed: %v", err)
	}
	cases := []struct {
		Name                  string
		GenCertsFunc          func() ([][]byte, error)
		ExpectedSubjectsCount int
		ExpectedError         error
	}{
		{
			Name: "normal: single cert",
			GenCertsFunc: func() ([][]byte, error) {
				cert, _, err := internalTesting.GenerateCert(nil, internalTesting.RootTmpl, nil, rootPriv)
				if err != nil {
					return nil, err
				}
				return [][]byte{cert}, nil
			},
			ExpectedSubjectsCount: 1,
		},
		{
			Name: "normal: multiple certs",
			GenCertsFunc: func() ([][]byte, error) {
				first, _, err := internalTesting.GenerateCert(nil, internalTesting.RootTmpl, nil, rootPriv)
				if err != nil {
					return nil, err
				}
				second, _, err := internalTesting.GenerateCert(nil, internalTesting.RootTmpl, nil, rootPriv)
				if err != nil {
					return nil, err
				}
				return [][]byte{first, second}, nil
			},
			ExpectedSubjectsCount: 2,
		},
		{
			Name: "normal: nil certs",
			GenCertsFunc: func() ([][]byte, error) {
				return nil, nil
			},
			ExpectedSubjectsCount: 0,
		},
		{
			Name: "error: random bytes passed as a cert",
			GenCertsFunc: func() ([][]byte, error) {
				first, _, err := internalTesting.GenerateCert(nil, internalTesting.RootTmpl, nil, rootPriv)
				if err != nil {
					return nil, err
				}
				second := []byte("this-is-not-a-certificate")
				return [][]byte{first, second}, nil
			},
			ExpectedSubjectsCount: 2,
			ExpectedError:         fmt.Errorf("failed to append certificate dGhpcy1pcy1ub3QtYS1jZXJ0aWZpY2F0ZQ== to the cert pool"),
		},
	}

	for i, tc := range cases {
		i, tc := i, tc
		t.Run(fmt.Sprintf("test case #%d, \"%s\"", i, tc.Name), func(t *testing.T) {
			t.Parallel()

			certs, err := tc.GenCertsFunc()
			if err != nil {
				t.Errorf("failed to generate certificates: %v", err)
				return
			}
			p, err := newCertPool(certs...)
			if err := internalTesting.CompareErrors(tc.ExpectedError, err); err != nil {
				t.Error(err)
				return
			}
			if err != nil {
				return
			}
			subjects := p.Subjects()
			if len(subjects) != tc.ExpectedSubjectsCount {
				t.Errorf("expected subjects length: %d, got: %d", tc.ExpectedSubjectsCount, len(subjects))
				return
			}
		})
	}
}

func generateSelfSignedCertWithX509Repr(tmpl *x509.Certificate) ([]byte, ed25519.PrivateKey, *x509.Certificate, error) {
	cert, certPriv, err := internalTesting.GenerateSelfSignedCert(tmpl, nil)
	if err != nil {
		return nil, nil, nil, err
	}
	x509Cert, err := pemToX509(cert)
	if err != nil {
		return nil, nil, nil, err
	}
	return cert, certPriv, x509Cert, nil
}

func generateCertWithX509Repr(certTemplate *x509.Certificate, parentTemplate *x509.Certificate, parentPriv interface{}) ([]byte, ed25519.PrivateKey, *x509.Certificate, error) {
	cert, certPriv, err := internalTesting.GenerateCert(certTemplate, parentTemplate, nil, parentPriv)
	if err != nil {
		return nil, nil, nil, err
	}
	x509Cert, err := pemToX509(cert)
	if err != nil {
		return nil, nil, nil, err
	}
	return cert, certPriv, x509Cert, nil
}

func pemToX509(cert []byte) (*x509.Certificate, error) {
	certPEMBlock, _ := pem.Decode(cert)
	if certPEMBlock == nil {
		return nil, fmt.Errorf("failed to get the certificate block from the cert PEM object")
	}
	x509Cert, err := x509.ParseCertificate(certPEMBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the PEM cert into an x509 cert: %w", err)
	}
	return x509Cert, nil
}
