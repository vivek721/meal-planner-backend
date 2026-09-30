package services

import (
	"math"
	"strings"

	"github.com/meal-planner/backend/internal/nutrition/measure"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/usda"
)

// Millilitres per unit, for converting FDC volume portions:
// 1 cup = 16 tbsp = 48 tsp = 236.6 ml.
const (
	mlPerCup  = 236.588
	mlPerTbsp = 14.7868
	mlPerTsp  = 4.92892
	mlPerFlOz = 29.5735
)

// pickMatch chooses the best FDC search result for a normalised ingredient
// name: first a description whose first comma segment matches the name
// (singular or plural), then one containing "raw", then FDC's own order.
// 0 means no match.
func pickMatch(name string, foods []usda.SearchFood) int {
	if len(foods) == 0 {
		return 0
	}
	for _, f := range foods {
		seg := strings.ToLower(strings.TrimSpace(strings.SplitN(f.Description, ",", 2)[0]))
		if seg == name || seg == name+"s" || seg+"s" == name {
			return f.FDCID
		}
	}
	for _, f := range foods {
		if strings.Contains(strings.ToLower(f.Description), "raw") {
			return f.FDCID
		}
	}
	return foods[0].FDCID
}

// gramsFor converts a parsed amount of the named ingredient into grams using
// the override and the matched food's portions. ok is false when the amount
// cannot be converted honestly (reason noPortion for Volume/Count).
func gramsFor(amt measure.Amount, name string, o overrides.Override, portions []usda.Portion) (float64, bool) {
	switch amt.Kind {
	case measure.Mass:
		return amt.Value, true
	case measure.Volume:
		for _, p := range portions {
			if ml := portionMl(p); ml > 0 && p.GramWeight > 0 {
				return amt.Value * p.GramWeight / ml, true
			}
		}
		return 0, false
	case measure.Count:
		if o.ItemGrams > 0 {
			return amt.Value * o.ItemGrams, true
		}
		words := []string{"whole", "large", "medium", "fruit", "unit", lastWord(name)}
		for _, p := range portions {
			if p.Amount <= 0 || p.GramWeight <= 0 {
				continue
			}
			desc := strings.ToLower(p.Unit + " " + p.Modifier)
			for _, w := range words {
				if w != "" && strings.Contains(desc, w) {
					return amt.Value * p.GramWeight / p.Amount, true
				}
			}
		}
		return 0, false
	default:
		return 0, false
	}
}

// portionMl returns the portion's total volume in millilitres, or 0 when its
// text names no known volume unit or its amount is not positive.
func portionMl(p usda.Portion) float64 {
	if p.Amount <= 0 {
		return 0
	}
	desc := strings.ToLower(p.Unit + " " + p.Modifier)
	var unit float64
	switch {
	case strings.Contains(desc, "cup"):
		unit = mlPerCup
	case strings.Contains(desc, "tbsp"), strings.Contains(desc, "tablespoon"):
		unit = mlPerTbsp
	case strings.Contains(desc, "tsp"), strings.Contains(desc, "teaspoon"):
		unit = mlPerTsp
	case strings.Contains(desc, "fl oz"), strings.Contains(desc, "fluid ounce"):
		unit = mlPerFlOz
	case hasWord(desc, "ml", "milliliter", "milliliters", "millilitre", "millilitres"):
		unit = 1
	default:
		return 0
	}
	return unit * p.Amount
}

// hasWord reports whether desc contains any of words as a whole token, so
// "small" is never read as "ml".
func hasWord(desc string, words ...string) bool {
	for _, tok := range strings.Fields(desc) {
		for _, w := range words {
			if tok == w {
				return true
			}
		}
	}
	return false
}

// lastWord returns the singular-ish last word of a normalised name
// ("chicken breasts" -> "breast"), used to spot per-item portions.
func lastWord(name string) string {
	fields := strings.Fields(name)
	if len(fields) == 0 {
		return ""
	}
	return strings.TrimSuffix(fields[len(fields)-1], "s")
}

// round1 rounds to one decimal place, the response's precision for grams.
func round1(x float64) float64 {
	return math.Round(x*10) / 10
}
