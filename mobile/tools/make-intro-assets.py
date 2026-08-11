#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
يحوّل طبقات شعار رحّال غو من SVG إلى صيغة أندرويد، ويستخرج خطّ منتصف الطريق.

    python mobile/tools/make-intro-assets.py
    python mobile/tools/make-intro-assets.py --check

═══════════════════════════════════════════════════════════════════════
لماذا سكربتٌ ولا يُحوَّل مرّةً باليد
═══════════════════════════════════════════════════════════════════════

الطبقاتُ أربعٌ، وكلُّ واحدةٍ تحتاج تحويلَين: مسارُها إلى صيغة أندرويد،
وموضعُها إلى لوحةٍ موحّدة. ومن حوّل ثلاثاً ونسي الرابعةَ تطلع حركتُه
بقطعةٍ في غير مكانها — ولا شيءَ ينبّهه.

وخطُّ منتصف الطريق يُحسب حسابا: الطريقُ شكلٌ مملوءٌ لا خطّ، فلا يُمكن
«رسمُه» تدريجيّاً كما يُرسم خطّ. فيُستخرج منه خطُّ منتصفٍ يُستعمل مرّتين:
مساراً للكشف التدريجيّ، ومساراً تمشي عليه الدرّاجة.

ولو حُسب في زمن التشغيل لأبطأ إقلاعَ التطبيق بلا سبب — وهو ثابتٌ لا يتغيّر.
فيُحسب هنا مرّةً ويُكتب رقماً في الشيفرة.

═══════════════════════════════════════════════════════════════════════
"""

import math
import re
import sys
from pathlib import Path

# **ومخرجُ الطرفيّة يُجبَر على UTF-8** — طرفيّةُ ويندوز تُخرج بترميزٍ عربيٍّ
# قديم (cp1256) فتسقط على أوّل رمزٍ خارجه، **والسكربتُ يفشل بعد أن أتمّ عمله.**
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
    sys.stderr.reconfigure(encoding="utf-8")

try:
    from PIL import Image, ImageDraw
except ImportError:
    sys.exit("يلزم Pillow:  pip install Pillow")

ROOT = Path(__file__).resolve().parents[1]           # mobile/
SRC = ROOT / "brand" / "intro"
OUT = ROOT / "design" / "src" / "main" / "res" / "drawable"
GEN = ROOT / "design" / "src" / "main" / "kotlin" / "com" / "rahalgo" / "design" / "intro"

# الطبقاتُ وأسماؤها في أندرويد.
#
# والأسماءُ تُكتب هنا لا تُشتقّ من أسماء الملفّات: أسماءُ الموارد في أندرويد
# لا تقبل أرقاماً في أوّلها ولا حروفاً كبيرة، وملفّاتُ المصدر تحمل الاثنين.
LAYERS = {
    "01_R_Arrow.svg": "intro_mark",
    "02_Location_Pin.svg": "intro_pin",
    "03_Orange_Route.svg": "intro_route",
    "04_Motorcycle.svg": "intro_motorcycle",
}
ROUTE = "03_Orange_Route.svg"
CANVAS = 1024.0


def read_svg(path: Path):
    """يعيد قائمةَ (بيانات المسار، لون التعبئة) بترتيب ظهورها."""
    text = path.read_text(encoding="utf-8")
    out = []
    for tag in re.findall(r"<path\b[^>]*/?>", text):
        d = re.search(r'\sd="([^"]+)"', tag)
        if not d:
            continue
        fill = re.search(r'fill="([^"]+)"', tag)
        out.append((d.group(1).strip(), (fill.group(1) if fill else "#000000")))
    if not out:
        sys.exit(f"لا مسارَ في {path.name}")
    return out


def vector_xml(paths, size=CANVAS) -> str:
    """يبني `VectorDrawable`.

    وبيانات المسار تُنقل كما هي: الملفّاتُ مرسومةٌ بأوامرَ بسيطة
    (`M` و`L` و`Z`) وهي مشتركةٌ بين SVG وأندرويد حرفاً بحرف.
    """
    body = "\n".join(
        f'    <path\n        android:fillColor="{fill}"\n        android:pathData="{d}" />'
        for d, fill in paths
    )
    return (
        '<?xml version="1.0" encoding="utf-8"?>\n'
        "<!-- مولَّد بـ mobile/tools/make-intro-assets.py — لا يُحرَّر باليد. -->\n"
        '<vector xmlns:android="http://schemas.android.com/apk/res/android"\n'
        f'    android:width="{size:.0f}dp"\n'
        f'    android:height="{size:.0f}dp"\n'
        f'    android:viewportWidth="{size:.0f}"\n'
        f'    android:viewportHeight="{size:.0f}">\n'
        f"{body}\n"
        "</vector>\n"
    )


# ══════════════════════════════════════════════════════════════════════
#  خطُّ منتصف الطريق
# ══════════════════════════════════════════════════════════════════════
#
# الطريقُ حرفُ `S` يبدأ أسفلَ اليسار مبسوطاً أفقيّاً، ثمّ يصعد متعرّجاً إلى
# أعلى اليمين حيث يلتقي الشعار.
#
# فلا يكفي مسحٌ واحد: المسحُ بالصادات (صفّاً صفّاً) يعطي منتصفاً دقيقاً في
# الجزء الصاعد — وقِيس: قفزةٌ متوسّطها ١٫٦ بكسل — **ويقفز ٩١ بكسلاً عند
# الذيل المبسوط**، لأنّ الصفَّ هناك يقطع الطريقَ طولاً لا عرضاً.
#
# فيُمسح الذيلُ بالسينات (عموداً عموداً) والصاعدُ بالصادات، ثمّ يُوصلان.

FLAT_WIDTH = 170  # عرضٌ يزيد عنه = الطريقُ مبسوطٌ أفقيّاً هنا
SMOOTH = 9        # نصفُ نافذة المتوسّط المتحرّك


def centerline(paths) -> list:
    mask = Image.new("L", (int(CANVAS), int(CANVAS)), 0)
    dr = ImageDraw.Draw(mask)
    for d, _ in paths:
        pts = [(float(x), float(y)) for x, y in re.findall(r"(-?[\d.]+)\s+(-?[\d.]+)", d)]
        if len(pts) > 2:
            dr.polygon(pts, fill=255)
    px = mask.load()
    n = int(CANVAS)

    def span_x(y):
        xs = [x for x in range(n) if px[x, y] > 128]
        return (xs[0], xs[-1]) if len(xs) > 3 else None

    def span_y(x):
        ys = [y for y in range(n) if px[x, y] > 128]
        return (ys[0], ys[-1]) if len(ys) > 3 else None

    flat_y = None
    for y in range(n):
        s = span_x(y)
        if s and (s[1] - s[0]) > FLAT_WIDTH:
            flat_y = y
            break

    tail = []
    if flat_y is not None:
        for x in range(n):
            s = span_y(x)
            if s and s[0] >= flat_y - 6:
                tail.append((x, (s[0] + s[1]) / 2))

    up = []
    for y in range(n):
        s = span_x(y)
        if s and (s[1] - s[0]) <= FLAT_WIDTH:
            up.append((y, (s[0] + s[1]) / 2))

    line = tail + [(cx, y) for y, cx in sorted(up, reverse=True)]
    if len(line) < 20:
        sys.exit("تعذّر استخراجُ خطّ المنتصف — أتبدّل شكلُ الطريق؟")

    out = []
    for i in range(len(line)):
        seg = line[max(0, i - SMOOTH): i + SMOOTH + 1]
        out.append((sum(a for a, _ in seg) / len(seg), sum(b for _, b in seg) / len(seg)))
    return out


def thin(line: list, step: float = 14.0) -> list:
    """يُبقي نقطةً كلَّ مسافةٍ — **ثلاثُ مئةِ نقطةٍ في الشيفرة ثقلٌ بلا فائدة.**"""
    out = [line[0]]
    for p in line[1:]:
        if math.dist(p, out[-1]) >= step:
            out.append(p)
    if out[-1] != line[-1]:
        out.append(line[-1])
    return out


def kotlin_path(line: list) -> str:
    pts = ",\n    ".join(f"{x:.1f}f to {y:.1f}f" for x, y in line)
    return (
        "package com.rahalgo.design.intro\n\n"
        "// مولَّد بـ mobile/tools/make-intro-assets.py — لا يُحرَّر باليد.\n"
        "//\n"
        "// خطُّ منتصف الطريق البرتقاليّ في لوحة الشعار (1024×1024)، من بدايته\n"
        "// أسفلَ اليسار إلى ملتقاه بالشعار أعلى اليمين.\n"
        "//\n"
        "// يُستعمل مرّتين: مساراً للكشف التدريجيّ عن الطريق، ومساراً تمشي عليه\n"
        "// الدرّاجة. **وواحدٌ للاثنين لا اثنان** — ولو افترقا لَمشت الدرّاجةُ\n"
        "// بجانب الطريق لا عليه.\n"
        "//\n"
        f"// النقاط: {len(line)} · اللوحة: {CANVAS:.0f}\n"
        "internal val RoutePoints: List<Pair<Float, Float>> = listOf(\n"
        f"    {pts},\n"
        ")\n"
    )


def main() -> None:
    check = "--check" in sys.argv
    changed = 0

    def put(path: Path, text: str):
        nonlocal changed
        data = text.encode("utf-8")
        if path.exists() and path.read_bytes() == data:
            return
        changed += 1
        if not check:
            path.parent.mkdir(parents=True, exist_ok=True)
            path.write_bytes(data)

    for src_name, res_name in LAYERS.items():
        src = SRC / src_name
        if not src.exists():
            sys.exit(f"طبقةٌ ناقصة: {src}")
        put(OUT / f"{res_name}.xml", vector_xml(read_svg(src)))
        print(f"  {src_name} → drawable/{res_name}.xml")

    # ══════════════════════════════════════════════════════════════════
    # **والشعارُ كاملاً في ملفٍّ واحدٍ أيضاً**
    # ══════════════════════════════════════════════════════════════════
    #
    # (شرطُ المالك ٢٠٢٦-٠٨-١١: «اللوغو صورةٌ واحدةٌ مقفولة، لا يُفكَّك ولا
    #  يُعاد رسمُه».)
    #
    # **والترتيبُ ترتيبُ أرقام الملفّات**: الحرفُ خلفَ الكلّ، ثمّ العلامةُ
    # في جوفه، ثمّ الطريقُ أمامَه، ثمّ الدرّاجةُ فوق الجميع.
    #
    # **ومتّجهٌ لا صورة**: يبقى حادّاً في أيّ حجمٍ وعلى أيّ شاشة، **وصورةٌ
    # نقطيّةٌ تُكبَّر تُهبّب حوافَّها.**
    merged = []
    for src_name in LAYERS:
        merged += read_svg(SRC / src_name)
    put(OUT / "intro_logo.xml", vector_xml(merged))
    print(f"  الشعارُ كاملاً → drawable/intro_logo.xml ({len(merged)} مساراً)")

    line = thin(centerline(read_svg(SRC / ROUTE)))
    put(GEN / "RoutePoints.kt", kotlin_path(line))
    print(f"  خطّ المنتصف: {len(line)} نقطة  "
          f"({line[0][0]:.0f},{line[0][1]:.0f} → {line[-1][0]:.0f},{line[-1][1]:.0f})")

    if check and changed:
        sys.exit(f"\n{changed} ملفّاً لا يطابق المصدر — شغّل: python mobile/tools/make-intro-assets.py")
    print(f"\n{'مطابق' if not changed else f'حُدّث {changed} ملفّاً'}")


if __name__ == "__main__":
    main()
