package store

import (
	"soldr/internal/hardening/luavm/store/simple"
	"soldr/internal/hardening/luavm/store/types"
)

type Store interface {
	GetLTAC(key []byte) (*types.LTAC, error)
	GetSBH(key []byte) ([]byte, error)
	StoreInitConnectionPack(p *types.InitConnectionPack, key []byte) error
	Reset() error
}

func NewStore(c *types.Config) (Store, error) {
	return simple.NewStore(c)
}
