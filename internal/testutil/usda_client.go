package testutil

import (
	"context"
	"sync"
	"time"

	"github.com/meal-planner/backend/internal/usda"
)

// USDAClient is a fake usda.Client backed by maps. It is safe for concurrent
// use; set its fields before the calls under test start.
type USDAClient struct {
	SearchResults map[string][]usda.SearchFood // keyed by query
	FoodsByID     map[int]usda.Food
	// Err, when set, is returned by every method; SearchErr and FoodsErr,
	// when set, are returned by that method only.
	Err, SearchErr, FoodsErr error
	// SearchCalls and FoodsCalls count calls. MaxConcurrentSearches records
	// the highest number of Search calls in flight at once.
	SearchCalls, FoodsCalls int
	MaxConcurrentSearches   int

	mu       sync.Mutex
	inFlight int
}

// NewUSDAClient returns an empty fake client.
func NewUSDAClient() *USDAClient {
	return &USDAClient{
		SearchResults: map[string][]usda.SearchFood{},
		FoodsByID:     map[int]usda.Food{},
	}
}

// Search returns SearchResults[query] (an empty list when absent). It sleeps
// briefly so overlapping calls are observable in MaxConcurrentSearches.
func (c *USDAClient) Search(_ context.Context, query string) ([]usda.SearchFood, error) {
	c.mu.Lock()
	c.SearchCalls++
	c.inFlight++
	if c.inFlight > c.MaxConcurrentSearches {
		c.MaxConcurrentSearches = c.inFlight
	}
	err := c.Err
	if err == nil {
		err = c.SearchErr
	}
	results := c.SearchResults[query]
	c.mu.Unlock()

	time.Sleep(2 * time.Millisecond)

	c.mu.Lock()
	c.inFlight--
	c.mu.Unlock()

	if err != nil {
		return nil, err
	}
	if results == nil {
		return []usda.SearchFood{}, nil
	}
	return results, nil
}

// Foods returns FoodsByID[id] for each requested id that exists; ids FDC
// does not know are simply omitted, as the real API does.
func (c *USDAClient) Foods(_ context.Context, ids []int) ([]usda.Food, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.FoodsCalls++
	if c.Err != nil {
		return nil, c.Err
	}
	if c.FoodsErr != nil {
		return nil, c.FoodsErr
	}
	out := make([]usda.Food, 0, len(ids))
	for _, id := range ids {
		if f, ok := c.FoodsByID[id]; ok {
			out = append(out, f)
		}
	}
	return out, nil
}
