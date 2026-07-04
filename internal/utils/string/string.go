package string

import (
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"strings"
	"unicode/utf8"
)

func Clean(s string) string {
	return strings.ToValidUTF8(s, "")
}

func Truncate(s string, maxBytes int) string {
	s = strings.ToValidUTF8(s, "")
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

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
