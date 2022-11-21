package vm

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"runtime"
	"sync"

	vxcommonErrors "soldr/internal/errors"
	"soldr/internal/hardening/luavm/certs"
	"soldr/internal/hardening/luavm/store"
	storeTypes "soldr/internal/hardening/luavm/store/types"
	"soldr/internal/protoagent"
	utilsErrors "soldr/internal/utils/errors"
	vxprotoTunnel "soldr/internal/vxproto/tunnel"
)

const certServerName = "example"

type InitConnectionAgentInfo struct {
	ID      string
	Version string
}

type TLSConfigurer interface {
	GetTLSConfigForInitConnection() (*tls.Config, error)
	GetTLSConfigForConnection() (*tls.Config, error)
}

type PingResponder interface {
	GeneratePingResponse(nonce []byte) ([]byte, error)
}

type scaPopper interface {
	PopSCA() ([]byte, error)
}

type ChallengeCalculator interface {
	PrepareChallengeResponse(nonce []byte) ([]byte, error)
}

type ISecureConfigEncryptor interface {
	DecryptData(data []byte) ([]byte, error)
	IsStoreKeyEmpty() (bool, error)
}

type VM interface {
	certs.CertProvider
	TLSConfigurer
	ISecureConfigEncryptor

	PrepareInitConnectionRequest(info *InitConnectionAgentInfo, agentInfo *protoagent.Information) ([]byte, error)
	ProcessInitConnectionResponse(resp []byte) error
	ResetInitConnection()

	ProcessConnectionChallengeRequest(ctx context.Context, req []byte) ([]byte, error)
	ProcessConnectionRequest(req []byte, encryptor vxprotoTunnel.PackEncryptor) ([]byte, error)

	GeneratePingResponse(nonce []byte) ([]byte, error)

	ResetConnection() error
}

type VMConfig struct {
	StoreConfig   *storeTypes.Config
	Store         store.Store
	CertProvider  certs.CertProvider
	ABHCalculator ABHCalculator
}

func NewVM(c *VMConfig) (VM, error) {
	vm, err := newVM(c)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the VM: %w", err)
	}
	return vm, nil
}

type vm struct {
	store.Store
	certs.CertProvider
	ABHCalculator
	ISecureConfigEncryptor

	agentID             string
	ltacTempl           *ltacTemplate
	ltacTemplateMux     *sync.Mutex
	sca                 []byte
	scaMux              *sync.Mutex
	challengeCalculator ChallengeCalculator
}

type ltacTemplate struct {
	Key []byte
}

func newVM(c *VMConfig) (*vm, error) {
	vm := &vm{
		agentID:         c.StoreConfig.AgentID,
		scaMux:          &sync.Mutex{},
		ltacTemplateMux: &sync.Mutex{},
	}
	var err error
	if c.Store == nil {
		vm.Store, err = store.NewStore(c.StoreConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize the store: %w", err)
		}
	} else {
		vm.Store = c.Store
	}
	if c.CertProvider == nil {
		vm.CertProvider, err = certs.NewCertProvider()
		if err != nil {
			return nil, fmt.Errorf("failed to initialize the cert provider: %w", err)
		}
	} else {
		vm.CertProvider = c.CertProvider
	}
	vm.ABHCalculator = c.ABHCalculator
	if vm.ABHCalculator == nil {
		vm.ABHCalculator = newABHCalculator()
	}
	vm.ISecureConfigEncryptor, err = NewSecureConfigEncryptor(vm.agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a secure config provider: %w", err)
	}
	vm.challengeCalculator, err = NewChallengeCalculator(vm.ABHCalculator, vm.agentID)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a challenge calculator: %w", err)
	}
	return vm, nil
}

// TODO(SSH): replace this logic by a more stable and flexible mechanism
func (v *vm) pushSCA(sca []byte) error {
	v.scaMux.Lock()
	defer v.scaMux.Unlock()

	if v.sca != nil {
		return fmt.Errorf("SCA is not nil, it seems that there are multiple initialization connections running")
	}
	v.sca = sca
	return nil
}

func (v *vm) popSCA() ([]byte, error) {
	v.scaMux.Lock()
	defer v.scaMux.Unlock()

	if v.sca == nil {
		return nil, fmt.Errorf("SCA is nil, it seems that there are multiple initialization connections running")
	}
	sca := v.sca
	v.sca = nil
	return sca, nil
}

func (v *vm) PrepareInitConnectionRequest(info *InitConnectionAgentInfo, agentInfo *protoagent.Information) (msg []byte, err error) {
	defer func() {
		if err != nil {
			v.popSCA()
		}
	}()
	csr, ltacKey, err := v.generateLTACRequest()
	if err != nil {
		return nil, fmt.Errorf("failed to generate an LTAC CSR: %w", err)
	}
	if err = v.pushLTACTemplKey(ltacKey); err != nil {
		return nil, fmt.Errorf("failed to store the LTAC key: %w", err)
	}
	abh, err := v.GetABH()
	if err != nil {
		return nil, fmt.Errorf("failed to get ABH: %w", err)
	}
	msg, err = prepareInitConnReqProtoMsg(csr, abh, info, agentInfo)
	if err != nil {
		return nil, err
	}
	return msg, nil
}

func (v *vm) ProcessInitConnectionResponse(respData []byte) (err error) {
	defer func() {
		if err != nil {
			v.popSCA()
		}
	}()
	var initConnResp protoagent.InitConnectionResponse
	if err := protoagent.UnpackProtoMessage(&initConnResp, respData, protoagent.Message_INIT_CONNECTION); err != nil {
		return fmt.Errorf("failed to unpack the init connection response: %w", err)
	}
	key, err := v.popLTACTemplKey()
	if err != nil {
		return fmt.Errorf("failed to get the LTAC template key: %w", err)
	}
	sca, err := v.popSCA()
	if err != nil {
		return fmt.Errorf("failed to get the SCA: %w", err)
	}
	if err = v.checkReceivedLTAC(initConnResp.Ltac, key, sca); err != nil {
		return fmt.Errorf("the received LTAC check failed: %w", err)
	}
	keyDER, err := x509.MarshalPKCS8PrivateKey(key)
	if err != nil {
		return fmt.Errorf("failed to marshal the private key: %w", err)
	}
	sbh, err := v.checkSBHOnInit(initConnResp.Sbh)
	if err != nil {
		return fmt.Errorf("failed to check the received SBH: %w", err)
	}
	if err = v.StoreInitConnectionPack(&storeTypes.InitConnectionPack{
		LTAC: &storeTypes.LTAC{
			LTACPublic: storeTypes.LTACPublic{
				Cert: initConnResp.Ltac,
				CA:   sca,
			},
			Key: keyDER,
		},
		SSALua: initConnResp.Ssa,
		SBH:    sbh,
	}, []byte(v.agentID)); err != nil {
		return fmt.Errorf("failed to store the init connection pack: %w", err)
	}
	return nil
}

func (v *vm) ResetInitConnection() {
	v.scaMux.Lock()
	defer v.scaMux.Unlock()
	v.ltacTemplateMux.Lock()
	defer v.ltacTemplateMux.Unlock()
	v.sca = nil
	v.ltacTempl = nil
}

func (vm *vm) ProcessConnectionRequest(req []byte, packEncryptor vxprotoTunnel.PackEncryptor) (resp []byte, err error) {
	defer func() {
		if err == nil {
			return
		}
		if errors.Is(err, storeTypes.ErrNotInitialized) {
			err = fmt.Errorf("%w: %v", vxcommonErrors.ErrConnectionInitializationRequired, err)
		}
	}()
	payload, msgType, err := protoagent.GetProtoMessagePayload(req)
	if err != nil {
		return nil, err
	}
	if msgType == protoagent.Message_AUTHENTICATION_RESPONSE {
		var authResp protoagent.AuthenticationResponse
		if err := protoagent.UnpackProtoMessagePayload(&authResp, payload); err != nil {
			return nil, err
		}
		if err := vm.processFailedConnectionChallengeResponse(&authResp); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("connection attempt has failed")
	}
	var connReq protoagent.ConnectionStartRequest
	if err = protoagent.UnpackProtoMessage(&connReq, req, protoagent.Message_CONNECTION_REQUEST); err != nil {
		return nil, fmt.Errorf("failed to unpack the connection request proto message: %w", err)
	}
	if err = vm.checkSBH(connReq.Sbh); err != nil {
		if resetErr := vm.Reset(); resetErr != nil {
			return nil, fmt.Errorf("failed to reset the VM store (%v), while processing the SBH verification error: %w", resetErr, err)
		}
		return nil, err
	}
	if err = packEncryptor.Reset(connReq.TunnelConfig); err != nil {
		return nil, fmt.Errorf("failed to configure the pack encryptor: %w", err)
	}
	resp, err = protoagent.PackProtoMessage(&protoagent.ConnectionStartResponse{}, protoagent.Message_CONNECTION_REQUEST)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare the start connection response: %w", err)
	}
	return resp, nil
}

func (v *vm) processFailedConnectionChallengeResponse(authResp *protoagent.AuthenticationResponse) error {
	if authResp.GetStatus() == utilsErrors.ErrFailedResponseTunnelError.Error() {
		if err := v.Store.Reset(); err != nil {
			return err
		}
	}
	return nil
}

func (v *vm) generateLTACRequest() ([]byte, ed25519.PrivateKey, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate the LTAC keypair: %w", err)
	}
	csr, err := x509.CreateCertificateRequest(rand.Reader, &x509.CertificateRequest{
		SignatureAlgorithm: x509.PureEd25519,
	}, priv)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create an LTAC CSR: %w", err)
	}
	return csr, priv, err
}

func (v *vm) pushLTACTemplKey(key []byte) error {
	v.ltacTemplateMux.Lock()
	defer v.ltacTemplateMux.Unlock()

	if v.ltacTempl != nil {
		return fmt.Errorf("ltac template is not nil, it seems that there are multiple initialization connections running")
	}
	v.ltacTempl = &ltacTemplate{
		Key: key,
	}
	return nil
}

func (v *vm) popLTACTemplKey() (ed25519.PrivateKey, error) {
	v.ltacTemplateMux.Lock()
	defer v.ltacTemplateMux.Unlock()

	if v.ltacTempl == nil {
		return nil, fmt.Errorf("ltac template is nil, it seems that there are multiple initialization connections running")
	}
	key := v.ltacTempl.Key
	v.ltacTempl = nil
	return ed25519.PrivateKey(key), nil
}

func checkKeyPairValidity(pub interface{}, priv []byte) error {
	pubKey, ok := pub.(ed25519.PublicKey)
	if !ok {
		return fmt.Errorf("passed public key is not an ed25519 public key")
	}
	privKey := ed25519.PrivateKey(priv)
	if !ok {
		return fmt.Errorf("passed private key is not an ed25519 private key")
	}
	if !pubKey.Equal(privKey.Public()) {
		return fmt.Errorf("passed public and private key do not correspond")
	}
	return nil
}

func prepareInitConnReqProtoMsg(csr []byte, abh []byte, info *InitConnectionAgentInfo, agentInfo *protoagent.Information) ([]byte, error) {
	os := runtime.GOOS
	arch := runtime.GOARCH
	req := &protoagent.InitConnectionRequest{
		Csr:     csr,
		Abh:     abh,
		AgentID: []byte(info.ID),
		AgentBinaryID: &protoagent.AgentBinaryID{
			Os:      &os,
			Arch:    &arch,
			Version: &info.Version,
		},
		Info: agentInfo,
	}
	payload, err := protoagent.PackProtoMessage(req, protoagent.Message_INIT_CONNECTION)
	if err != nil {
		return nil, fmt.Errorf("failed to pack the init connection message: %w", err)
	}
	return payload, nil
}

func (v *vm) GetTLSConfigForInitConnection() (*tls.Config, error) {
	iac, err := v.IAC()
	if err != nil {
		return nil, fmt.Errorf("failed to get IAC: %w", err)
	}
	vxcaPool, err := getVXCACertsPool(v.CertProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to get the VXCA pool: %w", err)
	}
	tlsConfig := &tls.Config{
		Certificates:          []tls.Certificate{*iac},
		RootCAs:               vxcaPool,
		VerifyPeerCertificate: v.initConnectionVerifyPeerCertificate,
		ServerName:            certServerName,
	}
	return tlsConfig, nil
}

func (v *vm) initConnectionVerifyPeerCertificate(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
	// TODO(SSH): change
	v.popSCA()
	if len(rawCerts) != 2 {
		return fmt.Errorf("expected to get two raw certs, actually got %d", len(rawCerts))
	}
	if len(verifiedChains) != 1 {
		return fmt.Errorf("expected to get one verified chain, actually got %d", len(verifiedChains))
	}
	if len(verifiedChains[0]) != 3 {
		return fmt.Errorf("expected to get three certificates in the verified chain, actually got %d", len(verifiedChains[0]))
	}
	if err := v.pushSCA(rawCerts[1]); err != nil {
		return fmt.Errorf("failed to save the passed SCA certificate: %w", err)
	}
	return nil
}

func (v *vm) GetTLSConfigForConnection() (*tls.Config, error) {
	vxcaPool, err := getVXCACertsPool(v.CertProvider)
	if err != nil {
		return nil, fmt.Errorf("failed to create the VXCA certificates pool: %w", err)
	}
	ltacCert, err := v.getLTACCertificate([]byte(v.agentID))
	if err != nil {
		return nil, fmt.Errorf("failed to get the LTAC certificate: %w", err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{*ltacCert},
		RootCAs:      vxcaPool,
		ServerName:   certServerName,
	}, nil
}

func (v *vm) ProcessConnectionChallengeRequest(ctx context.Context, req []byte) ([]byte, error) {
	var connChallengeReq protoagent.ConnectionChallengeRequest
	if err := protoagent.UnpackProtoMessage(&connChallengeReq, req, protoagent.Message_CONNECTION_CHALLENGE_REQUEST); err != nil {
		return nil, fmt.Errorf("failed to unpack the connection challenge request: %w", err)
	}
	ct, err := v.prepareChallengeResponseCT(&connChallengeReq)
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

func (v *vm) prepareChallengeResponseCT(req *protoagent.ConnectionChallengeRequest) ([]byte, error) {
	response, err := v.challengeCalculator.PrepareChallengeResponse(req.GetNonce())
	if err != nil {
		return nil, fmt.Errorf("failed to prepare a challenge response: %w", err)
	}
	return response, nil
}

func (vm *vm) GeneratePingResponse(nonce []byte) ([]byte, error) {
	return nonce, nil
}

func (vm *vm) ResetConnection() error {
	if err := vm.Store.Reset(); err != nil {
		return fmt.Errorf("failed to reset connection: %w", err)
	}
	return nil
}

func (vm *vm) checkSBHOnInit(sbh []byte) ([]byte, error) {
	cert, err := vm.getVXCACert()
	if err != nil {
		return nil, fmt.Errorf("failed to get the VXCA certificate: %w", err)
	}
	sbhParser := NewSBHParser()
	sbhContent, err := sbhParser.ParseSBHToken(cert, string(sbh))
	if err != nil {
		return nil, fmt.Errorf("SBH parsing or verification failed: %w", err)
	}
	msg, err := json.Marshal(sbhContent)
	if err != nil {
		return nil, fmt.Errorf("SBH build message failed: %w", err)
	}
	return msg, nil
}

func (vm *vm) checkSBH(sbh []byte) error {
	receivedSBH, err := vm.checkSBHOnInit(sbh)
	if err != nil {
		return err
	}
	storedSBH, err := vm.GetSBH([]byte(vm.agentID))
	if err != nil {
		return fmt.Errorf("failed to get SBH: %w", err)
	}
	if !bytes.Equal(receivedSBH, storedSBH) {
		return fmt.Errorf(
			"received SBH (%s) is different from stored SBH (%s)",
			hex.EncodeToString(receivedSBH),
			hex.EncodeToString(storedSBH),
		)
	}
	return nil
}
