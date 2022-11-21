package static

import (
	"crypto/ed25519"
	"fmt"
	"testing"

	internalTesting "soldr/internal/app/server/certs/provider/internal/testing"
)

func Test_getTLSCertFromPEMKeyPair(t *testing.T) {
	_, rootPriv, err := internalTesting.GenerateSelfSignedCert(internalTesting.RootTmpl, nil)
	if err != nil {
		t.Fatalf("failed to generate the root certificate: %v", err)
	}
	firstCert, firstCertPriv, err := internalTesting.GenerateCert(nil, internalTesting.RootTmpl, nil, rootPriv)
	if err != nil {
		t.Fatalf("failed to generate the first certificate: %v", err)
	}
	_, secondCertPriv, err := internalTesting.GenerateCert(nil, internalTesting.RootTmpl, nil, rootPriv)
	if err != nil {
		t.Fatalf("failed to generate the second certificate: %v", err)
	}

	cases := []struct {
		Name string

		Cert          []byte
		Key           ed25519.PrivateKey
		ExpectedError error
	}{
		{
			Name: "normal",
			Cert: firstCert,
			Key:  firstCertPriv,
		},
		{
			Name:          "error: cert and key do not match",
			Cert:          firstCert,
			Key:           secondCertPriv,
			ExpectedError: fmt.Errorf("failed to compose a TLS certificate from the passed keypair: tls: private key does not match public key"),
		},
	}

	for i, tc := range cases {
		i, tc := i, tc
		t.Run(fmt.Sprintf("test case #%d: %s", i, tc.Name), func(t *testing.T) {
			pemKey, err := internalTesting.PrivateKeyToPEM(tc.Key)
			if err != nil {
				t.Errorf("failed to parse the passed private key: %v", err)
				return
			}
			_, actualErr := getTLSCertFromPEMKeyPair(tc.Cert, nil, pemKey)
			if err := internalTesting.CompareErrors(tc.ExpectedError, actualErr); err != nil {
				t.Error(err)
				return
			}
		})
	}
}
