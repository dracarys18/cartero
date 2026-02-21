package template

import (
	"encoding/json"
	"fmt"
	htmltemplate "html/template"
	"os"
	texttemplate "text/template"
	"time"
)

// Template wraps either text/template or html/template
type Template struct {
	textTmpl *texttemplate.Template
	htmlTmpl *htmltemplate.Template
	isHTML   bool
}

// Load loads and parses a template file with custom functions
func (t *Template) Load(path string, isHTML bool, customFuncs interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read template file %s: %w", path, err)
	}

	t.isHTML = isHTML

	if isHTML {
		// Default HTML template functions
		defaultFuncs := htmltemplate.FuncMap{
			"json": toJSON,
			"timeAgo": func(t time.Time) string {
				duration := time.Since(t)
				if duration < time.Minute {
					return "just now"
				} else if duration < time.Hour {
					mins := int(duration.Minutes())
					if mins == 1 {
						return "1 minute ago"
					}
					return fmt.Sprintf("%d minutes ago", mins)
				} else if duration < 24*time.Hour {
					hours := int(duration.Hours())
					if hours == 1 {
						return "1 hour ago"
					}
					return fmt.Sprintf("%d hours ago", hours)
				} else {
					days := int(duration.Hours() / 24)
					if days == 1 {
						return "1 day ago"
					}
					return fmt.Sprintf("%d days ago", days)
				}
			},
		}

		// Merge custom functions
		if customFuncs != nil {
			if funcs, ok := customFuncs.(htmltemplate.FuncMap); ok {
				for k, v := range funcs {
					defaultFuncs[k] = v
				}
			}
		}

		tmpl, err := htmltemplate.New("template").Funcs(defaultFuncs).Parse(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse HTML template file %s: %w", path, err)
		}
		t.htmlTmpl = tmpl
	} else {
		// Default text template functions
		defaultFuncs := texttemplate.FuncMap{
			"json": toJSON,
		}

		// Merge custom functions
		if customFuncs != nil {
			if funcs, ok := customFuncs.(texttemplate.FuncMap); ok {
				for k, v := range funcs {
					defaultFuncs[k] = v
				}
			}
		}

		tmpl, err := texttemplate.New("template").Funcs(defaultFuncs).Parse(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse text template file %s: %w", path, err)
		}
		t.textTmpl = tmpl
	}

	return nil
}

// TextTemplate returns the underlying text/template.Template
func (t *Template) TextTemplate() *texttemplate.Template {
	return t.textTmpl
}

// HTMLTemplate returns the underlying html/template.Template
func (t *Template) HTMLTemplate() *htmltemplate.Template {
	return t.htmlTmpl
}

// IsHTML returns true if this is an HTML template
func (t *Template) IsHTML() bool {
	return t.isHTML
}

// toJSON converts a value to a JSON string
func toJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `""`
	}
	return string(b)
}
