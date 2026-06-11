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
	NoColor   bool // 色を使わない
	RedGreen  bool // 上昇=赤/下落=緑 に反転(日本式)
	TermWidth int  // 端末幅(0なら自動取得)
	TermRows  int  // 端末行数(0なら自動取得)
	Watch     bool // watch 時は高さを使い切る
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

// SparklineRows は数値系列から N 行の複数行スパークラインを生成する。
// 系列を width 列にダウンサンプルし、各列の値を rows*8 段階に量子化する。
// レベルを含む行に部分ブロック(▁▂▃▄▅▆▇█)、それより下の行は █、上は空白で描く。
// 返り値は上から下へ rows 行(各行 width 文字)。
func SparklineRows(series []float64, width, rows int) []string {
	if rows < 1 {
		rows = 1
	}
	if width < 1 {
		width = 1
	}
	out := make([]string, rows)
	if len(series) == 0 {
		blank := strings.Repeat(" ", width)
		for i := range out {
			out[i] = blank
		}
		return out
	}
	runes := []rune(sparkRunes) // 8段階
	// width 列にダウンサンプル
	pts := make([]float64, width)
	if len(series) == 1 {
		for i := range pts {
			pts[i] = series[0]
		}
	} else if len(series) >= width {
		for i := 0; i < width; i++ {
			idx := i * (len(series) - 1) / (width - 1)
			if width == 1 {
				idx = len(series) - 1
			}
			pts[i] = series[idx]
		}
	} else {
		// 系列が幅より短い場合も等間隔マッピング(値を引き伸ばす)
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
	span := max - min
	steps := rows * len(runes) // 総段階数
	// 各列の量子化レベル(0..steps-1)
	rowsBuf := make([][]rune, rows)
	for r := range rowsBuf {
		rowsBuf[r] = make([]rune, width)
		for c := range rowsBuf[r] {
			rowsBuf[r][c] = ' '
		}
	}
	for c, v := range pts {
		var level int
		if span == 0 {
			level = 0 // フラットは最下行の最低ブロック
		} else {
			level = int(math.Round((v - min) / span * float64(steps-1)))
		}
		if level < 0 {
			level = 0
		}
		if level >= steps {
			level = steps - 1
		}
		// level が属する行(下が rows-1, 上が 0)と行内レベル
		rowFromBottom := level / len(runes)
		within := level % len(runes)
		topRow := rows - 1 - rowFromBottom // 0=最上行
		// 描画: topRow に部分ブロック、下の行(topRow+1..rows-1)は █、上は空白(初期値)
		rowsBuf[topRow][c] = runes[within]
		for r := topRow + 1; r < rows; r++ {
			rowsBuf[r][c] = '█'
		}
	}
	for r := range rowsBuf {
		out[r] = string(rowsBuf[r])
	}
	return out
}

const minTileW = 24 // 最小タイル外形幅(全角=2考慮済み)

// distributeWidths は端末幅を cols 列に配分する。
// 基本幅 = termWidth/cols、余り桁は左の列から1桁ずつ配って合計が termWidth に一致するようにする。
// 返り値は各列のタイル外形幅。
func distributeWidths(termWidth, cols int) []int {
	if cols < 1 {
		cols = 1
	}
	base := termWidth / cols
	rem := termWidth % cols
	widths := make([]int, cols)
	for i := range widths {
		widths[i] = base
		if i < rem {
			widths[i]++
		}
	}
	return widths
}

// gridColumns は端末幅からタイルの列数を計算する(最小タイル幅 minTileW)。
func gridColumns(termWidth int) int {
	if termWidth <= 0 {
		termWidth = 80
	}
	cols := termWidth / minTileW
	if cols < 1 {
		cols = 1
	}
	return cols
}

// chartRows は watch 時の端末行数からチャート行数 N を計算する。
// tileRows = (termRows - header(1) - sectionTitles) / rowsOfTiles
// N = tileRows - 3(上下枠 + 情報行)。下限1・上限8。非watch(termRows<=0)は N=2。
func chartRows(termRows, headerLines, totalTileRows int) int {
	if termRows <= 0 || totalTileRows <= 0 {
		return 2
	}
	avail := termRows - headerLines
	if avail < 1 {
		avail = 1
	}
	tileH := avail / totalTileRows
	n := tileH - 3
	if n < 1 {
		n = 1
	}
	if n > 8 {
		n = 8
	}
	return n
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

// renderTile は1銘柄のタイルを行配列として返す。
// outerW = タイル外形幅(枠線込み)、chartN = チャート行数。
// 返り行数は chartN + 3(上枠 + 情報行 + チャートN行 + 下枠 ではなく、上枠+情報1+チャートN+下枠 = N+3)。
func renderTile(item symbols.Item, r *fetcher.Result, outerW, chartN int, useColor, redGreen, ascii bool) []string {
	if outerW < minTileW {
		outerW = minTileW
	}
	if chartN < 1 {
		chartN = 1
	}
	innerW := outerW - 2 // 枠線2桁を除いた内側幅
	contentW := innerW - 2 // 左右1桁ずつの余白を除いた内容幅

	bc := getBoxChars(ascii)
	border := brightBlk
	wclr := boldWhite
	rst := reset
	if !useColor {
		border, wclr, rst = "", "", ""
	}

	// 見出し付き上辺: ┌─ 名称 ─...─┐
	name := truncWidth(item.Name, innerW-4)
	nameW := stringWidth(name)
	dashAfter := innerW - 2 - nameW - 1 // "─ " と name の後の余白
	if dashAfter < 0 {
		dashAfter = 0
	}
	top := border + bc.tl + bc.h + " " + name + " " + strings.Repeat(bc.h, dashAfter) + bc.tr + rst
	bottom := border + bc.bl + strings.Repeat(bc.h, innerW) + bc.br + rst

	left := border + bc.v + rst
	right := border + bc.v + rst

	wrap := func(inner string) string {
		return left + " " + inner + " " + right
	}

	lines := make([]string, 0, chartN+3)
	lines = append(lines, top)

	if r == nil {
		na := padRight("N/A", contentW)
		lines = append(lines, wrap(na))
		blank := padRight("", contentW)
		for i := 0; i < chartN; i++ {
			lines = append(lines, wrap(blank))
		}
		lines = append(lines, bottom)
		return lines
	}

	clr := colorFor(r.Change, useColor, redGreen)
	priceS := fmtNum(r.Price, item.Decimals)
	changeS := fmtChange(r.Change, item.Decimals)
	pctS := arrow(r.Change) + fmtPct(r.ChangePct)

	// 情報行: 現在値(太字白)  前日比(騰落色)  ▲%(騰落色) を右寄せ気味に配置
	// レイアウト: price <gap1> change <gap2> pct となるよう均等割り
	plain := priceS + "  " + changeS + "  " + pctS
	plainW := stringWidth(plain)
	if plainW <= contentW {
		// 余白を price と change の間、change と pct の間に配分(右端に pct)
		extra := contentW - plainW
		gapL := extra / 2
		gapR := extra - gapL
		infoColored := wclr + priceS + rst +
			strings.Repeat(" ", 2+gapL) + clr + changeS + rst +
			strings.Repeat(" ", 2+gapR) + clr + pctS + rst
		lines = append(lines, wrap(infoColored))
	} else {
		// 収まらない場合は price と pct のみ(change を省略)
		alt := priceS + " " + pctS
		if stringWidth(alt) > contentW {
			alt = truncWidth(priceS, contentW)
			infoColored := wclr + alt + rst
			lines = append(lines, wrap(padPlainRight(infoColored, alt, contentW)))
		} else {
			gap := contentW - stringWidth(alt)
			infoColored := wclr + priceS + rst + strings.Repeat(" ", gap+1) + clr + pctS + rst
			lines = append(lines, wrap(infoColored))
		}
	}

	// チャート行: 複数行スパークライン(騰落色)
	rowsStr := SparklineRows(r.Series, contentW, chartN)
	for _, rowStr := range rowsStr {
		c := clr + rowStr + rst
		lines = append(lines, wrap(c))
	}

	lines = append(lines, bottom)
	return lines
}

// padPlainRight は色エスケープ込み文字列 colored(表示は plain)を contentW に右パディングする。
func padPlainRight(colored, plain string, contentW int) string {
	pad := contentW - stringWidth(plain)
	if pad < 0 {
		pad = 0
	}
	return colored + strings.Repeat(" ", pad)
}

// detectTermSize は端末の桁数・行数を取得する。
func detectTermSize() (cols, rows int) {
	cols, rows = ioctlSize()
	if c := os.Getenv("COLUMNS"); c != "" {
		var w int
		if _, err := fmt.Sscanf(c, "%d", &w); err == nil && w > 0 {
			cols = w
		}
	}
	if l := os.Getenv("LINES"); l != "" {
		var h int
		if _, err := fmt.Sscanf(l, "%d", &h); err == nil && h > 0 {
			rows = h
		}
	}
	if cols <= 0 {
		cols = 80
	}
	return cols, rows
}

// DetectTermSize は端末の桁数・行数を公開する(main から利用)。
func DetectTermSize() (cols, rows int) {
	return detectTermSize()
}

// detectTermWidth は $COLUMNS または ioctl から端末幅を取得する。不可なら 80。
func detectTermWidth() int {
	c, _ := detectTermSize()
	return c
}

// RenderDashboard は本家サイト風ダッシュボードを生成する
func RenderDashboard(data map[string]*fetcher.Result, sections []string, opt Options) string {
	useColor := !opt.NoColor
	ascii := opt.NoColor // 非カラー時は ASCII 罫線にフォールバック
	termWidth := opt.TermWidth
	termRows := opt.TermRows
	if termWidth <= 0 {
		termWidth = detectTermWidth()
	}
	cols := gridColumns(termWidth)
	colWidths := distributeWidths(termWidth, cols)

	keys := sections
	if len(keys) == 0 {
		keys = symbols.SectionOrder
	}

	// チャート行数 N の決定
	chartN := 2
	if opt.Watch {
		// 全タイル段数 = 各セクションの (タイル数 / cols 切り上げ) の合計
		totalTileRows := 0
		for _, secKey := range keys {
			sec := symbols.Sections[secKey]
			n := len(sec.Items)
			rows := (n + cols - 1) / cols
			totalTileRows += rows
		}
		headerLines := 1 + len(keys) // ヘッダー1行 + セクション見出し行数
		chartN = chartRows(termRows, headerLines, totalTileRows)
	}

	var lines []string
	now := time.Now().In(jst).Format("2006-01-02 15:04:05 JST")
	header := "世界の株価 ─ sekai-kabuka CLI    更新: " + now
	if ascii {
		header = "世界の株価 - sekai-kabuka CLI    更新: " + now
	}
	if useColor {
		// 端末幅まで反転を伸ばす
		h := truncWidth(header, termWidth)
		hw := stringWidth(h)
		pad := termWidth - hw
		if pad < 0 {
			pad = 0
		}
		lines = append(lines, reverse+h+strings.Repeat(" ", pad)+reset)
	} else {
		lines = append(lines, truncWidth(header, termWidth))
	}
	// ヘッダー直後の空行は入れない

	for _, secKey := range keys {
		sec := symbols.Sections[secKey]
		title := "■ " + sec.Title
		if ascii {
			title = "# " + sec.Title
		}
		if useColor {
			lines = append(lines, brightBlk+truncWidth(title, termWidth)+reset)
		} else {
			lines = append(lines, truncWidth(title, termWidth))
		}
		// 見出し直後の空行は入れない

		// cols 列のグリッドに並べる。各行で列幅 colWidths を使う。
		items := sec.Items
		tileH := chartN + 3 // 上枠+情報1+チャートN+下枠
		for i := 0; i < len(items); i += cols {
			end := i + cols
			if end > len(items) {
				end = len(items)
			}
			rowItems := items[i:end]
			// この行の各タイルを生成(列ごとに幅が異なる)
			var tiles [][]string
			for ci, item := range rowItems {
				w := colWidths[ci]
				tiles = append(tiles, renderTile(item, data[item.Symbol], w, chartN, useColor, opt.RedGreen, ascii))
			}
			// 行ごとに横連結(ギャップ0)
			for li := 0; li < tileH; li++ {
				var parts []string
				for _, t := range tiles {
					parts = append(parts, t[li])
				}
				lines = append(lines, strings.Join(parts, ""))
			}
		}
		// セクション間の空行は入れない
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
