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
		min   int // 期待する最小列数
	}{
		{80, 2},  // 80桁で2列以上
		{120, 4}, // 120桁で4列以上
		{10, 1},  // 極小でも1列
		{0, 2},   // 0は80扱い → 2列
	}
	for _, tt := range tests {
		got := gridColumns(tt.width)
		if got < tt.min {
			t.Errorf("gridColumns(%d) = %d, want >= %d", tt.width, got, tt.min)
		}
		if got < 1 {
			t.Errorf("gridColumns(%d) = %d, want >= 1", tt.width, got)
		}
	}
}

func TestRenderTileNA(t *testing.T) {
	item := symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2}
	lines := renderTile(item, nil, false, false, true)
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
	lines := renderTile(symbols.Item{Name: "日経平均", Symbol: "^N225", Decimals: 2}, r, false, false, true)
	joined := strings.Join(lines, "\n")
	if !strings.Contains(joined, "39,500.50") {
		t.Errorf("missing price:\n%s", joined)
	}
	if !strings.Contains(joined, "+500.50") {
		t.Errorf("missing change:\n%s", joined)
	}
	if !strings.Contains(joined, "15:00") {
		t.Errorf("missing time:\n%s", joined)
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
