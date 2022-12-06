//nolint:staticcheck
package storage

//TODO: io/ioutil is deprecated, replace to fs.FS
import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

// FS is main class for FS API
type FS struct {
	sLimits
}

// NewFS is function that construct FS driver with IStorage
func NewFS() (*FS, error) {
	return &FS{
		sLimits: sLimits{
			defPerm:     0644,
			maxFileSize: 1024 * 1024 * 1024,
			maxReadSize: 1024 * 1024 * 1024,
			maxNumObjs:  1024 * 1024,
		},
	}, nil
}

// ListDir is function that return listing directory with filea info
// Return key in the map is relative path from input base path.
func (fs *FS) ListDir(path string) (map[string]os.FileInfo, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, ErrListFailed
	}

	var numObjs int64
	tree := make(map[string]os.FileInfo)
	for _, f := range files {
		tree["/"+f.Name()] = f
		numObjs++
		if numObjs >= fs.maxNumObjs {
			return nil, ErrLimitExceeded
		}
	}

	return tree, nil
}

// addFile is additional function for walking on directory and return back info
func (fs *FS) addFile(base string, tree map[string]os.FileInfo) filepath.WalkFunc {
	var numObjs int64
	return func(path string, info os.FileInfo, err error) error {
		if err != nil || numObjs >= fs.maxNumObjs {
			return nil
		}

		tree[strings.TrimPrefix(normPath(path), base)] = info
		numObjs++

		return nil
	}
}

// ListDirRec is function that return listing directory with filea info
// Return key in the map is relative path from input base path.
func (fs *FS) ListDirRec(path string) (map[string]os.FileInfo, error) {
	tree := make(map[string]os.FileInfo)
	err := filepath.Walk(path, fs.addFile(normPath(path), tree))
	if err != nil {
		return nil, ErrListFailed
	}
	if int64(len(tree)) >= fs.maxNumObjs {
		return nil, ErrLimitExceeded
	}

	return tree, nil
}

// GetInfo is function that return file info
func (fs *FS) GetInfo(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, ErrOpenFailed
	}

	return info, nil
}

// IsExist is function that return true if file exists
func (fs *FS) IsExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

// IsNotExist is function that return true if file not exists
func (fs *FS) IsNotExist(path string) bool {
	return !fs.IsExist(path)
}

// ReadFile is function that return the file data
func (fs *FS) ReadFile(path string) ([]byte, error) {
	if info, err := fs.GetInfo(path); err != nil {
		return nil, err
	} else if info.IsDir() {
		return nil, ErrNotFound
	} else if info.Size() > fs.maxFileSize {
		return nil, ErrLimitExceeded
	}
	// #nosec G304
	if data, err := ioutil.ReadFile(path); err == nil {
		return data, nil
	}

	return nil, ErrReadFailed
}

// readFiles is additional function for reading files data by path
func (fs *FS) readFiles(files map[string]os.FileInfo, base string) (map[string][]byte, error) {
	var err error
	var numObjs, readSize int64
	tree := make(map[string][]byte)
	for name, info := range files {
		if info.IsDir() {
			continue
		}

		fpath := strings.Replace(filepath.Clean(filepath.Join(base, name)), "\\", "/", -1)
		if tree[name], err = fs.ReadFile(fpath); err != nil {
			return nil, err
		}
		numObjs++
		readSize += info.Size()
		if numObjs >= fs.maxNumObjs || readSize >= fs.maxReadSize {
			return nil, ErrLimitExceeded
		}
	}

	return tree, nil
}

// ReadDir is function that read all files in the directory
func (fs *FS) ReadDir(path string) (map[string][]byte, error) {
	files, err := fs.ListDir(path)
	if err != nil {
		return nil, err
	}

	return fs.readFiles(files, path)
}

// ReadDirRec is function that recursive read all files in the directory
func (fs *FS) ReadDirRec(path string) (map[string][]byte, error) {
	files, err := fs.ListDirRec(path)
	if err != nil {
		return nil, err
	}

	return fs.readFiles(files, path)
}

// CreateDir is function for create new directory if not exists
func (fs *FS) CreateDir(path string) error {
	if !fs.IsExist(path) {
		if os.Mkdir(path, fs.defPerm) != nil {
			return ErrCreateFailed
		}
		return nil
	}

	return ErrAlreadyExists
}

// CreateFile is function for create new file if not exists
func (fs *FS) CreateFile(path string) error {
	if !fs.IsExist(path) {
		// #nosec G304
		file, err := os.Create(path)
		if err != nil {
			return ErrCreateFailed
		}
		if file.Close() != nil {
			return ErrCloseFailed
		}
		return nil
	}

	return ErrAlreadyExists
}

// WriteFile is function that write (override) data to a file
func (fs *FS) WriteFile(path string, data []byte) error {
	// #nosec G304
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, fs.defPerm)
	if err != nil {
		return ErrOpenFailed
	}
	if file.Truncate(0) != nil {
		return ErrWriteFailed
	}
	if _, err := file.Write(data); err != nil {
		return ErrWriteFailed
	}
	if file.Close() != nil {
		return ErrCloseFailed
	}

	return nil
}

// AppendFile is function that append data to an exist file
func (fs *FS) AppendFile(path string, data []byte) error {
	// #nosec G304
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, fs.defPerm)
	if err != nil {
		return ErrOpenFailed
	}
	if _, err := file.Write(data); err != nil {
		return ErrWriteFailed
	}
	if file.Close() != nil {
		return ErrCloseFailed
	}

	return nil
}

// RemoveDir is function that remove an exist directory
func (fs *FS) RemoveDir(path string) error {
	info, err := fs.GetInfo(path)
	if err == nil && info.IsDir() {
		if os.RemoveAll(path) != nil {
			return ErrRemoveFailed
		}
		return nil
	}

	return err
}

// RemoveFile is function that remove an exist file
func (fs *FS) RemoveFile(path string) error {
	info, err := fs.GetInfo(path)
	if err == nil && !info.IsDir() {
		if os.Remove(path) != nil {
			return ErrRemoveFailed
		}
		return nil
	}

	return err
}

// Remove is function that remove any exist object
func (fs *FS) Remove(path string) error {
	if os.RemoveAll(path) != nil {
		return ErrRemoveFailed
	}

	return nil
}

// Rename is function that rename any exist object to new
func (fs *FS) Rename(src, dst string) error {
	if os.Rename(src, dst) != nil {
		return ErrRenameFailed
	}

	return nil
}

// CopyFile is function that copies a file from src to dst
// If src and dst files exist, and are the same, then return success.
func (fs *FS) CopyFile(src, dst string) error {
	sfi, err := fs.GetInfo(src)
	if err != nil {
		return err
	}
	if !sfi.Mode().IsRegular() {
		return ErrReadFailed
	}
	dfi, err := fs.GetInfo(dst)
	if err != nil {
		if err != ErrNotFound {
			return err
		}
	} else {
		if !dfi.Mode().IsRegular() {
			return ErrWriteFailed
		}
		if os.SameFile(sfi, dfi) {
			return nil
		}
	}

	// #nosec G304
	in, err := os.Open(src)
	if err != nil {
		return ErrOpenFailed
	}
	defer func(in *os.File) {
		e := in.Close()
		if e != nil {
			logrus.Errorf("failed to close file: %s", e)
		}
	}(in)
	// #nosec G304
	out, err := os.Create(dst)
	if err != nil {
		return ErrCreateFailed
	}
	defer func() {
		if out.Close() != nil {
			err = ErrCloseFailed
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return ErrCopyFailed
	}
	if out.Sync() != nil {
		return ErrCopyFailed
	}

	return err
}
