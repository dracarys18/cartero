package template

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"maps"
	"os"
	texttemplate "text/template"
)

type TemplateType int

const (
	HtmlTemplate TemplateType = iota
	TextTemplate
)

type Template struct {
	textTmpl *texttemplate.Template
	htmlTmpl *htmltemplate.Template
}

func (t *Template) Load(path string, kind TemplateType, customFuncs any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", path, err)
	}

	switch kind {
	case HtmlTemplate:
		{
			defaultFuncs := htmltemplate.FuncMap{
				"json": toJSON,
			}

			if customFuncs != nil {
				if funcs, ok := customFuncs.(htmltemplate.FuncMap); ok {
					maps.Copy(defaultFuncs, funcs)
				}
			}

			tmpl, err := htmltemplate.New("template").Funcs(defaultFuncs).Parse(string(data))
			if err != nil {
				return fmt.Errorf("failed to parse HTML template file %s: %w", path, err)
			}
			t.htmlTmpl = tmpl
		}
	case TextTemplate:
		{
			defaultFuncs := texttemplate.FuncMap{
				"json": toJSON,
			}

			if customFuncs != nil {
				if funcs, ok := customFuncs.(texttemplate.FuncMap); ok {
					maps.Copy(defaultFuncs, funcs)
				}
			}

			tmpl, err := texttemplate.New("template").Funcs(defaultFuncs).Parse(string(data))
			if err != nil {
				return fmt.Errorf("failed to parse text template file %s: %w", path, err)
			}
			t.textTmpl = tmpl
		}
	}
	return nil
}

func (t *Template) TextTemplate() *texttemplate.Template {
	return t.textTmpl
}

func (t *Template) HTMLTemplate() *htmltemplate.Template {
	return t.htmlTmpl
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `""`
	}
	return string(b)
}
