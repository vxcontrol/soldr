package utils

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// GetRef is function for returning referense of string
func GetRef(str string) *string {
	return &str
}

// RemoveUnusedTempDir is public function that should use into worker on start
func RemoveUnusedTempDir() {
	files, err := ioutil.ReadDir(os.TempDir())
	if err != nil {
		return
	}

	for _, f := range files {
		if f.IsDir() && strings.HasPrefix(f.Name(), "vxlua-") {
			pathToPID := filepath.Join(os.TempDir(), f.Name(), "lock.pid")
			fdata, err := ioutil.ReadFile(pathToPID)
			if err != nil {
				os.RemoveAll(filepath.Join(os.TempDir(), f.Name()))
				continue
			}
			pid, err := strconv.Atoi(string(fdata))
			if err != nil {
				os.RemoveAll(filepath.Join(os.TempDir(), f.Name()))
				continue
			}
			proc, _ := os.FindProcess(pid)
			if proc == nil || err != nil {
				os.RemoveAll(filepath.Join(os.TempDir(), f.Name()))
				continue
			}
		}
	}
}
