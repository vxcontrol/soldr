package utils

import (
	"path/filepath"

	"soldr/pkg/app/agent/utils"
)

func NewPathResolver(logDir string) (*utils.PathResolver, error) {
	const upgraderFilesDir = "upgrader_files"
	return utils.NewPathResolver(filepath.Join(logDir, upgraderFilesDir))
}
