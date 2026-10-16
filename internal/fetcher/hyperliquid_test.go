package fetcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

const hlMetaFixture = `[
  {"universe":[{"name":"xyz:JPY"},{"name":"xyz:JP225"}]},
  [{"oraclePx":"157.97","prevDayPx":"157.66"},{"oraclePx":"65487.0","prevDayPx":"64545.0","markPx":"65520.0"}]
]`

const hlCandleFixture = `[
  {"t":1792130000000,"o":"64500.0","c":"64600.0"},
  {"t":1792130300000,"o":"64600.0","c":"65510.0"},
  {"t":1792130600000,"o":"65510.0","c":"65502.0"}
]`

// hlServer serves the HyperLiquid info endpoint and records request bodies.
func hlServer(t *testing.T, meta string) func() []map[string]any {
	t.Helper()
	var mu sync.Mutex
	var bodies []map[string]any
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("unexpected method %s", r.Method)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("bad request body: %v", err)
		}
		mu.Lock()
		bodies = append(bodies, body)
		mu.Unlock()
		switch body["type"] {
		case "metaAndAssetCtxs":
			w.Write([]byte(meta))
		case "candleSnapshot":
			w.Write([]byte(hlCandleFixture))
		default:
			w.WriteHeader(400)
		}
	}))
	HyperliquidURLOverride = ts.URL
	t.Cleanup(func() {
		HyperliquidURLOverride = ""
		ts.Close()
	})
	return func() []map[string]any {
		mu.Lock()
		defer mu.Unlock()
		return append([]map[string]any(nil), bodies...)
	}
}

func TestHyperliquidFetch1D(t *testing.T) {
	requests := hlServer(t, hlMetaFixture)

	r, err := (&HyperliquidSource{}).Fetch(context.Background(), "hl:xyz:JP225", Range1D, &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Price != 65487 {
		t.Errorf("price: got %v, want oracle 65487", r.Price)
	}
	if r.PrevClose != 64545 || r.Change != 65487-64545 {
		t.Errorf("1d must compare with the price 24h ago: %+v", r)
	}
	want := []float64{64600, 65510, 65502}
	if len(r.Series) != len(want) {
		t.Fatalf("series: got %v, want %v", r.Series, want)
	}
	for i := range want {
		if r.Series[i] != want[i] {
			t.Errorf("series[%d]: got %v, want %v", i, r.Series[i], want[i])
		}
	}
	if r.Currency != "JPY" || r.Source != "hyperliquid" {
		t.Errorf("currency/source: got %q/%q", r.Currency, r.Source)
	}

	reqs := requests()
	if len(reqs) != 2 || reqs[0]["dex"] != "xyz" {
		t.Fatalf("unexpected requests: %v", reqs)
	}
	candle, _ := reqs[1]["req"].(map[string]any)
	if candle["coin"] != "xyz:JP225" || candle["interval"] != "5m" {
		t.Errorf("unexpected candle request: %v", candle)
	}
}

func TestHyperliquidFetchLongerRangeUsesWindowOpen(t *testing.T) {
	requests := hlServer(t, hlMetaFixture)

	r, err := (&HyperliquidSource{}).Fetch(context.Background(), "hl:xyz:JP225", Range5D, &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if r.PrevClose != 64500 || r.Change != 65487-64500 {
		t.Errorf("5d must compare with the window open: %+v", r)
	}
	candle, _ := requests()[1]["req"].(map[string]any)
	if candle["interval"] != "15m" {
		t.Errorf("5d interval: got %v, want 15m", candle["interval"])
	}
}

func TestHyperliquidFetchErrors(t *testing.T) {
	tests := []struct {
		name, symbol, meta string
	}{
		{"not listed", "hl:xyz:NOPE", hlMetaFixture},
		{"no dex", "hl:JP225", hlMetaFixture},
		{"invalid oracle", "hl:xyz:JP225", `[{"universe":[{"name":"xyz:JP225"}]},[{"oraclePx":"0","prevDayPx":"1"}]]`},
		{"malformed", "hl:xyz:JP225", `null`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hlServer(t, tt.meta)
			if _, err := (&HyperliquidSource{}).Fetch(context.Background(), tt.symbol, Range1D, &http.Client{}); err == nil {
				t.Error("expected error")
			}
		})
	}
}

func TestFetchAllRoutesHyperliquidAndRebases(t *testing.T) {
	hlServer(t, hlMetaFixture)
	changeBaseServer(t, false) // Yahoo: ^N225 previous close 67744.29

	results := FetchAll([]string{"hl:xyz:JP225"}, Range1D, &YahooSource{})
	if len(results) != 1 {
		t.Fatalf("results must contain only requested symbols, got %v", results)
	}
	r := results["hl:xyz:JP225"]
	if r == nil || r.Source != "hyperliquid" {
		t.Fatalf("hl symbol must be fetched from HyperLiquid regardless of sources: %+v", r)
	}
	if !approx(r.PrevClose, 67744.29) || !approx(r.Change, 65487-67744.29) {
		t.Errorf("1d change must be rebased on ^N225 previous close: %+v", r)
	}
}
