package files

import (
	"fmt"
	"io/ioutil"

	"soldr/pkg/app/agent/upgrader/starter/types"
)

type Starter struct {
	readinessFilePath string
}

func NewStarter(logDir string, component types.ComponentName) (*Starter, error) {
	readinessFilePath, err := getReadinessFilePath(logDir, component)
	if err != nil {
		return nil, fmt.Errorf(failedToGetReadinessFilePathMsg, err)
	}
	return &Starter{
		readinessFilePath: readinessFilePath,
	}, nil
}

func (s *Starter) SignalStart() error {
	if err := ioutil.WriteFile(s.readinessFilePath, nil, 0o600); err != nil {
		return fmt.Errorf("failed to write the readiness file %s: %w", s.readinessFilePath, err)
	}
	return nil
}
