# kabuto

A terminal dashboard for global markets — stock indices, futures, forex, crypto and commodities at a glance.

(Originally a CLI clone of sekai-kabuka.com.)

## Install

```bash
go install github.com/kzcat/kabuto/cmd/kabuto@latest
```

## Build

```bash
git clone https://github.com/kzcat/kabuto.git
cd kabuto
go build -o kabuto ./cmd/kabuto
```

## Usage

```bash
# Show every section once
./kabuto

# Japan section only
./kabuto -s japan

# Multiple sections
./kabuto -s japan -s us

# Auto-refresh every 30 seconds
./kabuto -w 30

# JSON output
./kabuto -j

# No colors (ASCII box drawing)
./kabuto --no-color

# Display times in a specific timezone
./kabuto --tz America/New_York

# Override the detected home market country
./kabuto --country JP

# Version
./kabuto -v
```

## Options

- `-s, --section NAME` — Show only these sections (repeatable).
- `-w, --watch SECONDS` — Auto-refresh every SECONDS seconds.
- `--rg` — Use red=up / green=down (Japanese convention).
- `-j, --json` — Output JSON instead of the dashboard.
- `--no-color` — Disable colors and use ASCII box drawing.
- `--tz NAME` — Display times in the given IANA timezone (e.g. `Asia/Tokyo`).
- `--country ISO2` — Override the detected home market country (e.g. `JP`, `US`, `DE`).
- `-v, --version` — Print version and exit.

The home market section is auto-detected from your `$LC_ALL` / `$LANG` locale
(falling back to `US`) and moved to the front. Times use your local timezone
unless `--tz` is given.

## Sections

`japan` / `us` / `us-futures` / `europe` / `asia` / `mideast-america` / `forex` / `crypto` / `commodity`

## Test

```bash
go test ./...
```
