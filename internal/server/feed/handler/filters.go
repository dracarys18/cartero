package handler

import (
	"fmt"
	"maps"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"cartero/internal/storage"
	utils "cartero/internal/utils/string"
)

const (
	dateLayout         = "2006-01-02"
	frontPageWindow    = 24 * time.Hour
	defaultArchiveDays = 7
)

type feedQuery struct {
	Archive bool
	From    time.Time
	To      time.Time
	Topics  []string
	Sources []string
	Page    int
}

type filterPill struct {
	Label string
	Href  string
}

type filterOption struct {
	Value   string
	Label   string
	Count   int
	Checked bool
}

type hiddenField struct {
	Name  string
	Value string
}

type filterMenu struct {
	Name      string
	Param     string
	Action    string
	Summary   string
	Set       bool
	Options   []filterOption
	Hidden    []hiddenField
	ClearHref string
}

func parseFeedQuery(r *http.Request, archive bool, now time.Time) feedQuery {
	q := r.URL.Query()
	fq := feedQuery{
		Archive: archive,
		Topics:  uniqueValues(q["topic"]),
		Sources: uniqueValues(q["source"]),
		Page:    parsePageParam(r),
	}
	if archive {
		yesterday := startOfDay(now).AddDate(0, 0, -1)
		fq.To = parseDate(q.Get("to"), now.Location(), yesterday)
		fq.From = parseDate(q.Get("from"), now.Location(), fq.To.AddDate(0, 0, 1-defaultArchiveDays))
		if fq.From.After(fq.To) {
			fq.From, fq.To = fq.To, fq.From
		}
	}
	return fq
}

func parseDate(v string, loc *time.Location, fallback time.Time) time.Time {
	if t, err := time.ParseInLocation(dateLayout, v, loc); err == nil {
		return t
	}
	return fallback
}

func startOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func uniqueValues(values []string) []string {
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			out = append(out, v)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

func without(values []string, drop string) []string {
	return slices.DeleteFunc(slices.Clone(values), func(v string) bool { return v == drop })
}

func (q feedQuery) filtered() bool {
	return len(q.Topics) > 0 || len(q.Sources) > 0
}

func (q feedQuery) path() string {
	if q.Archive {
		return "/archive"
	}
	return "/"
}

func (q feedQuery) cacheKey() string {
	return q.path() + "?" + q.values().Encode() + "|" + strconv.Itoa(q.Page)
}

func (q feedQuery) storageFilter(now time.Time) storage.EntryFilter {
	f := storage.EntryFilter{Topics: q.Topics, Sources: q.Sources}
	if q.Archive {
		f.Since, f.Until = q.From, q.To.AddDate(0, 0, 1)
	} else {
		f.Since, f.Until = now.Add(-frontPageWindow), now.Add(time.Hour)
	}
	return f
}

func (q feedQuery) values() url.Values {
	v := url.Values{}
	if q.Archive && !q.From.IsZero() {
		v.Set("from", q.From.Format(dateLayout))
		v.Set("to", q.To.Format(dateLayout))
	}
	for _, t := range q.Topics {
		v.Add("topic", t)
	}
	for _, s := range q.Sources {
		v.Add("source", s)
	}
	return v
}

func (q feedQuery) href() string {
	v := q.values()
	if q.Page > 1 {
		v.Set("page", strconv.Itoa(q.Page))
	}
	if len(v) == 0 {
		return q.path()
	}
	return q.path() + "?" + v.Encode()
}

func (q feedQuery) with(change func(*feedQuery)) string {
	next := q
	next.Page = 1
	change(&next)
	return next.href()
}

func (q feedQuery) pageHref(page int) string {
	next := q
	next.Page = page
	return next.href()
}

func (q feedQuery) archiveToggleHref() string {
	return q.with(func(n *feedQuery) { n.Archive, n.From, n.To = !q.Archive, time.Time{}, time.Time{} })
}

func (q feedQuery) dateHidden() []hiddenField {
	return hiddenFields(q.with(func(n *feedQuery) { n.From, n.To = time.Time{}, time.Time{} }))
}

func (q feedQuery) clearHref() string {
	return q.with(func(n *feedQuery) { n.Topics, n.Sources = nil, nil })
}

func (q feedQuery) topicMenu(facets []storage.Facet) filterMenu {
	return q.menu("Topic", "topic", q.Topics, facets, func(v string) string { return v },
		func(n *feedQuery) { n.Topics = nil })
}

func (q feedQuery) sourceMenu(facets []storage.Facet) filterMenu {
	return q.menu("Source", "source", q.Sources, facets, utils.Readable,
		func(n *feedQuery) { n.Sources = nil })
}

func (q feedQuery) menu(name, param string, selected []string, facets []storage.Facet, label func(string) string, reset func(*feedQuery)) filterMenu {
	m := filterMenu{
		Name:      name,
		Param:     param,
		Action:    q.path(),
		Summary:   summarize(selected, label),
		Set:       len(selected) > 0,
		Hidden:    hiddenFields(q.with(reset)),
		ClearHref: q.with(reset),
	}
	for _, f := range facets {
		m.Options = append(m.Options, filterOption{
			Value:   f.Value,
			Label:   label(f.Value),
			Count:   f.Count,
			Checked: slices.Contains(selected, f.Value),
		})
	}
	for _, v := range selected {
		if !slices.ContainsFunc(facets, func(f storage.Facet) bool { return f.Value == v }) {
			m.Options = append(m.Options, filterOption{Value: v, Label: label(v), Checked: true})
		}
	}
	return m
}

func hiddenFields(href string) []hiddenField {
	u, err := url.Parse(href)
	if err != nil {
		return nil
	}
	values := u.Query()
	var fields []hiddenField
	for _, key := range slices.Sorted(maps.Keys(values)) {
		for _, v := range values[key] {
			fields = append(fields, hiddenField{Name: key, Value: v})
		}
	}
	return fields
}

func summarize(selected []string, label func(string) string) string {
	switch len(selected) {
	case 0:
		return "All"
	case 1:
		return label(selected[0])
	default:
		return fmt.Sprintf("%d selected", len(selected))
	}
}

func (q feedQuery) activePills() []filterPill {
	var pills []filterPill
	for _, t := range q.Topics {
		pills = append(pills, filterPill{Label: t, Href: q.with(func(n *feedQuery) { n.Topics = without(n.Topics, t) })})
	}
	for _, s := range q.Sources {
		pills = append(pills, filterPill{Label: utils.Readable(s), Href: q.with(func(n *feedQuery) { n.Sources = without(n.Sources, s) })})
	}
	return pills
}

func (q feedQuery) rangeLabel() string {
	if q.From.Equal(q.To) {
		return q.From.Format("Jan 2, 2006")
	}
	if q.From.Year() == q.To.Year() {
		return q.From.Format("Jan 2") + " – " + q.To.Format("Jan 2, 2006")
	}
	return q.From.Format("Jan 2, 2006") + " – " + q.To.Format("Jan 2, 2006")
}
