"""
أيقوناتُ تثبيت الموقع على الشاشة الرئيسيّة — وألوانُ البيان من التوكنز.

(قرارُ المالك ٢٠٢٦-١٠-١٧: زبونُ آيفون يضيف الموقعَ للشاشة فيفتح كأنّه تطبيق.)

تُولَّد ولا تُبدَّل باليد — كأيقونات أندرويد (`mobile/tools/make-icon.py`):

    python web/scripts/make-pwa.py            # يكتب
    python web/scripts/make-pwa.py --check    # يفحص ولا يكتب

المصادرُ ثلاثةٌ لا رابعَ لها:
  - الرمز:      mobile/brand/logo.png          (واحدٌ للمنظومة كلِّها)
  - خلفيّتُه:   mobile/brand/icon.json  ← background  (الزبونُ أبيض — قرارُ المالك ٢٠٢٦-١٠-٠١)
  - لونُ الشاشة: web/packages/ui/src/theme.css ← --color-shell   (أرضُ الموقع نفسُها)

والناتج:
  - web/apps/rahalgo/public/pwa/*.png
  - web/apps/rahalgo/src/lib/pwa.gen.json   (يقرؤه البيانُ ووسمُ لون الشريط)
"""

import io
import json
import re
import sys
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parents[2]
BRAND = ROOT / "mobile" / "brand"
THEME = ROOT / "web" / "packages" / "ui" / "src" / "theme.css"
OUT = ROOT / "web" / "apps" / "rahalgo" / "public" / "pwa"
GEN = ROOT / "web" / "apps" / "rahalgo" / "src" / "lib" / "pwa.gen.json"

# المقاسات: ١٩٢ و٥١٢ للبيان · ٥١٢ «قابلةٌ للقصّ» يقصّها أندرويد بشكله ·
# ١٨٠ لآيفون (apple-touch-icon) — وآيفون لا يقرأ البيانَ للأيقونة.
ICONS = [
    ("icon-192.png", 192, 0.78),
    ("icon-512.png", 512, 0.78),
    ("maskable-512.png", 512, None),  # None ⇒ نسبةُ الأمان من icon.json
    ("apple-touch-icon.png", 180, 0.74),
]


def hex_rgba(value: str) -> tuple:
    v = value.lstrip("#")
    return tuple(int(v[i:i + 2], 16) for i in (0, 2, 4)) + (255,)


def shell_color() -> str:
    css = THEME.read_text(encoding="utf-8")
    hit = re.search(r"--color-shell:\s*(#[0-9a-fA-F]{6})\s*;", css)
    if not hit:
        sys.exit("لم يوجد --color-shell في theme.css")
    return hit.group(1).lower()


def render(mark: Image.Image, size: int, fill: float, bg: tuple) -> bytes:
    canvas = Image.new("RGBA", (size, size), bg)
    m = mark.copy()
    target = max(1, int(size * fill))
    m.thumbnail((target, target), Image.LANCZOS)
    canvas.paste(m, ((size - m.width) // 2, (size - m.height) // 2), m)
    buf = io.BytesIO()
    # **وبلا شفافيّة** — آيفون يملأ الشفّافَ أسود.
    canvas.convert("RGB").save(buf, format="PNG", optimize=True)
    return buf.getvalue()


def main() -> int:
    check = "--check" in sys.argv
    cfg = json.loads((BRAND / "icon.json").read_text(encoding="utf-8"))
    mark = Image.open(BRAND / cfg["source"]).convert("RGBA")
    box = mark.getbbox()
    if box:
        mark = mark.crop(box)
    bg = hex_rgba(cfg["background"])
    safe = float(cfg["safe"])

    gen = {
        "_note": "مولَّدٌ بـ web/scripts/make-pwa.py — لا يُحرَّر باليد.",
        "theme": shell_color(),
        "background": shell_color(),
        "icons": [{"src": f"/pwa/{name}", "size": size} for name, size, _ in ICONS],
    }
    gen_text = json.dumps(gen, ensure_ascii=False, indent=2) + "\n"

    diff = 0
    OUT.mkdir(parents=True, exist_ok=True)
    for name, size, fill in ICONS:
        data = render(mark, size, fill if fill is not None else safe, bg)
        path = OUT / name
        same = path.exists() and Image.open(path).size == (size, size)
        if check:
            if not same:
                print(f"ناقص أو بمقاسٍ خطأ: {path}")
                diff += 1
            continue
        path.write_bytes(data)
    if check:
        if not GEN.exists() or GEN.read_text(encoding="utf-8") != gen_text:
            print(f"شاخ: {GEN}")
            diff += 1
    else:
        with open(GEN, "w", encoding="utf-8", newline="") as f:
            f.write(gen_text)
    print("سليم" if diff == 0 else f"{diff} اختلافاً")
    return 1 if diff else 0


if __name__ == "__main__":
    sys.exit(main())
