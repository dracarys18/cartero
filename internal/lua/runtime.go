package lua

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

type Runtime struct {
	state      *lua.LState
	secureMode bool
}

type RuntimeOption func(*Runtime)

func WithLoader(loader Loader) RuntimeOption {
	return func(r *Runtime) {
		if loader != nil {
			SetupRequire(r.state, loader)
		}
	}
}

func WithSecureMode(secure bool) RuntimeOption {
	return func(r *Runtime) {
		r.secureMode = secure
	}
}

func NewRuntime(options ...RuntimeOption) *Runtime {
	L := lua.NewState()

	runtime := &Runtime{
		state:      L,
		secureMode: true,
	}

	for _, opt := range options {
		opt(runtime)
	}

	if runtime.secureMode {
		runtime.setupSecureState()
	}

	return runtime
}

func (r *Runtime) State() *lua.LState {
	return r.state
}

func (r *Runtime) setupSecureState() {
	r.state.SetGlobal("os", lua.LNil)
	r.state.SetGlobal("io", lua.LNil)
	r.state.SetGlobal("debug", lua.LNil)
	r.state.SetGlobal("dofile", lua.LNil)
	r.state.SetGlobal("loadfile", lua.LNil)
}

func (r *Runtime) LoadScript(scriptContent string) error {
	if err := r.state.DoString(scriptContent); err != nil {
		return fmt.Errorf("failed to load script: %w", err)
	}
	return nil
}

func (r *Runtime) Execute(functionName string, args ...interface{}) ([]interface{}, error) {
	fn := r.state.GetGlobal(functionName)
	if fn == lua.LNil {
		return nil, fmt.Errorf("function %s not found", functionName)
	}

	luaFn, ok := fn.(*lua.LFunction)
	if !ok {
		return nil, fmt.Errorf("%s is not a function", functionName)
	}

	luaArgs := make([]lua.LValue, len(args))
	for i, arg := range args {
		luaArgs[i] = ToLuaValue(r.state, arg)
	}

	r.state.Push(luaFn)
	for _, arg := range luaArgs {
		r.state.Push(arg)
	}

	if err := r.state.PCall(len(luaArgs), lua.MultRet, nil); err != nil {
		return nil, fmt.Errorf("lua execution error: %w", err)
	}

	numResults := r.state.GetTop()
	results := make([]interface{}, numResults)
	for i := 1; i <= numResults; i++ {
		results[i-1] = ToGoValue(r.state.Get(i))
	}

	r.state.SetTop(0)

	return results, nil
}

func (r *Runtime) Close() error {
	if r.state != nil {
		r.state.Close()
	}
	return nil
}
