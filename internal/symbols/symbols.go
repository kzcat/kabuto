package symbols

// Item は1銘柄の定義
type Item struct {
	Name     string
	Symbol   string
	Decimals int
}

// Section はセクション定義
type Section struct {
	Key   string
	Title string
	Items []Item
}

// SectionOrder はセクション表示順
var SectionOrder = []string{"japan", "us", "us-futures", "europe", "asia", "forex", "crypto", "commodity"}

// Sections は全セクション定義
var Sections = map[string]Section{
	"japan": {Key: "japan", Title: "日本", Items: []Item{
		{"日経平均", "^N225", 2},
		{"日経先物(CME)", "NKD=F", 2},
		{"ドル円", "USDJPY=X", 3},
	}},
	"us": {Key: "us", Title: "米国", Items: []Item{
		{"NYダウ", "^DJI", 2},
		{"S&P500", "^GSPC", 2},
		{"NASDAQ", "^IXIC", 2},
		{"SOX", "^SOX", 2},
		{"VIX", "^VIX", 2},
		{"米10年債利回り", "^TNX", 3},
	}},
	"us-futures": {Key: "us-futures", Title: "米国先物", Items: []Item{
		{"ダウ先物", "YM=F", 2},
		{"S&P先物", "ES=F", 2},
		{"NASDAQ先物", "NQ=F", 2},
	}},
	"europe": {Key: "europe", Title: "欧州", Items: []Item{
		{"FTSE100", "^FTSE", 2},
		{"DAX", "^GDAXI", 2},
		{"CAC40", "^FCHI", 2},
		{"ユーロストックス50", "^STOXX50E", 2},
	}},
	"asia": {Key: "asia", Title: "アジア", Items: []Item{
		{"香港ハンセン", "^HSI", 2},
		{"上海総合", "000001.SS", 2},
		{"台湾加権", "^TWII", 2},
		{"韓国KOSPI", "^KS11", 2},
		{"インドSENSEX", "^BSESN", 2},
		{"豪ASX200", "^AXJO", 2},
	}},
	"forex": {Key: "forex", Title: "為替", Items: []Item{
		{"ユーロ円", "EURJPY=X", 3},
		{"ユーロドル", "EURUSD=X", 4},
		{"ポンド円", "GBPJPY=X", 3},
		{"豪ドル円", "AUDJPY=X", 3},
	}},
	"crypto": {Key: "crypto", Title: "暗号資産", Items: []Item{
		{"BTC円", "BTC-JPY", 2},
		{"BTCドル", "BTC-USD", 2},
		{"ETHドル", "ETH-USD", 2},
	}},
	"commodity": {Key: "commodity", Title: "商品", Items: []Item{
		{"NY金", "GC=F", 2},
		{"NY原油WTI", "CL=F", 2},
		{"NY銀", "SI=F", 3},
		{"天然ガス", "NG=F", 3},
	}},
}
