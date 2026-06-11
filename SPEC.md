# sekaino-kabuka CLI 仕様書

https://sekai-kabuka.com (世界の株価) の CLI 版。
世界の株価指数・為替・暗号資産・商品先物をターミナルに一覧表示する。

## 方針

- **言語**: Go 1.22+、**外部依存なし**(標準ライブラリのみ。net/http / encoding/json / flag / sync)。単一バイナリ配布
- **データソース**: Yahoo Finance 公開エンドポイント(API キー不要)
  - `https://query2.finance.yahoo.com/v8/finance/chart/{symbol}?interval=1d&range=2d`(query2 優先、429 時は query1 にフォールバック)
  - レスポンスの `chart.result[0].meta` から `regularMarketPrice` / `chartPreviousClose` / `regularMarketTime` を取得
  - **User-Agent は素の `Mozilla/5.0` 固定**(検証済みの重要な知見: ブラウザ完全偽装 UA・curl UA・Go デフォルト UA は Yahoo 側で 429 になる。素の `Mozilla/5.0` のみ通る)
  - シンボルは URL エンコードすること(`^N225` → `%5EN225`、`NKD=F` → `NKD%3DF`)
  - 取得は goroutine で並列化(セマフォ等で同時 8 接続程度に制限)
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

## 表示(本家サイト風ダッシュボード UI)

本家 https://sekai-kabuka.com の UI(黒背景に銘柄タイルがグリッド状に並び、各タイルにミニチャートと騰落色付きの数値が表示される)を CLI で再現する。

### タイル

1銘柄 = 1タイル(罫線で囲んだボックス)。タイル内のレイアウト:

```text
┌─ 日経平均 ──────────────┐
│ 64,217.27        +0.06% │   ← 現在値(太字) と 前日比%(騰落色・▲/▼付き)
│ +38.00            15:45 │   ← 前日比(騰落色) と 取引所時刻
│ ▁▂▂▃▅▄▆▇█▇▆▅▆▇██▇▆▅▆▇█ │   ← 当日のスパークライン(Unicode ▁▂▃▄▅▆▇█、騰落色)
└─────────────────────────┘
```

- スパークラインは intraday の終値系列から生成。データ取得を `interval=5m&range=1d` に変更し、`meta` と同一レスポンスの `indicators.quote[0].close` 系列(null は前値で補間)を使う
- 騰落色: デフォルトは 上昇=緑 / 下落=赤。`--rg` で 上昇=赤 / 下落=緑 に反転(本家の「色を選ぶ」相当、日本式)
- タイル枠線・セクション見出しは暗めの色(bright black)、現在値は太字白

### グリッド配置

- セクション見出し(`■ 日本` など)の下に、そのセクションのタイルを横に並べる
- 列数は端末幅から自動計算(タイル幅 ~27 桁。80 桁端末で 2〜3 列、120 桁で 4 列程度)。`$COLUMNS`/ioctl で取得、取得不可なら 80 と仮定
- 最上部にヘッダー行: `世界の株価 ─ sekai-kabuka CLI    更新: 2026-06-12 00:44:08 JST`(反転表示)
- 全角文字の表示幅(=2)を考慮して桁揃え(既存実装を踏襲)

### 自動更新(-w)のちらつき解消 ★重要バグ修正

現状は画面クリア(クリア→再描画)のためちらつく。以下の方式に変更する:

- watch 開始時に代替スクリーンバッファへ切替(`ESC[?1049h`)+カーソル非表示(`ESC[?25l`)、終了時(Ctrl-C 含む)に必ず復元(`ESC[?1049l` `ESC[?25h`。signal.Notify で SIGINT/SIGTERM を捕捉)
- 再描画は **画面クリアせず** `ESC[H`(カーソルホーム)から全フレームを 1 つの文字列バッファに構築し、各行末に `ESC[K`(行末まで消去)、最後に `ESC[J`(画面末尾まで消去)を付けて **1 回の Write で出力** する
- データ取得中も前フレームを表示したままにする(取得完了後に一括差し替え)
- 非 watch(1回表示)は従来どおり通常スクリーンにそのまま出力

### その他

- `--no-color` または非 TTY(パイプ時)では色なし・罫線は ASCII(`+-|`)にフォールバック
- 数値フォーマット: 3桁カンマ区切り、小数は銘柄の桁に合わせ 2〜4 桁。前日比には符号(+/-)を付ける
- `--json` には sparkline 用の close 系列も `series` として含める

## CLI インターフェース

```
sekai-kabuka [options]
  (引数なし)            全セクションを1回表示して終了
  -s, --section NAME    指定セクションのみ表示(japan/us/us-futures/europe/asia/forex/crypto/commodity。複数指定可)
  -w, --watch SEC       自動更新モード(秒数指定必須・例 -w 30。0 で無効。Ctrl-C で終了。代替スクリーンでちらつきなしに再描画)
  --rg                  騰落色を日本式に反転(上昇=赤/下落=緑)
  -j, --json            JSON で出力(色なし)
  --no-color            色なし
  -v, --version         バージョン表示
```

## ファイル構成

```text
sekaino-kabuka/
├── SPEC.md
├── README.md                # 使い方・出力例・インストール方法(go install / go build)
├── go.mod                   # module github.com/kaz/sekai-kabuka
├── cmd/sekai-kabuka/
│   └── main.go              # flag パース・メインループ(watch 含む)
└── internal/
    ├── fetcher/
    │   ├── fetcher.go       # Yahoo Finance 取得(goroutine 並列、query2→query1)
    │   └── fetcher_test.go  # パースのテスト(httptest + 固定 JSON フィクスチャ、外部ネットワーク不要)
    ├── symbols/
    │   └── symbols.go       # 銘柄定義(セクション・表示名・シンボル・小数桁)
    └── render/
        ├── render.go        # テーブル描画・ANSI カラー・JSON 出力
        └── render_test.go   # フォーマット・色分けのテスト(ネットワーク不要)
```

## 品質要件

- テストは外部ネットワークアクセスなしで `go test ./...` で通ること(HTTP は `httptest.Server` でモック)
- `go vet ./...` がクリーンであること
- `go build -o sekai-kabuka ./cmd/sekai-kabuka` で単一バイナリが作れること
- HTTP タイムアウト 10 秒、リトライ 1 回(エンドポイントフォールバック込み)
- テーブル描画は日本語(全角)の表示幅を考慮して桁揃えすること(全角=2桁として計算)

## 備考(Python 版からの移行)

- 本リポジトリは当初 Python 実装だった。Go 移行に伴い `src/`, `tests/`, `pyproject.toml` は削除する
- CLI インターフェース・表示仕様・銘柄構成は Python 版と完全互換を保つ
