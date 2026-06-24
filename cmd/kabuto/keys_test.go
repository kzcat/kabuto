package main

import (
	"testing"

	"github.com/kzcat/kabuto/internal/render"
)

var testSections = []string{"japan", "us", "europe", "asia"}

func TestColorCycle(t *testing.T) {
	st := UIState{ColorMode: ColorNormal}
	st, _ = Dispatch(st, Key{R: 'c'}, 4, testSections, 10)
	if st.ColorMode != ColorJP {
		t.Fatalf("expected ColorJP, got %d", st.ColorMode)
	}
	st, _ = Dispatch(st, Key{R: 'c'}, 4, testSections, 10)
	if st.ColorMode != ColorNone {
		t.Fatalf("expected ColorNone, got %d", st.ColorMode)
	}
	st, _ = Dispatch(st, Key{R: 'c'}, 4, testSections, 10)
	if st.ColorMode != ColorNormal {
		t.Fatalf("expected ColorNormal, got %d", st.ColorMode)
	}
}

func TestSectionToggle(t *testing.T) {
	st := UIState{}
	// '1' -> first section only
	st, act := Dispatch(st, Key{R: '1'}, 4, testSections, 10)
	if act != ActionRedraw {
		t.Fatalf("expected redraw, got %v", act)
	}
	if len(st.Sections) != 1 || st.Sections[0] != "japan" {
		t.Fatalf("expected [japan], got %v", st.Sections)
	}
	// '1' again -> all (nil)
	st, _ = Dispatch(st, Key{R: '1'}, 4, testSections, 10)
	if st.Sections != nil {
		t.Fatalf("expected nil, got %v", st.Sections)
	}
	// '2' -> us
	st, _ = Dispatch(st, Key{R: '2'}, 4, testSections, 10)
	if len(st.Sections) != 1 || st.Sections[0] != "us" {
		t.Fatalf("expected [us], got %v", st.Sections)
	}
	// '0' -> all
	st, _ = Dispatch(st, Key{R: '0'}, 4, testSections, 10)
	if st.Sections != nil {
		t.Fatalf("expected nil after '0', got %v", st.Sections)
	}
	// 'a' -> all
	st.Sections = []string{"japan"}
	st, _ = Dispatch(st, Key{R: 'a'}, 4, testSections, 10)
	if st.Sections != nil {
		t.Fatalf("expected nil after 'a', got %v", st.Sections)
	}
	// Out of range: '9' with only 4 sections
	st, act = Dispatch(st, Key{R: '9'}, 4, testSections, 10)
	if act != ActionNone {
		t.Fatalf("expected ActionNone for out-of-range, got %v", act)
	}
}

func TestColumnAdjust(t *testing.T) {
	st := UIState{MinCols: 1, MaxCols: 10}
	// Start at auto (ForceCols=0), currentCols=4
	st, act := Dispatch(st, Key{R: '+'}, 4, testSections, 10)
	if act != ActionRedraw || st.ForceCols != 5 {
		t.Fatalf("expected ForceCols=5, got %d", st.ForceCols)
	}
	// '-' from ForceCols=5
	st, _ = Dispatch(st, Key{R: '-'}, 4, testSections, 10)
	if st.ForceCols != 4 {
		t.Fatalf("expected ForceCols=4, got %d", st.ForceCols)
	}
	// Clamp at min
	st.ForceCols = 1
	st, _ = Dispatch(st, Key{R: '-'}, 4, testSections, 10)
	if st.ForceCols != 1 {
		t.Fatalf("expected clamp at 1, got %d", st.ForceCols)
	}
	// Clamp at max
	st.ForceCols = 10
	st, _ = Dispatch(st, Key{R: '+'}, 4, testSections, 10)
	if st.ForceCols != 10 {
		t.Fatalf("expected clamp at 10, got %d", st.ForceCols)
	}
	// '=' resets
	st, _ = Dispatch(st, Key{R: '='}, 4, testSections, 10)
	if st.ForceCols != 0 {
		t.Fatalf("expected ForceCols=0 after '=', got %d", st.ForceCols)
	}
}

func TestPause(t *testing.T) {
	st := UIState{}
	st, _ = Dispatch(st, Key{R: ' '}, 4, testSections, 10)
	if !st.Paused {
		t.Fatal("expected Paused=true")
	}
	st, _ = Dispatch(st, Key{R: ' '}, 4, testSections, 10)
	if st.Paused {
		t.Fatal("expected Paused=false")
	}
}

func TestQuit(t *testing.T) {
	st := UIState{Sel: -1}
	_, act := Dispatch(st, Key{R: 'q'}, 4, testSections, 10)
	if act != ActionQuit {
		t.Fatal("expected quit on 'q'")
	}
	_, act = Dispatch(st, Key{Esc: true}, 4, testSections, 10)
	if act != ActionQuit {
		t.Fatal("expected quit on Esc")
	}
	// Ctrl+C (ETX byte 0x03) must quit even though raw mode disables ISIG.
	_, act = Dispatch(st, Key{R: 3}, 4, testSections, 10)
	if act != ActionQuit {
		t.Fatal("expected quit on Ctrl+C")
	}
	// Ctrl+C must quit even while the help overlay is open.
	_, act = Dispatch(UIState{ShowHelp: true}, Key{R: 3}, 4, testSections, 10)
	if act != ActionQuit {
		t.Fatal("expected quit on Ctrl+C with help open")
	}
}

func TestRefetch(t *testing.T) {
	st := UIState{ColorMode: ColorJP, Paused: true}
	newSt, act := Dispatch(st, Key{R: 'r'}, 4, testSections, 10)
	if act != ActionRefetch {
		t.Fatal("expected refetch")
	}
	// State unchanged
	if newSt.ColorMode != ColorJP || !newSt.Paused {
		t.Fatal("state should not change on refetch")
	}
}

func TestHelp(t *testing.T) {
	st := UIState{}
	st, act := Dispatch(st, Key{R: '?'}, 4, testSections, 10)
	if !st.ShowHelp || act != ActionRedraw {
		t.Fatal("expected ShowHelp=true")
	}
	// Any key closes help
	st, act = Dispatch(st, Key{R: 'x'}, 4, testSections, 10)
	if st.ShowHelp || act != ActionRedraw {
		t.Fatal("expected ShowHelp=false after 'x'")
	}
	// 'h' also opens help
	st, _ = Dispatch(st, Key{R: 'h'}, 4, testSections, 10)
	if !st.ShowHelp {
		t.Fatal("expected ShowHelp=true on 'h'")
	}
	// q in help -> quit
	_, act = Dispatch(st, Key{R: 'q'}, 4, testSections, 10)
	if act != ActionQuit {
		t.Fatal("expected quit in help on 'q'")
	}
}

func TestApplyTo(t *testing.T) {
	cases := []struct {
		mode    int
		noColor bool
		rg      bool
	}{
		{ColorNormal, false, false},
		{ColorJP, false, true},
		{ColorNone, true, false},
	}
	for _, c := range cases {
		st := UIState{ColorMode: c.mode, ForceCols: 3, FillHeight: true}
		opt := st.applyTo(render.Options{})
		if opt.NoColor != c.noColor {
			t.Errorf("mode %d: NoColor=%v want %v", c.mode, opt.NoColor, c.noColor)
		}
		if opt.RedGreen != c.rg {
			t.Errorf("mode %d: RedGreen=%v want %v", c.mode, opt.RedGreen, c.rg)
		}
		if opt.ForceCols != 3 {
			t.Errorf("ForceCols=%d want 3", opt.ForceCols)
		}
		if !opt.FillHeight {
			t.Error("FillHeight should be true")
		}
	}
}

func TestSelNext(t *testing.T) {
	// n from -1 -> 0
	st := UIState{Sel: -1}
	st, act := Dispatch(st, Key{R: 'n'}, 4, testSections, 5)
	if act != ActionRedraw || st.Sel != 0 {
		t.Fatalf("n from -1: want Sel=0, got %d", st.Sel)
	}
	// n increments
	st, _ = Dispatch(st, Key{R: 'n'}, 4, testSections, 5)
	if st.Sel != 1 {
		t.Fatalf("n from 0: want Sel=1, got %d", st.Sel)
	}
	// Tab (0x09) also moves next
	st, _ = Dispatch(st, Key{R: 9}, 4, testSections, 5)
	if st.Sel != 2 {
		t.Fatalf("Tab from 1: want Sel=2, got %d", st.Sel)
	}
	// Clamp at itemCount-1
	st.Sel = 4
	st, _ = Dispatch(st, Key{R: 'n'}, 4, testSections, 5)
	if st.Sel != 4 {
		t.Fatalf("n clamp: want Sel=4, got %d", st.Sel)
	}
}

func TestSelPrev(t *testing.T) {
	st := UIState{Sel: 2}
	st, act := Dispatch(st, Key{R: 'b'}, 4, testSections, 5)
	if act != ActionRedraw || st.Sel != 1 {
		t.Fatalf("b from 2: want Sel=1, got %d", st.Sel)
	}
	st, _ = Dispatch(st, Key{R: 'b'}, 4, testSections, 5)
	if st.Sel != 0 {
		t.Fatalf("b from 1: want Sel=0, got %d", st.Sel)
	}
	st, _ = Dispatch(st, Key{R: 'b'}, 4, testSections, 5)
	if st.Sel != -1 {
		t.Fatalf("b from 0: want Sel=-1, got %d", st.Sel)
	}
	// N also works
	st.Sel = 3
	st, _ = Dispatch(st, Key{R: 'N'}, 4, testSections, 5)
	if st.Sel != 2 {
		t.Fatalf("N from 3: want Sel=2, got %d", st.Sel)
	}
}

func TestDetailToggle(t *testing.T) {
	// Enter with Sel>=0 toggles Detail
	st := UIState{Sel: 2}
	st, act := Dispatch(st, Key{R: 13}, 4, testSections, 5)
	if act != ActionRedraw || !st.Detail {
		t.Fatal("Enter should toggle Detail on")
	}
	st, _ = Dispatch(st, Key{R: 13}, 4, testSections, 5)
	if st.Detail {
		t.Fatal("Enter should toggle Detail off")
	}
	// Enter with Sel=-1 does nothing meaningful (still returns Redraw)
	st = UIState{Sel: -1}
	st, _ = Dispatch(st, Key{R: 13}, 4, testSections, 5)
	if st.Detail {
		t.Fatal("Enter with Sel=-1 should not set Detail")
	}
}

func TestEscPriority(t *testing.T) {
	// Priority 1: Detail=true -> close detail only
	st := UIState{Sel: 2, Detail: true}
	st, act := Dispatch(st, Key{Esc: true}, 4, testSections, 5)
	if act != ActionRedraw || st.Detail || st.Sel != 2 {
		t.Fatalf("Esc with Detail: want Detail=false Sel=2, got Detail=%v Sel=%d", st.Detail, st.Sel)
	}
	// Priority 2: Sel>=0 -> deselect
	st, act = Dispatch(st, Key{Esc: true}, 4, testSections, 5)
	if act != ActionRedraw || st.Sel != -1 {
		t.Fatalf("Esc with Sel>=0: want Sel=-1, got %d", st.Sel)
	}
	// Priority 3: quit
	st, act = Dispatch(st, Key{Esc: true}, 4, testSections, 5)
	if act != ActionQuit {
		t.Fatal("Esc with Sel=-1 Detail=false should quit")
	}
}

func TestPresetCycle(t *testing.T) {
	st := UIState{Sel: -1}
	// 0 -> 1 (majors)
	st, act := Dispatch(st, Key{R: 'p'}, 4, testSections, 10)
	if act != ActionRedraw || st.Preset != 1 {
		t.Fatalf("p: want Preset=1, got %d", st.Preset)
	}
	if len(st.Sections) != 3 || st.Sections[0] != "japan" || st.Sections[1] != "us" || st.Sections[2] != "europe" {
		t.Fatalf("p preset 1: want [japan us europe], got %v", st.Sections)
	}
	// 1 -> 2 (fxcrypto)
	st, _ = Dispatch(st, Key{R: 'p'}, 4, testSections, 10)
	if st.Preset != 2 {
		t.Fatalf("p: want Preset=2, got %d", st.Preset)
	}
	if len(st.Sections) != 2 || st.Sections[0] != "forex" || st.Sections[1] != "crypto" {
		t.Fatalf("p preset 2: want [forex crypto], got %v", st.Sections)
	}
	// 2 -> 0 (all)
	st, _ = Dispatch(st, Key{R: 'p'}, 4, testSections, 10)
	if st.Preset != 0 || st.Sections != nil {
		t.Fatalf("p preset 0: want Preset=0 Sections=nil, got Preset=%d Sections=%v", st.Preset, st.Sections)
	}
}

func TestAllResetsSelDetail(t *testing.T) {
	st := UIState{Sel: 3, Detail: true, Sections: []string{"us"}}
	st, _ = Dispatch(st, Key{R: '0'}, 4, testSections, 10)
	if st.Sel != -1 || st.Detail || st.Sections != nil {
		t.Fatalf("'0' should reset Sel/Detail/Sections, got Sel=%d Detail=%v Sections=%v", st.Sel, st.Detail, st.Sections)
	}
	st = UIState{Sel: 2, Detail: true}
	st, _ = Dispatch(st, Key{R: 'a'}, 4, testSections, 10)
	if st.Sel != -1 || st.Detail {
		t.Fatalf("'a' should reset, got Sel=%d Detail=%v", st.Sel, st.Detail)
	}
}
