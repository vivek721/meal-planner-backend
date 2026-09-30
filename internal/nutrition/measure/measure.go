// Package measure parses TheMealDB's free-text ingredient measures ("3/4 cup",
// "1kg", "2 chicken breasts") into typed amounts. It is pure: no I/O.
package measure

import (
	"regexp"
	"strconv"
	"strings"
)

// Kind classifies a parsed measure.
type Kind int

// Kinds of amount. Unmeasurable covers "pinch", "to taste", empty text and
// anything else with no leading quantity.
const (
	Unmeasurable Kind = iota
	Mass              // Value is grams
	Volume            // Value is milliliters
	Count             // Value is a number of items
)

// Amount is a parsed measure. Value's unit depends on Kind.
type Amount struct {
	Kind  Kind
	Value float64
}

var massUnits = map[string]float64{
	"g": 1, "gram": 1, "grams": 1, "kg": 1000,
	"oz": 28.3495, "ounce": 28.3495, "ounces": 28.3495,
	"lb": 453.592, "lbs": 453.592, "pound": 453.592, "pounds": 453.592,
}

var volumeUnits = map[string]float64{
	"ml": 1, "l": 1000, "litre": 1000, "litres": 1000, "liter": 1000, "liters": 1000, //nolint:misspell // British spellings appear in recipe measures
	"tsp": 4.92892, "teaspoon": 4.92892, "teaspoons": 4.92892,
	"tbs": 14.7868, "tbsp": 14.7868, "tablespoon": 14.7868, "tablespoons": 14.7868,
	"cup": 236.588, "cups": 236.588, "pint": 473.176, "pints": 473.176,
}

const flOzMl = 29.5735

var unicodeFractions = strings.NewReplacer(
	"¼", " 1/4", "½", " 1/2", "¾", " 3/4", "⅓", " 1/3", "⅔", " 2/3", "⅛", " 1/8",
)

// numUnitRe splits an attached unit off a number: "1kg" -> "1 kg".
var numUnitRe = regexp.MustCompile(`(\d)([a-z])`)

var parenRe = regexp.MustCompile(`\(([^)]*)\)`)

// Parse turns a free-text measure into an Amount. It never guesses: text it
// cannot read confidently is Unmeasurable (or a bare Count).
func Parse(measure string) Amount {
	s := strings.ToLower(strings.TrimSpace(measure))
	s = strings.TrimSpace(unicodeFractions.Replace(s))
	s = numUnitRe.ReplaceAllString(s, "$1 $2")
	if s == "" {
		return Amount{Kind: Unmeasurable}
	}
	if a, ok := parsePackage(s); ok {
		return a
	}
	if a, ok := parseOfCount(s); ok {
		return a
	}
	qty, rest, ok := parseQuantity(s)
	if !ok {
		return Amount{Kind: Unmeasurable}
	}
	if g, ok := unitGrams(rest); ok {
		return Amount{Kind: Mass, Value: qty * g}
	}
	if ml, ok := unitMl(rest); ok {
		return Amount{Kind: Volume, Value: qty * ml}
	}
	return Amount{Kind: Count, Value: qty}
}

// parsePackage handles a parenthesised package size: "1 (12 oz.)" is 12 oz
// per package times the leading count. Anything unreadable falls through.
func parsePackage(s string) (Amount, bool) {
	m := parenRe.FindStringSubmatchIndex(s)
	if m == nil {
		return Amount{}, false
	}
	qty, rest, ok := parseQuantity(strings.TrimSpace(s[m[2]:m[3]]))
	if !ok {
		return Amount{}, false
	}
	mult := 1.0
	if lead := strings.TrimSpace(s[:m[0]]); lead != "" {
		if q, _, ok := parseQuantity(lead); ok && q > 0 {
			mult = q
		}
	}
	if g, ok := unitGrams(rest); ok {
		return Amount{Kind: Mass, Value: mult * qty * g}, true
	}
	if ml, ok := unitMl(rest); ok {
		return Amount{Kind: Volume, Value: mult * qty * ml}, true
	}
	return Amount{}, false
}

// parseOfCount handles "juice of 1" and "zest of 1" as item counts.
func parseOfCount(s string) (Amount, bool) {
	for _, p := range []string{"juice of ", "zest of "} {
		if strings.HasPrefix(s, p) {
			if q, _, ok := parseQuantity(strings.TrimPrefix(s, p)); ok {
				return Amount{Kind: Count, Value: q}, true
			}
		}
	}
	return Amount{}, false
}

// parseQuantity reads a leading quantity (integer, decimal, fraction, mixed
// number or range) and returns it with the remaining text.
func parseQuantity(s string) (qty float64, rest string, ok bool) {
	fields := strings.Fields(s)
	if len(fields) == 0 {
		return 0, "", false
	}
	if qty, ok = parseNumber(fields[0]); !ok {
		return 0, "", false
	}
	words := fields[1:]
	if len(words) > 0 { // mixed number: "1 1/2"
		if f, isFrac := parseFraction(words[0]); isFrac {
			qty += f
			words = words[1:]
		}
	}
	return qty, strings.Join(words, " "), true
}

func parseNumber(tok string) (float64, bool) {
	if lo, hi, ok := splitRange(tok); ok {
		return (lo + hi) / 2, true
	}
	if f, ok := parseFraction(tok); ok {
		return f, true
	}
	f, err := strconv.ParseFloat(tok, 64)
	if err != nil || f < 0 {
		return 0, false
	}
	return f, true
}

func parseFraction(tok string) (float64, bool) {
	n, d, ok := strings.Cut(tok, "/")
	if !ok {
		return 0, false
	}
	nf, e1 := strconv.ParseFloat(n, 64)
	df, e2 := strconv.ParseFloat(d, 64)
	if e1 != nil || e2 != nil || df == 0 {
		return 0, false
	}
	return nf / df, true
}

// splitRange reads "2-3" as a range and reports its bounds.
func splitRange(tok string) (lo, hi float64, ok bool) {
	l, h, found := strings.Cut(tok, "-")
	if !found {
		return 0, 0, false
	}
	lf, e1 := strconv.ParseFloat(l, 64)
	hf, e2 := strconv.ParseFloat(h, 64)
	if e1 != nil || e2 != nil {
		return 0, 0, false
	}
	return lf, hf, true
}

func firstWord(s string) string {
	f := strings.Fields(s)
	if len(f) == 0 {
		return ""
	}
	return strings.TrimSuffix(f[0], ".")
}

func unitGrams(rest string) (float64, bool) {
	g, ok := massUnits[firstWord(rest)]
	return g, ok
}

func unitMl(rest string) (float64, bool) {
	f := strings.Fields(rest)
	if len(f) >= 2 && strings.TrimSuffix(f[0], ".") == "fl" && strings.TrimSuffix(f[1], ".") == "oz" {
		return flOzMl, true
	}
	ml, ok := volumeUnits[firstWord(rest)]
	return ml, ok
}
