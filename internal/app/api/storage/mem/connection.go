package mem

import (
	"errors"
	"sync"

	"soldr/internal/storage"

	"github.com/jinzhu/gorm"
)

var ErrNotFound = errors.New("not found")

type DBConnectionStorage struct {
	mu    sync.RWMutex // protects map below
	store map[string]*gorm.DB
}

func NewDBConnectionStorage() *DBConnectionStorage {
	return &DBConnectionStorage{
		store: make(map[string]*gorm.DB),
	}
}

func (s *DBConnectionStorage) Get(hash string) (*gorm.DB, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conns, found := s.store[hash]
	if !found {
		return nil, ErrNotFound
	}
	return conns, nil
}

func (s *DBConnectionStorage) Set(hash string, db *gorm.DB) {
	s.mu.Lock()
	s.store[hash] = db
	s.mu.Unlock()
}

type S3ConnectionStorage struct {
	mu sync.RWMutex // protects map below
	// TODO: store S3 struct instead of interface
	store map[string]storage.IStorage
}

func NewS3ConnectionStorage() *S3ConnectionStorage {
	return &S3ConnectionStorage{
		store: make(map[string]storage.IStorage),
	}
}

func (s *S3ConnectionStorage) Get(hash string) (storage.IStorage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, found := s.store[hash]
	if !found {
		return nil, ErrNotFound
	}
	return conn, nil
}

func (s *S3ConnectionStorage) Set(hash string, s3 storage.IStorage) {
	s.mu.Lock()
	s.store[hash] = s3
	s.mu.Unlock()
}
