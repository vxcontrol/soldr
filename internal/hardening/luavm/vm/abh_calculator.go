package vm

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type ABH []byte

type ABHCalculator interface {
	GetABH() (ABH, error)
}

type abhCalculator struct{}

func newABHCalculator() ABHCalculator {
	return &abhCalculator{}
}

func (c *abhCalculator) GetABH() (ABH, error) {
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
