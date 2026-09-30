package usda

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func newTestClient(handler http.HandlerFunc) (*HTTPClient, *httptest.Server) {
	srv := httptest.NewServer(handler)
	return NewHTTPClient(srv.URL, "test-key", 2*time.Second), srv
}

func TestSearchSendsFiltersAndKey(t *testing.T) {
	var gotPath, gotQuery string
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		_, _ = w.Write([]byte(`{"foods":[
			{"fdcId":174277,"description":"Soy sauce made from soy and wheat (shoyu)","dataType":"SR Legacy"},
			{"fdcId":999,"description":"Something branded","dataType":"SR Legacy"}]}`))
	})
	defer srv.Close()

	got, err := c.Search(context.Background(), "soy sauce")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	want := []SearchFood{
		{FDCID: 174277, Description: "Soy sauce made from soy and wheat (shoyu)", DataType: "SR Legacy"},
		{FDCID: 999, Description: "Something branded", DataType: "SR Legacy"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Search = %+v, want %+v", got, want)
	}
	if gotPath != "/foods/search" {
		t.Errorf("path = %q", gotPath)
	}
	for _, part := range []string{"api_key=test-key", "dataType=Foundation%2CSR+Legacy", "pageSize=10", "query=soy+sauce"} {
		if !strings.Contains(gotQuery, part) {
			t.Errorf("query %q missing %q", gotQuery, part)
		}
	}
}

func TestSearchErrorsHideTheKey(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	})
	defer srv.Close()

	_, err := c.Search(context.Background(), "salt")
	if err == nil {
		t.Fatal("want error on 429")
	}
	if !strings.Contains(err.Error(), "429") {
		t.Errorf("err = %v, want the status code", err)
	}
	if strings.Contains(err.Error(), "test-key") {
		t.Errorf("err = %v, must not contain the API key", err)
	}
}

func TestSearchMalformedJSON(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{not json`))
	})
	defer srv.Close()
	if _, err := c.Search(context.Background(), "salt"); err == nil {
		t.Fatal("want error on malformed JSON")
	}
}

func TestSearchTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	c := NewHTTPClient(srv.URL, "test-key", 20*time.Millisecond)
	_, err := c.Search(context.Background(), "salt")
	if err == nil {
		t.Fatal("want error on timeout")
	}
	if strings.Contains(err.Error(), "test-key") {
		t.Errorf("timeout err = %v, must not contain the API key", err)
	}
}

func TestFoodsMapsNutrientsAndPortions(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/foods" {
			t.Errorf("got %s %s", r.Method, r.URL.Path)
		}
		var req struct {
			FDCIDs []int  `json:"fdcIds"`
			Format string `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Format != "full" {
			t.Errorf("bad body: %+v, %v", req, err)
		}
		_, _ = w.Write([]byte(`[{
			"fdcId":174277,"description":"Soy sauce (shoyu)","dataType":"SR Legacy",
			"foodNutrients":[
				{"nutrient":{"number":"208"},"amount":53.0},
				{"nutrient":{"number":"203"},"amount":8.14},
				{"nutrient":{"number":"204"},"amount":0.57},
				{"nutrient":{"number":"205"},"amount":4.93},
				{"nutrient":{"number":"307"},"amount":5493.0},
				{"nutrient":{"number":"999"},"amount":1.0},
				{"nutrient":{"number":"291"}}
			],
			"foodPortions":[
				{"amount":1,"gramWeight":16.0,"modifier":"","measureUnit":{"name":"tablespoon","abbreviation":"tbsp"}},
				{"amount":1,"gramWeight":255.0,"modifier":"cup","measureUnit":{"name":"undetermined","abbreviation":"undetermined"}}
			]}]`))
	})
	defer srv.Close()

	foods, err := c.Foods(context.Background(), []int{174277})
	if err != nil {
		t.Fatalf("Foods: %v", err)
	}
	f := foods[0]
	want := Nutrients{Calories: 53.0, Protein: 8.14, Fat: 0.57, Carbohydrate: 4.93, Sodium: 5493.0}
	if !reflect.DeepEqual(f.Per100g, want) {
		t.Errorf("Per100g = %v, want %v (fiber had no amount: absent; 999: ignored)", f.Per100g, want)
	}
	wantPortions := []Portion{
		{Amount: 1, Unit: "tbsp", Modifier: "", GramWeight: 16},
		{Amount: 1, Unit: "undetermined", Modifier: "cup", GramWeight: 255},
	}
	if !reflect.DeepEqual(f.Portions, wantPortions) {
		t.Errorf("Portions = %+v, want %+v", f.Portions, wantPortions)
	}
}

func TestFoodsEnergyFallback(t *testing.T) {
	cases := []struct {
		name, nutrients string
		want            Nutrients
	}{
		{"958 preferred over 957",
			`[{"nutrient":{"number":"957"},"amount":100.0},{"nutrient":{"number":"958"},"amount":90.0}]`,
			Nutrients{Calories: 90}},
		{"957 alone",
			`[{"nutrient":{"number":"957"},"amount":100.0}]`,
			Nutrients{Calories: 100}},
		{"208 wins over both",
			`[{"nutrient":{"number":"208"},"amount":80.0},{"nutrient":{"number":"958"},"amount":90.0}]`,
			Nutrients{Calories: 80}},
		{"none: calories absent", `[]`, Nutrients{}},
	}
	for _, tc := range cases {
		c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(`[{"fdcId":1,"description":"x","dataType":"Foundation","foodNutrients":` + tc.nutrients + `,"foodPortions":[]}]`))
		})
		foods, err := c.Foods(context.Background(), []int{1})
		srv.Close()
		if err != nil || !reflect.DeepEqual(foods[0].Per100g, tc.want) {
			t.Errorf("%s: Per100g = %v, %v; want %v", tc.name, foods[0].Per100g, err, tc.want)
		}
	}
}

func TestFoodsBatchesOver20IDs(t *testing.T) {
	var batches [][]int
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			FDCIDs []int `json:"fdcIds"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		batches = append(batches, req.FDCIDs)
		_, _ = w.Write([]byte(`[]`))
	})
	defer srv.Close()

	ids := make([]int, 25)
	for i := range ids {
		ids[i] = i + 1
	}
	if _, err := c.Foods(context.Background(), ids); err != nil {
		t.Fatalf("Foods: %v", err)
	}
	if len(batches) != 2 || len(batches[0]) != 20 || len(batches[1]) != 5 {
		t.Errorf("batches = %v, want sizes [20 5]", batches)
	}
}

func TestFoodsUpstreamError(t *testing.T) {
	c, srv := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	defer srv.Close()
	if _, err := c.Foods(context.Background(), []int{1}); err == nil {
		t.Fatal("want error on 500")
	}
}
