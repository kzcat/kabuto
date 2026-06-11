"""CLI エントリポイント"""

import argparse
import sys
import time

from sekai_kabuka import __version__
from sekai_kabuka.symbols import SECTIONS, SECTION_ORDER
from sekai_kabuka.fetcher import fetch_all
from sekai_kabuka.render import render_table, render_json


def _collect_symbols(sections: list[str] | None) -> list[str]:
    keys = sections if sections else SECTION_ORDER
    symbols = []
    for k in keys:
        for _, sym, _ in SECTIONS[k]["items"]:
            if sym not in symbols:
                symbols.append(sym)
    return symbols


def main():
    parser = argparse.ArgumentParser(prog="sekai-kabuka", description="世界の株価 CLI")
    parser.add_argument("-s", "--section", action="append", choices=list(SECTIONS.keys()), help="表示セクション(複数指定可)")
    parser.add_argument("-w", "--watch", nargs="?", const=30, type=int, metavar="SEC", help="自動更新(デフォルト30秒)")
    parser.add_argument("-j", "--json", action="store_true", help="JSON出力")
    parser.add_argument("--no-color", action="store_true", help="色なし")
    parser.add_argument("-v", "--version", action="version", version=f"%(prog)s {__version__}")
    args = parser.parse_args()

    sections = args.section
    no_color = args.no_color or args.json

    def run_once():
        symbols = _collect_symbols(sections)
        data = fetch_all(symbols)
        if args.json:
            print(render_json(data, sections))
        else:
            print(render_table(data, sections, no_color))

    if args.watch is not None:
        try:
            while True:
                sys.stdout.write("\033[2J\033[H")
                sys.stdout.flush()
                run_once()
                time.sleep(args.watch)
        except KeyboardInterrupt:
            pass
    else:
        run_once()
