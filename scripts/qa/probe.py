# -*- coding: utf-8 -*-
"""**طبقةُ القياس — تُلحَق بأيّ مشيةٍ فتقول أين ضاع الوقت.**

(طلبُ المالك ٢٠٢٦-٠٨-٢٤: «هل الأزرار تعمل · تتكرّر · هل يحصل تهنيج ·
 حجم الصور · زمن التحميل · زمن الاستجابة وكلّ شيء».)

# ولماذا لا تكفي المشيةُ القائمة

**`cycle.py` تسأل: هل نجحت الخطوة؟** — والبطءُ لا يُسقِط خطوةً،
**يُنجحها ببطء.** فتمشي المنصّةُ ستّاً وأربعين خطوةً وتردّ «تمّ» وهي
تزحف.

# وما تقيسه هذه

	الزمن        كم استغرقت الخطوةُ من ضغطةٍ إلى شاشةٍ مستقرّة
	البايتات     كم نزّل التطبيقُ فيها من الشبكة
	الإطارات     كم إطاراً تعثّر — وهو التهنيجُ بعينه
	الصور        كم صورةً طُلبت وأيَّ نسخةٍ منها وكم وزنُها
	الأزرار      أضُغط الزرُّ فتبدّلت الشاشةُ أم بقيت كما هي

# والبايتاتُ تُقرأ من النظام لا من التطبيق

**وأندرويد يعدّ لكلّ تطبيقٍ ما نزّله** (`TrafficStats` عبر
`dumpsys netstats`). **فلا نصدّق التطبيقَ في تقرير نفسِه.**

# ولا حكمَ بلا عتبة

**والعتباتُ هنا من تجربة الإنسان لا من ذوقي**: ما دون ٤٠٠ ملّي يُحسّ
فوريّاً · وما فوق ثانيةٍ يُحسّ بطيئاً · وما فوق ثلاثٍ يُظنّ معطّلاً.
"""
import re
import subprocess
import sys
import time

sys.stdout.reconfigure(encoding="utf-8")

PKG = "com.rahalgo.customer"

# **عتباتُ الإحساس البشريّ** — بالميلّي ثانية.
INSTANT = 400
SLOW = 1000
BROKEN = 3000


def adb(*a, t=90):
    try:
        return subprocess.run(
            ["adb", *a], capture_output=True, text=True, timeout=t, encoding="utf-8", errors="replace"
        ).stdout
    except Exception:
        return ""


def uid_of(pkg=PKG):
    """**معرّفُ التطبيق في النظام** — به تُقرأ بايتاتُه وحدَه.

    **و`dumpsys package` تُخرج آلافَ الأسطر** ولا تحمل `userId=` في
    كلّ جهاز. **و`pm list packages -U` تُخرج سطراً واحداً فيه المعرّف**
    — أسرعُ وأصدق. (قِيس ٢٠٢٦-٠٨-٢٤: الأولى ردّت فارغاً فصارت كلُّ
    البايتات صفراً.)
    """
    out = adb("shell", "pm", "list", "packages", "-U", pkg)
    m = re.search(r"uid:(\d+)", out)
    if m:
        return m.group(1)
    out = adb("shell", "dumpsys", "package", pkg)
    m = re.search(r"userId=(\d+)", out)
    return m.group(1) if m else None


def bytes_down(uid):
    """**ما نزّله التطبيقُ منذ الإقلاع** — بالبايت.

    **ويُقرأ من `netstats` لا من التطبيق** — فلا يُسأل متّهمٌ عن نفسِه.
    """
    if not uid:
        return 0
    out = adb("shell", "dumpsys", "netstats", "detail", t=120)
    total = 0
    grab = False
    for line in out.splitlines():
        if f"uid={uid}" in line:
            grab = True
            continue
        if grab:
            m = re.search(r"rb=(\d+)", line)
            if m:
                total += int(m.group(1))
            if line.strip().startswith("uid=") or not line.strip():
                grab = False
    return total


def jank_reset():
    adb("shell", "dumpsys", "gfxinfo", PKG, "reset")


def jank_read():
    """**الإطاراتُ المتعثّرة** — وهي التهنيجُ الذي يراه المستخدم."""
    out = adb("shell", "dumpsys", "gfxinfo", PKG)
    total = re.search(r"Total frames rendered: (\d+)", out)
    janky = re.search(r"Janky frames: (\d+)", out)
    p90 = re.search(r"90th percentile: (\d+)ms", out)
    return {
        "frames": int(total.group(1)) if total else 0,
        "janky": int(janky.group(1)) if janky else 0,
        "p90ms": int(p90.group(1)) if p90 else 0,
    }


def screen_hash():
    """**بصمةُ الشاشة** — بها يُعرف: أتبدّلت بعد الضغطة أم لا."""
    out = adb("shell", "uiautomator", "dump", "/sdcard/probe.xml")
    if "ERROR" in out:
        return ""
    xml = adb("shell", "cat", "/sdcard/probe.xml")
    texts = re.findall(r'text="([^"]*)"', xml)
    return "|".join(t for t in texts if t)[:4000]


class Probe:
    """**سجلٌّ لخطواتٍ مقيسة** — يُفتح مرّةً ويُطبع في آخره."""

    def __init__(self):
        self.uid = uid_of()
        self.rows = []
        self.warnings = []

    def step(self, name, action, settle=1.5):
        """**يقيس خطوةً**: بايتاتُها وزمنُها وتعثّرُها وهل تبدّلت الشاشة.

        `action` دالّةٌ تفعل الشيء (ضغطةٌ أو تمرير).
        """
        # ══════════════════════════════════════════════════════════════
        # **ولا يُحسب زمنُ الأداة في زمن التطبيق**
        # ══════════════════════════════════════════════════════════════
        #
        # **وقراءةُ شجرة الشاشة (`uiautomator dump`) ثانيةٌ ونصف**،
        # وقراءةُ عدّاد الشبكة مثلُها. **فلو دخلتا الساعةَ خرجت كلُّ
        # خطوةٍ أربعَ ثوانٍ** — وهو زمنُ المقياس لا زمنُ المقيس.
        # (قِيس ٢٠٢٦-٠٨-٢٤ فخرجت خمسُ خطواتٍ متطابقةً عند ٤٫٤ ثانية،
        # **وتطابقُها هو الذي كشف أنّ الرقمَ ليس منها.**)
        #
        # **فتُقرأ الشاشةُ قبل الساعة وبعدها** — والساعةُ على الفعل
        # واستقرارِه وحدَهما.
        before_bytes = bytes_down(self.uid)
        before_screen = screen_hash()
        jank_reset()
        t0 = time.time()
        action()
        # **ويُنتظر استقرارُ الشاشة لا انتهاءُ الضغطة** — والفرقُ
        # بينهما هو ما يحسّه المستخدمُ بطئاً.
        time.sleep(settle)
        ms = int((time.time() - t0) * 1000)
        after_screen = screen_hash()
        j = jank_read()
        kb = (bytes_down(self.uid) - before_bytes) / 1024.0
        changed = after_screen != before_screen
        row = {
            "name": name,
            "ms": ms,
            "kb": kb,
            "changed": changed,
            **j,
        }
        self.rows.append(row)
        if not changed:
            self.warnings.append(f"«{name}» — الشاشةُ لم تتبدّل بعد الفعل")
        if ms > BROKEN:
            self.warnings.append(f"«{name}» — {ms}ملّي: يُظنّ معطّلاً")
        if j["frames"] and j["janky"] * 100 // max(1, j["frames"]) > 20:
            self.warnings.append(
                f"«{name}» — تعثّر {j['janky']} من {j['frames']} إطاراً"
            )
        return row

    def report(self):
        print()
        print("═" * 74)
        print("  الخطوة                     الزمن     نُزّل      إطارات   تبدّلت")
        print("═" * 74)
        for r in self.rows:
            mark = "✔" if r["changed"] else "✘"
            speed = "●" if r["ms"] <= INSTANT else ("◐" if r["ms"] <= SLOW else "○")
            jank = (
                f"{r['janky']}/{r['frames']}" if r["frames"] else "—"
            )
            print(
                f"  {r['name'][:24]:<24} {speed} {r['ms']:>5}ملّي "
                f"{r['kb']:>8.0f}ك.ب {jank:>9}   {mark}"
            )
        print("═" * 74)
        total_ms = sum(r["ms"] for r in self.rows)
        total_kb = sum(r["kb"] for r in self.rows)
        print(f"  المجموع: {total_ms}ملّي · {total_kb:.0f}ك.ب · {len(self.rows)} خطوة")
        if self.warnings:
            print()
            print("  ── ما يستحقّ النظر ──")
            for w in self.warnings:
                print(f"    ! {w}")
        print()
        return len(self.warnings)

def crashes(pkg=PKG):
    """**الانهياراتُ وحدَها — لا كلُّ ما فيه `AndroidRuntime`.**

    **وقِيس ٢٠٢٦-٠٨-٢٥ أنّ العدَّ بالكلمة يكذب**: أداةُ `monkey`
    تُقلع فتكتب اثنَي عشرَ سطراً فيها `AndroidRuntime`، **فيُقرأ إقلاعُ
    الأداة اثنَي عشرَ انهياراً في التطبيق.**

    **والانهيارُ سطرُه `FATAL EXCEPTION` ويحمل اسمَ الحزمة** — وما عداه
    ضجيجُ نظام.
    """
    out = adb("logcat", "-d", "-b", "crash", t=90)
    hits = []
    for i, line in enumerate(out.splitlines()):
        if "FATAL EXCEPTION" in line:
            window = chr(10).join(out.splitlines()[i:i + 8])
            if pkg in window:
                hits.append(window)
    return hits
