package vm

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/base64"
	"fmt"

	"github.com/golang-jwt/jwt/v4"
)

type SBHContents struct {
	Nonce     string
	Timestamp float64
}

type SBHParser struct{}

func NewSBHParser() *SBHParser {
	return &SBHParser{}
}

func (p *SBHParser) ParseSBHToken(cert *x509.Certificate, sbh string) (*SBHContents, error) {
	token, err := jwt.Parse(sbh, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodEd25519); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		pk, ok := cert.PublicKey.(ed25519.PublicKey)
		if !ok {
			return nil, fmt.Errorf("passed public key is not an ed25519.PublicKey")
		}
		return pk, nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse the received SBH JWT token: %w", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("failed to cast token claims to jwt.MapClaims")
	}
	expiresAt, err := extractExpiresAtClaim(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to extract the expires_at claim: %w", err)
	}
	nonce, err := extractNonceClaim(claims)
	if err != nil {
		return nil, fmt.Errorf("failed to extract the nonce claim: %w", err)
	}
	return &SBHContents{
		Nonce:     string(nonce),
		Timestamp: expiresAt,
	}, nil
}

func extractExpiresAtClaim(claims jwt.MapClaims) (float64, error) {
	expiresAtRaw, ok := claims["exp"]
	if !ok {
		return 0, fmt.Errorf("the token does not contain the expiration time claim")
	}
	expiresAt, ok := expiresAtRaw.(float64)
	if !ok {
		return 0, fmt.Errorf("the expiration time claim is of an invalid type")
	}
	return expiresAt, nil
}

func extractNonceClaim(claims jwt.MapClaims) ([]byte, error) {
	nonceRaw, ok := claims["nonce"]
	if !ok {
		return nil, fmt.Errorf("the token does not contain the nonce claim")
	}
	nonce, ok := nonceRaw.(string)
	if !ok {
		return nil, fmt.Errorf("the nonce claim is of an invalid type")
	}
	return base64.StdEncoding.DecodeString(nonce)
}
