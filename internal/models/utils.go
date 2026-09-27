package models

import (
	"crypto/rand"
	"fmt"
	"time"
)

const idCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

// generateID generates a unique ID of the form <prefix>_<unix-nanos>_<9 random chars>.
func generateID(prefix string) (string, error) {
	randomPart, err := generateRandomString(9)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s_%d_%s", prefix, time.Now().UnixNano(), randomPart), nil
}

// generateRandomString returns a cryptographically random string of the given
// length drawn uniformly from idCharset.
func generateRandomString(length int) (string, error) {
	// Reject bytes >= maxByte so that every charset index is equally likely.
	const maxByte = 256 - 256%len(idCharset)

	out := make([]byte, 0, length)
	buf := make([]byte, length)
	for len(out) < length {
		if _, err := rand.Read(buf); err != nil {
			return "", fmt.Errorf("generate random ID: %w", err)
		}
		for _, b := range buf {
			if int(b) >= maxByte {
				continue
			}
			out = append(out, idCharset[int(b)%len(idCharset)])
			if len(out) == length {
				break
			}
		}
	}
	return string(out), nil
}
