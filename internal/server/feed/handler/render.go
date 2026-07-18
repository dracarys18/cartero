package handler

import (
	"bytes"
	"fmt"
	htmltemplate "html/template"
	"net/http"
	"strings"
	"time"

	utils "cartero/internal/utils/string"
)

func timeAgo(t time.Time) string {
	duration := time.Since(t)
	switch {
	case duration < time.Minute:
		return "just now"
	case duration < time.Hour:
		mins := int(duration.Minutes())
		if mins == 1 {
			return "1 minute ago"
		}
		return fmt.Sprintf("%d minutes ago", mins)
	case duration < 24*time.Hour:
		hours := int(duration.Hours())
		if hours == 1 {
			return "1 hour ago"
		}
		return fmt.Sprintf("%d hours ago", hours)
	default:
		days := int(duration.Hours() / 24)
		if days == 1 {
			return "1 day ago"
		}
		return fmt.Sprintf("%d days ago", days)
	}
}

func funcMap() htmltemplate.FuncMap {
	return htmltemplate.FuncMap{
		"timeAgo": timeAgo,
		"add":     func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"split": func(s, sep string) []string {
			if s == "" {
				return []string{}
			}
			return strings.Split(s, sep)
		},
		"formatSource": utils.Readable,
		"hueClass":     hueClass,
	}
}

func (h *Handler) renderBytes(data map[string]interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := h.tmpl.HTMLTemplate().Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func writeHTML(w http.ResponseWriter, r *http.Request, e cacheEntry) {
	w.Header().Set("ETag", e.etag)
	if match := r.Header.Get("If-None-Match"); match == e.etag {
		w.WriteHeader(http.StatusNotModified)
		return
	}
	_, _ = w.Write(e.html)
}

func hueClass(s string) string {
	hue := 0
	for _, c := range s {
		hue = hue*31 + int(c)
	}
	if hue < 0 {
		hue = -hue
	}
	return fmt.Sprintf("ph%d", hue%10)
}
