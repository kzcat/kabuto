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
	green = "\033[32m"
	red   = "\033[31m"
	bold  = "\033[1m"
	reset = "\033[0m"
)

var jst = time.FixedZone("JST", 9*3600)

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
	// insert commas
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

func colorize(text string, change float64, useColor bool) string {
	if !useColor {
		return text
	}
	if change > 0 {
		return green + text + reset
	} else if change < 0 {
		return red + text + reset
	}
	return text
}

// RenderTable はテーブル文字列を生成
func RenderTable(data map[string]*fetcher.Result, sections []string, noColor bool) string {
	color := UseColor(noColor)
	keys := sections
	if len(keys) == 0 {
		keys = symbols.SectionOrder
	}
	var lines []string
	now := time.Now().In(jst).Format("2006-01-02 15:04:05 JST")
	lines = append(lines, "更新: "+now)
	lines = append(lines, "")

	const nameW = 16
	const priceW = 14
	const changeW = 12
	const pctW = 9
	const timeW = 6

	for _, secKey := range keys {
		sec := symbols.Sections[secKey]
		if color {
			lines = append(lines, bold+"[ "+sec.Title+" ]"+reset)
		} else {
			lines = append(lines, "[ "+sec.Title+" ]")
		}
		header := padRight("名称", nameW) + " " + padLeft("現在値", priceW) + " " + padLeft("前日比", changeW) + " " + padLeft("前日比%", pctW) + " " + padLeft("時刻", timeW)
		lines = append(lines, header)
		lines = append(lines, strings.Repeat("-", 65))

		for _, item := range sec.Items {
			r := data[item.Symbol]
			if r == nil {
				row := padRight(item.Name, nameW) + " " + padLeft("N/A", priceW) + " " + padLeft("N/A", changeW) + " " + padLeft("N/A", pctW) + " " + padLeft("N/A", timeW)
				lines = append(lines, row)
			} else {
				priceS := fmtNum(r.Price, item.Decimals)
				changeS := fmtChange(r.Change, item.Decimals)
				pctS := fmtPct(r.ChangePct)
				timeS := r.Time
				row := padRight(item.Name, nameW) + " " + padLeft(priceS, priceW) + " " + padLeft(changeS, changeW) + " " + padLeft(pctS, pctW) + " " + padLeft(timeS, timeW)
				lines = append(lines, colorize(row, r.Change, color))
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}

// JSONItem はJSON出力の1銘柄
type JSONItem struct {
	Name      string   `json:"name"`
	Symbol    string   `json:"symbol"`
	Price     *float64 `json:"price"`
	Change    *float64 `json:"change"`
	ChangePct *float64 `json:"change_pct"`
	Time      *string  `json:"time"`
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
				items = append(items, JSONItem{
					Name:      item.Name,
					Symbol:    item.Symbol,
					Price:     &price,
					Change:    &change,
					ChangePct: &pct,
					Time:      &t,
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
