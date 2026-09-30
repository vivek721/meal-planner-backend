// Package overrides holds hand-checked corrections to automatic FDC
// matching: a pinned fdcId for known-bad matches and per-item gram weights
// for count measures FDC's portions do not cover.
package overrides

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed overrides.json
var embedded []byte

// Override corrects one ingredient. Either field may appear alone: FDCID
// pins the food (skipping search); ItemGrams is the weight of one item.
type Override struct {
	FDCID     int     `json:"fdcId,omitempty"`
	ItemGrams float64 `json:"itemGrams,omitempty"`
	Note      string  `json:"note,omitempty"`
}

// Set is a parsed, validated overrides file.
type Set struct {
	entries map[string]Override
	version string
}

// Load parses the embedded overrides.json. An invalid file is a startup
// error: the server must not run with silently-dropped overrides.
func Load() (*Set, error) {
	return Parse(embedded)
}

// Parse validates data and builds a Set. The version is a hash of the raw
// bytes, so any edit retires every cached result built from the old file.
func Parse(data []byte) (*Set, error) {
	var raw map[string]Override
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("overrides: invalid JSON: %w", err)
	}
	entries := make(map[string]Override, len(raw))
	for name, o := range raw {
		if o.FDCID < 0 {
			return nil, fmt.Errorf("overrides: %q: fdcId must be positive", name)
		}
		if o.ItemGrams < 0 {
			return nil, fmt.Errorf("overrides: %q: itemGrams must be positive", name)
		}
		if o.FDCID == 0 && o.ItemGrams == 0 {
			return nil, fmt.Errorf("overrides: %q: set fdcId and/or itemGrams", name)
		}
		entries[Normalize(name)] = o
	}
	sum := sha256.Sum256(data)
	return &Set{entries: entries, version: hex.EncodeToString(sum[:])[:12]}, nil
}

// Get returns the override for an ingredient name, normalising it first.
func (s *Set) Get(name string) (Override, bool) {
	o, ok := s.entries[Normalize(name)]
	return o, ok
}

// Version identifies the file's content; it is part of nutrition cache keys.
func (s *Set) Version() string {
	return s.version
}

// Normalize lower-cases name, treats "_" as a space and collapses runs of
// spaces — the same rule TheMealDB filter values use.
func Normalize(name string) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(strings.ToLower(name), "_", " ")), " ")
}
