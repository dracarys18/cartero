package lua

import (
	"encoding/json"
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

type JSONModule struct{}

func NewJSONModule() *JSONModule {
	return &JSONModule{}
}

func (j *JSONModule) Name() string {
	return "json"
}

func (j *JSONModule) Register(L *lua.LState) error {
	jsonTable := L.NewTable()

	L.SetField(jsonTable, "encode", L.NewFunction(j.jsonEncode))
	L.SetField(jsonTable, "decode", L.NewFunction(j.jsonDecode))

	L.SetGlobal("json", jsonTable)
	return nil
}

func (j *JSONModule) jsonEncode(L *lua.LState) int {
	value := L.CheckAny(1)

	goValue := ToGoValue(value)

	jsonBytes, err := json.Marshal(goValue)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to encode JSON: %s", err.Error())))
		return 2
	}

	L.Push(lua.LString(string(jsonBytes)))
	return 1
}

func (j *JSONModule) jsonDecode(L *lua.LState) int {
	jsonStr := L.CheckString(1)

	var result interface{}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to decode JSON: %s", err.Error())))
		return 2
	}

	luaValue := ToLuaValue(L, result)
	L.Push(luaValue)
	return 1
}
