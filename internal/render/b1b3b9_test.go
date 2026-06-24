package render

import (
	"strings"
	"testing"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/symbols"
)

// --- B3: RGB -> 256-color cube ---

func TestRGBTo256(t *testing.T) {
	cases := []struct {
		r, g, b int
		want    int
	}{
		{0, 0, 0, 16},        // cube origin
		{255, 255, 255, 231}, // cube max white
		{255, 0, 0, 196},     // pure red
		{0, 255, 0, 46},      // pure green
		{0, 0, 255, 21},      // pure blue
		{128, 128, 128, 244}, // mid gray -> grayscale ramp
	}
	for _, c := range cases {
		if got := rgbTo256(c.r, c.g, c.b); got != c.want {
			t.Errorf("rgbTo256(%d,%d,%d)=%d, want %d", c.r, c.g, c.b, got, c.want)
		}
	}
}

func TestRGBTo256Range(t *testing.T) {
	// All outputs must be valid 256-color indices (0..255).
	for r := 0; r <= 255; r += 51 {
		for g := 0; g <= 255; g += 51 {
			for b := 0; b <= 255; b += 51 {
				idx := rgbTo256(r, g, b)
				if idx < 0 || idx > 255 {
					t.Fatalf("rgbTo256(%d,%d,%d)=%d out of range", r, g, b, idx)
				}
			}
		}
	}
}

// --- B3: RGB -> 16-color ANSI ---

func TestRGBTo16(t *testing.T) {
	cases := []struct {
		r, g, b int
		want    int
	}{
		{0, 0, 0, 0},        // black
		{170, 0, 0, 1},      // red
		{0, 170, 0, 2},      // green
		{0, 0, 170, 4},      // blue
		{255, 255, 255, 15}, // bright white
		{0, 200, 0, 2},      // gain green base -> nearest normal green
		{220, 40, 40, 9},    // loss red base -> bright red
	}
	for _, c := range cases {
		if got := rgbTo16(c.r, c.g, c.b); got != c.want {
			t.Errorf("rgbTo16(%d,%d,%d)=%d, want %d", c.r, c.g, c.b, got, c.want)
		}
	}
}

func TestRGBTo16Range(t *testing.T) {
	for r := 0; r <= 255; r += 85 {
		for g := 0; g <= 255; g += 85 {
			for b := 0; b <= 255; b += 85 {
				idx := rgbTo16(r, g, b)
				if idx < 0 || idx > 15 {
					t.Fatalf("rgbTo16(%d,%d,%d)=%d out of range", r, g, b, idx)
				}
			}
		}
	}
}

// --- B3: fgDepth escapes per depth ---

func TestFgDepthTruecolorUnchanged(t *testing.T) {
	c := [3]int{0, 200, 0}
	if got, want := fgDepth(c, depthTruecolor), fg24(c); got != want {
		t.Errorf("truecolor fgDepth=%q, want %q (must equal fg24)", got, want)
	}
}

func TestFgDepth256(t *testing.T) {
	got := fgDepth([3]int{0, 200, 0}, depth256)
	if !strings.HasPrefix(got, "\033[38;5;") {
		t.Errorf("256 depth should use 38;5;N form, got %q", got)
	}
}

func TestFgDepth16(t *testing.T) {
	got := fgDepth([3]int{0, 200, 0}, depth16)
	// {0,200,0} nearest ANSI is index 2 (green) -> ESC[32m
	if got != "\033[32m" {
		t.Errorf("16 depth green=%q, want \\033[32m", got)
	}
}

// --- B1: block symbol mapping (table-driven) ---

func TestBlockSymbolFor(t *testing.T) {
	cases := []struct {
		name                 string
		level, cellRow, rows int
		want                 rune
	}{
		// rows=1: a single cell. level 0 -> 1/8 block, level 7 -> full.
		{"single bottom 1/8", 0, 0, 1, '▁'},
		{"single full", 7, 0, 1, '█'},
		{"single 4/8", 3, 0, 1, '▄'},
		// rows=2: bottom cell (cellRow=1) fills first, top cell (cellRow=0) later.
		{"2rows low: top empty", 3, 0, 2, ' '},
		{"2rows low: bottom partial", 3, 1, 2, '▄'}, // fill=4 in bottom cell -> 4/8
		{"2rows high: top partial", 11, 0, 2, '▄'},  // fill=12, cellBottom(top)=8, rem=4
		{"2rows high: bottom full", 11, 1, 2, '█'},
		// full-height column.
		{"2rows max: bottom full", 15, 1, 2, '█'},
		{"2rows max: top full", 15, 0, 2, '█'},
		// out-of-range level clamps.
		{"negative level", -5, 0, 1, '▁'},
	}
	for _, c := range cases {
		if got := blockSymbolFor(c.level, c.cellRow, c.rows); got != c.want {
			t.Errorf("%s: blockSymbolFor(%d,%d,%d)=%q, want %q",
				c.name, c.level, c.cellRow, c.rows, got, c.want)
		}
	}
}

// --- B1: chart symbol mode dispatch ---

func TestBuildChartLinesBlockUsesBlockRunes(t *testing.T) {
	cc := chartColors{use: true, base: greenRGB, mono: green, reset: reset, symbol: "block", depth: depthTruecolor}
	rows := buildChartLines([]float64{90, 100, 110}, 100, 20, 4, 0, 2, cc)
	joined := strings.Join(rows, "")
	hasBlock := false
	for _, r := range "▁▂▃▄▅▆▇█" {
		if strings.ContainsRune(joined, r) {
			hasBlock = true
			break
		}
	}
	if !hasBlock {
		t.Errorf("block mode should contain block runes:\n%s", joined)
	}
	if strings.ContainsRune(joined, '⠀') {
		t.Errorf("block mode must not use braille blank")
	}
}

func TestBuildChartLinesTTYAsciiOnly(t *testing.T) {
	cc := chartColors{use: false, symbol: "tty", reset: reset}
	rows := buildChartLines([]float64{90, 100, 110}, 100, 20, 4, 0, 2, cc)
	joined := strings.Join(rows, "")
	for _, r := range joined {
		if r > 127 {
			t.Errorf("tty mode (no color) must be ASCII-only, found %q", r)
		}
	}
}

func TestBuildChartLinesBrailleDefaultUnchanged(t *testing.T) {
	// symbol "" should behave like braille (existing tests rely on this).
	cc := chartColors{use: true, base: greenRGB, mono: green, reset: reset}
	rows := buildChartLines([]float64{90, 100, 110}, 100, 20, 4, 0, 2, cc)
	joined := strings.Join(rows, "")
	if !strings.ContainsRune(joined, '⠀') && !strings.Contains(joined, "\u2800") {
		// braille charts use U+2800 range; at least body cells must be braille.
		hasBraille := false
		for _, r := range joined {
			if r >= 0x2800 && r <= 0x28FF {
				hasBraille = true
				break
			}
		}
		if !hasBraille {
			t.Errorf("default symbol should render braille runes:\n%q", joined)
		}
	}
}

// --- B1: graph-symbol auto resolution ---

func TestResolveGraphSymbol(t *testing.T) {
	cases := []struct {
		mode    string
		noColor bool
		utf8    bool
		want    string
	}{
		{"auto", false, true, "braille"},
		{"auto", true, true, "tty"},   // no-color -> ascii -> tty
		{"auto", false, false, "tty"}, // non-UTF-8 -> tty
		{"braille", true, false, "braille"},
		{"block", false, true, "block"},
		{"tty", false, true, "tty"},
		{"", false, true, "braille"}, // empty defaults to auto
		{"bogus", false, true, "braille"},
	}
	for _, c := range cases {
		if got := resolveGraphSymbol(c.mode, c.noColor, c.utf8); got != c.want {
			t.Errorf("resolveGraphSymbol(%q,%v,%v)=%q, want %q", c.mode, c.noColor, c.utf8, got, c.want)
		}
	}
}

// --- B9: meterBar ---

func TestMeterBarWidthPlain(t *testing.T) {
	bar := meterBar(1.5, 8, defaultTheme, false, false, depthTruecolor)
	if got := stringWidth(bar); got != 8 {
		t.Errorf("meterBar plain width=%d, want 8: %q", got, bar)
	}
}

func TestMeterBarFull(t *testing.T) {
	bar := meterBar(10.0, 8, defaultTheme, false, false, depthTruecolor) // clamps to full
	if bar != strings.Repeat("█", 8) {
		t.Errorf("large pct should fill the bar: %q", bar)
	}
}

func TestMeterBarEmpty(t *testing.T) {
	bar := meterBar(0, 8, defaultTheme, false, false, depthTruecolor)
	if bar != strings.Repeat("░", 8) {
		t.Errorf("zero pct should be empty bar: %q", bar)
	}
}

func TestMeterBarColorDepth(t *testing.T) {
	// truecolor fill should use 38;2; ; 256 should use 38;5;
	tc := meterBar(2.0, 8, defaultTheme, true, false, depthTruecolor)
	if !strings.Contains(tc, "\033[38;2;") {
		t.Errorf("truecolor meter should use 38;2;: %q", tc)
	}
	c256 := meterBar(2.0, 8, defaultTheme, true, false, depth256)
	if !strings.Contains(c256, "\033[38;5;") {
		t.Errorf("256 meter should use 38;5;: %q", c256)
	}
}

// --- B2: bottom-border H/L label ---

func TestBottomBorderHL(t *testing.T) {
	r := &fetcher.Result{Price: 105, PrevClose: 100, Change: 5, ChangePct: 5, Series: []float64{95, 110, 90, 105}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 0, Country: "JP"}
	lines := renderTileLG(item, r, 40, 4, false, false, true, false, "Japan", "en", defaultTheme, "braille", 0)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "H:110") || !strings.Contains(joined, "L:90") {
		t.Errorf("bottom border should contain H:110 L:90:\n%s", joined)
	}
	// width must stay consistent
	for i, ln := range lines {
		if w := stringWidth(ln); w != 40 {
			t.Errorf("line %d width=%d want 40: %q", i, w, ln)
		}
	}
}

func TestBlockTileWidthConsistent(t *testing.T) {
	r := &fetcher.Result{Price: 39500.5, PrevClose: 39000, Change: 500.5, ChangePct: 1.28, Series: []float64{39000, 39200, 39500.5}, Currency: "JPY"}
	item := symbols.Item{Name: "Nikkei", Symbol: "^N225", Decimals: 2, Country: "JP"}
	// NoColor (ascii) so stringWidth reflects visible width.
	lines := renderTileLG(item, r, 40, 3, false, false, true, false, "Japan", "en", defaultTheme, "block", 0)
	for i, ln := range lines {
		if w := stringWidth(ln); w != 40 {
			t.Errorf("block tile line %d width=%d want 40: %q", i, w, ln)
		}
	}
}

func TestTTYTileWidthConsistent(t *testing.T) {
	r := &fetcher.Result{Price: 39500.5, PrevClose: 39000, Change: 500.5, ChangePct: 1.28, Series: []float64{39000, 39200, 39500.5}, Currency: "JPY"}
	item := symbols.Item{Name: "Nikkei", Symbol: "^N225", Decimals: 2, Country: "JP"}
	lines := renderTileLG(item, r, 40, 3, false, false, true, false, "Japan", "en", defaultTheme, "tty", 0)
	for i, ln := range lines {
		if w := stringWidth(ln); w != 40 {
			t.Errorf("tty tile line %d width=%d want 40: %q", i, w, ln)
		}
	}
}
