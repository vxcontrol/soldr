package cache

import (
	"fmt"
	"sync"
)

type lruTrackerItem struct {
	Key     *Key
	Counter int64
}

type lruTracker struct {
	store    []*lruTrackerItem
	storeMux *sync.Mutex
}

func newLRUTracker(size int) (*lruTracker, error) {
	if size <= 0 {
		return nil, fmt.Errorf("LRU tracker initialization failed: an invalid value of the size parameter passed: %d", size)
	}
	return &lruTracker{
		store:    make([]*lruTrackerItem, size),
		storeMux: &sync.Mutex{},
	}, nil
}

func (t *lruTracker) Use(key *Key) {
	idx := t.findIdxByKey(key)
	if idx == -1 {
		return
	}
	t.store[idx].Counter++
}

func (t *lruTracker) SetOrReplace(key *Key) *Key {
	var minCounter int64
	var minCounterIdx int
	for i, item := range t.store {
		if item == nil {
			minCounterIdx = i
			break
		}
		if item.Counter < minCounter {
			minCounter = item.Counter
			minCounterIdx = i
		}
	}
	itemFound := t.store[minCounterIdx]
	t.store[minCounterIdx] = &lruTrackerItem{
		Key:     key,
		Counter: 0,
	}
	if itemFound == nil {
		return nil
	}
	return itemFound.Key
}

func (t *lruTracker) Remove(key *Key) {
	idx := t.findIdxByKey(key)
	if idx == -1 {
		return
	}
	t.store[idx] = nil
}

func (t *lruTracker) findIdxByKey(key *Key) int {
	for i, item := range t.store {
		if item == nil || *item.Key != *key {
			continue
		}
		return i
	}
	return -1
}
