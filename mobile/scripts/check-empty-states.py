"""
══════════════════════════════════════════════════════════════════════
**حارسُ الحالِ الفارغة — ما وقع، وما يفعله المرءُ بعده**
══════════════════════════════════════════════════════════════════════

(شرطُ المالك ٢٠٢٦-٠٩-١٣، دفعةُ ما قبل الإطلاق: «Every empty state must
answer: WHAT HAPPENED? WHAT CAN THE USER DO NEXT?».)

# لماذا وُجد

**والمنصّةُ تُنزَّل قبل أن تمتلئ**: **يُثبِّت الزبونُ التطبيقَ ولا
متجرَ بعد.** **فما يراه في أوّل دقيقةٍ هو كلُّ ما يعرفه عنّا** — وشاشةٌ
بيضاءُ أو سطرٌ ناقصٌ يُقرأ «هذا التطبيقُ لا يعمل».

**وسطرٌ واحدٌ يُصلَح اليومَ يعود ناقصاً بعد شهر** — **ولا يمسكه إلّا
انتباهُ من يكتب، وهو أضعفُ حارس.**

# وما يمسكه

**١ · شاشةُ زبونٍ تعرض قائمةً بلا حالِ فراغٍ أصلاً** — فراغٌ أبيض.

**٢ · وحالُ فراغٍ بلا خطوةٍ تالية** في الشاشات التي لها خطوة.

**٣ · ونصٌّ تقنيٌّ يُعرَض للناس** — رمزُ خطأٍ أو JSON أو لاتينيّةٌ خامّة.

**٤ · ودوّارٌ بلا نهاية** — تحميلٌ لا يقابله فرعُ خطأٍ ولا فرعُ فراغ.

**٥ · وحالُ الإطلاق تُقرأ عطباً** — `launch_closed` يجب أن يُترجَم
حالاً مقصودةً لا انكساراً.

# ولماذا لا يُقاس بالعين

**وأربعُ حالاتٍ لكلّ شاشةٍ تجلب بياناً**: تحميلٌ · خطأٌ بإعادة · فراغٌ ·
محتوى. **ومن نسي واحدةً لم يُلاحظ إلّا حين وقعت على مستعمل.**
"""

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
UI = ROOT / "ui" / "src" / "main" / "kotlin" / "com" / "rahalgo" / "ui"
CUSTOMER = ROOT / "app-customer" / "src" / "main" / "kotlin" / "com" / "rahalgo" / "customer"

problems = []
notes = []


def read(p: Path) -> str:
    try:
        return p.read_text(encoding="utf-8")
    except OSError:
        return ""


def strip_comments(src: str) -> str:
    """**والتعليقُ ليس شيفرة** — ولا يُقاس ما كُتب شرحاً."""
    src = re.sub(r"/\*.*?\*/", "", src, flags=re.S)
    return re.sub(r"//[^\n]*", "", src)


# ── ١ · المكوّنُ المركزيُّ يقبل خطوةً تالية ──────────────────────────
kit = read(UI / "Kit.kt")
if not kit:
    problems.append("`ui/Kit.kt` لم يُقرأ — والحارسُ بلا مرجع")
else:
    for want in ("fun Empty(", "hint: String", "actionLabel: String", "onAction:"):
        if want not in kit:
            problems.append(
                "**مكوّنُ الفراغ المركزيُّ فقد %s** — "
                "**فلا يستطيع أن يقول الخطوةَ التالية**، "
                "**ومن احتاجها كتب مكوّناً ثانياً يفترق.**" % want
            )
    # **وحالُ التحميل والفشل بزرِّ إعادة** — شبكةُ الشارع تنقطع وتعود.
    if "fun LoadState(" not in kit or "onRetry" not in kit:
        problems.append("**حالُ الفشل بلا زرِّ إعادة** — والشبكةُ تنقطع وتعود")
    if not problems:
        notes.append("المكوّنُ المركزيُّ يقول ما وقع وما بعده · والفشلُ بإعادة")

# ── ٢ · وحالُ الإطلاق حالٌ مقصودةٌ لا عطب ────────────────────────────
errs = read(UI / "ApiErrors.kt")
if 'launch_closed' not in errs:
    problems.append(
        "**`launch_closed` غيرُ معروفٍ في مُترجِم الأخطاء** — "
        "**فيُعرَض خامّاً أو يُسجَّل حادثةَ عطب**، وهو حالٌ يضبطها المالك."
    )
else:
    # **ولا يُسجَّل حادثةً** — **مئاتٌ يوميّاً تُغرق السجلَّ فيُفقَد
    # فيه ما يعني شيئاً.**
    block = errs[errs.index('launch_closed'):]
    head = block[:600]
    if "Crash.soft" in head.split("return")[0]:
        problems.append("**حالُ الإطلاق تُسجَّل حادثةَ عطب** — وهي مقصودة")
    # **ونصُّ المالك يغلب نصَّ الحزمة** — ونصٌّ في حزمةٍ لا يُصحَّح إلّا بنشر.
    if 'details["notice"]' not in errs:
        problems.append(
            "**نصُّ المالك لا يُقرأ** — "
            "**ونصٌّ مكتوبٌ في الحزمة لا يُصحَّح إلّا بنشرٍ في المتجر**، "
            "**ويومَ يُفتح البابُ يبقى معروضاً.**"
        )
    if not any("الإطلاق" in p for p in problems):
        notes.append("حالُ الإطلاق مقصودةٌ لا عطب · ونصُّ المالك يغلب")

# ── ٣ · وشاشاتُ الزبون الحاملةُ لبيانٍ لها الحالاتُ الأربع ───────────
#
# **ولا تُفحَص كلُّ شاشة**: **من لا يجلب بياناً لا فراغَ له.** **والمقيسُ
# من ينادي `busy` أو `error` من نموذجه** — فتلك التي تنتظر شبكة.
SCREENS = [
    ("shop/ShopScreen.kt", True),
    ("orders/OrdersScreen.kt", True),
    ("mine/MineScreens.kt", True),
]
for rel, needsAction in SCREENS:
    src = read(CUSTOMER / rel)
    if not src:
        problems.append("شاشةٌ لم تُقرأ: %s" % rel)
        continue
    code = strip_comments(src)
    if "Empty(" not in code:
        problems.append(
            "**%s تعرض قائمةً بلا حالِ فراغ** — "
            "**وشاشةٌ بيضاءُ تُقرأ عطباً.**" % rel
        )
    if "LoadState(" not in code and "error" in code:
        problems.append(
            "**%s بلا حالِ فشلٍ بإعادة** — "
            "**وقائمةٌ فارغةٌ بعد فشلِ قراءةٍ تُقرأ خبراً، والخبرُ كاذب.**" % rel
        )
    # ── ودوّارٌ يقابله فرعان ─────────────────────────────────────
    if "RahalLoader()" in code and "isEmpty()" not in code:
        problems.append(
            "**%s دوّارٌ بلا فرعِ فراغٍ** — **فمن لا بيانَ له ينظر إلى "
            "دورانٍ لا ينتهي.**" % rel
        )

# ── ٤ · والسوقُ الفارغةُ حالٌ بذاتها ────────────────────────────────
#
# **وقِيس ٢٠٢٦-٠٩-١٣**: كانت تقول «لا أصناف في هذا القسم» **وهو لا يرى
# قسماً أصلاً** — فيظنّ أنّه أخطأ الطريقَ أو أنّ التطبيقَ معطوب.
shop = strip_comments(read(CUSTOMER / "shop/ShopScreen.kt"))
if "shop_market_empty" not in shop:
    problems.append(
        "**لا حالَ لسوقٍ فارغةٍ تماماً** — "
        "**والمنصّةُ تُنزَّل قبل أن تمتلئ**، "
        "**و«لا أصناف في هذا القسم» تُقال لمن لا يرى قسماً.**"
    )
elif "sections.isEmpty()" not in shop:
    problems.append(
        "**حالُ السوق الفارغة لا تفرّق السوقَ من القسم** — "
        "فتُقال في غير موضعها."
    )
else:
    notes.append("سوقٌ فارغةٌ حالٌ بذاتها — تُفرَّق عن قسمٍ فارغ")

# ── ٥ · ولا نصَّ تقنيٍّ يُعرَض للناس ────────────────────────────────
#
# **ورمزُ خطأٍ أو JSON على شاشةِ زبونٍ يُقرأ «هذا التطبيق مكسور»** —
# **ولا يفهمه أحدٌ ولا يستطيع فعلَ شيءٍ به.**
TECHNICAL = ("Exception", "stackTrace", "e.toString()", "response.body",
             "printStackTrace", "e.message")
for rel, _ in SCREENS:
    code = strip_comments(read(CUSTOMER / rel))
    for mark in TECHNICAL:
        if re.search(r"Text\([^)]*" + re.escape(mark), code):
            problems.append(
                "**نصٌّ تقنيٌّ يُعرَض في %s**: %s — "
                "**ولا يفهمه أحدٌ ولا يستطيع فعلَ شيءٍ به.**" % (rel, mark)
            )

# ── ٦ · وكلُّ تلميحٍ له نصٌّ في المعجم ──────────────────────────────
strings = read(ROOT / "app-customer" / "src" / "main" / "res" / "values" / "strings.xml")
strings += read(UI / ".." / ".." / ".." / "res" / "values" / "strings.xml")
strings += read(ROOT / "ui" / "src" / "main" / "res" / "values" / "strings.xml")
declared = set(re.findall(r'name="([a-z0-9_]+)"', strings))
used = set()
for rel, _ in SCREENS:
    used |= set(re.findall(r"R\.string\.([a-z0-9_]+)", read(CUSTOMER / rel)))
ghosts = sorted(u for u in used if u not in declared)
if ghosts:
    problems.append(
        "**مفاتيحُ نصٍّ تُنادى ولا وجودَ لها**: %s" % " · ".join(ghosts)
    )
else:
    notes.append("%d مفتاحَ نصٍّ في شاشات الزبون — كلُّها معرَّفة" % len(used))

if problems:
    print("الحالُ الفارغة — خلل:")
    for p in problems:
        print("  x " + p)
    sys.exit(1)
for n in notes:
    print("  - " + n)
print("كلُّ فراغٍ يقول ما وقع وما بعده · ولا نصَّ تقنيٍّ ولا دوّارَ بلا نهاية.")
