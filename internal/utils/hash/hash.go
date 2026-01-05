package hash

import (
	"crypto/sha256"
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
