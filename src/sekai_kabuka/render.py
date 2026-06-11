"""テーブル描画・ANSI カラー・JSON 出力"""

import json
import sys
from datetime import datetime, timezone, timedelta

from sekai_kabuka.symbols import SECTIONS, SECTION_ORDER

JST = timezone(timedelta(hours=9))

_GREEN = "\033[32m"
_RED = "\033[31m"
_RESET = "\033[0m"
_BOLD = "\033[1m"


def _use_color(no_color: bool) -> bool:
    if no_color:
        return False
    return sys.stdout.isatty()


def _fmt_num(value: float, decimals: int) -> str:
    """3桁カンマ区切り、指定小数桁"""
    formatted = f"{value:,.{decimals}f}"
    return formatted


def _fmt_change(value: float, decimals: int) -> str:
    sign = "+" if value > 0 else ""
    return f"{sign}{value:,.{decimals}f}"


def _fmt_pct(value: float) -> str:
    sign = "+" if value > 0 else ""
    return f"{sign}{value:.2f}%"


def _colorize(text: str, change: float, use_color: bool) -> str:
    if not use_color:
        return text
    if change > 0:
        return f"{_GREEN}{text}{_RESET}"
    elif change < 0:
        return f"{_RED}{text}{_RESET}"
    return text


def render_table(data: dict, sections: list[str] | None, no_color: bool) -> str:
    """テーブル文字列を生成"""
    color = _use_color(no_color)
    keys = sections if sections else SECTION_ORDER
    lines = []
    now = datetime.now(JST).strftime("%Y-%m-%d %H:%M:%S JST")
    lines.append(f"更新: {now}")
    lines.append("")

    for sec_key in keys:
        sec = SECTIONS[sec_key]
        title = sec["title"]
        if color:
            lines.append(f"{_BOLD}[ {title} ]{_RESET}")
        else:
            lines.append(f"[ {title} ]")
        header = f"{'名称':<16} {'現在値':>14} {'前日比':>12} {'前日比%':>9} {'時刻':>6}"
        lines.append(header)
        lines.append("-" * 65)
        for name, symbol, decimals in sec["items"]:
            result = data.get(symbol)
            if result is None:
                lines.append(f"{name:<16} {'N/A':>14} {'N/A':>12} {'N/A':>9} {'N/A':>6}")
            else:
                price_s = _fmt_num(result["price"], decimals)
                change_s = _fmt_change(result["change"], decimals)
                pct_s = _fmt_pct(result["change_pct"])
                time_s = result["time"]
                change_val = result["change"]
                row = f"{name:<16} {price_s:>14} {change_s:>12} {pct_s:>9} {time_s:>6}"
                lines.append(_colorize(row, change_val, color))
        lines.append("")
    return "\n".join(lines)


def render_json(data: dict, sections: list[str] | None) -> str:
    """JSON出力"""
    keys = sections if sections else SECTION_ORDER
    output = {}
    for sec_key in keys:
        sec = SECTIONS[sec_key]
        items = []
        for name, symbol, decimals in sec["items"]:
            result = data.get(symbol)
            if result is None:
                items.append({"name": name, "symbol": symbol, "price": None, "change": None, "change_pct": None, "time": None})
            else:
                items.append({
                    "name": name,
                    "symbol": symbol,
                    "price": round(result["price"], decimals),
                    "change": round(result["change"], decimals),
                    "change_pct": round(result["change_pct"], 2),
                    "time": result["time"],
                })
        output[sec_key] = {"title": sec["title"], "items": items}
    return json.dumps(output, ensure_ascii=False, indent=2)
