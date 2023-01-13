package pswatcher

import (
	"context"
	"fmt"
	"time"

	gops "github.com/mitchellh/go-ps"

	"soldr/pkg/app/agent/upgrader/errors"
)

// TODO(SSH): the go-ps library gets the whole list of existing processes
// to get the status of the agent process. We should instead consider the option
// to localize just the agent process
func WaitForProcessToFinish(ctx context.Context, pid int, checkInterval time.Duration) error {
	for {
		select {
		case <-ctx.Done():
			return errors.ErrUpgraderTimeout
		default:
		}
		time.Sleep(checkInterval)

		isExist, err := IsProcessExist(pid)
		if err != nil {
			return err
		}
		if !isExist {
			return nil
		}
	}
}

func IsProcessExist(pid int) (bool, error) {
	proc, err := gops.FindProcess(pid)
	if err != nil {
		return false, fmt.Errorf("failed to find a process with the PID %d: %w", pid, err)
	}
	return proc != nil, nil
}
