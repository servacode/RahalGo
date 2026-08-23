# -*- coding: utf-8 -*-
"""
══════════════════════════════════════════════════════════════════════
**حارسُ الإقلاع — يبني الإصدارَ ويُقلعه على جهازٍ حقيقيّ**
══════════════════════════════════════════════════════════════════════

(قِيس ٢٠٢٦-٠٨-٢٣، وسؤالُ المالك: «هل التطبيقُ جاهزٌ للانطلاق؟»)

# ولماذا وُجد

**بناءُ الإصدار كان ينهار قبل أن تُرسم شاشةٌ واحدة:**

    FATAL EXCEPTION: main
    NoSuchMethodException: androidx.work.impl.WorkDatabase_Impl.<init>

**حذف R8 بانيَ قاعدةِ بيانات `WorkManager`** — لأنّه لا يُنادى من
شيفرتنا بل بالانعكاس. **والمُشذِّبُ يقرأ النداءاتِ المكتوبةَ ولا يقرأ
الانعكاس.**

# والفجوةُ التي كشفها

**خمسُ مئةٍ وستّةٌ وسبعون اختباراً خضراء، وكلُّها على بناء التطوير.**
**ولا حارسَ واحدٌ يفتح ما يُرفع فعلاً.**

**وهذه أخطرُ عائلةِ عطبٍ في أندرويد**: لا تظهر في التطوير إطلاقاً،
**وتظهر في يد أوّل مستخدمٍ ينزّل من المتجر.**

# وماذا يفعل بالضبط

    ١ · يبني بناءَ الإصدار
    ٢ · يثبّته على جهازٍ موصول
    ٣ · يُقلعه ويصبر
    ٤ · **يسأل: أما زال حيّاً؟** — وأيّ انهيارٍ في السجلّ يُسقطه

**ولا يكفي أن يُبنى** — **حزمةٌ تُبنى ولا تُقلع تمرّ من كلّ فحصٍ
مكتوب.**

# ويحتاج جهازاً — وذاك مقصود

**ولا يُشغَّل في كلّ التزام** — يُشغَّل قبل الرفع. **وحارسٌ يحتاج
جهازاً لا يصلح بوّابةً آليّة**، لكنّه **يصلح شرطاً قبل النشر.**

    python mobile/scripts/check-release-boots.py app-driver
"""
import re
import subprocess
import sys
import time

sys.stdout.reconfigure(encoding="utf-8")

# **وثوانٍ يُنتظر فيها** — الإقلاعُ الباردُ مع خريطةٍ ليس لحظيّا.
SETTLE_SEC = 25


# **وأدواتُ النظام تُسمّى بلاحقتها على ويندوز** — **و`adb` وحدَها
# تُقرأ على لينكس ولا تُوجد هنا.**
ADB = "adb.exe" if sys.platform == "win32" else "adb"
GRADLE = "mobile\gradlew.bat" if sys.platform == "win32" else "mobile/gradlew"


def sh(args, timeout=2400):
    return subprocess.run(args, capture_output=True, text=True,
                          encoding="utf-8", errors="replace", timeout=timeout,
                          shell=(sys.platform == "win32"))


def device() -> str:
    """**أوّلُ جهازٍ متّصل** — ولا يُخمَّن اسمُه."""
    out = sh([ADB, "devices"], timeout=120).stdout
    for line in out.splitlines()[1:]:
        if "\tdevice" in line:
            return line.split("\t")[0]
    return ""


def app_id(module: str) -> str:
    """**هويّةُ الحزمة من المصدر لا من اسمٍ على الجهاز.**

    (حادثةُ ٢٠٢٦-٠٨-٢٢: استُنتجت من الجهاز فكانت غيرَها.)
    """
    txt = open(f"mobile/{module}/build.gradle.kts", encoding="utf-8").read()
    m = re.search(r'applicationId\s*=\s*"([^"]+)"', txt)
    if not m:
        raise SystemExit(f"لا `applicationId` في {module}")
    return m.group(1)


def main() -> int:
    module = sys.argv[1] if len(sys.argv) > 1 else "app-driver"
    pkg = app_id(module)
    print(f"الوحدة: {module} · الحزمة: {pkg}")

    print("-- بناءُ الإصدار")
    r = sh([GRADLE, "-p", "mobile", f":{module}:assembleRelease"])
    if r.returncode != 0:
        print((r.stdout or "")[-1500:])
        print((r.stderr or "")[-600:])
        print("✘ لم يُبنَ")
        return 1

    apk = f"mobile/{module}/build/outputs/apk/release/{module}-release.apk"
    dev = device()
    if not dev:
        # **ولا يُقال «نجح» بلا جهاز** — **وحارسٌ يمرّ بلا أن يفحص
        # أخطرُ من لا حارس.**
        print("✘ لا جهازَ موصول — البناءُ وحدَه لا يُثبت الإقلاع")
        return 2

    print(f"-- التثبيت على {dev}")
    r = sh([ADB, "-s", dev, "install", "-r", apk], timeout=1200)
    if "Success" not in r.stdout:
        print(r.stdout[-800:], r.stderr[-400:])
        print("✘ لم يُثبَّت")
        return 1

    sh([ADB, "-s", dev, "shell", "am", "force-stop", pkg], timeout=180)
    sh([ADB, "-s", dev, "logcat", "-c"], timeout=180)
    sh([ADB, "-s", dev, "shell", "monkey", "-p", pkg,
        "-c", "android.intent.category.LAUNCHER", "1"], timeout=300)
    print(f"-- يُصبر {SETTLE_SEC} ثانية")
    time.sleep(SETTLE_SEC)

    pid = sh([ADB, "-s", dev, "shell", "pidof", pkg], timeout=180).stdout.strip()
    log = sh([ADB, "-s", dev, "logcat", "-d", "-t", "3000"], timeout=300).stdout
    crashed = [l for l in log.splitlines()
               if "FATAL EXCEPTION" in l or f"Process: {pkg}," in l]

    if crashed:
        print("\n".join(crashed[:6]))
        print("✘ انهار بعد الإقلاع")
        return 1
    if not pid:
        print("✘ لم يبقَ حيّاً — لا عمليّةَ باسمه")
        return 1

    print(f"✔ أقلع وبقي حيّاً — pid {pid}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
