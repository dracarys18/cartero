package lua

import (
	"log/slog"

	lua "github.com/yuin/gopher-lua"
)

type LogModule struct {
	logger *slog.Logger
}

func NewLogModule(logger *slog.Logger) *LogModule {
	return &LogModule{
		logger: logger,
	}
}

func (l *LogModule) Name() string {
	return "log"
}

func (l *LogModule) Register(L *lua.LState) error {
	logTable := L.NewTable()

	L.SetField(logTable, "info", L.NewFunction(l.logInfo))
	L.SetField(logTable, "error", L.NewFunction(l.logError))
	L.SetField(logTable, "debug", L.NewFunction(l.logDebug))

	L.SetGlobal("log", logTable)
	return nil
}

func (l *LogModule) logInfo(L *lua.LState) int {
	message := L.CheckString(1)
	if l.logger != nil {
		l.logger.Info(message)
	}
	return 0
}

func (l *LogModule) logError(L *lua.LState) int {
	message := L.CheckString(1)
	if l.logger != nil {
		l.logger.Error(message)
	}
	return 0
}

func (l *LogModule) logDebug(L *lua.LState) int {
	message := L.CheckString(1)
	if l.logger != nil {
		l.logger.Debug(message)
	}
	return 0
}
