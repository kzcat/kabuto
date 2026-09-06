package render

import (
	"math"
	"strings"
)

// Sparkline generates a Unicode sparkline string from a numeric series.
// If width > 0, the series is downsampled to fit within width characters.
func Sparkline(series []float64, width int) string {
	if len(series) == 0 {
		return ""
	}
	runes := []rune(sparkRunes)
	// downsample if width is specified
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

// brailleBase is the braille block starting code point (U+2800).
const brailleBase = 0x2800

// brailleDotBits[col][rowInCell] maps braille dot bits per Unicode standard.
// Left column (col=0) top-to-bottom: dot1=0x01, dot2=0x02, dot3=0x04, dot7=0x40
// Right column (col=1) top-to-bottom: dot4=0x08, dot5=0x10, dot6=0x20, dot8=0x80
var brailleDotBits = [2][4]int{
	{0x01, 0x02, 0x04, 0x40}, // left column (top to bottom)
	{0x08, 0x10, 0x20, 0x80}, // right column (top to bottom)
}

// downsample reduces the series to n equally-spaced points (assumes null interpolation is done).
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

// BrailleRows renders a numeric series as a braille area chart in width cells x rows cells.
// Resolution is 2*width x-points by 4*rows y-levels. The series is downsampled to 2*width points,
// each point quantized to 0..(4*rows-1), and all dots below that height are set (area fill).
// Returns rows strings top-to-bottom (each width runes, all in braille range U+2800..U+28FF).
func BrailleRows(series []float64, width, rows int) []string {
	if rows < 1 {
		rows = 1
	}
	if width < 1 {
		width = 1
	}
	out := make([]string, rows)
	blankCell := rune(brailleBase) // empty cell (all dots off)
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

	// accumulate bit values per cell [row][col]
	cells := make([][]int, rows)
	for r := range cells {
		cells[r] = make([]int, width)
	}

	for x, v := range pts {
		// quantize: 0..yLevels-1 (0=bottom, yLevels-1=top)
		var level int
		if span == 0 {
			level = 0 // flat: bottom level
		} else {
			level = int(math.Round((v - min) / span * float64(yLevels-1)))
		}
		if level < 0 {
			level = 0
		}
		if level >= yLevels {
			level = yLevels - 1
		}
		col := x % 2 // left/right within cell (0=left, 1=right)
		cellX := x / 2
		if cellX >= width {
			cellX = width - 1
		}
		// set all dots from height level downward (area fill)
		for h := 0; h <= level; h++ {
			cellY := rows - 1 - h/4 // 0=top row
			rowInCell := 3 - h%4    // convert bottom-up h to top-down index within cell
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

// chartColors holds color escape sequences for chart rendering.
type chartColors struct {
	use       bool
	truecolor bool
	closed    bool   // market closed (grey monochrome)
	base      [3]int // gain/loss base RGB (truecolor gradient)
	mono      string // single-color escape (non-truecolor)
	reset     string
	symbol    string // chart symbol mode: ""|"braille"|"block"|"tty" (empty = braille)
	depth     int    // color depth (0 = legacy: truecolor field decides)
}

// gradFG returns the gradient foreground escape for the given chart row, honoring color depth.
// With cc.depth==0 (legacy) it falls back to truecolor when cc.truecolor is set, else mono.
func (cc chartColors) gradFG(row, rows int) string {
	rgb := gradientRGB(cc.base, row, rows)
	if cc.depth != 0 {
		return fgDepth(rgb, cc.depth)
	}
	if cc.truecolor {
		return fg24(rgb)
	}
	return cc.mono
}

// hasGradient reports whether colored gradient body cells should be used (vs mono).
func (cc chartColors) hasGradient() bool {
	if !cc.use || cc.closed {
		return false
	}
	if cc.depth != 0 {
		return true
	}
	return cc.truecolor
}

const redDashed = "\033[31m" // baseline (red)

// chartCellRowFor quantizes value v on scale [min,max] to a cell row index (0=top row).
// rows cells x 4 dots = 4*rows levels. Returns the cell row closest to v's height.
func chartCellRowFor(v, min, max float64, rows int) int {
	yLevels := 4 * rows
	var level int
	if max-min == 0 {
		level = 0
	} else {
		level = int(math.Round((v - min) / (max - min) * float64(yLevels-1)))
	}
	if level < 0 {
		level = 0
	}
	if level >= yLevels {
		level = yLevels - 1
	}
	// cell row (0=top) = rows-1 - level/4
	return rows - 1 - level/4
}

// buildChartLines renders a braille area chart (width x rows cells) with colored overlays:
//   - Scale includes both series min/max and prevClose
//   - Red dashed horizontal line at prevClose height (every other cell). Body dots take priority.
//   - rows>=4 and prevClose +/-1% in range: bright black dotted guideline (every 2 cells)
//   - labelW>0: right labelW columns reserved for high (top-right) and low (bottom-right) labels in bright black
//   - closed=true: chart and baseline drawn in bright black monochrome
func buildChartLines(series []float64, prevClose float64, width, rows, labelW int, decimals int, cc chartColors) []string {
	switch cc.symbol {
	case "block":
		return buildChartLinesBlock(series, prevClose, width, rows, labelW, decimals, cc)
	case "tty":
		return buildChartLinesTTY(series, prevClose, width, rows, labelW, decimals, cc)
	}
	// default / "braille": unchanged braille area chart
	if rows < 1 {
		rows = 1
	}
	if width < 1 {
		width = 1
	}
	chartW := width - labelW
	if chartW < 1 {
		chartW = width
		labelW = 0
	}

	out := make([]string, rows)
	blank := rune(brailleBase)

	// scale: include series min/max and prevClose
	hi, lo := prevClose, prevClose
	hasData := len(series) > 0
	if hasData {
		hi, lo = series[0], series[0]
		for _, v := range series {
			if v > hi {
				hi = v
			}
			if v < lo {
				lo = v
			}
		}
		if prevClose > hi {
			hi = prevClose
		}
		if prevClose < lo {
			lo = prevClose
		}
	}
	if hi == lo {
		hi += 1
		lo -= 1
	}

	// chart body cell bits (0=top row)
	cells := make([][]int, rows)
	bodyCell := make([][]bool, rows) // whether a cell has any body dots
	for r := range cells {
		cells[r] = make([]int, chartW)
		bodyCell[r] = make([]bool, chartW)
	}
	if hasData {
		xPoints := 2 * chartW
		yLevels := 4 * rows
		pts := downsample(series, xPoints)
		for x, v := range pts {
			level := int(math.Round((v - lo) / (hi - lo) * float64(yLevels-1)))
			if level < 0 {
				level = 0
			}
			if level >= yLevels {
				level = yLevels - 1
			}
			col := x % 2
			cellX := x / 2
			if cellX >= chartW {
				cellX = chartW - 1
			}
			for h := 0; h <= level; h++ {
				cellY := rows - 1 - h/4
				rowInCell := 3 - h%4
				if cellY < 0 || cellY >= rows {
					continue
				}
				cells[cellY][cellX] |= brailleDotBits[col][rowInCell]
				bodyCell[cellY][cellX] = true
			}
		}
	}

	// baseline (prevClose) cell row
	baseRow := chartCellRowFor(prevClose, lo, hi, rows)
	// +/-1% guideline cell rows (only if rows>=4 and in range)
	upGuide, downGuide := -1, -1
	if rows >= 4 {
		up := prevClose * 1.01
		dn := prevClose * 0.99
		if up <= hi && up >= lo {
			upGuide = chartCellRowFor(up, lo, hi, rows)
		}
		if dn <= hi && dn >= lo {
			downGuide = chartCellRowFor(dn, lo, hi, rows)
		}
	}

	mono := cc.mono
	rst := cc.reset
	guideClr := brightBlk
	baseClr := redDashed
	if cc.closed {
		// closed market: chart and baseline in bright black monochrome
		mono = brightBlk
		baseClr = brightBlk
	}
	if !cc.use {
		mono, rst, guideClr, baseClr = "", "", "", ""
	}

	// high/low label strings (bright black)
	var hiLabel, loLabel string
	if labelW > 0 && hasData {
		shi, slo := series[0], series[0]
		for _, v := range series {
			if v > shi {
				shi = v
			}
			if v < slo {
				slo = v
			}
		}
		hiLabel = truncWidth(fmtNum(shi, decimals), labelW)
		loLabel = truncWidth(fmtNum(slo, decimals), labelW)
	}

	for r := 0; r < rows; r++ {
		var b strings.Builder
		// chart body cells (color each cell character)
		for c := 0; c < chartW; c++ {
			cell := cells[r][c]
			isBody := bodyCell[r][c]
			switch {
			case isBody && cc.hasGradient():
				b.WriteString(cc.gradFG(r, rows))
				b.WriteRune(rune(brailleBase + cell))
				b.WriteString(rst)
			case isBody:
				b.WriteString(mono)
				b.WriteRune(rune(brailleBase + cell))
				b.WriteString(rst)
			case r == baseRow && c%2 == 0:
				// baseline: red dashed line (every other cell). Only where no body dots.
				b.WriteString(baseClr)
				b.WriteRune('\u2812') // mid-row dots (dot2+dot5) for horizontal line
				b.WriteString(rst)
			case (r == upGuide || r == downGuide) && c%2 == 0:
				// +/-1% guideline: bright black dotted (every 2 cells)
				b.WriteString(guideClr)
				b.WriteRune('\u2812')
				b.WriteString(rst)
			default:
				b.WriteRune(blank)
			}
		}
		// label area
		if labelW > 0 {
			lab := ""
			if r == 0 {
				lab = hiLabel
			} else if r == rows-1 {
				lab = loLabel
			}
			padded := padLeft(lab, labelW)
			if cc.use && lab != "" {
				b.WriteString(guideClr)
				b.WriteString(padded)
				b.WriteString(rst)
			} else {
				b.WriteString(padded)
			}
		}
		out[r] = b.String()
	}
	return out
}

// blockRunes is the 8-level block ramp (1/8 .. 8/8) plus blank at index 0.
var blockRunes = []rune{' ', '▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

// blockSymbolFor maps a quantized height level to a (cellRow, rune) for a block-element chart.
// level is in [0, 8*rows-1] (0=bottom). For the given cellRow (0=top), it returns:
//   - '█' (full) when the column reaches at or above the top of this cell,
//   - a partial block '▁'..'▇' for the topmost partially-filled cell,
//   - ' ' (blank) when the column does not reach this cell.
//
// It is a pure function to enable table-driven testing.
func blockSymbolFor(level, cellRow, rows int) rune {
	if rows < 1 {
		rows = 1
	}
	if level < 0 {
		level = 0
	}
	maxLevel := 8*rows - 1
	if level > maxLevel {
		level = maxLevel
	}
	// fill height measured from bottom, in 1/8 sub-cell units (0..8*rows).
	fill := level + 1
	// this cell spans sub-levels [cellBottom, cellBottom+8) measured from bottom.
	cellBottom := (rows - 1 - cellRow) * 8
	rem := fill - cellBottom
	switch {
	case rem >= 8:
		return '█'
	case rem <= 0:
		return ' '
	default:
		return blockRunes[rem]
	}
}

// buildChartLinesBlock renders an 8-level block-element area chart.
// Gradient color (depth-aware) is preserved; baseline/guidelines/labels match braille semantics.
func buildChartLinesBlock(series []float64, prevClose float64, width, rows, labelW int, decimals int, cc chartColors) []string {
	if rows < 1 {
		rows = 1
	}
	if width < 1 {
		width = 1
	}
	chartW := width - labelW
	if chartW < 1 {
		chartW = width
		labelW = 0
	}

	out := make([]string, rows)

	hi, lo := prevClose, prevClose
	hasData := len(series) > 0
	shi, slo := prevClose, prevClose
	if hasData {
		hi, lo = series[0], series[0]
		shi, slo = series[0], series[0]
		for _, v := range series {
			if v > hi {
				hi = v
			}
			if v < lo {
				lo = v
			}
			if v > shi {
				shi = v
			}
			if v < slo {
				slo = v
			}
		}
		if prevClose > hi {
			hi = prevClose
		}
		if prevClose < lo {
			lo = prevClose
		}
	}
	if hi == lo {
		hi += 1
		lo -= 1
	}

	// per-column quantized level (0=bottom), in [0, 8*rows-1].
	levels := make([]int, chartW)
	yLevels := 8 * rows
	if hasData {
		pts := downsample(series, chartW)
		for x, v := range pts {
			lv := int(math.Round((v - lo) / (hi - lo) * float64(yLevels-1)))
			if lv < 0 {
				lv = 0
			}
			if lv >= yLevels {
				lv = yLevels - 1
			}
			levels[x] = lv
		}
	}

	baseRow := chartCellRowFor(prevClose, lo, hi, rows)
	upGuide, downGuide := -1, -1
	if rows >= 4 {
		up := prevClose * 1.01
		dn := prevClose * 0.99
		if up <= hi && up >= lo {
			upGuide = chartCellRowFor(up, lo, hi, rows)
		}
		if dn <= hi && dn >= lo {
			downGuide = chartCellRowFor(dn, lo, hi, rows)
		}
	}

	rst := cc.reset
	guideClr := brightBlk
	baseClr := redDashed
	mono := cc.mono
	if cc.closed {
		mono = brightBlk
		baseClr = brightBlk
	}
	if !cc.use {
		rst, guideClr, baseClr, mono = "", "", "", ""
	}

	var hiLabel, loLabel string
	if labelW > 0 && hasData {
		hiLabel = truncWidth(fmtNum(shi, decimals), labelW)
		loLabel = truncWidth(fmtNum(slo, decimals), labelW)
	}

	for r := 0; r < rows; r++ {
		var b strings.Builder
		for c := 0; c < chartW; c++ {
			sym := blockSymbolFor(levels[c], r, rows)
			if sym != ' ' {
				if cc.hasGradient() {
					b.WriteString(cc.gradFG(r, rows))
					b.WriteRune(sym)
					b.WriteString(rst)
				} else {
					b.WriteString(mono)
					b.WriteRune(sym)
					b.WriteString(rst)
				}
				continue
			}
			// no body here: draw baseline / guideline markers (every other cell)
			switch {
			case r == baseRow && c%2 == 0:
				b.WriteString(baseClr)
				b.WriteRune('─')
				b.WriteString(rst)
			case (r == upGuide || r == downGuide) && c%2 == 0:
				b.WriteString(guideClr)
				b.WriteRune('·')
				b.WriteString(rst)
			default:
				b.WriteRune(' ')
			}
		}
		if labelW > 0 {
			lab := ""
			if r == 0 {
				lab = hiLabel
			} else if r == rows-1 {
				lab = loLabel
			}
			padded := padLeft(lab, labelW)
			if cc.use && lab != "" {
				b.WriteString(guideClr)
				b.WriteString(padded)
				b.WriteString(rst)
			} else {
				b.WriteString(padded)
			}
		}
		out[r] = b.String()
	}
	return out
}

// buildChartLinesTTY renders an ASCII-only area chart ('#'/'|'/'.').
// Colors use a single theme/mono escape (no gradient). Labels are kept; the
// baseline is a '-' dashed line and guidelines use '.'.
func buildChartLinesTTY(series []float64, prevClose float64, width, rows, labelW int, decimals int, cc chartColors) []string {
	if rows < 1 {
		rows = 1
	}
	if width < 1 {
		width = 1
	}
	chartW := width - labelW
	if chartW < 1 {
		chartW = width
		labelW = 0
	}

	out := make([]string, rows)

	hi, lo := prevClose, prevClose
	hasData := len(series) > 0
	shi, slo := prevClose, prevClose
	if hasData {
		hi, lo = series[0], series[0]
		shi, slo = series[0], series[0]
		for _, v := range series {
			if v > hi {
				hi = v
			}
			if v < lo {
				lo = v
			}
			if v > shi {
				shi = v
			}
			if v < slo {
				slo = v
			}
		}
		if prevClose > hi {
			hi = prevClose
		}
		if prevClose < lo {
			lo = prevClose
		}
	}
	if hi == lo {
		hi += 1
		lo -= 1
	}

	// per-column top cell row (0=top) reached by the column.
	topRow := make([]int, chartW)
	for i := range topRow {
		topRow[i] = rows // sentinel: nothing
	}
	if hasData {
		pts := downsample(series, chartW)
		for x, v := range pts {
			topRow[x] = chartCellRowFor(v, lo, hi, rows)
		}
	}

	baseRow := chartCellRowFor(prevClose, lo, hi, rows)

	rst := cc.reset
	mono := cc.mono
	baseClr := redDashed
	if cc.closed {
		mono = brightBlk
		baseClr = brightBlk
	}
	if !cc.use {
		rst, mono, baseClr = "", "", ""
	}

	var hiLabel, loLabel string
	if labelW > 0 && hasData {
		hiLabel = truncWidth(fmtNum(shi, decimals), labelW)
		loLabel = truncWidth(fmtNum(slo, decimals), labelW)
	}

	for r := 0; r < rows; r++ {
		var b strings.Builder
		for c := 0; c < chartW; c++ {
			switch {
			case hasData && r > topRow[c]:
				// below the top of the column: filled body
				b.WriteString(mono)
				b.WriteByte('|')
				b.WriteString(rst)
			case hasData && r == topRow[c]:
				// the column's top cell
				b.WriteString(mono)
				b.WriteByte('#')
				b.WriteString(rst)
			case r == baseRow && c%2 == 0:
				b.WriteString(baseClr)
				b.WriteByte('-')
				b.WriteString(rst)
			default:
				b.WriteByte('.')
			}
		}
		if labelW > 0 {
			lab := ""
			if r == 0 {
				lab = hiLabel
			} else if r == rows-1 {
				lab = loLabel
			}
			b.WriteString(padLeft(lab, labelW))
		}
		out[r] = b.String()
	}
	return out
}

// unavailableHistory labels unsupported history without drawing a synthetic chart.
func unavailableHistory(width, rows int) []string {
	out := make([]string, rows)
	for i := range out {
		out[i] = strings.Repeat(" ", width)
	}
	if rows > 0 {
		out[0] = padRight(truncWidth("History N/A", width), width)
	}
	return out
}
