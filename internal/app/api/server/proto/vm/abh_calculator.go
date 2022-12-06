package vm

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"soldr/internal/hardening/luavm/vm"

	"github.com/sirupsen/logrus"
)

type abhCalculator struct {
	abh    vm.ABH
	abhMux *sync.RWMutex
}

func newABHCalculator() (*abhCalculator, error) {
	abh, err := calculateABH()
	if err != nil {
		return nil, err
	}

	return &abhCalculator{
		abh:    abh,
		abhMux: &sync.RWMutex{},
	}, nil
}

func calculateABH() (vm.ABH, error) {
	execFile, err := getRunningExecutablePath()
	if err != nil {
		return nil, fmt.Errorf("failed to get the current executable path: %w", err)
	}
	// #nosec G304
	f, err := os.Open(execFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open the executable file %s: %w", execFile, err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			logrus.Errorf("failed to close file: %s", err)
		}
	}(f)
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return nil, fmt.Errorf("failed to get the hash of the executable file: %w", err)
	}
	return h.Sum(nil), nil
}

func getRunningExecutablePath() (string, error) {
	path, err := os.Executable()
	if err != nil {
		return "", err
	}
	path, err = filepath.EvalSymlinks(path)
	if err != nil {
		return "", fmt.Errorf("failed to evaluate the symbolic link to get the real path: %w", err)
	}
	return path, nil
}

func (a *abhCalculator) GetABH() (vm.ABH, error) {
	a.abhMux.RLock()
	defer a.abhMux.RUnlock()

	abhCopy := make([]byte, len(a.abh))
	if n := copy(abhCopy, a.abh); n != len(a.abh) {
		return nil, fmt.Errorf("expected to copy %d bytes, actually copied %d", len(a.abh), n)
	}
	return abhCopy, nil
}
