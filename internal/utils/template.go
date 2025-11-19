package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"text/template"
)

// LoadTemplate loads and parses a template file with custom functions
func LoadTemplate(path string) (*template.Template, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read template file %s: %w", path, err)
	}

	funcMap := template.FuncMap{
		"json": ToJSON,
	}

	tmpl, err := template.New("template").Funcs(funcMap).Parse(string(data))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template file %s: %w", path, err)
	}

	return tmpl, nil
}

// ToJSON converts a value to a JSON string
func ToJSON(v interface{}) string {
	b, err := json.Marshal(v)
	if err != nil {
		return `""`
	}
	return string(b)
}
