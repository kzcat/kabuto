package symbols

// Item defines a single market symbol.
type Item struct {
	Name     string
	Symbol   string
	Decimals int
	Country  string // ISO country code (JP/US/...). Empty for crypto, etc.
}

// Section defines a market section.
type Section struct {
	Key   string
	Title string
	Items []Item
}

// SectionOrder defines the display order of sections.
var SectionOrder = []string{"japan", "us", "us-futures", "europe", "asia", "mideast-america", "forex", "crypto", "commodity"}

// RegisterSection registers or merges a section into Sections.
// For existing keys, items are appended (duplicates by Symbol skipped).
// For new keys, the section is added. SectionOrder is not modified.
func RegisterSection(sec Section) {
	if existing, ok := Sections[sec.Key]; ok {
		seen := map[string]bool{}
		for _, it := range existing.Items {
			seen[it.Symbol] = true
		}
		for _, it := range sec.Items {
			if !seen[it.Symbol] {
				existing.Items = append(existing.Items, it)
				seen[it.Symbol] = true
			}
		}
		Sections[sec.Key] = existing
	} else {
		Sections[sec.Key] = sec
	}
}

// Sections holds all section definitions.
var Sections = map[string]Section{
	"japan": {Key: "japan", Title: "Japan", Items: []Item{
		{"Nikkei 225", "^N225", 2, "JP"},
		{"Nikkei 225 Futures (CME)", "NKD=F", 2, "JP"},
		{"TOPIX (ETF)", "1306.T", 2, "JP"},
		{"Growth 250 (ETF)", "2516.T", 2, "JP"},
		{"USD/JPY", "USDJPY=X", 3, "JP"},
	}},
	"us": {Key: "us", Title: "US", Items: []Item{
		{"Dow Jones", "^DJI", 2, "US"},
		{"S&P 500", "^GSPC", 2, "US"},
		{"NASDAQ", "^IXIC", 2, "US"},
		{"SOX", "^SOX", 2, "US"},
		{"FANG+", "^NYFANG", 2, "US"},
		{"VIX", "^VIX", 2, "US"},
		{"US 10Y Yield", "^TNX", 3, "US"},
	}},
	"us-futures": {Key: "us-futures", Title: "US Futures", Items: []Item{
		{"Dow Futures", "YM=F", 2, "US"},
		{"S&P Futures", "ES=F", 2, "US"},
		{"NASDAQ Futures", "NQ=F", 2, "US"},
	}},
	"europe": {Key: "europe", Title: "Europe", Items: []Item{
		{"FTSE 100", "^FTSE", 2, "GB"},
		{"DAX", "^GDAXI", 2, "DE"},
		{"CAC 40", "^FCHI", 2, "FR"},
		{"Euro Stoxx 50", "^STOXX50E", 2, "EU"},
		{"Swiss SMI", "^SSMI", 2, "CH"},
		{"FTSE MIB", "FTSEMIB.MI", 2, "IT"},
		{"MOEX", "IMOEX.ME", 2, "RU"},
	}},
	"asia": {Key: "asia", Title: "Asia", Items: []Item{
		{"Hang Seng", "^HSI", 2, "HK"},
		{"Shanghai Composite", "000001.SS", 2, "CN"},
		{"TAIEX", "^TWII", 2, "TW"},
		{"KOSPI", "^KS11", 2, "KR"},
		{"SENSEX", "^BSESN", 2, "IN"},
		{"Nifty 50", "^NSEI", 2, "IN"},
		{"STI", "^STI", 2, "SG"},
		{"KLCI", "^KLSE", 2, "MY"},
		{"JKSE", "^JKSE", 2, "ID"},
		{"SET Index", "^SET.BK", 2, "TH"},
		{"ASX 200", "^AXJO", 2, "AU"},
		{"NZX 50", "^NZ50", 2, "NZ"},
	}},
	"mideast-america": {Key: "mideast-america", Title: "Mid-East & Americas", Items: []Item{
		{"BIST 100", "XU100.IS", 2, "TR"},
		{"TA-35", "TA35.TA", 2, "IL"},
		{"TASI", "^TASI.SR", 2, "SA"},
		{"S&P/TSX", "^GSPTSE", 2, "CA"},
		{"IPC Mexico", "^MXX", 2, "MX"},
		{"Bovespa", "^BVSP", 2, "BR"},
	}},
	"forex": {Key: "forex", Title: "Forex", Items: []Item{
		{"EUR/JPY", "EURJPY=X", 3, "EU"},
		{"EUR/USD", "EURUSD=X", 4, "EU"},
		{"GBP/JPY", "GBPJPY=X", 3, "GB"},
		{"AUD/JPY", "AUDJPY=X", 3, "AU"},
	}},
	"crypto": {Key: "crypto", Title: "Crypto", Items: []Item{
		{"BTC/JPY", "BTC-JPY", 2, ""},
		{"BTC/USD", "BTC-USD", 2, ""},
		{"ETH/USD", "ETH-USD", 2, ""},
	}},
	"commodity": {Key: "commodity", Title: "Commodities", Items: []Item{
		{"Gold", "GC=F", 2, "US"},
		{"Crude Oil WTI", "CL=F", 2, "US"},
		{"Silver", "SI=F", 3, "US"},
		{"Natural Gas", "NG=F", 3, "US"},
	}},
}
