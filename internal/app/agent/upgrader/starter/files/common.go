package files

import (
	"fmt"

	"soldr/internal/app/agent/upgrader/starter/types"
	upgraderUtils "soldr/internal/app/agent/upgrader/utils"
)

const failedToGetReadinessFilePathMsg = "failed to get the readiness file path: %w"

func getReadinessFilePath(logDir string, component types.ComponentName) (string, error) {
	readinessFileName := fmt.Sprintf("%s-ready.lock", string(component))
	pathResolver, err := upgraderUtils.NewPathResolver(logDir)
	if err != nil {
		return "", err
	}
	return pathResolver.Resolve(readinessFileName), nil
}
