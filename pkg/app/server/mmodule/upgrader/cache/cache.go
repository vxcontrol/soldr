package cache

import (
	"context"
	"fmt"
	"path"
	"sync"
	"time"

	"soldr/pkg/filestorage"
	"soldr/pkg/vxproto"
)

type Key struct {
	version string
	os      string
	arch    string
	rev     string
}

func NewKey(version string, agentInfo *vxproto.AgentInfo) *Key {
	info := agentInfo.Info
	os := info.GetOs()

	return &Key{
		version: version,
		os:      os.GetType(),
		arch:    os.GetArch(),
		rev:     info.GetRevision(),
	}
}

func (k *Key) GetS3UpgraderPath() string {
	const upgraderDir = "vxagent"
	upgraderFileName := "vxagent"
	if k.rev != "" {
		upgraderFileName = k.rev
	}
	if k.os == "windows" {
		upgraderFileName += ".exe"
	}
	return path.Join(upgraderDir, k.version, k.os, k.arch, upgraderFileName)
}

func (k *Key) GetS3UpgraderThumbprintPath() string {
	return k.GetS3UpgraderPath() + ".thumbprint"
}

type Item struct {
	UpgraderThumbprint []byte
	Upgrader           []byte
}

type Cache struct {
	store       filestorage.Storage
	cache       map[Key]*Item
	cacheMux    *sync.Mutex
	tracker     *lruTracker
	watchersWG  *sync.WaitGroup
	ctx         context.Context
	cancelCtxFn func()
}

const (
	cacheSize = 6
)

func NewCache(store filestorage.Storage) (*Cache, error) {
	tracker, err := newLRUTracker(cacheSize)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Cache{
		store:       store,
		cache:       make(map[Key]*Item),
		cacheMux:    &sync.Mutex{},
		tracker:     tracker,
		watchersWG:  &sync.WaitGroup{},
		ctx:         ctx,
		cancelCtxFn: cancel,
	}, nil
}

func (c *Cache) Get(key *Key) (*Item, error) {
	c.cacheMux.Lock()
	defer c.cacheMux.Unlock()

	item, ok := c.cache[*key]
	if ok {
		c.tracker.Use(key)
		return item, nil
	}
	var err error
	item, err = c.fetch(key)
	if err != nil {
		return nil, err
	}
	c.put(key, item)
	return item, nil
}

func (c *Cache) Close() {
	c.cancelCtxFn()
	c.watchersWG.Wait()
}

func (c *Cache) fetch(key *Key) (*Item, error) {
	item := &Item{}
	var err error
	item.UpgraderThumbprint, err = c.getFileFromStore(key.GetS3UpgraderThumbprintPath())
	if err != nil {
		return nil, err
	}
	item.Upgrader, err = c.getFileFromStore(key.GetS3UpgraderPath())
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (c *Cache) getFileFromStore(path string) ([]byte, error) {
	contents, err := c.store.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read the file %s from store: %w", path, err)
	}
	return contents, nil
}

func (c *Cache) put(key *Key, item *Item) {
	replaceKey := c.tracker.SetOrReplace(key)
	if replaceKey != nil {
		delete(c.cache, *replaceKey)
	}
	c.cache[*key] = item
	c.watchersWG.Add(1)
	go c.watchCacheEntry(c.ctx, key)
}

const cacheEntryExpirationTime = time.Minute * 5

func (c *Cache) watchCacheEntry(cacheCtx context.Context, key *Key) {
	defer c.watchersWG.Done()

	expirationCtx, expirationCtxCancel := context.WithTimeout(context.Background(), cacheEntryExpirationTime)
	defer expirationCtxCancel()
	select {
	case <-cacheCtx.Done():
		return
	case <-expirationCtx.Done():
	}

	c.cacheMux.Lock()
	defer c.cacheMux.Unlock()

	delete(c.cache, *key)
}
