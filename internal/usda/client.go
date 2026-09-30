package usda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// maxBodyBytes caps how much of an FDC response is read.
const maxBodyBytes = 5 << 20

// HTTPClient implements Client over FDC's JSON API.
type HTTPClient struct {
	baseURL string
	apiKey  string
	http    *http.Client
}

// NewHTTPClient returns a client for baseURL (e.g.
// https://api.nal.usda.gov/fdc/v1). The key is sent as the api_key query
// parameter and never logged.
func NewHTTPClient(baseURL, apiKey string, timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		apiKey:  apiKey,
		http:    &http.Client{Timeout: timeout},
	}
}

// do sends a request to path and returns the body of a 200 response. Errors
// name the path only, never the full URL, so the key cannot leak into logs.
func (c *HTTPClient) do(ctx context.Context, method, path string, params url.Values, body []byte) ([]byte, error) {
	params.Set("api_key", c.apiKey)
	endpoint := c.baseURL + path + "?" + params.Encode()
	var reader io.Reader = http.NoBody
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return nil, fmt.Errorf("usda: build request for %s: %w", path, err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("usda: %s: %w", path, sanitizeURLError(err, endpoint, c.baseURL+path))
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			log.Printf("usda: close %s: %v", path, cerr)
		}
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("usda: %s returned status %d", path, resp.StatusCode)
	}
	out, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes))
	if err != nil {
		return nil, fmt.Errorf("usda: read %s: %w", path, err)
	}
	return out, nil
}

// sanitizeURLError strips the query string (which holds api_key) from
// transport errors, which embed the full request URL.
func sanitizeURLError(err error, full, safe string) error {
	return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), full, safe))
}

// Search returns generic foods matching query, best match first. Branded and
// survey foods are excluded: they are product- or survey-specific, not
// generic ingredients.
func (c *HTTPClient) Search(ctx context.Context, query string) ([]SearchFood, error) {
	params := url.Values{
		"query":    {query},
		"dataType": {"Foundation,SR Legacy"},
		"pageSize": {"10"},
	}
	body, err := c.do(ctx, http.MethodGet, "/foods/search", params, nil)
	if err != nil {
		return nil, err
	}
	var env struct {
		Foods []SearchFood `json:"foods"`
	}
	if err := json.Unmarshal(body, &env); err != nil {
		return nil, fmt.Errorf("usda: decode search: %w", err)
	}
	return env.Foods, nil
}

// foodsBatchSize is FDC's maximum ids per /foods request.
const foodsBatchSize = 20

// nutrientNumbers maps FDC nutrient numbers to tracked nutrients. Energy has
// Atwater fallbacks handled separately in toFood.
var nutrientNumbers = map[string]Nutrient{
	"203": Protein, "204": Fat, "205": Carbohydrate,
	"291": Fiber, "269": Sugars, "307": Sodium,
}

type rawFood struct {
	FDCID         int    `json:"fdcId"`
	Description   string `json:"description"`
	DataType      string `json:"dataType"`
	FoodNutrients []struct {
		Nutrient struct {
			Number string `json:"number"`
		} `json:"nutrient"`
		Amount *float64 `json:"amount"`
	} `json:"foodNutrients"`
	FoodPortions []struct {
		Amount      float64 `json:"amount"`
		GramWeight  float64 `json:"gramWeight"`
		Modifier    string  `json:"modifier"`
		MeasureUnit struct {
			Name         string `json:"name"`
			Abbreviation string `json:"abbreviation"`
		} `json:"measureUnit"`
	} `json:"foodPortions"`
}

// Foods returns full records for ids, splitting requests into batches of 20.
func (c *HTTPClient) Foods(ctx context.Context, ids []int) ([]Food, error) {
	out := make([]Food, 0, len(ids))
	for start := 0; start < len(ids); start += foodsBatchSize {
		end := min(start+foodsBatchSize, len(ids))
		body, err := json.Marshal(map[string]any{"fdcIds": ids[start:end], "format": "full"})
		if err != nil {
			return nil, fmt.Errorf("usda: encode foods request: %w", err)
		}
		resp, err := c.do(ctx, http.MethodPost, "/foods", url.Values{}, body)
		if err != nil {
			return nil, err
		}
		var raw []rawFood
		if err := json.Unmarshal(resp, &raw); err != nil {
			return nil, fmt.Errorf("usda: decode foods: %w", err)
		}
		for _, r := range raw {
			out = append(out, toFood(r))
		}
	}
	return out, nil
}

// toFood maps a raw record: nutrients by number (energy 208 falling back to
// 958 then 957) into Per100g, portions with the unit abbreviation or name.
func toFood(r rawFood) Food {
	per := Nutrients{}
	var atwaterSpecific, atwaterGeneral *float64
	for _, fn := range r.FoodNutrients {
		if fn.Amount == nil {
			continue
		}
		switch fn.Nutrient.Number {
		case "208":
			per[Calories] = *fn.Amount
		case "958":
			atwaterSpecific = fn.Amount
		case "957":
			atwaterGeneral = fn.Amount
		default:
			if n, ok := nutrientNumbers[fn.Nutrient.Number]; ok {
				per[n] = *fn.Amount
			}
		}
	}
	if _, ok := per[Calories]; !ok {
		if atwaterSpecific != nil {
			per[Calories] = *atwaterSpecific
		} else if atwaterGeneral != nil {
			per[Calories] = *atwaterGeneral
		}
	}
	portions := make([]Portion, 0, len(r.FoodPortions))
	for _, p := range r.FoodPortions {
		unit := p.MeasureUnit.Abbreviation
		if unit == "" {
			unit = p.MeasureUnit.Name
		}
		portions = append(portions, Portion{
			Amount: p.Amount, Unit: unit, Modifier: p.Modifier, GramWeight: p.GramWeight,
		})
	}
	return Food{
		FDCID: r.FDCID, Description: r.Description, DataType: r.DataType,
		Per100g: per, Portions: portions,
	}
}
