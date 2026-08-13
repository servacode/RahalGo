"""
══════════════════════════════════════════════════════════════════════
**حارسُ نصوص التطبيق — يُسقط البناءَ على نصٍّ خارج المعجم**
══════════════════════════════════════════════════════════════════════

(قرارُ المالك ٢٠٢٦-٠٨-١٣: «نعم أصلحها».)

# لماذا وُجد

**الويبُ فيه ثمانيةُ حرّاسٍ تُسقط البناء، والتطبيقُ بلا واحد.** فما
يُنظَّف اليوم يعود بعد شهر — **ولا يمسكه إلّا انتباهُ من يكتب**، وهو
أضعفُ حارس.

# وما يمسكه

**١ · نصٌّ عربيٌّ مكتوبٌ في شيفرة كوتلن** — لا في `strings.xml`.
فيبقى بلا ترجمةٍ يومَ تُضاف لغة، **ويُبدَّل في موضعٍ ويُنسى في موضعين.**

**٢ · مفتاحٌ يُنادى ولا وجودَ له** — يُسقط البناءَ أصلاً، **لكنّه
يُمسك هنا برسالةٍ تُقرأ.**

**٣ · مفتاحٌ ميّتٌ في المعجم** — نصٌّ لا تقوله شاشة. **ومن أراد تبديلَ
كلمةٍ يجده فيعدّله ولا يتغيّر شيء.**

# وما لا يمسكه عمدا

**سجلّاتُ `Log`** — تُقرأ في `adb` لا في شاشة، **والعربيّةُ فيها أوضحُ
لمن يقرؤها.** **والتعليقاتُ** — كلُّها عربيّةٌ بقاعدة المشروع.
"""

import re
import sys
from pathlib import Path

# **والمخرجُ عربيٌّ في طرفيّةٍ لاتينيّة** — صدفةُ ويندوز تكتب بـ`cp1256`
# فتسقط الرسالةُ نفسُها قبل أن تُقرأ. **وحارسٌ يسقط بترميزٍ لا بمخالفة**
# يُعطَّل بعد أوّل مرّة.
sys.stdout.reconfigure(encoding="utf-8")

ROOT = Path(__file__).resolve().parent.parent
APP = ROOT / "app-driver" / "src" / "main"
STRINGS = APP / "res" / "values" / "strings.xml"

ARABIC = re.compile(r"[؀-ۿ]")

# ══════════════════════════════════════════════════════════════════════
# **ومصدرُ الوحدات ملفٌّ واحدٌ معروف**
# ══════════════════════════════════════════════════════════════════════
#
# **«ل.س» و«كم» و«د» لواحقُ قياسٍ لا جملٌ تُترجَم** — وموضعُها
# `ui/Money.kt` وحدَه. **وهو نظيرُ `SOURCES` في حارس الويب**: اللونُ
# يُعرَّف في `theme.css` ويُمنع في غيره.
#
# **ولو مُنعت هنا أيضاً لَما بقي لها موضع** — أو لصارت في `strings.xml`
# بأسماءٍ تُنادى من دالّةٍ بلا `Context`، **فتُمرَّر عبر خمسِ طبقات.**
SOURCES = {"app-driver/src/main/kotlin/com/rahalgo/driver/ui/Money.kt"}
# **ما يُتخطّى**: سطرُ سجلٍّ، أو تعليق، أو وحدةُ قياسٍ في نصٍّ مُنسَّق.
SKIP_LINE = re.compile(r"^\s*(//|\*|/\*)|Log\.[a-z]")
LOG_CALL = re.compile(r"Log\.[a-z]+\(")


def kotlin_files():
    return sorted((APP / "kotlin").rglob("*.kt"))


def main() -> int:
    xml = STRINGS.read_text(encoding="utf-8")
    have = set(re.findall(r'<string name="(\w+)"', xml))

    used: set[str] = set()
    inline: list[tuple[str, int, str]] = []

    for f in kotlin_files():
        rel = f.relative_to(ROOT).as_posix()
        # ══════════════════════════════════════════════════════════════
        # **ونداءُ السجلّ يمتدّ أسطرا**
        # ══════════════════════════════════════════════════════════════
        #
        # `Log.i(` ثمّ `TAG,` ثمّ النصُّ في سطرٍ ثالث. **وحارسٌ يقرأ
        # سطراً سطراً يرى النصَّ وحدَه** فيشكو من سجلٍّ لا يقرؤه إلّا
        # من يفتح `adb`.
        #
        # **فتُعدّ الأقواسُ حتّى يُغلق النداء.**
        depth = 0
        for i, line in enumerate(f.read_text(encoding="utf-8").split("\n"), 1):
            used.update(re.findall(r"R\.string\.(\w+)", line))
            if depth > 0:
                depth += line.count("(") - line.count(")")
                continue
            if LOG_CALL.search(line):
                depth = line.count("(") - line.count(")")
            if SKIP_LINE.search(line) or rel in SOURCES:
                continue
            for m in re.finditer(r'"([^"\\]*)"', line):
                if ARABIC.search(m.group(1)):
                    inline.append((rel, i, m.group(1)[:40]))

    problems = 0

    missing = sorted(used - have)
    if missing:
        problems += len(missing)
        print("── مفاتيحُ تُنادى ولا وجودَ لها في strings.xml ──")
        for k in missing:
            print("   ", k)

    if inline:
        problems += len(inline)
        print("\n── نصٌّ عربيٌّ في شيفرة كوتلن — موضعُه strings.xml ──")
        for rel, i, t in inline:
            print(f"   {rel}:{i}  {t}")

    # **و`app_name` يُنادى من المخطّطة لا من الشيفرة** — `@string/app_name`
    # في `AndroidManifest.xml`. **وحارسٌ يشكو منه يُعلَّم أن يُتجاهَل.**
    dead = sorted(have - used - {"app_name"})
    if dead:
        problems += len(dead)
        print("\n── مفاتيحُ لا تقولها شاشة ──")
        for k in dead:
            print("   ", k)

    if problems:
        print(f"\nنصوصُ التطبيق: {problems} مخالفة")
        return 1
    print("نصوصُ التطبيق سليمة — لا نصَّ خارج المعجم ولا مفتاحَ ميّت.")
    return 0


if __name__ == "__main__":
    sys.exit(main())
