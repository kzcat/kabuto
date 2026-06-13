# kabuto 競合調査 (RESEARCH)

- 調査日: 2026-06-13
- 対象: ターミナル型グローバル市場ダッシュボード「kabuto」(Go, 標準ライブラリのみ, Yahoo Finance 非公式エンドポイント)
- 調査方法: GitHub API / 各リポジトリの README (raw) を取得して比較

## 参照 URL

- https://github.com/achannarasappa/ticker
- https://raw.githubusercontent.com/achannarasappa/ticker/master/README.md
- https://github.com/achannarasappa/ticker-static/blob/master/symbols.csv
- https://github.com/tarkah/tickrs
- https://raw.githubusercontent.com/tarkah/tickrs/master/README.md
- https://github.com/mop-tracker/mop
- https://raw.githubusercontent.com/mop-tracker/mop/master/README.md
- https://github.com/andriy-git/stocksTUI
- https://raw.githubusercontent.com/andriy-git/stocksTUI/main/README.md
- https://github.com/ni5arga/stock-tui
- https://raw.githubusercontent.com/ni5arga/stock-tui/main/README.md
- https://github.com/igoropaniuk/stonks-cli
- https://raw.githubusercontent.com/igoropaniuk/stonks-cli/master/README.md
- https://github.com/cointop-sh/cointop
- https://github.com/dakma-dev/gloomberg
- https://github.com/muk2/feedtui
- https://github.com/stxkxs/mkt
- https://github.com/query2.finance.yahoo.com (Yahoo Finance /v8/finance/chart 非公式エンドポイント)
- https://terminaltrove.com/categories/finance/
- https://github.com/rothgar/awesome-tuis
- https://docs.brew.sh/Acceptable-Formulae (Homebrew core 採用要件)

> 注: gloomberg は当初の `0xAlcibiades/gloomberg` がリネーム/移動済みで現存確認できず。NFT/web3 系の最も近い現存実装として `dakma-dev/gloomberg` (Go, NFT tx ライブ監視, PoC/WIP) を採用。cointop は crypto 専用の歴史的代表として参考掲載 (2024-04 アーカイブ済)。

---

## 1. 比較表 (tools x aspects)

| Tool | 言語 | データソース | APIキー | チャート | Watchlist/Portfolio | 配布 | License | Stars | 最終更新 | 際立つ特徴 |
|---|---|---|---|---|---|---|---|---|---|---|
| **kabuto** | Go (stdlib) | Yahoo 非公式 (query2→query1) | 不要 | Braille エリア (truecolor) | 固定51銘柄/なし | GoReleaser, brew tap, scoop, go install | MIT | (新規) | 開発中 | 全銘柄を 1 グリッドにタイル化, 7言語 i18n, 自国市場自動先頭化 |
| ticker | Go | Yahoo(既定)+Coinbase | 不要 | sparkline | watchlist+ロット別取得原価P/L | brew/winget/snap/macports/docker | GPL-3.0 | ~6,100 | 2026-05 | コストベーシス対応の本格ポートフォリオ |
| tickrs | Rust | Yahoo | 不要 | line/candle/kagi | watchlist/なし | cargo/brew | MIT | ~1,640 | 2026-05 | candle/kagi + pre/post 市場 + 複数時間軸 |
| mop | Go | Yahoo | 不要 | なし(表中心) | watchlist/取得原価P/L表示 | go install/source | MIT | ~2,190 | 2025-12 | 式ベースのリアルタイムフィルタ + 列ソート |
| stocksTUI | Python (Textual) | yfinance(Yahoo) | 不要 | 履歴チャート 1D〜Max | タグ付watchlist/なし | pipx/pip | GPL-3.0 | ~150 | 2026-03 | ニュース/ATH/PER/時価総額, 市場対応キャッシュ |
| stock-tui (ni5arga) | Go | Yahoo + CoinGecko | 不要(無料枠) | line/area/candle | 設定watchlist/なし | go install | MIT | ~52 | 2026-01 | プロバイダ切替, Vim 風操作 |
| stonks-cli | Python | yfinance(Yahoo) | 不要 | candle/OHLC | マルチportfolio+P&L | pip | MIT | ~49 | 2026-05 | バックテスト + AI チャット + ニュース |
| cointop | Go | CoinGecko 等 | 不要 | なし | crypto portfolio | brew/snap/source | Apache-2.0 | ~4,400 | 2024-04 (archived) | crypto 専用の高速 TUI 老舗 |
| gloomberg (dakma) | Go | Ethereum チェーン | RPC/key 要 | なし | なし | source | MIT | ~30 | 2024-03 | NFT tx ライブストリーム (web3) |

---

## 2. Features (資産クラス/カスタム銘柄/P&L/検索ソート/アラート/時間外/履歴)

**kabuto の位置づけ**

- 強み: 9 セクション・51銘柄で株指数/為替/暗号資産/商品/先物を**最初から横断網羅**。競合の多くは銘柄ゼロから手動登録が前提で、kabuto は「起動即グローバル俯瞰」が成立する数少ない設計。
- 弱み: 固定銘柄リストでカスタム銘柄不可。ポートフォリオ P/L なし、検索/ソート/フィルタなし、価格アラートなし、pre/post 市場の明示処理なし、時間軸が 5m intraday 固定。ticker(ロット別原価)・mop(式フィルタ)・stonks-cli(P&L+バックテスト)・tickrs(candle/kagi/時間外)に機能面で見劣り。

| 改善案 | 優先度 | 工数 | 差別化寄与 |
|---|---|---|---|
| カスタム銘柄の追加 (フラグ/設定ファイルで任意シンボル追記、固定リストは既定維持) | High | M | partly |
| 複数履歴/時間軸対応 (1D/5D/1M/1Y、range/interval をクエリ化) | High | M | no |
| ポートフォリオ P/L (数量・取得原価入力、ticker 互換 YAML) | Med | L | no |
| 価格アラート (閾値超で watch モード内通知/ベル) | Med | M | partly |
| pre/post 市場の明示表示 (Yahoo の prePost フラグ活用) | Med | S | no |
| watch モード内検索/ソート/フィルタ | Low | M | no |

## 3. UI/UX (レイアウト/チャート/テーマ/キーバインド/マウス/ヘルプ/レスポンシブ)

**kabuto の位置づけ**

- 強み: 全銘柄を **N×M 単一タイルグリッド**に詰める独自レイアウトは競合に類例なし(表形式の mop/ticker、リスト+単一チャートの tickrs/stock-tui と差別化)。Braille エリアチャート+前日終値赤基準線+±1%ガイド+高安ラベル+空セルの時計タイル、幅と高さ両方からの列数自動最適化、truecolor グラデーション、SIGWINCH 追従、フリッカーレス単一書込み、`?`/`h` ヘルプオーバーレイは UX 完成度が高い。
- 弱み: マウス未対応 (tickrs はクリック操作あり)。candle/kagi など複数チャート種なし。オンボーディング(初回ガイド)・デモ GIF なし。色テーマは固定 (`--rg` 反転のみ) で stocksTUI のテーマ機構に劣る。

| 改善案 | 優先度 | 工数 | 差別化寄与 |
|---|---|---|---|
| README にデモ GIF/asciinema 追加 (タイルグリッドの訴求力が現状伝わらない) | High | S | yes |
| カラーテーマ機構 (組込み数種 + 環境変数選択) | Med | M | partly |
| candle チャート種の追加 (Braille で OHLC 描画) | Med | M | partly |
| watch モードのマウスサポート (タイルクリックで拡大) | Low | M | no |
| 初回起動オンボーディング/キーヒント常時バー | Low | S | no |

## 4. Internationalization / global readiness (言語/数値通貨書式/市場網羅/FX建値/TZ/RTL)

**kabuto の位置づけ**

- 強み: **7言語 i18n (en/ja/zh-Hans/ko/de/fr/es) は競合中で突出**。ほぼ全競合が英語 UI のみ。`LANG` 国コードからの自国市場セクション自動先頭化、IANA TZ 対応、JP ロケール時 BTC を JPY 建てにする等、グローバル設計が明確な差別化軸。中東・アジア市場までカバー。
- 弱み: RTL 言語 (アラビア語/ヘブライ語) UI 非対応 (市場はカバーするが UI は LTR)。ロケール準拠の数値/通貨書式 (桁区切り・通貨記号) が明示実装か不明確。翻訳は名称中心でメッセージ全体の網羅度は要確認。

| 改善案 | 優先度 | 工数 | 差別化寄与 |
|---|---|---|---|
| ロケール準拠の数値/通貨書式 (桁区切り・小数・通貨記号) | High | M | yes |
| 言語追加 (pt-BR, hi, ar など新興市場圏) | Med | M | yes |
| RTL UI 対応 (アラビア語/ヘブライ語の行方向反転) | Low | L | yes |
| FX 建値の任意指定 (`--base-currency`) | Med | S | partly |

## 5. Configuration / extensibility (設定ファイル/プラグイン/データソース切替/APIキー)

**kabuto の位置づけ**

- 強み: 設定不要で即起動 (ゼロコンフィグ)。フラグ体系 (`--lang`/`--country`/`--tz`/列数/JSON 等) は整理されている。
- 弱み: **設定ファイルが一切なし**が最大の弱点。ticker(YAML)、stocksTUI(テーマ/タブ設定)、feedtui(config.toml)、stonks-cli(マルチ YAML)、mop(.moprc) はいずれも永続設定を持つ。プラグイン機構なし、データソース切替なし (stock-tui は Yahoo/CoinGecko 切替、ticker は Coinbase 併用)、API キー対応なし。

| 改善案 | 優先度 | 工数 | 差別化寄与 |
|---|---|---|---|
| 設定ファイル対応 (`~/.config/kabuto/config.toml`: 銘柄/言語/列数/色) | High | M | no |
| データソース抽象化 (interface 化し将来の代替ソース差替えを容易に) | High | M | partly |
| 代替ソース実装 (Stooq/CoinGecko 等を `--source` で切替) | Med | L | partly |
| API キー対応 (任意の有料/公式ソース用、環境変数読込) | Low | M | no |

## 6. Data source robustness (Yahoo 非公式リスク/レート制限/複数ソース/キャッシュ/失敗時挙動)

**kabuto の位置づけ**

- 強み: query2→query1 フォールバック、goroutine 並列+セマフォ ~8、失敗銘柄は N/A 表示で全体停止せず。429 回避のための bare `Mozilla/5.0` UA など実運用ノウハウが反映済み。
- 弱み: **単一データソース (Yahoo 非公式) への依存が脆弱性リスク**。エンドポイント仕様変更/恒久 429/ブロックで全停止しうる。レスポンスのローカルキャッシュなし (stocksTUI は市場対応キャッシュで週末/休日の無駄リクエスト回避)。代替ソースなし (ticker/stock-tui は複数ソース)。

| 改善案 | 優先度 | 工数 | 差別化寄与 |
|---|---|---|---|
| ローカルキャッシュ (短 TTL + 市場休場判定で無駄取得抑制) | High | M | partly |
| 第2データソース (Yahoo 障害時フォールバック先) | High | L | partly |
| レート制限のバックオフ/リトライ強化 (429 で指数バックオフ) | Med | S | no |
| 取得失敗時の前回値表示+鮮度タイムスタンプ | Med | S | no |

## 7. Distribution / discoverability (パッケージ/Homebrew core/バイナリサイズ/起動速度/掲載)

**kabuto の位置づけ**

- 強み: GoReleaser で 4 OS×2 arch、checksums/changelog、GitHub Actions タグ自動リリース、brew tap(kzcat/tap)、Scoop、go install。**単一静的バイナリ ~8.6 MB / 依存ゼロ**は Python 系 (stocksTUI/stonks-cli) より導入が圧倒的に容易で起動も速い。
- 弱み: Homebrew **core** 未掲載 (tap のみ)。Terminal Trove / awesome-tuis 未掲載で発見性が低い。AUR/Nix は計画段階。ticker は brew/winget/snap/macports/docker と配布チャネルが広い。

| 改善案 | 優先度 | 工数 | 差別化寄与 |
|---|---|---|---|
| Terminal Trove / awesome-tuis への掲載申請 | High | S | no |
| AUR / Nix パッケージ公開 (計画の実行) | Med | M | no |
| winget マニフェスト追加 (Windows ユーザー獲得) | Med | S | no |
| Homebrew core 申請 (stars/notability 要件充足後) | Low | M | no |

## 8. Quality / platform support (Windows/端末互換/アクセシビリティ/NO_COLOR/SIXEL)

**kabuto の位置づけ**

- 強み: Windows/Linux/macOS × amd64/arm64 をビルド配布。`--no-color` で ASCII ボックス描画フォールバック、`COLORTERM=truecolor` 検出。raw termios を syscall で直接扱い依存ゼロ。
- 弱み: **`NO_COLOR` 環境変数規約 (no-color.org) 未対応** (`--no-color` フラグのみ)。SIXEL/画像なし。Braille は等幅フォント依存で一部端末/フォントで崩れる懸念。アクセシビリティ (スクリーンリーダー/高コントラストモード) 配慮なし。テスト/CI の網羅度は要確認。

| 改善案 | 優先度 | 工数 | 差別化寄与 |
|---|---|---|---|
| `NO_COLOR` 環境変数規約サポート (業界標準、低コスト) | High | S | no |
| Windows ターミナル/各エミュレータでの Braille 描画検証 + フォント注意書き | Med | S | no |
| アクセシビリティ: 高コントラスト/カラーブラインド対応テーマ | Med | M | partly |
| テスト/CI 強化 (描画・パース・i18n のユニットテスト, バッジ表示) | Med | M | no |
| SIXEL チャート (対応端末で高精細描画) | Low | L | partly |

---

## 9. グローバル普及のための優先ロードマップ (TOP 改善)

優先度の高い順。差別化を伸ばす項目と、競合に対する致命的欠落を埋める項目を併記。

| # | 改善 | 優先度 | 工数 | 狙い |
|---|---|---|---|---|
| 1 | デモ GIF/asciinema を README に追加 | High | S | 唯一無二のタイルグリッド UI を可視化、導入の最大障壁を除去 |
| 2 | カスタム銘柄 + 設定ファイル (TOML) | High | M | 「固定リスト」という普及最大のブロッカーを解消 |
| 3 | データソース抽象化 + 第2ソース/キャッシュ | High | L | 単一 Yahoo 依存の脆弱性を緩和、信頼性訴求 |
| 4 | ロケール準拠の数値/通貨書式 | High | M | 7言語 i18n という差別化軸を「真にグローバル」へ完成 |
| 5 | `NO_COLOR` 規約 + Terminal Trove/awesome-tuis 掲載 | High | S | 低コストで規約準拠と発見性を同時改善 |
| 6 | 複数履歴/時間軸 (1D〜1Y) | Med | M | 競合標準機能へのキャッチアップ |
| 7 | カラーテーマ機構 | Med | M | UX 完成度向上、カラーブラインド対応の足場 |
| 8 | AUR/Nix/winget 配布拡充 | Med | M | プラットフォーム横断の到達範囲拡大 |

---

## 付録: kabuto の総括

- 最大の差別化: (a) 全銘柄を単一タイルグリッドに詰める俯瞰 UI、(b) 7言語 i18n + 自国市場自動先頭化、(c) 依存ゼロの軽量単一バイナリ。
- 最大のリスク: (a) 固定銘柄でカスタム不可、(b) 単一 Yahoo 非公式ソース依存、(c) 設定ファイル不在。
- 戦略: 「グローバル俯瞰 × 多言語 × ゼロ依存」を核に、カスタム銘柄・設定ファイル・データソース冗長化で競合の基本機能ギャップを埋めれば、ニッチで明確なポジションを確立できる。
