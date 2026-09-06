package render

import (
	"strings"
	"time"

	"github.com/kzcat/kabuto/internal/fetcher"
)

// renderDetail renders a full-width detail view for the selected item (B5).
func renderDetail(fi flatItem, data map[string]*fetcher.Result, opt Options, termWidth, termRows int, useColor, ascii, truecolor bool, depth int, symMode string, loc *time.Location) string {
	th := opt.Theme
	if th.Reset == "" {
		th = defaultTheme
	}
	rst := th.Reset
	wclr := th.BoldWhite
	if !useColor {
		rst, wclr = "", ""
	}

	r := data[fi.item.Symbol]

	var lines []string
	// header
	now := time.Now().In(loc).Format("2006-01-02 15:04:05 -07:00")
	header := "kabuto    Updated: " + now
	if opt.Loading {
		header = "kabuto    Loading..."
	}
	if opt.RangeLabel != "" {
		header += "  [" + opt.RangeLabel + "]"
	}
	if useColor {
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

	// Detail info line
	sym := currencySymbol("")
	priceS, changeS, pctS, prevS, hiS, loS, rngS := "N/A", "", "", "", "", "", ""
	var series []float64
	var prevClose float64
	change := 0.0
	if r != nil {
		sym = currencySymbol(r.Currency)
		priceS = sym + fmtNumLang(r.Price, fi.item.Decimals, opt.Lang)
		changeS = fmtChangeLang(r.Change, fi.item.Decimals, opt.Lang)
		pctS = fmtPct(r.ChangePct)
		prevS = sym + fmtNumLang(r.PrevClose, fi.item.Decimals, opt.Lang)
		if r.ChangeUnavailable {
			changeS, pctS, prevS = "N/A", "N/A", "N/A"
		}
		series = r.Series
		prevClose = r.PrevClose
		change = r.Change
		if len(series) > 0 {
			shi, slo := series[0], series[0]
			for _, v := range series {
				if v > shi {
					shi = v
				}
				if v < slo {
					slo = v
				}
			}
			hiS = sym + fmtNumLang(shi, fi.item.Decimals, opt.Lang)
			loS = sym + fmtNumLang(slo, fi.item.Decimals, opt.Lang)
			if slo != 0 {
				rngS = fmtPctPlain((shi - slo) / slo * 100)
			}
		}
	}

	clr := colorForTheme(change, useColor, opt.RedGreen, th)
	// Name + price line
	nameLine := wclr + fi.item.Name + rst + "  " + wclr + priceS + rst
	if changeS != "" {
		nameLine += "  " + clr + changeS + " (" + pctS + ")" + rst
	}
	lines = append(lines, nameLine)
	if r != nil && (r.Source != "" || r.Stale || !r.FetchedAt.IsZero()) {
		status := "Source: " + r.Source
		if r.Stale {
			status = "STALE  " + status
		}
		if !r.FetchedAt.IsZero() {
			status += "  Fetched: " + r.FetchedAt.In(loc).Format("2006-01-02 15:04:05 -07:00")
		}
		lines = append(lines, truncWidth(status, termWidth))
	}

	// Stats line
	statsLine := "Prev: " + prevS + "  High: " + hiS + "  Low: " + loS
	if rngS != "" {
		statsLine += "  Range: " + rngS
	}
	if opt.RangeLabel != "" {
		statsLine += "  [" + opt.RangeLabel + "]"
	}
	lines = append(lines, statsLine)
	lines = append(lines, "") // blank separator

	// Chart: use remaining terminal height
	chartH := termRows - len(lines) - 1
	if chartH < 1 {
		chartH = 4
	}
	chartW := termWidth
	if chartW < 1 {
		chartW = 80
	}

	cclr := chartColors{
		use:       useColor,
		truecolor: truecolor,
		closed:    r != nil && isClosed(r.Epoch, time.Now()),
		base:      baseRGBForTheme(change, opt.RedGreen, th),
		mono:      clr,
		reset:     rst,
		symbol:    symMode,
		depth:     depth,
	}
	chartLines := buildChartLines(series, prevClose, chartW, chartH, 0, fi.item.Decimals, cclr)
	if r != nil && r.HistoryUnavailable {
		chartLines = unavailableHistory(chartW, chartH)
	}
	lines = append(lines, chartLines...)

	return strings.Join(lines, "\n")
}
