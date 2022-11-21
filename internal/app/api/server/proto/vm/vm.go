package vm

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"fmt"
	"io"
	"time"

	"soldr/internal/hardening/luavm/certs"
	"soldr/internal/hardening/luavm/vm"
	"soldr/internal/protoagent"
	"soldr/internal/vxproto/tunnel"
)

type VM struct {
	tunnel.PackEncryptor
	vm.TLSConfigurer
	certs.CertProvider
	vm.PingResponder

	abhGetter vm.ABHCalculator
	sbhParser *vm.SBHParser
}

func NewVM(
	packEncrypter tunnel.PackEncryptor,
	certsProvider certs.CertProvider,
	ltacGetter vm.LTACGetter,
) (*VM, error) {
	vm := &VM{
		CertProvider:  certsProvider,
		TLSConfigurer: newTLSConfigurer(certsProvider, ltacGetter),
		PackEncryptor: packEncrypter,
		PingResponder: vm.NewSimplePingResponder(),
	}
	var err error
	vm.abhGetter, err = newABHCalculator()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize an ABH calculator: %w", err)
	}
	return vm, nil
}

func (v *VM) ProcessConnectionChallengeRequest(ctx context.Context, req []byte) ([]byte, error) {
	var connChallengeReq protoagent.ConnectionChallengeRequest
	if err := protoagent.UnpackProtoMessage(&connChallengeReq, req, protoagent.Message_CONNECTION_CHALLENGE_REQUEST); err != nil {
		return nil, fmt.Errorf("failed to unpack the connection challenge request: %w", err)
	}
	agentID, err := getAgentID(ctx)
	if err != nil {
		return nil, err
	}
	ct, err := v.prepareChallengeResponseCT(&connChallengeReq, agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare the connection challenge response: %w", err)
	}
	connChallengeResp := &protoagent.ConnectionChallengeResponse{
		Ct: ct,
	}
	msg, err := protoagent.PackProtoMessage(connChallengeResp, protoagent.Message_CONNECTION_CHALLENGE_REQUEST)
	if err != nil {
		return nil, fmt.Errorf("failed to pack the connection challenge response: %w", err)
	}
	return msg, nil
}

type CTXKey string

const CTXAgentIDKey CTXKey = "agent-id"

func getAgentID(ctx context.Context) (string, error) {
	agentIDVal := ctx.Value(CTXAgentIDKey)
	if agentIDVal == nil {
		return "", fmt.Errorf("failed to find the agent ID (key %s) in the passed context", CTXAgentIDKey)
	}
	agentID, ok := agentIDVal.(string)
	if !ok {
		return "", fmt.Errorf("the agent ID found in the context (%v) is of the wrong type (expected string)", agentIDVal)
	}
	return agentID, nil
}

func (v *VM) prepareChallengeResponseCT(req *protoagent.ConnectionChallengeRequest, agentID string) ([]byte, error) {
	abh, err := v.abhGetter.GetABH()
	if err != nil {
		return nil, fmt.Errorf("failed to get the ABH: %w", err)
	}
	key := GetChallengeKey(agentID, abh)
	ct, err := aesEncrypt(key, req.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to encrypt the challenge nonce: %w", err)
	}
	return ct, nil
}

type AESKey []byte

func GetChallengeKey(agentID string, abh []byte) AESKey {
	agentIDData := []byte(agentID)
	key := make([]byte, 0, len(agentIDData)+len(abh))
	key = append(append(key, agentIDData...), abh...)
	return getAESKey(key)
}

func getAESKey(k []byte) AESKey {
	keyHash := sha256.Sum256(k)
	return keyHash[:]
}

func aesGetCipherBlock(key AESKey) (cipher.Block, error) {
	keyLen := len(key)
	if keyLen != 16 && keyLen != 24 && keyLen != 32 {
		return nil, fmt.Errorf("bad key length (%d)", len(key))
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new cipher block: %w", err)
	}
	return block, nil
}

func aesEncrypt(key AESKey, pt []byte) ([]byte, error) {
	block, err := aesGetCipherBlock(key)
	if err != nil {
		return nil, err
	}
	ct := make([]byte, aes.BlockSize+len(pt))
	iv := ct[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, fmt.Errorf("failed to initialize an IV: %w", err)
	}
	cipher.NewCFBEncrypter(block, iv).
		XORKeyStream(ct[aes.BlockSize:], pt)
	return ct, nil
}

func (vm *VM) ProcessConnectionRequest(req []byte, packEncrypter tunnel.PackEncryptor) (resp []byte, err error) {
	payload, msgType, err := protoagent.GetProtoMessagePayload(req)
	if err != nil {
		return nil, err
	}
	if msgType == protoagent.Message_AUTHENTICATION_RESPONSE {
		var authResp protoagent.AuthenticationResponse
		if err := protoagent.UnpackProtoMessagePayload(&authResp, payload); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("connection attempt has failed (status: %s)", authResp.GetStatus())
	}
	var connReq protoagent.ConnectionStartRequest
	if err = protoagent.UnpackProtoMessage(&connReq, req, protoagent.Message_CONNECTION_REQUEST); err != nil {
		return nil, fmt.Errorf("failed to unpack the connection request proto message: %w", err)
	}
	if err = vm.checkSBH(connReq.Sbh); err != nil {
		return nil, fmt.Errorf("SBH check has failed: %w", err)
	}
	if err = packEncrypter.Reset(connReq.TunnelConfig); err != nil {
		return nil, fmt.Errorf("failed to configure the pack encrypter: %w", err)
	}
	resp, err = protoagent.PackProtoMessage(&protoagent.ConnectionStartResponse{}, protoagent.Message_CONNECTION_REQUEST)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare the start connection response: %w", err)
	}
	return resp, nil
}

func (vm *VM) checkSBH(sbh []byte) error {
	vxca, err := vm.CertProvider.VXCA()
	if err != nil {
		return fmt.Errorf("failed to get VXCA: %w", err)
	}
	cert, err := x509.ParseCertificate(vxca)
	if err != nil {
		return fmt.Errorf("failed to parse the VXCA cert: %w", err)
	}
	sbhTok, err := vm.sbhParser.ParseSBHToken(cert, string(sbh))
	if err != nil {
		return fmt.Errorf("failed to parse the SBH token: %w", err)
	}
	if time.Now().Unix() > int64(sbhTok.Timestamp) {
		return fmt.Errorf("received SBH token is not longer valid")
	}
	return nil
}

func (vm *VM) PrepareInitConnectionRequest(_ *vm.InitConnectionAgentInfo, _ *protoagent.Information) ([]byte, error) {
	return nil, fmt.Errorf("PrepareInitConnection is not implemented for an external connection")
}

func (vm *VM) ProcessInitConnectionResponse(resp []byte) error {
	return fmt.Errorf("ProcessInitConnectionResponse is not implemented for an external connection")
}

func (vm *VM) ResetInitConnection() {}

func (vm *VM) ResetConnection() error {
	return nil
}

func (v *VM) EncryptData(data []byte) ([]byte, error) {
	return nil, fmt.Errorf("EncryptData is not implemented for an external connection")
}

func (v *VM) DecryptData(data []byte) ([]byte, error) {
	return nil, fmt.Errorf("DecryptData is not implemented for an external connection")
}

func (v *VM) IsStoreKeyEmpty() (bool, error) {
	return false, fmt.Errorf("IsStoreKeyEmpty is not implemented for an external connection")
}
