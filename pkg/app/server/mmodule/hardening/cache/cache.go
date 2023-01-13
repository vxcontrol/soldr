package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

type Cache struct {
	ctx       context.Context
	connector *Connector

	data    interface{}
	dataMux *sync.RWMutex

	watcherFetchRequest  chan struct{}
	watcherFetchResponse chan error

	logger *logrus.Entry
}

type Config struct {
	RefreshInterval time.Duration
	Logger          *logrus.Entry
}

func NewCache(ctx context.Context, store interface{}, connectorParams *ConnectorParams, conf *Config) (*Cache, error) {
	conn, err := NewConnector(store, connectorParams)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize a connector: %w", err)
	}
	return initCache(ctx, conn, conf)
}

func initCache(ctx context.Context, connector *Connector, conf *Config) (*Cache, error) {
	data, err := connector.Fetch(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data via the passed connector: %w", err)
	}
	cache := &Cache{
		ctx:       ctx,
		connector: connector,

		data:    data,
		dataMux: &sync.RWMutex{},

		watcherFetchRequest:  make(chan struct{}),
		watcherFetchResponse: make(chan error),

		logger: conf.Logger,
	}
	go cache.watchData(ctx, conf.RefreshInterval)
	return cache, nil
}

func (c *Cache) Dump() interface{} {
	c.dataMux.RLock()
	defer c.dataMux.RUnlock()
	return c.data
}

func (c *Cache) Fetch() (interface{}, error) {
	c.watcherFetchRequest <- struct{}{}
	<-c.watcherFetchResponse
	return c.Dump(), nil
}

func (c *Cache) watchData(ctx context.Context, tickInterval time.Duration) {
	ticker := time.NewTicker(tickInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data, err := c.connector.Fetch(ctx)
			if err != nil {
				c.logger.WithError(err).Error("failed to fetch file")
				continue
			}
			c.storeData(data)
		case <-c.watcherFetchRequest:
			data, err := c.connector.Fetch(ctx)
			if err != nil {
				c.watcherFetchResponse <- err
				c.logger.WithError(err).Error("failed to fetch file")
				continue
			}
			c.storeData(data)
		}
	}
}

func (c *Cache) storeData(data interface{}) {
	c.dataMux.Lock()
	defer c.dataMux.Unlock()
	c.data = data
}
