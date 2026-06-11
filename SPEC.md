# sekaino-kabuka CLI 仕様書

https://sekai-kabuka.com (世界の株価) の CLI 版。
世界の株価指数・為替・暗号資産・商品先物をターミナルに一覧表示する。

## 方針

- **言語**: Python 3.11+、**外部依存なし**(標準ライブラリのみ。urllib / json / argparse / unittest)
- **データソース**: Yahoo Finance 公開エンドポイント(API キー不要)
  - `https://query1.finance.yahoo.com/v8/finance/chart/{symbol}?interval=1d&range=2d`
  - レスポンスの `chart.result[0].meta` から `regularMarketPrice` / `chartPreviousClose` / `regularMarketTime` を取得
  - User-Agent ヘッダ必須(ブラウザ風の UA を付けること。付けないと 429/403 になる)
  - 取得は `concurrent.futures.ThreadPoolExecutor` で並列化(max_workers=8 程度)
  - 取得失敗した銘柄は `N/A` 表示で続行(全体を落とさない)

## 表示銘柄(セクション順も本サイトに準拠)

| セクション | 銘柄 (Yahoo シンボル) |
|---|---|
| 日本 | 日経平均 `^N225` / 日経先物(CME) `NKD=F` / ドル円 `USDJPY=X` |
| 米国 | NYダウ `^DJI` / S&P500 `^GSPC` / NASDAQ `^IXIC` / SOX `^SOX` / VIX `^VIX` / 米10年債利回り `^TNX` |
| 米国先物 | ダウ先物 `YM=F` / S&P先物 `ES=F` / NASDAQ先物 `NQ=F` |
| 欧州 | FTSE100 `^FTSE` / DAX `^GDAXI` / CAC40 `^FCHI` / ユーロストックス50 `^STOXX50E` |
| アジア | 香港ハンセン `^HSI` / 上海総合 `000001.SS` / 台湾加権 `^TWII` / 韓国KOSPI `^KS11` / インドSENSEX `^BSESN` / 豪ASX200 `^AXJO` |
| 為替 | ユーロ円 `EURJPY=X` / ユーロドル `EURUSD=X` / ポンド円 `GBPJPY=X` / 豪ドル円 `AUDJPY=X` |
| 暗号資産 | BTC円 `BTC-JPY` / BTCドル `BTC-USD` / ETHドル `ETH-USD` |
| 商品 | NY金 `GC=F` / NY原油WTI `CL=F` / NY銀 `SI=F` / 天然ガス `NG=F` |

※ドル円は「日本」セクションに含め、「為替」には重複させない。

## 表示

- セクションごとに見出し付きのテーブル。列: `名称 / 現在値 / 前日比 / 前日比% / 更新時刻(JST)`
- ANSI カラー: 上昇=緑、下落=赤、変わらず=デフォルト色
- `--no-color` または非 TTY(パイプ時)では色なし
- 数値フォーマット: 3桁カンマ区切り、小数は銘柄の桁に合わせ 2〜4 桁。前日比には符号(+/-)を付ける

## CLI インターフェース

```
sekai-kabuka [options]
  (引数なし)            全セクションを1回表示して終了
  -s, --section NAME    指定セクションのみ表示(japan/us/us-futures/europe/asia/forex/crypto/commodity。複数指定可)
  -w, --watch [SEC]     自動更新モード(デフォルト 30 秒間隔、Ctrl-C で終了。画面クリアして再描画)
  -j, --json            JSON で出力(色なし)
  --no-color            色なし
  -v, --version         バージョン表示
```

## ファイル構成

```
sekaino-kabuka/
├── SPEC.md
├── README.md            # 使い方・スクリーンショット(テキスト例)・インストール方法
├── pyproject.toml       # [project.scripts] sekai-kabuka = "sekai_kabuka.cli:main"
├── src/sekai_kabuka/
│   ├── __init__.py      # __version__
│   ├── cli.py           # argparse・メインループ
│   ├── fetcher.py       # Yahoo Finance 取得(並列)
│   ├── symbols.py       # 上記銘柄定義(セクション・表示名・シンボル・小数桁)
│   └── render.py        # テーブル描画・ANSI カラー・JSON 出力
└── tests/
    ├── test_render.py   # フォーマット・色分けのテスト(ネットワーク不要)
    └── test_fetcher.py  # レスポンスパースのテスト(固定 JSON フィクスチャ使用、ネットワーク不要)
```

## 品質要件

- テストはネットワークアクセスなしで `python3 -m pytest` または `python3 -m unittest` で通ること
- `python3 -m sekai_kabuka` でも起動できること(`__main__.py`)
- タイムアウト 10 秒、リトライ 1 回
