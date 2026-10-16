package fetcher

import (
	"context"
	"net/http"
	"sync"
	"time"
)

const (
	userAgent  = "Mozilla/5.0"
	timeout    = 10 * time.Second
	maxWorkers = 8
)

// Result holds the fetch result for a single symbol.
type Result struct {
	Price              float64
	PrevClose          float64
	Change             float64
	ChangePct          float64
	Time               string
	Epoch              int64     // regularMarketTime epoch seconds
	Series             []float64 // intraday close series (nulls interpolated with previous value)
	Currency           string    // meta.currency from Yahoo
	Source             string    // source that actually supplied this quote
	FetchedAt          time.Time // last successful retrieval, not the market timestamp
	Stale              bool      // a failed refresh returned an expired cache entry
	ChangeUnavailable  bool      // the source did not supply a previous close
	HistoryUnavailable bool      // the source does not supply the requested history
}

// BaseURLOverride is a hook to replace URLs for testing.
var BaseURLOverride []string

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
				Currency           string  `json:"currency"`
			} `json:"meta"`
			Indicators struct {
				Quote []struct {
					Close []*float64 `json:"close"`
				} `json:"quote"`
			} `json:"indicators"`
		} `json:"result"`
	} `json:"chart"`
}

// buildSeries interpolates nulls in the close series with the previous value, returning []float64.
func buildSeries(raw []*float64, fallback float64) []float64 {
	series := make([]float64, 0, len(raw))
	prev := fallback
	for _, v := range raw {
		if v != nil {
			prev = *v
			series = append(series, *v)
		} else if len(series) > 0 || prev != 0 {
			// null: interpolate with previous value
			series = append(series, prev)
		}
	}
	return series
}

func fetchOne(symbol string, client *http.Client) *Result {
	r, _ := (&YahooSource{}).Fetch(context.Background(), symbol, Range1D, client)
	return r
}

// FetchAll fetches multiple symbols in parallel using the requested source policy.
func FetchAll(symbols []string, rng Range, sources ...Source) map[string]*Result {
	return FetchAllContext(context.Background(), symbols, rng, sources...)
}

// FetchAllContext cancels both active HTTP requests and queued work with ctx.
func FetchAllContext(ctx context.Context, symbols []string, rng Range, sources ...Source) map[string]*Result {
	if len(sources) == 0 {
		sources = []Source{&YahooSource{}}
	}
	// Change bases only redefine the previous close, so they apply to 1d only.
	var helpers map[string]bool
	if rng == Range1D {
		symbols, helpers = withChangeBases(symbols)
	}
	client := &http.Client{Timeout: timeout}
	results := make(map[string]*Result)
	var mu sync.Mutex
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup
	useCache := BaseURLOverride == nil && HyperliquidURLOverride == ""

schedule:
	for _, symbol := range symbols {
		if ctx.Err() != nil {
			break
		}
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
			break schedule
		}
		wg.Add(1)
		go func(symbol string) {
			defer wg.Done()
			defer func() { <-sem }()
			sources := sourcesFor(symbol, sources)
			var r *Result
			if useCache {
				if cached, fresh := cacheGet(symbol, rng, sources...); fresh {
					r = cached
				}
			}
			if r == nil {
				for _, src := range sources {
					if ctx.Err() != nil {
						return
					}
					res, err := src.Fetch(ctx, symbol, rng, client)
					if err == nil && res != nil {
						copy := *res
						r = &copy
						r.Source = src.Name()
						r.FetchedAt = time.Now()
						r.Stale = false
						// Only a successful retrieval can renew the cache.
						if useCache && ctx.Err() == nil {
							cacheSet(symbol, rng, r, sources...)
						}
						break
					}
				}
				if r == nil && useCache && ctx.Err() == nil {
					r, _ = cacheGet(symbol, rng, sources...)
				}
			}
			mu.Lock()
			results[symbol] = r
			mu.Unlock()
		}(symbol)
	}
	wg.Wait()
	if rng == Range1D {
		applyChangeBases(results)
	}
	// Drop base symbols that were fetched only to rebase another symbol.
	for sym := range helpers {
		delete(results, sym)
	}
	return results
}
