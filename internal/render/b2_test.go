package render

import (
	"strings"
	"testing"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/symbols"
)

// --- Number formatting per locale ---

func TestFmtNumLangEN(t *testing.T) {
	if got := fmtNumLang(1234.56, 2, "en"); got != "1,234.56" {
		t.Errorf("en: got %q, want %q", got, "1,234.56")
	}
}

func TestFmtNumLangJA(t *testing.T) {
	if got := fmtNumLang(1234.56, 2, "ja"); got != "1,234.56" {
		t.Errorf("ja: got %q, want %q", got, "1,234.56")
	}
}

func TestFmtNumLangDE(t *testing.T) {
	if got := fmtNumLang(1234.56, 2, "de"); got != "1.234,56" {
		t.Errorf("de: got %q, want %q", got, "1.234,56")
	}
}

func TestFmtNumLangFR(t *testing.T) {
	if got := fmtNumLang(1234.56, 2, "fr"); got != "1 234,56" {
		t.Errorf("fr: got %q, want %q", got, "1 234,56")
	}
}

func TestFmtNumLangLargeInt(t *testing.T) {
	cases := []struct {
		lang, want string
	}{
		{"en", "1,234,567"},
		{"ja", "1,234,567"},
		{"de", "1.234.567"},
		{"fr", "1 234 567"},
	}
	for _, c := range cases {
		if got := fmtNumLang(1234567, 0, c.lang); got != c.want {
			t.Errorf("lang=%s: got %q, want %q", c.lang, got, c.want)
		}
	}
}

func TestFmtNumLangNegative(t *testing.T) {
	if got := fmtNumLang(-1234.56, 2, "de"); got != "-1.234,56" {
		t.Errorf("de negative: got %q, want %q", got, "-1.234,56")
	}
	if got := fmtNumLang(-1234.56, 2, "fr"); got != "-1 234,56" {
		t.Errorf("fr negative: got %q, want %q", got, "-1 234,56")
	}
}

func TestFmtChangeLang(t *testing.T) {
	if got := fmtChangeLang(500.5, 2, "de"); got != "+500,50" {
		t.Errorf("de change: got %q, want %q", got, "+500,50")
	}
	if got := fmtChangeLang(-200, 2, "fr"); got != "-200,00" {
		t.Errorf("fr change: got %q, want %q", got, "-200,00")
	}
}

// --- Currency symbol ---

func TestCurrencySymbol(t *testing.T) {
	cases := []struct {
		code, want string
	}{
		{"JPY", "¥"},
		{"CNY", "¥"},
		{"USD", "$"},
		{"EUR", "€"},
		{"GBP", "£"},
		{"KRW", "₩"},
		{"AUD", "$"},
		{"", ""},
		{"XYZ", ""},
	}
	for _, c := range cases {
		if got := currencySymbol(c.code); got != c.want {
			t.Errorf("currencySymbol(%q) = %q, want %q", c.code, got, c.want)
		}
	}
}

func TestCurrencySymbolWidth(t *testing.T) {
	// All currency symbols should be width-1 (half-width)
	for _, sym := range []string{"¥", "$", "€", "£", "₩"} {
		if w := stringWidth(sym); w != 1 {
			t.Errorf("stringWidth(%q) = %d, want 1", sym, w)
		}
	}
}

func TestCurrencyPrefixTileAlignment(t *testing.T) {
	// Render tile with currency symbol; all lines must have identical stringWidth
	r := &fetcher.Result{Price: 39500.50, PrevClose: 39000.0, Change: 500.50, ChangePct: 1.28, Series: []float64{39000, 39200, 39500.5}, Currency: "JPY"}
	item := symbols.Item{Name: "Nikkei", Symbol: "^N225", Decimals: 2, Country: "JP"}
	lines := renderTileL(item, r, 40, 3, false, false, true, false, "Japan", "ja", defaultTheme)
	expected := 40
	for i, ln := range lines {
		if w := stringWidth(ln); w != expected {
			t.Errorf("line %d width = %d, want %d: %q", i, w, expected, ln)
		}
	}
	// Verify the ¥ symbol is present in the price line
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "¥") {
		t.Errorf("expected ¥ symbol in tile:\n%s", joined)
	}
}

func TestCurrencyPrefixWidthConsistency(t *testing.T) {
	// Test that adding a currency symbol keeps stringWidth consistent
	// (symbol adds exactly 1 to width)
	base := fmtNumLang(39500.50, 2, "en")
	withSym := "¥" + base
	if stringWidth(withSym) != stringWidth(base)+1 {
		t.Errorf("currency prefix should add exactly 1 to width: base=%d, withSym=%d",
			stringWidth(base), stringWidth(withSym))
	}
}

// --- NO_COLOR resolution ---

func TestResolveNoColor(t *testing.T) {
	// Flag overrides everything
	if !ResolveNoColor(true, "", false) {
		t.Error("flag=true should disable color")
	}
	// NO_COLOR present (empty value) disables
	if !ResolveNoColor(false, "", true) {
		t.Error("NO_COLOR present (empty) should disable color")
	}
	// NO_COLOR present (non-empty) disables
	if !ResolveNoColor(false, "1", true) {
		t.Error("NO_COLOR present should disable color")
	}
	// Neither flag nor env -> no disable
	if ResolveNoColor(false, "", false) {
		t.Error("no flag, no env should not disable color")
	}
}

// --- Theme selection ---

func TestThemeByNameDefault(t *testing.T) {
	th := ThemeByName("default")
	if th.UpColor == "" || th.DownColor == "" {
		t.Error("default theme should have up/down colors")
	}
}

func TestThemeByNameMono(t *testing.T) {
	th := ThemeByName("mono")
	if th.UpColor != "" || th.DownColor != "" {
		t.Errorf("mono theme should have empty up/down colors, got up=%q down=%q", th.UpColor, th.DownColor)
	}
}

func TestThemeByNameHighcontrast(t *testing.T) {
	th := ThemeByName("highcontrast")
	def := ThemeByName("default")
	if th.UpColor == def.UpColor || th.DownColor == def.DownColor {
		t.Error("highcontrast should differ from default")
	}
}

func TestThemeByNameLight(t *testing.T) {
	th := ThemeByName("light")
	def := ThemeByName("default")
	if th.UpColor == def.UpColor && th.DownColor == def.DownColor {
		t.Error("light should differ from default in at least one color")
	}
}

func TestThemeByNameUnknownFallback(t *testing.T) {
	th := ThemeByName("nonexistent")
	def := ThemeByName("default")
	if th.UpColor != def.UpColor {
		t.Error("unknown name should fall back to default")
	}
}

func TestThemeRGComposition(t *testing.T) {
	th := ThemeByName("highcontrast")
	// Normal: up=blue, down=orange
	up := colorForTheme(1.0, true, false, th)
	down := colorForTheme(-1.0, true, false, th)
	// With --rg: reversed
	upRG := colorForTheme(1.0, true, true, th)
	downRG := colorForTheme(-1.0, true, true, th)
	if up != th.UpColor {
		t.Errorf("up should be theme.UpColor")
	}
	if down != th.DownColor {
		t.Errorf("down should be theme.DownColor")
	}
	// rg reversal
	if upRG != th.DownColor {
		t.Errorf("rg up should be theme.DownColor, got %q", upRG)
	}
	if downRG != th.UpColor {
		t.Errorf("rg down should be theme.UpColor, got %q", downRG)
	}
}

func TestMonoThemeNoGreenRed(t *testing.T) {
	th := ThemeByName("mono")
	// colorForTheme should return empty for mono
	if got := colorForTheme(1.0, true, false, th); got != "" {
		t.Errorf("mono up color should be empty, got %q", got)
	}
	if got := colorForTheme(-1.0, true, false, th); got != "" {
		t.Errorf("mono down color should be empty, got %q", got)
	}
}

// --- RangeLabel in header ---

func TestRangeLabelInHeader(t *testing.T) {
	out := RenderDashboard(nil, []string{"japan"}, Options{NoColor: true, TermWidth: 120, RangeLabel: "1mo"})
	header := strings.SplitN(out, "\n", 2)[0]
	if !strings.Contains(header, "[1mo]") {
		t.Errorf("header should contain [1mo]: %q", header)
	}
}

func TestRangeLabelEmpty(t *testing.T) {
	out := RenderDashboard(nil, []string{"japan"}, Options{NoColor: true, TermWidth: 120, RangeLabel: ""})
	header := strings.SplitN(out, "\n", 2)[0]
	if strings.Contains(header, "[") {
		t.Errorf("header should not contain bracket when RangeLabel empty: %q", header)
	}
}

// --- Full-width alignment with currency ---

func TestTileAllLinesEqualWidth(t *testing.T) {
	// Render a tile with EUR currency and verify every line has the same stringWidth
	r := &fetcher.Result{Price: 15000.50, PrevClose: 14900, Change: 100.50, ChangePct: 0.67, Series: []float64{14900, 15000, 15000.5}, Currency: "EUR"}
	item := symbols.Item{Name: "DAX", Symbol: "^GDAXI", Decimals: 2, Country: "DE"}
	lines := renderTileL(item, r, 36, 3, false, false, true, false, "Europe", "de", defaultTheme)
	for i, ln := range lines {
		if w := stringWidth(ln); w != 36 {
			t.Errorf("line %d width = %d, want 36: %q", i, w, ln)
		}
	}
}
