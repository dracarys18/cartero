package lua

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	lua "github.com/yuin/gopher-lua"
)

type Loader interface {
	Load(identifier string) (string, error)
}

type EmbeddedLoader struct {
	fs       embed.FS
	basePath string
}

func NewEmbeddedLoader(fs embed.FS, basePath string) *EmbeddedLoader {
	return &EmbeddedLoader{
		fs:       fs,
		basePath: basePath,
	}
}

func (e *EmbeddedLoader) Load(identifier string) (string, error) {
	path := filepath.Join(e.basePath, identifier)
	if !strings.HasSuffix(path, ".lua") {
		path = path + ".lua"
	}

	data, err := e.fs.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to load embedded script %s: %w", identifier, err)
	}

	return string(data), nil
}

type FilesystemLoader struct {
	basePath string
}

func NewFilesystemLoader(basePath string) *FilesystemLoader {
	return &FilesystemLoader{
		basePath: basePath,
	}
}

func (f *FilesystemLoader) Load(identifier string) (string, error) {
	path := filepath.Join(f.basePath, identifier)
	if !filepath.IsAbs(path) {
		absPath, err := filepath.Abs(path)
		if err != nil {
			return "", fmt.Errorf("failed to resolve absolute path: %w", err)
		}
		path = absPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to load script %s: %w", identifier, err)
	}

	return string(data), nil
}

func SetupRequire(L *lua.LState, loader Loader) {
	originalRequire := L.GetGlobal("require")

	customRequire := L.NewFunction(func(L *lua.LState) int {
		module := L.CheckString(1)

		pkg := L.GetField(L.Get(lua.EnvironIndex), "package")
		preload := L.GetField(pkg, "preload")

		if tbl, ok := preload.(*lua.LTable); ok {
			preloadFn := L.GetField(tbl, module)
			if preloadFn != lua.LNil {
				if fn, ok := originalRequire.(*lua.LFunction); ok {
					L.Push(fn)
					L.Push(lua.LString(module))
					L.Call(1, 1)
					return 1
				}
			}
		}

		scriptContent, err := loader.Load(module)
		if err != nil {
			L.RaiseError("failed to require module %s: %s", module, err.Error())
			return 0
		}

		fn, err := L.LoadString(scriptContent)
		if err != nil {
			L.RaiseError("failed to load module %s: %s", module, err.Error())
			return 0
		}

		L.Push(fn)
		L.Call(0, lua.MultRet)

		return L.GetTop()
	})

	if originalRequire != lua.LNil {
		L.SetGlobal("_original_require", originalRequire)
	}
	L.SetGlobal("require", customRequire)
}
