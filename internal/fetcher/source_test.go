package fetcher

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestYahooSourceFetch(t *testing.T) {
	fixture := `{
  "chart": {
    "result": [{
      "meta": {
        "regularMarketPrice": 39500.5,
        "chartPreviousClose": 39000.0,
        "regularMarketTime": 1718100000,
        "currency": "JPY"
      },
      "indicators": {
        "quote": [{
          "close": [39100.0, null, 39300.0, 39500.5]
        }]
      }
    }]
  }
}`
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Mozilla/5.0" {
			t.Errorf("unexpected UA: %s", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(200)
		w.Write([]byte(fixture))
	}))
	defer ts.Close()

	BaseURLOverride = []string{ts.URL + "/%s"}
	defer func() { BaseURLOverride = nil }()

	src := &YahooSource{}
	r, err := src.Fetch("^N225", Range1D, &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Price != 39500.5 {
		t.Errorf("price: got %f, want 39500.5", r.Price)
	}
	if r.Currency != "JPY" {
		t.Errorf("currency: got %q, want JPY", r.Currency)
	}
}

func TestYahooSourceRangeURL(t *testing.T) {
	var gotURL string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotURL = r.URL.String()
		w.WriteHeader(200)
		w.Write([]byte(fixtureJSON))
	}))
	defer ts.Close()

	// Use BaseURLOverride=nil so real URL construction is used, but intercept via custom transport
	// Instead, we test the yahooURLs helper directly
	BaseURLOverride = nil
	_ = ts
	_ = gotURL

	tests := []struct {
		rng      Range
		interval string
		rngStr   string
	}{
		{Range1D, "5m", "1d"},
		{Range5D, "15m", "5d"},
		{Range1Mo, "1d", "1mo"},
		{Range6Mo, "1d", "6mo"},
		{Range1Y, "1wk", "1y"},
	}
	for _, tt := range tests {
		urls := yahooURLs("TEST", tt.rng)
		for _, u := range urls {
			if !contains(u, "interval="+tt.interval) {
				t.Errorf("range %s: URL %q missing interval=%s", tt.rng, u, tt.interval)
			}
			if !contains(u, "range="+tt.rngStr) {
				t.Errorf("range %s: URL %q missing range=%s", tt.rng, u, tt.rngStr)
			}
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestStooqSourceFetch(t *testing.T) {
	csv := "Symbol,Date,Time,Open,High,Low,Close,Volume\n^dji,2026-06-13,16:00:00,39000.0,39800.0,38900.0,39500.0,350000000\n"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(csv))
	}))
	defer ts.Close()

	StooqBaseURLOverride = ts.URL + "/?s=%s"
	defer func() { StooqBaseURLOverride = "" }()

	src := &StooqSource{}
	r, err := src.Fetch("^DJI", Range1D, &http.Client{})
	if err != nil {
		t.Fatal(err)
	}
	if r.Price != 39500.0 {
		t.Errorf("price: got %f, want 39500.0", r.Price)
	}
	if r.PrevClose != 39000.0 {
		t.Errorf("prevclose: got %f, want 39000.0", r.PrevClose)
	}
	if r.Change != 500.0 {
		t.Errorf("change: got %f, want 500.0", r.Change)
	}
}

func TestStooqSourceUnsupported(t *testing.T) {
	src := &StooqSource{}
	_, err := src.Fetch("UNKNOWN", Range1D, &http.Client{})
	if err == nil {
		t.Fatal("expected error for unsupported symbol")
	}
}

func TestAutoFallback(t *testing.T) {
	// Yahoo fails, Stooq succeeds
	yahooTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(429)
	}))
	defer yahooTs.Close()

	csv := "Symbol,Date,Time,Open,High,Low,Close,Volume\n^dji,2026-06-13,16:00:00,39000.0,39800.0,38900.0,39500.0,350000000\n"
	stooqTs := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(csv))
	}))
	defer stooqTs.Close()

	BaseURLOverride = []string{yahooTs.URL + "/%s"}
	StooqBaseURLOverride = stooqTs.URL + "/?s=%s"
	defer func() {
		BaseURLOverride = nil
		StooqBaseURLOverride = ""
	}()

	sources := []Source{&YahooSource{}, &StooqSource{}}
	results := FetchAll([]string{"^DJI"}, Range1D, sources...)
	r := results["^DJI"]
	if r == nil {
		t.Fatal("expected result from Stooq fallback, got nil")
	}
	if r.Price != 39500.0 {
		t.Errorf("expected stooq price 39500, got %f", r.Price)
	}
}

func TestCacheFreshHit(t *testing.T) {
	dir := t.TempDir()
	CacheDirOverride = dir
	defer func() { CacheDirOverride = "" }()

	// Manually cache a result
	r := &Result{Price: 12345.0, PrevClose: 12300.0, Change: 45.0, ChangePct: 0.37}
	cacheSet("^TEST", Range1D, r)

	// Should get fresh hit
	cached, fresh := cacheGet("^TEST", Range1D)
	if !fresh {
		t.Fatal("expected fresh cache")
	}
	if cached.Price != 12345.0 {
		t.Errorf("expected 12345.0, got %f", cached.Price)
	}
}

func TestCacheStaleOnFailure(t *testing.T) {
	dir := t.TempDir()
	CacheDirOverride = dir
	defer func() { CacheDirOverride = "" }()

	// Write a stale cache entry
	r := &Result{Price: 99999.0}
	entry := fmt.Sprintf(`{"result":{"Price":%f},"timestamp":"%s"}`, r.Price, time.Now().Add(-2*time.Minute).Format(time.RFC3339Nano))
	key := cacheKey("^STALE", Range1D)
	os.MkdirAll(dir, 0o755)
	os.WriteFile(filepath.Join(dir, key+".json"), []byte(entry), 0o644)

	// Test cacheGet returns stale result
	cached, fresh := cacheGet("^STALE", Range1D)
	if fresh {
		t.Fatal("expected stale cache")
	}
	if cached == nil {
		t.Fatal("expected stale cache entry")
	}
	if cached.Price != 99999.0 {
		t.Errorf("expected 99999, got %f", cached.Price)
	}
}

func TestRangeMapping(t *testing.T) {
	tests := []struct {
		rng      Range
		interval string
		yahoo    string
	}{
		{Range1D, "5m", "1d"},
		{Range5D, "15m", "5d"},
		{Range1Mo, "1d", "1mo"},
		{Range6Mo, "1d", "6mo"},
		{Range1Y, "1wk", "1y"},
	}
	for _, tt := range tests {
		if got := tt.rng.Interval(); got != tt.interval {
			t.Errorf("Range %s: Interval()=%s, want %s", tt.rng, got, tt.interval)
		}
		if got := tt.rng.YahooRange(); got != tt.yahoo {
			t.Errorf("Range %s: YahooRange()=%s, want %s", tt.rng, got, tt.yahoo)
		}
	}
}

func TestRangePrevNext(t *testing.T) {
	if Range1D.Prev() != Range1D {
		t.Error("1d.Prev() should be 1d")
	}
	if Range1Y.Next() != Range1Y {
		t.Error("1y.Next() should be 1y")
	}
	if Range1D.Next() != Range5D {
		t.Error("1d.Next() should be 5d")
	}
	if Range5D.Prev() != Range1D {
		t.Error("5d.Prev() should be 1d")
	}
}

func TestParseRange(t *testing.T) {
	tests := []struct {
		s    string
		want Range
	}{
		{"1d", Range1D},
		{"5d", Range5D},
		{"1mo", Range1Mo},
		{"6mo", Range6Mo},
		{"1y", Range1Y},
		{"invalid", Range1D},
	}
	for _, tt := range tests {
		if got := ParseRange(tt.s); got != tt.want {
			t.Errorf("ParseRange(%q)=%v, want %v", tt.s, got, tt.want)
		}
	}
}
