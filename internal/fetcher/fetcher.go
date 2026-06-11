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
	"https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d",
	"https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=1d&range=2d",
}

// Result は1銘柄の取得結果
type Result struct {
	Price     float64
	PrevClose float64
	Change    float64
	ChangePct float64
	Time      string
}

// BaseURLOverride はテスト用にURLを差し替えるためのフック
var BaseURLOverride []string

func getBaseURLs() []string {
	if BaseURLOverride != nil {
		return BaseURLOverride
	}
	return baseURLs
}

var jst = time.FixedZone("JST", 9*3600)

type chartResponse struct {
	Chart struct {
		Result []struct {
			Meta struct {
				RegularMarketPrice float64 `json:"regularMarketPrice"`
				ChartPreviousClose float64 `json:"chartPreviousClose"`
				RegularMarketTime  int64   `json:"regularMarketTime"`
			} `json:"meta"`
		} `json:"result"`
	} `json:"chart"`
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
		t := time.Unix(meta.RegularMarketTime, 0).In(jst)
		return &Result{
			Price:     meta.RegularMarketPrice,
			PrevClose: meta.ChartPreviousClose,
			Change:    change,
			ChangePct: pct,
			Time:      t.Format("15:04"),
		}
	}
	return nil
}

// FetchAll は複数銘柄を並列取得する
func FetchAll(symbols []string) map[string]*Result {
	client := &http.Client{Timeout: timeout}
	results := make(map[string]*Result)
	var mu sync.Mutex
	sem := make(chan struct{}, maxWorkers)
	var wg sync.WaitGroup

	for _, s := range symbols {
		wg.Add(1)
		sem <- struct{}{}
		go func(symbol string) {
			defer wg.Done()
			defer func() { <-sem }()
			r := fetchOne(symbol, client)
			mu.Lock()
			results[symbol] = r
			mu.Unlock()
		}(s)
	}
	wg.Wait()
	return results
}
