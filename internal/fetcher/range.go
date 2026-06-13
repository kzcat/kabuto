package fetcher

// Range represents a time range for chart data.
type Range int

const (
	Range1D  Range = iota
	Range5D
	Range1Mo
	Range6Mo
	Range1Y
)

// AllRanges is the ordered list of valid ranges.
var AllRanges = []Range{Range1D, Range5D, Range1Mo, Range6Mo, Range1Y}

// String returns the flag representation.
func (r Range) String() string {
	switch r {
	case Range5D:
		return "5d"
	case Range1Mo:
		return "1mo"
	case Range6Mo:
		return "6mo"
	case Range1Y:
		return "1y"
	default:
		return "1d"
	}
}

// ParseRange converts a string to a Range.
func ParseRange(s string) Range {
	switch s {
	case "5d":
		return Range5D
	case "1mo":
		return Range1Mo
	case "6mo":
		return Range6Mo
	case "1y":
		return Range1Y
	default:
		return Range1D
	}
}

// Interval returns the Yahoo interval for this range.
func (r Range) Interval() string {
	switch r {
	case Range5D:
		return "15m"
	case Range1Mo:
		return "1d"
	case Range6Mo:
		return "1d"
	case Range1Y:
		return "1wk"
	default:
		return "5m"
	}
}

// YahooRange returns the Yahoo range parameter.
func (r Range) YahooRange() string {
	switch r {
	case Range5D:
		return "5d"
	case Range1Mo:
		return "1mo"
	case Range6Mo:
		return "6mo"
	case Range1Y:
		return "1y"
	default:
		return "1d"
	}
}

// Prev returns the shorter range, or self if already shortest.
func (r Range) Prev() Range {
	for i, v := range AllRanges {
		if v == r && i > 0 {
			return AllRanges[i-1]
		}
	}
	return r
}

// Next returns the longer range, or self if already longest.
func (r Range) Next() Range {
	for i, v := range AllRanges {
		if v == r && i < len(AllRanges)-1 {
			return AllRanges[i+1]
		}
	}
	return r
}
