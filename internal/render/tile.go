package render

import (
	"fmt"
	"strings"
	"time"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/symbols"
)

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

// buildBottomBorderHL constructs the bottom border with a right-aligned
// "H:<high> L:<low>" (and optional range-width %) label embedded in bright black.
// innerW is the inner display width. When the label does not fit, a plain border
// is returned. border/lblClr/rst are color escapes (empty when color is disabled).
func buildBottomBorderHL(bc boxChars, border, lblClr, rst, label string, innerW int) string {
	lblW := stringWidth(label)
	// need: dash + " " + label + " " + (corner handled separately). Reserve a left dash run.
	// layout inner: <dash...> " " label " "
	if lblW == 0 || lblW+3 > innerW {
		return border + bc.bl + strings.Repeat(bc.h, innerW) + bc.br + rst
	}
	dashN := innerW - (1 + lblW + 1)
	if dashN < 1 {
		return border + bc.bl + strings.Repeat(bc.h, innerW) + bc.br + rst
	}
	var b strings.Builder
	b.WriteString(border)
	b.WriteString(bc.bl)
	b.WriteString(strings.Repeat(bc.h, dashN))
	b.WriteString(" ")
	if lblClr != "" {
		b.WriteString(lblClr)
		b.WriteString(label)
		b.WriteString(border) // back to border color for trailing pieces
	} else {
		b.WriteString(label)
	}
	b.WriteString(" ")
	b.WriteString(bc.br)
	b.WriteString(rst)
	return b.String()
}

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
	// Legacy entry point: braille chart, legacy color depth (truecolor flag decides).
	return renderTileLG(item, r, outerW, chartN, useColor, redGreen, ascii, truecolor, secName, lang, th, "braille", 0)
}

// renderTileLG is renderTileL with explicit graph-symbol mode and color depth.
// symMode: ""|"braille"|"block"|"tty". depth: 0 (legacy)|16|8(256)|24(truecolor).
func renderTileLG(item symbols.Item, r *fetcher.Result, outerW, chartN int, useColor, redGreen, ascii, truecolor bool, secName string, lang string, th Theme, symMode string, depth int) []string {
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
	if r.ChangeUnavailable {
		changeS, pctText = "N/A", "N/A"
	}

	// top row: change% badge (left-aligned)
	badge := buildBadgeTheme(pctText, r.Change, useColor, redGreen, th)
	badgePlainW := stringWidth(" " + pctText + " ")
	trail := maxInt(0, contentW-badgePlainW)
	// B9: small gradient meter bar in the trailing space (block/tty modes only,
	// to keep the default braille appearance unchanged). Omitted when too narrow.
	badgeLine := badge + strings.Repeat(" ", trail)
	if !r.ChangeUnavailable && useColor && (symMode == "block" || symMode == "tty") && trail >= 7 {
		mw := trail - 1 // leave one space before the meter
		if mw > 10 {
			mw = 10
		}
		meter := meterBar(r.ChangePct, mw, th, true, redGreen, depth)
		badgeLine = badge + strings.Repeat(" ", trail-mw) + meter
	}
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
		symbol:    symMode,
		depth:     depth,
	}
	rowsStr := buildChartLines(r.Series, r.PrevClose, contentW, chartN, labelW, item.Decimals, cclr)
	if r.HistoryUnavailable {
		rowsStr = unavailableHistory(contentW, chartN)
	}
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

	// B2: bottom border with day high/low (+ range width %) label, right-aligned.
	dayBottom := bottom
	if outerW >= 30 && len(r.Series) > 0 {
		shi, slo := r.Series[0], r.Series[0]
		for _, v := range r.Series {
			if v > shi {
				shi = v
			}
			if v < slo {
				slo = v
			}
		}
		label := "H:" + fmtNum(shi, item.Decimals) + " L:" + fmtNum(slo, item.Decimals)
		// add range width % when it still fits.
		if slo != 0 {
			rng := (shi - slo) / slo * 100
			withRng := label + " " + fmtPctPlain(rng)
			if stringWidth(withRng)+3 <= innerW {
				label = withRng
			}
		}
		lblClr := th.BrightBlk
		if !useColor {
			lblClr = ""
		}
		dayBottom = buildBottomBorderHL(bc, border, lblClr, rst, label, innerW)
	}

	if r.Stale {
		dayBottom = buildBottomBorderHL(bc, border, border, rst, "STALE", innerW)
	}
	lines = append(lines, dayBottom)
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

// highlightTile replaces the border escapes in a rendered tile with highlighted (bold+bright) borders.
func highlightTile(tileLines []string, outerW int, hlColor, rst string, ascii bool) []string {
	bc := getBoxChars(ascii)
	innerW := outerW - 2
	if innerW < 0 {
		innerW = 0
	}
	out := make([]string, len(tileLines))
	for i, line := range tileLines {
		if i == 0 {
			out[i] = hlColor + bc.tl + strings.Repeat(bc.h, innerW) + bc.tr + rst
		} else if i == len(tileLines)-1 {
			out[i] = hlColor + bc.bl + strings.Repeat(bc.h, innerW) + bc.br + rst
			if strings.Contains(line, "STALE") {
				out[i] = buildBottomBorderHL(bc, hlColor, hlColor, rst, "STALE", innerW)
			}
		} else {
			out[i] = hlColor + bc.v + rst + extractInner(line) + hlColor + bc.v + rst
		}
	}
	return out
}

// extractInner extracts the raw inner content of a tile line (between the first and last │ or | border chars),
// including any escape sequences that are part of the inner content.
func extractInner(line string) string {
	// Find byte positions of first and last border character
	firstBox := strings.Index(line, "│")
	lastBox := strings.LastIndex(line, "│")
	if firstBox >= 0 && lastBox > firstBox {
		return line[firstBox+3 : lastBox] // │ is 3 bytes
	}
	// ASCII fallback
	firstPipe := strings.IndexByte(line, '|')
	lastPipe := strings.LastIndexByte(line, '|')
	if firstPipe >= 0 && lastPipe > firstPipe {
		return line[firstPipe+1 : lastPipe]
	}
	return line
}
