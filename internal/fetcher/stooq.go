package fetcher

import (
	"encoding/csv"
	"fmt"
	"net/http"
	"strconv"
	"strings"
)

// StooqBaseURLOverride is a test hook for Stooq (full URL with %s for symbol).
var StooqBaseURLOverride string

// stooqSymbolMap maps Yahoo symbols to Stooq symbols (major indices only).
var stooqSymbolMap = map[string]string{
	"^DJI":   "^dji",
	"^GSPC":  "^spx",
	"^IXIC":  "^ndq",
	"^N225":  "^nkx",
	"^FTSE":  "^ukx",
	"^GDAXI": "^dax",
	"^FCHI":  "^cac",
	"^HSI":   "^hsi",
	"^AXJO":  "^aord",
	"^KS11":  "^kospi",
}

// StooqSource fetches data from Stooq's CSV endpoint.
type StooqSource struct{}

func (s *StooqSource) Name() string { return "stooq" }

func (s *StooqSource) Fetch(symbol string, rng Range, client *http.Client) (*Result, error) {
	stooqSym, ok := stooqSymbolMap[symbol]
	if !ok {
		return nil, fmt.Errorf("stooq: unsupported symbol %s", symbol)
	}

	var u string
	if StooqBaseURLOverride != "" {
		u = fmt.Sprintf(StooqBaseURLOverride, stooqSym)
	} else {
		u = fmt.Sprintf("https://stooq.com/q/l/?s=%s&f=sd2t2ohlcv&e=csv", stooqSym)
	}

	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("stooq: status %d", resp.StatusCode)
	}

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("stooq: csv parse error: %w", err)
	}
	if len(records) < 2 {
		return nil, fmt.Errorf("stooq: no data rows")
	}

	// Header: Symbol,Date,Time,Open,High,Low,Close,Volume
	// Find column indices from header
	header := records[0]
	colIdx := map[string]int{}
	for i, h := range header {
		colIdx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	row := records[1]

	getFloat := func(name string) float64 {
		idx, ok := colIdx[name]
		if !ok || idx >= len(row) {
			return 0
		}
		v, _ := strconv.ParseFloat(strings.TrimSpace(row[idx]), 64)
		return v
	}

	open := getFloat("open")
	close_ := getFloat("close")

	change := close_ - open
	var pct float64
	if open != 0 {
		pct = change / open * 100
	}

	return &Result{
		Price:     close_,
		PrevClose: open,
		Change:    change,
		ChangePct: pct,
		Series:    []float64{open, close_},
		Currency:  "",
	}, nil
}
