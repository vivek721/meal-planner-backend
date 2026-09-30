// Package usda is a small client for USDA FoodData Central
// (https://fdc.nal.usda.gov/) that turns its responses into clean Go types.
package usda

import "context"

// Nutrient names a tracked nutrient; the string value is its JSON name in
// API responses.
type Nutrient string

// The seven tracked nutrients. Sodium is in mg; the rest are grams, except
// Calories (kcal). All are per 100 g of food.
const (
	Calories     Nutrient = "calories"
	Protein      Nutrient = "protein"
	Carbohydrate Nutrient = "carbohydrate"
	Fat          Nutrient = "fat"
	Fiber        Nutrient = "fiber"
	Sugars       Nutrient = "sugars"
	Sodium       Nutrient = "sodium"
)

// All lists every tracked nutrient, in display order.
var All = []Nutrient{Calories, Protein, Carbohydrate, Fat, Fiber, Sugars, Sodium}

// Nutrients holds per-100 g values. An absent key means FDC did not report
// that nutrient — absent, not zero.
type Nutrients map[Nutrient]float64

// Portion is one of FDC's household portions with its gram weight.
type Portion struct {
	Amount     float64 `json:"amount"`
	Unit       string  `json:"unit"`
	Modifier   string  `json:"modifier"`
	GramWeight float64 `json:"gramWeight"`
}

// SearchFood is one search result.
type SearchFood struct {
	FDCID       int    `json:"fdcId"`
	Description string `json:"description"`
	DataType    string `json:"dataType"`
}

// Food is a full food record. It is cached as JSON, so tags are stable.
type Food struct {
	FDCID       int       `json:"fdcId"`
	Description string    `json:"description"`
	DataType    string    `json:"dataType"`
	Per100g     Nutrients `json:"per100g"`
	Portions    []Portion `json:"portions"`
}

// Client fetches foods from FDC.
type Client interface {
	Search(ctx context.Context, query string) ([]SearchFood, error)
	Foods(ctx context.Context, ids []int) ([]Food, error)
}
