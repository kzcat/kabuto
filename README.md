# kabuto

A terminal dashboard for global markets — stock indices, futures, forex, crypto and commodities at a glance.

![kabuto screenshot](demo/screenshot.png)

> Regenerate with [vhs](https://github.com/charmbracelet/vhs): `vhs demo/screenshot.tape` (still image) or `vhs demo/demo.tape` (GIF)

(Originally a CLI clone of sekai-kabuka.com.)

## Installation

### Homebrew

```bash
brew install kzcat/tap/kabuto
```

### AUR (Arch Linux)

```bash
yay -S kabuto-bin
```

### Nix

```bash
nix run github:kzcat/kabuto
```

### winget (Windows)

```powershell
winget install kzcat.kabuto
```

### Scoop (Windows)

```powershell
scoop bucket add kzcat https://github.com/kzcat/scoop-bucket
scoop install kabuto
```

### go install

```bash
go install github.com/kzcat/kabuto/cmd/kabuto@latest
```

### GitHub Releases

Download the latest binary for your platform from the
[Releases](https://github.com/kzcat/kabuto/releases) page, extract it,
and place it somewhere in your `$PATH`.

## Usage

```bash
# Show every section once
kabuto

# Japan section only
kabuto -s japan

# Multiple sections
kabuto -s japan -s us

# Auto-refresh every 30 seconds (interactive watch mode)
kabuto -w 30

# JSON output
kabuto -j

# No colors (ASCII box drawing)
kabuto --no-color

# Red=up / green=down (Japanese convention)
kabuto --rg

# Display times in a specific timezone
kabuto --tz America/New_York

# Override the detected home market country
kabuto --country JP

# Add ad-hoc symbols to watchlist
kabuto --add AAPL --add TSLA:US:2

# Use a specific data source
kabuto --source stooq

# Select history range
kabuto --range 1mo

# Use a color theme
kabuto --theme mono

# Custom config file
kabuto --config ~/my-kabuto.json

# Version
kabuto -v
```

## Options

| Flag | Description |
|------|-------------|
| `-s, --section NAME` | Show only these sections (repeatable) |
| `-w, --watch SECONDS` | Auto-refresh every N seconds |
| `--rg` | Red=up / green=down (Japanese convention) |
| `-j, --json` | Output JSON |
| `--no-color` | Disable colors, use ASCII box drawing |
| `--tz NAME` | IANA timezone for displayed times |
| `--country ISO2` | Override home market country (e.g. JP, US, DE) |
| `--add SYMBOL[:CC[:DEC]]` | Add ad-hoc symbol to Watchlist (repeatable) |
| `--config PATH` | Config file (default `~/.config/kabuto/config.json`) |
| `--source auto\|yahoo\|stooq` | Data source (default `auto`) |
| `--range 1d\|5d\|1mo\|6mo\|1y` | History range (default `1d`) |
| `--theme NAME` | Color theme (`default`\|`mono`\|`light`\|`highcontrast`\|`dracula`\|`nord`\|`gruvbox`\|`solarized`) |
| `-v, --version` | Print version and exit |

The home market section is auto-detected from `$LC_ALL` / `$LANG`
(falling back to US) and moved to the front.

## Data sources and cache

`auto` tries Yahoo first, then Stooq for supported indices. The current Stooq
integration provides a quote only: the previous close and percentage change are
shown as `N/A`, and charts show `History N/A` for every range. Use `--source yahoo`
for history. Missing or invalid source prices are treated as failed fetches.

Successful results are cached for 60 seconds, separately for each symbol, range,
and source policy. If a refresh fails, an expired quote may be displayed with a
`STALE` marker. Its original retrieval time is preserved, and subsequent refreshes
retry the source. The detail view shows the actual source and retrieval time.

JSON items include `source`, `fetched_at` (RFC 3339), `stale`, `prev_close`, and
`history_available` alongside the existing fields. Unavailable changes and market
timestamps are `null`; unsupported history is an empty `series` with
`history_available: false`. `fetched_at` is the successful retrieval time;
`epoch` is the market timestamp, when available.

## Custom sections

Sections from a config file can be selected with `-s` after registration:

```json
{
  "sections": [
    {
      "key": "stocks",
      "title": "My Stocks",
      "items": [{"symbol": "AAPL", "country": "US", "decimals": 2}]
    }
  ]
}
```

```bash
kabuto --config ./config.json -s stocks
kabuto --add AAPL -s watch
```

Without `-s`, custom sections are included in the dashboard. `section_order` in
the config can set their position.

## Themes

Built-in color themes set the up/down colors and the chart gradient.
Pick one with `--theme <name>`:

`default` · `mono` · `light` · `highcontrast` · `dracula` · `nord` · `gruvbox` · `solarized`

`mono` and `highcontrast` are color-blind friendly; `--rg` (red=up) works on
top of any theme.

## Interactive Keys (watch mode)

| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `q` / `Esc` | Quit | `1`–`9` | Toggle section |
| `r` | Refetch now | `0` / `a` | Show all sections |
| `c` | Cycle color mode | `f` | Toggle full-height |
| `+` / `-` | Adjust columns | `?` / `h` | Help overlay |
| `=` | Auto columns | `Space` | Pause/resume |
| `n` / `Tab` | Select next item | `b` | Select previous item |
| `Enter` | Detail view (toggle) | `p` | Layout preset cycle |
| `Esc` | Back (detail → select → quit) | | |

Watch mode shows a loading frame immediately. `q`, `Esc`, or Ctrl+C can quit
while a fetch is pending; changing ranges with `[` / `]` cancels the old request
and starts the new one immediately.

## Sections

`japan` · `us` · `us-futures` · `europe` · `asia` · `mideast-america` · `forex` · `crypto` · `commodity`

## Build

```bash
git clone https://github.com/kzcat/kabuto.git
cd kabuto
go build -o kabuto ./cmd/kabuto
```

## Test

```bash
go test -race ./...
go vet ./...
```

## License

MIT
