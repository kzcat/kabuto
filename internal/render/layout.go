package render

import (
	"fmt"
	"os"
)

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
