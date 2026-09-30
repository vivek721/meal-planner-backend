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
