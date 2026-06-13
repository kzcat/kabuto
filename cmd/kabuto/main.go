package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/i18n"
	"github.com/kzcat/kabuto/internal/locale"
	"github.com/kzcat/kabuto/internal/render"
	"github.com/kzcat/kabuto/internal/symbols"
	"github.com/kzcat/kabuto/internal/term"
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
	fmt.Fprintf(w, "      --lang CODE      UI language (en, ja, zh, ko, de, fr, es; default from $LANG)\n")
	fmt.Fprintf(w, "      --source auto|yahoo|stooq  Data source (default auto)\n")
	fmt.Fprintf(w, "      --range 1d|5d|1mo|6mo|1y   History range (default 1d)\n")
	fmt.Fprintf(w, "      --theme NAME     Color theme (default|mono|light|highcontrast)\n")
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
	var lang string
	var sourceFlag string
	var rangeFlag string
	var themeFlag string

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
	flag.StringVar(&lang, "lang", "", "UI language (en, ja, zh, ko, de, fr, es)")
	flag.BoolVar(&showVersion, "v", false, "Print version")
	flag.BoolVar(&showVersion, "version", false, "Print version")
	flag.StringVar(&sourceFlag, "source", "auto", "Data source (auto|yahoo|stooq)")
	flag.StringVar(&rangeFlag, "range", "1d", "Time range (1d|5d|1mo|6mo|1y)")
	flag.StringVar(&themeFlag, "theme", "default", "Color theme (default|mono|light|highcontrast)")
	flag.Parse()

	if showVersion {
		fmt.Printf("kabuto %s\n", version)
		os.Exit(0)
	}

	if jsonOut {
		noColor = true
	}
	// Honor NO_COLOR env (https://no-color.org)
	if _, ok := os.LookupEnv("NO_COLOR"); ok {
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

	// Resolve UI language: flag > env > en.
	resolvedLang := i18n.ResolveLang(lang)

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

	opt := render.Options{NoColor: noColor, RedGreen: redGreen, Loc: loc, CryptoItems: cryptoItems, Lang: resolvedLang, RangeLabel: rangeFlag, Theme: render.ThemeByName(themeFlag)}

	// Resolve range and sources
	rng := fetcher.ParseRange(rangeFlag)
	sources := resolveSources(sourceFlag)

	if watchSec > 0 && !jsonOut {
		runWatch(watchSec, collectSymbols, renderSections, opt, rng, sources)
	} else {
		syms := collectSymbols()
		data := fetcher.FetchAll(syms, rng, sources...)
		if jsonOut {
			fmt.Println(render.RenderJSON(data, renderSections, loc, resolvedLang))
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

// runWatch is the flicker-free auto-refresh loop with interactive key handling.
func runWatch(sec int, collect func() []string, sections []string, opt render.Options, rng fetcher.Range, sources []fetcher.Source) {
	out := os.Stdout

	// Try to enter raw mode for interactive keys.
	var rawState *term.State
	rawState, _ = term.MakeRaw(int(os.Stdin.Fd()))

	restore := func() {
		if rawState != nil {
			term.Restore(int(os.Stdin.Fd()), rawState)
		}
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

	// capture terminal resize (SIGWINCH on Unix, no-op elsewhere)
	winCh := watchResizeChan()

	fmt.Fprint(out, enterAlt+hideCur)
	defer restore()

	render1 := func(content string) {
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

	// UI state
	uiState := UIState{
		FillHeight: true,
		MinCols:    1,
		Range:      rng,
	}
	// Determine initial ColorMode from opt
	if opt.NoColor {
		uiState.ColorMode = ColorNone
	} else if opt.RedGreen {
		uiState.ColorMode = ColorJP
	}

	var lastData map[string]*fetcher.Result
	var lastCols int // last actual column count used

	draw := func() {
		cols, rows := render.DetectTermSize()
		o := uiState.applyTo(opt)
		o.TermWidth = cols
		o.TermRows = rows
		o.Watch = true

		// Determine effective sections
		drawSections := sections
		if uiState.Sections != nil {
			drawSections = uiState.Sections
		}

		content := render.RenderDashboard(lastData, drawSections, o)

		// Track actual cols for +/- key
		// Approximate: use what render would compute
		lastCols = render.ComputeCols(o, len(flatItems(drawSections, opt)))
		uiState.MaxCols = len(flatItems(drawSections, opt))

		if uiState.ShowHelp {
			content = overlayHelp(content, cols, rows, opt.Lang)
		}
		render1(content)
	}

	// Start key reader goroutine
	keyCh := make(chan Key, 8)
	if rawState != nil {
		go readKeys(os.Stdin, keyCh)
	} else {
		// Non-TTY: read stdin for 'q' or EOF
		go readKeys(os.Stdin, keyCh)
	}

	// Initial fetch
	lastData = fetcher.FetchAll(collect(), uiState.Range, sources...)
	draw()

	ticker := time.NewTicker(time.Duration(sec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case key, ok := <-keyCh:
			if !ok {
				// stdin closed (EOF)
				return
			}
			newState, action := Dispatch(uiState, key, lastCols, sections)
			uiState = newState
			switch action {
			case ActionQuit:
				return
			case ActionRefetch:
				drawSections := sections
				if uiState.Sections != nil {
					drawSections = uiState.Sections
				}
				_ = drawSections
				opt.RangeLabel = uiState.Range.String()
				lastData = fetcher.FetchAll(collect(), uiState.Range, sources...)
				draw()
			default:
				draw()
			}
		case <-ticker.C:
			if !uiState.Paused {
				lastData = fetcher.FetchAll(collect(), uiState.Range, sources...)
			}
			draw()
		case <-winCh:
			draw()
		}
	}
}

// resolveSources returns the Source slice for the --source flag value.
func resolveSources(s string) []fetcher.Source {
	switch s {
	case "yahoo":
		return []fetcher.Source{&fetcher.YahooSource{}}
	case "stooq":
		return []fetcher.Source{&fetcher.StooqSource{}}
	default: // "auto"
		return []fetcher.Source{&fetcher.YahooSource{}, &fetcher.StooqSource{}}
	}
}

// flatItems counts items for the given sections.
func flatItems(secs []string, opt render.Options) []struct{} {
	count := 0
	for _, k := range secs {
		sec := symbols.Sections[k]
		items := sec.Items
		if k == "crypto" && opt.CryptoItems != nil {
			items = opt.CryptoItems
		}
		count += len(items)
	}
	return make([]struct{}, count)
}

// readKeys reads from r one byte at a time and sends Keys on ch.
// Closes ch on EOF or error.
func readKeys(r io.Reader, ch chan<- Key) {
	defer close(ch)
	buf := make([]byte, 1)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			b := buf[0]
			if b == 0x1b {
				ch <- Key{Esc: true}
			} else {
				ch <- Key{R: rune(b)}
			}
		}
		if err != nil {
			return
		}
	}
}
