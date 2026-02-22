package feed

import (
	"context"
	"fmt"
	htmltemplate "html/template"
	"net/http"
	"sort"
	"time"

	"cartero/internal/storage"
	"cartero/internal/template"

	"github.com/gorilla/feeds"
)

type Config struct {
	Port     string
	FeedSize int
	MaxItems int
}

type Server struct {
	name         string
	config       Config
	feedStore    storage.FeedStore
	server       *http.Server
	startCh      chan error
	htmlTemplate *template.Template
}

func New(name string, config Config, feedStore storage.FeedStore) *Server {
	if config.Port == "" {
		config.Port = "8080"
	}
	if config.FeedSize == 0 {
		config.FeedSize = 100
	}
	if config.MaxItems == 0 {
		config.MaxItems = 50
	}

	funcMap := htmltemplate.FuncMap{
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
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
		},
	}

	tmpl := &template.Template{}
	if err := tmpl.Load("templates/homepage.gotmpl", template.HtmlTemplate, funcMap); err != nil {
		panic(err.Error())
	}

	return &Server{
		name:         name,
		config:       config,
		feedStore:    feedStore,
		startCh:      make(chan error, 1),
		htmlTemplate: tmpl,
	}
}

func (s *Server) Start(ctx context.Context) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.handleHomepage)
	mux.HandleFunc("/feed.rss", s.handleRSSFeed)
	mux.HandleFunc("/feed.atom", s.handleAtomFeed)
	mux.HandleFunc("/feed.json", s.handleJSONFeed)
	mux.HandleFunc("/feed.health", s.handleHealth)

	// Serve static assets
	fileServer := http.FileServer(http.Dir("assets"))
	mux.Handle("/assets/", http.StripPrefix("/assets/", fileServer))

	s.server = &http.Server{
		Addr:    ":" + s.config.Port,
		Handler: mux,
	}

	go func() {
		err := s.server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			s.startCh <- err
			fmt.Printf("Feed server error: %v\n", err)
		}
	}()

	select {
	case err := <-s.startCh:
		return fmt.Errorf("failed to start feed server on port %s: %w", s.config.Port, err)
	case <-time.After(1 * time.Second):
		return nil
	}
}

func (s *Server) Shutdown(ctx context.Context) error {
	if s.server != nil {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := s.server.Shutdown(shutdownCtx); err != nil && err != context.Canceled && err != http.ErrServerClosed {
			fmt.Printf("Feed server shutdown error: %v\n", err)
		}

	}
	return nil
}

func (s *Server) handleRSSFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := s.feedStore.ListRecentEntries(r.Context(), s.config.FeedSize)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	rss, err := feed.ToRss()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, rss)
}

func (s *Server) handleAtomFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := s.feedStore.ListRecentEntries(r.Context(), s.config.FeedSize)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	atom, err := feed.ToAtom()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/xml; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, atom)
}

func (s *Server) handleJSONFeed(w http.ResponseWriter, r *http.Request) {
	entries, err := s.feedStore.ListRecentEntries(r.Context(), s.config.FeedSize)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	feed := s.buildFeed(entries)
	jsonStr, err := feed.ToJSON()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Content-Disposition", "inline")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	fmt.Fprint(w, jsonStr)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"status":"ok","name":"%s","time":"%s"}`, s.name, time.Now().UTC().Format(time.RFC3339))
}

func (s *Server) buildFeed(entries []storage.FeedEntry) *feeds.Feed {
	items := make([]*feeds.Item, 0, len(entries))

	for _, entry := range entries {
		item := &feeds.Item{
			Id:          entry.ID,
			Title:       entry.Title,
			Link:        &feeds.Link{Href: entry.Link},
			Description: entry.Description,
			Content:     entry.Content,
			Author:      &feeds.Author{Name: entry.Author},
			Created:     entry.PublishedAt,
		}
		items = append(items, item)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Created.After(items[j].Created)
	})

	if len(items) > s.config.MaxItems {
		items = items[:s.config.MaxItems]
	}

	return &feeds.Feed{
		Title:       fmt.Sprintf("Cartero Feed (%s)", s.name),
		Link:        &feeds.Link{Href: "http://localhost/"},
		Description: "Content aggregation feed from Cartero",
		Author:      &feeds.Author{Name: "Cartero"},
		Created:     time.Now().UTC(),
		Items:       items,
	}
}

func (s *Server) handleHomepage(w http.ResponseWriter, r *http.Request) {
	page := s.parsePageParam(r)
	dateRange := s.parseDateParam(r)
	perPage := 20

	startDate, endDate := s.calculateDateRange(dateRange)

	result, err := s.feedStore.ListEntriesPaginated(r.Context(), page, perPage, startDate, endDate)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Error: %v", err)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")

	data := map[string]interface{}{
		"Title":      fmt.Sprintf("Cartero - %s", s.name),
		"Entries":    result.Entries,
		"Now":        time.Now(),
		"Page":       result.Page,
		"TotalPages": result.TotalPages,
		"HasNext":    result.HasNext,
		"HasPrev":    result.HasPrevious,
		"DateRange":  dateRange,
		"Total":      result.Total,
	}

	if err := s.htmlTemplate.HTMLTemplate().Execute(w, data); err != nil {
		fmt.Printf("Template error: %v\n", err)
	}
}

func (s *Server) parsePageParam(r *http.Request) int {
	pageStr := r.URL.Query().Get("page")
	if pageStr == "" {
		return 1
	}
	page := 1
	fmt.Sscanf(pageStr, "%d", &page)
	if page < 1 {
		page = 1
	}
	return page
}

func (s *Server) parseDateParam(r *http.Request) string {
	dateRange := r.URL.Query().Get("date")
	if dateRange == "" {
		return "today"
	}
	if dateRange != "today" && dateRange != "yesterday" {
		return "today"
	}
	return dateRange
}

func (s *Server) calculateDateRange(dateRange string) (time.Time, time.Time) {
	now := time.Now()

	switch dateRange {
	case "yesterday":
		return s.yesterdayRange(now)
	default:
		return s.todayRange(now)
	}
}

func (s *Server) todayRange(now time.Time) (time.Time, time.Time) {
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfToday := startOfToday.Add(24 * time.Hour)
	return startOfToday, endOfToday
}

func (s *Server) yesterdayRange(now time.Time) (time.Time, time.Time) {
	startOfYesterday := time.Date(now.Year(), now.Month(), now.Day()-1, 0, 0, 0, 0, now.Location())
	endOfYesterday := startOfYesterday.Add(24 * time.Hour)
	return startOfYesterday, endOfYesterday
}
