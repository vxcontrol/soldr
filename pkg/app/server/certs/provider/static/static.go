package static

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"io/fs"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"soldr/pkg/app/server/certs/config"
	"soldr/pkg/app/server/certs/provider"
)

type x509Cert struct {
	CertPEM []byte
	Cert    *x509.Certificate
	Key     interface{}
}

type StaticProvider struct {
	vxcas     map[string]*x509Cert
	vxcasBlob []byte
	scas      map[string]*x509Cert
	certs     map[string]*tls.Certificate
	certsBlob []tls.Certificate
}

func NewProvider(c *config.StaticProvider) (*StaticProvider, error) {
	p := &StaticProvider{}
	var err error
	p.vxcas, err = getVXCAs(c.CertsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get the VXCA certificates: %w", err)
	}
	if len(p.vxcas) == 0 {
		return nil, fmt.Errorf("no VXCA certificate found")
	}
	p.vxcasBlob = composeVXCAsBlob(p.vxcas)
	p.scas, err = getSCAs(c.CertsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get the SCA certificates: %w", err)
	}
	if len(p.scas) == 0 {
		return nil, fmt.Errorf("no SCA certificate found")
	}
	p.certs, err = getCerts(c.CertsDir, p.vxcas, p.scas)
	if err != nil {
		return nil, fmt.Errorf("failed to get SCs: %w", err)
	}
	if len(p.certs) == 0 {
		return nil, fmt.Errorf("no SC found")
	}
	p.certsBlob = composeCertsBlob(p.certs)
	return p, nil
}

func getVXCAs(certsDir string) (map[string]*x509Cert, error) {
	const vxcasDir = "vxca"
	vxcasPath := getPath(certsDir, vxcasDir)
	filesInfo, err := getFilesInfoFromDir(vxcasPath)
	if err != nil {
		return nil, err
	}
	vxcas := make(map[string]*x509Cert, 1)
	const certExt = ".cert"
	for _, fi := range filesInfo {
		if fi.IsDir() {
			continue
		}
		if filepath.Ext(fi.Name()) != ".cert" {
			continue
		}
		fPath := filepath.Join(vxcasPath, fi.Name())
		data, err := os.ReadFile(fPath)
		if err != nil {
			return nil, err
		}
		cert, err := pemToX509Cert(data)
		if err != nil {
			return nil, fmt.Errorf("failed to get an X509 SCA cert: %w", err)
		}
		nameWithoutExt := strings.TrimSuffix(fi.Name(), certExt)
		vxcas[nameWithoutExt] = &x509Cert{
			CertPEM: data,
			Cert:    cert,
		}
	}
	return vxcas, nil
}

func composeVXCAsBlob(vxcas map[string]*x509Cert) []byte {
	blob := make([]byte, 0)
	for _, p := range vxcas {
		blob = append(blob, p.CertPEM...)
	}
	return blob
}

func getSCAs(certsDir string) (map[string]*x509Cert, error) {
	const scaDir = "sca"
	scasPath := getPath(certsDir, scaDir)
	filesInfo, err := getFilesInfoFromDir(scasPath)
	if err != nil {
		return nil, err
	}
	ckFiles, err := getCertKeyFiles(filesInfo, scasPath)
	if err != nil {
		return nil, err
	}
	scas := make(map[string]*x509Cert, len(ckFiles))
	for name, ck := range ckFiles {
		sca, err := composeSCA(ck.cert, ck.key)
		if err != nil {
			return nil, fmt.Errorf("failed to compose the %s SCA: %w", name, err)
		}
		scas[name] = sca
	}
	return scas, nil
}

func getPath(parts ...string) string {
	path := filepath.Join(parts...)
	return path
}

type certKeyFile struct {
	cert []byte
	key  []byte
}

type certKeyFiles map[string]*certKeyFile

func getCertKeyFiles(filesInfo []fs.FileInfo, baseDir string) (certKeyFiles, error) {
	const (
		certExt = ".cert"
		keyExt  = ".key"
	)
	result := make(certKeyFiles)
	for _, fi := range filesInfo {
		if fi.IsDir() {
			continue
		}
		ext := filepath.Ext(fi.Name())
		if ext != certExt && ext != keyExt {
			continue
		}
		fPath := filepath.Join(baseDir, fi.Name())
		data, err := os.ReadFile(fPath)
		if err != nil {
			return nil, err
		}
		nameWithoutExt := strings.TrimSuffix(fi.Name(), ext)
		var ckFile *certKeyFile
		var ok bool
		if ckFile, ok = result[nameWithoutExt]; !ok {
			ckFile = &certKeyFile{}
			result[nameWithoutExt] = ckFile
		}
		if ext == certExt {
			ckFile.cert = data
		}
		if ext == keyExt {
			ckFile.key = data
		}
	}
	for name, ck := range result {
		if ck.cert == nil || ck.key == nil {
			delete(result, name)
		}
	}
	return result, nil
}

func composeSCA(certData []byte, keyData []byte) (*x509Cert, error) {
	sca := &x509Cert{
		CertPEM: certData,
	}
	var err error
	sca.Cert, err = pemToX509Cert(certData)
	if err != nil {
		return nil, fmt.Errorf("failed to get an X509 SCA cert: %w", err)
	}
	sca.Key, err = getSCAKey(keyData, sca.Cert.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to get the SCA key: %w", err)
	}
	return sca, nil
}

func getCerts(certsDir string, vxcas, scas map[string]*x509Cert) (map[string]*tls.Certificate, error) {
	const scDir = "sc"
	scsPath := getPath(certsDir, scDir)
	filesInfo, err := getFilesInfoFromDir(scsPath)
	if err != nil {
		return nil, err
	}
	ckFiles, err := getCertKeyFiles(filesInfo, scsPath)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*tls.Certificate)
	for name, ck := range ckFiles {
		vxca, ok := vxcas[name]
		if !ok {
			return nil, fmt.Errorf("VXCA %s not found", name)
		}
		sca, ok := scas[name]
		if !ok {
			return nil, fmt.Errorf("SCA %s not found", name)
		}
		result[name], err = composeCert(ck, vxca.CertPEM, sca.CertPEM)
		if err != nil {
			return nil, fmt.Errorf("failed to compose a certificate %s: %w", name, err)
		}
	}
	return result, nil
}

func composeCert(ck *certKeyFile, vxca []byte, sca []byte) (*tls.Certificate, error) {
	cert, err := getTLSCertFromPEMKeyPair(ck.cert, sca, ck.key)
	if err != nil {
		return nil, fmt.Errorf("failed to get the SC TLS certificate: %w", err)
	}
	if err := checkTLSCertValidity(cert, vxca); err != nil {
		return nil, err
	}
	return cert, nil
}

func composeCertsBlob(certs map[string]*tls.Certificate) []tls.Certificate {
	blob := make([]tls.Certificate, 0, len(certs))
	for _, c := range certs {
		blob = append(blob, *c)
	}
	return blob
}

func getFilesInfoFromDir(dirPath string) ([]fs.FileInfo, error) {
	filesInfo, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}
	return filesInfo, nil
}

func (p *StaticProvider) VXCA() ([]byte, error) {
	return copyBytes(p.vxcasBlob), nil
}

func (p *StaticProvider) SC() ([]tls.Certificate, error) {
	certs := make([]tls.Certificate, len(p.certsBlob))
	if n := copy(certs, p.certsBlob); n != len(p.certsBlob) {
		return nil, fmt.Errorf("expected to copy %d elements, actually copied %d", len(p.certsBlob), n)
	}
	return certs, nil
}

func (p *StaticProvider) CreateLTACFromCSR(tlsConnState *tls.ConnectionState, csrData []byte) ([]byte, error) {
	csr, err := x509.ParseCertificateRequest(csrData)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the LTAC CSR: %w", err)
	}
	if err := csr.CheckSignature(); err != nil {
		return nil, fmt.Errorf("CSR signature check failed: %w", err)
	}
	sca, err := p.chooseSCACertToSignCSR(tlsConnState)
	if err != nil {
		return nil, fmt.Errorf("failed to choose an SCA to sign the received CSR: %w", err)
	}
	ltacSerialNum, err := generateCertSerialNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate a serial number for the new LTAC certificate: %w", err)
	}
	ltacTempl := &x509.Certificate{
		Signature:          csr.Signature,
		SignatureAlgorithm: csr.SignatureAlgorithm,
		PublicKeyAlgorithm: csr.PublicKeyAlgorithm,
		PublicKey:          csr.PublicKey,
		Subject:            csr.Subject,
		Issuer:             sca.Cert.Issuer,

		SerialNumber: ltacSerialNum,
		NotBefore:    time.Now().UTC().AddDate(0, 0, -1),
		NotAfter:     time.Now().UTC().AddDate(1, 0, 0),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		// TODO(SSH): the LTAC check on the client would not work without x509.ExtKeyUsageServerAuth,
		// is it OK or did I miss something during the CA certificates creation?
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth, x509.ExtKeyUsageServerAuth},
	}
	ltac, err := x509.CreateCertificate(rand.Reader, ltacTempl, sca.Cert, csr.PublicKey, sca.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create an LTAC certificate: %w", err)
	}
	f, err := ioutil.TempFile("", "vxcert")
	if err != nil {
		return nil, fmt.Errorf("failed to open an ltac cert file: %w", err)
	}
	defer func() { _ = f.Close() }()

	err = pem.Encode(f, &pem.Block{
		Type:  "CERTIFICATE",
		Bytes: ltac,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to encode PEM: %w", err)
	}

	return ltac, nil
}

func (p *StaticProvider) chooseSCACertToSignCSR(
	tlsConnState *tls.ConnectionState,
) (*x509Cert, error) {
	for _, chain := range tlsConnState.VerifiedChains {
		rootCA := chain[len(chain)-1]
		agentRootPK, err := getPK(rootCA)
		if err != nil {
			return nil, fmt.Errorf("failed to extract the PK from the agent's root certificate: %w", err)
		}
		sca, err := p.findCorrespondingSCA(agentRootPK)
		if err != nil {
			if errors.Is(err, errCorrespondingSCANotFound) {
				continue
			}
			return nil, err
		}
		return sca, nil
	}
	return nil, fmt.Errorf("no corresponding SCA found")
}

var errCorrespondingSCANotFound = errors.New("corresponding SCA not found")

func (p *StaticProvider) findCorrespondingSCA(agentPK ed25519.PublicKey) (*x509Cert, error) {
	for name, vxca := range p.vxcas {
		pk, err := getPK(vxca.Cert)
		if err != nil {
			return nil, fmt.Errorf("failed to extract the PK from a stored VXCA: %w", err)
		}
		if !pk.Equal(agentPK) {
			continue
		}
		sca, ok := p.scas[name]
		if !ok {
			return nil, fmt.Errorf("an SCA \"%s\" not found", name)
		}
		return sca, nil
	}
	return nil, errCorrespondingSCANotFound
}

func getPK(cert *x509.Certificate) (ed25519.PublicKey, error) {
	pk, ok := cert.PublicKey.(ed25519.PublicKey)
	if !ok {
		return pk, fmt.Errorf("an unexpected PK (%T)", cert.PublicKey)
	}
	return pk, nil
}

func getTLSCertFromPEMKeyPair(cert []byte, ca []byte, key []byte) (*tls.Certificate, error) {
	chain := copyBytes(cert)
	chain = append(chain, ca...)
	tlsCert, err := tls.X509KeyPair(chain, key)
	if err != nil {
		return nil, fmt.Errorf("failed to compose a TLS certificate from the passed keypair: %w", err)
	}
	return &tlsCert, nil
}

func checkTLSCertValidity(cert *tls.Certificate, root []byte) error {
	if len(cert.Certificate) != 2 {
		return fmt.Errorf("expected the certificate chain to contain two certificates, got %d", len(cert.Certificate))
	}
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return fmt.Errorf("failed to extract the X509 certificate: %w", err)
	}
	caPEM := certToPEM(cert.Certificate[1])
	if err := provider.CheckValidity(x509Cert, [][]byte{caPEM}, root); err != nil {
		return fmt.Errorf("server cert validity check failed: %w", err)
	}
	return nil
}

func certToPEM(cert []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})
}

func copyBytes(src []byte) []byte {
	res := make([]byte, len(src))
	copy(res, src)
	return res
}

func pemToX509Cert(certData []byte) (*x509.Certificate, error) {
	cert, _ := pem.Decode(certData)
	if cert == nil {
		return nil, fmt.Errorf("failed to get the certificate block from the cert PEM object")
	}
	x509Cert, err := x509.ParseCertificate(cert.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the PEM cert into an x509 cert: %w", err)
	}
	return x509Cert, nil
}

func getSCAKey(scaKeyData []byte, publicKey interface{}) (ed25519.PrivateKey, error) {
	scaKey, err := pemToKey(scaKeyData)
	if err != nil {
		return nil, err
	}
	if err := isKeysMatch(publicKey, scaKey); err != nil {
		return nil, err
	}
	key, ok := scaKey.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("the parsed key is not an ed25519 Private Key")
	}
	return key, nil
}

func isKeysMatch(public interface{}, private interface{}) error {
	switch publicKey := public.(type) {
	case ed25519.PublicKey:
		privateKey, ok := private.(ed25519.PrivateKey)
		if !ok {
			return fmt.Errorf("private key expected to be an ed25519 Private key, but type assertion has failed")
		}
		if !publicKey.Equal(privateKey.Public()) {
			return fmt.Errorf("public and private keys do not match")
		}
		return nil
	default:
		return fmt.Errorf("a public key of an unexpected type passed")
	}
}

func pemToKey(keyData []byte) (interface{}, error) {
	keyBlock, _ := pem.Decode(keyData)
	if keyBlock == nil {
		return nil, fmt.Errorf("failed to decode the PEM-encoded key file: no PEM block found")
	}
	key, err := x509.ParsePKCS8PrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the PEM-encoded private key: %w", err)
	}
	return key, nil
}

const maxInt64 = int64((^uint64(0)) >> 1)

func generateCertSerialNumber() (*big.Int, error) {
	unixNano := strconv.FormatInt(time.Now().UnixNano(), 10)
	nonce, err := rand.Int(rand.Reader, big.NewInt(maxInt64))
	if err != nil {
		return nil, fmt.Errorf("failed to get a random nonce: %w", err)
	}
	serialNumStr := unixNano + nonce.String()
	serialNum, ok := new(big.Int).SetString(serialNumStr, 10)
	if !ok {
		return nil, fmt.Errorf("failed to convert the string %s to a big int", serialNumStr)
	}
	return serialNum, nil
}
