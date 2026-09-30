package services

import (
	"testing"

	"github.com/meal-planner/backend/internal/nutrition/measure"
	"github.com/meal-planner/backend/internal/nutrition/overrides"
	"github.com/meal-planner/backend/internal/usda"
)

func TestPickMatchRanking(t *testing.T) {
	foods := []usda.SearchFood{
		{FDCID: 1, Description: "Soup, chicken noodle, canned"},
		{FDCID: 2, Description: "Chicken breast, raw"},
		{FDCID: 3, Description: "Chicken breasts, oven-roasted"},
	}
	// 1st rule: first comma segment equals the name (singular or plural).
	if got := pickMatch("chicken breasts", foods); got != 2 {
		t.Errorf("segment match (plural tolerant) = %d, want 2", got)
	}
	// 2nd rule: contains "raw".
	if got := pickMatch("poultry", foods); got != 2 {
		t.Errorf("raw preference = %d, want 2", got)
	}
	// 3rd rule: FDC's own order.
	cooked := []usda.SearchFood{{FDCID: 7, Description: "Soup, canned"}, {FDCID: 8, Description: "Broth, cubes"}}
	if got := pickMatch("stock", cooked); got != 7 {
		t.Errorf("fallback order = %d, want 7", got)
	}
	if got := pickMatch("anything", nil); got != 0 {
		t.Errorf("no results = %d, want 0", got)
	}
}

// TestPickMatchOnRecordedSearches ranks real FDC search results recorded
// on 2026-09-30 (testdata/fdc_searches.json) for the sample meals.
func TestPickMatchOnRecordedSearches(t *testing.T) {
	var searches map[string][]usda.SearchFood
	mustLoad(t, "testdata/fdc_searches.json", &searches)

	want := map[string]int{
		"carrots":      170393, // Carrots, raw — not "Carrot, dehydrated"
		"red pepper":   170108, // Peppers, sweet, red, raw (SR Legacy has household portions)
		"garlic":       169230, // Garlic, raw (SR Legacy, not the RACC-only Foundation twin)
		"green beans":  169961, // Beans, snap, green, raw (SR Legacy)
		"fennel":       169385, // Fennel, bulb, raw (SR Legacy)
		"mustard":      172234, // Mustard, prepared, yellow (SR Legacy)
		"rapeseed oil": 172336, // Oil, canola (SR Legacy)
		"water":        174158, // Water, bottled, generic — name match beats "Water convolvulus,raw"
		"onions":       170000, // Onions, raw
		"lemon":        167746, // Lemons, raw, without peel
		"salmon":       173688, // Fish, salmon, chinook, raw
		"chicken":      171116, // Chicken, ground, raw — never "Chicken, meatless"
	}
	for name, id := range want {
		foods, ok := searches[name]
		if !ok {
			t.Fatalf("no recorded search for %q", name)
		}
		if got := pickMatch(name, foods); got != id {
			t.Errorf("pickMatch(%q) = %d, want %d", name, got, id)
		}
	}
}

func TestGramsForMass(t *testing.T) {
	g, ok := gramsFor(measure.Amount{Kind: measure.Mass, Value: 340}, "beef", overrides.Override{}, nil)
	if !ok || g != 340 {
		t.Errorf("mass = %v, %v", g, ok)
	}
}

func TestGramsForVolumeUsesAnyVolumePortion(t *testing.T) {
	// Only a tbsp portion exists (1 tbsp = 16 g); 177.44 ml (3/4 cup) of it:
	// 177.44 * 16 / 14.7868 = 192.0 g.
	portions := []usda.Portion{
		{Amount: 1, Unit: "slice", Modifier: "", GramWeight: 28},
		{Amount: 1, Unit: "tbsp", Modifier: "", GramWeight: 16},
	}
	g, ok := gramsFor(measure.Amount{Kind: measure.Volume, Value: 177.44}, "soy sauce", overrides.Override{}, portions)
	if !ok || g < 191 || g > 193 {
		t.Errorf("volume = %v, %v; want about 192", g, ok)
	}
	// No volume portion at all -> not counted.
	if _, ok := gramsFor(measure.Amount{Kind: measure.Volume, Value: 100}, "x", overrides.Override{},
		[]usda.Portion{{Amount: 1, Unit: "slice", GramWeight: 28}}); ok {
		t.Error("no volume portion must fail")
	}
	// "small" contains the letters "ml" but is not a volume unit.
	if _, ok := gramsFor(measure.Amount{Kind: measure.Volume, Value: 100}, "x", overrides.Override{},
		[]usda.Portion{{Amount: 1, Unit: "undetermined", Modifier: "1 small", GramWeight: 30}}); ok {
		t.Error("'small' must not be read as millilitres")
	}
}

func TestGramsForVolumeIgnoresParentheticalUnits(t *testing.T) {
	// Real FDC portions for "Vegetable oil, palm kernel" (171422). The first
	// is 2 tbsp; its "(1/8 cup)" note must not turn it into 2 cups.
	portions := []usda.Portion{
		{Amount: 2, Unit: "undetermined", Modifier: "tbsp (1/8 cup)", GramWeight: 27},
		{Amount: 1, Unit: "undetermined", Modifier: "tablespoon", GramWeight: 13.6},
		{Amount: 1, Unit: "undetermined", Modifier: "cup", GramWeight: 218},
	}
	// 1 tbs = 14.7868 ml -> 13.5 g
	g, ok := gramsFor(measure.Amount{Kind: measure.Volume, Value: 14.7868}, "vegetable oil", overrides.Override{}, portions)
	if !ok || g < 13.4 || g > 13.6 {
		t.Errorf("1 tbsp oil = %v g, %v; want about 13.5", g, ok)
	}
}

func TestGramsForCount(t *testing.T) {
	// Override itemGrams wins.
	g, ok := gramsFor(measure.Amount{Kind: measure.Count, Value: 2}, "chicken breasts",
		overrides.Override{ItemGrams: 174}, nil)
	if !ok || g != 348 {
		t.Errorf("override count = %v, %v", g, ok)
	}
	// Portion whose text names one item: unit/modifier containing whole,
	// large, medium, fruit, unit, or the name's own last word.
	portions := []usda.Portion{
		{Amount: 1, Unit: "cup", Modifier: "chopped", GramWeight: 140},
		{Amount: 1, Unit: "undetermined", Modifier: "breast, bone removed", GramWeight: 172},
	}
	g, ok = gramsFor(measure.Amount{Kind: measure.Count, Value: 2}, "chicken breasts", overrides.Override{}, portions)
	if !ok || g != 344 {
		t.Errorf("last-word portion = %v, %v; want 344", g, ok)
	}
	g, ok = gramsFor(measure.Amount{Kind: measure.Count, Value: 3},
		"lemon", overrides.Override{}, []usda.Portion{{Amount: 1, Unit: "fruit", GramWeight: 58}})
	if !ok || g != 174 {
		t.Errorf("fruit portion = %v, %v", g, ok)
	}
	// Nothing usable -> not counted.
	if _, ok := gramsFor(measure.Amount{Kind: measure.Count, Value: 1}, "saffron", overrides.Override{},
		[]usda.Portion{{Amount: 1, Unit: "cup", GramWeight: 100}}); ok {
		t.Error("count with no item portion must fail")
	}
}

func TestGramsForCountSkipsPartPortions(t *testing.T) {
	// Real FDC portions for "Onions, raw" (170000), in FDC's order. "slice,
	// medium" names a slice, not a whole onion; the whole medium onion is
	// 110 g.
	portions := []usda.Portion{
		{Amount: 1, Unit: "undetermined", Modifier: "cup, chopped", GramWeight: 160},
		{Amount: 1, Unit: "undetermined", Modifier: `slice, medium (1/8" thick)`, GramWeight: 14},
		{Amount: 1, Unit: "undetermined", Modifier: `medium (2-1/2" dia)`, GramWeight: 110},
		{Amount: 1, Unit: "undetermined", Modifier: "large", GramWeight: 150},
		{Amount: 10, Unit: "undetermined", Modifier: "rings", GramWeight: 60},
		{Amount: 1, Unit: "undetermined", Modifier: "tbsp chopped", GramWeight: 10},
		{Amount: 1, Unit: "undetermined", Modifier: "small", GramWeight: 70},
	}
	g, ok := gramsFor(measure.Amount{Kind: measure.Count, Value: 2}, "onions", overrides.Override{}, portions)
	if !ok || g != 220 {
		t.Errorf("2 onions = %v g, %v; want 220 (2 x medium)", g, ok)
	}
}

func TestGramsForSkipsZeroWeightPortions(t *testing.T) {
	// gramWeight or amount of 0 must be skipped, never divided by.
	bad := []usda.Portion{
		{Amount: 0, Unit: "cup", GramWeight: 0},
		{Amount: 1, Unit: "cup", GramWeight: 0},
	}
	if _, ok := gramsFor(measure.Amount{Kind: measure.Volume, Value: 100}, "x", overrides.Override{}, bad); ok {
		t.Error("zero-weight volume portions must be unusable")
	}
	if _, ok := gramsFor(measure.Amount{Kind: measure.Count, Value: 1}, "lemon", overrides.Override{},
		[]usda.Portion{{Amount: 0, Unit: "fruit", GramWeight: 0}}); ok {
		t.Error("zero-weight count portions must be unusable")
	}
}

func TestGramsForUnmeasurable(t *testing.T) {
	if _, ok := gramsFor(measure.Amount{Kind: measure.Unmeasurable}, "salt", overrides.Override{}, nil); ok {
		t.Error("unmeasurable must never convert")
	}
}

func TestRound1(t *testing.T) {
	if round1(191.2599) != 191.3 || round1(2.04) != 2.0 {
		t.Error("round1 must round to one decimal")
	}
}
