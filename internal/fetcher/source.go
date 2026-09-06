package fetcher

import (
	"context"
	"net/http"
)

// Source is the interface for a market data source.
type Source interface {
	Name() string
	Fetch(ctx context.Context, symbol string, rng Range, client *http.Client) (*Result, error)
}
