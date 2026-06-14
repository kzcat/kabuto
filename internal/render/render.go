package render

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/i18n"
	"github.com/kzcat/kabuto/internal/symbols"
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

// Base RGB for gain/loss colors (truecolor gradient).
var (
	greenRGB = [3]int{0, 200, 0}
	redRGB   = [3]int{220, 40, 40}
)

// Options holds rendering options.
type Options struct {
	NoColor     bool           // disable color
	RedGreen    bool           // invert: up=red, down=green (Japan-style)
	TermWidth   int            // terminal width (0 = auto-detect)
	TermRows    int            // terminal rows (0 = auto-detect)
	Watch       bool           // fill height in watch mode
	FillHeight  bool           // fill height when stdout is a TTY (even for one-shot output)
	ForceCols   int            // manual column count (0 = auto)
	Loc         *time.Location // display timezone (nil = time.Local)
	CryptoItems []symbols.Item // reordered crypto items (nil = definition order)
	Lang        string         // UI language (empty = en)
	RangeLabel  string         // time-range label for display (e.g. "1d", "1mo")
	Theme       Theme          // color theme
}

// locOf returns opt.Loc, defaulting to time.Local if nil.
func locOf(loc *time.Location) *time.Location {
	if loc == nil {
		return time.Local
	}
	return loc
}

// ResolveNoColor is a pure helper for unit testing NO_COLOR logic.
// Priority: flag --no-color > NO_COLOR env (any value) > default (false).
func ResolveNoColor(flagNoColor bool, noColorEnv string, envPresent bool) bool {
	if flagNoColor {
		return true
	}
	if envPresent {
		return true
	}
	return false
}

// UseColor determines whether color output is enabled.
func UseColor(noColor bool) bool {
	if noColor {
		return false
	}
	// Check NO_COLOR env (https://no-color.org)
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
		return false
	}
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// stringWidth returns the display width (fullwidth=2, halfwidth=1).
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

// isWide checks whether a rune is East Asian Wide/Fullwidth.
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

// truncWidth truncates s to fit within the given display width.
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

// padRight pads s with spaces on the right to the given display width.
func padRight(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return s + strings.Repeat(" ", width-sw)
}

// padLeft pads s with spaces on the left to the given display width.
func padLeft(s string, width int) string {
	sw := stringWidth(s)
	if sw >= width {
		return s
	}
	return strings.Repeat(" ", width-sw) + s
}

func fmtNum(value float64, decimals int) string {
	return fmtNumLang(value, decimals, "en")
}

func fmtNumLang(value float64, decimals int, lang string) string {
	neg := value < 0
	if neg {
		value = -value
	}
	s := fmt.Sprintf("%.*f", decimals, value)
	parts := strings.Split(s, ".")
	intPart := parts[0]
	n := len(intPart)
	grp := i18n.GroupSep(lang)
	dec := i18n.DecimalSep(lang)
	if n > 3 {
		var buf strings.Builder
		rem := n % 3
		if rem > 0 {
			buf.WriteString(intPart[:rem])
			if n > rem {
				buf.WriteString(grp)
			}
		}
		for i := rem; i < n; i += 3 {
			buf.WriteString(intPart[i : i+3])
			if i+3 < n {
				buf.WriteString(grp)
			}
		}
		intPart = buf.String()
	}
	result := intPart
	if len(parts) > 1 {
		result += dec + parts[1]
	}
	if neg {
		result = "-" + result
	}
	return result
}

func fmtChange(value float64, decimals int) string {
	return fmtChangeLang(value, decimals, "en")
}

func fmtChangeLang(value float64, decimals int, lang string) string {
	if value > 0 {
		return "+" + fmtNumLang(value, decimals, lang)
	}
	return fmtNumLang(value, decimals, lang)
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

// colorFor returns the ANSI color escape for the given change value (supports rg inversion).
func colorFor(change float64, useColor, redGreen bool) string {
	return colorForTheme(change, useColor, redGreen, defaultTheme)
}

// colorForTheme returns the color escape using the given Theme.
func colorForTheme(change float64, useColor, redGreen bool, th Theme) string {
	if !useColor {
		return ""
	}
	up, down := th.UpColor, th.DownColor
	if redGreen {
		up, down = down, up
	}
	if change > 0 {
		return up
	} else if change < 0 {
		return down
	}
	return ""
}

// arrow returns an indicator symbol based on the change direction.
func arrow(change float64) string {
	if change > 0 {
		return "▲"
	} else if change < 0 {
		return "▼"
	}
	return "・"
}

// truecolorSupported checks if COLORTERM env is truecolor or 24bit.
func truecolorSupported() bool {
	ct := strings.ToLower(os.Getenv("COLORTERM"))
	return ct == "truecolor" || ct == "24bit"
}

// baseRGBFor returns the base RGB for the given change value (supports rg inversion).
func baseRGBFor(change float64, redGreen bool) [3]int {
	return baseRGBForTheme(change, redGreen, defaultTheme)
}

// baseRGBForTheme returns the RGB using the given Theme.
func baseRGBForTheme(change float64, redGreen bool, th Theme) [3]int {
	up, down := th.UpRGB, th.DownRGB
	if redGreen {
		up, down = down, up
	}
	if change < 0 {
		return down
	}
	return up // up/flat uses the up color as base
}

// fg24 returns a truecolor foreground escape sequence.
func fg24(c [3]int) string {
	return fmt.Sprintf("\033[38;2;%d;%d;%dm", c[0], c[1], c[2])
}

// gradientRGB linearly interpolates from base (top row) to ~50% darker (bottom row) for the given row.
func gradientRGB(base [3]int, row, rows int) [3]int {
	if rows <= 1 {
		return base
	}
	// t=0 (top row) -> 1 (bottom row). Bottom row is ~50% of base.
	t := float64(row) / float64(rows-1)
	factor := 1.0 - 0.5*t
	var out [3]int
	for i := 0; i < 3; i++ {
		out[i] = int(math.Round(float64(base[i]) * factor))
	}
	return out
}

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
			case isBody && cc.use && cc.truecolor && !cc.closed:
				b.WriteString(fg24(gradientRGB(cc.base, r, rows)))
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

const minTileW = 24 // minimum tile outer width (fullwidth=2 accounted for)

// distributeWidths distributes the terminal width across cols columns.
// Base width = termWidth/cols; remainder columns (left-to-right) get +1 so the total equals termWidth.
// Returns the outer width for each column.
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

// gridColumns calculates the tile column count from terminal width (minimum tile width: minTileW).
// Columns do not exceed itemCount (no constraint when itemCount<=0).
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

// chartRows calculates the chart row count N (base value shared across all stages) from terminal rows.
// tileRows = (termRows - header) / rowsOfTiles
// N = tileRows - tileChrome. Clamped to [1, 12]. termRows<=0 returns N=2 (no height calculation).
func chartRows(termRows, headerLines, totalTileRows int) int {
	if termRows <= 0 || totalTileRows <= 0 {
		return 2
	}
	avail := termRows - headerLines
	if avail < 1 {
		avail = 1
	}
	tileH := avail / totalTileRows
	n := tileH - tileChrome
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
	// tileChrome is the number of non-chart lines per tile (top border + badge row + value row + bottom border).
	// Tile outer height = chart rows N + tileChrome. Currently 4.
	tileChrome = 4
)

// chartRowsPerStage distributes terminal rows across totalTileRows stages, returning chart rows N per stage (top to bottom).
// Base N = tileH - tileChrome (tileH = avail / totalTileRows). Remainder rows from integer division are
// distributed one per stage from the top so the last row reaches the screen bottom (no leftover rows).
// Clamped to [1, 12]. Undistributable remainder (due to N cap) is left over.
func chartRowsPerStage(termRows, headerLines, totalTileRows int) []int {
	if totalTileRows <= 0 {
		return nil
	}
	out := make([]int, totalTileRows)
	if termRows <= 0 {
		// non-TTY (no height calculation): fixed N=2
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
	rem := avail % totalTileRows // leftover rows from integer division (unfilled bottom rows)
	baseN := tileH - tileChrome
	if baseN < chartNMin {
		baseN = chartNMin
	}
	if baseN > chartNMax {
		baseN = chartNMax
	}
	for i := range out {
		n := baseN
		// add one remainder row per stage from top (within N cap 12)
		if i < rem && n < chartNMax {
			n++
		}
		out[i] = n
	}
	return out
}

// usedRowsForCols returns the number of rows actually used for the given column count.
// stages = ceil(itemCount/cols), avail = termRows - headerLines.
// Per-stage N follows chartRowsPerStage rules (baseN = avail/stages - tileChrome, clamped;
// remainder rows added one per stage from top, within chartNMax).
// Used rows = sum(tileChrome + N). Returns 0 when termRows<=0 or itemCount<=0 (not height-optimizable).
func usedRowsForCols(termRows, headerLines, itemCount, cols int) int {
	if termRows <= 0 || itemCount <= 0 || cols < 1 {
		return 0
	}
	stages := (itemCount + cols - 1) / cols
	avail := termRows - headerLines
	if avail < 1 {
		avail = 1
	}
	tileH := avail / stages
	rem := avail % stages
	baseN := tileH - tileChrome
	if baseN < chartNMin {
		baseN = chartNMin
	}
	if baseN > chartNMax {
		baseN = chartNMax
	}
	used := 0
	for i := 0; i < stages; i++ {
		n := baseN
		if i < rem && n < chartNMax {
			n++
		}
		used += tileChrome + n
	}
	return used
}

// optimalColumns determines the column count C by exhaustive search over width and height (for TTY).
// Candidates: C in [1, min(itemCount, termWidth/minTileW)].
// Selects C that maximizes usedRowsForCols. Ties broken by larger C.
// C whose used rows exceed avail is excluded (would overflow the screen).
// Falls back to width-only gridColumns when no valid C exists or termRows<=0.
func optimalColumns(termWidth, termRows, headerLines, itemCount int) int {
	if termWidth <= 0 {
		termWidth = 80
	}
	if itemCount <= 0 {
		return 1
	}
	if termRows <= 0 {
		return gridColumns(termWidth, itemCount)
	}
	maxC := termWidth / minTileW
	if maxC < 1 {
		maxC = 1
	}
	if maxC > itemCount {
		maxC = itemCount
	}
	avail := termRows - headerLines
	if avail < 1 {
		avail = 1
	}
	bestC := 0
	bestUsed := -1
	for c := 1; c <= maxC; c++ {
		used := usedRowsForCols(termRows, headerLines, itemCount, c)
		if used > avail {
			// layout overflows the screen; skip.
			continue
		}
		// maximize used rows. Ties: larger C wins (>= for last-wins = larger C).
		if used >= bestUsed {
			bestUsed = used
			bestC = c
		}
	}
	if bestC == 0 {
		// all C overflow (extremely short terminal): fall back to width-only.
		return gridColumns(termWidth, itemCount)
	}
	return bestC
}

// box-drawing character sets
type boxChars struct {
	tl, tr, bl, br, h, v string
}

func getBoxChars(ascii bool) boxChars {
	if ascii {
		return boxChars{"+", "+", "+", "+", "-", "|"}
	}
	return boxChars{"┌", "┐", "└", "┘", "─", "│"}
}

// buildTopBorder constructs the top border. innerW is the inner display width (fullwidth=2).
// Layout: bc.tl + bc.h + " " + name + " " + <dash...> + [" " + secName + " " + bc.h] + bc.tr
// If secName is empty, no section name is embedded. Dash count fills innerW exactly.
func buildTopBorder(bc boxChars, border, rst, name, secName string, innerW int) string {
	var b strings.Builder
	b.WriteString(border)
	b.WriteString(bc.tl)
	// build innerW columns of content.
	var inner strings.Builder
	inner.WriteString(bc.h)
	inner.WriteString(" ")
	inner.WriteString(name)
	inner.WriteString(" ")
	usedLeft := 2 + stringWidth(name) // bc.h(1) + " "(1) + name + " "(1) = 2 + nameW(+1 below)
	usedLeft++                        // trailing " "
	if secName != "" {
		// right side: <dash...> + " " + secName + " " + bc.h
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

// buildTopBorderW is like buildTopBorder but accepts explicit nameW since name may contain color escapes.
// secName is rendered in bright black (border color).
func buildTopBorderW(bc boxChars, border, rst, name string, nameW int, secName string, innerW int) string {
	var b strings.Builder
	b.WriteString(border)
	b.WriteString(bc.tl)
	var inner strings.Builder
	inner.WriteString(bc.h)
	inner.WriteString(" ")
	inner.WriteString(name)
	inner.WriteString(" ")
	usedLeft := 2 + nameW
	usedLeft++ // trailing " "
	if secName != "" {
		secW := stringWidth(secName)
		rightFixed := 1 + secW + 1 + 1
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

// isClosed returns true if regularMarketTime (epoch seconds) is more than 30 minutes old.
// epoch<=0 (unavailable) is not treated as closed.
func isClosed(epoch int64, now time.Time) bool {
	if epoch <= 0 {
		return false
	}
	return now.Sub(time.Unix(epoch, 0)) > 30*time.Minute
}

// currencySymbol maps a currency code to its display symbol.
func currencySymbol(code string) string {
	switch code {
	case "JPY", "CNY":
		return "¥"
	case "USD", "AUD", "CAD", "HKD", "SGD", "NZD":
		return "$"
	case "EUR":
		return "€"
	case "GBP":
		return "£"
	case "KRW":
		return "₩"
	default:
		return ""
	}
}

// outerW = tile outer width (including borders), chartN = chart row count.
// When secName is non-empty and tile width >= 30, the section name is embedded at the right end of the top border in bright black.
// Country code [XX] (bright black) is prepended to the symbol name on the top border (omitted if Country is empty).
// Layout: top border + badge row (left-aligned) + chart N rows + value row (bold) + change + bottom border.
// Returns chartN + 4 lines.
func renderTile(item symbols.Item, r *fetcher.Result, outerW, chartN int, useColor, redGreen, ascii, truecolor bool, secName string) []string {
	return renderTileL(item, r, outerW, chartN, useColor, redGreen, ascii, truecolor, secName, "en", defaultTheme)
}

func renderTileL(item symbols.Item, r *fetcher.Result, outerW, chartN int, useColor, redGreen, ascii, truecolor bool, secName string, lang string, th Theme) []string {
	if outerW < minTileW {
		outerW = minTileW
	}
	if chartN < 1 {
		chartN = 1
	}
	innerW := outerW - 2   // inner width excluding 2 border columns
	contentW := innerW - 2 // content width excluding 1-col margin on each side

	bc := getBoxChars(ascii)
	border := th.BrightBlk
	wclr := th.BoldWhite
	rst := th.Reset
	cc := th.BrightBlk // country code color
	if !useColor {
		border, wclr, rst, cc = "", "", "", ""
	}

	// symbol name for top border (country code prefix). Country code in bright black [XX].
	cc2 := ""
	if item.Country != "" {
		cc2 = "[" + item.Country + "]"
	}
	// display width estimate: [XX] + name
	maxName := innerW - 4 - stringWidth(cc2)
	if maxName < 1 {
		maxName = 1
	}
	name := truncWidth(item.Name, maxName)
	// pass "country code (colored) + name" as a single display string to buildTopBorder.
	displayName := name
	if cc2 != "" {
		if useColor {
			displayName = cc + cc2 + rst + name
		} else {
			displayName = cc2 + name
		}
	}
	nameW := stringWidth(cc2) + stringWidth(name)
	usedLeft := 3 + nameW
	secLabel := ""
	if outerW >= 30 && secName != "" {
		secW := stringWidth(secName)
		need := secW + 4
		if usedLeft+1+need <= innerW {
			secLabel = secName
		}
	}
	top := buildTopBorderW(bc, border, rst, displayName, nameW, secLabel, innerW)

	bottom := border + bc.bl + strings.Repeat(bc.h, innerW) + bc.br + rst

	left := border + bc.v + rst
	right := border + bc.v + rst

	wrap := func(inner string) string {
		return left + " " + inner + " " + right
	}

	lines := make([]string, 0, chartN+tileChrome)
	lines = append(lines, top)

	if r == nil {
		na := padRight("N/A", contentW)
		lines = append(lines, wrap(na))
		blank := strings.Repeat(string(rune(brailleBase)), contentW)
		for i := 0; i < chartN; i++ {
			lines = append(lines, wrap(blank))
		}
		lines = append(lines, wrap(padRight("", contentW)))
		lines = append(lines, bottom)
		return lines
	}

	clr := colorForTheme(r.Change, useColor, redGreen, th)
	// Currency symbol prefix for the price
	sym := currencySymbol(r.Currency)
	priceS := sym + fmtNumLang(r.Price, item.Decimals, lang)
	changeS := fmtChangeLang(r.Change, item.Decimals, lang)
	pctText := arrow(r.Change) + fmtPct(r.ChangePct)

	// top row: change% badge (left-aligned)
	badge := buildBadgeTheme(pctText, r.Change, useColor, redGreen, th)
	badgePlainW := stringWidth(" " + pctText + " ")
	badgeLine := badge + strings.Repeat(" ", maxInt(0, contentW-badgePlainW))
	lines = append(lines, wrap(badgeLine))

	// chart rows: braille area chart (baseline, guidelines, labels, closed-market support)
	closed := isClosed(r.Epoch, time.Now())
	labelW := 0
	if outerW >= 30 {
		labelW = 9
		if labelW > contentW-2 {
			labelW = 0
		}
	}
	cclr := chartColors{
		use:       useColor,
		truecolor: truecolor,
		closed:    closed,
		base:      baseRGBForTheme(r.Change, redGreen, th),
		mono:      clr,
		reset:     rst,
	}
	rowsStr := buildChartLines(r.Series, r.PrevClose, contentW, chartN, labelW, item.Decimals, cclr)
	for _, rowStr := range rowsStr {
		lines = append(lines, wrap(rowStr))
	}

	// bottom row: price (bold white) + change (gain/loss color)
	plainW := stringWidth(priceS) + 2 + stringWidth(changeS)
	if plainW <= contentW {
		gap := contentW - plainW
		valLine := wclr + priceS + rst + strings.Repeat(" ", 2+gap) + clr + changeS + rst
		lines = append(lines, wrap(valLine))
	} else if stringWidth(priceS) <= contentW {
		valLine := wclr + priceS + rst
		lines = append(lines, wrap(padPlainRight(valLine, priceS, contentW)))
	} else {
		alt := truncWidth(priceS, contentW)
		lines = append(lines, wrap(padPlainRight(wclr+alt+rst, alt, contentW)))
	}

	lines = append(lines, bottom)
	return lines
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// buildBadge generates a change% badge like ' ▲+0.06% '.
// Gain/loss colored background + bold white text. Down=red bg, flat=bright black bg.
// redGreen inverts the background color. useColor=false returns plain text.
func buildBadge(text string, change float64, useColor, redGreen bool) string {
	return buildBadgeTheme(text, change, useColor, redGreen, defaultTheme)
}

func buildBadgeTheme(text string, change float64, useColor, redGreen bool, th Theme) string {
	content := " " + text + " "
	if !useColor {
		return content
	}
	// background color SGR code
	bgGreen, bgRed, bgGray := 42, 41, 100 // 100 = bright black bg
	up, down := bgGreen, bgRed
	if redGreen {
		up, down = bgRed, bgGreen
	}
	// For mono theme, always use gray badge
	if th.Name == "mono" {
		return fmt.Sprintf("\033[1;37;%dm%s\033[0m", bgGray, content)
	}
	// For highcontrast theme, use blue/orange backgrounds
	if th.Name == "highcontrast" {
		up = 44   // blue bg
		down = 43 // yellow/orange bg (closest standard)
		if redGreen {
			up, down = down, up
		}
	}
	var bg int
	if change > 0 {
		bg = up
	} else if change < 0 {
		bg = down
	} else {
		bg = bgGray
	}
	// bold + white text + background color
	return fmt.Sprintf("\033[1;37;%dm%s\033[0m", bg, content)
}

// padPlainRight pads a colored string (whose visible text is plain) to contentW with trailing spaces.
func padPlainRight(colored, plain string, contentW int) string {
	pad := contentW - stringWidth(plain)
	if pad < 0 {
		pad = 0
	}
	return colored + strings.Repeat(" ", pad)
}

// detectTermSize retrieves terminal columns and rows.
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

// DetectTermSize exposes terminal columns and rows (used from main).
func DetectTermSize() (cols, rows int) {
	return detectTermSize()
}

// ComputeCols returns the column count that RenderDashboard would use for the given options and item count.
func ComputeCols(opt Options, itemCount int) int {
	if opt.ForceCols > 0 {
		c := opt.ForceCols
		if c < 1 {
			c = 1
		}
		if itemCount > 0 && c > itemCount {
			c = itemCount
		}
		return c
	}
	termWidth := opt.TermWidth
	if termWidth <= 0 {
		termWidth = 80
	}
	termRows := opt.TermRows
	tty := opt.Watch || opt.FillHeight
	if tty && termRows > 0 {
		return optimalColumns(termWidth, termRows, 1, itemCount)
	}
	return gridColumns(termWidth, itemCount)
}

// detectTermWidth retrieves the terminal width from $COLUMNS or ioctl. Defaults to 80.
func detectTermWidth() int {
	c, _ := detectTermSize()
	return c
}

// flatItem represents a single symbol in the flat layout (with its section name).
type flatItem struct {
	item    symbols.Item
	secName string
}

// RenderDashboard generates a dashboard similar to the original site.
// All display symbols are laid out sequentially in an N×M grid in definition order (section headings removed).
func RenderDashboard(data map[string]*fetcher.Result, sections []string, opt Options) string {
	useColor := !opt.NoColor
	ascii := opt.NoColor // non-color mode falls back to ASCII borders
	truecolor := useColor && truecolorSupported()
	loc := locOf(opt.Loc)
	termWidth := opt.TermWidth
	termRows := opt.TermRows
	if termWidth <= 0 {
		termWidth = detectTermWidth()
	}

	keys := sections
	if len(keys) == 0 {
		keys = symbols.SectionOrder
	}

	// flatten all display symbols in definition order.
	// crypto section uses reordered items (opt.CryptoItems) if provided.
	var flat []flatItem
	for _, secKey := range keys {
		sec := symbols.Sections[secKey]
		items := sec.Items
		if secKey == "crypto" && opt.CryptoItems != nil {
			items = opt.CryptoItems
		}
		secTitle := i18n.SectionTitle(opt.Lang, secKey)
		for _, it := range items {
			fi := it
			fi.Name = i18n.SymbolName(opt.Lang, it.Symbol, it.Name)
			flat = append(flat, flatItem{item: fi, secName: secTitle})
		}
	}

	headerLines := 1 // header only (section heading lines removed)

	// determine column count:
	//   ForceCols > 0: manual (clamped to 1..len(flat))
	//   TTY (Watch || FillHeight) with known height: exhaustive width+height optimization
	//   non-TTY: width-only (gridColumns)
	tty := opt.Watch || opt.FillHeight
	var cols int
	if opt.ForceCols > 0 {
		cols = opt.ForceCols
		if cols < 1 {
			cols = 1
		}
		if len(flat) > 0 && cols > len(flat) {
			cols = len(flat)
		}
	} else if tty && termRows > 0 {
		cols = optimalColumns(termWidth, termRows, headerLines, len(flat))
	} else {
		cols = gridColumns(termWidth, len(flat))
	}
	colWidths := distributeWidths(termWidth, cols)

	// total tile rows = ceil(items / cols)
	totalTileRows := 0
	if len(flat) > 0 {
		totalTileRows = (len(flat) + cols - 1) / cols
	}

	// determine chart rows N (per stage)
	var stageN []int
	if opt.Watch || opt.FillHeight {
		// TTY: fill height. Distribute N per stage (remainder rows added from top).
		stageN = chartRowsPerStage(termRows, headerLines, totalTileRows)
	} else {
		// non-TTY (pipe/redirect): no height calculation, fixed N=2
		stageN = make([]int, totalTileRows)
		for i := range stageN {
			stageN[i] = 2
		}
	}

	var lines []string
	now := time.Now().In(loc).Format("2006-01-02 15:04:05 -07:00")
	header := "kabuto    Updated: " + now
	if opt.RangeLabel != "" {
		header += "  [" + opt.RangeLabel + "]"
	}

	th := opt.Theme
	if th.Reset == "" {
		th = defaultTheme
	}

	if useColor {
		// extend reverse highlight to terminal width
		h := truncWidth(header, termWidth)
		hw := stringWidth(h)
		pad := termWidth - hw
		if pad < 0 {
			pad = 0
		}
		lines = append(lines, th.Reverse+h+strings.Repeat(" ", pad)+th.Reset)
	} else {
		lines = append(lines, truncWidth(header, termWidth))
	}
	// no blank line after header

	// lay out in cols-column grid sequentially (row-major). Last row may be partial.
	stageIdx := 0
	for i := 0; i < len(flat); i += cols {
		end := i + cols
		if end > len(flat) {
			end = len(flat)
		}
		rowItems := flat[i:end]
		// chart rows N for this stage (may vary per stage)
		chartN := 2
		if stageIdx < len(stageN) {
			chartN = stageN[stageIdx]
		}
		tileH := chartN + tileChrome // top border + badge + chart N + value row + bottom border
		// generate tiles for this row (each column may have different width)
		var tiles [][]string
		for ci, fi := range rowItems {
			w := colWidths[ci]
			tiles = append(tiles, renderTileL(fi.item, data[fi.item.Symbol], w, chartN, useColor, opt.RedGreen, ascii, truecolor, fi.secName, opt.Lang, th))
		}
		// empty cells in the last row are left blank (no tile placed).
		// concatenate tiles horizontally (zero gap)
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

// JSONItem represents a single symbol in JSON output.
type JSONItem struct {
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	Country   string    `json:"country"`
	Price     *float64  `json:"price"`
	Change    *float64  `json:"change"`
	ChangePct *float64  `json:"change_pct"`
	Time      *string   `json:"time"`
	Epoch     *int64    `json:"epoch"`
	Series    []float64 `json:"series"`
}

// JSONSection represents a single section in JSON output.
type JSONSection struct {
	Title string     `json:"title"`
	Items []JSONItem `json:"items"`
}

// RenderJSON generates the JSON output string. Times are formatted according to loc (nil = time.Local).
func RenderJSON(data map[string]*fetcher.Result, sections []string, loc *time.Location, lang string) string {
	l := locOf(loc)
	keys := sections
	if len(keys) == 0 {
		keys = symbols.SectionOrder
	}
	output := make(map[string]JSONSection)
	for _, secKey := range keys {
		sec := symbols.Sections[secKey]
		var items []JSONItem
		for _, item := range sec.Items {
			localName := i18n.SymbolName(lang, item.Symbol, item.Name)
			r := data[item.Symbol]
			if r == nil {
				items = append(items, JSONItem{Name: localName, Symbol: item.Symbol, Country: item.Country})
			} else {
				price := roundTo(r.Price, item.Decimals)
				change := roundTo(r.Change, item.Decimals)
				pct := roundTo(r.ChangePct, 2)
				// regenerate "15:04" from Epoch in loc timezone (reflects --tz).
				t := r.Time
				if r.Epoch > 0 {
					t = time.Unix(r.Epoch, 0).In(l).Format("15:04")
				}
				epoch := r.Epoch
				series := make([]float64, len(r.Series))
				for i, v := range r.Series {
					series[i] = roundTo(v, item.Decimals)
				}
				items = append(items, JSONItem{
					Name:      localName,
					Symbol:    item.Symbol,
					Country:   item.Country,
					Price:     &price,
					Change:    &change,
					ChangePct: &pct,
					Time:      &t,
					Epoch:     &epoch,
					Series:    series,
				})
			}
		}
		output[secKey] = JSONSection{Title: i18n.SectionTitle(lang, secKey), Items: items}
	}
	b, _ := json.MarshalIndent(output, "", "  ")
	return string(b)
}

func roundTo(v float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(v*pow) / pow
}
