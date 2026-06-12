package render

import (
	"strings"
	"testing"
	"time"

	"github.com/kaz/sekai-kabuka/internal/fetcher"
	"github.com/kaz/sekai-kabuka/internal/symbols"
)

// TestTileLayoutOrder は新レイアウト(上=バッジ、中=チャート、下=現在値+前日比)を検証する。
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

// TestCountryCodeOnBorder は上枠線に国コード [JP] が銘柄名の前に入ることを検証する。
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

// TestCountryCodeOmitted は Country が空のとき国コードが付かないことを検証する。
func TestCountryCodeOmitted(t *testing.T) {
	item := symbols.Item{Name: "BTCドル", Symbol: "BTC-USD", Decimals: 2, Country: ""}
	r := &fetcher.Result{Price: 100, Change: 1, ChangePct: 1, Series: []float64{1, 2, 3}}
	lines := renderTile(item, r, 40, 2, false, false, false, false, "暗号資産")
	if strings.Contains(lines[0], "[") {
		t.Errorf("no country code expected: %q", lines[0])
	}
}

// TestChartCellRowFor は基準線のセル行量子化を検証する(0=最上行)。
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

// TestBaselineRendered は前日終値の赤い基準線がチャート行に描かれることを検証する(useColor時)。
func TestBaselineRendered(t *testing.T) {
	r := &fetcher.Result{Price: 105, PrevClose: 100, Change: 5, ChangePct: 5, Series: []float64{95, 100, 105}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 2}
	lines := renderTile(item, r, 28, 4, true, false, false, false, "日本")
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, redDashed) {
		t.Errorf("baseline (red) escape not found in chart:\n%s", joined)
	}
}

// TestGuidelinesWhenTall は N>=4 かつ +/-1% が範囲内のときガイドライン(bright black 点線)が描かれることを検証する。
func TestGuidelinesWhenTall(t *testing.T) {
	cc := chartColors{use: true, base: greenRGB, mono: green, reset: reset}
	rows := buildChartLines([]float64{90, 100, 110}, 100, 20, 6, 0, 2, cc)
	joined := strings.Join(rows, "")
	if !strings.Contains(joined, brightBlk) {
		t.Errorf("guideline (bright black) not rendered")
	}
}

// TestNoGuidelinesWhenShort は N<4 のときガイドラインが描かれないことを検証する。
func TestNoGuidelinesWhenShort(t *testing.T) {
	cc := chartColors{use: true, base: greenRGB, mono: green, reset: reset}
	rows := buildChartLines([]float64{90, 100, 110}, 100, 20, 3, 0, 2, cc)
	joined := strings.Join(rows, "")
	if strings.Contains(joined, brightBlk) {
		t.Errorf("no guideline expected for N<4, but bright black found")
	}
}

// TestHighLowLabels はタイル幅>=30 のとき高値(右上)・安値(右下)ラベルが描かれることを検証する。
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

// TestNoLabelsNarrow はタイル幅<30 のとき高値・安値ラベルが付かないことを検証する。
func TestNoLabelsNarrow(t *testing.T) {
	r := &fetcher.Result{Price: 105, PrevClose: 100, Change: 5, ChangePct: 5, Series: []float64{95, 12345, 90, 105}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 2}
	lines := renderTile(item, r, 28, 4, false, false, false, false, "日本")
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "12,345") || strings.Contains(joined, "12345") {
		t.Errorf("labels should be omitted for narrow tile:\n%s", joined)
	}
}

// TestIsClosed は閉場判定(epoch が30分以上古い)を検証する。
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

// TestClosedMarketGrey は閉場銘柄のチャートが bright black 単色で描かれ、赤基準線も grey になることを検証する。
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

// TestClockTilePresent は最終行に空きセルがあるとき時計タイルが描かれることを検証する。
func TestClockTilePresent(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: 60})
	if !strings.Contains(out, "時計") {
		t.Errorf("clock tile should be present when last row has empty cell:\n%s", out)
	}
}

// TestClockTileAbsent は最終行が埋まっているとき時計タイルが描かれないことを検証する。
// japan は5銘柄。幅150 → cols=min(5,6)=5 → 1段5銘柄で空きなし → 時計なし。
func TestClockTileAbsent(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: 150})
	if strings.Contains(out, "時計") {
		t.Errorf("clock tile should be absent when last row is full:\n%s", out)
	}
}

// TestClockTileDimensions は時計タイルの行数・幅が通常タイルと一致することを検証する。
func TestClockTileDimensions(t *testing.T) {
	lines := renderClockTile(40, 3, false, false)
	if len(lines) != 3+tileChrome {
		t.Errorf("clock tile lines = %d, want %d", len(lines), 3+tileChrome)
	}
	for i, ln := range lines {
		if w := stringWidth(ln); w != 40 {
			t.Errorf("clock tile line %d width = %d, want 40: %q", i, w, ln)
		}
	}
	if !strings.Contains(lines[0], "時計") {
		t.Errorf("clock tile title missing: %q", lines[0])
	}
}

// TestClockTileNoColorCompat は --no-color 時に時計タイルが ANSI エスケープを含まないことを検証する。
func TestClockTileNoColorCompat(t *testing.T) {
	lines := renderClockTile(40, 3, false, true)
	for _, ln := range lines {
		if strings.Contains(ln, "\033[") {
			t.Errorf("no-color clock tile must not contain ANSI escapes: %q", ln)
		}
	}
}

// TestNewLayoutNoColorCompat は新レイアウトでも --no-color 出力が ANSI エスケープを含まないことを検証する。
func TestNewLayoutNoColorCompat(t *testing.T) {
	r := &fetcher.Result{Price: 39500.5, PrevClose: 39000, Change: 500, ChangePct: 1.28, Series: []float64{39000, 39500.5}}
	lines := renderTile(symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2, Country: "JP"}, r, 40, 4, false, false, true, false, "日本")
	for _, ln := range lines {
		if strings.Contains(ln, "\033[") {
			t.Errorf("no-color tile must not contain ANSI escapes: %q", ln)
		}
	}
}
