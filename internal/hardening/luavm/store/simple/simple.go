package simple

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"

	"soldr/internal/hardening/luavm/store/encryptor"
	"soldr/internal/hardening/luavm/store/types"
)

type Store struct {
	dir       string
	mux       *sync.RWMutex
	encryptor encryptor.Encryptor
}

type storeBlob struct {
	LTAC    types.LTACPublic `json:"ltac"`
	LTACKey []byte           `json:"ltac_key"`
	SBH     []byte           `json:"sbh"`
}

func NewStore(c *types.Config) (*Store, error) {
	if err := checkStoreDir(c.Dir); err != nil {
		return nil, err
	}
	s := &Store{
		dir: c.Dir,
		mux: &sync.RWMutex{},
	}
	var err error
	s.encryptor, err = encryptor.New()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize the store encryptor object: %w", err)
	}
	return s, nil
}

func checkStoreDir(dir string) error {
	info, err := os.Stat(dir)
	if err == nil {
		if info.IsDir() {
			return nil
		}
		return fmt.Errorf("passed value %s is not a dir", dir)
	}
	if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to get info about the directory %s: %w", dir, err)
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("failed to create a store dir %s: %w", dir, err)
	}
	return nil
}

func (s *Store) GetLTAC(key []byte) (*types.LTAC, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	blob, err := s.getBlob(key)
	if err != nil {
		return nil, err
	}
	return &types.LTAC{
		LTACPublic: blob.LTAC,
		Key:        blob.LTACKey,
	}, nil
}

func (s *Store) GetSBH(key []byte) ([]byte, error) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	blob, err := s.getBlob(key)
	if err != nil {
		return nil, err
	}
	return blob.SBH, nil
}

const (
	blobFile = "blob.ssa"
)

func (s *Store) StoreInitConnectionPack(p *types.InitConnectionPack, key []byte) error {
	s.mux.Lock()
	defer s.mux.Unlock()

	blob := &storeBlob{
		LTACKey: p.LTAC.Key,
		LTAC:    p.LTAC.LTACPublic,
		SBH:     p.SBH,
	}
	blobData, err := json.Marshal(blob)
	if err != nil {
		return fmt.Errorf("failed to marshal the LTAC data: %w", err)
	}
	blobData, err = s.encryptor.Encrypt(blobData, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt the init connection pack: %w", err)
	}
	if err := s.writeFile(blobFile, blobData); err != nil {
		return fmt.Errorf("failed to store the init data: %w", err)
	}
	return nil
}

func (s *Store) getBlob(key []byte) (blob *storeBlob, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("%w: %v", types.ErrNotInitialized, err)
		}
	}()
	blobData, err := s.readFile(blobFile)
	if err != nil {
		return nil, fmt.Errorf("failed to get the blob data: %w", err)
	}
	blobData, err = s.encryptor.Decrypt(blobData, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt the blob data: %w", err)
	}
	blob = &storeBlob{}
	if err := json.Unmarshal(blobData, blob); err != nil {
		return nil, fmt.Errorf("failed to unmarshal the store blob: %w", err)
	}
	return blob, nil
}

func (s *Store) readFile(name string) ([]byte, error) {
	contents, err := ioutil.ReadFile(s.getPath(name))
	if err != nil {
		return nil, fmt.Errorf("failed to read the file %s: %w", name, err)
	}
	return contents, nil
}

func (s *Store) writeFile(name string, data []byte) error {
	path := s.getPath(name)
	if err := checkStoreDir(filepath.Dir(path)); err != nil {
		return err
	}
	if err := ioutil.WriteFile(path, data, 0o600); err != nil {
		return fmt.Errorf("failed to write the file %s: %w", name, err)
	}
	return nil
}

func (s *Store) Reset() error {
	s.mux.Lock()
	defer s.mux.Unlock()

	blobFilePath := s.getPath(blobFile)
	if err := os.Remove(blobFilePath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("failed to remove the blob file %s: %w", blobFilePath, err)
	}
	return nil
}

func (s *Store) getPath(fileName string) string {
	return filepath.Join(s.dir, fileName)
}
