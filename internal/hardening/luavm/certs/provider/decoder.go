package provider

import (
	"fmt"
)

// These values are injected during linking
var (
	vxcaDecodeKey   string = ""
	iacDecodeKey    string = ""
	iacKeyDecodeKey string = ""
)

type Decoder interface {
	DecodeVXCA(vxca []byte) ([]byte, error)
	DecodeIAC(iac []byte) ([]byte, error)
	DecodeIACKey(iacKey []byte) ([]byte, error)
}

func NewDecoder() (Decoder, error) {
	return newXORDecoder(&xorDecoderConfig{
		vxcaDecodeKey:   []byte(vxcaDecodeKey),
		iacDecodeKey:    []byte(iacDecodeKey),
		iacKeyDecodeKey: []byte(iacKeyDecodeKey),
	})
}

type xorDecoder struct {
	config *xorDecoderConfig
}

type xorDecoderConfig struct {
	vxcaDecodeKey   []byte
	iacDecodeKey    []byte
	iacKeyDecodeKey []byte
}

func newXORDecoder(c *xorDecoderConfig) (*xorDecoder, error) {
	// TODO(SSH): rewrite with validator
	if c == nil {
		return nil, fmt.Errorf("passed configuration object is nil")
	}
	if len(c.vxcaDecodeKey) == 0 ||
		len(c.iacDecodeKey) == 0 ||
		len(c.iacKeyDecodeKey) == 0 {
		return nil, fmt.Errorf("one of the decode keys is empty")
	}
	return &xorDecoder{
		config: c,
	}, nil
}

func (d *xorDecoder) DecodeVXCA(vxca []byte) ([]byte, error) {
	return d.decode(vxca, d.config.vxcaDecodeKey)
}

func (d *xorDecoder) DecodeIAC(iac []byte) ([]byte, error) {
	return d.decode(iac, d.config.iacDecodeKey)
}

func (d *xorDecoder) DecodeIACKey(iacKey []byte) ([]byte, error) {
	return d.decode(iacKey, d.config.iacKeyDecodeKey)
}

func (d *xorDecoder) decode(data []byte, key []byte) ([]byte, error) {
	if len(key) == 0 {
		return nil, fmt.Errorf("passed decoding key is empty")
	}
	keyPos := 0
	res := make([]byte, len(data))
	for i, d := range data {
		if keyPos == len(key) {
			keyPos = 0
		}
		k := key[keyPos]
		res[i] = d ^ k
		keyPos++
	}
	return res, nil
}
