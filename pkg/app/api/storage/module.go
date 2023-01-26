package storage

import (
	"fmt"
	"sync"

	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/models"
)

type ModuleStorage struct {
	mu    sync.RWMutex
	store map[uint64][]models.ModuleAShort
}

func NewModulesStorage() *ModuleStorage {
	return &ModuleStorage{
		store: make(map[uint64][]models.ModuleAShort),
	}
}

func (s *ModuleStorage) GetModulesAShortByGroup(groupID uint64) []models.ModuleAShort {
	return s.GetModulesAShort([]uint64{groupID})[groupID]
}

func (s *ModuleStorage) GetModulesAShort(groupIDs []uint64) map[uint64][]models.ModuleAShort {
	if groupIDs == nil || len(groupIDs) == 0 {
		return map[uint64][]models.ModuleAShort{}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make(map[uint64][]models.ModuleAShort)
	for _, groupID := range groupIDs {
		modules, found := s.store[groupID]
		if found {
			result[groupID] = modules
		}
	}

	return result
}

func (s *ModuleStorage) Refresh(serviceDB *gorm.DB) error {
	var modules []models.ModuleAShort
	resultList := make(map[uint64][]models.ModuleAShort)

	s.mu.Lock()
	defer s.mu.Unlock()

	err := serviceDB.Model(&models.ModuleAShort{}).
		Group("modules.id").
		Joins(`LEFT JOIN groups_to_policies gtp ON gtp.policy_id = modules.policy_id`).
		Find(&modules, "status = 'joined'").Error
	if err != nil {
		return err
	}

	for _, m := range modules {
		id := m.ID
		name := m.Info.Name
		policyID := m.PolicyID
		if err = m.Valid(); err != nil {
			return fmt.Errorf("error validating module data '%d' '%s': %w", id, name, err)
		}
		if _, ok := resultList[policyID]; !ok {
			resultList[policyID] = make([]models.ModuleAShort, 0)
		}
		resultList[policyID] = append(resultList[policyID], m)
	}

	s.store = resultList

	return nil
}
