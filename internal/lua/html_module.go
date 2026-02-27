package lua

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
	lua "github.com/yuin/gopher-lua"
)

const (
	luaDocumentTypeName = "html_document"
	luaElementTypeName  = "html_element"
)

type HTMLModule struct{}

func NewHTMLModule() *HTMLModule {
	return &HTMLModule{}
}

func (h *HTMLModule) Name() string {
	return "html"
}

func (h *HTMLModule) Register(L *lua.LState) error {
	h.registerDocumentType(L)
	h.registerElementType(L)

	htmlTable := L.NewTable()

	L.SetField(htmlTable, "parse", L.NewFunction(h.htmlParse))
	L.SetField(htmlTable, "select", L.NewFunction(h.htmlSelect))
	L.SetField(htmlTable, "select_one", L.NewFunction(h.htmlSelectOne))
	L.SetField(htmlTable, "text", L.NewFunction(h.htmlText))
	L.SetField(htmlTable, "attr", L.NewFunction(h.htmlAttr))
	L.SetField(htmlTable, "html", L.NewFunction(h.htmlHTML))

	L.SetGlobal("html", htmlTable)
	return nil
}

func (h *HTMLModule) registerDocumentType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaDocumentTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{}))
}

func (h *HTMLModule) registerElementType(L *lua.LState) {
	mt := L.NewTypeMetatable(luaElementTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{}))
}

func (h *HTMLModule) htmlParse(L *lua.LState) int {
	htmlContent := L.CheckString(1)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to parse HTML: %s", err.Error())))
		return 2
	}

	ud := L.NewUserData()
	ud.Value = doc
	L.SetMetatable(ud, L.GetTypeMetatable(luaDocumentTypeName))

	L.Push(ud)
	return 1
}

func (h *HTMLModule) htmlSelect(L *lua.LState) int {
	docUD := L.CheckUserData(1)
	selector := L.CheckString(2)

	var selection *goquery.Selection

	switch v := docUD.Value.(type) {
	case *goquery.Document:
		selection = v.Find(selector)
	case *goquery.Selection:
		selection = v.Find(selector)
	default:
		L.ArgError(1, "expected html document or element")
		return 0
	}

	elements := L.NewTable()
	selection.Each(func(i int, s *goquery.Selection) {
		ud := L.NewUserData()
		ud.Value = s
		L.SetMetatable(ud, L.GetTypeMetatable(luaElementTypeName))
		elements.Append(ud)
	})

	L.Push(elements)
	return 1
}

func (h *HTMLModule) htmlSelectOne(L *lua.LState) int {
	docUD := L.CheckUserData(1)
	selector := L.CheckString(2)

	var selection *goquery.Selection

	switch v := docUD.Value.(type) {
	case *goquery.Document:
		selection = v.Find(selector).First()
	case *goquery.Selection:
		selection = v.Find(selector).First()
	default:
		L.ArgError(1, "expected html document or element")
		return 0
	}

	if selection.Length() == 0 {
		L.Push(lua.LNil)
		return 1
	}

	ud := L.NewUserData()
	ud.Value = selection
	L.SetMetatable(ud, L.GetTypeMetatable(luaElementTypeName))

	L.Push(ud)
	return 1
}

func (h *HTMLModule) htmlText(L *lua.LState) int {
	elemUD := L.CheckUserData(1)

	selection, ok := elemUD.Value.(*goquery.Selection)
	if !ok {
		L.ArgError(1, "expected html element")
		return 0
	}

	text := strings.TrimSpace(selection.Text())
	L.Push(lua.LString(text))
	return 1
}

func (h *HTMLModule) htmlAttr(L *lua.LState) int {
	elemUD := L.CheckUserData(1)
	attrName := L.CheckString(2)

	selection, ok := elemUD.Value.(*goquery.Selection)
	if !ok {
		L.ArgError(1, "expected html element")
		return 0
	}

	attrValue, exists := selection.Attr(attrName)
	if !exists {
		L.Push(lua.LNil)
		return 1
	}

	L.Push(lua.LString(attrValue))
	return 1
}

func (h *HTMLModule) htmlHTML(L *lua.LState) int {
	elemUD := L.CheckUserData(1)

	selection, ok := elemUD.Value.(*goquery.Selection)
	if !ok {
		L.ArgError(1, "expected html element")
		return 0
	}

	htmlContent, err := selection.Html()
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(fmt.Sprintf("failed to get HTML: %s", err.Error())))
		return 2
	}

	L.Push(lua.LString(htmlContent))
	return 1
}
