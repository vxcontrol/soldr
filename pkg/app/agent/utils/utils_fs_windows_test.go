//go:build windows
// +build windows

package utils

import (
	"errors"
	"os"
	"testing"
)

func Test_getDirsToCreate_winRoot(t *testing.T) {
	tmpDir, cleanupTmpDir, err := createTmpDir(t, "C:\\", "vxagent_test_getDirsToCreate_")
	if err != nil {
		t.Errorf("failed to create a temp dir %s: %v", tmpDir, err)
		return
	}
	cleanupTmpDir()
	expectedDirsToCreate := []string{tmpDir}
	actualDirsToCreate, err := getDirsToCreate(tmpDir)
	if err != nil {
		t.Errorf("getDirsToCreateFailed: %v", err)
		return
	}
	if err := compareStringSlices(expectedDirsToCreate, actualDirsToCreate); err != nil {
		t.Errorf("expected and actual dirs to create are different: %v", err)
		return
	}
}

func Test_checkDstDir_winRoot(t *testing.T) {
	tmpDir, cleanupTmpDir, err := createTmpDir(t, "C:\\", "vxagent_test_getDirsToCreate_")
	if err != nil {
		t.Errorf("failed to create a temp dir %s: %v", tmpDir, err)
		return
	}
	cleanupTmpDir()

	cleanupDir, err := checkDstDir(tmpDir)
	if err != nil {
		t.Errorf("checkDstDir failed: %v", err)
		return
	}
	type testErr error
	var someErr testErr = testErr(errors.New("some err"))

	err = cleanupDir(someErr)
	if !errors.Is(err, someErr) || err.Error() != someErr.Error() {
		t.Errorf("expected error from the cleanup function: %v, got: %v", someErr, err)
		return
	}
	if _, err := os.Stat("C:\\"); err != nil {
		t.Errorf("failed to get the info of the dir C:\\ expected to exist: %w", err)
	}
	_, err = os.Stat(tmpDir)
	if err == nil {
		t.Errorf("got the info of the dir %s expected to not exist: %w", tmpDir, err)
		return
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("while getting the dir %s info, got an unexpected error: %w", tmpDir, err)
		return
	}
}
