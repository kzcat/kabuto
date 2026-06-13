package fetcher

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

const fixtureJSON = `{
  "chart": {
    "result": [{
      "meta": {
        "regularMarketPrice": 39500.5,
        "chartPreviousClose": 39000.0,
        "regularMarketTime": 1718100000
      },
      "indicators": {
        "quote": [{
          "close": [39100.0, null, 39300.0, 39500.5]
        }]
      }
    }]
  }
}`

func TestFetchOne(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != "Mozilla/5.0" {
			t.Errorf("unexpected UA: %s", r.Header.Get("User-Agent"))
		}
		w.WriteHeader(200)
		w.Write([]byte(fixtureJSON))
	}))
	defer ts.Close()

	BaseURLOverride = []string{ts.URL + "/%s"}
	defer func() { BaseURLOverride = nil }()

	client := &http.Client{}
	r := fetchOne("^N225", client)
	if r == nil {
		t.Fatal("expected result, got nil")
	}
	if r.Price != 39500.5 {
		t.Errorf("price: got %f, want 39500.5", r.Price)
	}
	if r.PrevClose != 39000.0 {
		t.Errorf("prev_close: got %f, want 39000.0", r.PrevClose)
	}
	expectedChange := 500.5
	if r.Change != expectedChange {
		t.Errorf("change: got %f, want %f", r.Change, expectedChange)
	}
	if r.Epoch != 1718100000 {
		t.Errorf("epoch: got %d, want 1718100000", r.Epoch)
	}
}

func TestFetchOneSeries(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(fixtureJSON))
	}))
	defer ts.Close()

	BaseURLOverride = []string{ts.URL + "/%s"}
	defer func() { BaseURLOverride = nil }()

	r := fetchOne("^N225", &http.Client{})
	if r == nil {
		t.Fatal("expected result")
	}
	// close は [39100, null, 39300, 39500.5] → null は前値 39100 で補間
	want := []float64{39100.0, 39100.0, 39300.0, 39500.5}
	if len(r.Series) != len(want) {
		t.Fatalf("series length: got %d, want %d (%v)", len(r.Series), len(want), r.Series)
	}
	for i := range want {
		if r.Series[i] != want[i] {
			t.Errorf("series[%d]: got %f, want %f", i, r.Series[i], want[i])
		}
	}
}

func TestBuildSeriesInterpolation(t *testing.T) {
	f := func(v float64) *float64 { return &v }
	raw := []*float64{nil, f(10), nil, nil, f(20)}
	// 先頭 null は fallback(100)で補間、以降は前値で補間
	got := buildSeries(raw, 100)
	want := []float64{100, 10, 10, 10, 20}
	if len(got) != len(want) {
		t.Fatalf("length: got %d want %d (%v)", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("[%d]: got %f want %f", i, got[i], want[i])
		}
	}
}

func TestFetchOneFallback(t *testing.T) {
	call := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call++
		if call == 1 {
			w.WriteHeader(429)
			return
		}
		w.WriteHeader(200)
		w.Write([]byte(fixtureJSON))
	}))
	defer ts.Close()

	BaseURLOverride = []string{ts.URL + "/q2/%s", ts.URL + "/q1/%s"}
	defer func() { BaseURLOverride = nil }()

	client := &http.Client{}
	r := fetchOne("^N225", client)
	if r == nil {
		t.Fatal("expected result after fallback, got nil")
	}
	if call != 2 {
		t.Errorf("expected 2 calls, got %d", call)
	}
}

func TestFetchAll(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(fixtureJSON))
	}))
	defer ts.Close()

	BaseURLOverride = []string{ts.URL + "/%s"}
	defer func() { BaseURLOverride = nil }()

	results := FetchAll([]string{"^N225", "^DJI"}, Range1D)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for sym, r := range results {
		if r == nil {
			t.Errorf("nil result for %s", sym)
		}
	}
}
