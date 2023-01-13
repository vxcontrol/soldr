package provider

import (
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"fmt"
)

// These values are injected during linking
var (
	vxca   = ""
	iac    = ""
	iacKey = ""
)

type Static struct {
	decoder Decoder
}

func NewStatic() (*Static, error) {
	decoder, err := NewDecoder()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the certificates decoder: %w", err)
	}
	return &Static{
		decoder: decoder,
	}, nil
}

func (s *Static) VXCA() ([]byte, error) {
	vxcaBytes, err := decodeCert(vxca, s.decoder.DecodeVXCA)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the VXCA cert: %w", err)
	}
	block, rest := pem.Decode(vxcaBytes)
	if len(rest) != 0 {
		return nil, fmt.Errorf("failed to decode the VXCA certificate: the file contains the certificate and something else")
	}
	return block.Bytes, nil
}

func (s *Static) IAC() (*tls.Certificate, error) {
	iacBytes, err := decodeCert(iac, s.decoder.DecodeIAC)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the IAC cert: %w", err)
	}
	iacKeyBytes, err := decodeCert(iacKey, s.decoder.DecodeIACKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the IAC key: %w", err)
	}
	cert, err := tls.X509KeyPair(iacBytes, iacKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to get the IAC certificate: %w", err)
	}
	return &cert, nil
}

func decodeCert(cert string, decoder func([]byte) ([]byte, error)) ([]byte, error) {
	decodedHex, err := hex.DecodeString(cert)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the hex string: %w", err)
	}
	certBytes, err := decoder(decodedHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode the passed data: %w", err)
	}
	decodedCert, err := base64.StdEncoding.DecodeString(string(certBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to decode the base64-encoded string: %w", err)
	}
	return decodedCert, nil
}
