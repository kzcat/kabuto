package fetcher

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// YahooSource fetches data from Yahoo Finance (query2 then query1 fallback).
type YahooSource struct{}

func (y *YahooSource) Name() string { return "yahoo" }

func (y *YahooSource) Fetch(symbol string, rng Range, client *http.Client) (*Result, error) {
	encoded := url.PathEscape(symbol)
	urls := yahooURLs(encoded, rng)
	for _, u := range urls {
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
		}, nil
	}
	return nil, fmt.Errorf("yahoo: all URLs failed for %s", symbol)
}

func yahooURLs(encodedSymbol string, rng Range) []string {
	if BaseURLOverride != nil {
		// Test hook: use override URLs as-is (they have hardcoded interval/range)
		urls := make([]string, len(BaseURLOverride))
		for i, tpl := range BaseURLOverride {
			urls[i] = fmt.Sprintf(tpl, encodedSymbol)
		}
		return urls
	}
	interval := rng.Interval()
	yahooRange := rng.YahooRange()
	return []string{
		fmt.Sprintf("https://query2.finance.yahoo.com/v8/finance/chart/%s?interval=%s&range=%s", encodedSymbol, interval, yahooRange),
		fmt.Sprintf("https://query1.finance.yahoo.com/v8/finance/chart/%s?interval=%s&range=%s", encodedSymbol, interval, yahooRange),
	}
}
