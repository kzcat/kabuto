package fetcher

import (
	"context"
	"net/http"
	"strings"
)

// Source is the interface for a market data source.
type Source interface {
	Name() string
	Fetch(ctx context.Context, symbol string, rng Range, client *http.Client) (*Result, error)
}

// sourcesFor returns the sources to try for symbol. HyperLiquid symbols are
// only available from HyperLiquid, whatever the source policy.
func sourcesFor(symbol string, sources []Source) []Source {
	if strings.HasPrefix(symbol, hlPrefix) {
		return []Source{&HyperliquidSource{}}
	}
	return sources
}
