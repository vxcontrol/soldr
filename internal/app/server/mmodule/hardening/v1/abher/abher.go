package abher

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"

	cache2 "soldr/internal/app/server/mmodule/hardening/cache"
	"soldr/internal/app/server/mmodule/hardening/v1/abher/types"
	"soldr/internal/vxproto"
)

type ABH struct {
	cache *cache2.Cache
}

const abhRefreshInterval = time.Second * 60

func NewABH(ctx context.Context, store interface{}, basePath string) (*ABH, error) {
	cacheInitParams := cache2.ConnectorParams{
		FileFetcher: getABHListFromFile(basePath, NewABHList()),
		DBFetcher:   getFnABHListFromDB,
	}
	return NewABHWithCacheParams(ctx, store, &cacheInitParams)
}

func NewABHWithCacheParams(ctx context.Context, store interface{}, cacheParams *cache2.ConnectorParams) (*ABH, error) {
	c, err := cache2.NewCache(ctx, store, cacheParams, &cache2.Config{
		RefreshInterval: abhRefreshInterval,
		Logger: logrus.WithFields(logrus.Fields{
			"component": "abh_cache",
			"module":    "main",
		}),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create the cache object: %w", err)
	}
	return &ABH{
		cache: c,
	}, nil
}

func (a *ABH) GetABH(t vxproto.AgentType, id *types.AgentBinaryID) ([]byte, error) {
	abhList, ok := a.cache.Dump().(*ABHList)
	if !ok {
		return nil, fmt.Errorf("failed to get the ABH list from cache: the stored object is not of the type *abhList")
	}
	var abi string
	switch t {
	case vxproto.VXAgent:
		abi = id.String()
	case vxproto.Browser, vxproto.External:
		abi = id.Version
	default:
		return nil, fmt.Errorf("unknown connection type: %d", t)
	}
	abh, err := abhList.Get(t, abi)
	if err != nil {
		return nil, err
	}
	return abh, nil
}

func GetABH(cache *cache2.Cache, t vxproto.AgentType, id *types.AgentBinaryID) ([]byte, error) {
	abhList, ok := cache.Dump().(*ABHList)
	if !ok {
		return nil, fmt.Errorf("failed to get the ABH list from cache: the stored object is not of the type *abhList")
	}
	var abi string
	switch t {
	case vxproto.VXAgent:
		abi = id.String()
	case vxproto.Browser, vxproto.External:
		abi = id.Version
	default:
		return nil, fmt.Errorf("unknown connection type: %d", t)
	}
	abh, err := abhList.Get(t, abi)
	if err != nil {
		return nil, err
	}
	return abh, nil
}

func (a *ABH) GetABHWithSocket(t vxproto.AgentType, socket vxproto.IAgentSocket) ([]byte, error) {
	aid, err := getAgentBinaryIDFromSocket(socket)
	if err != nil {
		return nil, fmt.Errorf("failed to get the agent binary ID from socket: %w", err)
	}
	abh, err := a.GetABH(t, aid)
	if err != nil {
		return nil, err
	}
	return abh, nil
}

func getAgentBinaryIDFromSocket(socket vxproto.IAgentSocket) (*types.AgentBinaryID, error) {
	if socket == nil {
		return nil, fmt.Errorf("passed socket object is nil")
	}
	info := socket.GetPublicInfo()
	if info == nil {
		return nil, fmt.Errorf("socket public info is nil")
	}
	var os, arch string
	if info.Info != nil {
		os = info.Info.GetOs().GetType()
		arch = info.Info.GetOs().GetArch()
	}
	return &types.AgentBinaryID{
		Version: info.Ver,
		OS:      os,
		Arch:    arch,
	}, nil
}
