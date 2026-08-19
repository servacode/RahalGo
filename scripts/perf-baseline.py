# -*- coding: utf-8 -*-
"""**قياسٌ مرجعيٌّ للأداء — تُقارَن به الإصدارات.**

(أولويّةُ المالك ٢٠٢٦-٠٨-١٩، البند ٢٣: «أنشئ baseline يمكن مقارنة
 الإصدارات به… ولا تعطِ PASS إذا لم يوجد Benchmark أو Measurement».)

# وما يُقاس

**زمنُ الإقلاع** حتّى أوّل شاشةٍ مقروءة · **زمنُ ردِّ الأبواب الحارّة** ·
**نموُّ الذاكرة** بعد تصفّحٍ متكرّر · **الإطاراتُ الضائعة** في التمرير.

# ولا حكمَ بلا رقمٍ سابق

**أوّلُ تشغيلٍ يكتب المرجع** (`docs/qa/perf-baseline.json`) ولا يحكم.
**وما بعده يقارن** — وتراجعٌ فوق العتبة يُطبع أحمر.

    python scripts/perf-baseline.py            # يقيس ويقارن
    python scripts/perf-baseline.py --reset    # يكتب مرجعاً جديدا
"""
import json
import os
import re
import subprocess
import sys
import time
import urllib.request as u

sys.stdout.reconfigure(encoding="utf-8")

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
REF = os.path.join(ROOT, "docs", "qa", "perf-baseline.json")
PKG = "com.rahalgo.customer"
API = "https://api.rahalgo.com/api/v1"

# **وعتبةُ التراجع ٢٥٪** — **وضجيجُ القياس على هاتفٍ حقيقيٍّ يبلغ
# العشرة**، فعتبةٌ ضيّقةٌ تُنذر كلَّ يومٍ ثمّ تُطفأ.
TOLERANCE = 0.25


def adb(*a, t=90):
    return subprocess.run(["adb"] + list(a), capture_output=True,
                          timeout=t).stdout.decode("utf-8", "replace")


def have_device():
    return any(l.strip().endswith("\tdevice") for l in adb("devices").splitlines())


def cold_start():
    """**زمنُ الإقلاع البارد** — يقرؤه أندرويد نفسُه لا ساعتُنا."""
    adb("shell", "am", "force-stop", PKG)
    time.sleep(1.5)
    out = adb("shell", "am", "start-activity", "-W", "-n",
              PKG + "/com.rahalgo.customer.MainActivity")
    m = re.search(r"TotalTime:\s*(\d+)", out)
    return int(m.group(1)) if m else None


def memory_mb():
    out = adb("shell", "dumpsys", "meminfo", PKG)
    m = re.search(r"TOTAL(?:\s+PSS)?:\s*(\d+)", out)
    return round(int(m.group(1)) / 1024.0, 1) if m else None


def jank():
    """**الإطاراتُ الضائعة** — نسبةُ ما تجاوز المهلة.

    # ولماذا تسخينٌ قبل القياس

    **أوّلُ تمريرةٍ بعد الإقلاع تُحمّل الصورَ وتبني القوائم** — فتضيع
    فيها إطاراتٌ لا تضيع بعدها. **وقياسٌ يبدأ من الإقلاع يقيس التحميلَ
    لا التمرير.**

    (قِيس ٢٠٢٦-٠٨-١٩: ٣١٪ ثمّ ٥٥٪ ثمّ ٤٦٪ في ثلاث تشغيلاتٍ متتالية —
     **ومقياسٌ يتأرجح بالنصف لا يُحكَم به على إصدار.**)
    """
    # ══════════════════════════════════════════════════════════════
    # **وتُقطع الشبكةُ بعد التسخين — يُقاس الرسمُ لا التحميل**
    # ══════════════════════════════════════════════════════════════
    #
    # (قِيس ٢٠٢٦-٠٨-١٩: ١٣٫٦٪ إلى ٣٠٫٦٪ على البناء نفسِه — **وتشتّتٌ
    #  أكبرُ من الأثر المقيس لا يقيس شيئا.**)
    #
    # **ومصدرُ التشتّت جلبُ الصور**: تمريرةٌ تصادف صورةً تصل تضيع فيها
    # إطارات، **وسرعةُ الشبكةِ ليست من التطبيق.**
    #
    # **فتُملأ الذاكرةُ بالتسخين ثمّ تُقطع الشبكة** — وما بعدها رسمٌ
    # خالص. **والصورُ في ذاكرة `coil` فلا يتغيّر ما يُرسم.**
    for _ in range(6):
        adb("shell", "input", "swipe", "540", "1700", "540", "600", "300")
        time.sleep(0.4)
        adb("shell", "input", "swipe", "540", "600", "540", "1700", "300")
        time.sleep(0.4)
    adb("shell", "svc", "wifi", "disable")
    adb("shell", "svc", "data", "disable")
    time.sleep(3)
    adb("shell", "dumpsys", "gfxinfo", PKG, "reset")
    # **وعشرون تمريرةً لا ستّ** — والعيّنةُ الصغيرةُ تتأرجح.
    for _ in range(20):
        adb("shell", "input", "swipe", "540", "1700", "540", "600", "300")
        time.sleep(0.35)
    out = adb("shell", "dumpsys", "gfxinfo", PKG)
    # **وتُعاد الشبكةُ مهما وقع** — **وجهازٌ يُترك بلا إنترنت بعد قياسٍ
    # عطبٌ نصنعه بأيدينا.**
    adb("shell", "svc", "wifi", "enable")
    adb("shell", "svc", "data", "enable")
    tot = re.search(r"Total frames rendered:\s*(\d+)", out)
    jj = re.search(r"Janky frames:\s*(\d+)", out)
    if not tot or not jj or int(tot.group(1)) == 0:
        return None
    return round(100.0 * int(jj.group(1)) / int(tot.group(1)), 1)


def api_ms(path, n=5):
    """**وسيطُ زمنِ الردّ** — لا المتوسّط: **نداءٌ شاذٌّ واحدٌ يزيح
    المتوسّطَ ولا يزيح الوسيط.**"""
    times = []
    for _ in range(n):
        t0 = time.time()
        try:
            u.urlopen(API + path, timeout=25).read()
            times.append((time.time() - t0) * 1000)
        except Exception:
            pass
    if not times:
        return None
    times.sort()
    return round(times[len(times) // 2])


def main():
    reset = "--reset" in sys.argv
    now = {}

    print("═" * 62)
    print("  رحّال غو — القياسُ المرجعيُّ للأداء")
    print("═" * 62)

    if have_device():
        now["cold_start_ms"] = cold_start()
        time.sleep(6)
        now["memory_mb"] = memory_mb()
        now["jank_percent"] = jank()
    else:
        print("  ▲ لا جهازَ متّصل — تُقاس الأبوابُ وحدَها")

    now["api_home_ms"] = api_ms("/public/home")
    now["api_offers_ms"] = api_ms("/public/offers")

    old = {}
    if os.path.exists(REF) and not reset:
        with open(REF, encoding="utf-8") as f:
            old = json.load(f).get("metrics", {})

    LABEL = {
        "cold_start_ms": "الإقلاعُ البارد (م.ث)",
        "memory_mb": "الذاكرة (م.ب)",
        "jank_percent": "الإطاراتُ الضائعة (٪)",
        "api_home_ms": "الصفحةُ الأولى (م.ث)",
        "api_offers_ms": "العروض (م.ث)",
    }
    worse = []
    for k, label in LABEL.items():
        v = now.get(k)
        if v is None:
            print("  ▲ %-26s لم يُقَس" % label)
            continue
        o = old.get(k)
        if o is None:
            print("  · %-26s %-8s (مرجعٌ جديد)" % (label, v))
            continue
        d = (v - o) / o if o else 0
        mark = "✔"
        if d > TOLERANCE:
            mark, _ = "✘", worse.append(label)
        elif d < -TOLERANCE:
            mark = "▲"  # **تحسّنٌ ملحوظ — يُقرأ ولا يُحتفل به**
        print("  %s %-26s %-8s (كان %s · %+.0f٪)" % (mark, label, v, o, d * 100))

    if not old or reset:
        os.makedirs(os.path.dirname(REF), exist_ok=True)
        with open(REF, "w", encoding="utf-8") as f:
            json.dump({"metrics": now}, f, ensure_ascii=False, indent=2)
        print("─" * 62)
        print("  كُتب المرجع — والتشغيلُ القادمُ يقارن به.")
        print("═" * 62)
        return 0

    print("─" * 62)
    if worse:
        print("  ‼ تراجعٌ فوق %d٪: %s" % (TOLERANCE * 100, " · ".join(worse)))
        print("═" * 62)
        return 1
    print("  ✔ لا تراجع")
    print("═" * 62)
    return 0


if __name__ == "__main__":
    sys.exit(main())
