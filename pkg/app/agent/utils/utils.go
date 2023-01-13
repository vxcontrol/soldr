package utils

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type CopyFileOptions struct {
	Mode os.FileMode
	Flag int
}

func CopyFile(src string, dst string, opts ...*CopyFileOptions) (err error) {
	opt, err := getCopyFileOpt(opts)
	if err != nil {
		return
	}
	cleanDstDir, err := checkSrcDstForCopy(src, dst)
	if err != nil {
		return
	}
	defer func() {
		err = cleanDstDir(err)
	}()

	if err = copyContents(src, dst, opt.Mode); err != nil {
		return
	}
	return
}

type MoveFileOptions struct {
	Mode os.FileMode
}

func MoveFile(src string, dst string, opts ...*MoveFileOptions) (err error) {
	opt, err := getMoveFileOpt(opts)
	if err != nil {
		return
	}

	cleanDstDir, err := checkSrcDstForCopy(src, dst)
	if err != nil {
		return
	}
	defer func() {
		err = cleanDstDir(err)
	}()

	if err = moveFile(src, dst, opt); err != nil {
		return
	}
	return
}

func checkSrcDstForCopy(src string, dst string) (func(error) error, error) {
	if err := checkCopySrcAndDst(src, dst); err != nil {
		return nil, err
	}
	if err := checkFileExists(src); err != nil {
		return nil, err
	}

	cleanDstDir, err := checkDstDir(dst)
	if err != nil {
		return nil, err
	}
	return cleanDstDir, nil
}

func getCopyFileOpt(opts []*CopyFileOptions) (*CopyFileOptions, error) {
	defaultOpt := getDefaultCopyFileOptions()
	if len(opts) == 0 {
		return defaultOpt, nil
	}
	if len(opts) > 1 {
		return nil, fmt.Errorf("only one option can be passed to CopyFile, got %d options", len(opts))
	}
	return setMissingCopyOptsFields(opts[0]), nil
}

func getMoveFileOpt(opts []*MoveFileOptions) (*MoveFileOptions, error) {
	if len(opts) == 0 {
		return getDefaultMoveFileOptions(), nil
	}
	if len(opts) > 1 {
		return nil, fmt.Errorf("only one option can be passed to MoveFile, got %d options", len(opts))
	}
	return setMissingMoveOptsFields(opts[0]), nil
}

func getDefaultCopyFileOptions() *CopyFileOptions {
	return &CopyFileOptions{
		Mode: 0,
		Flag: os.O_RDWR | os.O_CREATE | os.O_TRUNC,
	}
}

func getDefaultMoveFileOptions() *MoveFileOptions {
	return &MoveFileOptions{
		Mode: 0,
	}
}

func setMissingCopyOptsFields(opts *CopyFileOptions) *CopyFileOptions {
	defaultOpts := getDefaultCopyFileOptions()
	if opts == nil {
		return defaultOpts
	}
	if opts.Mode == 0 {
		opts.Mode = defaultOpts.Mode
	}
	if opts.Flag == 0 {
		opts.Flag = defaultOpts.Flag
	}
	return opts
}

func setMissingMoveOptsFields(opts *MoveFileOptions) *MoveFileOptions {
	defaultOpts := getDefaultMoveFileOptions()
	if opts == nil {
		return defaultOpts
	}
	if opts.Mode == 0 {
		opts.Mode = defaultOpts.Mode
	}
	return opts
}

func checkCopySrcAndDst(src string, dst string) error {
	if len(src) == 0 {
		return fmt.Errorf("source file path cannot be empty")
	}
	if len(dst) == 0 {
		return fmt.Errorf("destination file path cannot be empty")
	}
	if src == dst {
		return fmt.Errorf("source file path and destination file path are equal: %s", src)
	}
	return nil
}

func checkFileExists(path string) error {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("file %s does not exist", path)
		}
		return fmt.Errorf("failed to get the file %s metadata: %w", path, err)
	}
	return nil
}

func copyContents(src string, dst string, mode os.FileMode) error {
	srcF, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open the source file %s: %w", src, err)
	}
	defer func() {
		_ = srcF.Close()
	}()
	fMode := mode
	if fMode == 0 {
		fMode, err = getFileMode(srcF)
		if err != nil {
			return fmt.Errorf("failed to get the source file %s mode: %w", src, err)
		}
	}
	dstF, err := os.OpenFile(dst, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fMode)
	if err != nil {
		return fmt.Errorf("failed to open the destination file %s: %w", dst, err)
	}
	defer func() {
		_ = dstF.Close()
	}()
	if _, err := io.Copy(dstF, srcF); err != nil {
		return fmt.Errorf("failed to copy the contents of the source file %s into the destination file %s: %w", src, dst, err)
	}
	return nil
}

func moveFile(src string, dst string, opt *MoveFileOptions) error {
	if err := os.Rename(src, dst); err != nil {
		return fmt.Errorf("failed to replace the file %s with the file %s: %w", dst, src, err)
	}
	if opt.Mode != 0 {
		if err := os.Chmod(dst, opt.Mode); err != nil {
			return fmt.Errorf("failed to change the destination file permissions to %v: %w", opt.Mode, err)
		}
	}
	return nil
}

func getFileMode(f *os.File) (os.FileMode, error) {
	fInfo, err := f.Stat()
	if err != nil {
		return 0, fmt.Errorf("failed to get the file info: %w", err)
	}
	return fInfo.Mode(), nil
}

func checkDstDir(dst string) (func(e error) error, error) {
	dstDir := filepath.Dir(dst)
	_, err := os.Stat(dstDir)
	if err == nil {
		// Dst directory was not created, so the teardown function does nothing
		return func(e error) error {
			return e
		}, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("failed to get the destination file directory %s metadata: %w", dstDir, err)
	}
	dirsToCreate, err := getDirsToCreate(dstDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get the list of dirs to create: %w", err)
	}
	if err = os.MkdirAll(dstDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create the destination file directory %s: %w", dstDir, err)
	}
	return func(e error) error {
		if e == nil {
			return nil
		}
		for _, d := range dirsToCreate {
			if dirRmErr := os.Remove(d); dirRmErr != nil {
				return fmt.Errorf(
					"while removing the created directories %v and processing %s, "+
						"the following error has occurred: %s; "+
						"the underlying error is: %w",
					dirsToCreate,
					d,
					dirRmErr.Error(),
					err,
				)
			}
		}
		return e
	}, nil
}

func getDirsToCreate(dstDir string) ([]string, error) {
	dirsToCreate := make([]string, 0)
	currDir := dstDir
	for {
		_, err := os.Stat(currDir)
		if err == nil {
			break
		}
		if !errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("failed to get the information on the directory %s: %w", currDir, err)
		}
		dirsToCreate = append(dirsToCreate, currDir)
		currDirParent := filepath.Dir(currDir)
		if currDirParent == currDir {
			break
		}
		currDir = currDirParent
	}
	return dirsToCreate, nil
}

func RemoveIfExists(path string) (bool, error) {
	if _, err := os.Stat(path); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		return false, fmt.Errorf("failed to get the information on %s: %w", path, err)
	}
	if err := os.Remove(path); err != nil {
		return false, fmt.Errorf("failed to delete %s: %w", path, err)
	}
	return true, nil
}

type PathResolver struct {
	baseDir string
}

func NewPathResolver(baseDir string) (*PathResolver, error) {
	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to get the absolute path to %s: %w", baseDir, err)
	}
	return &PathResolver{
		baseDir: absBaseDir,
	}, nil
}

func (r *PathResolver) Resolve(path string) string {
	return filepath.Join(r.baseDir, filepath.Join("/", path))
}

func (r *PathResolver) GetBaseDir() string {
	return r.baseDir
}
