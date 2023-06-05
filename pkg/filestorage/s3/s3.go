package s3

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"soldr/pkg/filestorage"
)

type Client struct {
	endpoint   string
	accessKey  string
	secretKey  string
	bucketName string
	host       string
	isSSL      bool
	con        *minio.Client
	*filestorage.Limits
}

type Config struct {
	Endpoint   string
	AccessKey  string
	SecretKey  string
	BucketName string
}

func New(cfg *Config) (*Client, error) {
	if cfg == nil {
		var err error
		cfg, err = getConnParamsFromEnvVars()
		if err != nil {
			return nil, err
		}
	}
	s := &Client{
		Limits: filestorage.NewLimits(
			0644,
			1024*1024*1024,
			1024*1024*1024,
			1024*1024,
		),
		endpoint:   cfg.Endpoint,
		accessKey:  cfg.AccessKey,
		secretKey:  cfg.SecretKey,
		bucketName: cfg.BucketName,
	}

	if stURL, err := url.Parse(s.endpoint); err == nil {
		s.isSSL = stURL.Scheme == "https"
		s.host = stURL.Host
	} else {
		return nil, filestorage.ErrInternal
	}

	var err error
	s.con, err = minio.New(s.host, &minio.Options{
		Creds:  credentials.NewStaticV4(s.accessKey, s.secretKey, ""),
		Secure: s.isSSL,
	})
	if err != nil {
		return nil, filestorage.ErrInternal
	}

	return s, nil
}

func getConnParamsFromEnvVars() (*Config, error) {
	params := &Config{}
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
func (c *Client) ListDir(path string) (map[string]os.FileInfo, error) {
	var numObjs int64
	tree := make(map[string]os.FileInfo)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	path = filestorage.NormPath(path)
	objectCh := c.con.ListObjects(ctx, c.bucketName, minio.ListObjectsOptions{
		Prefix:    path,
		Recursive: true,
	})
	for object := range objectCh {
		if object.Err != nil {
			return nil, filestorage.ErrListFailed
		}
		shortPath := strings.TrimPrefix(object.Key, path+"/")
		if !strings.HasPrefix(shortPath, "/") {
			shortPath = "/" + shortPath
		}
		dirName, fileName := filepath.Split(shortPath)
		spl := strings.Split(dirName, "/")
		if dirName == "/" {
			tree[shortPath] = &FileInfo{
				isDir:      false,
				path:       fileName,
				ObjectInfo: &object,
			}
			numObjs++
		} else if len(spl) >= 2 {
			dir := "/" + spl[1]
			if _, ok := tree[dir]; !ok {
				tree[dir] = &FileInfo{
					isDir: true,
					path:  spl[1],
				}
				numObjs++
			}
		}
		if numObjs >= c.Limits.MaxNumObjs() {
			return nil, filestorage.ErrLimitExceeded
		}
	}

	return tree, nil
}

// addFile is additional function for walking on directory and return back info
func (c *Client) addFile(base string, tree map[string]os.FileInfo, numObjs *int64) error {
	ttree, err := c.ListDir(base)
	if err != nil {
		return err
	}

	for path, info := range ttree {
		npath := filestorage.NormPath(base + path)
		if _, ok := tree[npath]; !ok {
			tree[npath] = info
			(*numObjs)++
			if info.IsDir() {
				if err = c.addFile(npath, tree, numObjs); err != nil {
					return err
				}
			}
			if *numObjs >= c.Limits.MaxNumObjs() {
				return filestorage.ErrLimitExceeded
			}
		}
	}

	return nil
}

// ListDirRec is function that return listing directory with filea info
func (c *Client) ListDirRec(path string) (map[string]os.FileInfo, error) {
	var numObjs int64
	tree := make(map[string]os.FileInfo)
	path = filestorage.NormPath(path)
	if err := c.addFile(path, tree, &numObjs); err != nil {
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
func (c *Client) GetInfo(path string) (os.FileInfo, error) {
	path = filestorage.NormPath(path)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	objInfo, err := c.con.StatObject(ctx, c.bucketName, path, minio.StatObjectOptions{})
	if err != nil {
		tree, err := c.ListDir(path)
		if err != nil || len(tree) == 0 {
			return nil, filestorage.ErrNotFound
		}
		return &FileInfo{
			isDir: true,
			path:  "/",
		}, nil
	}
	_, fileName := filepath.Split(path)

	return &FileInfo{
		isDir:      false,
		path:       fileName,
		ObjectInfo: &objInfo,
	}, nil
}

// IsExist is function that return true if file exists
func (c *Client) IsExist(path string) bool {
	if _, err := c.GetInfo(path); err != nil {
		return false
	}

	return true
}

// IsNotExist is function that return true if file not exists
func (c *Client) IsNotExist(path string) bool {
	return !c.IsExist(path)
}

// ReadFile is function that return the file data
func (c *Client) ReadFile(path string) ([]byte, error) {
	var objData []byte
	var objInfo minio.ObjectInfo
	if info, err := c.GetInfo(path); err != nil {
		return nil, err
	} else if info.IsDir() {
		return nil, filestorage.ErrNotFound
	} else if info.Size() > c.Limits.MaxReadSize() {
		return nil, filestorage.ErrLimitExceeded
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	obj, err := c.con.GetObject(ctx, c.bucketName, path, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("%w: %v", filestorage.ErrReadFailed, err)
	}
	defer obj.Close()
	if objInfo, err = obj.Stat(); err != nil {
		return nil, fmt.Errorf("%w: %v", filestorage.ErrReadFailed, err)
	}
	objData = make([]byte, objInfo.Size)
	if n, err := io.ReadFull(obj, objData); err != nil || n != int(objInfo.Size) {
		return nil, fmt.Errorf("%w: %v", filestorage.ErrReadFailed, err)
	}

	return objData, nil
}

// readFiles is additional function for reading files data by path
func (c *Client) readFiles(files map[string]os.FileInfo, base string) (map[string][]byte, error) {
	var err error
	var numObjs, readSize int64
	tree := make(map[string][]byte)
	for name, info := range files {
		if info.IsDir() {
			continue
		}

		fpath := filestorage.NormPath(base + name)
		if tree[name], err = c.ReadFile(fpath); err != nil {
			return nil, err
		}
		numObjs++
		readSize += info.Size()
		if numObjs >= c.Limits.MaxNumObjs() || readSize >= c.Limits.MaxReadSize() {
			return nil, filestorage.ErrLimitExceeded
		}
	}

	return tree, nil
}

// ReadDir is function that read all files in the directory
func (c *Client) ReadDir(path string) (map[string][]byte, error) {
	path = filestorage.NormPath(path)
	files, err := c.ListDir(path)
	if err != nil {
		return nil, err
	}

	return c.readFiles(files, path)
}

// ReadDirRec is function that recursive read all files in the directory
func (c *Client) ReadDirRec(path string) (map[string][]byte, error) {
	path = filestorage.NormPath(path)
	files, err := c.ListDirRec(path)
	if err != nil {
		return nil, err
	}

	return c.readFiles(files, path)
}

// CreateDir is function for create new directory if not exists
func (c *Client) CreateDir(path string) error {
	path = filestorage.NormPath(path)
	if !c.IsExist(path) {
		return nil
	}

	return filestorage.ErrAlreadyExists
}

// CreateFile is function for create new file if not exists
func (c *Client) CreateFile(path string) error {
	path = filestorage.NormPath(path)
	if !c.IsExist(path) {
		r := bytes.NewReader([]byte{})
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		_, err := c.con.PutObject(ctx, c.bucketName, path[1:], r, 0,
			minio.PutObjectOptions{ContentType: "application/octet-stream"})
		if err != nil {
			return filestorage.ErrCreateFailed
		}
		return nil
	}

	return filestorage.ErrAlreadyExists
}

// WriteFile is function that write (override) data to a file
func (c *Client) WriteFile(path string, data []byte) error {
	path = filestorage.NormPath(path)
	r := bytes.NewReader(data)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, err := c.con.PutObject(ctx, c.bucketName, path[1:], r, r.Size(),
		minio.PutObjectOptions{ContentType: "application/octet-stream"})
	if err != nil {
		return filestorage.ErrWriteFailed
	}

	return nil
}

// AppendFile is function that append data to an exist file
func (c *Client) AppendFile(path string, data []byte) error {
	rdata, err := c.ReadFile(path)
	if err != nil {
		return err
	}

	return c.WriteFile(path, append(rdata, data...))
}

// RemoveDir is function that remove an exist directory
func (c *Client) RemoveDir(path string) error {
	path = filestorage.NormPath(path)
	info, err := c.GetInfo(path)
	if err == nil && info.IsDir() {
		files, err := c.ListDirRec(path)
		if err != nil {
			return filestorage.ErrRemoveFailed
		}
		for fpath, info := range files {
			if !info.IsDir() {
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				if c.con.RemoveObject(ctx, c.bucketName, path+fpath,
					minio.RemoveObjectOptions{}) != nil {
					return filestorage.ErrRemoveFailed
				}
			}
		}
		return nil
	}

	return err
}

// RemoveFile is function that remove an exist file
func (c *Client) RemoveFile(path string) error {
	path = filestorage.NormPath(path)
	info, err := c.GetInfo(path)
	if err == nil && !info.IsDir() {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		if c.con.RemoveObject(ctx, c.bucketName, path,
			minio.RemoveObjectOptions{}) != nil {
			return filestorage.ErrRemoveFailed
		}
		return nil
	}

	return err
}

// Remove is function that remove any exist object
func (c *Client) Remove(path string) error {
	info, err := c.GetInfo(path)
	if err == nil && !info.IsDir() {
		return c.RemoveFile(path)
	} else if err == nil && info.IsDir() {
		return c.RemoveDir(path)
	}

	return filestorage.ErrRemoveFailed
}

// Rename is function that rename any exist object to new
func (c *Client) Rename(src, dst string) error {
	if err := c.CopyFile(src, dst); err != nil {
		return err
	}

	if c.RemoveFile(src) != nil {
		return filestorage.ErrRenameFailed
	}

	return nil
}

// CopyFile is function that copies a file from src to dst
func (c *Client) CopyFile(src, dst string) error {
	isrc, err := c.GetInfo(src)
	if err != nil || isrc.IsDir() {
		return filestorage.ErrNotFound
	}
	if c.IsExist(dst) {
		return filestorage.ErrAlreadyExists
	}

	nsrc := minio.CopySrcOptions{
		Bucket: c.bucketName,
		Object: filestorage.NormPath(src)[1:],
	}
	ndst := minio.CopyDestOptions{
		Bucket: c.bucketName,
		Object: filestorage.NormPath(dst)[1:],
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if _, err = c.con.CopyObject(ctx, ndst, nsrc); err != nil {
		return filestorage.ErrCopyFailed
	}

	return nil
}

// FileInfo is struct with interface os.FileInfo
type FileInfo struct {
	isDir bool
	path  string
	*minio.ObjectInfo
}

// Name is function that return file name
func (f *FileInfo) Name() string {
	return f.path
}

// Size is function that return file size
func (f *FileInfo) Size() int64 {
	if f.ObjectInfo == nil {
		return 0
	}

	return f.ObjectInfo.Size
}

// Mode is function that return file mod structure
func (f *FileInfo) Mode() os.FileMode {
	return 0644
}

// ModTime is function that return last modification time
func (f *FileInfo) ModTime() time.Time {
	if f.ObjectInfo == nil {
		return time.Now()
	}

	return f.ObjectInfo.LastModified
}

// IsDir is function that return true if it's directory
func (f *FileInfo) IsDir() bool {
	return f.isDir
}

// Sys is function that return dummy info
func (f *FileInfo) Sys() interface{} {
	return nil
}
