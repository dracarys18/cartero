package hash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/url"
)

type Hash struct {
	data []byte
}

func newHash(data []byte) Hash {
	return Hash{data: data}
}

func (h Hash) computeHash() string {
	hash := sha256.Sum256(h.data)
	return fmt.Sprintf("%x", hash)
}

func HashURL(u *url.URL) string {
	link := ""
	if u != nil {
		link = u.String()
	}
	data, _ := json.Marshal(map[string]interface{}{
		"link": link,
	})
	return newHash(data).computeHash()
}
