package fetcher

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestStooqRejectsMissingAndNonFinitePrices(t *testing.T) {
	for _, csv := range []string{
		"Symbol,Open\n^dji,100\n",
		"Symbol,Close\n^dji,N/D\n",
		"Symbol,Close\n^dji,\n",
		"Symbol,Close\n^dji,NaN\n",
		"Symbol,Close\n^dji,+Inf\n",
		"Symbol,Close\n^dji,-Inf\n",
		"Symbol,Close\n^dji,1e999\n",
		"Symbol,Close\n",
	} {
		t.Run(csv, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, csv) }))
			defer server.Close()
			StooqBaseURLOverride = server.URL + "/?s=%s"
			defer func() { StooqBaseURLOverride = "" }()
			r, err := (&StooqSource{}).Fetch(context.Background(), "^DJI", Range1D, server.Client())
			if err == nil || r != nil {
				t.Fatalf("invalid quote accepted: %+v, %v", r, err)
			}
		})
	}
}

func TestStooqDoesNotInventChangeOrHistory(t *testing.T) {
	// Open is deliberately different from both the price and any prior close.
	csv := "Symbol,Date,Time,Open,High,Low,Close,Volume\n^dji,2026-09-07,16:00:00,90,110,80,100,1000\n"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, csv) }))
	defer server.Close()
	StooqBaseURLOverride = server.URL + "/?s=%s"
	defer func() { StooqBaseURLOverride = "" }()
	for _, rng := range AllRanges {
		r, err := (&StooqSource{}).Fetch(context.Background(), "^DJI", rng, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		if r.Price != 100 || !r.ChangeUnavailable || r.PrevClose != 0 || r.Change != 0 || r.ChangePct != 0 {
			t.Fatalf("range %s: invented previous close/change: %+v", rng, r)
		}
		if !r.HistoryUnavailable || len(r.Series) != 0 {
			t.Fatalf("range %s: invented history: %+v", rng, r)
		}
	}
}
