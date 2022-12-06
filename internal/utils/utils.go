//nolint:staticcheck
package utils

//TODO: io/ioutil is deprecated, replace to fs.FS and delete "nolint:staticcheck"
import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
)

// GetRef is function for returning reference of string
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
			fdata, err := ioutil.ReadFile(filepath.Join(os.TempDir(), f.Name(), "lock.pid"))
			path := filepath.Join(os.TempDir(), f.Name())
			if err != nil {
				removeAll(path)
				continue
			}
			pid, err := strconv.Atoi(string(fdata))
			if err != nil {
				removeAll(path)
				continue
			}
			proc, _ := os.FindProcess(pid)
			if proc == nil || err != nil {
				removeAll(path)
				continue
			}
		}
	}
}

func removeAll(path string) {
	e := os.RemoveAll(path)
	if e != nil {
		logrus.Errorf("failed to remove all: %s", e)
	}
}
