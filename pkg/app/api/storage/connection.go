package storage

import (
	"errors"
	"sync"

	"github.com/jinzhu/gorm"

	"soldr/pkg/filestorage"
)

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
		return nil, errors.New("not found")
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
	// TODO: store RemoteStorage struct instead of interface
	store map[string]filestorage.Storage
}

func NewS3ConnectionStorage() *S3ConnectionStorage {
	return &S3ConnectionStorage{
		store: make(map[string]filestorage.Storage),
	}
}

func (s *S3ConnectionStorage) Get(hash string) (filestorage.Storage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, found := s.store[hash]
	if !found {
		return nil, errors.New("not found")
	}
	return conn, nil
}

func (s *S3ConnectionStorage) Set(hash string, s3 filestorage.Storage) {
	s.mu.Lock()
	s.store[hash] = s3
	s.mu.Unlock()
}
