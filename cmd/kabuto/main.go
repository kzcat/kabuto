package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/locale"
	"github.com/kzcat/kabuto/internal/render"
	"github.com/kzcat/kabuto/internal/symbols"
)

var version = "0.2.0"

const (
	enterAlt  = "\033[?1049h"
	leaveAlt  = "\033[?1049l"
	hideCur   = "\033[?25l"
	showCur   = "\033[?25h"
	cursorTop = "\033[H"
	clrLine   = "\033[K" // clear to end of line
	clrBelow  = "\033[J" // clear to end of screen
)

type sectionFlag []string

func (s *sectionFlag) String() string { return strings.Join(*s, ",") }
func (s *sectionFlag) Set(v string) error {
	valid := map[string]bool{}
	for _, k := range symbols.SectionOrder {
		valid[k] = true
	}
	if !valid[v] {
		return fmt.Errorf("invalid section: %s (valid: %s)", v, strings.Join(symbols.SectionOrder, ", "))
	}
	*s = append(*s, v)
	return nil
}

func usage() {
	w := flag.CommandLine.Output()
	fmt.Fprintf(w, "Usage: kabuto [options]\n\n")
	fmt.Fprintf(w, "kabuto shows global market indices, forex, crypto and commodities in your terminal.\n")
	fmt.Fprintf(w, "With no arguments it prints every section once.\n\n")
	fmt.Fprintf(w, "Options:\n")
	fmt.Fprintf(w, "  -s, --section NAME   Show only these sections (repeatable: %s)\n", strings.Join(symbols.SectionOrder, ","))
	fmt.Fprintf(w, "  -w, --watch SECONDS  Auto-refresh every SECONDS seconds\n")
	fmt.Fprintf(w, "      --rg             Use red=up / green=down (Japanese convention)\n")
	fmt.Fprintf(w, "  -j, --json           Output JSON instead of the dashboard\n")
	fmt.Fprintf(w, "      --no-color       Disable colors and use ASCII box drawing\n")
	fmt.Fprintf(w, "      --tz NAME        Display times in the given IANA timezone (e.g. Asia/Tokyo)\n")
	fmt.Fprintf(w, "      --country ISO2   Override detected home market country (e.g. JP, US, DE)\n")
	fmt.Fprintf(w, "  -v, --version        Print version and exit\n")
}

func main() {
	var sections sectionFlag
	var watchSec int
	var jsonOut bool
	var noColor bool
	var redGreen bool
	var showVersion bool
	var tz string
	var country string

	flag.Usage = usage

	flag.Var(&sections, "s", "Show only these sections (repeatable)")
	flag.Var(&sections, "section", "Show only these sections (repeatable)")
	flag.IntVar(&watchSec, "w", 0, "Auto-refresh interval in seconds")
	flag.IntVar(&watchSec, "watch", 0, "Auto-refresh interval in seconds")
	flag.BoolVar(&jsonOut, "j", false, "Output JSON")
	flag.BoolVar(&jsonOut, "json", false, "Output JSON")
	flag.BoolVar(&noColor, "no-color", false, "Disable colors")
	flag.BoolVar(&redGreen, "rg", false, "Use red=up / green=down (Japanese convention)")
	flag.StringVar(&tz, "tz", "", "Display timezone (IANA name)")
	flag.StringVar(&country, "country", "", "Override home market country (ISO2)")
	flag.BoolVar(&showVersion, "v", false, "Print version")
	flag.BoolVar(&showVersion, "version", false, "Print version")
	flag.Parse()

	if showVersion {
		fmt.Printf("kabuto %s\n", version)
		os.Exit(0)
	}

	if jsonOut {
		noColor = true
	}
	// whether stdout is a TTY (not a pipe/redirect)
	isTTY := render.UseColor(false)
	// non-TTY (piped) → no color + ASCII fallback
	if !render.UseColor(noColor) {
		noColor = true
	}

	// Determine home market country: flag > env > default US.
	cc := locale.ResolveCountry(country)

	// Home-market-first ordering applies only when -s is not given.
	explicit := len(sections) > 0
	order := symbols.SectionOrder
	if !explicit {
		order = locale.HomeFirstOrder(cc)
	}

	// Determine display location: --tz > time.Local.
	loc := locale.ResolveLocation(tz)

	// crypto items reordered by country (BTC-JPY first for JP, BTC-USD otherwise).
	cryptoItems := locale.CryptoItems(cc)

	collectSymbols := func() []string {
		keys := []string(sections)
		if len(keys) == 0 {
			keys = order
		}
		seen := map[string]bool{}
		var syms []string
		for _, k := range keys {
			items := symbols.Sections[k].Items
			if k == "crypto" {
				items = cryptoItems
			}
			for _, item := range items {
				if !seen[item.Symbol] {
					seen[item.Symbol] = true
					syms = append(syms, item.Symbol)
				}
			}
		}
		return syms
	}

	// effective section list passed to the renderer (home-first when applicable).
	var renderSections []string
	if explicit {
		renderSections = []string(sections)
	} else {
		renderSections = order
	}

	opt := render.Options{NoColor: noColor, RedGreen: redGreen, Loc: loc, CryptoItems: cryptoItems}

	if watchSec > 0 && !jsonOut {
		runWatch(watchSec, collectSymbols, renderSections, opt)
	} else {
		syms := collectSymbols()
		data := fetcher.FetchAll(syms)
		if jsonOut {
			fmt.Println(render.RenderJSON(data, renderSections, loc))
		} else {
			// single one-shot render. If stdout is a TTY, fill the height.
			o := opt
			if isTTY {
				cols, rows := render.DetectTermSize()
				o.TermWidth = cols
				o.TermRows = rows
				o.FillHeight = true
			}
			fmt.Println(render.RenderDashboard(data, renderSections, o))
		}
	}
}

// runWatch is the flicker-free auto-refresh loop.
// It enters the alternate screen buffer with a hidden cursor and always
// restores on SIGINT/SIGTERM. SIGWINCH triggers an immediate redraw from the
// last data without re-fetching.
func runWatch(sec int, collect func() []string, sections []string, opt render.Options) {
	out := os.Stdout

	restore := func() {
		fmt.Fprint(out, showCur+leaveAlt)
	}

	// restore screen and exit on SIGINT/SIGTERM
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		restore()
		os.Exit(0)
	}()

	// capture SIGWINCH (terminal resize)
	winCh := make(chan os.Signal, 1)
	signal.Notify(winCh, syscall.SIGWINCH)

	fmt.Fprint(out, enterAlt+hideCur)
	defer restore()

	render1 := func(content string) {
		// build into one buffer from ESC[H without clearing. ESC[K per line, ESC[J at end.
		var b strings.Builder
		b.WriteString(cursorTop)
		lines := strings.Split(content, "\n")
		for i, ln := range lines {
			b.WriteString(ln)
			b.WriteString(clrLine)
			if i < len(lines)-1 {
				b.WriteString("\r\n")
			}
		}
		b.WriteString(clrBelow)
		fmt.Fprint(out, b.String())
	}

	// cache the last data so resizes can reuse it
	var lastData map[string]*fetcher.Result

	draw := func() {
		// re-read terminal size every frame before drawing
		cols, rows := render.DetectTermSize()
		o := opt
		o.TermWidth = cols
		o.TermRows = rows
		o.Watch = true
		render1(render.RenderDashboard(lastData, sections, o))
	}

	// initial fetch
	lastData = fetcher.FetchAll(collect())
	draw()

	ticker := time.NewTicker(time.Duration(sec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-winCh:
			// resize: redraw from last data without re-fetching
			draw()
		case <-ticker.C:
			// periodic refresh: keep previous frame while fetching, then swap
			lastData = fetcher.FetchAll(collect())
			draw()
		}
	}
}
