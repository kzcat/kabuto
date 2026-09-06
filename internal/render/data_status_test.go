package render

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/symbols"
)

func TestUnavailableAndStaleDataRendering(t *testing.T) {
	r := &fetcher.Result{Price: 100, Source: "stooq", FetchedAt: time.Date(2026, 9, 7, 1, 2, 3, 0, time.UTC), Stale: true, ChangeUnavailable: true, HistoryUnavailable: true}
	data := map[string]*fetcher.Result{"^DJI": r}
	item := symbols.Item{Name: "Dow", Symbol: "^DJI", Decimals: 2}
	for _, width := range []int{24, 40} {
		for _, mode := range []string{"braille", "block", "tty"} {
			for _, colored := range []bool{false, true} {
				lines := renderTileLG(item, r, width, 2, colored, false, !colored, false, "US", "en", defaultTheme, mode, 16)
				out := strings.Join(lines, "\n")
				if !strings.Contains(out, "STALE") || !strings.Contains(out, "History N/A") || !strings.Contains(out, "100.00") || strings.Contains(out, "0.00%") {
					t.Fatalf("misleading tile: %s", out)
				}
				if len(lines) != 6 {
					t.Fatalf("tile height changed: %d", len(lines))
				}
				if !colored {
					for _, line := range lines {
						if stringWidth(line) != width {
							t.Fatalf("width %d: %q", width, line)
						}
					}
				}
				selected := strings.Join(highlightTile(lines, width, boldWhite, reset, !colored), "\n")
				if !strings.Contains(selected, "STALE") {
					t.Fatal("selection hid stale marker")
				}
			}
		}
	}
	detail := renderDetail(flatItem{item: item}, data, Options{Loc: time.UTC}, 100, 20, false, true, false, 16, "tty", time.UTC)
	for _, text := range []string{"STALE", "Source: stooq", "Fetched: 2026-09-07 01:02:03", "Prev: N/A", "History N/A"} {
		if !strings.Contains(detail, text) {
			t.Fatalf("missing %q: %s", text, detail)
		}
	}
	if strings.Contains(detail, "0.00%") {
		t.Fatalf("unavailable change rendered as zero: %s", detail)
	}
	var out map[string]JSONSection
	if err := json.Unmarshal([]byte(RenderJSON(data, []string{"us"}, time.UTC, "en")), &out); err != nil {
		t.Fatal(err)
	}
	got := out["us"].Items[0]
	if got.Price == nil || *got.Price != 100 || got.Change != nil || got.ChangePct != nil || got.PrevClose != nil || got.Time != nil || got.Epoch != nil {
		t.Fatalf("missing values must be JSON null: %+v", got)
	}
	if !got.Stale || got.Source != "stooq" || got.FetchedAt == nil || !got.FetchedAt.Equal(r.FetchedAt) || got.HistoryAvailable || len(got.Series) != 0 {
		t.Fatalf("incorrect provenance/history: %+v", got)
	}
}

func TestHistoryDoesNotOverrideUnsupportedSource(t *testing.T) {
	r := &fetcher.Result{Price: 100, HistoryUnavailable: true}
	got := applyHistory(map[string]*fetcher.Result{"X": r}, map[string][]float64{"X": {1, 2, 3}})
	if len(got["X"].Series) != 0 {
		t.Fatal("watch history concealed unsupported source history")
	}
}

func TestLoadingDashboard(t *testing.T) {
	out := RenderDashboard(nil, []string{"us"}, Options{NoColor: true, TermWidth: 80, Loading: true})
	if !strings.Contains(out, "Loading...") || strings.Contains(out, "Updated:") {
		t.Fatalf("initial fetch must show loading: %s", out)
	}
}
