package storage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
)

// Error list of storage package
var (
	ErrInternal      = errors.New("internal error")
	ErrAlreadyExists = errors.New("already exists")
	ErrNotFound      = errors.New("not found")
	ErrLimitExceeded = errors.New("limit exceeded")
	ErrListFailed    = errors.New("can't list")
	ErrCopyFailed    = errors.New("can't copy")
	ErrOpenFailed    = errors.New("can't open")
	ErrCreateFailed  = errors.New("can't create")
	ErrReadFailed    = errors.New("can't read")
	ErrWriteFailed   = errors.New("can't write")
	ErrCloseFailed   = errors.New("can't close")
	ErrRemoveFailed  = errors.New("can't remove")
	ErrRenameFailed  = errors.New("can't rename")
)

// IStorage is main interface for using external storages
type IStorage interface {
	ListDir(path string) (map[string]os.FileInfo, error)
	ListDirRec(path string) (map[string]os.FileInfo, error)
	GetInfo(path string) (os.FileInfo, error)
	IsExist(path string) bool
	IsNotExist(path string) bool
	ReadDir(path string) (map[string][]byte, error)
	ReadDirRec(path string) (map[string][]byte, error)
	CreateDir(path string) error
	CreateFile(path string) error
	WriteFile(path string, data []byte) error
	AppendFile(path string, data []byte) error
	RemoveDir(path string) error
	RemoveFile(path string) error
	Remove(path string) error
	Rename(old, new string) error
	CopyFile(src, dst string) error
	IFileReader
	ILimits
}

type IFileReader interface {
	ReadFile(path string) ([]byte, error)
}

// ILimits is additional interface for limits control
type ILimits interface {
	DefPerm() os.FileMode
	SetDefPerm(perm os.FileMode)
	MaxFileSize() int64
	SetMaxFileSize(max int64)
	MaxReadSize() int64
	SetMaxReadSize(max int64)
	MaxNumObjs() int64
	SetMaxNumObjs(max int64)
}

type sLimits struct {
	defPerm     os.FileMode
	maxFileSize int64
	maxReadSize int64
	maxNumObjs  int64
}

func (l *sLimits) DefPerm() os.FileMode {
	return l.defPerm
}

func (l *sLimits) SetDefPerm(perm os.FileMode) {
	l.defPerm = perm
}

func (l *sLimits) MaxFileSize() int64 {
	return l.maxFileSize
}

func (l *sLimits) SetMaxFileSize(max int64) {
	l.maxFileSize = max
}

func (l *sLimits) MaxReadSize() int64 {
	return l.maxReadSize
}

func (l *sLimits) SetMaxReadSize(max int64) {
	l.maxReadSize = max
}

func (l *sLimits) MaxNumObjs() int64 {
	return l.maxNumObjs
}

func (l *sLimits) SetMaxNumObjs(max int64) {
	l.maxNumObjs = max
}

// S3FileInfo is struct with interface os.FileInfo
type S3FileInfo struct {
	isDir bool
	path  string
	*minio.ObjectInfo
}

// Name is function that return file name
func (si *S3FileInfo) Name() string {
	return si.path
}

// Size is function that return file size
func (si *S3FileInfo) Size() int64 {
	if si.ObjectInfo == nil {
		return 0
	}

	return si.ObjectInfo.Size
}

// Mode is function that return file mod structure
func (si *S3FileInfo) Mode() os.FileMode {
	return 0644
}

// ModTime is function that return last modification time
func (si *S3FileInfo) ModTime() time.Time {
	if si.ObjectInfo == nil {
		return time.Now()
	}

	return si.ObjectInfo.LastModified
}

// IsDir is function that return true if it's directory
func (si *S3FileInfo) IsDir() bool {
	return si.isDir
}

// Sys is function that return dummy info
func (si *S3FileInfo) Sys() interface{} {
	return nil
}

// normPath is function that return path which was normalization
func normPath(path string) string {
	if path != "" {
		path = strings.Replace(filepath.Clean(path), "\\", "/", -1)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return strings.TrimSuffix(path, "/")
}
