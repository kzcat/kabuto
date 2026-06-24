package main

// histLimit is the maximum number of samples retained per symbol in the
// rolling history buffer (B7). Roughly matches the widest chart we draw.
const histLimit = 256

// appendHist appends v to a rolling buffer, trimming from the front so the
// length never exceeds limit. It returns the (possibly reallocated) slice.
// A non-positive limit disables trimming.
func appendHist(buf []float64, v float64, limit int) []float64 {
	buf = append(buf, v)
	if limit > 0 && len(buf) > limit {
		buf = buf[len(buf)-limit:]
	}
	return buf
}

// seedHist initializes a rolling buffer from an intraday series, trimming to
// the most recent `limit` samples. Returns a fresh slice (never aliases src).
func seedHist(src []float64, limit int) []float64 {
	if len(src) == 0 {
		return nil
	}
	s := src
	if limit > 0 && len(s) > limit {
		s = s[len(s)-limit:]
	}
	out := make([]float64, len(s))
	copy(out, s)
	return out
}
