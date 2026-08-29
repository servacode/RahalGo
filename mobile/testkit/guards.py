# -*- coding: utf-8 -*-
"""
══════════════════════════════════════════════════════════════════════
 الطبقةُ الأولى — حرّاسٌ ساكنون على شيفرة التطبيقات الأربعة
══════════════════════════════════════════════════════════════════════

**كلُّ حارسٍ هنا عطبٌ وقع فعلاً** — لا قاعدةٌ عامّةٌ ولا ذوق. ومن أراد
أن يعرف لماذا وُجد حارسٌ فليقرأ سببَه المكتوبَ معه.

**ولماذا حرّاسٌ ساكنون قبل الاختبار الحيّ:**

**الاختبارُ الحيُّ يكشف ما مررتَ عليه** — والحارسُ يكشف ما لم تفتحه
أصلاً. **وتطبيقُ المندوب مشى أشهراً بلا إشعاراتٍ ولم يسقط اختبارٌ
واحد**: لم يكن ثمّة ما يطرق البابَ ليجده مغلقاً.

**وهذه المنظومة تبقى وتُحدَّث** — كلُّ عطبٍ يُكتشف يصير حارساً، فلا
يعود مرّتين.
"""

import io
import os
import re
import json

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
APPS = ["app-customer", "app-driver", "app-merchant", "app-rep"]

AR = re.compile(u"[؀-ۿ]")
HEX_COLOR = re.compile(r"Color\(0x[0-9A-Fa-f]{8}\)")
QUOTED = re.compile(r'"([^"\\]*)"')
LOGLINE = re.compile(r"\bLog\.[wide]\(|android\.util\.Log")
HOST = re.compile(r"https://\w+\.rahalgo\.com")


def read(p):
    return io.open(p, encoding="utf-8").read()


def kt_files(module, where="src/main"):
    base = os.path.join(ROOT, module, where)
    for dp, _, fs in os.walk(base):
        for f in fs:
            if f.endswith(".kt"):
                yield os.path.join(dp, f)


def strings_of(module):
    p = os.path.join(ROOT, module, "src/main/res/values/strings.xml")
    if not os.path.exists(p):
        return {}
    return dict(
        (m.group(1), m.group(2).strip())
        for m in re.finditer(r'<string name="([^"]+)"[^>]*>(.*?)</string>', read(p), re.S)
    )


def manifest_of(module):
    p = os.path.join(ROOT, module, "src/main/AndroidManifest.xml")
    return read(p) if os.path.exists(p) else ""


def gradle_of(module):
    p = os.path.join(ROOT, module, "build.gradle.kts")
    return read(p) if os.path.exists(p) else ""


def rel(p):
    return p.replace(ROOT + os.sep, "").replace(os.sep, "/")


def is_comment(line):
    s = line.strip()
    return s.startswith("//") or s.startswith("*") or s.startswith("/*")


# **وسمةُ @Deprecated ليست شاشة** — نصُّها لمبرمجٍ يقرؤه في المحرِّر.
ANNOT = re.compile(r"^@(Deprecated|Suppress|RequiresPermission)")


ALLOW = {
    u"لا نصَّ عربيٌّ في كوتلن": {
        # **وحداتُ العرض في :ui مركزيّةٌ أصلاً** — «ل.س» و«كم» و«دقيقة»
        # مكتوبةٌ في مكانٍ واحدٍ تقرؤه التطبيقاتُ الأربعة، **وهي المقصودُ
        # من المركزيّة لا مخالفتُها.** ونقلُها إلى strings.xml تحسينٌ
        # مؤجَّلٌ لا عطبٌ قائم.
        "ui/src/main/kotlin/com/rahalgo/ui/Money.kt":
            u"وحداتُ عرضٍ مركزيّةٌ — مكانٌ واحدٌ للأربعة.",
        # **وإسنادُ الخرائط نصٌّ قانونيٌّ حرفيّ** — ترخيصُ OpenStreetMap
        # يشترط ظهورَه كما هو، **ومن ترجمه خالف الترخيص.**
        "ui/src/main/kotlin/com/rahalgo/ui/AccountScreen.kt":
            u"إسنادُ OpenStreetMap — نصٌّ يفرضه الترخيصُ حرفيّاً.",
        # **ورسالةُ استثناءٍ للمبرمج** — لا تصل شاشةً أبداً.
        "ui/src/main/kotlin/com/rahalgo/ui/Core.kt":
            u"رسالةُ استثناءٍ داخليّةٌ يقرؤها مبرمجٌ في logcat.",
        # **وسطرُ تشخيصِ الصوت والملاحة يُقرأ في logcat** — بُني بقالبٍ
        # يمتدّ أسطراً فلا يمسكه تخطّي السجلّ، **وهو ليس شاشةً.**
        "app-driver/src/main/kotlin/com/rahalgo/driver/trip/VoiceOrchestrator.kt":
            u"سطرُ تشخيصٍ متعدّدُ الأسطر — يُقرأ في logcat لا في شاشة.",
        "app-driver/src/main/kotlin/com/rahalgo/driver/trip/AndroidSpeaker.kt":
            u"selectedVoice سطرُ تشخيصٍ يُطبع في السجلّ.",
        "app-driver/src/main/kotlin/com/rahalgo/driver/orders/OrdersViewModel.kt":
            u"سطرُ تشخيصٍ متعدّدُ الأسطر في السجلّ.",
    },
    u"لا first()/last() بلا حارس": {
        "app-driver/src/main/kotlin/com/rahalgo/driver/trip/AndroidSpeaker.kt":
            u"محروسٌ في السطر ١٩٧: if (arabic.isEmpty()) return — قبل الفرز بأربعين سطراً.",
    },
}


GUARDS = []


def guard(name, why):
    """يُسجّل حارساً. الدالّةُ تُعيد قائمةَ مخالفات (ملفّ، سطر، بيان)."""
    def deco(fn):
        GUARDS.append((name, why, fn))
        return fn
    return deco


# ══════════════════════════════════════════════════════════════════════
#  ١ · الشكل — الثيمُ والتوكنز
# ══════════════════════════════════════════════════════════════════════

@guard(u"لا لونَ مكتوبٌ بيد",
       u"لونٌ خارجَ التوكنز لا يعرف الغامقَ من الفاتح. بقي في "
       u"TripMapControls.kt رماديّاً في الثيمين حتّى ٢٠٢٦-٠٨-٢٦.")
def g_colors():
    bad = []
    for m in APPS:
        for f in kt_files(m):
            for i, l in enumerate(read(f).split(chr(10)), 1):
                if HEX_COLOR.search(l) and not is_comment(l):
                    bad.append((rel(f), i, l.strip()[:70]))
    return bad


@guard(u"لا زاويةَ مكتوبةٌ بيد",
       u"RoundedCornerShape في شاشةٍ تفترق عن Rahal.shape، "
       u"فتصير الكروتُ بزوايا مختلفةٍ في شاشتين ولا يشتكي أحد.")
def g_shapes():
    bad = []
    for m in APPS:
        for f in kt_files(m):
            for i, l in enumerate(read(f).split(chr(10)), 1):
                if "RoundedCornerShape(" in l and not is_comment(l):
                    bad.append((rel(f), i, l.strip()[:70]))
    return bad


@guard(u"لا نصَّ عربيٌّ في كوتلن",
       u"نصٌّ في الشيفرة لا يُصحَّح من مكانٍ واحدٍ ولا يُترجم.")
def g_arabic():
    bad = []
    for m in APPS + ["ui"]:
        for f in kt_files(m):
            lines = read(f).split(chr(10))
            depth = 0          # عمقُ التعليق الكتليّ /* */
            logtail = 0        # كم سطراً بقي من نداءِ سجلٍّ مفتوح
            for i, l in enumerate(lines, 1):
                stripped = l.strip()
                if depth > 0:
                    if "*/" in l:
                        depth -= 1
                    continue
                if stripped.startswith("/*"):
                    if "*/" not in stripped:
                        depth += 1
                    continue
                if stripped.startswith("//") or stripped.startswith("*"):
                    continue
                # **ونداءُ السجلّ يمتدّ أسطراً** — فتُتخطّى معه.
                if ANNOT.match(stripped):
                    continue
                if LOGLINE.search(l):
                    logtail = 4
                    continue
                if logtail > 0:
                    logtail -= 1
                    if stripped.endswith(("+", ",", "(")) or stripped.startswith(("+", '"')):
                        continue
                code = l.split("//")[0]
                for q in QUOTED.findall(code):
                    if not AR.search(q):
                        continue
                    words = [w for w in re.split(r"[^؀-ۿ]+", q) if len(w) > 1]
                    # **كلمتان فأكثر جملةٌ تُقرأ** — و«كم» و«د» وحدةُ قياس.
                    if len(words) >= 2:
                        bad.append((rel(f), i, q[:50]))
    return bad


# ══════════════════════════════════════════════════════════════════════
#  ٢ · المركزيّة — ما يجب ألّا يُكتب مرّتين
# ══════════════════════════════════════════════════════════════════════

@guard(u"لا لفظَ مكرّرٌ بين تطبيقين",
       u"سبعةٌ وعشرون لفظاً كانت في أكثر من تطبيقٍ بأسماءِ مفاتيحَ مختلفة. "
       u"فمن صحّح «متوفر/غير متوفر» في تطبيقٍ تركها خطأً في اثنين.")
def g_dup_strings():
    vals = {}
    for a in APPS:
        for k, v in strings_of(a).items():
            vals.setdefault(v, []).append((a, k))
    bad = []
    for v, l in vals.items():
        apps = sorted(set(x[0] for x in l))
        if len(apps) >= 2:
            who = u"·".join(x.replace("app-", "") for x in apps)
            keys = u"، ".join(k for _, k in l)
            bad.append((who, 0, u"«%s» — %s" % (v[:34], keys)))
    return bad


@guard(u"لا مفتاحَ يحجب مفتاحاً في :ui",
       u"مفتاحٌ في تطبيقٍ بالاسم نفسِه يحجب المشترك — فيُصلَح المشتركُ "
       u"ولا يتغيّر شيءٌ على الشاشة.")
def g_shadow_strings():
    ui = strings_of("ui")
    bad = []
    for a in APPS:
        for k, v in strings_of(a).items():
            if k in ui:
                bad.append((a, 0, u"%s: التطبيق «%s» · المشترك «%s»" % (k, v[:22], ui[k][:22])))
    return bad


@guard(u"لا ملفَّ مكرّرٌ حرفاً بحرف",
       u"ShellViewModel كانت ١٤١ سطراً مطابقةً في ثلاثة تطبيقات — "
       u"إصلاحٌ يصل واحدةً ويترك اثنتين، ولا يصرخ بناءٌ ولا اختبار.")
def g_dup_files():
    seen = {}
    bad = []
    for a in APPS:
        for f in kt_files(a):
            body = re.sub(r"^package .*$", "", read(f), flags=re.M).strip()
            if len(body) < 400:
                continue
            key = hash(body)
            if key in seen:
                bad.append((rel(f), 0, u"مطابقٌ لـ " + seen[key]))
            else:
                seen[key] = rel(f)
    return bad


@guard(u"لا مضيفَ مكتوبٌ بيد",
       u"api.rahalgo.com كان في أربعة ملفّات — ومن نسي واحداً بنى "
       u"تطبيقاً يكلّم خادماً قديماً بلا شكوى.")
def g_hosts():
    bad = []
    for m in APPS:
        for f in kt_files(m):
            for i, l in enumerate(read(f).split(chr(10)), 1):
                if HOST.search(l) and not is_comment(l):
                    bad.append((rel(f), i, l.strip()[:70]))
    return bad


# ══════════════════════════════════════════════════════════════════════
#  ٣ · الإقلاع — ما يجب أن يُنادى في كلّ تطبيق
# ══════════════════════════════════════════════════════════════════════

@guard(u"كلُّ تطبيقٍ يرسل رقمَ نسخته",
       u"من أرسل صفراً لا تصله بوّابةُ التحديث أبداً — أيّاً كان الحدُّ "
       u"الأدنى في اللوحة. مشى السائقُ والمتجرُ والمندوبُ هكذا حتّى ٢٠٢٦-٠٨-٢٦.")
def g_version():
    bad = []
    for a in APPS:
        src = "".join(read(f) for f in kt_files(a))
        if "AppCore.install" not in src:
            bad.append((a, 0, u"لا نداءَ AppCore.install إطلاقاً"))
        elif "BuildConfig.VERSION_CODE" not in src:
            bad.append((a, 0, u"AppCore.install بلا BuildConfig.VERSION_CODE"))
    return bad


@guard(u"كلُّ تطبيقٍ يثبّت المُحمِّلَ ومراقبَ الشبكة",
       u"Images.install غيابُها يعني تنزيلَ الصور في كلّ إقلاع. "
       u"و Net.install غيابُها يعني شريطَ انقطاعٍ لا يظهر أبداً.")
def g_installs():
    bad = []
    for a in APPS:
        src = "".join(read(f) for f in kt_files(a))
        for call in ["Images.install", "Net.install"]:
            if call not in src:
                bad.append((a, 0, u"لا نداءَ " + call))
    return bad


@guard(u"كلُّ تطبيقٍ داخل AppFrame",
       u"AppFrame تحمل السمةَ وشريطَ الانقطاع وبوّابةَ التحديث وطلبَ "
       u"الأذون. ومن خرج عنها فقدها كلَّها بلا أن يُخطئ بناء.")
def g_appframe():
    bad = []
    for a in APPS:
        src = "".join(read(f) for f in kt_files(a))
        if "AppFrame" not in src:
            bad.append((a, 0, u"لا يستعمل AppFrame"))
    return bad


@guard(u"كلُّ إذنٍ مُعلَنٍ يُطلب فعلاً",
       u"إذنٌ في البيان لا يُطلب في الشيفرة إذنٌ ميّت — يظهر للمستخدم "
       u"في المتجر ولا يُستعمل. كان POST_NOTIFICATIONS في المندوب "
       u"معلَناً بلا خدمةٍ تتلقّى.")
def g_permissions():
    runtime = {
        "POST_NOTIFICATIONS": ["POST_NOTIFICATIONS"],
        "ACCESS_FINE_LOCATION": ["ACCESS_FINE_LOCATION"],
        "ACCESS_BACKGROUND_LOCATION": ["ACCESS_BACKGROUND_LOCATION"],
        "CAMERA": ["CAMERA"],
    }
    bad = []
    for a in APPS:
        man = manifest_of(a)
        src = "".join(read(f) for f in kt_files(a))
        shared = "".join(read(f) for f in kt_files("ui"))
        for perm, needles in runtime.items():
            if "android.permission." + perm not in man:
                continue
            if not any(n in src or n in shared for n in needles):
                bad.append((a, 0, perm + u" معلَنٌ ولا يُطلب"))
    return bad


@guard(u"كلُّ تطبيقٍ بإشعاراتٍ حزمتُه في Firebase",
       u"إضافةُ غوغل تُسقط البناءَ إن لم تجد اسمَ الحزمة — وبلا الإضافة "
       u"لا يصل إشعارٌ واحد. وملفُّ المندوب كان نسخةً من ملفّ الزبون.")
def g_firebase():
    bad = []
    for a in APPS:
        p = os.path.join(ROOT, a, "google-services.json")
        has_service = ".push.PushService" in manifest_of(a) or "PushService" in manifest_of(a)
        if not os.path.exists(p):
            if has_service:
                bad.append((a, 0, u"خدمةُ إشعاراتٍ بلا google-services.json"))
            continue
        pkgs = [c["client_info"]["android_client_info"]["package_name"]
                for c in json.loads(read(p)).get("client", [])]
        want = re.search(r'applicationId\s*=\s*"([^"]+)"', gradle_of(a))
        want = want.group(1) if want else None
        if want and want not in pkgs:
            bad.append((a, 0, u"الحزمة %s ليست في google-services.json" % want))
        g = gradle_of(a)
        if "google.services" in g and "crashlytics" not in g:
            bad.append((a, 0, u"google-services بلا crashlytics — ينهار عند أوّل إقلاع"))
    return bad


# ══════════════════════════════════════════════════════════════════════
#  ٤ · الانهيارُ الصامت — ما يسقط على جهازٍ ولا يسقط في بناء
# ══════════════════════════════════════════════════════════════════════

@guard(u"لا !! على نتيجةِ شبكة",
       u"!! على حقلٍ يأتي من المحرّك انهيارٌ مؤكَّدٌ يومَ يرسل المحرّكُ "
       u"فراغاً. ولا يظهر في بناءٍ ولا في اختبارٍ بمعطياتٍ سليمة.")
def g_bang_bang():
    bad = []
    for m in APPS:
        for f in kt_files(m):
            for i, l in enumerate(read(f).split(chr(10)), 1):
                if is_comment(l):
                    continue
                for hit in re.finditer(r"(\w+)!!", l):
                    name = hit.group(1)
                    # حالاتُ الواجهةِ المحلّيّة مقبولة (vm.reset!! ونحوها)
                    if name in ("it", "this"):
                        continue
                    if re.search(r"\b(order|item|user|me|data|res|resp|body|page|store)\b",
                                 name, re.I):
                        bad.append((rel(f), i, l.strip()[:70]))
    return bad


@guard(u"لا first()/last() بلا حارس",
       u"first() على قائمةٍ فارغةٍ يرمي NoSuchElementException. "
       u"والقائمةُ تأتي من المحرّك — فتفرغ يومَ يُحذف صنف.")
def g_first():
    bad = []
    for m in APPS:
        for f in kt_files(m):
            for i, l in enumerate(read(f).split(chr(10)), 1):
                if is_comment(l):
                    continue
                if re.search(r"\.(first|last)\(\)", l) and "OrNull" not in l:
                    bad.append((rel(f), i, l.strip()[:70]))
    return bad


@guard(u"لا runBlocking ولا Thread.sleep",
       u"كلاهما يجمّد الخيطَ الرئيس. والنتيجةُ ANR — «التطبيق لا يستجيب» "
       u"— وهو أسوأُ من انهيارٍ لأنّه لا يُسجَّل في Crashlytics.")
def g_blocking():
    bad = []
    for m in APPS:
        for f in kt_files(m):
            for i, l in enumerate(read(f).split(chr(10)), 1):
                if is_comment(l):
                    continue
                if "runBlocking" in l or "Thread.sleep" in l:
                    bad.append((rel(f), i, l.strip()[:70]))
    return bad


@guard(u"لا GlobalScope",
       u"مهمّةٌ في GlobalScope لا تموت بموت الشاشة — تكتب في حالةٍ "
       u"مرميّةٍ وتسرّب الذاكرة. والعطبُ يظهر بعد ساعةٍ من الاستعمال.")
def g_globalscope():
    bad = []
    for m in APPS:
        for f in kt_files(m):
            for i, l in enumerate(read(f).split(chr(10)), 1):
                if "GlobalScope" in l and not is_comment(l):
                    bad.append((rel(f), i, l.strip()[:70]))
    return bad


@guard(u"لا رسالةَ خطأٍ خام تُعرض",
       u"e.message نصٌّ إنجليزيٌّ من مكتبةٍ يظهر للمستخدم. والمترجمُ "
       u"المركزيُّ apiError موجودٌ ويعرف ثمانين رمزاً.")
def g_raw_errors():
    bad = []
    for m in APPS:
        for f in kt_files(m):
            for i, l in enumerate(read(f).split(chr(10)), 1):
                if is_comment(l):
                    continue
                if re.search(r"error\s*=\s*\w+\.message", l) or re.search(r"Text\(\s*\w+\.message", l):
                    bad.append((rel(f), i, l.strip()[:70]))
    return bad


@guard(u"لا حقلَ يخالف لوحةَ الثيم",
       u"قِيس ٢٠٢٦-٠٨-٢٧: ثمانيةٌ وأربعون حقلاً في الأربعة، **صفرُ لونٍ "
       u"مخصّصٍ وصفرُ زاويةٍ خارج التوكنز** — فهي متّسقةٌ بلا غلاف. "
       u"وحقلٌ يضع ألوانَه بيده يخرج عن الغامق والفاتح معاً، ولا يشتكي بناء.")
def g_field_theme():
    bad = []
    for m in APPS + ["ui"]:
        for f in kt_files(m):
            src = read(f)
            lines = src.split(chr(10))
            for i, l in enumerate(lines, 1):
                if is_comment(l):
                    continue
                if "TextFieldDefaults" in l or "OutlinedTextFieldDefaults" in l:
                    bad.append((rel(f), i, l.strip()[:70]))
                    continue
                # **زاويةٌ بعددٍ على حقلٍ** — التوكنز تُقبل، والأرقامُ لا.
                if re.search(r"shape\s*=\s*RoundedCornerShape\(\s*\d", l):
                    bad.append((rel(f), i, l.strip()[:70]))
    return bad


def run():
    """يُشغّل الحرّاسَ كلَّهم ويعيد قائمةَ (اسم، سبب، مخالفات)."""
    out = []
    for name, why, fn in GUARDS:
        try:
            hits = [h for h in fn() if h[0] not in ALLOW.get(name, {})]
            out.append((name, why, hits))
        except Exception as e:  # حارسٌ عاطلٌ لا يُسقط المنظومة
            out.append((name, why, [("—", 0, u"الحارسُ نفسُه تعطّل: %s" % e)]))
    return out
