package models

import (
	"fmt"
	"math/rand"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// generateID generates a unique ID with a prefix
func generateID(prefix string) string {
	timestamp := time.Now().UnixNano()
	randomPart := generateRandomString(9)
	return fmt.Sprintf("%s_%d_%s", prefix, timestamp, randomPart)
}

// generateRandomString generates a random alphanumeric string of specified length
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[rand.Intn(len(charset))]
	}
	return string(b)
}
