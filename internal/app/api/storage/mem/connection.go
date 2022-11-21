package mem

import (
	"errors"
	"fmt"
	"strconv"
	"sync"

	"soldr/internal/storage"

	"github.com/jinzhu/gorm"

	"soldr/internal/app/api/models"
	"soldr/internal/app/api/utils"
)

var ErrNotFound = errors.New("not found")

type ServiceDBConnectionStorage struct {
	mu    sync.RWMutex // protects map below
	store map[string]*gorm.DB
}

func NewServiceDBConnectionStorage() *ServiceDBConnectionStorage {
	return &ServiceDBConnectionStorage{
		store: make(map[string]*gorm.DB),
	}
}

func (s *ServiceDBConnectionStorage) Get(hash string) (*gorm.DB, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conns, found := s.store[hash]
	if !found {
		return nil, ErrNotFound
	}
	return conns, nil
}

func (s *ServiceDBConnectionStorage) Load(services []models.Service) error {
	if len(services) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range services {
		service := services[i]

		_, exists := s.store[service.Hash]
		if exists {
			continue
		}

		db := utils.GetDB(
			service.Info.DB.User,
			service.Info.DB.Pass,
			service.Info.DB.Host,
			strconv.Itoa(int(service.Info.DB.Port)),
			service.Info.DB.Name,
		)
		if db == nil {
			return fmt.Errorf("could not connect to service database: %d", service.ID)
		}
		s.store[service.Hash] = db
	}
	return nil
}

type ServiceS3ConnectionStorage struct {
	mu    sync.RWMutex // protects map below
	store map[string]storage.IStorage
}

func NewServiceS3ConnectionStorage() *ServiceS3ConnectionStorage {
	return &ServiceS3ConnectionStorage{
		store: make(map[string]storage.IStorage),
	}
}

func (s *ServiceS3ConnectionStorage) Get(hash string) (storage.IStorage, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	conn, found := s.store[hash]
	if !found {
		return nil, ErrNotFound
	}
	return conn, nil
}

func (s *ServiceS3ConnectionStorage) Load(services []models.Service) error {
	if len(services) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range services {
		service := services[i]

		_, exists := s.store[service.Hash]
		if exists {
			continue
		}

		// TODO: return S3 struct instead of interface
		conn, err := storage.NewS3(service.Info.S3.ToS3ConnParams())
		if err != nil {
			return fmt.Errorf("could not create service S3 client: %d", service.ID)
		}
		s.store[service.Hash] = conn
	}
	return nil
}
