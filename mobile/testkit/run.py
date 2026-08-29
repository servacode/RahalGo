# -*- coding: utf-8 -*-
"""
══════════════════════════════════════════════════════════════════════
 منظومةُ اختبار تطبيقات رحّال غو
══════════════════════════════════════════════════════════════════════

    python mobile/testkit/run.py              الطبقتان اللتان لا تحتاجان جهازاً
    python mobile/testkit/run.py --device     ومعهما الجهاز
    python mobile/testkit/run.py --only guards
    python mobile/testkit/run.py --json out.json

# ثلاثُ طبقاتٍ لأنّ العطبَ ثلاثةُ أنواع

**١ · حرّاسٌ ساكنون** — يقرؤون الشيفرةَ كلَّها بلا تشغيل. **يمسكون ما
لا تفتحه**: تطبيقٌ بلا إشعارات، ولفظٌ مكرّرٌ سيفترق، ومضيفٌ منسيّ.

**٢ · عقدُ المحرّك** — نداءاتٌ حيّةٌ إلى الإنتاج. **تمسك عطبَ النشر**:
حقلٌ نُشر ناقصاً وصورةٌ لا تُخدَم وترويسةٌ ضاعت.

**٣ · الجهاز** — يفتح ويعبث ويقرأ السجلّ. **يمسك الانهيارَ والتجميدَ
والتسريب** — وهي وحدَها التي لا يراها بناءٌ ولا مراجعة.

# وهي تبقى وتُحدَّث

**كلُّ عطبٍ يُكتشف يصير حارساً هنا** — فلا يعود مرّتين. **وهذه هي
الفائدةُ الوحيدةُ من منظومةِ اختبار**: لا أن تقول «سليم» اليوم، **بل
ألّا يعود ما أُصلح.**
"""

import io
import json
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.abspath(__file__)))
if hasattr(sys.stdout, "reconfigure"):
    sys.stdout.reconfigure(encoding="utf-8")

import guards   # noqa: E402
import api      # noqa: E402
import device   # noqa: E402

LINE = u"═" * 66
THIN = u"─" * 66


def head(t):
    print(u"")
    print(LINE)
    print(u"  " + t)
    print(LINE)


def section(layer, rows, verbose):
    """يطبع طبقةً ويعيد (سليم، ساقط، مخالفات)."""
    ok = fail = 0
    total_hits = 0
    for name, why, hits in rows:
        if hits:
            fail += 1
            total_hits += len(hits)
            print(u"  ✗ %-42s %d" % (name, len(hits)))
            for h in hits[:6]:
                where = str(h[0])
                if len(h) == 3 and h[1]:
                    where += u":" + str(h[1])
                print(u"      · %s" % where[-72:])
                print(u"        %s" % (h[-1])[:78])
            if len(hits) > 6:
                print(u"      · … و%d غيرُها" % (len(hits) - 6))
            if verbose:
                print(u"      » %s" % why)
        else:
            ok += 1
            print(u"  ✓ %s" % name)
    print(THIN)
    print(u"  %s: %d سليم · %d ساقط · %d مخالفة" % (layer, ok, fail, total_hits))
    return ok, fail, total_hits


def main():
    args = sys.argv[1:]
    want_device = "--device" in args
    verbose = "--why" in args
    only = None
    if "--only" in args:
        only = args[args.index("--only") + 1]
    out_json = None
    if "--json" in args:
        out_json = args[args.index("--json") + 1]

    head(u"منظومةُ اختبار تطبيقات رحّال غو")
    summary = {}

    if not only or only == "guards":
        head(u"الطبقةُ ١ — حرّاسٌ ساكنون على الشيفرة")
        rows = guards.run()
        summary["guards"] = section(u"الحرّاس", rows, verbose)

    if not only or only == "api":
        head(u"الطبقةُ ٢ — عقدُ المحرّك (نداءاتٌ حيّةٌ إلى الإنتاج)")
        rows = api.run()
        summary["api"] = section(u"العقد", rows, verbose)

    if want_device or only == "device":
        head(u"الطبقةُ ٣ — الجهاز: إقلاعٌ · عبثٌ عشوائيٌّ · قراءةُ السجلّ")
        dev, res = device.run()
        if dev is None:
            print(u"  ⊘ لا جهازَ موصول — الطبقةُ لم تُشغَّل.")
            print(u"    الوصل: فعّل «تصحيح لاسلكيّ» ثمّ:  adb pair <ip:port>")
            summary["device"] = (0, 0, 0)
        else:
            print(u"  الجهاز: %s" % dev)
            rows = [(m, u"", h) for m, h in res]
            summary["device"] = section(u"الجهاز", rows, verbose)
    elif not only:
        head(u"الطبقةُ ٣ — الجهاز")
        print(u"  ⊘ لم تُطلب. أضف --device والجهازُ موصول.")

    head(u"الحصيلة")
    bad = 0
    for k, (ok, fail, hits) in summary.items():
        print(u"  %-10s %d سليم · %d ساقط" % (k, ok, fail))
        bad += fail
    print(THIN)
    print(u"  " + (u"كلُّ ما فُحص سليم." if bad == 0 else u"%d فحصاً ساقطاً." % bad))

    if out_json:
        payload = {}
        for layer, rows in (("guards", guards.run()), ("api", api.run())):
            payload[layer] = [
                {"name": n, "why": w, "hits": [list(map(str, h)) for h in hits]}
                for n, w, hits in rows
            ]
        io.open(out_json, "w", encoding="utf-8").write(
            json.dumps(payload, ensure_ascii=False, indent=2))
        print(u"  التقريرُ كُتب: %s" % out_json)

    return 1 if bad else 0


if __name__ == "__main__":
    sys.exit(main())
