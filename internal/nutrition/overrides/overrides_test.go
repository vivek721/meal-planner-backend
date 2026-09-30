package overrides

import (
	"testing"
)

func TestParseAndGet(t *testing.T) {
	set, err := Parse([]byte(`{
		"Chicken_Breasts": { "fdcId": 171077, "itemGrams": 174 },
		"salt":            { "fdcId": 173468 },
		"egg  yolks":      { "itemGrams": 17 }
	}`))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if o, ok := set.Get("chicken breasts"); !ok || o.FDCID != 171077 || o.ItemGrams != 174 {
		t.Errorf("Get(chicken breasts) = %+v, %v", o, ok)
	}
	if o, ok := set.Get("Chicken  Breasts"); !ok || o.FDCID != 171077 {
		t.Errorf("lookup must normalise; got %+v, %v", o, ok)
	}
	if o, ok := set.Get("egg yolks"); !ok || o.FDCID != 0 || o.ItemGrams != 17 {
		t.Errorf("itemGrams-only entry: %+v, %v", o, ok)
	}
	if _, ok := set.Get("butter"); ok {
		t.Error("Get(butter) must miss")
	}
}

func TestParseRejectsBadEntries(t *testing.T) {
	cases := map[string]string{
		"invalid JSON":       `{`,
		"negative fdcId":     `{"x": {"fdcId": -1}}`,
		"negative itemGrams": `{"x": {"fdcId": 1, "itemGrams": -2}}`,
		"empty entry":        `{"x": {}}`,
	}
	for name, data := range cases {
		if _, err := Parse([]byte(data)); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
}

func TestVersionTracksContent(t *testing.T) {
	a, err := Parse([]byte(`{"salt": {"fdcId": 1}}`))
	if err != nil {
		t.Fatal(err)
	}
	b, err := Parse([]byte(`{"salt": {"fdcId": 2}}`))
	if err != nil {
		t.Fatal(err)
	}
	if a.Version() == b.Version() {
		t.Error("different content must give different versions")
	}
	if len(a.Version()) != 12 {
		t.Errorf("version %q: want 12 hex chars", a.Version())
	}
}

func TestLoadEmbeddedFile(t *testing.T) {
	if _, err := Load(); err != nil {
		t.Fatalf("embedded overrides.json must be valid: %v", err)
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize(" Chicken_ Breasts "); got != "chicken breasts" {
		t.Errorf("Normalize = %q", got)
	}
}
