package main

import "testing"

func TestParseEscapeSeqArrows(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want Key
	}{
		{"up", "[A", Key{Up: true}},
		{"down", "[B", Key{Down: true}},
		{"right", "[C", Key{Right: true}},
		{"left", "[D", Key{Left: true}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			k, consumed, complete := parseEscapeSeq([]byte(c.in))
			if !complete {
				t.Fatalf("%s: expected complete", c.name)
			}
			if consumed != len(c.in) {
				t.Fatalf("%s: consumed=%d want %d", c.name, consumed, len(c.in))
			}
			if k != c.want {
				t.Fatalf("%s: got %+v want %+v", c.name, k, c.want)
			}
		})
	}
}

func TestParseEscapeSeqLoneEsc(t *testing.T) {
	k, consumed, complete := parseEscapeSeq(nil)
	if !complete || consumed != 0 || !k.Esc {
		t.Fatalf("lone ESC: got %+v consumed=%d complete=%v", k, consumed, complete)
	}
	// ESC followed by a non-'[' byte: treated as lone ESC, consumes nothing.
	k, consumed, complete = parseEscapeSeq([]byte("x"))
	if !complete || consumed != 0 || !k.Esc {
		t.Fatalf("ESC+x: got %+v consumed=%d complete=%v", k, consumed, complete)
	}
}

func TestParseEscapeSeqIncomplete(t *testing.T) {
	// Only "ESC [" — incomplete, await more bytes.
	_, _, complete := parseEscapeSeq([]byte("["))
	if complete {
		t.Fatal(`"[" should be incomplete`)
	}
	// Unterminated CSI sequence (no final byte yet) — incomplete.
	_, _, complete = parseEscapeSeq([]byte("[<0;10;5"))
	if complete {
		t.Fatal("unterminated CSI should be incomplete")
	}
}

func TestDispatchArrowKeys(t *testing.T) {
	// Down/Right advance selection (like 'n').
	st := UIState{Sel: -1}
	st, act := Dispatch(st, Key{Down: true}, 4, testSections, 5)
	if act != ActionRedraw || st.Sel != 0 {
		t.Fatalf("Down from -1: want Sel=0, got %d", st.Sel)
	}
	st, _ = Dispatch(st, Key{Right: true}, 4, testSections, 5)
	if st.Sel != 1 {
		t.Fatalf("Right: want Sel=1, got %d", st.Sel)
	}
	// Up/Left go back (like 'b').
	st, _ = Dispatch(st, Key{Up: true}, 4, testSections, 5)
	if st.Sel != 0 {
		t.Fatalf("Up: want Sel=0, got %d", st.Sel)
	}
	st, _ = Dispatch(st, Key{Left: true}, 4, testSections, 5)
	if st.Sel != -1 {
		t.Fatalf("Left: want Sel=-1, got %d", st.Sel)
	}
}

func TestAppendHist(t *testing.T) {
	var buf []float64
	for i := 0; i < 5; i++ {
		buf = appendHist(buf, float64(i), 3)
	}
	if len(buf) != 3 {
		t.Fatalf("len=%d want 3", len(buf))
	}
	// Should retain the most recent values: 2,3,4.
	if buf[0] != 2 || buf[1] != 3 || buf[2] != 4 {
		t.Fatalf("ring contents=%v want [2 3 4]", buf)
	}
	// Non-positive limit disables trimming.
	buf2 := appendHist([]float64{1, 2}, 3, 0)
	if len(buf2) != 3 {
		t.Fatalf("no-limit len=%d want 3", len(buf2))
	}
}

func TestSeedHist(t *testing.T) {
	src := []float64{1, 2, 3, 4, 5}
	out := seedHist(src, 3)
	if len(out) != 3 || out[0] != 3 || out[2] != 5 {
		t.Fatalf("seed trim: got %v", out)
	}
	// Must be a copy, not aliasing src.
	out[0] = 99
	if src[2] == 99 {
		t.Fatal("seedHist must not alias src")
	}
	if seedHist(nil, 3) != nil {
		t.Fatal("seed of empty should be nil")
	}
}
