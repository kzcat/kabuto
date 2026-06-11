"""銘柄定義: (表示名, Yahooシンボル, 小数桁数)"""

SECTIONS = {
    "japan": {
        "title": "日本",
        "items": [
            ("日経平均", "^N225", 2),
            ("日経先物(CME)", "NKD=F", 2),
            ("ドル円", "USDJPY=X", 3),
        ],
    },
    "us": {
        "title": "米国",
        "items": [
            ("NYダウ", "^DJI", 2),
            ("S&P500", "^GSPC", 2),
            ("NASDAQ", "^IXIC", 2),
            ("SOX", "^SOX", 2),
            ("VIX", "^VIX", 2),
            ("米10年債利回り", "^TNX", 3),
        ],
    },
    "us-futures": {
        "title": "米国先物",
        "items": [
            ("ダウ先物", "YM=F", 2),
            ("S&P先物", "ES=F", 2),
            ("NASDAQ先物", "NQ=F", 2),
        ],
    },
    "europe": {
        "title": "欧州",
        "items": [
            ("FTSE100", "^FTSE", 2),
            ("DAX", "^GDAXI", 2),
            ("CAC40", "^FCHI", 2),
            ("ユーロストックス50", "^STOXX50E", 2),
        ],
    },
    "asia": {
        "title": "アジア",
        "items": [
            ("香港ハンセン", "^HSI", 2),
            ("上海総合", "000001.SS", 2),
            ("台湾加権", "^TWII", 2),
            ("韓国KOSPI", "^KS11", 2),
            ("インドSENSEX", "^BSESN", 2),
            ("豪ASX200", "^AXJO", 2),
        ],
    },
    "forex": {
        "title": "為替",
        "items": [
            ("ユーロ円", "EURJPY=X", 3),
            ("ユーロドル", "EURUSD=X", 4),
            ("ポンド円", "GBPJPY=X", 3),
            ("豪ドル円", "AUDJPY=X", 3),
        ],
    },
    "crypto": {
        "title": "暗号資産",
        "items": [
            ("BTC円", "BTC-JPY", 2),
            ("BTCドル", "BTC-USD", 2),
            ("ETHドル", "ETH-USD", 2),
        ],
    },
    "commodity": {
        "title": "商品",
        "items": [
            ("NY金", "GC=F", 2),
            ("NY原油WTI", "CL=F", 2),
            ("NY銀", "SI=F", 3),
            ("天然ガス", "NG=F", 3),
        ],
    },
}

SECTION_ORDER = ["japan", "us", "us-futures", "europe", "asia", "forex", "crypto", "commodity"]
