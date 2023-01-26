package sbher

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"time"

	"github.com/sirupsen/logrus"

	cache2 "soldr/pkg/app/server/mmodule/hardening/cache"
	"soldr/pkg/filestorage"
)

type SBH struct {
	cache *cache2.Cache
}

const (
	sbhRefreshInterval = time.Second * 60
)

func NewSBH(ctx context.Context, store interface{}, basePath string, logger *logrus.Entry) (*SBH, error) {
	connectorParams := &cache2.ConnectorParams{
		FileFetcher: getFileFetcher(basePath),
	}
	return NewSBHWithConnectorParams(ctx, store, connectorParams, logger)
}

func NewSBHWithConnectorParams(
	ctx context.Context,
	store interface{},
	connectorParams *cache2.ConnectorParams,
	logger *logrus.Entry,
) (*SBH, error) {
	c, err := cache2.NewCache(ctx, store, connectorParams, &cache2.Config{
		RefreshInterval: sbhRefreshInterval,
		Logger: logger.WithFields(logrus.Fields{
			"component": "sbh_cache",
			"module":    "main",
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create the cache object: %w", err)
	}
	return &SBH{
		cache: c,
	}, nil
}

func (s *SBH) Get(ctx context.Context, version string) ([]byte, error) {
	data := s.cache.Dump()
	sbhFile, ok := data.(SBHFileData)
	if !ok {
		return nil, fmt.Errorf("an unexpected type of the fetched data (%T)", data)
	}
	const oldVersionName = "old"
	sbh, ok := sbhFile[version]
	if !ok {
		sbh, ok = sbhFile[oldVersionName]
		if !ok {
			return nil, fmt.Errorf("the requested version (%s or \"%s\") not found", version, oldVersionName)
		}
	}
	return sbh, nil
}

type sbhFile struct {
	Data SBHFileData `json:"v1"`
}

type SBHFileData map[string][]byte

func getFileFetcher(basePath string) cache2.FetchDataFromFile {
	const sbhFileName = "sbh.json"
	return func(ctx context.Context, connector filestorage.Reader) (interface{}, error) {
		sbhFilePath := path.Join(basePath, "lic", sbhFileName)
		data, err := connector.ReadFile(sbhFilePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read the file %s: %w", sbhFilePath, err)
		}
		var sbhs sbhFile
		if err := json.Unmarshal(data, &sbhs); err != nil {
			return nil, fmt.Errorf("failed to unmarshal the file %s: %w", sbhFilePath, err)
		}
		return sbhs.Data, nil
	}
}
