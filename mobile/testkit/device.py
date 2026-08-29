# -*- coding: utf-8 -*-
"""
══════════════════════════════════════════════════════════════════════
 الطبقةُ الثالثة — الجهازُ: صيدُ الانهيارات وما لا يظهر في بناء
══════════════════════════════════════════════════════════════════════

**وهذه الطبقةُ وحدَها تمسك ما طلبتَه**: الانهياراتِ والأخطاءَ التي لا
تُكتشف بسهولة.

# أربعةُ أشياءَ لا يراها بناءٌ ولا حارس

**١ · الانهيارُ عند الإقلاع** — كما وقع في المتجر ٢٠٢٦-٠٨-٢٦:
`The Crashlytics build ID is missing`. **والانهيارُ في مزوّد محتوى فلا
شاشةَ خطأٍ ولا سطرَ سبب** — يُغلق التطبيقُ صامتاً، **ويظنّ صاحبُه أنّه
لم يفتح.**

**٢ · التجميدُ (ANR)** — «التطبيق لا يستجيب». **وهو أسوأُ من انهيار**:
لا يُسجَّل في Crashlytics غالباً، **والمستخدمُ يظنّ جهازَه بطيئاً.**

**٣ · تسريبُ الذاكرة** — يظهر بعد نصف ساعةٍ من الاستعمال لا بعد دقيقة.
**فيُقاس قبلَ العبث وبعدَه.**

**٤ · ما يكسره الضغطُ العشوائيّ** — `monkey` يضرب آلافَ الضغطات في
دقائق. **ويمرّ على شاشاتٍ لا يفتحها مختبِرٌ بشريٌّ أبداً.**

# ولماذا لا يُشغَّل هذا تلقائيّاً كلَّ ليلة

**يحتاج جهازاً موصولاً** — والتصحيحُ اللاسلكيّ يحتاج إذنَ صاحبِه في كلّ
جلسة. **فيُشغَّل حين يكون الجهازُ حاضراً**، وتبقى الطبقتان الأخريان
تعملان بلا جهاز.
"""

import re
import subprocess
import time

PKGS = {
    "app-customer": "com.rahalgo.customer",
    "app-driver": "com.rahalgo.driver",
    "app-merchant": "com.rahalgo.merchant",
    "app-rep": "com.rahalgo.rep",
}

# **أنماطُ العطب في السجلّ** — كلُّ واحدٍ منها شكوى مستخدمٍ لم تُكتب بعد.
FATAL = [
    (r"FATAL EXCEPTION", u"انهيارٌ كامل"),
    (r"ANR in ", u"تجميدٌ — التطبيق لا يستجيب"),
    (r"StrictMode policy violation", u"عملُ قرصٍ أو شبكةٍ على الخيط الرئيس"),
    (r"E AndroidRuntime", u"استثناءٌ غيرُ ملتقَط"),
    (r"Compose Runtime.*Exception", u"استثناءٌ في رسم الشاشة"),
    (r"^E.*(Row too big|CursorWindow)", u"صفٌّ أكبرُ من نافذة القاعدة"),
    (r"Skipped \d{3,} frames", u"إطاراتٌ مفقودةٌ بالمئات — الخيطُ الرئيس مشغول"),
    (r"OutOfMemoryError", u"نفدت الذاكرة"),
    (r"NetworkOnMainThreadException", u"نداءُ شبكةٍ على الخيط الرئيس"),
]


def sh(args, timeout=120):
    try:
        r = subprocess.run(args, capture_output=True, timeout=timeout)
        return r.returncode, r.stdout.decode("utf-8", "replace"), r.stderr.decode("utf-8", "replace")
    except Exception as e:
        return -1, "", str(e)


def serial():
    """**رقمُ الجهاز الموصول** — أو لا شيء."""
    code, out, _e = sh(["adb", "devices"])
    if code != 0:
        return None
    for line in out.split(chr(10))[1:]:
        parts = line.split()
        if len(parts) >= 2 and parts[1] == "device":
            return parts[0]
    return None


def adb(dev, *args, **kw):
    return sh(["adb", "-s", dev] + list(args), **kw)


def mem_kb(dev, pkg):
    """**ذاكرةُ العمليّة بالكيلوبايت** — أو صفرٌ إن لم تكن حيّة."""
    _c, out, _e = adb(dev, "shell", "dumpsys", "meminfo", pkg)
    m = re.search(r"TOTAL(?:\s+PSS)?:?\s+(\d+)", out)
    return int(m.group(1)) if m else 0


def alive(dev, pkg):
    _c, out, _e = adb(dev, "shell", "pidof", pkg)
    return out.strip() != ""


def awake(dev):
    """
    **أمستيقظٌ الجهاز؟** — وإلّا فكلُّ نتيجةٍ بعده كذب.

    **وقِيس ٢٠٢٦-٠٨-٢٧**: نامت الشاشةُ في منتصف التشغيلة، **فمات
    تطبيقان بعد الإقلاع بستّ ثوانٍ** — ولا انهيارَ فيهما ولا سطرَ
    خطأ. **حارسٌ يتّهم بريئاً أسوأُ من حارسٍ لا يرى.**
    """
    _c, out, _e = adb(dev, "shell", "dumpsys", "power")
    m = re.search(r"mWakefulness=(\w+)", out)
    return (m.group(1) if m else "") == "Awake"


def wake(dev):
    """**يوقظ الشاشة** — ولا يفتح قفلاً ولا يلمس بيانات."""
    adb(dev, "shell", "input", "keyevent", "KEYCODE_WAKEUP")
    time.sleep(2)
    return awake(dev)


def smoke(dev, module, pkg, fuzz_events=600):
    """
    **يفتح التطبيقَ · يعبث به · يقيس** — ويعيد قائمةَ مخالفات.

    **والسجلُّ يُمحى قبل الفتح** — وإلّا اختلط انهيارُ أمسٍ بانهيار
    اليوم، **ومن قرأه ظنّ عطباً أصلحه عاد.**
    """
    bad = []
    # **ولا يُقاس على شاشةٍ نائمة** — انظر `awake`.
    if not awake(dev) and not wake(dev):
        return [(module, u"الجهازُ نائم — لا تُقاس نتيجةٌ عليه")]
    adb(dev, "logcat", "-c")
    adb(dev, "shell", "am", "force-stop", pkg)

    # ── الإقلاع ─────────────────────────────────────────────────────
    # ══════════════════════════════════════════════════════════════
    # **والإقلاعُ بـ`am start` لا بـ`monkey`**
    # ══════════════════════════════════════════════════════════════
    #
    # **قِيس على جهاز سامسونغ ٢٠٢٦-٠٨-٢٧**: `monkey -c LAUNCHER 1`
    # يخرج بنجاحٍ ولا يُقلع شيئاً — **فتُقرأ ثلاثةُ تطبيقاتٍ «ماتت بعد
    # الإقلاع» وهي لم تُقلع أصلاً**، وصفرُ انهيارٍ في السجلّ.
    #
    # **و`am start` يقول ما فعل**: يطبع `Starting: Intent` أو خطأً
    # صريحاً. **وأداةٌ تكذب بصمتٍ أخطرُ من أداةٍ تعطب.**
    _c, resolved, _e = adb(dev, "shell", "cmd", "package", "resolve-activity",
                           "--brief", pkg)
    comp = ""
    for line in resolved.split(chr(10)):
        line = line.strip()
        if line.startswith(pkg + "/"):
            comp = line
            break
    t0 = time.time()
    if comp:
        _c, out, _e = adb(dev, "shell", "am", "start", "-W", "-n", comp)
    else:
        _c, out, _e = adb(dev, "shell", "monkey", "-p", pkg,
                          "-c", "android.intent.category.LAUNCHER", "1")
    if "No activities found" in out or "Error:" in out:
        return [(module, u"لا نشاطَ يُفتح · " + out.strip()[:90])]

    # **والزمنُ يُقرأ قبل الانتظار لا بعده** — كان `sleep(6)` داخلَ
    # القياس، **فكلُّ تطبيقٍ يُقرأ أبطأَ بستّ ثوانٍ ممّا هو** (قِيس
    # ٢٠٢٦-٠٨-٢٧: تسعُ ثوانٍ لتطبيقٍ يُقلع في ثلاث).
    #
    # **و`am start -W` يعطي رقمَه بنفسه** — يُؤخذ منه إن وُجد، وإلّا
    # فساعتُنا.
    boot_ms = int((time.time() - t0) * 1000)
    m = re.search(r"(?:TotalTime|WaitTime):\s*(\d+)", out)
    if m:
        boot_ms = int(m.group(1))

    # **ثمّ يُترك ليستقرّ** — الانهيارُ يقع بعد أوّل إطارٍ لا قبله.
    time.sleep(6)

    if not alive(dev, pkg):
        _c, log, _e = adb(dev, "logcat", "-d", "-v", "brief")
        tail = [l for l in log.split(chr(10)) if "AndroidRuntime" in l or "FATAL" in l]
        bad.append((module, u"مات بعد الإقلاع بستّ ثوانٍ · " + (tail[-1][:120] if tail else u"بلا سطرِ سبب")))
        return bad

    if boot_ms > 5000:
        bad.append((module, u"الإقلاعُ %d ملّي" % boot_ms))

    mem_before = mem_kb(dev, pkg)

    # ── العبثُ العشوائيّ ────────────────────────────────────────────
    #
    # **والبذرةُ ثابتة** — فالانهيارُ الذي يقع اليومَ يقع غداً بالضغطات
    # نفسِها، **وبلا ثباتٍ لا يُعاد عطبٌ ولا يُثبَت إصلاح.**
    adb(dev, "shell", "monkey", "-p", pkg, "-s", "20260826",
        "--throttle", "120", "--pct-syskeys", "0", "--ignore-timeouts",
        str(fuzz_events), timeout=420)
    time.sleep(3)

    mem_after = mem_kb(dev, pkg)
    still = alive(dev, pkg)
    if not still and not awake(dev):
        # **نامت الشاشةُ أثناء العبث** — والنظامُ يُنهي ما في المقدّمة.
        return [(module, u"نامت الشاشةُ أثناء الفحص — أعد التشغيل والشاشةُ مضاءة")]

    # ── قراءةُ السجلّ ───────────────────────────────────────────────
    # ══════════════════════════════════════════════════════════════
    # **والسجلُّ يُصفّى إلى عمليّتنا وحدَها**
    # ══════════════════════════════════════════════════════════════
    #
    # **الجهازُ يكتب لكلّ ما يعمل فيه** — و`system_server` يبثّ تحذيراتِ
    # قواعدَ ووسائطَ طولَ الوقت. **وقِيس ٢٠٢٦-٠٨-٢٧**: سطرُ
    # `A resource failed to call release … CursorWindow` من العمليّة
    # ١٦٨٨ حُسب علينا، **وليس لنا.**
    #
    # **وحارسٌ يصرخ لما ليس منه يُطفأ بعد ثالث مرّة** — فالتصفيةُ شرطُ
    # بقائه.
    _c, pid_out, _e = adb(dev, "shell", "pidof", pkg)
    pids = set(pid_out.split())
    _c, log, _e = adb(dev, "logcat", "-d", "-v", "brief")
    lines = log.split(chr(10))
    if pids:
        mine = []
        for l in lines:
            m = re.search(r"\(\s*(\d+)\)", l)
            # **وما لا رقمَ فيه يبقى** — الانهيارُ يُطبع أحياناً بلا رقم.
            if m is None or m.group(1) in pids:
                mine.append(l)
        lines = mine
    for pat, what in FATAL:
        hits = [l for l in lines if re.search(pat, l)]
        if hits:
            bad.append((module, u"%s (%d مرّة) · %s" % (what, len(hits), hits[0][:110])))

    if not still:
        bad.append((module, u"سقط أثناء العبث — %d ضغطة" % fuzz_events))

    if mem_before and mem_after > mem_before * 2.5 and mem_after > 300000:
        bad.append((module, u"الذاكرةُ من %d إلى %d ك.ب — تسريبٌ محتمل"
                    % (mem_before, mem_after)))

    adb(dev, "shell", "am", "force-stop", pkg)
    return bad


def run(fuzz_events=600, only=None):
    dev = serial()
    if not dev:
        return None, [(u"—", u"لا جهازَ موصول")]
    installed = set()
    _c, out, _e = adb(dev, "shell", "pm", "list", "packages")
    for line in out.split(chr(10)):
        installed.add(line.replace("package:", "").strip())

    results = []
    for module, pkg in PKGS.items():
        if only and module not in only:
            continue
        if pkg not in installed:
            results.append((module, [(module, u"غيرُ مثبَّتٍ على الجهاز")]))
            continue
        results.append((module, smoke(dev, module, pkg, fuzz_events)))
    return dev, results
