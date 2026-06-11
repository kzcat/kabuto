package render

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/kaz/sekai-kabuka/internal/fetcher"
	"github.com/kaz/sekai-kabuka/internal/symbols"
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
	// 単調増加 → 最初が最小(▁)、最後が最大(█)
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
	// 全要素のルーンが spark セットに含まれる
	for _, r := range rs {
		if !strings.ContainsRune(sparkRunes, r) {
			t.Errorf("rune %q not in spark set", string(r))
		}
	}
}

func TestSparklineFlat(t *testing.T) {
	// 全て同値 → span=0 で最低レベル、パニックしない
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
	// 100点を10幅にダウンサンプル
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
		width int
		want  int // 期待する列数(minTileW=24)
	}{
		{60, 2},   // 60/24 = 2
		{100, 4},  // 100/24 = 4
		{200, 8},  // 200/24 = 8
		{80, 3},   // 80/24 = 3
		{10, 1},   // 極小でも1列
		{0, 3},    // 0は80扱い → 3列
	}
	for _, tt := range tests {
		got := gridColumns(tt.width)
		if got != tt.want {
			t.Errorf("gridColumns(%d) = %d, want %d", tt.width, got, tt.want)
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
		// 余りは左の列から配分: 隣接列の差は高々1で左>=右
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
	// 非watch相当(termRows<=0)は2
	if got := chartRows(0, 5, 10); got != 2 {
		t.Errorf("chartRows(0,..) = %d, want 2", got)
	}
	// 下限1
	if got := chartRows(10, 5, 20); got < 1 {
		t.Errorf("chartRows lower bound: got %d", got)
	}
	if got := chartRows(10, 5, 20); got != 1 {
		t.Errorf("tiny terminal: got %d, want 1", got)
	}
	// 上限8
	if got := chartRows(500, 5, 2); got != 8 {
		t.Errorf("large terminal: got %d, want 8", got)
	}
	// 通常: termRows=50, header=5, tileRows=10 → avail=45, tileH=4, N=1
	if got := chartRows(50, 5, 10); got != 1 {
		t.Errorf("chartRows(50,5,10) = %d, want 1", got)
	}
	// termRows=80, header=5, tileRows=5 → avail=75, tileH=15, N=12→8
	if got := chartRows(80, 5, 5); got != 8 {
		t.Errorf("chartRows(80,5,5) = %d, want 8", got)
	}
}

func TestSparklineRows(t *testing.T) {
	// 単調増加・2行: 行数2、各行幅5
	rows := SparklineRows([]float64{1, 2, 3, 4, 5}, 5, 2)
	if len(rows) != 2 {
		t.Fatalf("got %d rows, want 2", len(rows))
	}
	for _, r := range rows {
		if w := len([]rune(r)); w != 5 {
			t.Errorf("row width: got %d, want 5", w)
		}
	}
	// 最大値の列(最後)は下の行が█、上の行も埋まっているはず
	bottom := []rune(rows[1])
	if bottom[len(bottom)-1] != '█' {
		t.Errorf("max column bottom row: got %q, want █", string(bottom[len(bottom)-1]))
	}
	// 最小値の列(最初)は下の行に部分ブロック、上の行は空白
	top := []rune(rows[0])
	if top[0] != ' ' {
		t.Errorf("min column top row: got %q, want space", string(top[0]))
	}
}

func TestSparklineRowsEmpty(t *testing.T) {
	rows := SparklineRows(nil, 4, 3)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	for _, r := range rows {
		if strings.TrimSpace(r) != "" {
			t.Errorf("empty series should yield blanks, got %q", r)
		}
		if len([]rune(r)) != 4 {
			t.Errorf("blank row width: got %d, want 4", len([]rune(r)))
		}
	}
}

func TestSparklineRowsFlat(t *testing.T) {
	// 全同値はパニックしない
	rows := SparklineRows([]float64{5, 5, 5}, 3, 2)
	if len(rows) != 2 {
		t.Errorf("got %d rows", len(rows))
	}
}

// TestLayoutWidths は幅 60/100/200 でレイアウトが端末幅を超えないこと、
// 期待列数で並ぶことを検証する。
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
		if gridColumns(c.width) != c.wantCols {
			t.Errorf("width %d: cols = %d, want %d", c.width, gridColumns(c.width), c.wantCols)
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
	lines := renderTile(item, nil, 27, 2, false, false, true)
	// 上枠+情報1+チャート2+下枠 = 5
	if len(lines) != 5 {
		t.Fatalf("expected 5 lines, got %d", len(lines))
	}
	if !strings.Contains(lines[1], "N/A") {
		t.Errorf("expected N/A in tile, got %q", lines[1])
	}
	// ASCII 罫線
	if !strings.Contains(lines[0], "+") {
		t.Errorf("expected ASCII border, got %q", lines[0])
	}
}

func TestRenderTileWithData(t *testing.T) {
	r := &fetcher.Result{Price: 39500.50, PrevClose: 39000.0, Change: 500.50, ChangePct: 1.28, Time: "15:00", Series: []float64{1, 2, 3, 4}}
	lines := renderTile(symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2}, r, 40, 2, false, false, true)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "39,500.50") {
		t.Errorf("missing price:\n%s", joined)
	}
	if !strings.Contains(joined, "+500.50") {
		t.Errorf("missing change:\n%s", joined)
	}
	// 行数 = 2(チャート) + 3 = 5
	if len(lines) != 5 {
		t.Errorf("expected 5 lines, got %d", len(lines))
	}
}

func TestRenderTileChartRows(t *testing.T) {
	r := &fetcher.Result{Price: 100, Change: 1, ChangePct: 1, Series: []float64{1, 2, 3, 4, 5}}
	for _, n := range []int{1, 4, 8} {
		lines := renderTile(symbols.Item{Name: "X", Symbol: "X", Decimals: 2}, r, 30, n, false, false, true)
		if len(lines) != n+3 {
			t.Errorf("chartN=%d: expected %d lines, got %d", n, n+3, len(lines))
		}
	}
}

func TestRenderDashboardNA(t *testing.T) {
	data := map[string]*fetcher.Result{"^N225": nil}
	out := RenderDashboard(data, []string{"japan"}, Options{NoColor: true, TermWidth: 80})
	if !strings.Contains(out, "N/A") {
		t.Error("expected N/A in output")
	}
	if !strings.Contains(out, "日本") {
		t.Error("expected section title")
	}
	// 非カラーは ANSI エスケープを含まない
	if strings.Contains(out, "\033[") {
		t.Error("no-color output must not contain ANSI escapes")
	}
}

func TestRenderJSON(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.50, PrevClose: 39000.0, Change: 500.50, ChangePct: 1.28, Time: "15:00", Series: []float64{39000, 39500.5}},
		"NKD=F":    nil,
		"USDJPY=X": nil,
	}
	out := RenderJSON(data, []string{"japan"})
	if !strings.Contains(out, `"price": 39500.5`) {
		t.Errorf("unexpected JSON:\n%s", out)
	}
	if !strings.Contains(out, `"price": null`) {
		t.Errorf("expected null for missing data:\n%s", out)
	}
	if !strings.Contains(out, `"series"`) {
		t.Errorf("expected series field:\n%s", out)
	}
	// series が正しくパースできるか
	var parsed map[string]JSONSection
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	jp := parsed["japan"]
	if len(jp.Items) == 0 || len(jp.Items[0].Series) != 2 {
		t.Errorf("series not serialized correctly: %+v", jp.Items)
	}
}
