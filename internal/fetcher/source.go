package fetcher

import "net/http"

// Source is the interface for a market data source.
type Source interface {
	Name() string
	Fetch(symbol string, rng Range, client *http.Client) (*Result, error)
}
