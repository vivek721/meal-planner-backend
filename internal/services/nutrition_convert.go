package services

import (
	"math"
	"regexp"
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

// substituteWords mark foods that stand in for something else, such as
// "Chicken, meatless": never the ingredient a recipe names.
var substituteWords = []string{"meatless", "imitation", "substitute"}

// pickMatch chooses the best FDC search result for a normalised ingredient
// name. Results are ranked, each rule breaking ties in the one before:
//  1. not a substitute food ("meatless", "imitation", "substitute")
//  2. the description's first comma segment matches the name (singular or plural)
//  3. the description contains "raw"
//  4. SR Legacy before Foundation: Foundation foods often list only a
//     reference serving (RACC), not the household portions conversion needs
//  5. FDC's own order
//
// 0 means no match.
func pickMatch(name string, foods []usda.SearchFood) int {
	best, bestScore := 0, -1
	for _, f := range foods {
		if s := matchScore(name, f); s > bestScore {
			best, bestScore = f.FDCID, s
		}
	}
	return best
}

// matchScore encodes pickMatch's rules 1-4 as bits, most significant first;
// the strict comparison in pickMatch keeps FDC's order for equal scores.
func matchScore(name string, f usda.SearchFood) int {
	desc := strings.ToLower(f.Description)
	score := 0
	if !containsAny(desc, substituteWords) {
		score |= 8
	}
	seg := strings.TrimSpace(strings.SplitN(desc, ",", 2)[0])
	if seg == name || seg == name+"s" || seg+"s" == name {
		score |= 4
	}
	if strings.Contains(desc, "raw") {
		score |= 2
	}
	if f.DataType == "SR Legacy" {
		score |= 1
	}
	return score
}

func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
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
			desc := portionText(p)
			if namesPart(desc) {
				continue
			}
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
	desc := portionText(p)
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

// parenNoteRe matches parenthesised notes in FDC portion text, such as the
// "(1/8 cup)" in "tbsp (1/8 cup)" or the size in `medium (2-1/2" dia)`.
var parenNoteRe = regexp.MustCompile(`\([^)]*\)`)

// portionText is the lower-cased unit and modifier of a portion with
// parenthesised notes removed, so a note's unit is never read as the
// portion's own.
func portionText(p usda.Portion) string {
	return strings.ToLower(parenNoteRe.ReplaceAllString(p.Unit+" "+p.Modifier, " "))
}

// partWords mark a portion as a piece or a volume of the food rather than
// one whole item: `slice, medium (1/8" thick)` is not a medium onion.
var partWords = []string{
	"slice", "ring", "wedge", "strip", "cup", "tbsp", "tablespoon", "tsp", "teaspoon",
	"chopped", "diced", "minced", "grated", "shredded",
}

func namesPart(desc string) bool {
	return containsAny(desc, partWords)
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
