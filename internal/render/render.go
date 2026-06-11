package render

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/kaz/sekai-kabuka/internal/fetcher"
	"github.com/kaz/sekai-kabuka/internal/symbols"
)

const (
	green      = "\033[32m"
	red        = "\033[31m"
	bold       = "\033[1m"
	boldWhite  = "\033[1;37m"
	brightBlk  = "\033[90m"
	reverse    = "\033[7m"
	reset      = "\033[0m"
	sparkRunes = "▁▂▃▄▅▆▇█"
)

var jst = time.FixedZone("JST", 9*3600)

// Options は描画オプション
type Options struct {
	NoColor    bool // 色を使わない
	RedGreen   bool // 上昇=赤/下落=緑 に反転(日本式)
	TermWidth  int  // 端末幅(0なら自動取得)
}

// UseColor は色を使うかどうか判定
func UseColor(noColor bool) bool {
	if noColor {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// stringWidth は表示幅を返す(全角=2, 半角=1)
func stringWidth(s string) int {
	w := 0
	for _, r := range s {
		if isWide(r) {
			w += 2
		} else {
			w++
		}
	}
	return w
}

// isWide は East Asian Wide/Fullwidth を判定
func isWide(r rune) bool {
	if r >= 0x1100 && r <= 0x115F {
		return true
	}
	if r >= 0x2E80 && r <= 0x303E {
		return true
	}
	if r >= 0x3040 && r <= 0x33BF {
		return true
	}
	if r >= 0x3400 && r <= 0x4DBF {
		return true
	}
	if r >= 0x4E00 && r <= 0x9FFF {
		return true
	}
	if r >= 0xA000 && r <= 0xA4CF {
		return true
	}
	if r >= 0xAC00 && r <= 0xD7AF {
		return true
	}
	if r >= 0xF900 && r <= 0xFAFF {
		return true
	}
	if r >= 0xFE30 && r <= 0xFE6F {
		return true
	}
	if r >= 0xFF01 && r <= 0xFF60 {
		return true
	}
	if r >= 0xFFE0 && r <= 0xFFE6 {
		return true
	}
	if r >= 0x20000 && r <= 0x2FFFF {
		return true
	}
	if r >= 0x30000 && r <= 0x3FFFF {
		return true
	}
	// halfwidth katakana
	if r >= 0xFF61 && r <= 0xFF9F {
		return false
	}
	_ = unicode.Han
	return false
}

// truncWidth は表示幅 width に収まるよう文字列を切り詰める
func truncWidth(s string, width int) string {
	w := 0
	var b strings.Builder
	for _, r := range s {
		rw := 1
		if isWide(r) {
			rw = 2
		}
		if w+rw > width {
			break
		}
		b.WriteRune(r)
		w += rw
	}
	return b.String()
}

// padRight は表示幅 width に右パディング
func padRight(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}

// padLeft は表示幅 width に左パディング
func padLeft(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return strings.Repeat(" ", width-sw) + s
}

func fmtNum(value float64, decimals int) string {
	neg := value < 0
	if neg {
		value = -value
	}
	s := fmt.Sprintf("%.*f", decimals, value)
	parts := strings.Split(s, ".")
	intPart := parts[0]
	n := len(intPart)
	if n > 3 {
		var buf strings.Builder
		rem := n % 3
		if rem > 0 {
			buf.WriteString(intPart[:rem])
			if n > rem {
				buf.WriteByte(',')
			}
		}
		for i := rem; i < n; i += 3 {
			buf.WriteString(intPart[i : i+3])
			if i+3 < n {
				buf.WriteByte(',')
			}
		}
		intPart = buf.String()
	}
	result := intPart
	if len(parts) > 1 {
		result += "." + parts[1]
	}
	if neg {
		result = "-" + result
	}
	return result
}

func fmtChange(value float64, decimals int) string {
	if value > 0 {
		return "+" + fmtNum(value, decimals)
	}
	return fmtNum(value, decimals)
}

func fmtPct(value float64) string {
	s := fmt.Sprintf("%.2f%%", math.Abs(value))
	if value > 0 {
		return "+" + s
	} else if value < 0 {
		return "-" + s
	}
	return s
}

// colorFor は騰落値に応じた色エスケープを返す(rg反転対応)
func colorFor(change float64, useColor, redGreen bool) string {
	if !useColor {
		return ""
	}
	up, down := green, red
	if redGreen {
		up, down = red, green
	}
	if change > 0 {
		return up
	} else if change < 0 {
		return down
	}
	return ""
}

// arrow は騰落に応じた記号を返す
func arrow(change float64) string {
	if change > 0 {
		return "▲"
	} else if change < 0 {
		return "▼"
	}
	return "・"
}

// Sparkline は数値系列から Unicode スパークライン文字列を生成する。
// width が正なら系列を等間隔にダウンサンプルして width 文字に収める。
func Sparkline(series []float64, width int) string {
	if len(series) == 0 {
		return ""
	}
	runes := []rune(sparkRunes)
	// width 指定があればダウンサンプル
	pts := series
	if width > 0 && len(series) > width {
		pts = make([]float64, width)
		for i := 0; i < width; i++ {
			idx := i * (len(series) - 1) / (width - 1)
			if width == 1 {
				idx = len(series) - 1
			}
			pts[i] = series[idx]
		}
	}
	min, max := pts[0], pts[0]
	for _, v := range pts {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	var b strings.Builder
	span := max - min
	for _, v := range pts {
		var level int
		if span == 0 {
			level = 0
		} else {
			level = int(math.Round((v - min) / span * float64(len(runes)-1)))
		}
		if level < 0 {
			level = 0
		}
		if level >= len(runes) {
			level = len(runes) - 1
		}
		b.WriteRune(runes[level])
	}
	return b.String()
}

// 罫線文字セット
type boxChars struct {
	tl, tr, bl, br, h, v string
}

func getBoxChars(ascii bool) boxChars {
	if ascii {
		return boxChars{"+", "+", "+", "+", "-", "|"}
	}
	return boxChars{"┌", "┐", "└", "┘", "─", "│"}
}

const tileInnerW = 25 // タイル内側の表示幅
const tileOuterW = tileInnerW + 2

// renderTile は1銘柄のタイルを行配列として返す。色付けはエスケープ込み。
func renderTile(item symbols.Item, r *fetcher.Result, useColor, redGreen, ascii bool) []string {
	bc := getBoxChars(ascii)
	border := brightBlk
	wclr := boldWhite
	rst := reset
	if !useColor {
		border, wclr, rst = "", "", ""
	}

	// 見出し付き上辺: ┌─ 名称 ─...─┐
	name := truncWidth(item.Name, tileInnerW-4)
	nameW := stringWidth(name)
	dashAfter := tileInnerW - 2 - nameW - 1 // "─ " と name の後
	if dashAfter < 0 {
		dashAfter = 0
	}
	top := border + bc.tl + bc.h + " " + name + " " + strings.Repeat(bc.h, dashAfter) + bc.tr + rst

	bottom := border + bc.bl + strings.Repeat(bc.h, tileInnerW) + bc.br + rst

	left := border + bc.v + rst
	right := border + bc.v + rst

	var line1, line2, line3 string
	if r == nil {
		na := padRight("N/A", tileInnerW-2)
		line1 = left + " " + na + " " + right
		line2 = left + " " + padRight("", tileInnerW-2) + " " + right
		line3 = left + " " + padRight("", tileInnerW-2) + " " + right
		return []string{top, line1, line2, line3, bottom}
	}

	clr := colorFor(r.Change, useColor, redGreen)
	priceS := fmtNum(r.Price, item.Decimals)
	pctS := arrow(r.Change) + fmtPct(r.ChangePct)
	changeS := fmtChange(r.Change, item.Decimals)
	timeS := r.Time

	// 行1: 現在値(左, 太字白) ... 前日比%(右, 騰落色)
	gap1 := tileInnerW - 2 - stringWidth(priceS) - stringWidth(pctS)
	if gap1 < 1 {
		gap1 = 1
	}
	c1 := wclr + priceS + rst + strings.Repeat(" ", gap1) + clr + pctS + rst
	line1 = left + " " + c1 + " " + right

	// 行2: 前日比(左, 騰落色) ... 時刻(右)
	gap2 := tileInnerW - 2 - stringWidth(changeS) - stringWidth(timeS)
	if gap2 < 1 {
		gap2 = 1
	}
	c2 := clr + changeS + rst + strings.Repeat(" ", gap2) + timeS
	line2 = left + " " + c2 + " " + right

	// 行3: スパークライン(騰落色)
	spark := Sparkline(r.Series, tileInnerW-2)
	spark = padRight(spark, tileInnerW-2)
	c3 := clr + spark + rst
	line3 = left + " " + c3 + " " + right

	return []string{top, line1, line2, line3, bottom}
}

// gridColumns は端末幅からタイルの列数を計算する(全角=2考慮済みの固定タイル幅)
func gridColumns(termWidth int) int {
	if termWidth <= 0 {
		termWidth = 80
	}
	cols := termWidth / tileOuterW
	if cols < 1 {
		cols = 1
	}
	return cols
}

// detectTermWidth は $COLUMNS または ioctl から端末幅を取得する。不可なら 80。
func detectTermWidth() int {
	if c := os.Getenv("COLUMNS"); c != "" {
		var w int
		if _, err := fmt.Sscanf(c, "%d", &w); err == nil && w > 0 {
			return w
		}
	}
	if w := ioctlWidth(); w > 0 {
		return w
	}
	return 80
}

// RenderDashboard は本家サイト風ダッシュボードを生成する
func RenderDashboard(data map[string]*fetcher.Result, sections []string, opt Options) string {
	useColor := !opt.NoColor
	ascii := opt.NoColor // 非カラー時は ASCII 罫線にフォールバック
	termWidth := opt.TermWidth
	if termWidth <= 0 {
		termWidth = detectTermWidth()
	}
	cols := gridColumns(termWidth)

	keys := sections
	if len(keys) == 0 {
		keys = symbols.SectionOrder
	}

	var lines []string
	now := time.Now().In(jst).Format("2006-01-02 15:04:05 JST")
	// ascii モードでも日本語は表示できる前提(罫線のみフォールバック対象)
	header := "世界の株価 ─ sekai-kabuka CLI    更新: " + now
	if ascii {
		header = "世界の株価 - sekai-kabuka CLI    更新: " + now
	}
	if useColor {
		lines = append(lines, reverse+" "+header+" "+reset)
	} else {
		lines = append(lines, header)
	}
	lines = append(lines, "")

	for _, secKey := range keys {
		sec := symbols.Sections[secKey]
		title := "■ " + sec.Title
		if ascii {
			title = "# " + sec.Title
		}
		if useColor {
			lines = append(lines, bold+title+reset)
		} else {
			lines = append(lines, title)
		}

		// セクションのタイルを cols 列のグリッドに並べる
		var tiles [][]string
		for _, item := range sec.Items {
			tiles = append(tiles, renderTile(item, data[item.Symbol], useColor, opt.RedGreen, ascii))
		}
		for i := 0; i < len(tiles); i += cols {
			end := i + cols
			if end > len(tiles) {
				end = len(tiles)
			}
			row := tiles[i:end]
			// 各タイルは5行。行ごとに横連結
			for li := 0; li < 5; li++ {
				var parts []string
				for _, t := range row {
					parts = append(parts, t[li])
				}
				lines = append(lines, strings.Join(parts, " "))
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// JSONItem はJSON出力の1銘柄
type JSONItem struct {
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	Price     *float64  `json:"price"`
	Change    *float64  `json:"change"`
	ChangePct *float64  `json:"change_pct"`
	Time      *string   `json:"time"`
	Series    []float64 `json:"series"`
}

// JSONSection はJSON出力の1セクション
type JSONSection struct {
	Title string     `json:"title"`
	Items []JSONItem `json:"items"`
}

// RenderJSON はJSON出力文字列を生成
func RenderJSON(data map[string]*fetcher.Result, sections []string) string {
	keys := sections
	if len(keys) == 0 {
		keys = symbols.SectionOrder
	}
	output := make(map[string]JSONSection)
	for _, secKey := range keys {
		sec := symbols.Sections[secKey]
		var items []JSONItem
		for _, item := range sec.Items {
			r := data[item.Symbol]
			if r == nil {
				items = append(items, JSONItem{Name: item.Name, Symbol: item.Symbol})
			} else {
				price := roundTo(r.Price, item.Decimals)
				change := roundTo(r.Change, item.Decimals)
				pct := roundTo(r.ChangePct, 2)
				t := r.Time
				series := make([]float64, len(r.Series))
				for i, v := range r.Series {
					series[i] = roundTo(v, item.Decimals)
				}
				items = append(items, JSONItem{
					Name:      item.Name,
					Symbol:    item.Symbol,
					Price:     &price,
					Change:    &change,
					ChangePct: &pct,
					Time:      &t,
					Series:    series,
				})
			}
		}
		output[secKey] = JSONSection{Title: sec.Title, Items: items}
	}
	b, _ := json.MarshalIndent(output, "", "  ")
	return string(b)
}

func roundTo(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
