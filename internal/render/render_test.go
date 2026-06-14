package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/symbols"
)

func TestFmtNum(t *testing.T) {
	tests := []struct {
		val      float64
		dec      int
		expected string
	}{
		{39500.5, 2, "39,500.50"},
		{145.123, 3, "145.123"},
		{1.2345, 4, "1.2345"},
		{100000.0, 2, "100,000.00"},
		{-500.5, 2, "-500.50"},
	}
	for _, tt := range tests {
		got := fmtNum(tt.val, tt.dec)
		if got != tt.expected {
			t.Errorf("fmtNum(%f, %d) = %q, want %q", tt.val, tt.dec, got, tt.expected)
		}
	}
}

func TestFmtChange(t *testing.T) {
	if got := fmtChange(500.5, 2); got != "+500.50" {
		t.Errorf("got %q", got)
	}
	if got := fmtChange(-200.0, 2); got != "-200.00" {
		t.Errorf("got %q", got)
	}
}

func TestFmtPct(t *testing.T) {
	if got := fmtPct(1.5); got != "+1.50%" {
		t.Errorf("got %q", got)
	}
	if got := fmtPct(-0.5); got != "-0.50%" {
		t.Errorf("got %q", got)
	}
}

func TestStringWidth(t *testing.T) {
	if w := stringWidth("abc"); w != 3 {
		t.Errorf("got %d", w)
	}
	if w := stringWidth("日経平均"); w != 8 {
		t.Errorf("got %d, want 8", w)
	}
	if w := stringWidth("S&P500"); w != 6 {
		t.Errorf("got %d", w)
	}
}

func TestSparkline(t *testing.T) {
	// monotonically increasing: first is min (▁), last is max (█)
	s := Sparkline([]float64{1, 2, 3, 4, 5, 6, 7, 8}, 0)
	rs := []rune(s)
	if len(rs) != 8 {
		t.Fatalf("length: got %d, want 8", len(rs))
	}
	if rs[0] != '▁' {
		t.Errorf("first rune: got %q, want ▁", string(rs[0]))
	}
	if rs[len(rs)-1] != '█' {
		t.Errorf("last rune: got %q, want █", string(rs[len(rs)-1]))
	}
	// all runes are in the spark set
	for _, r := range rs {
		if !strings.ContainsRune(sparkRunes, r) {
			t.Errorf("rune %q not in spark set", string(r))
		}
	}
}

func TestSparklineFlat(t *testing.T) {
	// all same value: span=0, lowest level, no panic
	s := Sparkline([]float64{5, 5, 5, 5}, 0)
	if len([]rune(s)) != 4 {
		t.Errorf("got %d runes", len([]rune(s)))
	}
}

func TestSparklineEmpty(t *testing.T) {
	if s := Sparkline(nil, 0); s != "" {
		t.Errorf("expected empty, got %q", s)
	}
}

func TestSparklineDownsample(t *testing.T) {
	// downsample 100 points to width 10
	series := make([]float64, 100)
	for i := range series {
		series[i] = float64(i)
	}
	s := Sparkline(series, 10)
	if len([]rune(s)) != 10 {
		t.Errorf("downsample width: got %d, want 10", len([]rune(s)))
	}
}

func TestGridColumns(t *testing.T) {
	tests := []struct {
		width     int
		itemCount int
		want      int // expected columns (minTileW=24)
	}{
		{60, 100, 2},  // 60/24 = 2
		{100, 100, 4}, // 100/24 = 4
		{200, 100, 8}, // 200/24 = 8
		{80, 100, 3},  // 80/24 = 3
		{10, 100, 1},  // very small still yields 1 column
		{0, 100, 3},   // 0 treated as 80 -> 3 columns
		{300, 3, 3},   // 300/24=12 but only 3 items -> 3 columns
		{300, 1, 1},   // 1 item -> 1 column
		{200, 0, 8},   // itemCount<=0 means no constraint
	}
	for _, tt := range tests {
		got := gridColumns(tt.width, tt.itemCount)
		if got != tt.want {
			t.Errorf("gridColumns(%d,%d) = %d, want %d", tt.width, tt.itemCount, got, tt.want)
		}
	}
}

func TestDistributeWidths(t *testing.T) {
	tests := []struct {
		termWidth, cols int
	}{
		{60, 2}, {100, 4}, {200, 8}, {80, 3}, {10, 1},
	}
	for _, tt := range tests {
		ws := distributeWidths(tt.termWidth, tt.cols)
		if len(ws) != tt.cols {
			t.Fatalf("distributeWidths(%d,%d): got %d widths, want %d", tt.termWidth, tt.cols, len(ws), tt.cols)
		}
		sum := 0
		for _, w := range ws {
			sum += w
		}
		if sum != tt.termWidth {
			t.Errorf("distributeWidths(%d,%d): sum=%d, want %d", tt.termWidth, tt.cols, sum, tt.termWidth)
		}
		// remainder distributed left-to-right: adjacent widths differ by at most 1 with left >= right
		for i := 1; i < len(ws); i++ {
			if ws[i-1] < ws[i] {
				t.Errorf("widths not left-loaded: %v", ws)
			}
			if ws[i-1]-ws[i] > 1 {
				t.Errorf("widths differ by >1: %v", ws)
			}
		}
	}
}

func TestChartRows(t *testing.T) {
	// non-TTY equivalent (termRows<=0): N=2
	if got := chartRows(0, 5, 10); got != 2 {
		t.Errorf("chartRows(0,..) = %d, want 2", got)
	}
	// lower bound 1
	if got := chartRows(10, 5, 20); got < 1 {
		t.Errorf("chartRows lower bound: got %d", got)
	}
	if got := chartRows(10, 5, 20); got != 1 {
		t.Errorf("tiny terminal: got %d, want 1", got)
	}
	// cap 12: very large terminal
	if got := chartRows(500, 5, 2); got != 12 {
		t.Errorf("large terminal: got %d, want 12", got)
	}
	// normal: termRows=50, header=5, tileRows=10 -> avail=45, tileH=4, N=1
	if got := chartRows(50, 5, 10); got != 1 {
		t.Errorf("chartRows(50,5,10) = %d, want 1", got)
	}
	// termRows=80, header=5, tileRows=5 → avail=75, tileH=15, N=15-4=11
	if got := chartRows(80, 5, 5); got != 11 {
		t.Errorf("chartRows(80,5,5) = %d, want 11", got)
	}
	// cap 12: tileH=16 (avail=80) -> N=12
	if got := chartRows(85, 5, 5); got != 12 {
		t.Errorf("cap 12: got %d", got)
	}
}

func TestChartRowsPerStage(t *testing.T) {
	// non-TTY (termRows<=0): all stages N=2 fixed
	ns := chartRowsPerStage(0, 5, 4)
	if len(ns) != 4 {
		t.Fatalf("got %d stages, want 4", len(ns))
	}
	for i, n := range ns {
		if n != 2 {
			t.Errorf("non-TTY stage %d: got %d, want 2", i, n)
		}
	}

	// remainder distribution: termRows=46, header=4 -> avail=42, totalTileRows=4
	// tileH = 42/4 = 10, rem = 2, baseN = 10-4 = 6
	// top 2 stages get +1 -> [7, 7, 6, 6]
	ns = chartRowsPerStage(46, 4, 4)
	want := []int{7, 7, 6, 6}
	for i := range want {
		if ns[i] != want[i] {
			t.Errorf("remainder distribution stage %d: got %d, want %d (all=%v)", i, ns[i], want[i], ns)
		}
	}
	// with remainder, total tile height equals avail (no leftover rows)
	sum := 0
	for _, n := range ns {
		sum += n + tileChrome // outer height per stage = N+4
	}
	if sum != 42 {
		t.Errorf("total tile height = %d, want avail=42 (no leftover rows)", sum)
	}

	// lower bound 1: very small terminal
	ns = chartRowsPerStage(10, 5, 20)
	for i, n := range ns {
		if n < 1 {
			t.Errorf("lower bound stage %d: got %d", i, n)
		}
	}

	// cap 12: remainder addition does not exceed 12
	// avail=200, totalTileRows=4 -> tileH=50, baseN=47->12, rem=0 -> all stages 12
	ns = chartRowsPerStage(204, 4, 4)
	for i, n := range ns {
		if n != 12 {
			t.Errorf("cap 12 stage %d: got %d, want 12", i, n)
		}
	}
}

func TestBrailleRowsDimensions(t *testing.T) {
	rows := BrailleRows([]float64{1, 2, 3, 4, 5}, 5, 2)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	for _, r := range rows {
		if w := len([]rune(r)); w != 5 {
			t.Errorf("row width: got %d, want 5", w)
		}
		// all in braille range U+2800..U+28FF
		for _, ru := range r {
			if ru < 0x2800 || ru > 0x28FF {
				t.Errorf("rune %U out of braille range", ru)
			}
		}
	}
}

func TestBrailleRowsAreaFill(t *testing.T) {
	// monotonically increasing: the rightmost x-point (max value) has nearly all dots set in the bottom row cell.
	// The bottom row cell of the max-value column should have bottom dots (dot3/dot6/dot7/dot8) set.
	rows := BrailleRows([]float64{1, 2, 3, 4, 5, 6, 7, 8}, 4, 2)
	bottom := []rune(rows[len(rows)-1])
	last := bottom[len(bottom)-1]
	bits := int(last) - 0x2800
	// bottom-row dots (left col dot3=0x04, dot7=0x40 / right col dot6=0x20, dot8=0x80): at least one set
	if bits&(0x04|0x40|0x20|0x80) == 0 {
		t.Errorf("max column bottom cell has no bottom dots: %08b", bits)
	}
	// min-value column (leftmost) top row should be empty (all dots off = U+2800)
	top := []rune(rows[0])
	if top[0] != 0x2800 {
		t.Errorf("min column top cell should be empty, got %U", top[0])
	}
}

func TestBrailleBitLayout(t *testing.T) {
	// single point, max height: verify the column is fully filled from bottom to top (area fill).
	// width=1, rows=1 -> 2 x-points x 4 levels. Maximize values to confirm all dots are set.
	rows := BrailleRows([]float64{0, 1}, 1, 1)
	r := []rune(rows[0])[0]
	bits := int(r) - 0x2800
	// x=0 is value 0 (level=0, bottom only), x=1 is value 1 (level=3, all levels)
	// left col (x=0): bottom dot dot7=0x40. right col (x=1): dot4..dot8 all = 0x08|0x10|0x20|0x80
	wantRight := 0x08 | 0x10 | 0x20 | 0x80
	if bits&wantRight != wantRight {
		t.Errorf("right column not fully filled: %08b", bits)
	}
	if bits&0x40 == 0 {
		t.Errorf("left column bottom dot (dot7=0x40) not set: %08b", bits)
	}
}

func TestBrailleQuantize(t *testing.T) {
	// bottom-to-top quantization: higher values activate higher levels.
	// rows=2 -> 8 levels. Max-value column reaches the top cell.
	rows := BrailleRows([]float64{0, 10}, 1, 2)
	topCell := []rune(rows[0])[0]
	// max value level=7 -> h up to 7 -> cellY = 2-1-7/4 = 0 (top row) reached
	if topCell == 0x2800 {
		t.Errorf("max value should reach top cell, got empty")
	}
}

func TestBrailleRowsEmpty(t *testing.T) {
	rows := BrailleRows(nil, 4, 3)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	for _, r := range rows {
		if len([]rune(r)) != 4 {
			t.Errorf("blank row width: got %d, want 4", len([]rune(r)))
		}
		for _, ru := range r {
			if ru != 0x2800 {
				t.Errorf("empty series should yield U+2800, got %U", ru)
			}
		}
	}
}

func TestBrailleRowsFlat(t *testing.T) {
	// all same values: no panic, within braille range
	rows := BrailleRows([]float64{5, 5, 5}, 3, 2)
	if len(rows) != 2 {
		t.Fatalf("got %d rows", len(rows))
	}
	for _, r := range rows {
		for _, ru := range r {
			if ru < 0x2800 || ru > 0x28FF {
				t.Errorf("rune out of braille range: %U", ru)
			}
		}
	}
}

func TestGradientRGB(t *testing.T) {
	base := [3]int{200, 100, 50}
	// top row (row=0) equals the base color
	top := gradientRGB(base, 0, 4)
	if top != base {
		t.Errorf("top row should equal base: got %v", top)
	}
	// bottom row (row=rows-1) is ~50% darker
	bottom := gradientRGB(base, 3, 4)
	want := [3]int{100, 50, 25}
	if bottom != want {
		t.Errorf("bottom row should be ~50%%: got %v, want %v", bottom, want)
	}
	// rows=1 returns base as-is
	if got := gradientRGB(base, 0, 1); got != base {
		t.Errorf("single row: got %v", got)
	}
}

func TestBadgeColorSwitching(t *testing.T) {
	// no-color: plain text with leading/trailing space
	plain := buildBadge("▲+0.06%", 0.06, false, false)
	if plain != " ▲+0.06% " {
		t.Errorf("no-color badge: got %q", plain)
	}
	// up=green bg(42), down=red bg(41), flat=bright black(100)
	if !strings.Contains(buildBadge("x", 1, true, false), "42m") {
		t.Errorf("up badge should use green bg")
	}
	if !strings.Contains(buildBadge("x", -1, true, false), "41m") {
		t.Errorf("down badge should use red bg")
	}
	if !strings.Contains(buildBadge("x", 0, true, false), "100m") {
		t.Errorf("flat badge should use bright black bg")
	}
	// rg inverts bg: up uses red bg
	if !strings.Contains(buildBadge("x", 1, true, true), "41m") {
		t.Errorf("rg up badge should use red bg")
	}
}

func TestChartGradientSwitch(t *testing.T) {
	r := &fetcher.Result{Price: 100, Change: 5, ChangePct: 1, Series: []float64{1, 2, 3, 4, 5}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 2}
	// truecolor enabled: chart rows contain 38;2;
	tc := renderTile(item, r, 30, 3, true, false, false, true, "日本")
	joinedTC := strings.Join(tc, "\n")
	if !strings.Contains(joinedTC, "38;2;") {
		t.Errorf("truecolor chart should contain 24bit fg escape")
	}
	// truecolor disabled: single color (ESC[32m green), no 38;2;
	sc := renderTile(item, r, 30, 3, true, false, false, false, "日本")
	joinedSC := strings.Join(sc, "\n")
	if strings.Contains(joinedSC, "38;2;") {
		t.Errorf("single-color chart must not contain 24bit fg escape")
	}
}

// TestLayoutWidths verifies that layouts at width 60/100/200 do not exceed the terminal width
// and are arranged in the expected number of columns.
func TestLayoutWidths(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	cases := []struct {
		width    int
		wantCols int
	}{
		{60, 2}, {100, 4}, {200, 8},
	}
	for _, c := range cases {
		out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: c.width})
		if gridColumns(c.width, 100) != c.wantCols {
			t.Errorf("width %d: cols = %d, want %d", c.width, gridColumns(c.width, 100), c.wantCols)
		}
		for _, ln := range strings.Split(out, "\n") {
			if w := stringWidth(ln); w > c.width {
				t.Errorf("width %d: line exceeds term width (%d): %q", c.width, w, ln)
			}
		}
	}
}

func TestRenderDashboardNoBlankLines(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225": {Price: 100, Change: 1, ChangePct: 1, Series: []float64{1, 2, 3}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: 100})
	for i, ln := range strings.Split(out, "\n") {
		if strings.TrimSpace(ln) == "" {
			t.Errorf("blank line at %d (gaps must be 0)", i)
		}
	}
}

func TestRenderTileNA(t *testing.T) {
	item := symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2}
	lines := renderTile(item, nil, 27, 2, false, false, true, false, "日本")
	// top border + badge row + 2 chart rows + value row + bottom border = 6
	if len(lines) != 6 {
		t.Fatalf("expected 6 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "N/A") {
		t.Errorf("expected N/A in tile, got %q", lines[1])
	}
	// ASCII border
	if !strings.Contains(lines[0], "+") {
		t.Errorf("expected ASCII border, got %q", lines[0])
	}
}

func TestRenderTileWithData(t *testing.T) {
	r := &fetcher.Result{Price: 39500.50, PrevClose: 39000.0, Change: 500.50, ChangePct: 1.28, Time: "15:00", Series: []float64{1, 2, 3, 4}}
	lines := renderTile(symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2}, r, 40, 2, false, false, true, false, "日本")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "39,500.50") {
		t.Errorf("missing price:\n%s", joined)
	}
	if !strings.Contains(joined, "+500.50") {
		t.Errorf("missing change:\n%s", joined)
	}
	// line count = 2 (chart) + 4 = 6
	if len(lines) != 6 {
		t.Errorf("expected 6 lines, got %d", len(lines))
	}
}

func TestRenderTileChartRows(t *testing.T) {
	r := &fetcher.Result{Price: 100, Change: 1, ChangePct: 1, Series: []float64{1, 2, 3, 4, 5}}
	for _, n := range []int{1, 4, 8} {
		lines := renderTile(symbols.Item{Name: "X", Symbol: "X", Decimals: 2}, r, 30, n, false, false, true, false, "日本")
		if len(lines) != n+4 {
			t.Errorf("chartN=%d: expected %d lines, got %d", n, n+4, len(lines))
		}
	}
}

func TestRenderDashboardNA(t *testing.T) {
	data := map[string]*fetcher.Result{"^N225": nil}
	// japan section only. Use wide width so tile width >= 30 and section name appears in border.
	out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: 200})
	if !strings.Contains(out, "N/A") {
		t.Error("expected N/A in output")
	}
	if !strings.Contains(out, "Japan") {
		t.Error("expected section name embedded in tile border")
	}
	// no-color output must not contain ANSI escapes
	if strings.Contains(out, "\033[") {
		t.Error("no-color output must not contain ANSI escapes")
	}
}

// TestNoSectionHeadings verifies that section heading lines (starting with ■ or #) are not present in output.
func TestNoSectionHeadings(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan", "us"}, Options{NoColor: true, TermWidth: 100})
	for i, ln := range strings.Split(out, "\n") {
		trimmed := strings.TrimSpace(ln)
		if strings.HasPrefix(trimmed, "■") || strings.HasPrefix(trimmed, "# ") {
			t.Errorf("section heading line found at %d: %q", i, ln)
		}
	}
}

// TestSectionNameOnBorder verifies that the section name is embedded in the tile top border with full-width alignment.
// The top border display width matches the tile outer width and includes the section name (box-drawing mode uses ┌).
func TestSectionNameOnBorder(t *testing.T) {
	item := symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2}
	r := &fetcher.Result{Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}}
	// outerW=40 (>=30) -> section name embedded in top border. Box-drawing mode (ascii=false).
	lines := renderTile(item, r, 40, 2, false, false, false, false, "日本")
	top := lines[0]
	if w := stringWidth(top); w != 40 {
		t.Errorf("top border width = %d, want 40 (full-width alignment): %q", w, top)
	}
	if !strings.Contains(top, "日本") {
		t.Errorf("section name not on border: %q", top)
	}
	if !strings.HasPrefix(top, "┌") {
		t.Errorf("expected box-drawing top border, got %q", top)
	}
	// also contains the name
	if !strings.Contains(top, "日経平均") {
		t.Errorf("name missing from border: %q", top)
	}
}

// TestSectionNameOmittedNarrow verifies that the section name is omitted when tile width < 30.
func TestSectionNameOmittedNarrow(t *testing.T) {
	item := symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2}
	// outerW=27 (<30) -> no section name. Box-drawing mode.
	lines := renderTile(item, nil, 27, 2, false, false, false, false, "日本")
	if strings.Contains(lines[0], "日本") {
		t.Errorf("section name should be omitted for narrow tile (<30): %q", lines[0])
	}
	// top border display width equals outer width
	if w := stringWidth(lines[0]); w != 27 {
		t.Errorf("narrow top border width = %d, want 27", w)
	}
}

// TestColsNotExceedItemCount verifies that columns do not exceed item count (3 items, 300 cols terminal -> 3 columns).
func TestColsNotExceedItemCount(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	// japan = 3 items, terminal 300 cols. 300/24=12 but only 3 items -> 3 columns.
	if c := gridColumns(300, 3); c != 3 {
		t.Errorf("gridColumns(300,3) = %d, want 3", c)
	}
	// top border (ASCII mode starts with +): 1 tile-row, 3 tiles.
	out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: 300})
	lines := strings.Split(out, "\n")
	topLineCount := 0
	for _, ln := range lines {
		if strings.HasPrefix(ln, "+") {
			topLineCount++
		}
	}
	// ASCII top and bottom borders both start with +. 1 tile-row = top+bottom = 2 lines.
	if topLineCount != 2 {
		t.Errorf("expected 2 border lines (1 tile-row: top+bottom), got %d", topLineCount)
	}
	// all tile lines (except header) have display width exactly 300.
	for i := 1; i < len(lines); i++ {
		if w := stringWidth(lines[i]); w != 300 {
			t.Errorf("tile line %d width = %d, want 300: %q", i, w, lines[i])
		}
	}
}

// TestTermWidth300LineWidths verifies that when TermWidth=300, every tile line has display width (fullwidth=2) exactly 300.
// Uses an item count that fits in 1 row (japan 3 items, cols=3) so all rows are full width.
func TestTermWidth300LineWidths(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: 300})
	lines := strings.Split(out, "\n")
	// skip header line; every tile line should have display width 300.
	for i := 1; i < len(lines); i++ {
		if w := stringWidth(lines[i]); w != 300 {
			t.Errorf("line %d width = %d, want 300: %q", i, w, lines[i])
		}
	}
}

func TestRenderJSON(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.50, PrevClose: 39000.0, Change: 500.50, ChangePct: 1.28, Time: "15:00", Series: []float64{39000, 39500.5}},
		"NKD=F":    nil,
		"USDJPY=X": nil,
	}
	out := RenderJSON(data, []string{"japan"}, nil, "en")
	if !strings.Contains(out, `"price": 39500.5`) {
		t.Errorf("unexpected JSON:\n%s", out)
	}
	if !strings.Contains(out, `"price": null`) {
		t.Errorf("expected null for missing data:\n%s", out)
	}
	if !strings.Contains(out, `"series"`) {
		t.Errorf("expected series field:\n%s", out)
	}
	// verify series is parseable
	var parsed map[string]JSONSection
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	jp := parsed["japan"]
	if len(jp.Items) == 0 || len(jp.Items[0].Series) != 2 {
		t.Errorf("series not serialized correctly: %+v", jp.Items)
	}
}

// TestFillHeightExact verifies that FillHeight + TermRows produces exactly TermRows output lines (no leftover rows).
// japan(3 items), width 30 -> cols=min(3,1)=1 -> 3 tile rows. header=1 (section headings removed).
// TermRows=40 -> avail=40-1=39, tileH=39/3=13, baseN=10, rem=0 -> stageN=[10,10,10].
// tile outer height = 3*(10+3)=39. output = header 1 + 39 = 40.
func TestFillHeightExact(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{
		NoColor: true, TermWidth: 30, TermRows: 40, FillHeight: true,
	})
	got := len(strings.Split(out, "\n"))
	if got != 40 {
		t.Errorf("FillHeight output lines = %d, want TermRows=40", got)
	}
}

// TestFillHeightCappedDiff verifies column selection after width+height optimization.
// japan(3 items), width 100, header=1, TermRows=40 -> avail=39.
//
//	C=1: 3 stages, tileH=13, baseN=10, rem=0, used=3*13=39(=avail, chosen)
//	C=2: 2 stages, tileH=19, baseN=16->12, used=2*15=30
//	C=3: 1 stage, baseN->12, used=15
//
// C=1 maximizes height usage, output = header 1 + 39 = 40 (no leftover).
func TestFillHeightCappedDiff(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{
		NoColor: true, TermWidth: 100, TermRows: 40, FillHeight: true,
	})
	got := len(strings.Split(out, "\n"))
	if got != 40 {
		t.Errorf("FillHeight lines = %d, want 40 (height-optimized C=1)", got)
	}
}

// TestNonTTYFixedN verifies that in non-TTY mode (FillHeight=false, Watch=false),
// N=2 is fixed regardless of TermRows.
func TestNonTTYFixedN(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	// width 30 -> cols=min(items,1)=1 -> one stage per item. N=2 fixed -> height 5 per stage.
	// output = header 1 + stages*(2+tileChrome) (section headings removed). Unchanged with different TermRows.
	stages := len(symbols.Sections["japan"].Items)
	for _, rows := range []int{0, 40, 100} {
		out := RenderDashboard(data, []string{"japan"}, Options{
			NoColor: true, TermWidth: 30, TermRows: rows, // FillHeight=false, Watch=false
		})
		got := len(strings.Split(out, "\n"))
		want := 1 + stages*(2+tileChrome)
		if got != want {
			t.Errorf("non-TTY TermRows=%d: lines = %d, want %d (N=2 fixed)", rows, got, want)
		}
	}
}

// TestPerStageVaryingN verifies that rendering with per-stage varying N (remainder row distribution)
// does not break layout (no line exceeds terminal width, no blank lines).
func TestPerStageVaryingN(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{
		NoColor: true, TermWidth: 30, TermRows: 40, FillHeight: true,
	})
	for i, ln := range strings.Split(out, "\n") {
		if w := stringWidth(ln); w > 30 {
			t.Errorf("line %d exceeds width 30: w=%d %q", i, w, ln)
		}
	}
}

// TestUsedRowsForCols verifies that the used-row calculation is consistent with chartRowsPerStage.
// 33 items, width 300, height 90, header 1. tileChrome=4.
//
//	C=6:  stages=6, avail=89, tileH=14, baseN=10, rem=5 -> stageN=[11,11,11,11,11,10]
//	      used = 5*(4+11)+(4+10) = 75+14 = 89
//	C=12: stages=3, avail=89, tileH=29, baseN=25->12(cap), rem=2 -> stageN=[12,12,12]
//	      used = 3*(4+12) = 48
func TestUsedRowsForCols(t *testing.T) {
	if got := usedRowsForCols(90, 1, 33, 6); got != 89 {
		t.Errorf("usedRowsForCols(C=6) = %d, want 89", got)
	}
	if got := usedRowsForCols(90, 1, 33, 12); got != 48 {
		t.Errorf("usedRowsForCols(C=12) = %d, want 48", got)
	}
	if got := usedRowsForCols(0, 1, 33, 6); got != 0 {
		t.Errorf("usedRowsForCols(termRows=0) = %d, want 0", got)
	}
}

// TestOptimalColumnsSpecExample verifies the SPEC example:
// 33 items, 300 cols x 90 rows -> C ~6 (not 12), used rows ~89/89.
func TestOptimalColumnsSpecExample(t *testing.T) {
	c := optimalColumns(300, 90, 1, 33)
	if c < 5 || c > 7 {
		t.Errorf("optimalColumns(33, 300x90) = %d, want ~6", c)
	}
	used := usedRowsForCols(90, 1, 33, c)
	avail := 90 - 1
	if avail-used > 2 {
		t.Errorf("used=%d (avail=%d): 画面下部が余りすぎ", used, avail)
	}
	// confirm width-only logic would choose C=12 (shows the improvement).
	if w := gridColumns(300, 33); w != 12 {
		t.Errorf("gridColumns(300,33) = %d, want 12 (width-only)", w)
	}
}

// TestOptimalColumnsSmallTerm verifies that column count does not exceed the width-based max in small terminals.
// width 40 -> maxC = 40/24 = 1. C=1 regardless of height.
func TestOptimalColumnsSmallTerm(t *testing.T) {
	if c := optimalColumns(40, 24, 1, 10); c != 1 {
		t.Errorf("optimalColumns(small term) = %d, want 1", c)
	}
}

// TestOptimalColumnsFewItems verifies that C does not exceed itemCount when items < candidate columns.
// width 300 -> maxC=12, but 3 items -> C<=3.
func TestOptimalColumnsFewItems(t *testing.T) {
	c := optimalColumns(300, 90, 1, 3)
	if c < 1 || c > 3 {
		t.Errorf("optimalColumns(3 items) = %d, want 1..3", c)
	}
}

// TestOptimalColumnsNonTTYFallback verifies that when termRows<=0 (unknown height),
// it falls back to width-only gridColumns.
func TestOptimalColumnsNonTTYFallback(t *testing.T) {
	if c := optimalColumns(300, 0, 1, 33); c != gridColumns(300, 33) {
		t.Errorf("optimalColumns(termRows=0) = %d, want gridColumns fallback %d", c, gridColumns(300, 33))
	}
}

// TestRenderDashboardNonTTYCompat verifies that in non-TTY (FillHeight=false, Watch=false),
// columns are determined by width only (gridColumns), confirmed via output line count.
// japan, width 100 -> cols=min(items, 100/24=4) -> ceil(items/cols) stages. N=2 fixed.
// output = header 1 + stages*(2+tileChrome).
func TestRenderDashboardNonTTYCompat(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{
		NoColor: true, TermWidth: 100, TermRows: 40, FillHeight: false, Watch: false,
	})
	got := len(strings.Split(out, "\n"))
	n := len(symbols.Sections["japan"].Items)
	cols := gridColumns(100, n)
	stages := (n + cols - 1) / cols
	want := 1 + stages*(2+tileChrome)
	if got != want {
		t.Errorf("non-TTY lines = %d, want %d (width-only cols, N=2 fixed)", got, want)
	}
}

// TestRenderJSONFields verifies JSON output includes country/epoch fields (no network required).
func TestRenderJSONFields(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225": {Price: 39500.5, PrevClose: 39000.0, Change: 500.5, ChangePct: 1.28, Time: "15:00", Epoch: 1718100000, Series: []float64{39100, 39300, 39500.5}},
	}
	out := RenderJSON(data, []string{"japan"}, nil, "en")

	var parsed map[string]JSONSection
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	jp, ok := parsed["japan"]
	if !ok {
		t.Fatal("missing japan section")
	}
	var n225 *JSONItem
	for i := range jp.Items {
		if jp.Items[i].Symbol == "^N225" {
			n225 = &jp.Items[i]
			break
		}
	}
	if n225 == nil {
		t.Fatal("missing ^N225 item")
	}
	if n225.Country != "JP" {
		t.Errorf("country: got %q, want JP", n225.Country)
	}
	if n225.Epoch == nil || *n225.Epoch != 1718100000 {
		t.Errorf("epoch: got %v, want 1718100000", n225.Epoch)
	}
	if n225.Price == nil || *n225.Price != 39500.5 {
		t.Errorf("price: got %v, want 39500.5", n225.Price)
	}
}

// TestRenderJSONNA verifies that when fetch fails (nil), country is still output and epoch/price are null.
func TestRenderJSONNA(t *testing.T) {
	data := map[string]*fetcher.Result{"^N225": nil}
	out := RenderJSON(data, []string{"japan"}, nil, "en")

	var parsed map[string]JSONSection
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("json unmarshal: %v", err)
	}
	for _, it := range parsed["japan"].Items {
		if it.Symbol == "^N225" {
			if it.Country != "JP" {
				t.Errorf("country: got %q, want JP", it.Country)
			}
			if it.Price != nil {
				t.Errorf("price should be null, got %v", it.Price)
			}
			if it.Epoch != nil {
				t.Errorf("epoch should be null, got %v", it.Epoch)
			}
			return
		}
	}
	t.Fatal("missing ^N225 item")
}

// TestMideastAmericaSection verifies that the new section is in SectionOrder and has items.
func TestMideastAmericaSection(t *testing.T) {
	found := false
	for _, k := range symbols.SectionOrder {
		if k == "mideast-america" {
			found = true
		}
	}
	if !found {
		t.Error("mideast-america not in SectionOrder")
	}
	sec := symbols.Sections["mideast-america"]
	if len(sec.Items) == 0 {
		t.Error("mideast-america has no items")
	}
	for _, it := range sec.Items {
		if it.Country == "" {
			t.Errorf("%s missing country", it.Symbol)
		}
	}
}
