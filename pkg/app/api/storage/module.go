package storage

import (
	"fmt"

	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/models"
)

type ModuleStorage struct {
}

func NewModulesStorage() *ModuleStorage {
	return &ModuleStorage{}
}

func (s *ModuleStorage) GetModulesAShortByGroup(serviceDB *gorm.DB, groupID uint64) ([]models.ModuleAShort, error) {
	return s.GetModulesAShortByGroups(serviceDB, []uint64{groupID})
}

func (s *ModuleStorage) GetModulesAShortByGroups(serviceDB *gorm.DB,
	groupIDs []uint64) ([]models.ModuleAShort, error) {
	modules := []models.ModuleAShort{}
	if len(groupIDs) == 0 {
		return modules, nil
	}

	err := serviceDB.Model(&models.ModuleAShort{}).
		Group("modules.id").
		Joins(`LEFT JOIN groups_to_policies gtp ON gtp.policy_id = modules.policy_id`).
		Find(&modules, "gtp.group_id IN (?) AND status = 'joined'", groupIDs).Error
	if err != nil {
		return nil, fmt.Errorf("error finding modules: %w", err)
	} else {
		for _, module := range modules {
			id := module.ID
			name := module.Info.Name
			if err = module.Valid(); err != nil {
				return nil, fmt.Errorf("error validating module data '%d' '%s': %w", id, name, err)
			}
		}
	}

	return modules, nil
}

func (s *ModuleStorage) GetPoliciesModulesAShortByGroups(serviceDB *gorm.DB,
	groupIDs []uint64) (map[uint64][]models.ModuleAShort, error) {
	modulesToPolicies := make(map[uint64][]models.ModuleAShort)
	if len(groupIDs) == 0 {
		return modulesToPolicies, nil
	}

	modules, err := s.GetModulesAShortByGroups(serviceDB, groupIDs)
	if err != nil {
		return nil, err
	}
	for _, module := range modules {
		policyID := module.PolicyID
		if mods, ok := modulesToPolicies[policyID]; ok {
			modulesToPolicies[policyID] = append(mods, module)
		} else {
			modulesToPolicies[policyID] = []models.ModuleAShort{module}
		}
	}

	return modulesToPolicies, nil
}
