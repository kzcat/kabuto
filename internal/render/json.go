package render

import (
	"encoding/json"
	"math"
	"time"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/i18n"
	"github.com/kzcat/kabuto/internal/symbols"
)

// JSONItem represents a single symbol in JSON output.
type JSONItem struct {
	Name             string     `json:"name"`
	Symbol           string     `json:"symbol"`
	Country          string     `json:"country"`
	Price            *float64   `json:"price"`
	Change           *float64   `json:"change"`
	ChangePct        *float64   `json:"change_pct"`
	Time             *string    `json:"time"`
	Epoch            *int64     `json:"epoch"`
	Series           []float64  `json:"series"`
	PrevClose        *float64   `json:"prev_close"`
	Source           string     `json:"source,omitempty"`
	FetchedAt        *time.Time `json:"fetched_at,omitempty"`
	Stale            bool       `json:"stale"`
	HistoryAvailable bool       `json:"history_available"`
}

// JSONSection represents a single section in JSON output.
type JSONSection struct {
	Title string     `json:"title"`
	Items []JSONItem `json:"items"`
}

// RenderJSON generates the JSON output string. Times are formatted according to loc (nil = time.Local).
func RenderJSON(data map[string]*fetcher.Result, sections []string, loc *time.Location, lang string) string {
	l := locOf(loc)
	keys := sections
	if len(keys) == 0 {
		keys = symbols.SectionOrder
	}
	output := make(map[string]JSONSection)
	for _, secKey := range keys {
		sec := symbols.Sections[secKey]
		var items []JSONItem
		for _, item := range sec.Items {
			localName := i18n.SymbolName(lang, item.Symbol, item.Name)
			r := data[item.Symbol]
			if r == nil {
				items = append(items, JSONItem{Name: localName, Symbol: item.Symbol, Country: item.Country})
			} else {
				price := roundTo(r.Price, item.Decimals)
				change := roundTo(r.Change, item.Decimals)
				pct := roundTo(r.ChangePct, 2)
				// regenerate "15:04" from Epoch in loc timezone (reflects --tz).
				t := r.Time
				if r.Epoch > 0 {
					t = time.Unix(r.Epoch, 0).In(l).Format("15:04")
				}
				epoch := r.Epoch
				series := make([]float64, len(r.Series))
				for i, v := range r.Series {
					series[i] = roundTo(v, item.Decimals)
				}
				entry := JSONItem{
					Name:             localName,
					Symbol:           item.Symbol,
					Country:          item.Country,
					Price:            &price,
					Change:           &change,
					ChangePct:        &pct,
					Time:             &t,
					Epoch:            &epoch,
					Series:           series,
					Source:           r.Source,
					Stale:            r.Stale,
					HistoryAvailable: !r.HistoryUnavailable && len(r.Series) > 0,
				}
				prevClose := roundTo(r.PrevClose, item.Decimals)
				entry.PrevClose = &prevClose
				if r.ChangeUnavailable {
					entry.PrevClose, entry.Change, entry.ChangePct = nil, nil, nil
				}
				if r.Epoch <= 0 {
					entry.Epoch = nil
				}
				if t == "" {
					entry.Time = nil
				}
				if !r.FetchedAt.IsZero() {
					stamp := r.FetchedAt
					entry.FetchedAt = &stamp
				}
				items = append(items, entry)
			}
		}
		output[secKey] = JSONSection{Title: sectionTitle(lang, sec), Items: items}
	}
	b, _ := json.MarshalIndent(output, "", "  ")
	return string(b)
}

func roundTo(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
