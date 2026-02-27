package lua

import lua "github.com/yuin/gopher-lua"

type Module interface {
	Name() string
	Register(L *lua.LState) error
}

func RegisterModule(L *lua.LState, module Module) error {
	return module.Register(L)
}
