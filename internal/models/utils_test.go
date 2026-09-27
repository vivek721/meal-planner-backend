package models

import (
	"regexp"
	"testing"
)

var idPattern = regexp.MustCompile(`^user_\d+_[a-z0-9]{9}$`)

func TestGenerateID_Format(t *testing.T) {
	id, err := generateID("user")
	if err != nil {
		t.Fatalf("generateID() error = %v", err)
	}
	if !idPattern.MatchString(id) {
		t.Errorf("generateID() = %q, want match for %s", id, idPattern)
	}
}

func TestGenerateID_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id, err := generateID("user")
		if err != nil {
			t.Fatal(err)
		}
		if seen[id] {
			t.Fatalf("duplicate ID %q after %d IDs", id, i)
		}
		seen[id] = true
	}
}

func TestGenerateRandomString(t *testing.T) {
	for _, n := range []int{0, 1, 9, 64} {
		s, err := generateRandomString(n)
		if err != nil {
			t.Fatalf("generateRandomString(%d) error = %v", n, err)
		}
		if len(s) != n {
			t.Errorf("len(generateRandomString(%d)) = %d", n, len(s))
		}
		for _, c := range s {
			if (c < 'a' || c > 'z') && (c < '0' || c > '9') {
				t.Errorf("generateRandomString(%d) = %q contains %q", n, s, c)
			}
		}
	}
}

func TestGenerateRandomString_UsesWholeCharset(t *testing.T) {
	s, err := generateRandomString(5000)
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[rune]bool)
	for _, c := range s {
		seen[c] = true
	}
	if len(seen) != len(idCharset) {
		t.Errorf("saw %d distinct characters in 5000 draws, want %d", len(seen), len(idCharset))
	}
}
