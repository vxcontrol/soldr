package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// S3 is main class for S3 API
type S3 struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	host       string
	isSSL      bool
	con        *minio.Client
	sLimits
}

type S3ConnParams struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
}

// NewS3 is function that construct S3 driver with IStorage
func NewS3(connParams *S3ConnParams) (IStorage, error) {
	if connParams == nil {
		var err error
		connParams, err = getConnParamsFromEnvVars()
		if err != nil {
			return nil, err
		}
	}
	s := &S3{
		sLimits: sLimits{
			defPerm:     0644,
			maxFileSize: 1024 * 1024 * 1024,
			maxReadSize: 1024 * 1024 * 1024,
			maxNumObjs:  1024 * 1024,
		},
		endpoint:   connParams.Endpoint,
		accessKey:  connParams.AccessKey,
		secretKey:  connParams.SecretKey,
		bucketName: connParams.BucketName,
	}

	if stURL, err := url.Parse(s.endpoint); err == nil {
		s.isSSL = stURL.Scheme == "https"
		s.host = stURL.Host
	} else {
		return nil, ErrInternal
	}

	var err error
	s.con, err = minio.New(s.host, &minio.Options{
		Creds:  credentials.NewStaticV4(s.accessKey, s.secretKey, ""),
		Secure: s.isSSL,
	})
	if err != nil {
		return nil, ErrInternal
	}

	return s, nil
}

func getConnParamsFromEnvVars() (*S3ConnParams, error) {
	params := &S3ConnParams{}
	var ok bool
	genErr := func(missingEnvVar string) error {
		return fmt.Errorf("environment variable %s is undefined", missingEnvVar)
	}
	const (
		envVarS3Endpoint   = "MINIO_ENDPOINT"
		envVarS3AccessKey  = "MINIO_ACCESS_KEY"
		envVarS3SecretKey  = "MINIO_SECRET_KEY"
		envVarS3BucketName = "MINIO_BUCKET_NAME"
	)
	if params.Endpoint, ok = os.LookupEnv(envVarS3Endpoint); !ok {
		return nil, genErr(envVarS3Endpoint)
	}
	if params.AccessKey, ok = os.LookupEnv(envVarS3AccessKey); !ok {
		return nil, genErr(envVarS3AccessKey)
	}
	if params.SecretKey, ok = os.LookupEnv(envVarS3SecretKey); !ok {
		return nil, genErr(envVarS3SecretKey)
	}
	if params.BucketName, ok = os.LookupEnv(envVarS3BucketName); !ok {
		return nil, genErr(envVarS3BucketName)
	}
	return params, nil
}

// ListDir is function that return listing directory with filea info
func (s *S3) ListDir(path string) (map[string]os.FileInfo, error) {
	var numObjs int64
	tree := make(map[string]os.FileInfo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path = normPath(path)
	objectCh := s.con.ListObjects(ctx, s.bucketName, minio.ListObjectsOptions{
		Prefix:    path,
		Recursive: true,
	})
	for object := range objectCh {
		if object.Err != nil {
			return nil, ErrListFailed
		}
		shortPath := strings.TrimPrefix(object.Key, path)
		if !strings.HasPrefix(shortPath, "/") {
			shortPath = "/" + shortPath
		}
		dirName, fileName := filepath.Split(shortPath)
		spl := strings.Split(dirName, "/")
		if dirName == "/" {
			tree[shortPath] = &S3FileInfo{
				isDir:      false,
				path:       fileName,
				ObjectInfo: &object,
			}
			numObjs++
		} else if len(spl) >= 2 {
			dir := "/" + spl[1]
			if _, ok := tree[dir]; !ok {
				tree[dir] = &S3FileInfo{
					isDir: true,
					path:  spl[1],
				}
				numObjs++
			}
		}
		if numObjs >= s.maxNumObjs {
			return nil, ErrLimitExceeded
		}
	}

	return tree, nil
}

// addFile is additional function for walking on directory and return back info
func (s *S3) addFile(base string, tree map[string]os.FileInfo, numObjs *int64) error {
	ttree, err := s.ListDir(base)
	if err != nil {
		return err
	}

	for path, info := range ttree {
		npath := normPath(base + path)
		if _, ok := tree[npath]; !ok {
			tree[npath] = info
			(*numObjs)++
			if info.IsDir() {
				if err = s.addFile(npath, tree, numObjs); err != nil {
					return err
				}
			}
			if *numObjs >= s.maxNumObjs {
				return ErrLimitExceeded
			}
		}
	}

	return nil
}

// ListDirRec is function that return listing directory with filea info
func (s *S3) ListDirRec(path string) (map[string]os.FileInfo, error) {
	var numObjs int64
	tree := make(map[string]os.FileInfo)
	path = normPath(path)
	if err := s.addFile(path, tree, &numObjs); err != nil {
		return nil, err
	}

	if path != "" {
		rtree := make(map[string]os.FileInfo)
		for fpath, info := range tree {
			rtree[strings.TrimPrefix(fpath, path)] = info
		}
		return rtree, nil
	}

	return tree, nil
}

// GetInfo is function that return file info
func (s *S3) GetInfo(path string) (os.FileInfo, error) {
	path = normPath(path)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	objInfo, err := s.con.StatObject(ctx, s.bucketName, path, minio.StatObjectOptions{})
	if err != nil {
		tree, err := s.ListDir(path)
		if err != nil || len(tree) == 0 {
			return nil, ErrNotFound
		}
		return &S3FileInfo{
			isDir: true,
			path:  "/",
		}, nil
	}
	_, fileName := filepath.Split(path)

	return &S3FileInfo{
		isDir:      false,
		path:       fileName,
		ObjectInfo: &objInfo,
	}, nil
}

// IsExist is function that return true if file exists
func (s *S3) IsExist(path string) bool {
	if _, err := s.GetInfo(path); err != nil {
		return false
	}

	return true
}

// IsNotExist is function that return true if file not exists
func (s *S3) IsNotExist(path string) bool {
	return !s.IsExist(path)
}

// ReadFile is function that return the file data
func (s *S3) ReadFile(path string) ([]byte, error) {
	var objData []byte
	var objInfo minio.ObjectInfo
	if info, err := s.GetInfo(path); err != nil {
		return nil, err
	} else if info.IsDir() {
		return nil, ErrNotFound
	} else if info.Size() > s.maxFileSize {
		return nil, ErrLimitExceeded
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	obj, err := s.con.GetObject(ctx, s.bucketName, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}
	defer obj.Close()
	if objInfo, err = obj.Stat(); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}
	objData = make([]byte, objInfo.Size)
	if n, err := io.ReadFull(obj, objData); err != nil || n != int(objInfo.Size) {
		return nil, fmt.Errorf("%w: %v", ErrReadFailed, err)
	}

	return objData, nil
}

// readFiles is additional function for reading files data by path
func (s *S3) readFiles(files map[string]os.FileInfo, base string) (map[string][]byte, error) {
	var err error
	var numObjs, readSize int64
	tree := make(map[string][]byte)
	for name, info := range files {
		if info.IsDir() {
			continue
		}

		fpath := normPath(base + name)
		if tree[name], err = s.ReadFile(fpath); err != nil {
			return nil, err
		}
		numObjs++
		readSize += info.Size()
		if numObjs >= s.maxNumObjs || readSize >= s.maxReadSize {
			return nil, ErrLimitExceeded
		}
	}

	return tree, nil
}

// ReadDir is function that read all files in the directory
func (s *S3) ReadDir(path string) (map[string][]byte, error) {
	path = normPath(path)
	files, err := s.ListDir(path)
	if err != nil {
		return nil, err
	}

	return s.readFiles(files, path)
}

// ReadDirRec is function that recursive read all files in the directory
func (s *S3) ReadDirRec(path string) (map[string][]byte, error) {
	path = normPath(path)
	files, err := s.ListDirRec(path)
	if err != nil {
		return nil, err
	}

	return s.readFiles(files, path)
}

// CreateDir is function for create new directory if not exists
func (s *S3) CreateDir(path string) error {
	path = normPath(path)
	if !s.IsExist(path) {
		return nil
	}

	return ErrAlreadyExists
}

// CreateFile is function for create new file if not exists
func (s *S3) CreateFile(path string) error {
	path = normPath(path)
	if !s.IsExist(path) {
		r := bytes.NewReader([]byte{})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_, err := s.con.PutObject(ctx, s.bucketName, path[1:], r, 0,
			minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			return ErrCreateFailed
		}
		return nil
	}

	return ErrAlreadyExists
}

// WriteFile is function that write (override) data to a file
func (s *S3) WriteFile(path string, data []byte) error {
	path = normPath(path)
	r := bytes.NewReader(data)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := s.con.PutObject(ctx, s.bucketName, path[1:], r, r.Size(),
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return ErrWriteFailed
	}

	return nil
}

// AppendFile is function that append data to an exist file
func (s *S3) AppendFile(path string, data []byte) error {
	rdata, err := s.ReadFile(path)
	if err != nil {
		return err
	}

	return s.WriteFile(path, append(rdata, data...))
}

// RemoveDir is function that remove an exist directory
func (s *S3) RemoveDir(path string) error {
	path = normPath(path)
	info, err := s.GetInfo(path)
	if err == nil && info.IsDir() {
		files, err := s.ListDirRec(path)
		if err != nil {
			return ErrRemoveFailed
		}
		for fpath, info := range files {
			if !info.IsDir() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if s.con.RemoveObject(ctx, s.bucketName, path+fpath,
					minio.RemoveObjectOptions{}) != nil {
					return ErrRemoveFailed
				}
			}
		}
		return nil
	}

	return err
}

// RemoveFile is function that remove an exist file
func (s *S3) RemoveFile(path string) error {
	path = normPath(path)
	info, err := s.GetInfo(path)
	if err == nil && !info.IsDir() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if s.con.RemoveObject(ctx, s.bucketName, path,
			minio.RemoveObjectOptions{}) != nil {
			return ErrRemoveFailed
		}
		return nil
	}

	return err
}

// Remove is function that remove any exist object
func (s *S3) Remove(path string) error {
	info, err := s.GetInfo(path)
	if err == nil && !info.IsDir() {
		return s.RemoveFile(path)
	} else if err == nil && info.IsDir() {
		return s.RemoveDir(path)
	}

	return ErrRemoveFailed
}

// Rename is function that rename any exist object to new
func (s *S3) Rename(src, dst string) error {
	if err := s.CopyFile(src, dst); err != nil {
		return err
	}

	if s.RemoveFile(src) != nil {
		return ErrRenameFailed
	}

	return nil
}

// CopyFile is function that copies a file from src to dst
func (s *S3) CopyFile(src, dst string) error {
	isrc, err := s.GetInfo(src)
	if err != nil || isrc.IsDir() {
		return ErrNotFound
	}
	if s.IsExist(dst) {
		return ErrAlreadyExists
	}

	nsrc := minio.CopySrcOptions{
		Bucket: s.bucketName,
		Object: normPath(src)[1:],
	}
	ndst := minio.CopyDestOptions{
		Bucket: s.bucketName,
		Object: normPath(dst)[1:],
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err = s.con.CopyObject(ctx, ndst, nsrc); err != nil {
		return ErrCopyFailed
	}

	return nil
}
