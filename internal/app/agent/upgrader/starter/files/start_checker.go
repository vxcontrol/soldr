package files

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	"soldr/internal/app/agent/upgrader/starter/types"
	"soldr/internal/app/agent/utils"
)

type StartChecker struct {
	readinessFilePath string
}

func NewStartChecker(logDir string, component types.ComponentName) (*StartChecker, error) {
	readinessFilePath, err := getReadinessFilePath(logDir, component)
	if err != nil {
		return nil, fmt.Errorf(failedToGetReadinessFilePathMsg, err)
	}
	c := &StartChecker{
		readinessFilePath: readinessFilePath,
	}
	if _, err := utils.RemoveIfExists(c.readinessFilePath); err != nil {
		return nil, err
	}
	return c, nil
}

func (checker *StartChecker) WaitForStart() error {
	const (
		upgraderStartTimeout         = time.Second * 10
		upgraderStartPollingInterval = time.Millisecond * 200
	)
	logrus.WithField("lock file", checker.readinessFilePath).Debug("waiting for a start signal")
	upgraderStartTimer, cancelUpgraderStartTimer := context.WithTimeout(context.Background(), upgraderStartTimeout)
	defer cancelUpgraderStartTimer()
	for {
		isRemoved, err := utils.RemoveIfExists(checker.readinessFilePath)
		if err != nil {
			return fmt.Errorf("failed to remove the upgrader readiness file: %w", err)
		}
		if isRemoved {
			return nil
		}
		select {
		case <-upgraderStartTimer.Done():
			return fmt.Errorf("upgrader readiness timeout has been exceeded")
		default:
			time.Sleep(upgraderStartPollingInterval)
		}
	}
}
