package utils

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
)

func Readable(s string) string {
	caser := cases.Title(language.English)
	words := strings.FieldsFunc(s, func(r rune) bool {
		return r == '_' || r == '-'
	})
	result := make([]string, len(words))
	for i, word := range words {
		result[i] = caser.String(strings.ToLower(word))
	}
	return strings.Join(result, " ")

}
