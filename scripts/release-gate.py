# -*- coding: utf-8 -*-
"""**بوّابةُ الإصدار — أمرٌ واحدٌ يقول: تُنشر أو تُحجب.**

(قرارُ المالك ٢٠٢٦-٠٨-١٩: «أريد أمراً أو Pipeline واضحاً… وفي النهاية
 يصدر RELEASE GATE PASS أو RELEASE BLOCKED».)

# ولماذا بايثون لا سكربت صدفة

**العربيّةُ لا تمرّ في سطر أوامرِ هذا الجهاز** (CLAUDE.md)، **وPowerShell
تُفسد أيَّ ملفٍّ فيه عربيّة.** فالتقريرُ يُكتب من بايثون بترميزٍ صريح.

# والحجبُ بالخطورة لا بالعدد

**اختبارٌ أحمرُ لا يعني الحجبَ دائما**: `OBS-*` ملاحظاتٌ مسجَّلةٌ حمراءَ
عمداً حتّى تُصلَح — **وهي لا تحجب.** وما يحجب مكتوبٌ في
`docs/qa/RELEASE_GATES.md` ولا يُقرَّر هنا بالمزاج.

    python scripts/release-gate.py            # الحزمةُ كاملة
    python scripts/release-gate.py --smoke    # السريعةُ بعد كلّ تعديل
"""
import json
import os
import re
import subprocess
import sys
import time

# ══════════════════════════════════════════════════════════════════════
# **وأمرُ Gradle بمسارِه الكامل**
# ══════════════════════════════════════════════════════════════════════
#
# (وقع ٢٠٢٦-٠٨-١٩ مرّتين: أوّلاً «'.' is not recognized» لأنّ `cmd` لا
#  تعرف `./`، **ثمّ «'gradlew.bat' is not recognized» وهي موجودةٌ في
#  مجلّد العمل** — لأنّ `cmd` لا تبحث في مجلّد العمل حين يُضبط
#  `NoDefaultCurrentDirectoryInExePath`.)
#
# **والعطبان قُرئا سقوطَ اختبارات** — وهما عطبُ أمرٍ لا عطبُ تطبيق.
# **فالمسارُ كاملٌ ولا يُترك للصدفة أن تجده.**
_ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
GRADLE = os.path.join(_ROOT, "mobile", "gradlew.bat" if os.name == "nt" else "gradlew")
GRADLE = '"%s"' % GRADLE

sys.stdout.reconfigure(encoding="utf-8")

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
DB = os.environ.get(
    "TEST_DATABASE_URL",
    "postgres://rahalgo:rahalgo_dev@localhost:5434/rahalgo_test?sslmode=disable",
)

# **ما يحجب الإصدار** — والقائمةُ صريحةٌ لا مُخمَّنةٌ من الأسماء.
BLOCKING_PREFIXES = ("AUTH-", "SEC-", "FIN-", "VAL-", "IDEM-", "E2E-", "LIFE-")
# **وما يُسجَّل ولا يحجب** — ملاحظاتٌ حمراءُ عمداً.
OBSERVATION_PREFIXES = ("OBS-",)


class Stage:
    def __init__(self, name, cmd, cwd, blocking=True, smoke=False):
        self.name, self.cmd, self.cwd = name, cmd, cwd
        self.blocking, self.smoke = blocking, smoke
        self.code, self.out, self.secs = None, "", 0.0

    def run(self):
        t0 = time.time()
        p = subprocess.run(
            self.cmd, cwd=os.path.join(ROOT, self.cwd), shell=True,
            capture_output=True,
            env={**os.environ, "TEST_DATABASE_URL": DB, },
        )
        self.secs = time.time() - t0
        self.out = (p.stdout + p.stderr).decode("utf-8", "replace")
        self.code = p.returncode
        return self


STAGES = [
    Stage("البناء · المحرّك", "go build ./...", "backend", smoke=True),
    Stage("التنسيق", "gofmt -l internal/ cmd/", "backend", blocking=False, smoke=True),
    Stage("الوحدة والتكامل · المحرّك", "go test -count=1 -p 1 ./internal/... ./cmd/...",
          "backend", smoke=False),
    Stage("منظومة QA · API وأمنٌ ومنعُ تكرار",
          "go test -count=1 -v ./internal/qa/", "backend", smoke=True),
    # ══════════════════════════════════════════════════════════════════
    # **ولا `moneycheck` ولا `mediacheck` هنا**
    # ══════════════════════════════════════════════════════════════════
    #
    # **هما تدقيقُ بياناتِ إنتاجٍ لا اختبارُ شيفرة**: يقرآن دفتراً
    # حقيقيّاً ووسائطَ حقيقيّة. **وتشغيلُهما على قاعدة الاختبار يعطي
    # «خللٌ في ٧ من ١٣» دائماً** — وهو صحيحٌ ولا معنى له.
    #
    # **وحارسٌ يُنذر في كلّ تشغيلٍ يُتجاهَل بعد أسبوع** — ثمّ يُتجاهَل
    # معه الإنذارُ الحقيقيّ. فيُشغَّلان على الإنتاج بأمرِهما:
    #
    #     cd backend && DATABASE_URL=<الإنتاج> go run ./cmd/moneycheck
    Stage("حرّاسُ الويب", "pnpm check:guards", "web", smoke=True),
    Stage("الأنواع", "pnpm typecheck", "web", blocking=False),
    Stage("وحدةُ أندرويد", GRADLE + " testDebugUnitTest --console=plain",
          "mobile", smoke=True),
    # ══════════════════════════════════════════════════════════════════
    # **وواجهةُ أندرويد — تحتاج جهازاً متّصلا**
    # ══════════════════════════════════════════════════════════════════
    #
    # **وتسقط بلا جهازٍ فتُقرأ عطباً في الشيفرة** — فليست حاجبة،
    # **وغيابُ جهازٍ ليس عطباً في التطبيق.** والمرحلةُ تُطبع ▲ ويُقرأ
    # سببُها. **ومن أراد حجبَها فليجعلها `blocking=True` في مزرعةِ
    # أجهزة.**
    #
    # **والشاشةُ تبقى مستيقظة**: النشاطُ المضيف يموت إن أُقفلت، فيسقط
    # الاختبارُ بـ`Activity has been destroyed` — **وهو عطبُ بيئةٍ
    # يُقرأ عطبَ واجهة.** (وقع ٢٠٢٦-٠٨-١٩.)
    # **والأداءُ يُقاس ولا يحجب** — تراجعٌ يُقرأ ويُقرَّر، **وعتبةٌ
    # تحجب الإصدارَ على ضجيجِ هاتفٍ تُطفأ بعد أسبوع.**
    Stage("الأداءُ المرجعيّ", "python scripts/perf-baseline.py", ".",
          blocking=False, smoke=False),
    Stage("واجهةُ أندرويد · على جهاز",
          "adb shell svc power stayon usb && " + GRADLE + " connectedDebugAndroidTest --console=plain",
          "mobile", blocking=False, smoke=False),
]


GRADLE_CASE = re.compile(r"^(\S+) > (\S+)\[?.*?\]? (FAILED|PASSED)", re.M)


def parse_gradle(out):
    """**حالاتُ Gradle من سطرِ النتيجة** — وصيغتُها غيرُ صيغةِ Go."""
    return [(m.group(2), m.group(3).replace("PASSED", "PASS").replace("FAILED", "FAIL"))
            for m in GRADLE_CASE.finditer(out)]


def parse_go(out):
    """**يستخرج كلَّ حالةٍ من مخرجات `go test -v`.**"""
    cases = []
    for line in out.splitlines():
        m = re.match(r"\s*--- (PASS|FAIL|SKIP): (\S+)", line)
        if m:
            cases.append((m.group(2).split("/")[-1], m.group(1)))
    return cases


def ident(name):
    """**المعرّفُ من اسم الحالة** — `TestIDEM_001_x` صار `IDEM-001`.

    **والعائلةُ تُقرأ من البادئة لا من الرقم**: `TestOBS_Foreign…` بلا
    رقمٍ يليها، **ومن اشترط الرقمَ صنّفها حاجبةً** — وهو ما وقع في أوّل
    تشغيل.
    """
    fam = re.match(r"Test([A-Z]+)", name)
    num = re.search(r"(?:IDOR[_-]?)?(\d{3})", name)
    if not fam:
        return name
    return "%s-%s" % (fam.group(1), num.group(1) if num else "…")


def main():
    smoke = "--smoke" in sys.argv
    stages = [s for s in STAGES if s.smoke] if smoke else STAGES

    print("═" * 66)
    print("  رحّال غو — %s" % ("الحزمةُ السريعة (SMOKE)" if smoke else "تحقّقُ الإصدار"))
    print("═" * 66)

    blocked, notes, totals = [], [], {"PASS": 0, "FAIL": 0, "SKIP": 0}
    for st in stages:
        st.run()
        cases = parse_go(st.out) or parse_gradle(st.out)
        for name, verdict in cases:
            totals[verdict] = totals.get(verdict, 0) + 1
            if verdict != "FAIL":
                continue
            i = ident(name)
            if i.startswith(OBSERVATION_PREFIXES):
                notes.append(i)
            elif i.startswith(BLOCKING_PREFIXES) or st.blocking:
                blocked.append(i)

        ok = st.code == 0
        # **ومرحلةٌ ساقطةٌ بلا حالاتٍ مسمّاةٍ تُحجب كلُّها** — سقوطُ بناءٍ
        # لا يُقرأ حالةَ اختبار.
        if not ok and not cases and st.blocking:
            blocked.append(st.name)
        mark = "✔" if ok else ("✘" if st.blocking else "▲")
        extra = ""
        if cases:
            p = sum(1 for _, v in cases if v == "PASS")
            extra = " · %d/%d" % (p, len(cases))
        print("  %s %-40s %6.1fث%s" % (mark, st.name, st.secs, extra))
        if not ok and not cases:
            for line in st.out.strip().splitlines()[-4:]:
                print("      │ " + line[:96])

    print("─" * 66)
    print("  الحالات: %d ناجحة · %d ساقطة · %d متخطّاة"
          % (totals["PASS"], totals["FAIL"], totals["SKIP"]))
    if notes:
        print("  ملاحظاتٌ مسجَّلة (لا تحجب): " + " · ".join(sorted(set(notes))))
    print("═" * 66)

    report = {
        "smoke": smoke, "totals": totals,
        "blocking_failures": sorted(set(blocked)),
        "observations": sorted(set(notes)),
        "stages": [{"name": s.name, "code": s.code, "seconds": round(s.secs, 1)}
                   for s in stages],
    }
    out = os.path.join(ROOT, "docs", "qa", "last-run.json")
    os.makedirs(os.path.dirname(out), exist_ok=True)
    with open(out, "w", encoding="utf-8") as f:
        json.dump(report, f, ensure_ascii=False, indent=2)

    if blocked:
        print("  ‼ الإصدارُ محجوب — %s" % " · ".join(sorted(set(blocked))))
        print("═" * 66)
        return 1
    print("  ✔ الإصدارُ مسموح")
    print("═" * 66)
    return 0


if __name__ == "__main__":
    sys.exit(main())
