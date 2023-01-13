package cache

import (
	"context"
	"fmt"

	"github.com/jinzhu/gorm"

	"soldr/pkg/storage"
)

type (
	fetchData         func(ctx context.Context, connector interface{}) (interface{}, error)
	FetchDataFromDB   func(ctx context.Context, connector *gorm.DB) (interface{}, error)
	FetchDataFromFile func(ctx context.Context, connector storage.IFileReader) (interface{}, error)
)

type ConnectorParams struct {
	DBFetcher   FetchDataFromDB
	FileFetcher FetchDataFromFile
}

type Connector struct {
	store interface{}
	fetch fetchData
}

func NewConnector(store interface{}, params *ConnectorParams) (*Connector, error) {
	if store == nil {
		return nil, fmt.Errorf("passed store is nil")
	}
	if params == nil {
		return nil, fmt.Errorf("passed params object is nil")
	}
	fetch, err := chooseFetcher(store, params)
	if err != nil {
		return nil, err
	}
	return &Connector{
		store: store,
		fetch: fetch,
	}, nil
}

func chooseFetcher(connector interface{}, initParams *ConnectorParams) (fetchData, error) {
	var fetch fetchData
	var err error
	switch connector.(type) {
	case *gorm.DB:
		fetch, err = getDBFetcher(initParams.DBFetcher)
	case storage.IFileReader:
		fetch, err = getFileFetcher(initParams.FileFetcher)
	default:
		return nil, fmt.Errorf("a store of an unknown type passed")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get the fetcher function: %w", err)
	}
	if fetch == nil {
		return nil, fmt.Errorf("no fetcher function defined for the configured store type")
	}
	return fetch, nil
}

func (c *Connector) Fetch(ctx context.Context) (interface{}, error) {
	return c.fetch(ctx, c.store)
}

func getDBFetcher(fetcher FetchDataFromDB) (fetchData, error) {
	if fetcher == nil {
		return nil, fmt.Errorf("DB fetcher function has not been implemented")
	}
	return func(ctx context.Context, connector interface{}) (interface{}, error) {
		gdbc, ok := connector.(*gorm.DB)
		if !ok {
			return nil, fmt.Errorf("passed connector in not of the type *grom.DB")
		}
		return fetcher(ctx, gdbc)
	}, nil
}

func getFileFetcher(fetcher FetchDataFromFile) (fetchData, error) {
	if fetcher == nil {
		return nil, fmt.Errorf("fileStore fetcher function has not been implemented")
	}
	return func(ctx context.Context, connector interface{}) (interface{}, error) {
		fileStore, ok := connector.(storage.IStorage)
		if !ok {
			return nil, fmt.Errorf("passed connector in not of the type IStorage")
		}
		if fileStore == nil {
			return nil, fmt.Errorf("passed connector object is nil")
		}
		return fetcher(ctx, fileStore)
	}, nil
}
