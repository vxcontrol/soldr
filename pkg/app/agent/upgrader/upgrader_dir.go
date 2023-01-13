package upgrader

import (
	"context"
	"errors"
	"fmt"
	"os"

	upgraderUtils "soldr/pkg/app/agent/upgrader/utils"
	utils2 "soldr/pkg/app/agent/utils"
)

func createUpgraderLock(pr *utils2.PathResolver) error {
	if err := utils2.CreateLockFile(getUpgraderLockPath(pr)); err != nil {
		return fmt.Errorf("failed to create a lock file: %w", err)
	}
	return nil
}

func CleanUpgradeDir(ctx context.Context, logDir string) error {
	pr, err := upgraderUtils.NewPathResolver(logDir)
	if err != nil {
		return fmt.Errorf("failed to create a path resolver: %w", err)
	}
	if err := utils2.WatchLockFile(ctx, getUpgraderLockPath(pr)); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// if lockfile does not exist, check if the upgrade directory exists; if it does not do not report anything
			if _, err := os.Stat(pr.GetBaseDir()); err != nil {
				if errors.Is(err, os.ErrNotExist) {
					return nil
				}
			}
		}
		return fmt.Errorf("waiting for the upgrader lockfile release failed: %w", err)
	}
	if err := os.RemoveAll(pr.GetBaseDir()); err != nil {
		return fmt.Errorf("failed to remove the upgrader dir: %w", err)
	}
	return nil
}

func getUpgraderLockPath(pr *utils2.PathResolver) string {
	const upgraderLock = "upgrader.lock"
	return pr.Resolve(upgraderLock)
}
