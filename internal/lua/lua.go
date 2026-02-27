package lua

import lualib "github.com/yuin/gopher-lua"

type Lua struct {
	state *lualib.LState
}

func NewLua() *Lua {
	return &Lua{}
}

func (l *Lua) NewState() {
	l.state = lualib.NewState()
}

func (l *Lua) Close() {
	if l.state != nil {
		l.state.Close()
	}
}
