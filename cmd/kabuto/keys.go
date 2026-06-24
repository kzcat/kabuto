package main

import (
	"github.com/kzcat/kabuto/internal/fetcher"
	"github.com/kzcat/kabuto/internal/render"
)

// Color mode constants.
const (
	ColorNormal = 0
	ColorJP     = 1
	ColorNone   = 2
)

// Action represents what the main loop should do after a key press.
type Action int

const (
	ActionNone Action = iota
	ActionRedraw
	ActionRefetch
	ActionQuit
)

// Key represents a parsed key input.
type Key struct {
	R   rune
	Esc bool

	// Arrow keys (parsed from ESC [ A/B/C/D).
	Up    bool
	Down  bool
	Right bool
	Left  bool
}

// UIState holds the interactive UI state (independent of network/TTY).
type UIState struct {
	ColorMode  int
	Sections   []string // nil = all
	ForceCols  int      // 0 = auto
	FillHeight bool
	Paused     bool
	ShowHelp   bool
	MinCols    int
	MaxCols    int
	Range      fetcher.Range
	Sel        int  // grid selection index (-1 = none)
	Detail     bool // detail view active
	Preset     int  // layout preset (0=all, 1=majors, 2=fxcrypto)
}

// Dispatch is a pure function: given current state + key + context, returns new state + action.
// itemCount is the total number of grid items (used to clamp Sel).
func Dispatch(st UIState, key Key, currentCols int, allSections []string, itemCount int) (UIState, Action) {
	// Ctrl+C (ETX, byte 0x03) always quits. Raw mode disables ISIG, so it
	// arrives as a literal byte instead of raising SIGINT.
	if key.R == 3 {
		return st, ActionQuit
	}
	// If help is showing, any key closes it (except q/Esc which quit).
	if st.ShowHelp {
		if key.Esc || key.R == 'q' {
			return st, ActionQuit
		}
		st.ShowHelp = false
		return st, ActionRedraw
	}

	// Esc: priority Detail → Sel → quit
	if key.Esc {
		if st.Detail {
			st.Detail = false
			return st, ActionRedraw
		}
		if st.Sel >= 0 {
			st.Sel = -1
			return st, ActionRedraw
		}
		return st, ActionQuit
	}

	// Arrow keys: up/left = previous selection (like 'b'); down/right = next (like 'n').
	if key.Up || key.Left {
		st.Sel--
		if st.Sel < -1 {
			st.Sel = -1
		}
		return st, ActionRedraw
	}
	if key.Down || key.Right {
		if st.Sel < 0 {
			st.Sel = 0
		} else {
			st.Sel++
		}
		if itemCount > 0 && st.Sel >= itemCount {
			st.Sel = itemCount - 1
		}
		return st, ActionRedraw
	}

	switch key.R {
	case 'q':
		return st, ActionQuit
	case 'r':
		return st, ActionRefetch
	case 'c':
		st.ColorMode = (st.ColorMode + 1) % 3
		return st, ActionRedraw
	case '+':
		base := currentCols
		if st.ForceCols > 0 {
			base = st.ForceCols
		}
		st.ForceCols = clampCols(base+1, st.MinCols, st.MaxCols)
		return st, ActionRedraw
	case '-':
		base := currentCols
		if st.ForceCols > 0 {
			base = st.ForceCols
		}
		st.ForceCols = clampCols(base-1, st.MinCols, st.MaxCols)
		return st, ActionRedraw
	case '=':
		st.ForceCols = 0
		return st, ActionRedraw
	case '0', 'a':
		st.Sections = nil
		st.Sel = -1
		st.Detail = false
		return st, ActionRedraw
	case 'f':
		st.FillHeight = !st.FillHeight
		return st, ActionRedraw
	case ' ':
		st.Paused = !st.Paused
		return st, ActionRedraw
	case '?', 'h':
		st.ShowHelp = true
		return st, ActionRedraw
	case '[':
		prev := st.Range.Prev()
		if prev != st.Range {
			st.Range = prev
			return st, ActionRefetch
		}
		return st, ActionNone
	case ']':
		next := st.Range.Next()
		if next != st.Range {
			st.Range = next
			return st, ActionRefetch
		}
		return st, ActionNone
	case 'n', 9: // n or Tab (0x09)
		if st.Sel < 0 {
			st.Sel = 0
		} else {
			st.Sel++
		}
		if itemCount > 0 && st.Sel >= itemCount {
			st.Sel = itemCount - 1
		}
		return st, ActionRedraw
	case 'b', 'N':
		st.Sel--
		if st.Sel < -1 {
			st.Sel = -1
		}
		return st, ActionRedraw
	case 13: // Enter
		if st.Sel >= 0 {
			st.Detail = !st.Detail
		}
		return st, ActionRedraw
	case 'p':
		st.Preset = (st.Preset + 1) % 3
		switch st.Preset {
		case 0:
			st.Sections = nil
		case 1:
			st.Sections = []string{"japan", "us", "europe"}
		case 2:
			st.Sections = []string{"forex", "crypto"}
		}
		return st, ActionRedraw
	}

	// 1-9: section toggle
	if key.R >= '1' && key.R <= '9' {
		idx := int(key.R - '1')
		if idx >= len(allSections) {
			return st, ActionNone
		}
		target := allSections[idx]
		if len(st.Sections) == 1 && st.Sections[0] == target {
			st.Sections = nil
		} else {
			st.Sections = []string{target}
		}
		return st, ActionRedraw
	}

	return st, ActionNone
}

func clampCols(v, min, max int) int {
	if min > 0 && v < min {
		return min
	}
	if max > 0 && v > max {
		return max
	}
	return v
}

// applyTo converts UIState into render.Options fields.
// Preserves Theme already set on opt.
func (st UIState) applyTo(opt render.Options) render.Options {
	switch st.ColorMode {
	case ColorNormal:
		opt.NoColor = false
		opt.RedGreen = false
	case ColorJP:
		opt.NoColor = false
		opt.RedGreen = true
	case ColorNone:
		opt.NoColor = true
		opt.RedGreen = false
	}
	opt.ForceCols = st.ForceCols
	opt.FillHeight = st.FillHeight
	opt.SelIndex = st.Sel
	opt.DetailView = st.Detail
	return opt
}
