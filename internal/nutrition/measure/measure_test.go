package measure

import (
	"math"
	"testing"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in   string
		kind Kind
		val  float64
	}{
		// mass, including attached units and package sizes
		{"1kg", Mass, 1000},
		{"200g", Mass, 200},
		{"1 (12 oz.)", Mass, 340.194},
		{"3 oz", Mass, 85.0485},
		{"1 lb", Mass, 453.592},
		{"2 lbs", Mass, 907.184},
		{"8 ounces", Mass, 226.796},
		// volume
		{"200ml", Volume, 200},
		{"1 l", Volume, 1000},
		{"3/4 cup", Volume, 177.441},
		{"¼ cup", Volume, 59.147},
		{"1 1/2 cups", Volume, 354.882},
		{"2 tbs", Volume, 29.5736},
		{"4 Tablespoons", Volume, 59.1472},
		{"1/2 teaspoon", Volume, 2.46446},
		{"2 fl oz", Volume, 59.147},
		{"1 pint", Volume, 473.176},
		// counts
		{"2", Count, 2},
		{"2 chopped", Count, 2},
		{"2 free-range", Count, 2},
		{"5 thinly sliced", Count, 5},
		{"1 whole", Count, 1},
		{"2 chicken breasts", Count, 2},
		{"Juice of 1", Count, 1},
		{"Zest of 1", Count, 1},
		{"2-3", Count, 2.5},
		{"1¼", Count, 1.25},
		{"1.2", Count, 1.2},
		// a parenthesised size the parser cannot read falls back to a count,
		// never a guessed mass
		{"2 (16-oz.)", Count, 2},
		// every remaining distinct measure from sample meals 52772, 52874,
		// 52959, 52940 and 52795 (fetched 2026-09-30)
		{"25g", Mass, 25},
		{"300g", Mass, 300},
		{"1.2 kg", Mass, 1200},
		{"400ml", Volume, 400},
		{"1 tbs", Volume, 14.7868},
		{"2 tbs chopped", Volume, 29.5736},
		{"1 tsp", Volume, 4.92892},
		{"3 tsp Dried", Volume, 14.7868},
		{"1 cup", Volume, 236.588},
		{"2 cups", Volume, 473.176},
		{"3 cups", Volume, 709.764},
		{"½ cup", Volume, 118.294},
		{"1 finely sliced", Count, 1},
		{"2 finely chopped", Count, 2},
		{"2 medium", Count, 2},
		{"3 sprigs", Count, 3},
		{"8 cloves chopped", Count, 8},
		// unmeasurable
		{"", Unmeasurable, 0},
		{"pinch", Unmeasurable, 0},
		{"dash", Unmeasurable, 0},
		{"to taste", Unmeasurable, 0},
		{"to serve", Unmeasurable, 0},
		{"to garnish", Unmeasurable, 0},
		{"for frying", Unmeasurable, 0},
		{"drizzle", Unmeasurable, 0},
		{"handful", Unmeasurable, 0},
	}
	for _, c := range cases {
		got := Parse(c.in)
		if got.Kind != c.kind || math.Abs(got.Value-c.val) > 0.01 {
			t.Errorf("Parse(%q) = {%v %v}, want {%v %v}", c.in, got.Kind, got.Value, c.kind, c.val)
		}
	}
}
