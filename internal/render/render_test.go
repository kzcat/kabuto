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
		{60, 2},  // 60/24 = 2
		{100, 4}, // 100/24 = 4
		{200, 8}, // 200/24 = 8
		{80, 3},  // 80/24 = 3
		{10, 1},  // 極小でも1列
		{0, 3},   // 0は80扱い → 3列
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
	// 非TTY相当(termRows<=0)は2
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
	// 上限12: 巨大端末
	if got := chartRows(500, 5, 2); got != 12 {
		t.Errorf("large terminal: got %d, want 12", got)
	}
	// 通常: termRows=50, header=5, tileRows=10 → avail=45, tileH=4, N=1
	if got := chartRows(50, 5, 10); got != 1 {
		t.Errorf("chartRows(50,5,10) = %d, want 1", got)
	}
	// termRows=80, header=5, tileRows=5 → avail=75, tileH=15, N=12
	if got := chartRows(80, 5, 5); got != 12 {
		t.Errorf("chartRows(80,5,5) = %d, want 12", got)
	}
	// 上限ちょうど: tileH=15 → N=12
	if got := chartRows(80, 5, 5); got != 12 {
		t.Errorf("cap exactly 12: got %d", got)
	}
}

func TestChartRowsPerStage(t *testing.T) {
	// 非TTY(termRows<=0): 全段 N=2 固定
	ns := chartRowsPerStage(0, 5, 4)
	if len(ns) != 4 {
		t.Fatalf("got %d stages, want 4", len(ns))
	}
	for i, n := range ns {
		if n != 2 {
			t.Errorf("non-TTY stage %d: got %d, want 2", i, n)
		}
	}

	// 余り行配分: termRows=46, header=4 → avail=42, totalTileRows=4
	// tileH = 42/4 = 10, rem = 2, baseN = 10-3 = 7
	// 上の2段は +1 → [8, 8, 7, 7]
	ns = chartRowsPerStage(46, 4, 4)
	want := []int{8, 8, 7, 7}
	for i := range want {
		if ns[i] != want[i] {
			t.Errorf("remainder distribution stage %d: got %d, want %d (all=%v)", i, ns[i], want[i], ns)
		}
	}
	// 余り行を含めると合計タイル高が avail に一致(余白行なし)
	sum := 0
	for _, n := range ns {
		sum += n + 3 // 各段の外形高 = N+3
	}
	if sum != 42 {
		t.Errorf("total tile height = %d, want avail=42 (no leftover rows)", sum)
	}

	// 下限1: 極小端末
	ns = chartRowsPerStage(10, 5, 20)
	for i, n := range ns {
		if n < 1 {
			t.Errorf("lower bound stage %d: got %d", i, n)
		}
	}

	// 上限12: 余りで加算しても 12 を超えない
	// avail=200, totalTileRows=4 → tileH=50, baseN=47→12, rem=0 → 全段12
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
		// すべて点字レンジ U+2800〜U+28FF
		for _, ru := range r {
			if ru < 0x2800 || ru > 0x28FF {
				t.Errorf("rune %U out of braille range", ru)
			}
		}
	}
}

func TestBrailleRowsAreaFill(t *testing.T) {
	// 単調増加: 最大値の x 点(右端)は最下行のセルがフル(全ドット立つ)に近い。
	// 最大値の列の最下行は、下端のドット(dot3/dot6/dot7/dot8)が立つはず。
	rows := BrailleRows([]float64{1, 2, 3, 4, 5, 6, 7, 8}, 4, 2)
	bottom := []rune(rows[len(rows)-1])
	last := bottom[len(bottom)-1]
	bits := int(last) - 0x2800
	// 最下段ドット(左列 dot3=0x04, dot7=0x40 / 右列 dot6=0x20, dot8=0x80)のいずれかが立つ
	if bits&(0x04|0x40|0x20|0x80) == 0 {
		t.Errorf("max column bottom cell has no bottom dots: %08b", bits)
	}
	// 最小値の列(左端)の最上行は空(全ドット消灯=U+2800)
	top := []rune(rows[0])
	if top[0] != 0x2800 {
		t.Errorf("min column top cell should be empty, got %U", top[0])
	}
}

func TestBrailleBitLayout(t *testing.T) {
	// 単一点・最大高さ: 最下段から最上段まで1列が全部立つことを確認(面塗り)
	// width=1, rows=1 → 2点 × 4段階。値を最大化して全ドットが立つことを確認。
	rows := BrailleRows([]float64{0, 1}, 1, 1)
	r := []rune(rows[0])[0]
	bits := int(r) - 0x2800
	// x=0 は値0(level=0、最下段のみ)、x=1 は値1(level=3、全段)
	// 左列(x=0): 最下段ドット dot7=0x40。右列(x=1): dot4..dot8 全部 = 0x08|0x10|0x20|0x80
	wantRight := 0x08 | 0x10 | 0x20 | 0x80
	if bits&wantRight != wantRight {
		t.Errorf("right column not fully filled: %08b", bits)
	}
	if bits&0x40 == 0 {
		t.Errorf("left column bottom dot (dot7=0x40) not set: %08b", bits)
	}
}

func TestBrailleQuantize(t *testing.T) {
	// 下から上への量子化: 値が大きいほど高い段が立つ。
	// rows=2 → 8段階。最大値の列は上段セルにもドットが及ぶ。
	rows := BrailleRows([]float64{0, 10}, 1, 2)
	topCell := []rune(rows[0])[0]
	// 最大値 level=7 → h=7 まで立つ → cellY = 2-1-7/4 = 0(最上行)に到達
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
	// 全同値はパニックしない・点字レンジ内
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
	// 最上行(row=0)は基準色そのまま
	top := gradientRGB(base, 0, 4)
	if top != base {
		t.Errorf("top row should equal base: got %v", top)
	}
	// 最下行(row=rows-1)は約50%
	bottom := gradientRGB(base, 3, 4)
	want := [3]int{100, 50, 25}
	if bottom != want {
		t.Errorf("bottom row should be ~50%%: got %v, want %v", bottom, want)
	}
	// rows=1 はそのまま
	if got := gradientRGB(base, 0, 1); got != base {
		t.Errorf("single row: got %v", got)
	}
}

func TestBadgeColorSwitching(t *testing.T) {
	// no-color は色なしテキスト(前後空白1)
	plain := buildBadge("▲+0.06%", 0.06, false, false)
	if plain != " ▲+0.06% " {
		t.Errorf("no-color badge: got %q", plain)
	}
	// 上昇は緑背景(42)、下落は赤背景(41)、変わらずは bright black(100)
	if !strings.Contains(buildBadge("x", 1, true, false), "42m") {
		t.Errorf("up badge should use green bg")
	}
	if !strings.Contains(buildBadge("x", -1, true, false), "41m") {
		t.Errorf("down badge should use red bg")
	}
	if !strings.Contains(buildBadge("x", 0, true, false), "100m") {
		t.Errorf("flat badge should use bright black bg")
	}
	// rg で背景反転: 上昇が赤背景
	if !strings.Contains(buildBadge("x", 1, true, true), "41m") {
		t.Errorf("rg up badge should use red bg")
	}
}

func TestChartGradientSwitch(t *testing.T) {
	r := &fetcher.Result{Price: 100, Change: 5, ChangePct: 1, Series: []float64{1, 2, 3, 4, 5}}
	item := symbols.Item{Name: "X", Symbol: "X", Decimals: 2}
	// truecolor 有効: チャート行に 38;2; を含む
	tc := renderTile(item, r, 30, 3, true, false, false, true)
	joinedTC := strings.Join(tc, "\n")
	if !strings.Contains(joinedTC, "38;2;") {
		t.Errorf("truecolor chart should contain 24bit fg escape")
	}
	// truecolor 無効: 単色(ESC[32m 緑)で 38;2; を含まない
	sc := renderTile(item, r, 30, 3, true, false, false, false)
	joinedSC := strings.Join(sc, "\n")
	if strings.Contains(joinedSC, "38;2;") {
		t.Errorf("single-color chart must not contain 24bit fg escape")
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
	lines := renderTile(item, nil, 27, 2, false, false, true, false)
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
	lines := renderTile(symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2}, r, 40, 2, false, false, true, false)
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
		lines := renderTile(symbols.Item{Name: "X", Symbol: "X", Decimals: 2}, r, 30, n, false, false, true, false)
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

// TestFillHeightExact は FillHeight + TermRows 指定時に、
// 出力行数が TermRows ちょうど(余白行なし)になることを検証する。
// japan(3銘柄), 幅30 → cols=1 → 3タイル段。header=1, sectionTitle=1。
// TermRows=40 → avail=40-2=38, tileH=38/3=12, baseN=9, rem=2 → 段N=[10,10,9]。
// タイル外形高 = (10+3)+(10+3)+(9+3)=38。出力 = header1+title1+38 = 40。
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

// TestFillHeightCappedDiff は N が上限12で頭打ちになる場合、
// 出力行数が TermRows 未満で、その差が「12上限とタイル段数の制約」で説明できることを検証する。
// japan(3銘柄), 幅100 → cols=4 → 1タイル段。header=1, title=1, TermRows=40。
// avail=38, tileH=38, baseN=35→12(上限), rem=0 → 段N=[12]。
// 出力 = 1+1+(12+3)=17。差 = 40-17=23 はチャート上限12の制約による説明可能な残り。
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
	want := 1 + 1 + (12 + 3) // header + title + 1段(N=12)
	if got != want {
		t.Errorf("capped FillHeight lines = %d, want %d (cap 12)", got, want)
	}
	if got >= 40 {
		t.Errorf("capped output should be < TermRows, got %d", got)
	}
}

// TestNonTTYFixedN は非TTY(FillHeight=false, Watch=false)では
// TermRows を指定しても N=2 固定で表示が変わらないことを検証する。
func TestNonTTYFixedN(t *testing.T) {
	data := map[string]*fetcher.Result{
		"^N225":    {Price: 39500.5, Change: 100, ChangePct: 0.25, Series: []float64{1, 2, 3}},
		"NKD=F":    {Price: 39400, Change: -50, ChangePct: -0.13, Series: []float64{3, 2, 1}},
		"USDJPY=X": {Price: 145.12, Change: 0.3, ChangePct: 0.2, Series: []float64{1, 1, 2}},
	}
	// 幅30→cols=1→3段。N=2固定なら各段 高さ5。出力 = 1+1+3*5 = 17。
	// TermRows を 40/100 と変えても結果は不変。
	for _, rows := range []int{0, 40, 100} {
		out := RenderDashboard(data, []string{"japan"}, Options{
			NoColor: true, TermWidth: 30, TermRows: rows, // FillHeight=false, Watch=false
		})
		got := len(strings.Split(out, "\n"))
		want := 1 + 1 + 3*(2+3)
		if got != want {
			t.Errorf("non-TTY TermRows=%d: lines = %d, want %d (N=2 fixed)", rows, got, want)
		}
	}
}

// TestPerStageVaryingN は段ごとに N が異なる描画(余り行配分)で
// レイアウトが破綻しない(各行が端末幅を超えない・空行がない)ことを検証する。
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
