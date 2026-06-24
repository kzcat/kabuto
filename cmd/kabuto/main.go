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

	"github.com/kzcat/kabuto/internal/config"
	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/i18n"
	"github.com/kzcat/kabuto/internal/locale"
	"github.com/kzcat/kabuto/internal/render"
	"github.com/kzcat/kabuto/internal/symbols"
	"github.com/kzcat/kabuto/internal/term"
)

var version = "0.3.0"

// normalizeArgs expands getopt-style short flags that glue a value to the flag
// (e.g. "-w1" or "-sus") into the separate "-w 1" / "-s us" form that Go's flag
// package understands. Only the value-taking short flags -s and -w are handled;
// boolean flags like -j/-v and any long ("--watch") flags are passed through
// unchanged. A "-w=1" form is left alone because flag already handles it.
// The input slice is never mutated; a new slice is returned.
func normalizeArgs(args []string) []string {
	out := make([]string, 0, len(args))
	for _, a := range args {
		if len(a) > 2 && a[0] == '-' && a[1] != '-' {
			c := a[1]
			if c == 's' || c == 'w' {
				rest := a[2:]
				if rest[0] != '=' {
					out = append(out, a[:2], rest)
					continue
				}
			}
		}
		out = append(out, a)
	}
	return out
}

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

type addFlag []string

func (a *addFlag) String() string { return strings.Join(*a, ",") }
func (a *addFlag) Set(v string) error {
	*a = append(*a, v)
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
	fmt.Fprintf(w, "      --theme NAME     Color theme (default|mono|light|highcontrast|dracula|nord|gruvbox|solarized)\n")
	fmt.Fprintf(w, "      --graph MODE     Chart symbol mode (auto|braille|block|tty; default auto)\n")
	fmt.Fprintf(w, "      --add SYMBOL[:CC[:DEC]]  Add ad-hoc symbol to Watchlist (repeatable)\n")
	fmt.Fprintf(w, "      --config PATH    Config file (default ~/.config/kabuto/config.json)\n")
	fmt.Fprintf(w, "  -v, --version        Print version and exit\n")
}

func main() {
	var sections sectionFlag
	var addSpecs addFlag
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
	var configPath string
	var graphFlag string

	flag.Usage = usage

	flag.Var(&sections, "s", "Show only these sections (repeatable)")
	flag.Var(&sections, "section", "Show only these sections (repeatable)")
	flag.Var(&addSpecs, "add", "Add ad-hoc symbol to Watchlist (repeatable)")
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
	flag.StringVar(&themeFlag, "theme", "default", "Color theme (default|mono|light|highcontrast|dracula|nord|gruvbox|solarized)")
	flag.StringVar(&configPath, "config", "", "Config file path")
	flag.StringVar(&graphFlag, "graph", "auto", "Chart symbol mode (auto|braille|block|tty)")
	flag.CommandLine.Parse(normalizeArgs(os.Args[1:]))

	if showVersion {
		fmt.Printf("kabuto %s\n", version)
		os.Exit(0)
	}

	// Load config
	cfgPath := configPath
	if cfgPath == "" {
		cfgPath = config.DefaultPath()
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Priority: CLI flag > config > env > default
	// Use flag.Visit to detect explicitly set flags
	set := map[string]bool{}
	flag.Visit(func(f *flag.Flag) { set[f.Name] = true })

	if !set["lang"] && cfg.Lang != "" {
		lang = cfg.Lang
	}
	if !set["country"] && cfg.Country != "" {
		country = cfg.Country
	}
	if !set["theme"] && cfg.Theme != "" {
		themeFlag = cfg.Theme
	}
	if !set["range"] && cfg.Range != "" {
		rangeFlag = cfg.Range
	}
	if !set["source"] && cfg.Source != "" {
		sourceFlag = cfg.Source
	}

	// Register custom sections from config
	config.RegisterSections(cfg.Sections)

	// Register --add items into "watch" section
	if len(addSpecs) > 0 {
		var items []symbols.Item
		for _, spec := range addSpecs {
			ic, err := config.ParseAddSpec(spec)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			items = append(items, ic.ToItem())
		}
		symbols.RegisterSection(symbols.Section{Key: "watch", Title: "Watchlist", Items: items})
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

	// Apply config section_order if -s not given
	if !explicit && len(cfg.SectionOrder) > 0 {
		var newOrder []string
		for _, k := range cfg.SectionOrder {
			if _, ok := symbols.Sections[k]; ok {
				newOrder = append(newOrder, k)
			}
		}
		// Append remaining built-in sections not in config order
		inOrder := map[string]bool{}
		for _, k := range newOrder {
			inOrder[k] = true
		}
		for _, k := range order {
			if !inOrder[k] {
				newOrder = append(newOrder, k)
			}
		}
		order = newOrder
	}

	// If "watch" section exists and not in order, prepend it (non-explicit only)
	if !explicit {
		if _, ok := symbols.Sections["watch"]; ok {
			hasWatch := false
			for _, k := range order {
				if k == "watch" {
					hasWatch = true
					break
				}
			}
			if !hasWatch {
				order = append([]string{"watch"}, order...)
			}
		}
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

	opt := render.Options{NoColor: noColor, RedGreen: redGreen, Loc: loc, CryptoItems: cryptoItems, Lang: resolvedLang, RangeLabel: rangeFlag, Theme: render.ThemeByName(themeFlag), GraphSymbol: graphFlag}

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
	if rawState != nil {
		// Poll-style reads (VMIN=0/VTIME=1, ~0.1s) so the key reader can tell a
		// lone ESC apart from the start of an arrow-key sequence.
		_ = term.SetReadTimeout(int(os.Stdin.Fd()), 1)
	}

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
		Sel:        -1,
	}
	// Determine initial ColorMode from opt
	if opt.NoColor {
		uiState.ColorMode = ColorNone
	} else if opt.RedGreen {
		uiState.ColorMode = ColorJP
	}

	var lastData map[string]*fetcher.Result
	var lastCols int // last actual column count used

	// B7: rolling history per symbol. Only accumulated/used when the active
	// range is 1d; for other ranges the API Series is used as-is.
	hist := map[string][]float64{}
	histInit := false // whether hist has been seeded from the first 1d fetch

	// recordHist updates the rolling buffers from a fresh fetch result set.
	// On the first 1d fetch it seeds from each result's intraday Series; on
	// subsequent fetches it appends the latest price. Cleared when not 1d.
	recordHist := func(data map[string]*fetcher.Result) {
		if uiState.Range != fetcher.Range1D {
			// Non-1d: history not used; reset so a later switch back to 1d reseeds.
			hist = map[string][]float64{}
			histInit = false
			return
		}
		if !histInit {
			for sym, r := range data {
				if r != nil {
					hist[sym] = seedHist(r.Series, histLimit)
				}
			}
			histInit = true
			return
		}
		for sym, r := range data {
			if r != nil {
				hist[sym] = appendHist(hist[sym], r.Price, histLimit)
			}
		}
	}

	draw := func() {
		cols, rows := render.DetectTermSize()
		o := uiState.applyTo(opt)
		o.TermWidth = cols
		o.TermRows = rows
		o.Watch = true
		// B7: only feed rolling history to the renderer for the 1d range.
		if uiState.Range == fetcher.Range1D {
			o.History = hist
		}

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
	// rawState != nil means VTIME polling is active (ignore timeout EOFs);
	// otherwise stdin is a pipe where a real EOF should close the reader.
	go readKeys(os.Stdin, rawState != nil, keyCh)

	// Initial fetch
	lastData = fetcher.FetchAll(collect(), uiState.Range, sources...)
	recordHist(lastData)
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
			newState, action := Dispatch(uiState, key, lastCols, sections, uiState.MaxCols)
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
				recordHist(lastData)
				draw()
			default:
				draw()
			}
		case <-ticker.C:
			if !uiState.Paused {
				lastData = fetcher.FetchAll(collect(), uiState.Range, sources...)
				recordHist(lastData)
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

// readKeys reads from r and sends parsed Keys on ch. It understands ESC
// sequences (arrow keys) by buffering bytes after a 0x1b and
// delegating to parseEscapeSeq. To distinguish a lone ESC (back/quit) from the
// start of a sequence, the caller is expected to have set the tty to a polling
// read (VMIN=0/VTIME=1) so Read returns promptly when no further byte arrives.
// When hasTimeout is set the read is poll-style: a 0-byte read (reported as
// io.EOF by os.File on a tty) means "no key yet", not a real close, so it is
// ignored. Without it (non-tty/pipe), io.EOF closes ch.
func readKeys(r io.Reader, hasTimeout bool, ch chan<- Key) {
	defer close(ch)
	rd := make([]byte, 64)
	var pending []byte // buffered bytes belonging to an in-progress ESC sequence

	flushPending := func() {
		// We have an ESC followed by `pending` bytes (without the ESC). Try to
		// parse; if still incomplete after a timeout, treat as a lone ESC and
		// re-emit the buffered bytes as ordinary keys.
		for {
			if len(pending) == 0 {
				ch <- Key{Esc: true}
				return
			}
			key, consumed, complete := parseEscapeSeq(pending)
			if !complete {
				// Incomplete and no more data arrived: lone ESC, then replay.
				ch <- Key{Esc: true}
				for _, b := range pending {
					ch <- Key{R: rune(b)}
				}
				pending = nil
				return
			}
			ch <- key
			pending = pending[consumed:]
			if len(pending) == 0 {
				return
			}
			// Loop to parse any remaining buffered bytes (another ESC or key).
			if pending[0] == 0x1b {
				pending = pending[1:]
				continue
			}
			b := pending[0]
			pending = pending[1:]
			ch <- Key{R: rune(b)}
		}
	}

	for {
		n, err := r.Read(rd)
		if err == io.EOF && hasTimeout {
			// VMIN=0/VTIME poll: a 0-byte tty read surfaces as io.EOF but just
			// means no key arrived in this interval. Keep polling.
			err = nil
		}
		if n > 0 {
			data := rd[:n]
			for len(data) > 0 {
				b := data[0]
				if b == 0x1b {
					data = data[1:]
					// Collect the rest of this read as the candidate sequence.
					pending = append(pending[:0], data...)
					key, consumed, complete := parseEscapeSeq(pending)
					if complete {
						ch <- key
						data = pending[consumed:]
						pending = nil
						continue
					}
					// Incomplete: wait for the next read (or timeout) to finish.
					data = nil
				} else if len(pending) > 0 {
					// Continuation bytes of an in-progress sequence.
					pending = append(pending, b)
					data = data[1:]
					key, consumed, complete := parseEscapeSeq(pending)
					if complete {
						ch <- key
						rem := pending[consumed:]
						pending = nil
						// Prepend any leftover to remaining data.
						if len(rem) > 0 {
							data = append(append([]byte{}, rem...), data...)
						}
					}
				} else {
					ch <- Key{R: rune(b)}
					data = data[1:]
				}
			}
		} else if len(pending) > 0 {
			// Timed-out read (VTIME) with a buffered partial sequence: resolve it.
			flushPending()
		}
		if err != nil {
			if len(pending) > 0 {
				flushPending()
			}
			return
		}
	}
}
