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
	ActionNone    Action = iota
	ActionRedraw
	ActionRefetch
	ActionQuit
)

// Key represents a parsed key input.
type Key struct {
	R   rune
	Esc bool
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
}

// Dispatch is a pure function: given current state + key + context, returns new state + action.
func Dispatch(st UIState, key Key, currentCols int, allSections []string) (UIState, Action) {
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

	if key.Esc {
		return st, ActionQuit
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
	return opt
}
