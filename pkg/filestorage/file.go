package filestorage

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// Storage is main interface for using external storages
type Storage interface {
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
	Reader
	Limiter
}

type Reader interface {
	ReadFile(path string) ([]byte, error)
}

// Limiter is additional interface for limits control
type Limiter interface {
	DefPerm() os.FileMode
	SetDefPerm(perm os.FileMode)
	MaxFileSize() int64
	SetMaxFileSize(max int64)
	MaxReadSize() int64
	SetMaxReadSize(max int64)
	MaxNumObjs() int64
	SetMaxNumObjs(max int64)
}

type Limits struct {
	defPerm     os.FileMode
	maxFileSize int64
	maxReadSize int64
	maxNumObjs  int64
}

func NewLimits(defPerm os.FileMode, maxFileSize int64, maxReadSize int64, maxNumObjs int64) *Limits {
	return &Limits{defPerm: defPerm, maxFileSize: maxFileSize, maxReadSize: maxReadSize, maxNumObjs: maxNumObjs}
}

func (l *Limits) DefPerm() os.FileMode {
	return l.defPerm
}

func (l *Limits) SetDefPerm(perm os.FileMode) {
	l.defPerm = perm
}

func (l *Limits) MaxFileSize() int64 {
	return l.maxFileSize
}

func (l *Limits) SetMaxFileSize(max int64) {
	l.maxFileSize = max
}

func (l *Limits) MaxReadSize() int64 {
	return l.maxReadSize
}

func (l *Limits) SetMaxReadSize(max int64) {
	l.maxReadSize = max
}

func (l *Limits) MaxNumObjs() int64 {
	return l.maxNumObjs
}

func (l *Limits) SetMaxNumObjs(max int64) {
	l.maxNumObjs = max
}

// NormPath is function that return path which was normalization
func NormPath(path string) string {
	if path != "" {
		path = strings.Replace(filepath.Clean(path), "\\", "/", -1)
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return strings.TrimSuffix(path, "/")
}
