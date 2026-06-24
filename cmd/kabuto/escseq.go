package main

// parseEscapeSeq parses a terminal escape sequence that follows a leading ESC
// (0x1b) byte. The caller passes the bytes after ESC in `rest`.
//
// Return values:
//   - key:      the parsed Key (Esc=true for a lone/unknown ESC)
//   - consumed: number of bytes from `rest` that were consumed (not counting
//     the leading ESC handled by the caller)
//   - complete: true when a full sequence (or a definitive lone-ESC) was
//     recognized; false when `rest` holds only a partial/incomplete sequence
//     and the caller should wait for more bytes.
//
// Recognized sequences:
//   - lone ESC (rest empty)                -> Key{Esc:true}, complete
//   - CSI arrows  ESC [ A/B/C/D            -> Key{Up/Down/Right/Left}
func parseEscapeSeq(rest []byte) (key Key, consumed int, complete bool) {
	// Lone ESC: no following bytes.
	if len(rest) == 0 {
		return Key{Esc: true}, 0, true
	}
	// Must start a CSI ('['); anything else treated as a standalone ESC plus a
	// following key is out of scope — report a lone ESC, consuming nothing so
	// the trailing byte is processed normally on the next read.
	if rest[0] != '[' {
		return Key{Esc: true}, 0, true
	}
	if len(rest) < 2 {
		return Key{}, 0, false // incomplete: only "ESC ["
	}

	// Arrow keys: ESC [ A/B/C/D
	switch rest[1] {
	case 'A':
		return Key{Up: true}, 2, true
	case 'B':
		return Key{Down: true}, 2, true
	case 'C':
		return Key{Right: true}, 2, true
	case 'D':
		return Key{Left: true}, 2, true
	}

	// Unknown CSI sequence: consume up to and including a final byte in the
	// range 0x40..0x7E to resync; if none found yet, it's incomplete.
	for i := 1; i < len(rest); i++ {
		b := rest[i]
		if b >= 0x40 && b <= 0x7e {
			return Key{}, i + 1, true // recognized-but-ignored
		}
	}
	return Key{}, 0, false
}
