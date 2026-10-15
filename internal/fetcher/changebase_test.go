package fetcher

import (
	"fmt"
	"math"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func quoteJSON(price, prevClose float64) string {
	return fmt.Sprintf(`{"chart":{"result":[{"meta":{"regularMarketPrice":%v,"chartPreviousClose":%v,"regularMarketTime":1718100000},"indicators":{"quote":[{"close":[%v]}]}}]}}`, price, prevClose, price)
}

// changeBaseServer serves NKD=F (CME settle 66775) and ^N225 (prev close 67744.29).
// If n225Down is true, ^N225 requests fail. It records the requested symbols.
func changeBaseServer(t *testing.T, n225Down bool) (*httptest.Server, func() []string) {
	t.Helper()
	var mu sync.Mutex
	var paths []string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		paths = append(paths, r.URL.Path)
		mu.Unlock()
		switch r.URL.Path {
		case "/NKD=F":
			w.Write([]byte(quoteJSON(64600, 66775)))
		case "/^N225":
			if n225Down {
				w.WriteHeader(500)
				return
			}
			w.Write([]byte(quoteJSON(64425.95, 67744.29)))
		default:
			w.WriteHeader(404)
		}
	}))
	BaseURLOverride = []string{ts.URL + "/%s"}
	t.Cleanup(func() {
		BaseURLOverride = nil
		ts.Close()
	})
	return ts, func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), paths...)
	}
}

func approx(a, b float64) bool { return math.Abs(a-b) < 1e-6 }

func TestChangeBaseNikkeiFutures(t *testing.T) {
	changeBaseServer(t, false)

	results := FetchAll([]string{"^N225", "NKD=F"}, Range1D)
	nkd := results["NKD=F"]
	if nkd == nil {
		t.Fatal("NKD=F: nil result")
	}
	if !approx(nkd.PrevClose, 67744.29) {
		t.Errorf("NKD=F prev_close: got %v, want 67744.29 (^N225 previous close)", nkd.PrevClose)
	}
	if !approx(nkd.Change, 64600-67744.29) {
		t.Errorf("NKD=F change: got %v, want %v", nkd.Change, 64600-67744.29)
	}
	if !approx(nkd.ChangePct, (64600-67744.29)/67744.29*100) {
		t.Errorf("NKD=F change_pct: got %v", nkd.ChangePct)
	}
	n225 := results["^N225"]
	if n225 == nil || !approx(n225.PrevClose, 67744.29) || !approx(n225.Change, 64425.95-67744.29) {
		t.Errorf("^N225 must keep its own change, got %+v", n225)
	}
}

func TestChangeBaseFetchesMissingBase(t *testing.T) {
	_, requested := changeBaseServer(t, false)

	results := FetchAll([]string{"NKD=F"}, Range1D)
	if len(results) != 1 {
		t.Fatalf("results must contain only requested symbols, got %d: %v", len(results), results)
	}
	if nkd := results["NKD=F"]; nkd == nil || !approx(nkd.PrevClose, 67744.29) {
		t.Errorf("NKD=F not rebased on ^N225: %+v", nkd)
	}
	fetchedBase := false
	for _, p := range requested() {
		if p == "/^N225" {
			fetchedBase = true
		}
	}
	if !fetchedBase {
		t.Error("^N225 was not fetched as a change base")
	}
}

func TestChangeBaseOnlyFor1D(t *testing.T) {
	_, requested := changeBaseServer(t, false)

	results := FetchAll([]string{"NKD=F"}, Range5D)
	if nkd := results["NKD=F"]; nkd == nil || !approx(nkd.PrevClose, 66775) || !approx(nkd.Change, 64600-66775) {
		t.Errorf("NKD=F must keep its own range-start base for 5d: %+v", nkd)
	}
	for _, p := range requested() {
		if p == "/^N225" {
			t.Error("^N225 must not be fetched for non-1d ranges")
		}
	}
}

func TestChangeBaseFallsBackWhenBaseUnavailable(t *testing.T) {
	changeBaseServer(t, true)

	results := FetchAll([]string{"NKD=F"}, Range1D)
	if nkd := results["NKD=F"]; nkd == nil || !approx(nkd.PrevClose, 66775) || !approx(nkd.Change, 64600-66775) {
		t.Errorf("NKD=F must keep its own change when ^N225 fails: %+v", nkd)
	}
}

func TestChangeBaseIgnoresBaseWithoutPrevClose(t *testing.T) {
	results := map[string]*Result{
		"NKD=F": {Price: 64600, PrevClose: 66775, Change: -2175},
		"^N225": {Price: 64425.95, ChangeUnavailable: true},
	}
	applyChangeBases(results)
	if nkd := results["NKD=F"]; !approx(nkd.PrevClose, 66775) || !approx(nkd.Change, -2175) {
		t.Errorf("NKD=F must not be rebased on a base without a previous close: %+v", nkd)
	}
}
