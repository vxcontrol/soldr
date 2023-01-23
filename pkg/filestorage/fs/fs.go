package fs

import (
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"soldr/pkg/filestorage"
)

// LocalStorage is main class for LocalStorage API
type LocalStorage struct {
	*filestorage.Limits
}

// New is function that construct LocalStorage driver with Storage
func New() (*LocalStorage, error) {
	return &LocalStorage{
		Limits: filestorage.NewLimits(
			0644,
			1024*1024*1024,
			1024*1024*1024,
			1024*1024,
		),
	}, nil
}

// ListDir is function that return listing directory with filea info
// Return key in the map is relative path from input base path.
func (s *LocalStorage) ListDir(path string) (map[string]os.FileInfo, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, filestorage.ErrListFailed
	}

	var numObjs int64
	tree := make(map[string]os.FileInfo)
	for _, f := range files {
		tree["/"+f.Name()] = f
		numObjs++
		if numObjs >= s.Limits.MaxNumObjs() {
			return nil, filestorage.ErrLimitExceeded
		}
	}

	return tree, nil
}

// addFile is additional function for walking on directory and return back info
func (s *LocalStorage) addFile(base string, tree map[string]os.FileInfo) filepath.WalkFunc {
	var numObjs int64
	return func(path string, info os.FileInfo, err error) error {
		if err != nil || numObjs >= s.Limits.MaxNumObjs() {
			return nil
		}

		tree[strings.TrimPrefix(filestorage.NormPath(path), base)] = info
		numObjs++

		return nil
	}
}

// ListDirRec is function that return listing directory with filea info
// Return key in the map is relative path from input base path.
func (s *LocalStorage) ListDirRec(path string) (map[string]os.FileInfo, error) {
	tree := make(map[string]os.FileInfo)
	err := filepath.Walk(path, s.addFile(filestorage.NormPath(path), tree))
	if err != nil {
		return nil, filestorage.ErrListFailed
	}
	if int64(len(tree)) >= s.Limits.MaxNumObjs() {
		return nil, filestorage.ErrLimitExceeded
	}

	return tree, nil
}

// GetInfo is function that return file info
func (s *LocalStorage) GetInfo(path string) (os.FileInfo, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, filestorage.ErrNotFound
	}
	if err != nil {
		return nil, filestorage.ErrOpenFailed
	}

	return info, nil
}

// IsExist is function that return true if file exists
func (s *LocalStorage) IsExist(path string) bool {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return false
	}

	return true
}

// IsNotExist is function that return true if file not exists
func (s *LocalStorage) IsNotExist(path string) bool {
	return !s.IsExist(path)
}

// ReadFile is function that return the file data
func (s *LocalStorage) ReadFile(path string) ([]byte, error) {
	if info, err := s.GetInfo(path); err != nil {
		return nil, err
	} else if info.IsDir() {
		return nil, filestorage.ErrNotFound
	} else if info.Size() > s.Limits.MaxFileSize() {
		return nil, filestorage.ErrLimitExceeded
	}
	if data, err := ioutil.ReadFile(path); err == nil {
		return data, nil
	}

	return nil, filestorage.ErrReadFailed
}

// readFiles is additional function for reading files data by path
func (s *LocalStorage) readFiles(files map[string]os.FileInfo, base string) (map[string][]byte, error) {
	var err error
	var numObjs, readSize int64
	tree := make(map[string][]byte)
	for name, info := range files {
		if info.IsDir() {
			continue
		}

		fpath := strings.Replace(filepath.Clean(filepath.Join(base, name)), "\\", "/", -1)
		if tree[name], err = s.ReadFile(fpath); err != nil {
			return nil, err
		}
		numObjs++
		readSize += info.Size()
		if numObjs >= s.Limits.MaxNumObjs() || readSize >= s.Limits.MaxReadSize() {
			return nil, filestorage.ErrLimitExceeded
		}
	}

	return tree, nil
}

// ReadDir is function that read all files in the directory
func (s *LocalStorage) ReadDir(path string) (map[string][]byte, error) {
	files, err := s.ListDir(path)
	if err != nil {
		return nil, err
	}

	return s.readFiles(files, path)
}

// ReadDirRec is function that recursive read all files in the directory
func (s *LocalStorage) ReadDirRec(path string) (map[string][]byte, error) {
	files, err := s.ListDirRec(path)
	if err != nil {
		return nil, err
	}

	return s.readFiles(files, path)
}

// CreateDir is function for create new directory if not exists
func (s *LocalStorage) CreateDir(path string) error {
	if !s.IsExist(path) {
		if os.Mkdir(path, s.Limits.DefPerm()) != nil {
			return filestorage.ErrCreateFailed
		}
		return nil
	}

	return filestorage.ErrAlreadyExists
}

// CreateFile is function for create new file if not exists
func (s *LocalStorage) CreateFile(path string) error {
	if !s.IsExist(path) {
		file, err := os.Create(path)
		if err != nil {
			return filestorage.ErrCreateFailed
		}
		if file.Close() != nil {
			return filestorage.ErrCloseFailed
		}
		return nil
	}

	return filestorage.ErrAlreadyExists
}

// WriteFile is function that write (override) data to a file
func (s *LocalStorage) WriteFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, s.Limits.DefPerm())
	if err != nil {
		return filestorage.ErrOpenFailed
	}
	if file.Truncate(0) != nil {
		return filestorage.ErrWriteFailed
	}
	if _, err := file.Write(data); err != nil {
		return filestorage.ErrWriteFailed
	}
	if file.Close() != nil {
		return filestorage.ErrCloseFailed
	}

	return nil
}

// AppendFile is function that append data to an exist file
func (s *LocalStorage) AppendFile(path string, data []byte) error {
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, s.Limits.DefPerm())
	if err != nil {
		return filestorage.ErrOpenFailed
	}
	if _, err := file.Write(data); err != nil {
		return filestorage.ErrWriteFailed
	}
	if file.Close() != nil {
		return filestorage.ErrCloseFailed
	}

	return nil
}

// RemoveDir is function that remove an exist directory
func (s *LocalStorage) RemoveDir(path string) error {
	info, err := s.GetInfo(path)
	if err == nil && info.IsDir() {
		if os.RemoveAll(path) != nil {
			return filestorage.ErrRemoveFailed
		}
		return nil
	}

	return err
}

// RemoveFile is function that remove an exist file
func (s *LocalStorage) RemoveFile(path string) error {
	info, err := s.GetInfo(path)
	if err == nil && !info.IsDir() {
		if os.Remove(path) != nil {
			return filestorage.ErrRemoveFailed
		}
		return nil
	}

	return err
}

// Remove is function that remove any exist object
func (s *LocalStorage) Remove(path string) error {
	if os.RemoveAll(path) != nil {
		return filestorage.ErrRemoveFailed
	}

	return nil
}

// Rename is function that rename any exist object to new
func (s *LocalStorage) Rename(src, dst string) error {
	if os.Rename(src, dst) != nil {
		return filestorage.ErrRenameFailed
	}

	return nil
}

// CopyFile is function that copies a file from src to dst
// If src and dst files exist, and are the same, then return success.
func (s *LocalStorage) CopyFile(src, dst string) error {
	sfi, err := s.GetInfo(src)
	if err != nil {
		return err
	}
	if !sfi.Mode().IsRegular() {
		return filestorage.ErrReadFailed
	}
	dfi, err := s.GetInfo(dst)
	if err != nil {
		if err != filestorage.ErrNotFound {
			return err
		}
	} else {
		if !dfi.Mode().IsRegular() {
			return filestorage.ErrWriteFailed
		}
		if os.SameFile(sfi, dfi) {
			return nil
		}
	}

	in, err := os.Open(src)
	if err != nil {
		return filestorage.ErrOpenFailed
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return filestorage.ErrCreateFailed
	}
	defer func() {
		if out.Close() != nil {
			err = filestorage.ErrCloseFailed
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return filestorage.ErrCopyFailed
	}
	if out.Sync() != nil {
		return filestorage.ErrCopyFailed
	}

	return err
}
