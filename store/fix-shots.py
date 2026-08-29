#!/usr/bin/env python3
"""
**لقطاتُ المتجر — تُهيّأ لشروط بلاي**

(قِيس ٢٠٢٦-٠٨-٢٤ أنّ لقطاتِ الجهاز تُرفض بسببين.)

**والحدُّ الأقصى للنسبة ٢:١** — و`1080x2400` تساوي ٢٫٢٢. **وبلاي تشترط
`24-bit PNG` بلا قناة شفافيّة**، ولقطاتُ أندرويد فيها قناة.

**والعلاجُ حشوةٌ جانبيّةٌ لا قصّ**: القصُّ يأكل من الشاشة.
"""
import glob
import os

from PIL import Image

TARGET_W, TARGET_H = 1200, 2400
# **ولونُ الحشوة لونُ أرض الخريطة** — فتبدو مقصودةً لا خطأً.
BG = (247, 244, 238)

SRC = os.path.join(os.path.dirname(__file__), "shots")
OUT = os.path.join(os.path.dirname(__file__), "shots-play")


def main() -> int:
    os.makedirs(OUT, exist_ok=True)
    for path in sorted(glob.glob(os.path.join(SRC, "*.png"))):
        im = Image.open(path)
        w, h = im.size
        canvas = Image.new("RGB", (TARGET_W, TARGET_H), BG)
        at = ((TARGET_W - w) // 2, (TARGET_H - h) // 2)
        if im.mode in ("RGBA", "LA", "P"):
            im = im.convert("RGBA")
            canvas.paste(im, at, im)
        else:
            canvas.paste(im.convert("RGB"), at)
        dest = os.path.join(OUT, os.path.basename(path))
        canvas.save(dest, "PNG", optimize=True)
        print(f"  {os.path.basename(path)} -> {TARGET_W}x{TARGET_H}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
