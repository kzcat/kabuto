"""render のテスト(ネットワーク不要)"""

import unittest
import sys
import os

sys.path.insert(0, os.path.join(os.path.dirname(__file__), "..", "src"))

from sekai_kabuka.render import _fmt_num, _fmt_change, _fmt_pct, _colorize, render_table, render_json


class TestFormat(unittest.TestCase):
    def test_fmt_num_comma(self):
        self.assertEqual(_fmt_num(39500.5, 2), "39,500.50")
        self.assertEqual(_fmt_num(155.123, 3), "155.123")

    def test_fmt_change_positive(self):
        self.assertEqual(_fmt_change(500.5, 2), "+500.50")

    def test_fmt_change_negative(self):
        self.assertEqual(_fmt_change(-200.0, 2), "-200.00")

    def test_fmt_pct(self):
        self.assertEqual(_fmt_pct(1.5), "+1.50%")
        self.assertEqual(_fmt_pct(-0.5), "-0.50%")

    def test_colorize_no_color(self):
        self.assertEqual(_colorize("test", 1.0, False), "test")

    def test_colorize_green(self):
        result = _colorize("test", 1.0, True)
        self.assertIn("\033[32m", result)

    def test_colorize_red(self):
        result = _colorize("test", -1.0, True)
        self.assertIn("\033[31m", result)


class TestRenderTable(unittest.TestCase):
    def test_na_display(self):
        output = render_table({}, ["japan"], no_color=True)
        self.assertIn("N/A", output)
        self.assertIn("日本", output)

    def test_with_data(self):
        data = {
            "^N225": {"price": 39500.50, "prev_close": 39000.0, "change": 500.50, "change_pct": 1.28, "time": "15:00"},
            "NKD=F": {"price": 39600.0, "prev_close": 39500.0, "change": 100.0, "change_pct": 0.25, "time": "06:00"},
            "USDJPY=X": {"price": 155.123, "prev_close": 154.800, "change": 0.323, "change_pct": 0.21, "time": "15:00"},
        }
        output = render_table(data, ["japan"], no_color=True)
        self.assertIn("39,500.50", output)
        self.assertIn("155.123", output)


class TestRenderJson(unittest.TestCase):
    def test_json_output(self):
        data = {"^N225": {"price": 39500.5, "prev_close": 39000.0, "change": 500.5, "change_pct": 1.28, "time": "15:00"}}
        output = render_json(data, ["japan"])
        import json
        parsed = json.loads(output)
        self.assertIn("japan", parsed)
        self.assertEqual(parsed["japan"]["items"][0]["price"], 39500.5)

    def test_json_na(self):
        output = render_json({}, ["japan"])
        import json
        parsed = json.loads(output)
        self.assertIsNone(parsed["japan"]["items"][0]["price"])


if __name__ == "__main__":
    unittest.main()
