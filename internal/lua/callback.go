package lua

import (
	"context"
	"fmt"
	"sync"

	"github.com/vxcontrol/golua/lua"
	"github.com/vxcontrol/luar"
)

type luaCallback struct {
	refTr  int
	refCb  int
	lsm    *lua.State
	lst    *lua.State
	mx     *sync.Mutex
	closed bool
}

func newLuaCallback(L *lua.State) *luaCallback {
	if !L.IsFunction(-1) {
		if !L.GetMetaField(-1, "__call") {
			L.Pop(1)
			return nil
		}
		// There leave the __call metamethod on stack.
		L.Remove(-2)
	}

	lst := L.NewThread()
	refTr := L.Ref(lua.LUA_REGISTRYINDEX)
	refCb := L.Ref(lua.LUA_REGISTRYINDEX)

	return &luaCallback{lsm: L, lst: lst, refCb: refCb, refTr: refTr, mx: &sync.Mutex{}}
}

func (lc *luaCallback) Call(_ context.Context, results interface{}, args ...interface{}) error {
	lc.mx.Lock()
	defer lc.mx.Unlock()

	if lc.closed {
		return fmt.Errorf("callback is already closed")
	}

	lc.lst.Lock()
	defer lc.lst.Unlock()

	top := lc.lst.GetTop()

	// Push the callable value.
	lc.lst.RawGeti(lua.LUA_REGISTRYINDEX, lc.refCb)

	// Push the args.
	for _, arg := range args {
		luar.GoToLua(lc.lst, arg)
	}

	if err := lc.lst.Call(len(args), 1); err != nil {
		lc.lst.SetTop(top)
		return err
	}

	if err := luar.LuaToGo(lc.lst, -1, results); err != nil {
		lc.lst.SetTop(top)
		return err
	}

	lc.lst.SetTop(top)
	return nil
}

func (lc *luaCallback) Close() {
	lc.mx.Lock()
	defer lc.mx.Unlock()

	if !lc.closed {
		lc.lsm.Unref(lua.LUA_REGISTRYINDEX, lc.refCb)
		lc.lsm.Unref(lua.LUA_REGISTRYINDEX, lc.refTr)
		lc.closed = true
	}
}
