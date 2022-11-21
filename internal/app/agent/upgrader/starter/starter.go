package starter

import (
	"fmt"

	files2 "soldr/internal/app/agent/upgrader/starter/files"
	"soldr/internal/app/agent/upgrader/starter/types"
)

type Starter interface {
	SignalStart() error
}

type StartChecker interface {
	WaitForStart() error
}

func NewStarter(logDir string, component types.ComponentName) (Starter, error) {
	s, err := files2.NewStarter(logDir, component)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a file-based upgrader starter: %w", err)
	}
	return s, nil
}

func NewStartChecker(logDir string, component types.ComponentName) (StartChecker, error) {
	checker, err := files2.NewStartChecker(logDir, component)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a file-based upgrader start checker: %w", err)
	}
	return checker, nil
}
