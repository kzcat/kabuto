# kabuto 仕様書

**kabuto** — 世界の株価指数・為替・暗号資産・商品先物をターミナルに一覧表示するグローバル市場ダッシュボード。
当初 https://sekai-kabuka.com (世界の株価) の CLI クローンとして出発し、独自アプリ `kabuto` に発展。
名称由来: 兜町(東京の金融街=日本のウォール街)+ 株(kabu)。

- **module path**: `github.com/kzcat/kabuto`(git ユーザー kzcat 前提。リポジトリ未作成なら作成後にこのパスへ)
- **コマンド名 / バイナリ名**: `kabuto`(`cmd/kabuto/main.go`)
- **UI chrome(ヘッダー・ヘルプ・バージョン・曜日など装飾文言)は英語**。銘柄名などデータラベルは現状の表記を維持
  - ヘッダー例: `kabuto ─ global markets    Updated: 2026-06-13 01:00:29 +09:00`
  - 旧 `世界の株価 ─ sekai-kabuka CLI` 表記・旧 `sekai-kabuka` 文字列は全て置換する

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

| セクション | 銘柄 (Yahoo シンボル) [国コード] |
|---|---|
| 日本 | 日経平均 `^N225` [JP] / 日経先物(CME) `NKD=F` [JP] / TOPIX(ETF) `1306.T` [JP] / グロース250(ETF) `2516.T` [JP] / ドル円 `USDJPY=X` [JP] |
| 米国 | NYダウ `^DJI` / S&P500 `^GSPC` / NASDAQ `^IXIC` / SOX `^SOX` / FANG+ `^NYFANG` / VIX `^VIX` / 米10年債利回り `^TNX`(すべて [US]) |
| 米国先物 | ダウ先物 `YM=F` / S&P先物 `ES=F` / NASDAQ先物 `NQ=F`(すべて [US]) |
| 欧州 | FTSE100 `^FTSE` [GB] / DAX `^GDAXI` [DE] / CAC40 `^FCHI` [FR] / ユーロストックス50 `^STOXX50E` [EU] / スイスSMI `^SSMI` [CH] / イタリアMIB `FTSEMIB.MI` [IT] / ロシアMOEX `IMOEX.ME` [RU] |
| アジア | 香港ハンセン `^HSI` [HK] / 上海総合 `000001.SS` [CN] / 台湾加権 `^TWII` [TW] / 韓国KOSPI `^KS11` [KR] / インドSENSEX `^BSESN` [IN] / インドNifty `^NSEI` [IN] / シンガポールSTI `^STI` [SG] / マレーシアKLCI `^KLSE` [MY] / インドネシアJKSE `^JKSE` [ID] / タイSET `^SET.BK` [TH] / 豪ASX200 `^AXJO` [AU] / NZ50 `^NZ50` [NZ] |
| 中東・米州 | トルコBIST100 `XU100.IS` [TR] / イスラエルTA35 `TA35.TA` [IL] / サウジTASI `^TASI.SR` [SA] / カナダTSX `^GSPTSE` [CA] / メキシコIPC `^MXX` [MX] / ブラジルBOVESPA `^BVSP` [BR] |
| 為替 | ユーロ円 `EURJPY=X` [EU] / ユーロドル `EURUSD=X` [EU] / ポンド円 `GBPJPY=X` [GB] / 豪ドル円 `AUDJPY=X` [AU] |
| 暗号資産 | BTC円 `BTC-JPY` / BTCドル `BTC-USD` / ETHドル `ETH-USD`(国コードなし=省略) |
| 商品 | NY金 `GC=F` / NY原油WTI `CL=F` / NY銀 `SI=F` / 天然ガス `NG=F`(すべて [US]) |

※ドル円は「日本」セクションに含め、「為替」には重複させない。
※国コードは銘柄定義(symbols.go)に持たせる。空文字なら表示しない。
※**新規追加シンボルは実装時に必ずライブで取得検証**し、データが返らないものは削除して報告すること(Yahoo に存在しない/廃止された場合があるため)。セクション `-s` の指定名に `mideast-america`(中東・米州)を追加。

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

### グリッド配置(本家の「全画面表示モード」を再現)

本家の全画面モードはチャートタイルが画面全体に隙間なく敷き詰められる。CLI でも端末を使い切る:

- **余白の最小化**: タイル間の水平・垂直ギャップは 0(枠線同士が隣接)。ヘッダー直後の空行も入れない
- **全銘柄を1つの N×M グリッドに連続配置**: セクションごとに段を分けると幅の広い端末で右側が空くため、**セクション見出し行は廃止**し、表示対象の全銘柄(`-s` 指定時はその部分集合)を定義順のまま1つのグリッドに行送りで敷き詰める(最終行のみ欠けてよい)。所属が分かるよう、各タイルの**上枠線の右端に bright black でセクション名**を埋め込む: `┌─ 日経平均 ────── 日本 ─┐`(タイル幅が 30 桁未満で入り切らない場合はセクション名を省略)
- **列数は幅と高さの両方から最適化**: 列数を幅だけで決める(termWidth/24)と、縦に長い端末でチャート行数が上限に達して画面下部が余る。そこで stdout が TTY のときは列数 C を全探索で選ぶ:
  - 候補: `C ∈ [1, min(表示銘柄数, termWidth/最小タイル幅24)]`
  - 各 C について `段数 = ceil(銘柄数/C)`、`タイル高さ = 利用可能行数(termRows−ヘッダー1)/段数`、`チャート行数 N = clamp(タイル高さ−3, 1, 12)`、`使用行数 = 段数×(3+N) + 配分できる余り行` を計算
  - **使用行数が最大の C を採用**。同点なら C が大きい方(タイルが細かく並ぶ方)を採用
  - 例: 33銘柄・300桁×90行 → C=12(N上限到達で45行)ではなく C=6(6段×タイル高さ約15行=89行)が選ばれ、タイルが大きくなって画面全体が埋まる
  - 非 TTY 時は従来どおり幅のみで決定(C = max(1, termWidth/24)、銘柄数上限、N=2)
- `タイル幅 = termWidth / 列数`(余り桁は左の列から 1 桁ずつ配分して合計が端末幅に一致するように)。スパークラインや名称はタイル幅に追従して伸びる
- **タイルの圧縮**: タイル内は「情報行 1 行 + チャート行 N 行」に圧縮する
  - 上枠線に国コードと銘柄名を埋め込む: `┌─ [JP]日経平均 ─────┐`(国コードは bright black の2文字 `[JP][US][GB][DE][FR][HK][CN]...`。絵文字の国旗は端末の幅計算が不安定なため使わない)
  - **情報の優先順位は本家に合わせる**(騰落率が主役):
    - タイル内 上段 = 騰落率バッジ `▲+2.81%` を左寄せで大きく見せる(騰落色背景 + 太字白)
    - タイル内 下段(チャートの下、下枠線の上)= `現在値(太字) 前日比(騰落色)`: `│ 66,020.04  +1,802.77 │`
    - これによりタイル最小高さは 5 行(枠2 + %行 + チャート1 + 値行)。チャート行数 N = タイル高さ − 4 に変更
  - チャート行 N 行: **点字エリアチャート**(下記)
  - 前日比% は**バッジ表示**: 騰落色の背景 + 太字白文字で ` ▲+0.06% `(前後に空白1)。下落は赤背景・変わらずは bright black 背景。`--rg` で背景色も反転。`--no-color` 時は従来どおり色なしテキスト

### 点字エリアチャート(btop/gping 方式)

ブロック文字(▁▂▃▄▅▆▇█、8段階/セル)に代えて、点字文字(U+2800〜U+28FF、1セル=横2×縦4ドット)で高解像度のエリアチャートを描く:

- チャート領域がセル幅 W × セル高 N のとき、解像度は **横 2W 点 × 縦 4N 段階**。系列は 2W 点にダウンサンプル(null 補間済みの series を使用)
- 各 x 点の値を 0〜4N-1 に量子化し、**その高さから下のドットをすべて立てる**(= 面で塗るエリアチャート。線だけにしない)
- 点字の組み立て: セル内ドットのビット配置は Unicode 標準(dot1..8 = 左列上から 0x01,0x02,0x04,0x40 / 右列上から 0x08,0x10,0x20,0x80)。全ドット消灯のセルは空白ではなく U+2800 を使ってもよいが、表示幅は 1 として扱う
- **前日終値の基準線(本家の赤い水平線)**: チャートの縦スケールは「当日系列の min/max と前日終値の両方」を含めて取り、前日終値の高さに**赤の水平破線**(1セルおき・ドット行)を重ねる。チャート本体のドットと重なった位置はチャート本体を優先。これにより前日終値より上で取引されているか下かが一目で分かる
- **±1% ガイドライン**: チャート高さ N ≥ 4 かつ前日終値±1% がスケール範囲に入る場合のみ、bright black の点線(2セルおき)で重ねる(基準線より控えめに)
- **高値・安値ラベル**: タイル幅 ≥ 30 のとき、チャート領域の右端 ~9 桁をラベル用に確保し、右上に当日高値・右下に当日安値を bright black で表示(系列から計算)。チャート本体の幅はその分縮める。タイル幅 < 30 ではラベルなし(現行どおり全幅チャート)
- **閉場市場のグレー表現(本家のグレータイル)**: `regularMarketTime` が現在時刻より 30 分以上古い銘柄は閉場とみなし、チャート・基準線を bright black 単色で描く(バッジと数値の色はそのまま)。fetcher は market time の epoch 秒を結果に含めること
- **時計タイルは廃止**: グリッド最終行の空きセルには何も描かない(空白のまま)。以前は空きセルに時計タイルを置いていたが不要のため削除した
- **色**: チャート全体を騰落色とし、truecolor 対応時(環境変数 `COLORTERM` が `truecolor` または `24bit`)は行ごとに上→下へ明→暗のグラデーション(`ESC[38;2;r;g;bm`、最上行を基準色、最下行を約50%の暗さに線形補間)。truecolor 非対応時は単色(現行どおり ESC[31/32m)
- `--no-color` 時は色なしの点字のみ。非 TTY(ASCII 罫線モード)でもチャートは点字のまま(罫線だけ ASCII)
- 既存の `SparklineRows`(ブロック方式)は削除し、テストも点字方式(`BrailleRows` 等)に置き換える
- **高さも使い切る(stdout が TTY なら常時)**: 端末の行数(ioctl で取得)から ヘッダー1行 + セクション見出し行数 を引いた残りを、タイルの段数で割ってタイル高さを決め、チャート行数 N = タイル高さ − 3(上下枠 + 情報行)とする(N の下限 1、上限 12)。端末が小さく収まらない場合は N=1 まで圧縮し、それでも溢れる分はそのまま流す(スクロール)
  - **余り行の配分**: 均等割りで余った行数は、上の段から順に 1 段につき 1 行ずつチャート行 N に加算し、最終行が画面下端に届くようにする(余白行を残さない。ただし N 上限 12 とタイル段数の制約で配り切れない分は残してよい)
  - この高さ計算は watch / 非 watch(1回表示)共通。非 watch では画面制御(代替スクリーン・カーソル移動)は行わず、フルハイトの内容をそのまま出力するだけ
- 非 TTY(パイプ・リダイレクト)時のみ高さ計算をせず N=2 固定。幅の規則は同じ
- 最上部にヘッダー行: `世界の株価 ─ sekai-kabuka CLI    更新: 2026-06-12 00:44:08 JST`(反転表示・端末幅まで反転を伸ばす)
- 全角文字の表示幅(=2)を考慮して桁揃え(既存実装を踏襲)

### 端末リサイズ追従(watch 時)

- `SIGWINCH` を捕捉し、リサイズ時は**再取得せず**直近データで即座に再描画する(列数・タイル幅・チャート高さを再計算)
- 毎フレームの描画前にも端末サイズを取り直す

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

---

## アプリ化 (kabuto) — 機能拡張

### 1. ヘルプ・UI chrome の英語化

- `flag.Usage` を独自定義し、ヘルプ全文を英語に。各フラグ説明も英語。
  - 例: `-s, --section NAME   Show only these sections (repeatable: japan,us,...)`
- `-v/--version` 出力: `kabuto <version>`
- ヘッダー行・時計タイルの曜日(`Sat` など英略)・エラーメッセージも英語
- 銘柄名(日経平均など)はデータラベルなので変更しない

### 2. ロケール / TZ による出し分け

- **TZ**: 更新時刻の JST ハードコードを廃止し、`time.Local`(=`$TZ` 反映)で表示。タイムゾーンオフセット表記(例 `+09:00`)を併記。`--tz <IANA名>` で上書き可
- **国検出**: `$LC_ALL` → `$LANG` の順に読み、`xx_YY` の `YY`(国コード)を抽出(例 `ja_JP.UTF-8` → `JP`)。取れなければ `US` を既定
- **ホーム市場優先**: 検出国に対応するセクションを先頭へ並べ替える(例 JP→japan 先頭、US→us 先頭、GB/DE/FR→europe 先頭)。並べ替えのみで銘柄自体は全件維持
- **BTC 建値**: 円圏(JP)は BTC-JPY を主表示、それ以外は BTC-USD を主表示(暗号資産セクションの並び順を入れ替えるだけ)
- `--country <ISO2>` で国を明示上書き可。フラグ > 環境変数 > 既定(US)の優先順位
- 非対話・JSON 出力時も TZ はローカル準拠にする

### 3. ショートカットキー(対話的 watch モード)

- **依存方針: 純標準ライブラリのみ**。raw mode は既存の ioctl 実装(`ioctl_unix.go`)同様 `syscall` で termios を操作(TCGETS/TCSETS 相当、`unix` build tag)。非 Unix 向けは no-op フォールバック(`ioctl_other.go` と同方針)
- watch モード(`-w`)で stdin を raw + 非カノニカルにし、別 goroutine でキー読み取り。終了時(`q`/`Ctrl-C`/SIGINT/SIGTERM)に termios と画面(alt-screen・カーソル)を必ず復元
- **Ctrl-C**: raw mode は ISIG を無効化するため Ctrl-C は SIGINT を発生させずバイト 0x03 として届く。キーディスパッチで 0x03 を終了として扱う(ヘルプ表示中でも終了)。終了処理は他の終了経路と同一
- 既存のちらつき防止(alt-screen + 1バッファ書き込み)・SIGWINCH 追従を壊さないこと
- キー割り当て:

  | キー | 動作 | キー | 動作 |
  |------|------|------|------|
  | `q` / `Esc` / `Ctrl-C` | 終了 | `1`-`9` | 該当セクションのみ表示(トグル) |
  | `r` | 即時再取得 | `0` / `a` | 全セクション表示 |
  | `c` | 色モード循環(通常→日本式→色なし) | `f` | フルハイト ON/OFF |
  | `+` / `-` | 列数を手動増減(自動に戻すキー `=`) | `?` / `h` | ヘルプ・オーバーレイ表示 |
  | `Space` | 自動更新の一時停止/再開 | | |

- 非 TTY(パイプ)時はキー処理を行わず従来どおり1回出力
- ヘルプ・オーバーレイは画面中央にキー一覧を枠表示。任意キーで閉じる

### 4. 配布

- **GoReleaser** (`.goreleaser.yaml`): darwin/linux/windows × amd64/arm64 のバイナリ、チェックサム、changelog
- **GitHub Actions** (`.github/workflows/release.yml`): タグ push で goreleaser 実行
- **Homebrew tap**: goreleaser の `brews:`(別リポ `kzcat/homebrew-tap` を想定)。`brew install kzcat/tap/kabuto`
- **go install**: `go install github.com/kzcat/kabuto/cmd/kabuto@latest`(公開モジュール化が前提)
- **Scoop / AUR / Nix**: goreleaser の `scoops:` を設定。AUR・Nix は README に手順を記載(後追い)
- README に各インストール方法を英語で記載
- **注意**: 実際の GitHub リポジトリ作成・タグ push・初回リリースは外部公開アクションのためユーザー確認後に実施。設定ファイル一式の用意までを本作業の範囲とする

### 品質要件(更新)

- `go vet ./...` クリーン / `go test ./...` 全パス(ネットワーク不要)
- `go build -o kabuto ./cmd/kabuto` で単一バイナリ
- 旧 `sekai-kabuka` の文字列・ディレクトリ・module path が残っていないこと(`grep -rn sekai-kabuka` がヒットしない。README の沿革説明を除く)

### 5. 多言語対応 (i18n)

収録言語: **en, ja, zh(簡体), ko, de, fr, es** の7言語。**en をベース/フォールバック**とする。

- **新規パッケージ `internal/i18n`**:
  - 言語検出: `$LC_ALL` → `$LANG` → `$LANGUAGE` の順に読み、先頭の言語サブタグを抽出(`ja_JP.UTF-8`→`ja`、`zh_CN`→`zh`、`zh_Hans`→`zh`)。収録外/未取得は `en`
  - `--lang <code>` フラグで上書き(優先順位: フラグ > 環境変数 > 既定 en)。**言語(ラベル)と国(並び順, --country)は独立**
  - UI 文字列カタログ: `map[lang]map[key]string`。キーが無ければ en にフォールバック、en にも無ければキー名をそのまま返す
  - 銘柄名は **Yahoo シンボルをキー**に、セクション名は **Section.Key をキー**に、それぞれ言語別訳を引く(どちらも en フォールバック)
- **symbols.go の変更**: `Item.Name` / `Section.Title` の基準値を**英語の国際標準名**にする(例: `Nikkei 225` `Dow Jones` `S&P 500` `NASDAQ` `Hang Seng` `Shanghai Composite` `TAIEX` `KOSPI` `Gold` `Crude Oil WTI` 等)。日本語名(日経平均 等)は i18n カタログの `ja` に移す
- **翻訳方針**:
  - セクション名(Japan/US/US Futures/Europe/Asia/Mid-East & Americas/Forex/Crypto/Commodities)・商品名(Gold/Crude Oil WTI/Silver/Natural Gas)・記述的語(futures, 10Y yield 等)は各言語へ自然に翻訳する
  - 指数の固有名詞(NASDAQ, DAX, CAC 40, FTSE 100, S&P 500 等)は各言語の慣習に従い、ラテン文字のままでよい言語はそのまま(例: 独仏西では DAX/NASDAQ はそのまま、CJK では現地慣習名: 日「ナスダック/日経平均」、中「纳斯达克/日经平均」、韓「나스닥/닛케이」等)
  - **データラベル(セクション名・銘柄名・商品名)・ヘッダー語(global markets)・時計タイトル/曜日・キーオーバーレイのラベル**は7言語すべて翻訳必須
  - `--help` のフラグ詳細説明は en/ja を整備し、他言語は en フォールバック可(プロローグ翻訳の負荷を抑えるため。スコープ外の手抜きではなく明示的な許容)
- **render / JSON**: 表示名は解決後の言語名を使う。`--json` の `name` も `--lang` で解決した名前にする(`symbol` キーは不変なので機械可読性は保たれる)
- **回帰防止**: 既存機能(描画・キー操作・ロケール並べ替え・配布)を壊さない。`go vet`/`go test` が通り続けること
- **テスト**: 言語検出(各種 LANG 値→lang)、`--lang` 優先順位、カタログのフォールバック(未収録キー→en→キー名)、シンボル/セクション名解決を純粋関数でテスト(ネットワーク不要)

---

## グローバル普及対応(改善実装) — RESEARCH.md ロードマップの実装仕様

依存ゼロ(Go 標準ライブラリのみ)の原則は維持する。外部 TOML ライブラリ等は追加しない。

### G1. デモ(README に GIF/asciinema)
- charmbracelet/vhs のスクリプト `demo/demo.tape` を用意し、ダッシュボード表示→`c`色循環→`1`絞り込み→`?`ヘルプ→`q`終了 を見せる
- vhs があれば `demo/kabuto.gif` を生成、無ければ生成手順を README に明記
- README 冒頭に GIF を埋め込む

### G2. カスタム銘柄 + 設定ファイル
- 設定ファイル(**JSON、stdlib の encoding/json のみ**。TOML は外部依存になるため不採用): 既定 `~/.config/kabuto/config.json`。`--config PATH` で上書き
- スキーマ例:
  ```json
  {
    "lang": "ja", "country": "JP", "theme": "default", "range": "1d", "source": "auto",
    "sections": [{"key":"watch","title":"Watchlist","items":[{"name":"Tesla","symbol":"TSLA","country":"US","decimals":2}]}],
    "section_order": ["watch","japan","us"]
  }
  ```
- カスタムセクションは組み込みセクションとマージ。`section_order` で表示順制御
- `--add SYMBOL[:COUNTRY[:DECIMALS]]`(repeatable)で一時的に "Watchlist" セクションへ銘柄追加
- 優先順位: CLI フラグ > 設定ファイル > 環境変数 > 既定。設定ファイルが無くても従来どおり動く(後方互換)

### G3. データソース抽象化 + 第2ソース + キャッシュ
- `Source` インターフェース化。`YahooSource`(現行 query2→query1)を実装1とし、**APIキー不要の第2ソース**(Stooq の CSV `https://stooq.com/q/l/` 等)を実装2に
- `--source auto|yahoo|stooq`(既定 auto = Yahoo 優先、失敗時 stooq フォールバック)
- ディスクキャッシュ `~/.cache/kabuto/`(symbol+range キー、TTL 短め例 60s)。レート制限/障害時はキャッシュで凌ぐ
- テストはネットワーク不要(httptest + フィクスチャ)を維持

### G4. ロケール準拠の数値/通貨書式
- 桁区切り・小数点を言語/ロケールに合わせる(en `1,234.56` / de `1.234,56` / fr `1 234,56` 等)。i18n に数値書式ルールを持たせる
- 通貨記号/表記: fetcher が `meta.currency` を取得済みなのでそれを使い、価格に通貨を反映(例 `¥66,020` `$51,373` `€8,345`)。バッジ/前日比は従来どおり
- 桁揃え(全角=2)を壊さないこと

### G5. NO_COLOR 規約 + 発見性
- 環境変数 `NO_COLOR`(値の有無に関わらず存在すれば)で色無効化(https://no-color.org 準拠)。`--no-color` と同等。優先: `--no-color` フラグ > `NO_COLOR` env > 既定
- `docs/SUBMISSIONS.md` に Terminal Trove / awesome-tuis への掲載ドラフト文を用意(実際の投稿は公開後)

### G6. 履歴/時間軸
- `--range 1d|5d|1mo|6mo|1y`(既定 1d)。Yahoo の range/interval にマップ(1d→5m, 5d→15m, 1mo→1d, 6mo→1d, 1y→1wk 等)
- 対話キーで時間軸切替(`[` 短く / `]` 長く、またはキー表に追記)。チャートは既存の点字エリアに系列を流すだけ

### G7. カラーテーマ
- 色定数(green/red/RGB 等)を Theme 構造体に集約。`--theme default|mono|light|highcontrast`(既定 default)+ 設定ファイル
- カラーブラインド配慮の `highcontrast`/`mono` を用意。`--rg`(日本式反転)はテーマと直交で維持

### G8. 配布拡充(AUR/Nix/winget)
- GoReleaser に winget(`winget:` または publisher 連携)、AUR(`aurs:` で PKGBUILD 自動生成、`kzcat/kabuto-bin` AUR リポジトリ想定)、Nix(`flake.nix` をリポジトリに用意)を追加
- README の Installation に各手順を追記

### CLI フラグ最終形(英語ヘルプに反映)
```
--add SYMBOL[:CC[:DEC]]  Add ad-hoc symbol to Watchlist (repeatable)
--config PATH            Config file (default ~/.config/kabuto/config.json)
--source auto|yahoo|stooq Data source (default auto)
--range 1d|5d|1mo|6mo|1y History range (default 1d)
--theme NAME             Color theme (default|mono|light|highcontrast)
```

## B. btop 由来 UI リッチ化 (v0.3.0)

`cmd/kabuto/main.go` の `version` を `"0.3.0"` に更新する。
ファイル競合回避のため以下のフェーズで順次実装する(各フェーズ = 1 回の kiro 委譲):

- **Phase 1**(render プリミティブ・対話状態なし): B1, B2, B3, B9
- **Phase 2a**(選択/ナビ): B5, B8, B10
- **Phase 2b**(入力/履歴): B6, B7
- **Phase 3**(テーマ): B4

すべて **標準ライブラリのみ**。既存の全角=2 桁揃え・点字チャート・truecolor グラデーション・NO_COLOR/テーマ/i18n を壊さないこと。

### B1. グラフシンボル切替 (braille / block / tty)
- `render.Options` に `GraphSymbol string`(`"auto"|"braille"|"block"|"tty"`、既定 `auto`)を追加。`--graph` フラグ(`main.go`)で指定。
- `auto`: ASCII フォールバック時(`opt.NoColor` で ascii box のとき)または非 UTF-8 ロケール時は `tty`、それ以外は `braille`。
- `braille`: 現状の点字エリアチャート(変更なし)。
- `block`: 8 段ブロック要素 `▁▂▃▄▅▆▇█` で各列の高さを表現(フォント非依存)。チャートの行数 `rows` を満たすよう、full 行は `█`、最上段のみ部分ブロック。グラデーション色は維持。
- `tty`: ASCII のみ(例 `#`/`|`/`.` 等)。色はテーマ単色で可。
- `buildChartLines` をシンボルモードで分岐させる。high/low ラベル・前日比ベースライン・±1% ガイドラインは可能な範囲で維持(tty は簡略可)。

### B2. ボックス四隅/枠ラベル
- 各タイルの枠に high/low と当日レンジ幅(%)を埋め込む。`buildTopBorderW`(上枠)に加え下枠 or 右寄せで `H:<high> L:<low>` を表示。
- 桁揃え(全角=2)・既存の name/secName 表示を壊さない。市場閉場(grey)時も整合。

### B3. カラー深度フォールバック (truecolor → 256 → 16)
- カラー深度検出関数を追加: truecolor(`COLORTERM=truecolor|24bit`)→ 256(`TERM` に `256` を含む)→ 16(それ以外)。
- `fg24`/`gradientRGB` 系を、深度に応じて 256 色(`ESC[38;5;Nm`、RGB→6x6x6 キューブ近似)/ 16 色(最近傍 ANSI)へ劣化させる。truecolor 環境の見た目は不変。
- `--graph`/テーマ/`--rg` と直交。テスト可能なよう純関数(RGB→256 index, RGB→16 index)を切り出す。

### B9. グラデーション水平メーターバー
- 騰落率の大きさを btop 風メーター `█████░░░` で色付き表示するヘルパー `meterBar(pct, width, th, useColor, redGreen, depth) string` を追加。
- タイル内の価格/前日比の近くに小さく配置(レイアウト幅を圧迫しない。狭いときは省略可)。色は B3 の深度に追従。

### B5. フォーカス詳細ビュー
- `UIState` に `Focus int`(-1 = 無効)を追加。`Enter`(`key.R == 13`)で現在の(または先頭の)銘柄にフォーカス。`←↑→↓` 矢印(B6 の ESC シーケンス解析後でも、まずは `j/k`/`n/p` 等の代替キーで可)で銘柄移動。`Esc` は**フォーカス中は詳細を閉じるだけで quit しない**(フォーカス無効時のみ従来どおり quit)。
- フォーカス中は全幅の大きい点字チャート + 統計(銘柄名・現在値・前日比・前日終値・当日 high/low・レンジ幅・range ラベル)を表示する `render.RenderDetail(...)` を追加。
- `Dispatch` の Esc/quit 分岐を Focus 状態で条件分けする。

### B8. レイアウトプリセット
- `UIState` に `Preset int` を追加。キー `p` で循環: `all`(全セクション)→ `majors`(japan/us/europe 等の主要株価指数)→ `fxcrypto`(forex+crypto)→ all。
- プリセットは `st.Sections` を設定する形で実装(既存の 1-9 トグルと整合)。`0`/`a` で all に戻るのは従来どおり。

### B10. フォーカス枠ハイライト
- B5 の Focus 中のタイルの枠を `Theme.Bold`/明色で強調(視線誘導)。グリッド表示時に有効。
- 非フォーカス時は従来どおり。color 無効時はハイライト無効 or 太字のみ。

### B6. 矢印キー対応(マウスは廃止)
- 当初 SGR マウス(クリック/ホイール)も実装したが、**ユーザー判断で廃止**(2026-06-25)。クリック→タイルの座標近似が端末サイズ依存で不正確だったのも理由。
- **矢印キーのみ採用**: `readKeys` が `ESC[A/B/C/D` を解析して `Key{Up/Down/Right/Left}` を返し、`Dispatch` で 上/左→前の銘柄、下/右→次の銘柄(`n`/`b` と同等)に割り当てる。
- マウス有効化シーケンス・SGR 解析・`tileIndexAt`・`Key` のマウス用フィールドは削除済み。

### B7. ローリング履歴バッファ
- watch ループで銘柄ごとに直近値のローリング系列を保持し、リフレッシュ間でチャートが左へ滑らかに流れるようにする。
- `runWatch` 内に `map[symbol][]float64`(上限長 = チャート幅相当)を持ち、各リフレッシュで最新値を push。チャート描画はこの系列を優先使用(初回は従来の intraday 系列で初期化)。
- 1d 以外の range では従来の API 系列をそのまま使う(履歴蓄積は 1d/watch のみ)。

### B4. ビルトインテーマ拡充(ファイルテーマは廃止)
- 当初 `~/.config/kabuto/themes/*.json` のロード可能テーマを実装したが、**ユーザー判断でファイル(config)方式は廃止**(2026-06-25)。ローダ(`theme_file.go`/`LoadTheme`/`themesDir`/`rgbToANSI`)は削除済み。
- 代わりに `internal/render/theme.go` の**ビルトインテーマを拡充**: 既存 `default`/`mono`/`light`/`highcontrast` に加え、定番パレット `dracula`/`nord`/`gruvbox`/`solarized` を追加(計8種)。
- 各テーマは `UpColor`/`DownColor`(16色エスケープ・変動テキスト用)と `UpRGB`/`DownRGB`(チャートのグラデーション色、B3 の深度で 256/16 に劣化)を持つ。`--theme <name>` は `ThemeByName` でビルトインのみ解決、未知名は default。
- `--theme` のヘルプ(usage/flag)と README の Options 表・Themes 節を更新。
