#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
يُولّد أيقونات تطبيقات أندرويد من لوغو واحد.

    python mobile/tools/make-icon.py              # كلّ التطبيقات
    python mobile/tools/make-icon.py app-driver   # واحدٌ بعينه
    python mobile/tools/make-icon.py --check      # يفحص ولا يكتب

═══════════════════════════════════════════════════════════════════════
لماذا سكربتٌ ولا يُبدَّل الملفُّ باليد
═══════════════════════════════════════════════════════════════════════

الأيقونةُ الواحدةُ خمسةَ عشرَ ملفّاً: ثلاثةُ أشكالٍ (مربّعٌ · مدوّرٌ ·
طبقةٌ أماميّة) في خمس كثافاتِ شاشة. ومن بدّل واحداً منها فقط تطلع أيقونتُه
حادّةً على جهازٍ ومهبَّبةً على غيره — ولا شيءَ يُنبّهه.

وفيها شغلٌ لا يُرى: قصُّ الكلمة من أسفل اللوغو، وتوسيطُ الرمز في المنطقة
التي لا يقصّها النظام، وقصُّ الشكل الدائريّ.

وأربعةُ تطبيقاتٍ قادمة. فلو تُرك للذاكرة لَتفرّقت أيقوناتُها بعد شهر.

═══════════════════════════════════════════════════════════════════════
تبديلُ اللوغو صار: ضعه مكان mobile/brand/logo.png ثمّ شغّل هذا السطر.
═══════════════════════════════════════════════════════════════════════
"""

import json
import sys
from pathlib import Path

try:
    from PIL import Image, ImageDraw
except ImportError:
    sys.exit("يلزم Pillow:  pip install Pillow")

ROOT = Path(__file__).resolve().parents[1]          # mobile/
BRAND = ROOT / "brand"

# كثافاتُ الشاشة ومقاسُ كلٍّ منها بالبكسل.
#
# الطبقةُ الأماميّةُ للأيقونة التكيّفيّة لوحتُها ١٠٨ نقطة، والقديمةُ ٤٨.
# والنسبةُ بين الكثافات ثابتة: mdpi ×1 · hdpi ×1.5 · xhdpi ×2 · xxhdpi ×3
# · xxxhdpi ×4.
FOREGROUND = {"mdpi": 108, "hdpi": 162, "xhdpi": 216, "xxhdpi": 324, "xxxhdpi": 432}
LEGACY = {"mdpi": 48, "hdpi": 72, "xhdpi": 96, "xxhdpi": 144, "xxxhdpi": 192}

# نسبةُ الرمز من اللوحة في الأيقونة القديمة — أوسعُ من التكيّفيّة لأنّ
# النظامَ لا يقصّها.
LEGACY_FILL = 0.78
LEGACY_ROUND_FILL = 0.72


def load_config() -> dict:
    cfg = json.loads((BRAND / "icon.json").read_text(encoding="utf-8"))
    for key in ("source", "crop_bottom", "safe", "background", "apps"):
        if key not in cfg:
            sys.exit(f"إعدادُ الأيقونة ناقصٌ: {key}")
    return cfg


def load_mark(cfg: dict) -> Image.Image:
    """اللوغو بعد قصّ الكلمة وتشذيب الفراغ حوله."""
    src = BRAND / cfg["source"]
    if not src.exists():
        sys.exit(f"لا لوغو في {src}")
    im = Image.open(src).convert("RGBA")
    w, h = im.size
    keep = int(h * (1.0 - float(cfg["crop_bottom"])))
    im = im.crop((0, 0, w, keep))

    # وتشذيبُ الفراغ الشفّاف حوله — وإلّا صار الرمزُ صغيراً في وسط لوحةٍ
    # فارغة، ويُقرأ ضامراً بين أيقونات الجوّال.
    box = im.getbbox()
    if box:
        im = im.crop(box)
    return im


def hex_rgba(value: str) -> tuple:
    v = value.lstrip("#")
    return tuple(int(v[i:i + 2], 16) for i in (0, 2, 4)) + (255,)


def render(mark: Image.Image, size: int, fill: float, bg=None, circle=False) -> Image.Image:
    canvas = Image.new("RGBA", (size, size), bg or (0, 0, 0, 0))
    m = mark.copy()
    target = max(1, int(size * fill))
    m.thumbnail((target, target), Image.LANCZOS)
    canvas.paste(m, ((size - m.width) // 2, (size - m.height) // 2), m)
    if circle:
        mask = Image.new("L", (size, size), 0)
        ImageDraw.Draw(mask).ellipse((0, 0, size - 1, size - 1), fill=255)
        canvas.putalpha(mask)
    return canvas


def build(app: str, cfg: dict, mark: Image.Image, check: bool) -> int:
    """يكتب أيقوناتِ تطبيقٍ واحد — ويعيد عددَ ما اختلف."""
    res = ROOT / app / "src" / "main" / "res"
    if not res.exists():
        sys.exit(f"لا مجلّدَ موارد في {res} — أهذا اسمُ تطبيقٍ صحيح؟")

    bg = hex_rgba(cfg["background"])
    changed = 0

    def put(path: Path, img: Image.Image):
        nonlocal changed
        path.parent.mkdir(parents=True, exist_ok=True)
        import io
        buf = io.BytesIO()
        img.save(buf, "PNG")
        new = buf.getvalue()
        if path.exists() and path.read_bytes() == new:
            return
        changed += 1
        if not check:
            path.write_bytes(new)

    for dens, size in FOREGROUND.items():
        put(res / f"mipmap-{dens}" / "ic_launcher_foreground.png",
            render(mark, size, float(cfg["safe"])))

    for dens, size in LEGACY.items():
        put(res / f"mipmap-{dens}" / "ic_launcher.png",
            render(mark, size, LEGACY_FILL, bg=bg))
        put(res / f"mipmap-{dens}" / "ic_launcher_round.png",
            render(mark, size, LEGACY_ROUND_FILL, bg=bg, circle=True))

    # ملفّا الأيقونة التكيّفيّة ولونُ خلفيّتها — يُكتبان مرّةً ويبقيان.
    adaptive = (
        '<?xml version="1.0" encoding="utf-8"?>\n'
        "<!-- مولَّد بـ mobile/tools/make-icon.py — لا يُحرَّر باليد. -->\n"
        '<adaptive-icon xmlns:android="http://schemas.android.com/apk/res/android">\n'
        '    <background android:drawable="@color/ic_launcher_background" />\n'
        '    <foreground android:drawable="@mipmap/ic_launcher_foreground" />\n'
        '    <monochrome android:drawable="@mipmap/ic_launcher_foreground" />\n'
        "</adaptive-icon>\n"
    )
    color = (
        '<?xml version="1.0" encoding="utf-8"?>\n'
        "<!-- مولَّد بـ mobile/tools/make-icon.py — لا يُحرَّر باليد. -->\n"
        "<resources>\n"
        f'    <color name="ic_launcher_background">{cfg["background"]}</color>\n'
        "</resources>\n"
    )

    def put_text(path: Path, text: str):
        nonlocal changed
        path.parent.mkdir(parents=True, exist_ok=True)
        data = text.encode("utf-8")
        if path.exists() and path.read_bytes() == data:
            return
        changed += 1
        if not check:
            path.write_bytes(data)

    put_text(res / "mipmap-anydpi-v26" / "ic_launcher.xml", adaptive)
    put_text(res / "mipmap-anydpi-v26" / "ic_launcher_round.xml", adaptive)
    put_text(res / "values" / "ic_launcher_background.xml", color)
    return changed


def main() -> None:
    args = [a for a in sys.argv[1:] if a != "--check"]
    check = "--check" in sys.argv

    cfg = load_config()
    mark = load_mark(cfg)
    apps = args or cfg["apps"]

    total = 0
    for app in apps:
        n = build(app, cfg, mark, check)
        total += n
        state = "يختلف" if (check and n) else ("لم يتغيّر" if not n else "حُدِّث")
        print(f"  {app}: {state}" + (f" ({n} ملفّاً)" if n else ""))

    if check and total:
        sys.exit(
            f"\nأيقوناتُ {total} ملفّاً لا تطابق اللوغو — "
            "شغّل: python mobile/tools/make-icon.py"
        )
    print(f"\nالرمز: {mark.size[0]}×{mark.size[1]} · الخلفية {cfg['background']} · "
          f"{len(FOREGROUND)} كثافات")


if __name__ == "__main__":
    main()
