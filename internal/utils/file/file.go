package file

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"time"
)

type File struct {
	Href string `json:"href"`
}

func NewFile(href string) File {
	return File{Href: href}
}

func (f File) Get() ([]byte, error) {
	if f.IsLink() {
		return getLinkContent(f.Href)
	} else {
		return getFileContent(f.Href)
	}
}

func getFileContent(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func getLinkContent(url string) ([]byte, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch OPML: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch OPML: status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read OPML response: %w", err)
	}

	return data, nil
}

func (f File) IsLink() bool {
	_, err := url.Parse(f.Href)
	return err == nil
}
