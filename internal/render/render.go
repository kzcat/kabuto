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

// 騰落色のベース RGB(truecolor グラデーション用)
var (
	greenRGB = [3]int{0, 200, 0}
	redRGB   = [3]int{220, 40, 40}
)

var jst = time.FixedZone("JST", 9*3600)

// Options は描画オプション
type Options struct {
	NoColor    bool // 色を使わない
	RedGreen   bool // 上昇=赤/下落=緑 に反転(日本式)
	TermWidth  int  // 端末幅(0なら自動取得)
	TermRows   int  // 端末行数(0なら自動取得)
	Watch      bool // watch 時は高さを使い切る
	FillHeight bool // stdout が TTY のとき高さを使い切る(非watchの1回表示でも適用)
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

// truecolorSupported は環境変数 COLORTERM が truecolor / 24bit かを判定する。
func truecolorSupported() bool {
	ct := strings.ToLower(os.Getenv("COLORTERM"))
	return ct == "truecolor" || ct == "24bit"
}

// baseRGBFor は騰落値に応じたベース RGB を返す(rg反転対応)。
func baseRGBFor(change float64, redGreen bool) [3]int {
	up, down := greenRGB, redRGB
	if redGreen {
		up, down = redRGB, greenRGB
	}
	if change < 0 {
		return down
	}
	return up // 上昇・変わらずは up 色を基準にする
}

// fg24 は truecolor 前景色エスケープを返す。
func fg24(c [3]int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c[0], c[1], c[2])
}

// gradientRGB は最上行(基準色)から最下行(約50%暗)へ線形補間した row 番目の RGB を返す。
func gradientRGB(base [3]int, row, rows int) [3]int {
	if rows <= 1 {
		return base
	}
	// t=0(最上行)→1(最下行)。最下行は base の約50%。
	t := float64(row) / float64(rows-1)
	factor := 1.0 - 0.5*t
	var out [3]int
	for i := 0; i < 3; i++ {
		out[i] = int(math.Round(float64(base[i]) * factor))
	}
	return out
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

// brailleBase は点字ブロックの開始コードポイント(U+2800)。
const brailleBase = 0x2800

// brailleDotBits[col][rowInCell] は点字ドットのビット(Unicode 標準)。
// 左列(col=0)上から: dot1=0x01, dot2=0x02, dot3=0x04, dot7=0x40
// 右列(col=1)上から: dot4=0x08, dot5=0x10, dot6=0x20, dot8=0x80
var brailleDotBits = [2][4]int{
	{0x01, 0x02, 0x04, 0x40}, // 左列(上→下)
	{0x08, 0x10, 0x20, 0x80}, // 右列(上→下)
}

// downsample は系列を n 点に等間隔でダウンサンプルする(null 補間済みを前提)。
func downsample(series []float64, n int) []float64 {
	if n < 1 {
		n = 1
	}
	pts := make([]float64, n)
	if len(series) == 0 {
		return pts
	}
	if len(series) == 1 || n == 1 {
		v := series[len(series)-1]
		for i := range pts {
			pts[i] = v
		}
		return pts
	}
	for i := 0; i < n; i++ {
		idx := i * (len(series) - 1) / (n - 1)
		pts[i] = series[idx]
	}
	return pts
}

// BrailleRows は数値系列を点字エリアチャートとして width セル × rows セルで描く。
// 解像度は横 2*width 点 × 縦 4*rows 段階。系列を 2*width 点にダウンサンプルし、
// 各 x 点の値を 0..(4*rows-1) に量子化、その高さから下のドットをすべて立てる(面塗り)。
// 返り値は上から下へ rows 行(各行 width ルーン、すべて点字 U+2800〜U+28FF)。
func BrailleRows(series []float64, width, rows int) []string {
	if rows < 1 {
		rows = 1
	}
	if width < 1 {
		width = 1
	}
	out := make([]string, rows)
	blankCell := rune(brailleBase) // 全ドット消灯のセル
	if len(series) == 0 {
		blank := strings.Repeat(string(blankCell), width)
		for i := range out {
			out[i] = blank
		}
		return out
	}

	xPoints := 2 * width
	yLevels := 4 * rows
	pts := downsample(series, xPoints)

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

	// 各セルのビット値を蓄積する [row][col]
	cells := make([][]int, rows)
	for r := range cells {
		cells[r] = make([]int, width)
	}

	for x, v := range pts {
		// 量子化: 0..yLevels-1(下が 0、上が yLevels-1)
		var level int
		if span == 0 {
			level = 0 // フラットは最下段
		} else {
			level = int(math.Round((v - min) / span * float64(yLevels-1)))
		}
		if level < 0 {
			level = 0
		}
		if level >= yLevels {
			level = yLevels - 1
		}
		col := x % 2 // セル内左右(0=左, 1=右)
		cellX := x / 2
		if cellX >= width {
			cellX = width - 1
		}
		// 高さ level から下のドットをすべて立てる(面塗り)
		for h := 0; h <= level; h++ {
			cellY := rows - 1 - h/4 // 0=最上行
			rowInCell := 3 - h%4    // 下から数えた h を「セル内上→下」インデックスに変換
			if cellY < 0 || cellY >= rows {
				continue
			}
			cells[cellY][cellX] |= brailleDotBits[col][rowInCell]
		}
	}

	for r := 0; r < rows; r++ {
		var b strings.Builder
		for c := 0; c < width; c++ {
			b.WriteRune(rune(brailleBase + cells[r][c]))
		}
		out[r] = b.String()
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
// 列数は表示銘柄数 itemCount を超えない(itemCount<=0 のときは制約しない)。
func gridColumns(termWidth, itemCount int) int {
	if termWidth <= 0 {
		termWidth = 80
	}
	cols := termWidth / minTileW
	if cols < 1 {
		cols = 1
	}
	if itemCount > 0 && cols > itemCount {
		cols = itemCount
	}
	return cols
}

// chartRows は端末行数からチャート行数 N(全段共通の基準値)を計算する。
// tileRows = (termRows - header(1) - sectionTitles) / rowsOfTiles
// N = tileRows - 3(上下枠 + 情報行)。下限1・上限12。termRows<=0 は N=2(高さ計算なし)。
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
	if n > 12 {
		n = 12
	}
	return n
}

const (
	chartNMin = 1
	chartNMax = 12
)

// chartRowsPerStage は端末行数を totalTileRows 段に配分し、各段(上→下)のチャート行数 N を返す。
// 基準 N = tileH - 3(tileH = avail / totalTileRows)。均等割りで余った行数は、
// 上の段から順に 1 段につき 1 行ずつ N に加算して最終行が画面下端に届くようにする(余白行を残さない)。
// 下限 1・上限 12。N 上限とタイル段数の制約で配り切れない余りは残す。
func chartRowsPerStage(termRows, headerLines, totalTileRows int) []int {
	if totalTileRows <= 0 {
		return nil
	}
	out := make([]int, totalTileRows)
	if termRows <= 0 {
		// 非TTY(高さ計算なし): N=2 固定
		for i := range out {
			out[i] = 2
		}
		return out
	}
	avail := termRows - headerLines
	if avail < 1 {
		avail = 1
	}
	tileH := avail / totalTileRows
	rem := avail % totalTileRows // タイル高さの均等割りで余った行数(=配り切れていない下端の行)
	baseN := tileH - 3
	if baseN < chartNMin {
		baseN = chartNMin
	}
	if baseN > chartNMax {
		baseN = chartNMax
	}
	for i := range out {
		n := baseN
		// 余り行を上の段から 1 段 1 行ずつ加算(上限 12 を超えない範囲)
		if i < rem && n < chartNMax {
			n++
		}
		out[i] = n
	}
	return out
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

// buildTopBorder は上枠線を構築する。innerW は枠線内側の表示幅(全角=2換算)。
// 構成: bc.tl + bc.h + " " + name + " " + <dash...> + [" " + secName + " " + bc.h] + bc.tr
// secName が空ならセクション名を埋め込まない。dash 部の本数で innerW を厳密に充填する。
func buildTopBorder(bc boxChars, border, rst, name, secName string, innerW int) string {
	var b strings.Builder
	b.WriteString(border)
	b.WriteString(bc.tl)
	// 内側 innerW 桁を構築する。
	var inner strings.Builder
	inner.WriteString(bc.h)
	inner.WriteString(" ")
	inner.WriteString(name)
	inner.WriteString(" ")
	usedLeft := 2 + stringWidth(name) // bc.h(1) + " "(1) + name + " "(1) = 2 + nameW(+1 below)
	usedLeft++                        // 末尾の " "
	if secName != "" {
		// 右端: <dash...> + " " + secName + " " + bc.h
		secW := stringWidth(secName)
		rightFixed := 1 + secW + 1 + 1 // " "(1) + secName + " "(1) + bc.h(1)
		dashN := innerW - usedLeft - rightFixed
		if dashN < 1 {
			dashN = 1
		}
		inner.WriteString(strings.Repeat(bc.h, dashN))
		inner.WriteString(" ")
		inner.WriteString(secName)
		inner.WriteString(" ")
		inner.WriteString(bc.h)
	} else {
		dashN := innerW - usedLeft
		if dashN < 0 {
			dashN = 0
		}
		inner.WriteString(strings.Repeat(bc.h, dashN))
	}
	b.WriteString(inner.String())
	b.WriteString(bc.tr)
	b.WriteString(rst)
	return b.String()
}


// outerW = タイル外形幅(枠線込み)、chartN = チャート行数。
// secName が非空かつタイル幅 >= 30 桁のとき、上枠線の右端に bright black でセクション名を埋め込む。
// 返り行数は chartN + 3(上枠 + 情報行 + チャートN行 + 下枠)。
func renderTile(item symbols.Item, r *fetcher.Result, outerW, chartN int, useColor, redGreen, ascii, truecolor bool, secName string) []string {
	if outerW < minTileW {
		outerW = minTileW
	}
	if chartN < 1 {
		chartN = 1
	}
	innerW := outerW - 2   // 枠線2桁を除いた内側幅
	contentW := innerW - 2 // 左右1桁ずつの余白を除いた内容幅

	bc := getBoxChars(ascii)
	border := brightBlk
	wclr := boldWhite
	rst := reset
	if !useColor {
		border, wclr, rst = "", "", ""
	}

	// 見出し付き上辺: ┌─ 名称 ─...─ セクション ─┐
	// タイル幅 30 桁未満のときはセクション名を省略する。
	name := truncWidth(item.Name, innerW-4)
	nameW := stringWidth(name)
	// 上辺の構成(内側 innerW 桁分):
	//   bc.h + " " + name + " " + <dash...> + bc.tr
	// セクション名を入れる場合は <dash...> の右端に "─ secName ─" を埋め込む。
	// 全体: bc.h + " " + name + " " + dashLeft*"─" + " " + secName + " " + bc.h*1 まで
	// 内側幅 innerW のうち、固定で使う桁:
	//   先頭の bc.h(1) + " "(1) + name(nameW) + " "(1) = 3 + nameW
	usedLeft := 3 + nameW
	secLabel := ""
	if outerW >= 30 && secName != "" {
		// "─ secName ─" の表示幅 = 1(─) + 1(空) + secW + 1(空) + 1(─) = secW + 4
		secW := stringWidth(secName)
		need := secW + 4
		// 名称部の後に最低1本の dash を残せること
		if usedLeft+1+need <= innerW {
			secLabel = secName
		}
	}
	top := buildTopBorder(bc, border, rst, name, secLabel, innerW)

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
		blank := strings.Repeat(string(rune(brailleBase)), contentW)
		if ascii {
			// ASCII フォールバックでも点字は維持(SPEC: 罫線だけ ASCII)
			blank = strings.Repeat(string(rune(brailleBase)), contentW)
		}
		for i := 0; i < chartN; i++ {
			lines = append(lines, wrap(blank))
		}
		lines = append(lines, bottom)
		return lines
	}

	clr := colorFor(r.Change, useColor, redGreen)
	priceS := fmtNum(r.Price, item.Decimals)
	changeS := fmtChange(r.Change, item.Decimals)
	pctText := arrow(r.Change) + fmtPct(r.ChangePct)
	// 前日比% バッジ: 前後空白1 + 太字白文字 + 騰落色背景
	badge := buildBadge(pctText, r.Change, useColor, redGreen)
	badgeW := stringWidth(" " + pctText + " ") // バッジの表示幅(背景込み)

	// 情報行: 現在値(太字白)  前日比(騰落色)  バッジ(右寄せ)
	plainW := stringWidth(priceS) + 2 + stringWidth(changeS) + 2 + badgeW
	if plainW <= contentW {
		extra := contentW - plainW
		gapL := extra / 2
		gapR := extra - gapL
		infoColored := wclr + priceS + rst +
			strings.Repeat(" ", 2+gapL) + clr + changeS + rst +
			strings.Repeat(" ", 2+gapR) + badge
		lines = append(lines, wrap(infoColored))
	} else if stringWidth(priceS)+1+badgeW <= contentW {
		// 収まらない場合は price と バッジのみ
		gap := contentW - stringWidth(priceS) - badgeW
		infoColored := wclr + priceS + rst + strings.Repeat(" ", gap) + badge
		lines = append(lines, wrap(infoColored))
	} else {
		alt := truncWidth(priceS, contentW)
		infoColored := wclr + alt + rst
		lines = append(lines, wrap(padPlainRight(infoColored, alt, contentW)))
	}

	// チャート行: 点字エリアチャート
	rowsStr := BrailleRows(r.Series, contentW, chartN)
	base := baseRGBFor(r.Change, redGreen)
	for i, rowStr := range rowsStr {
		var c string
		switch {
		case !useColor:
			c = rowStr
		case truecolor:
			c = fg24(gradientRGB(base, i, len(rowsStr))) + rowStr + rst
		default:
			c = clr + rowStr + rst
		}
		lines = append(lines, wrap(c))
	}

	lines = append(lines, bottom)
	return lines
}

// buildBadge は前日比% バッジ ' ▲+0.06% ' を生成する。
// 騰落色の背景 + 太字白文字。下落=赤背景、変わらず=bright black 背景。
// redGreen で背景色を反転。useColor=false なら従来の色なしテキスト。
func buildBadge(text string, change float64, useColor, redGreen bool) string {
	content := " " + text + " "
	if !useColor {
		return content
	}
	// 背景色 SGR コード
	bgGreen, bgRed, bgGray := 42, 41, 100 // 100 = bright black bg
	up, down := bgGreen, bgRed
	if redGreen {
		up, down = bgRed, bgGreen
	}
	var bg int
	if change > 0 {
		bg = up
	} else if change < 0 {
		bg = down
	} else {
		bg = bgGray
	}
	// 太字 + 白文字 + 背景色
	return fmt.Sprintf("\033[1;37;%dm%s\033[0m", bg, content)
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

// flatItem は連続配置用の1銘柄(セクション名付き)。
type flatItem struct {
	item    symbols.Item
	secName string
}

// RenderDashboard は本家サイト風ダッシュボードを生成する。
// 表示対象の全銘柄を定義順のまま1つの N×M グリッドに行送りで敷き詰める(セクション見出しは廃止)。
func RenderDashboard(data map[string]*fetcher.Result, sections []string, opt Options) string {
	useColor := !opt.NoColor
	ascii := opt.NoColor // 非カラー時は ASCII 罫線にフォールバック
	truecolor := useColor && truecolorSupported()
	termWidth := opt.TermWidth
	termRows := opt.TermRows
	if termWidth <= 0 {
		termWidth = detectTermWidth()
	}

	keys := sections
	if len(keys) == 0 {
		keys = symbols.SectionOrder
	}

	// 表示対象の全銘柄を定義順のまま1列に展開する。
	var flat []flatItem
	for _, secKey := range keys {
		sec := symbols.Sections[secKey]
		for _, it := range sec.Items {
			flat = append(flat, flatItem{item: it, secName: sec.Title})
		}
	}

	// 列数 = max(1, termWidth/24)。ただし表示銘柄数を超えない。
	cols := gridColumns(termWidth, len(flat))
	colWidths := distributeWidths(termWidth, cols)

	// 全タイル段数 = ceil(銘柄数 / cols)
	totalTileRows := 0
	if len(flat) > 0 {
		totalTileRows = (len(flat) + cols - 1) / cols
	}
	headerLines := 1 // ヘッダー1行のみ(セクション見出し行は廃止)

	// チャート行数 N(段ごと)の決定
	var stageN []int
	if opt.Watch || opt.FillHeight {
		// TTY: 高さを使い切る。段ごとに N を配分(余り行は上の段から加算)。
		stageN = chartRowsPerStage(termRows, headerLines, totalTileRows)
	} else {
		// 非TTY(パイプ・リダイレクト): 高さ計算をせず N=2 固定
		stageN = make([]int, totalTileRows)
		for i := range stageN {
			stageN[i] = 2
		}
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

	// cols 列のグリッドに連続配置(行送り)。最終行のみ欠けてよい。
	stageIdx := 0
	for i := 0; i < len(flat); i += cols {
		end := i + cols
		if end > len(flat) {
			end = len(flat)
		}
		rowItems := flat[i:end]
		// この段のチャート行数 N(段ごとに異なりうる)
		chartN := 2
		if stageIdx < len(stageN) {
			chartN = stageN[stageIdx]
		}
		tileH := chartN + 3 // 上枠+情報1+チャートN+下枠
		// この行の各タイルを生成(列ごとに幅が異なる)
		var tiles [][]string
		for ci, fi := range rowItems {
			w := colWidths[ci]
			tiles = append(tiles, renderTile(fi.item, data[fi.item.Symbol], w, chartN, useColor, opt.RedGreen, ascii, truecolor, fi.secName))
		}
		// 行ごとに横連結(ギャップ0)
		for li := 0; li < tileH; li++ {
			var parts []string
			for _, t := range tiles {
				parts = append(parts, t[li])
			}
			lines = append(lines, strings.Join(parts, ""))
		}
		stageIdx++
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
