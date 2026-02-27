package lua

import (
	"fmt"

	lua "github.com/yuin/gopher-lua"
)

func ToLuaValue(L *lua.LState, value interface{}) lua.LValue {
	switch v := value.(type) {
	case nil:
		return lua.LNil
	case bool:
		return lua.LBool(v)
	case int:
		return lua.LNumber(v)
	case int64:
		return lua.LNumber(v)
	case float64:
		return lua.LNumber(v)
	case string:
		return lua.LString(v)
	case map[string]interface{}:
		table := L.NewTable()
		for key, val := range v {
			table.RawSetString(key, ToLuaValue(L, val))
		}
		return table
	case []interface{}:
		table := L.NewTable()
		for i, val := range v {
			table.RawSetInt(i+1, ToLuaValue(L, val))
		}
		return table
	default:
		return lua.LString(fmt.Sprintf("%v", v))
	}
}

func ToGoValue(lv lua.LValue) interface{} {
	switch v := lv.(type) {
	case *lua.LNilType:
		return nil
	case lua.LBool:
		return bool(v)
	case lua.LNumber:
		return float64(v)
	case lua.LString:
		return string(v)
	case *lua.LTable:
		maxn := v.MaxN()
		if maxn > 0 {
			slice := make([]interface{}, 0, maxn)
			for i := 1; i <= maxn; i++ {
				slice = append(slice, ToGoValue(v.RawGetInt(i)))
			}
			return slice
		}

		m := make(map[string]interface{})
		v.ForEach(func(key, value lua.LValue) {
			keyStr, ok := key.(lua.LString)
			if ok {
				m[string(keyStr)] = ToGoValue(value)
			}
		})
		return m
	default:
		return nil
	}
}

func ToGoSlice(lv lua.LValue) ([]interface{}, error) {
	table, ok := lv.(*lua.LTable)
	if !ok {
		return nil, fmt.Errorf("expected table, got %s", lv.Type())
	}

	maxn := table.MaxN()
	if maxn == 0 {
		return []interface{}{}, nil
	}

	result := make([]interface{}, 0, maxn)
	for i := 1; i <= maxn; i++ {
		result = append(result, ToGoValue(table.RawGetInt(i)))
	}

	return result, nil
}
