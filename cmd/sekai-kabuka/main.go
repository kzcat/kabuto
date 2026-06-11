package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kaz/sekai-kabuka/internal/fetcher"
	"github.com/kaz/sekai-kabuka/internal/render"
	"github.com/kaz/sekai-kabuka/internal/symbols"
)

var version = "0.1.0"

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

func main() {
	var sections sectionFlag
	var watchSec int
	var jsonOut bool
	var noColor bool
	var showVersion bool

	flag.Var(&sections, "s", "表示セクション(複数指定可)")
	flag.Var(&sections, "section", "表示セクション(複数指定可)")
	flag.IntVar(&watchSec, "w", 0, "自動更新間隔(秒)")
	flag.IntVar(&watchSec, "watch", 0, "自動更新間隔(秒)")
	flag.BoolVar(&jsonOut, "j", false, "JSON出力")
	flag.BoolVar(&jsonOut, "json", false, "JSON出力")
	flag.BoolVar(&noColor, "no-color", false, "色なし")
	flag.BoolVar(&showVersion, "v", false, "バージョン表示")
	flag.BoolVar(&showVersion, "version", false, "バージョン表示")
	flag.Parse()

	if showVersion {
		fmt.Printf("sekai-kabuka %s\n", version)
		os.Exit(0)
	}

	if jsonOut {
		noColor = true
	}

	collectSymbols := func() []string {
		keys := []string(sections)
		if len(keys) == 0 {
			keys = symbols.SectionOrder
		}
		seen := map[string]bool{}
		var syms []string
		for _, k := range keys {
			sec := symbols.Sections[k]
			for _, item := range sec.Items {
				if !seen[item.Symbol] {
					seen[item.Symbol] = true
					syms = append(syms, item.Symbol)
				}
			}
		}
		return syms
	}

	runOnce := func() {
		syms := collectSymbols()
		data := fetcher.FetchAll(syms)
		if jsonOut {
			fmt.Println(render.RenderJSON(data, sections))
		} else {
			fmt.Println(render.RenderTable(data, sections, noColor))
		}
	}

	if watchSec > 0 {
		for {
			fmt.Print("\033[2J\033[H")
			runOnce()
			time.Sleep(time.Duration(watchSec) * time.Second)
		}
	} else {
		runOnce()
	}
}
