package utils

import (
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"soldr/internal/app/agent/utils/pswatcher"
)

func CreateLockFile(name string) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o666)
	if err != nil {
		return fmt.Errorf("failed to create a lock file: %w", err)
	}
	// the file.Close() function is explicitly not deferred: https://www.joeshaw.org/dont-defer-close-on-writable-files/

	if err := writePID(f); err != nil {
		f.Close()
		return fmt.Errorf("failed to write the PID to the lock file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to properly close the lock file: %w", err)
	}
	return nil
}

func writePID(f *os.File) error {
	pid := strconv.Itoa(os.Getpid())
	if _, err := f.Write([]byte(pid)); err != nil {
		return fmt.Errorf("failed to write the process PID to the lock file: %w", err)
	}
	return nil
}

func WatchLockFile(ctx context.Context, name string) error {
	pid, err := getPID(name)
	if err != nil {
		return fmt.Errorf("failed to get PID from the lock file: %w", err)
	}
	if err := pswatcher.WaitForProcessToFinish(ctx, pid, time.Millisecond*100); err != nil {
		return err
	}
	if err := os.Remove(name); err != nil {
		return fmt.Errorf("failed to remove the lock file: %w", err)
	}
	return nil
}

func getPID(name string) (int, error) {
	pidBytes, err := ioutil.ReadFile(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("lockfile %s does not exist: %w", name, err)
		}
		return 0, fmt.Errorf("failed to get the file %s info: %w", name, err)
	}
	pid64, err := strconv.ParseInt(string(pidBytes), 0, 0)
	if err != nil {
		return 0, fmt.Errorf("failed to parse the PID string: %w", err)
	}
	return int(pid64), err
}
