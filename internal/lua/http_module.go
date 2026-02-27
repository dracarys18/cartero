package lua

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	lua "github.com/yuin/gopher-lua"
)

type HTTPModule struct {
	client *http.Client
}

func NewHTTPModule() *HTTPModule {
	return &HTTPModule{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (h *HTTPModule) Name() string {
	return "http"
}

func (h *HTTPModule) Register(L *lua.LState) error {
	httpTable := L.NewTable()

	L.SetField(httpTable, "get", L.NewFunction(h.httpGet))
	L.SetField(httpTable, "post", L.NewFunction(h.httpPost))

	L.SetGlobal("http", httpTable)
	return nil
}

func (h *HTTPModule) httpGet(L *lua.LState) int {
	url := L.CheckString(1)
	options := L.OptTable(2, L.NewTable())

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to create request: %s", err.Error())))
		return 2
	}

	h.applyHeaders(L, req, options)

	resp, err := h.client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("request failed: %s", err.Error())))
		return 2
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to read response: %s", err.Error())))
		return 2
	}

	result := h.createResponseTable(L, resp, body)
	L.Push(result)
	return 1
}

func (h *HTTPModule) httpPost(L *lua.LState) int {
	url := L.CheckString(1)
	body := L.OptString(2, "")
	options := L.OptTable(3, L.NewTable())

	req, err := http.NewRequest("POST", url, bytes.NewBufferString(body))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to create request: %s", err.Error())))
		return 2
	}

	h.applyHeaders(L, req, options)

	if req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := h.client.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("request failed: %s", err.Error())))
		return 2
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to read response: %s", err.Error())))
		return 2
	}

	result := h.createResponseTable(L, resp, respBody)
	L.Push(result)
	return 1
}

func (h *HTTPModule) applyHeaders(L *lua.LState, req *http.Request, options *lua.LTable) {
	headers := options.RawGetString("headers")
	if headersTable, ok := headers.(*lua.LTable); ok {
		headersTable.ForEach(func(key, value lua.LValue) {
			if keyStr, ok := key.(lua.LString); ok {
				if valStr, ok := value.(lua.LString); ok {
					req.Header.Set(string(keyStr), string(valStr))
				}
			}
		})
	}
}

func (h *HTTPModule) createResponseTable(L *lua.LState, resp *http.Response, body []byte) *lua.LTable {
	result := L.NewTable()

	result.RawSetString("status", lua.LNumber(resp.StatusCode))
	result.RawSetString("body", lua.LString(string(body)))

	headers := L.NewTable()
	for key, values := range resp.Header {
		if len(values) > 0 {
			headers.RawSetString(key, lua.LString(values[0]))
		}
	}
	result.RawSetString("headers", headers)

	return result
}
