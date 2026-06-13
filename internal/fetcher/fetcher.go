package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	userAgent  = "Mozilla/5.0"
	timeout    = 10 * time.Second
	maxWorkers = 8
)

var baseURLs = []string{
	"https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=5m&range=1d",
	"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=5m&range=1d",
}

// Result は1銘柄の取得結果
type Result struct {
	Price     float64
	PrevClose float64
	Change    float64
	ChangePct float64
	Time      string
	Epoch     int64     // regularMarketTime の epoch 秒
	Series    []float64 // intraday の終値系列(null は前値で補間)
	Currency  string    // meta.currency from Yahoo
}

// BaseURLOverride はテスト用にURLを差し替えるためのフック
var BaseURLOverride []string

func getBaseURLs() []string {
	if BaseURLOverride != nil {
		return BaseURLOverride
	}
	return baseURLs
}

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

// buildSeries は close 系列の null を前値で補間して []float64 にする
func buildSeries(raw []*float64, fallback float64) []float64 {
	series := make([]float64, 0, len(raw))
	prev := fallback
	for _, v := range raw {
		if v != nil {
			prev = *v
			series = append(series, *v)
		} else if len(series) > 0 || prev != 0 {
			// null は前値で補間
			series = append(series, prev)
		}
	}
	return series
}

func fetchOne(symbol string, client *http.Client) *Result {
	encoded := url.PathEscape(symbol)
	for _, urlTpl := range getBaseURLs() {
		u := fmt.Sprintf(urlTpl, encoded)
		req, err := http.NewRequest("GET", u, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", userAgent)
		resp, err := client.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()
		if resp.StatusCode != 200 {
			continue
		}
		var cr chartResponse
		if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
			continue
		}
		if len(cr.Chart.Result) == 0 {
			continue
		}
		meta := cr.Chart.Result[0].Meta
		change := meta.RegularMarketPrice - meta.ChartPreviousClose
		var pct float64
		if meta.ChartPreviousClose != 0 {
			pct = change / meta.ChartPreviousClose * 100
		}
		t := time.Unix(meta.RegularMarketTime, 0).In(time.Local)
		var series []float64
		if len(cr.Chart.Result[0].Indicators.Quote) > 0 {
			series = buildSeries(cr.Chart.Result[0].Indicators.Quote[0].Close, meta.ChartPreviousClose)
		}
		return &Result{
			Price:     meta.RegularMarketPrice,
			PrevClose: meta.ChartPreviousClose,
			Change:    change,
			ChangePct: pct,
			Time:      t.Format("15:04"),
			Epoch:     meta.RegularMarketTime,
			Series:    series,
			Currency:  meta.Currency,
		}
	}
	return nil
}

// FetchAll は複数銘柄を並列取得する (Range-aware, Source-aware)
func FetchAll(symbols []string, rng Range, sources ...Source) map[string]*Result {
	client := &http.Client{Timeout: timeout}
	results := make(map[string]*Result)
	var mu sync.Mutex
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	// Disable cache when BaseURLOverride is set (test mode)
	useCache := BaseURLOverride == nil

	for _, s := range symbols {
		wg.Add(1)
		sem <- struct{}{}
		go func(symbol string) {
			defer wg.Done()
			defer func() { <-sem }()

			var r *Result

			// Check cache first
			if useCache {
				if cached, fresh := cacheGet(symbol, rng); fresh && cached != nil {
					mu.Lock()
					results[symbol] = cached
					mu.Unlock()
					return
				}
			}

			if len(sources) > 0 {
				for _, src := range sources {
					res, err := src.Fetch(symbol, rng, client)
					if err == nil && res != nil {
						r = res
						break
					}
				}
			} else {
				// Legacy path: use fetchOne (for backward compat with existing tests)
				r = fetchOne(symbol, client)
			}

			// On failure, try stale cache
			if r == nil && useCache {
				if cached, _ := cacheGet(symbol, rng); cached != nil {
					r = cached
				}
			}

			// Update cache on success
			if r != nil && useCache {
				cacheSet(symbol, rng, r)
			}

			mu.Lock()
			results[symbol] = r
			mu.Unlock()
		}(s)
	}
	wg.Wait()
	return results
}
