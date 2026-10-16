package fetcher

// changeBaseSymbols maps a symbol to the symbol whose previous close is used as
// its 1d change base. Nikkei futures and the after-hours JP225 perpetual are
// compared against the Nikkei 225 cash previous close rather than the CME
// settlement or the price 24h ago (as Japanese market sites do for "日経時間外"),
// so they share the index tile's baseline.
var changeBaseSymbols = map[string]string{
	"NKD=F":        "^N225",
	"hl:xyz:JP225": "^N225",
}

// withChangeBases returns syms plus any change-base symbols they need that are
// not already present, and the set of symbols that were added.
func withChangeBases(syms []string) ([]string, map[string]bool) {
	seen := make(map[string]bool, len(syms))
	for _, s := range syms {
		seen[s] = true
	}
	var added map[string]bool
	out := syms
	for _, s := range syms {
		base, ok := changeBaseSymbols[s]
		if !ok || seen[base] {
			continue
		}
		if added == nil {
			added = map[string]bool{}
			out = append([]string(nil), syms...)
		}
		seen[base] = true
		added[base] = true
		out = append(out, base)
	}
	return out, added
}

// applyChangeBases recomputes the change of each symbol with a change base
// against the base's previous close. Symbols whose base has no usable previous
// close keep their own change.
func applyChangeBases(results map[string]*Result) {
	for sym, base := range changeBaseSymbols {
		r := results[sym]
		b := results[base]
		if r == nil || b == nil || b.ChangeUnavailable || b.PrevClose == 0 {
			continue
		}
		rebased := *r
		rebased.PrevClose = b.PrevClose
		rebased.Change = r.Price - b.PrevClose
		rebased.ChangePct = rebased.Change / b.PrevClose * 100
		rebased.ChangeUnavailable = false
		results[sym] = &rebased
	}
}
