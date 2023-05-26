package vm

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"soldr/pkg/hardening/luavm/vm"
)

var abhCurrentBinary vm.ABH

type abhCalculator struct {
	abh    vm.ABH
	abhMux *sync.RWMutex
}

func newABHCalculator() (*abhCalculator, error) {
	if abhCurrentBinary != nil {
		return &abhCalculator{
			abh:    abhCurrentBinary,
			abhMux: &sync.RWMutex{},
		}, nil
	}

	abh, err := calculateABH()
	if err != nil {
		return nil, err
	} else {
		abhCurrentBinary = make([]byte, len(abh))
		copy(abhCurrentBinary, abh)
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
	f, err := os.Open(execFile)
	if err != nil {
		return nil, fmt.Errorf("failed to open the executable file %s: %w", execFile, err)
	}
	defer f.Close()
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
