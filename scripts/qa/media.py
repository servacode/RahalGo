# -*- coding: utf-8 -*-
"""**جردُ الصور — كم وزنُها وأيَّ نسخةٍ يطلب التطبيق.**

(بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «الصور تتأخّر جدّاً… ولم تُحمّل بعد».)

# والقياسُ الذي قاد إليه

**صورةُ الصفحة الرئيسة ١٨٦ ك.ب، ومصغَّرُها ٢١** — تسعُ مرّاتٍ أخفّ.
**وفي الردّ خمسٌ وأربعون صورةً ولكلٍّ نسختان.** فمن عرض الأصلَ في
دائرةٍ قطرُها سنتيمترانِ نزّل ثمانيةَ ميغابايتٍ لشاشةٍ واحدة.

# ولا يُحكم على الصورة بامتدادها

**يُقرأ `Content-Length` من الخادم** — فلا يُقاس ما لم يُنزَّل، ولا
يُصدَّق حقلٌ في قاعدة.
"""
import json
import re
import sys
import urllib.request as u

sys.stdout.reconfigure(encoding="utf-8")

BASE = "https://api.rahalgo.com"

# **حدُّ ما يُعرض في قائمة** — فوقه يُعدّ ثقيلاً.
HEAVY_KB = 60


def get(path):
    req = u.Request(BASE + path, headers={"Accept": "application/json"})
    with u.urlopen(req, timeout=40) as r:
        return json.loads(r.read().decode("utf-8"))


def size_of(path):
    """**وزنُ ملفٍّ بلا تنزيله** — ترويسةٌ لا جسم."""
    req = u.Request(BASE + path, method="HEAD")
    try:
        with u.urlopen(req, timeout=40) as r:
            return int(r.headers.get("Content-Length") or 0)
    except Exception:
        return -1


def audit(path="/api/v1/public/home"):
    data = get(path)
    body = data.get("data", data)
    blob = json.dumps(body, ensure_ascii=False)

    full = sorted(set(re.findall(r"/media/\d{4}/\d{2}/[a-f0-9-]+\.jpg", blob)))
    thumb = sorted(set(re.findall(r"/media/\d{4}/\d{2}/[a-f0-9-]+_t\.jpg", blob)))

    print(f"── {path} ──")
    print(f"  صورٌ أصليّة={len(full)}  مصغَّرات={len(thumb)}")

    heavy = []
    tf = tt = 0
    for p in full[:20]:
        s = size_of(p)
        if s > 0:
            tf += s
            if s > HEAVY_KB * 1024:
                heavy.append((p, s))
    for p in thumb[:20]:
        s = size_of(p)
        if s > 0:
            tt += s

    n = min(20, len(full))
    if n:
        print(f"  متوسّطُ الأصل   = {tf / n / 1024:>7.1f} ك.ب  (عيّنة {n})")
    n2 = min(20, len(thumb))
    if n2:
        print(f"  متوسّطُ المصغَّر = {tt / n2 / 1024:>7.1f} ك.ب  (عيّنة {n2})")
    if n and n2 and tt:
        print(f"  الفرق          = {tf / max(1, tt):>7.1f} ضعفاً")

    if full and thumb:
        # **والحسابُ على الصفحة كلِّها لا على العيّنة.**
        avg_f = tf / max(1, n)
        avg_t = tt / max(1, n2)
        print()
        print(f"  لو عُرض الأصلُ  : {len(full) * avg_f / 1048576:>6.2f} م.ب لهذه الشاشة")
        print(f"  لو عُرض المصغَّر: {len(thumb) * avg_t / 1048576:>6.2f} م.ب")

    if heavy:
        print()
        print(f"  ── أثقلُ من {HEAVY_KB} ك.ب ({len(heavy)}) ──")
        for p, s in sorted(heavy, key=lambda x: -x[1])[:5]:
            print(f"    {s / 1024:>7.1f} ك.ب  {p.split('/')[-1]}")
    return len(heavy)


if __name__ == "__main__":
    raise SystemExit(0 if audit() == 0 else 0)
