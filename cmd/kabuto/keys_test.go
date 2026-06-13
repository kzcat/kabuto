package main

import (
	"testing"

	"github.com/kzcat/kabuto/internal/render"
)

var testSections = []string{"japan", "us", "europe", "asia"}

func TestColorCycle(t *testing.T) {
	st := UIState{ColorMode: ColorNormal}
	st, _ = Dispatch(st, Key{R: 'c'}, 4, testSections)
	if st.ColorMode != ColorJP {
		t.Fatalf("expected ColorJP, got %d", st.ColorMode)
	}
	st, _ = Dispatch(st, Key{R: 'c'}, 4, testSections)
	if st.ColorMode != ColorNone {
		t.Fatalf("expected ColorNone, got %d", st.ColorMode)
	}
	st, _ = Dispatch(st, Key{R: 'c'}, 4, testSections)
	if st.ColorMode != ColorNormal {
		t.Fatalf("expected ColorNormal, got %d", st.ColorMode)
	}
}

func TestSectionToggle(t *testing.T) {
	st := UIState{}
	// '1' -> first section only
	st, act := Dispatch(st, Key{R: '1'}, 4, testSections)
	if act != ActionRedraw {
		t.Fatalf("expected redraw, got %v", act)
	}
	if len(st.Sections) != 1 || st.Sections[0] != "japan" {
		t.Fatalf("expected [japan], got %v", st.Sections)
	}
	// '1' again -> all (nil)
	st, _ = Dispatch(st, Key{R: '1'}, 4, testSections)
	if st.Sections != nil {
		t.Fatalf("expected nil, got %v", st.Sections)
	}
	// '2' -> us
	st, _ = Dispatch(st, Key{R: '2'}, 4, testSections)
	if len(st.Sections) != 1 || st.Sections[0] != "us" {
		t.Fatalf("expected [us], got %v", st.Sections)
	}
	// '0' -> all
	st, _ = Dispatch(st, Key{R: '0'}, 4, testSections)
	if st.Sections != nil {
		t.Fatalf("expected nil after '0', got %v", st.Sections)
	}
	// 'a' -> all
	st.Sections = []string{"japan"}
	st, _ = Dispatch(st, Key{R: 'a'}, 4, testSections)
	if st.Sections != nil {
		t.Fatalf("expected nil after 'a', got %v", st.Sections)
	}
	// Out of range: '9' with only 4 sections
	st, act = Dispatch(st, Key{R: '9'}, 4, testSections)
	if act != ActionNone {
		t.Fatalf("expected ActionNone for out-of-range, got %v", act)
	}
}

func TestColumnAdjust(t *testing.T) {
	st := UIState{MinCols: 1, MaxCols: 10}
	// Start at auto (ForceCols=0), currentCols=4
	st, act := Dispatch(st, Key{R: '+'}, 4, testSections)
	if act != ActionRedraw || st.ForceCols != 5 {
		t.Fatalf("expected ForceCols=5, got %d", st.ForceCols)
	}
	// '-' from ForceCols=5
	st, _ = Dispatch(st, Key{R: '-'}, 4, testSections)
	if st.ForceCols != 4 {
		t.Fatalf("expected ForceCols=4, got %d", st.ForceCols)
	}
	// Clamp at min
	st.ForceCols = 1
	st, _ = Dispatch(st, Key{R: '-'}, 4, testSections)
	if st.ForceCols != 1 {
		t.Fatalf("expected clamp at 1, got %d", st.ForceCols)
	}
	// Clamp at max
	st.ForceCols = 10
	st, _ = Dispatch(st, Key{R: '+'}, 4, testSections)
	if st.ForceCols != 10 {
		t.Fatalf("expected clamp at 10, got %d", st.ForceCols)
	}
	// '=' resets
	st, _ = Dispatch(st, Key{R: '='}, 4, testSections)
	if st.ForceCols != 0 {
		t.Fatalf("expected ForceCols=0 after '=', got %d", st.ForceCols)
	}
}

func TestPause(t *testing.T) {
	st := UIState{}
	st, _ = Dispatch(st, Key{R: ' '}, 4, testSections)
	if !st.Paused {
		t.Fatal("expected Paused=true")
	}
	st, _ = Dispatch(st, Key{R: ' '}, 4, testSections)
	if st.Paused {
		t.Fatal("expected Paused=false")
	}
}

func TestQuit(t *testing.T) {
	st := UIState{}
	_, act := Dispatch(st, Key{R: 'q'}, 4, testSections)
	if act != ActionQuit {
		t.Fatal("expected quit on 'q'")
	}
	_, act = Dispatch(st, Key{Esc: true}, 4, testSections)
	if act != ActionQuit {
		t.Fatal("expected quit on Esc")
	}
}

func TestRefetch(t *testing.T) {
	st := UIState{ColorMode: ColorJP, Paused: true}
	newSt, act := Dispatch(st, Key{R: 'r'}, 4, testSections)
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
	st, act := Dispatch(st, Key{R: '?'}, 4, testSections)
	if !st.ShowHelp || act != ActionRedraw {
		t.Fatal("expected ShowHelp=true")
	}
	// Any key closes help
	st, act = Dispatch(st, Key{R: 'x'}, 4, testSections)
	if st.ShowHelp || act != ActionRedraw {
		t.Fatal("expected ShowHelp=false after 'x'")
	}
	// 'h' also opens help
	st, _ = Dispatch(st, Key{R: 'h'}, 4, testSections)
	if !st.ShowHelp {
		t.Fatal("expected ShowHelp=true on 'h'")
	}
	// q in help -> quit
	_, act = Dispatch(st, Key{R: 'q'}, 4, testSections)
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
