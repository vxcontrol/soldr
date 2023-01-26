package controller

import (
	"soldr/pkg/filestorage/fs"
	"soldr/pkg/filestorage/s3"
)

// sFiles is universal container for modules files loader
type sFiles struct {
	flt tFilesLoaderType
	IFilesLoader
}

// NewFilesFromS3 is function which constructed Files loader object
func NewFilesFromS3(connParams *s3.Config) (IFilesLoader, error) {
	sc, err := s3.New(connParams)
	if err != nil {
		return nil, generateDriverInitErrMsg(driverTypeS3, err)
	}
	return &sFiles{
		flt:          eS3FilesLoader,
		IFilesLoader: &filesLoaderS3{sc: sc},
	}, nil
}

// NewFilesFromFS is function which constructed Files loader object
func NewFilesFromFS(path string) (IFilesLoader, error) {
	sc, err := fs.New()
	if err != nil {
		return nil, generateDriverInitErrMsg(driverTypeFS, err)
	}
	return &sFiles{
		flt:          eFSFilesLoader,
		IFilesLoader: &filesLoaderFS{path: path, sc: sc},
	}, nil
}
