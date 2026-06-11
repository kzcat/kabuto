package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kaz/sekai-kabuka/internal/fetcher"
	"github.com/kaz/sekai-kabuka/internal/render"
	"github.com/kaz/sekai-kabuka/internal/symbols"
)

var version = "0.2.0"

const (
	enterAlt  = "\033[?1049h"
	leaveAlt  = "\033[?1049l"
	hideCur   = "\033[?25l"
	showCur   = "\033[?25h"
	cursorTop = "\033[H"
	clrLine   = "\033[K" // 行末まで消去
	clrBelow  = "\033[J" // 画面末尾まで消去
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

func main() {
	var sections sectionFlag
	var watchSec int
	var jsonOut bool
	var noColor bool
	var redGreen bool
	var showVersion bool

	flag.Var(&sections, "s", "表示セクション(複数指定可)")
	flag.Var(&sections, "section", "表示セクション(複数指定可)")
	flag.IntVar(&watchSec, "w", 0, "自動更新間隔(秒)")
	flag.IntVar(&watchSec, "watch", 0, "自動更新間隔(秒)")
	flag.BoolVar(&jsonOut, "j", false, "JSON出力")
	flag.BoolVar(&jsonOut, "json", false, "JSON出力")
	flag.BoolVar(&noColor, "no-color", false, "色なし")
	flag.BoolVar(&redGreen, "rg", false, "上昇=赤/下落=緑 に反転(日本式)")
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
	// パイプ時(非TTY)は色なし・ASCIIフォールバック
	if !render.UseColor(noColor) {
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

	opt := render.Options{NoColor: noColor, RedGreen: redGreen}

	frame := func() string {
		syms := collectSymbols()
		data := fetcher.FetchAll(syms)
		if jsonOut {
			return render.RenderJSON(data, sections)
		}
		return render.RenderDashboard(data, sections, opt)
	}

	if watchSec > 0 && !jsonOut {
		runWatch(watchSec, frame)
	} else {
		fmt.Println(frame())
	}
}

// runWatch はちらつき解消版の自動更新ループ。
// 代替スクリーンバッファ+カーソル非表示で開始し、SIGINT/SIGTERM で必ず復元する。
func runWatch(sec int, frame func() string) {
	out := os.Stdout

	restore := func() {
		fmt.Fprint(out, showCur+leaveAlt)
	}

	// シグナル捕捉して画面状態を復元
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		restore()
		os.Exit(0)
	}()

	fmt.Fprint(out, enterAlt+hideCur)
	defer restore()

	render1 := func(content string) {
		// 画面クリアのエスケープは使わず、ESC[H から1バッファに構築。
		// 各行末に ESC[K、最後に ESC[J を付けて1回の Write で出力。
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

	for {
		// データ取得中も前フレームは表示したまま。取得完了後に一括差し替え。
		content := frame()
		render1(content)
		time.Sleep(time.Duration(sec) * time.Second)
	}
}
