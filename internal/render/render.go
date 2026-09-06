package render

import (
	"os"
	"strings"
	"time"

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
	GraphSymbol string         // chart symbol mode: "auto"|"braille"|"block"|"tty" (empty = auto)
	SelIndex    int            // selected grid item index (-1 = none)
	Loading     bool           // a requested data fetch has not completed yet
	DetailView  bool           // show detail view for selected item

	// History holds an accumulated rolling price series per symbol (B7). When
	// non-nil and an entry exists for a symbol, it replaces that result's
	// intraday Series for chart rendering. nil = use API Series as-is.
	History map[string][]float64
}

// applyHistory returns a data map whose results have their Series replaced by
// the corresponding entry in hist (when present and non-empty). The originals
// are never mutated: each substituted Result is shallow-copied. When hist is
// nil/empty the input map is returned unchanged.
func applyHistory(data map[string]*fetcher.Result, hist map[string][]float64) map[string]*fetcher.Result {
	if len(hist) == 0 || len(data) == 0 {
		return data
	}
	out := make(map[string]*fetcher.Result, len(data))
	for sym, r := range data {
		if r != nil && !r.HistoryUnavailable {
			if h, ok := hist[sym]; ok && len(h) > 0 {
				cp := *r
				cp.Series = h
				out[sym] = &cp
				continue
			}
		}
		out[sym] = r
	}
	return out
}

// localeIsUTF8 reports whether the current locale env indicates a UTF-8 charset.
// Used by graph-symbol "auto" resolution: non-UTF-8 locales fall back to "tty".
func localeIsUTF8() bool {
	for _, k := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		if v := os.Getenv(k); v != "" {
			return strings.Contains(strings.ToUpper(v), "UTF-8") || strings.Contains(strings.ToUpper(v), "UTF8")
		}
	}
	// No locale env set: assume modern UTF-8 terminal.
	return true
}

// resolveGraphSymbol resolves the "auto" graph-symbol mode to a concrete mode.
// auto -> "tty" when color is disabled (ascii fallback) or the locale is non-UTF-8;
// otherwise "braille". Explicit modes are returned unchanged (unknown -> "braille").
func resolveGraphSymbol(mode string, noColor, utf8 bool) string {
	switch mode {
	case "braille", "block", "tty":
		return mode
	default: // "auto" or empty/unknown
		if noColor || !utf8 {
			return "tty"
		}
		return "braille"
	}
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
	depth := colorDepth()
	symMode := resolveGraphSymbol(opt.GraphSymbol, opt.NoColor, localeIsUTF8())
	loc := locOf(opt.Loc)
	termWidth := opt.TermWidth
	termRows := opt.TermRows
	if termWidth <= 0 {
		termWidth = detectTermWidth()
	}

	// B7: if a rolling history is provided, substitute each symbol's intraday
	// Series with the accumulated history. Work on shallow copies so the cached
	// Result objects (shared with the fetcher cache) are never mutated.
	data = applyHistory(data, opt.History)

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
		secTitle := sectionTitle(opt.Lang, sec)
		for _, it := range items {
			fi := it
			fi.Name = i18n.SymbolName(opt.Lang, it.Symbol, it.Name)
			flat = append(flat, flatItem{item: fi, secName: secTitle})
		}
	}

	// B5: detail view
	if opt.DetailView && opt.SelIndex >= 0 && opt.SelIndex < len(flat) {
		return renderDetail(flat[opt.SelIndex], data, opt, termWidth, termRows, useColor, ascii, truecolor, depth, symMode, loc)
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
	if opt.Loading {
		header = "kabuto    Loading..."
	}
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

	// B10: highlight color for selected tile border
	selBorder := th.Bold + th.BoldWhite
	if !useColor {
		selBorder = ""
	}

	// lay out in cols-column grid sequentially (row-major). Last row may be partial.
	stageIdx := 0
	flatIdx := 0
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
			tileLines := renderTileLG(fi.item, data[fi.item.Symbol], w, chartN, useColor, opt.RedGreen, ascii, truecolor, fi.secName, opt.Lang, th, symMode, depth)
			// B10: highlight selected tile
			if useColor && flatIdx+ci == opt.SelIndex {
				tileLines = highlightTile(tileLines, w, selBorder, th.Reset, ascii)
			}
			tiles = append(tiles, tileLines)
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
		flatIdx += len(rowItems)
		stageIdx++
	}
	return strings.Join(lines, "\n")
}

func sectionTitle(lang string, section symbols.Section) string {
	title := i18n.SectionTitle(lang, section.Key)
	if title == section.Key && section.Title != "" {
		return section.Title
	}
	return title
}
