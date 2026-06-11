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

	results := FetchAll([]string{"^N225", "^DJI"})
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for sym, r := range results {
		if r == nil {
			t.Errorf("nil result for %s", sym)
		}
	}
}
