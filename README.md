# 世界の株価 CLI (sekai-kabuka)

ターミナルで世界の株価指数・為替・暗号資産・商品先物を一覧表示するCLIツール。

## インストール

```bash
pip install -e .
```

## 使い方

```bash
# 全セクション表示
sekai-kabuka

# 日本と米国のみ
sekai-kabuka -s japan -s us

# 30秒間隔で自動更新
sekai-kabuka -w

# JSON出力
sekai-kabuka -j

# パッケージとして実行
python3 -m sekai_kabuka
```

## 表示例

```
更新: 2024-06-12 15:30:00 JST

[ 日本 ]
名称                   現在値       前日比    前日比%   時刻
-----------------------------------------------------------------
日経平均            39,500.50     +500.50    +1.28%  15:00
日経先物(CME)       39,600.00     +100.00    +0.25%  06:00
ドル円                155.123      +0.323    +0.21%  15:00
```

## オプション

| オプション | 説明 |
|---|---|
| `-s`, `--section` | 表示セクション(japan/us/us-futures/europe/asia/forex/crypto/commodity) |
| `-w`, `--watch [SEC]` | 自動更新(デフォルト30秒) |
| `-j`, `--json` | JSON出力 |
| `--no-color` | 色なし |
| `-v`, `--version` | バージョン表示 |

## 要件

- Python 3.11+
- 外部依存なし(標準ライブラリのみ)
