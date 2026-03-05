package hash

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
)

type Hash struct {
	data []byte
}

func NewHash(data []byte) Hash {
	return Hash{data: data}
}

func (h Hash) ComputeHash() string {
	hash := sha256.Sum256(h.data)
	return fmt.Sprintf("%x", hash)
}

func HashURL(url string) string {
	data, _ := json.Marshal(map[string]interface{}{
		"link": url,
	})
	return NewHash(data).ComputeHash()
}
