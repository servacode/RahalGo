#!/usr/bin/env python3
# -*- coding: utf-8 -*-
"""
يحرس تعليقات Kotlin من التداخل غير المقصود.

    python mobile/tools/check-kotlin-comments.py

═══════════════════════════════════════════════════════════════════════
لماذا حارس لهذا
═══════════════════════════════════════════════════════════════════════

Kotlin يسمح بتعليقات متداخلة: شرطة مائلة ثمّ نجمة داخل تعليق قائم **تفتح
تعليقا ثانيا** يبقى مفتوحا إلى آخر الملفّ.

ووقع مرّتين في يوم واحد (٢٠٢٦-٠٨-١١):

  - `Theme.kt`  — كُتب مسار ملفّ فيه نجمة داخل تعليق التوثيق
  - `AuthApi.kt` — كُتب مسار API فيه نجمة كذلك

**والرسالة تضلّل**: «تعليق لم يُغلق» عند **آخر سطر في الملفّ**، لا عند
موضع العطب. فيُبحث في المكان الخطأ.

**وهو من عائلة العطب نفسها المكتوبة في `CLAUDE.md`**: العلامة الخلفيّة
تُنهي النصّ الخام في Go. **ولغة تسمح بشيء داخل تعليق تجعل التعليق شيفرة.**

═══════════════════════════════════════════════════════════════════════
"""

import re
import sys
from pathlib import Path

if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")
    sys.stderr.reconfigure(encoding="utf-8")

ROOT = Path(__file__).resolve().parents[1]
OPEN = "/" + "*"
CLOSE = "*" + "/"


def offenders(path: Path) -> list:
    """أسطر تفتح تعليقا داخل تعليق — أو تُغلق ما لم يُفتح."""
    out = []
    depth = 0
    for i, line in enumerate(path.read_text(encoding="utf-8").split("\n"), 1):
        opens = line.count(OPEN)
        closes = line.count(CLOSE)
        if depth > 0 and opens > closes:
            out.append((i, line.strip()[:90]))
        depth += opens - closes
    if depth != 0:
        out.append((0, f"الملفّ ينتهي وعمق التعليق {depth}"))
    return out


def main() -> None:
    bad = {}
    for f in sorted(ROOT.rglob("*.kt")):
        if "/build/" in f.as_posix():
            continue
        hits = offenders(f)
        if hits:
            bad[f] = hits

    if bad:
        print(f"\nتعليقات متداخلة: {len(bad)} ملفّاً\n")
        for f, hits in bad.items():
            rel = f.relative_to(ROOT.parent).as_posix()
            for line, text in hits:
                where = f"{rel}:{line}" if line else rel
                print(f"   {where}\n      {text}")
        print(
            "\n**شرطة مائلة ثمّ نجمة داخل تعليق تفتح تعليقا ثانيا يبتلع بقية الملفّ.**\n"
            "اكتب المسار بلا نجمة، أو صِفه بالكلام.\n"
        )
        sys.exit(1)

    n = sum(1 for f in ROOT.rglob("*.kt") if "/build/" not in f.as_posix())
    print(f"تعليقات Kotlin سليمة — فُحص {n} ملفّاً.")


if __name__ == "__main__":
    main()
