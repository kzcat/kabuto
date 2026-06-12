package symbols

// Item は1銘柄の定義
type Item struct {
	Name     string
	Symbol   string
	Decimals int
	Country  string // ISO国コード(JP/US/...)。暗号資産など省略時は空文字
}

// Section はセクション定義
type Section struct {
	Key   string
	Title string
	Items []Item
}

// SectionOrder はセクション表示順
var SectionOrder = []string{"japan", "us", "us-futures", "europe", "asia", "mideast-america", "forex", "crypto", "commodity"}

// Sections は全セクション定義
var Sections = map[string]Section{
	"japan": {Key: "japan", Title: "日本", Items: []Item{
		{"日経平均", "^N225", 2, "JP"},
		{"日経先物(CME)", "NKD=F", 2, "JP"},
		{"TOPIX(ETF)", "1306.T", 2, "JP"},
		{"グロース250(ETF)", "2516.T", 2, "JP"},
		{"ドル円", "USDJPY=X", 3, "JP"},
	}},
	"us": {Key: "us", Title: "米国", Items: []Item{
		{"NYダウ", "^DJI", 2, "US"},
		{"S&P500", "^GSPC", 2, "US"},
		{"NASDAQ", "^IXIC", 2, "US"},
		{"SOX", "^SOX", 2, "US"},
		{"FANG+", "^NYFANG", 2, "US"},
		{"VIX", "^VIX", 2, "US"},
		{"米10年債利回り", "^TNX", 3, "US"},
	}},
	"us-futures": {Key: "us-futures", Title: "米国先物", Items: []Item{
		{"ダウ先物", "YM=F", 2, "US"},
		{"S&P先物", "ES=F", 2, "US"},
		{"NASDAQ先物", "NQ=F", 2, "US"},
	}},
	"europe": {Key: "europe", Title: "欧州", Items: []Item{
		{"FTSE100", "^FTSE", 2, "GB"},
		{"DAX", "^GDAXI", 2, "DE"},
		{"CAC40", "^FCHI", 2, "FR"},
		{"ユーロストックス50", "^STOXX50E", 2, "EU"},
		{"スイスSMI", "^SSMI", 2, "CH"},
		{"イタリアMIB", "FTSEMIB.MI", 2, "IT"},
		{"ロシアMOEX", "IMOEX.ME", 2, "RU"},
	}},
	"asia": {Key: "asia", Title: "アジア", Items: []Item{
		{"香港ハンセン", "^HSI", 2, "HK"},
		{"上海総合", "000001.SS", 2, "CN"},
		{"台湾加権", "^TWII", 2, "TW"},
		{"韓国KOSPI", "^KS11", 2, "KR"},
		{"インドSENSEX", "^BSESN", 2, "IN"},
		{"インドNifty", "^NSEI", 2, "IN"},
		{"シンガポールSTI", "^STI", 2, "SG"},
		{"マレーシアKLCI", "^KLSE", 2, "MY"},
		{"インドネシアJKSE", "^JKSE", 2, "ID"},
		{"タイSET", "^SET.BK", 2, "TH"},
		{"豪ASX200", "^AXJO", 2, "AU"},
		{"NZ50", "^NZ50", 2, "NZ"},
	}},
	"mideast-america": {Key: "mideast-america", Title: "中東・米州", Items: []Item{
		{"トルコBIST100", "XU100.IS", 2, "TR"},
		{"イスラエルTA35", "TA35.TA", 2, "IL"},
		{"サウジTASI", "^TASI.SR", 2, "SA"},
		{"カナダTSX", "^GSPTSE", 2, "CA"},
		{"メキシコIPC", "^MXX", 2, "MX"},
		{"ブラジルBOVESPA", "^BVSP", 2, "BR"},
	}},
	"forex": {Key: "forex", Title: "為替", Items: []Item{
		{"ユーロ円", "EURJPY=X", 3, "EU"},
		{"ユーロドル", "EURUSD=X", 4, "EU"},
		{"ポンド円", "GBPJPY=X", 3, "GB"},
		{"豪ドル円", "AUDJPY=X", 3, "AU"},
	}},
	"crypto": {Key: "crypto", Title: "暗号資産", Items: []Item{
		{"BTC円", "BTC-JPY", 2, ""},
		{"BTCドル", "BTC-USD", 2, ""},
		{"ETHドル", "ETH-USD", 2, ""},
	}},
	"commodity": {Key: "commodity", Title: "商品", Items: []Item{
		{"NY金", "GC=F", 2, "US"},
		{"NY原油WTI", "CL=F", 2, "US"},
		{"NY銀", "SI=F", 3, "US"},
		{"天然ガス", "NG=F", 3, "US"},
	}},
}
