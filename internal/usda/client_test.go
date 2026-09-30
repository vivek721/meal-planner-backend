package usda

import (
	"context"
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
