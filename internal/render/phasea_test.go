package render

import (
	"strings"
	"testing"
	"time"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/symbols"
)

// TestTileLayoutOrder verifies the new layout (top=badge, middle=chart, bottom=price+change).
func TestTileLayoutOrder(t *testing.T) {
	r := &fetcher.Result{Price: 39500.50, PrevClose: 39000.0, Change: 500.50, ChangePct: 1.28, Series: []float64{39000, 39200, 39500.5}}
	item := symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2, Country: "JP"}
	lines := renderTile(item, r, 40, 3, false, false, true, false, "日本")
	if len(lines) != 7 {
		t.Fatalf("expected 7 lines (N=3+4), got %d", len(lines))
	}
	if !strings.Contains(lines[1], "1.28%") {
		t.Errorf("badge row should contain pct, got %q", lines[1])
	}
	if !strings.Contains(lines[5], "39,500.50") {
		t.Errorf("value row should contain price, got %q", lines[5])
	}
	if !strings.Contains(lines[5], "+500.50") {
		t.Errorf("value row should contain change, got %q", lines[5])
	}
	if strings.Contains(lines[1], "39,500.50") {
		t.Errorf("badge row should not contain price: %q", lines[1])
	}
}

// TestCountryCodeOnBorder verifies that the country code [JP] appears before the symbol name on the top border.
func TestCountryCodeOnBorder(t *testing.T) {
	item := symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2, Country: "JP"}
	r := &fetcher.Result{Price: 100, Change: 1, ChangePct: 1, Series: []float64{1, 2, 3}}
	lines := renderTile(item, r, 40, 2, false, false, false, false, "日本")
	top := lines[0]
	if !strings.Contains(top, "[JP]") {
		t.Errorf("country code not on border: %q", top)
	}
	if strings.Index(top, "[JP]") > strings.Index(top, "日経平均") {
		t.Errorf("country code should precede name: %q", top)
	}
	if w := stringWidth(top); w != 40 {
		t.Errorf("top border width = %d, want 40", w)
	}
}

// TestCountryCodeOmitted verifies that no country code is shown when Country is empty.
func TestCountryCodeOmitted(t *testing.T) {
	item := symbols.Item{Name: "BTCドル", Symbol: "BTC-USD", Decimals: 2, Country: ""}
	r := &fetcher.Result{Price: 100, Change: 1, ChangePct: 1, Series: []float64{1, 2, 3}}
	lines := renderTile(item, r, 40, 2, false, false, false, false, "暗号資産")
	if strings.Contains(lines[0], "[") {
		t.Errorf("no country code expected: %q", lines[0])
	}
}

// TestChartCellRowFor verifies baseline cell-row quantization (0=top row).
func TestChartCellRowFor(t *testing.T) {
	if got := chartCellRowFor(10, 0, 10, 4); got != 0 {
		t.Errorf("max value cell row = %d, want 0 (top)", got)
	}
	if got := chartCellRowFor(0, 0, 10, 4); got != 3 {
		t.Errorf("min value cell row = %d, want 3 (bottom)", got)
	}
	if got := chartCellRowFor(5, 5, 5, 4); got != 3 {
		t.Errorf("flat cell row = %d, want 3", got)
	}
}

// TestBaselineRendered verifies that the red baseline (prevClose) is drawn in chart rows (useColor).
func TestBaselineRendered(t *testing.T) {
	r := &fetcher.Result{Price: 105, PrevClose: 100, Change: 5, ChangePct: 5, Series: []float64{95, 100, 105}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 2}
	lines := renderTile(item, r, 28, 4, true, false, false, false, "日本")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, redDashed) {
		t.Errorf("baseline (red) escape not found in chart:\n%s", joined)
	}
}

// TestGuidelinesWhenTall verifies that guidelines (bright black dotted) are drawn when N>=4 and +/-1% is in range.
func TestGuidelinesWhenTall(t *testing.T) {
	cc := chartColors{use: true, base: greenRGB, mono: green, reset: reset}
	rows := buildChartLines([]float64{90, 100, 110}, 100, 20, 6, 0, 2, cc)
	joined := strings.Join(rows, "")
	if !strings.Contains(joined, brightBlk) {
		t.Errorf("guideline (bright black) not rendered")
	}
}

// TestNoGuidelinesWhenShort verifies that guidelines are not drawn when N<4.
func TestNoGuidelinesWhenShort(t *testing.T) {
	cc := chartColors{use: true, base: greenRGB, mono: green, reset: reset}
	rows := buildChartLines([]float64{90, 100, 110}, 100, 20, 3, 0, 2, cc)
	joined := strings.Join(rows, "")
	if strings.Contains(joined, brightBlk) {
		t.Errorf("no guideline expected for N<4, but bright black found")
	}
}

// TestHighLowLabels verifies that high (top-right) and low (bottom-right) labels are drawn when tile width >= 30.
func TestHighLowLabels(t *testing.T) {
	r := &fetcher.Result{Price: 105, PrevClose: 100, Change: 5, ChangePct: 5, Series: []float64{95, 110, 90, 105}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 2}
	lines := renderTile(item, r, 40, 4, false, false, false, false, "日本")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "110") {
		t.Errorf("high label (110) not found:\n%s", joined)
	}
	if !strings.Contains(joined, "90") {
		t.Errorf("low label (90) not found:\n%s", joined)
	}
}

// TestNoLabelsNarrow verifies that high/low labels are omitted when tile width < 30.
func TestNoLabelsNarrow(t *testing.T) {
	r := &fetcher.Result{Price: 105, PrevClose: 100, Change: 5, ChangePct: 5, Series: []float64{95, 12345, 90, 105}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 2}
	lines := renderTile(item, r, 28, 4, false, false, false, false, "日本")
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "12,345") || strings.Contains(joined, "12345") {
		t.Errorf("labels should be omitted for narrow tile:\n%s", joined)
	}
}

// TestIsClosed verifies closed-market detection (epoch older than 30 minutes).
func TestIsClosed(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	if !isClosed(now.Add(-31*time.Minute).Unix(), now) {
		t.Error("31min old should be closed")
	}
	if isClosed(now.Add(-10*time.Minute).Unix(), now) {
		t.Error("10min old should be open")
	}
	if isClosed(0, now) {
		t.Error("epoch 0 should not be closed")
	}
}

// TestClosedMarketGrey verifies that closed-market charts are drawn in bright black monochrome and the red baseline becomes grey.
func TestClosedMarketGrey(t *testing.T) {
	old := time.Now().Add(-2 * time.Hour).Unix()
	r := &fetcher.Result{Price: 105, PrevClose: 100, Change: 5, ChangePct: 5, Epoch: old, Series: []float64{95, 100, 110}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 2}
	lines := renderTile(item, r, 28, 4, true, false, false, true, "日本")
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "38;2;") {
		t.Errorf("closed market chart must not use truecolor gradient:\n%s", joined)
	}
	if !strings.Contains(joined, brightBlk) {
		t.Errorf("closed market chart should use bright black:\n%s", joined)
	}
	if strings.Contains(joined, redDashed) {
		t.Errorf("closed market should not draw red baseline:\n%s", joined)
	}
	if !strings.Contains(joined, "42m") {
		t.Errorf("badge color should be preserved even when closed:\n%s", joined)
	}
}

// TestNoClockTile verifies that no clock tile is rendered even when the last row has empty cells.
func TestNoClockTile(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: 60})
	if strings.Contains(out, "Clock") {
		t.Errorf("clock tile must not be rendered:\n%s", out)
	}
}

// TestNewLayoutNoColorCompat verifies that --no-color output contains no ANSI escapes even with the new layout.
func TestNewLayoutNoColorCompat(t *testing.T) {
	r := &fetcher.Result{Price: 39500.5, PrevClose: 39000, Change: 500, ChangePct: 1.28, Series: []float64{39000, 39500.5}}
	lines := renderTile(symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2, Country: "JP"}, r, 40, 4, false, false, true, false, "日本")
	for _, ln := range lines {
		if strings.Contains(ln, "\033[") {
			t.Errorf("no-color tile must not contain ANSI escapes: %q", ln)
		}
	}
}
