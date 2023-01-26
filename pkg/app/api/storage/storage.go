package storage

import (
	"github.com/jinzhu/gorm"

	"soldr/pkg/app/api/models"
)

func GetAgentName(db *gorm.DB, hash string) (string, error) {
	var agent models.Agent
	if err := db.Take(&agent, "hash = ?", hash).Error; err != nil {
		return "", err
	}
	return agent.Description, nil
}

func GetGroupName(db *gorm.DB, hash string) (string, error) {
	var group models.Group
	if err := db.Take(&group, "hash = ?", hash).Error; err != nil {
		return "", err
	}
	return group.Info.Name.En, nil
}
