package utils

import (
	"crypto/sha256"
	"fmt"
)

func ComputeHash(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}
