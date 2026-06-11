# sekai-kabuka

世界の株価 CLI — 世界の株価指数・為替・暗号資産・商品先物をターミナルに一覧表示

## インストール

```bash
go install github.com/kaz/sekai-kabuka/cmd/sekai-kabuka@latest
```

## ビルド

```bash
git clone https://github.com/kaz/sekai-kabuka.git
cd sekai-kabuka
go build -o sekai-kabuka ./cmd/sekai-kabuka
```

## 使い方

```bash
# 全セクション表示
./sekai-kabuka

# 日本セクションのみ
./sekai-kabuka -s japan

# 複数セクション
./sekai-kabuka -s japan -s us

# 自動更新(30秒間隔)
./sekai-kabuka -w 30

# JSON出力
./sekai-kabuka -j

# 色なし
./sekai-kabuka --no-color

# バージョン
./sekai-kabuka -v
```

## セクション名

`japan` / `us` / `us-futures` / `europe` / `asia` / `forex` / `crypto` / `commodity`

## テスト

```bash
go test ./...
```
