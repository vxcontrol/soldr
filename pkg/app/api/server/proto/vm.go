package proto

import (
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"soldr/pkg/hardening/luavm/certs"
	vmstore "soldr/pkg/hardening/luavm/store"
	"soldr/pkg/hardening/luavm/store/types"
	"soldr/pkg/version"
	"soldr/pkg/vxproto"
)

type validator struct{}

func (v *validator) HasAgentInfoValid(ctx context.Context, iasocket vxproto.IAgentSocket) error {
	return nil
}

func (v *validator) HasTokenValid(token string, agentID string, agentType vxproto.AgentType) bool {
	return true
}

func (v *validator) HasTokenCRCValid(token string) bool {
	return true
}

func (v *validator) NewToken(agentID string, agentType vxproto.AgentType) (string, error) {
	return "", nil
}

type informator struct{}

func (i *informator) GetVersion() string {
	return version.GetBinaryVersion()
}

type certProvider struct {
	dir string
}

func NewCertProvider(dir string) certs.CertProvider {
	return &certProvider{dir: dir}
}

func (cp *certProvider) VXCA() ([]byte, error) {
	vxcaPath := filepath.Join(cp.dir, "vxca.cert")
	return readPEM(vxcaPath)
}

func (cp *certProvider) IAC() (*tls.Certificate, error) {
	return nil, fmt.Errorf("getting IAC has not implemented yet")
}

type store struct {
	dir string
}

func NewStore(dir string) vmstore.Store {
	return &store{dir: dir}
}

func (s *store) GetLTAC(key []byte) (*types.LTAC, error) {
	var (
		ltac        types.LTAC
		ltacCAPath  = filepath.Join(s.dir, "ca.cert")
		ltacCrtPath = filepath.Join(s.dir, "ltac.cert")
		ltacKeyPath = filepath.Join(s.dir, "ltac.key")
	)
	if blob, err := readPEM(ltacCAPath); err != nil {
		return nil, fmt.Errorf("failed to read LTAC CA file: %w", err)
	} else {
		ltac.CA = blob
	}
	if blob, err := readPEM(ltacCrtPath); err != nil {
		return nil, fmt.Errorf("failed to read LTAC Crt file: %w", err)
	} else {
		ltac.Cert = blob
	}
	if blob, err := readPEM(ltacKeyPath); err != nil {
		return nil, fmt.Errorf("failed to read LTAC Key file: %w", err)
	} else {
		ltac.Key = blob
	}
	return &ltac, nil
}

func (s *store) GetSBH(key []byte) ([]byte, error) {
	sbhPath := filepath.Join(s.dir, "sbh.json")
	encodedBlob, err := ioutil.ReadFile(sbhPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read SBH from FS '%s': %w", sbhPath, err)
	}
	blob, err := base64.StdEncoding.DecodeString(string(encodedBlob))
	if err != nil {
		return nil, fmt.Errorf("failed to decode the SBH blob: %w", err)
	}
	return blob, nil
}

func (s *store) StoreInitConnectionPack(p *types.InitConnectionPack, key []byte) error {
	return fmt.Errorf("storing data has not implemented yet")
}

func (s *store) Reset() error {
	return fmt.Errorf("resetting store has not implemented yet")
}

func readPEM(path string) ([]byte, error) {
	certBytes, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read file from FS '%s': %w", path, err)
	}
	block, rest := pem.Decode(certBytes)
	if len(rest) != 0 {
		return nil, fmt.Errorf("failed to decode: the file wrong format")
	}
	return block.Bytes, nil
}
