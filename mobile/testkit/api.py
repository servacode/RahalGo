# -*- coding: utf-8 -*-
"""
══════════════════════════════════════════════════════════════════════
 الطبقةُ الثانية — عقدُ المحرّك: هل يردّ بما تتوقّعه التطبيقات؟
══════════════════════════════════════════════════════════════════════

**والبناءُ لا يكشف هذا إطلاقاً**: التطبيقُ يُبنى ويُشغَّل، **ثمّ يردّ
المحرّكُ حقلاً باسمٍ آخر** فتقع شاشةٌ فارغةٌ عند مستخدمٍ في الرقّة —
**ولا سطرَ في أيّ سجلّ.**

**وهذه الطبقةُ تطرق الإنتاج قراءةً لا كتابة** — لا حسابَ ولا طلب،
**وكلُّ ما تناديه نقاطٌ عامّةٌ يفتحها أيُّ زائر.**

# ولماذا الإنتاجُ لا خادمٌ محلّيّ

**العطبُ الذي نخاف منه عطبُ نشرٍ** — حقلٌ نُشر ناقصاً، أو صورةٌ لا
تُخدَم، أو ترويسةٌ ضاعت خلف الوكيل. **وخادمٌ محلّيٌّ لا يعرف شيئاً من
ذلك.**
"""

import json
import ssl
import time
import urllib.error
import urllib.request

BASE = "https://api.rahalgo.com"
TIMEOUT = 15

CHECKS = []


def check(name, why):
    def deco(fn):
        CHECKS.append((name, why, fn))
        return fn
    return deco


def fastest(path, tries=3, **kw):
    """
    **أسرعُ ثلاثِ محاولات** — لا متوسّطُها.

    # ولماذا الأصغرُ لا المتوسّط

    **قِيس ٢٠٢٦-٠٨-٢٩**: `‎/healthz` ردَّ في ٢٫٨٩ ثانية، **٢٫٣١ منها
    مصافحةُ شبكة** (DNS ٠٫٦٤ · TCP ٠٫٥٠ · TLS ١٫١٧) **والخادمُ ردَّ في
    ٠٫٥٨.**

    **وجهازُ المالك خلف في‑بي‑إن** — يدوّر مخارجَه وتتبدّل مسافتُه.
    **فقياسٌ واحدٌ يقيس الطريقَ لا الخادم**، ويصرخ الحارسُ على ما ليس
    عطباً. **وحارسٌ يتّهم بريئاً يُطفأ بعد ثالث مرّة.**

    **والأصغرُ أقربُ ما نملك إلى زمن الخادم** — تعثّرُ الشبكة يزيد ولا
    ينقص.
    """
    best = None
    for _ in range(tries):
        st, body, ms = get(path, **kw)
        if best is None or ms < best[2]:
            best = (st, body, ms)
        if ms < 800:
            break
    return best


def get(path, client="android-customer", version=None, raw=False):
    """نداءٌ واحد — يعيد (الحالة، الجسم، الزمن بالملّي)."""
    req = urllib.request.Request(BASE + path)
    req.add_header("X-RahalGo-Client", client)
    if version is not None:
        req.add_header("X-RahalGo-Version", str(version))
    ctx = ssl.create_default_context()
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT, context=ctx) as r:
            body = r.read()
            ms = int((time.time() - t0) * 1000)
            return r.status, (body if raw else json.loads(body.decode("utf-8"))), ms
    except urllib.error.HTTPError as e:
        body = e.read()
        ms = int((time.time() - t0) * 1000)
        try:
            return e.code, json.loads(body.decode("utf-8")), ms
        except Exception:
            return e.code, body, ms
    except Exception as e:
        return 0, {"_error": str(e)}, int((time.time() - t0) * 1000)


def walk(node, key, out):
    if isinstance(node, dict):
        for k, v in node.items():
            if k == key and v:
                out.append(v)
            else:
                walk(v, key, out)
    elif isinstance(node, list):
        for x in node:
            walk(x, key, out)


# ══════════════════════════════════════════════════════════════════════
#  النقاطُ التي يفتحها كلُّ تطبيقٍ عند الإقلاع
# ══════════════════════════════════════════════════════════════════════

@check(u"المحرّكُ حيّ",
       u"إن سقطت هذه فما بعدها لا معنى له.")
def c_health():
    st, _b, ms = fastest("/healthz")
    if st != 200:
        return [(u"/healthz", u"الحالة %s" % st)]
    if ms > 4000:
        return [(u"/healthz", u"ردَّ في %d ملّي — بطيءٌ جدّاً" % ms)]
    return []


@check(u"الشاشةُ الرئيسة تردّ كاملةً",
       u"أوّلُ نداءٍ يفتحه الزبون. حقلٌ ناقصٌ هنا شبكةُ صورٍ فارغة.")
def c_home():
    st, b, ms = fastest("/api/v1/public/home")
    if st != 200:
        return [(u"/public/home", u"الحالة %s" % st)]
    bad = []
    if ms > 5000:
        bad.append((u"/public/home", u"ردَّ في %d ملّي" % ms))
    thumbs, fulls = [], []
    walk(b, "image_thumb_url", thumbs)
    walk(b, "image_url", fulls)
    if not fulls:
        bad.append((u"/public/home", u"لا صورةَ واحدةٌ في الرد"))
    missing = len(fulls) - len(thumbs)
    if missing > 0:
        bad.append((u"/public/home",
                    u"%d صورةً بلا مصغّرة — تُنزَّل كاملةً (٢١٢ ك.ب بدل ٢٤)" % missing))
    return bad


@check(u"المصغّراتُ تُخدَم فعلاً",
       u"عنوانٌ في الرد لا يعني ملفّاً على القرص. وقعت الحادثةُ "
       u"٢٠٢٦-٠٨-٠٩: نصفُ الوسائط في مجلّدٍ ونصفُها في آخر، "
       u"والقاعدةُ تشير إليها كلِّها فتردّ ٤٠٤ بلا خطأٍ في سجلّ.")
def c_media():
    st, b, _ = get("/api/v1/public/home")
    if st != 200:
        return [(u"/public/home", u"لم يُقرأ")]
    thumbs = []
    walk(b, "image_thumb_url", thumbs)
    bad = []
    for u in thumbs[:12]:
        s2, body, _ms = get(u, raw=True)
        if s2 != 200:
            bad.append((u, u"الحالة %s" % s2))
        elif len(body) < 500:
            bad.append((u, u"حجمٌ %d بايت — ملفٌّ مقطوع" % len(body)))
    return bad


@check(u"ترويسةُ التخزين على الوسائط",
       u"بلا max-age تُنزَّل الصورُ في كلّ إقلاع. الشاشةُ الرئيسة "
       u"٤٥ صورةً — ميغا كاملٌ في كلّ فتحةٍ على شبكة الرقّة.")
def c_cache():
    st, b, _ = get("/api/v1/public/home")
    thumbs = []
    walk(b, "image_thumb_url", thumbs)
    if st != 200 or not thumbs:
        return [(u"/public/home", u"لا صورةَ تُفحص")]
    req = urllib.request.Request(BASE + thumbs[0], method="HEAD")
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as r:
            cc = r.headers.get("Cache-Control", "")
    except Exception as e:
        return [(thumbs[0], u"تعذّرت القراءة: %s" % e)]
    if "max-age" not in cc:
        return [(thumbs[0], u"بلا Cache-Control: «%s»" % cc)]
    return []


@check(u"بوّابةُ التحديث تعمل للتطبيقات الأربعة",
       u"المحرّكُ يجب أن يميّز نوعَ العميل. وكانت لا تعرف المتجرَ ولا "
       u"المندوبَ حتّى ٢٠٢٦-٠٨-٢٦ — فمهما رفعتَ الحدَّ لا يُجبَران.")
def c_update_gate():
    bad = []
    for kind in ["customer", "driver", "merchant", "rep"]:
        st, b, _ = get("/api/v1/auth/me", client="android-" + kind, version=1)
        # ٤٢٦ = يجب التحديث · ٤٠١ = الحدُّ صفرٌ فمرّ ثمّ رُفض لغياب الجلسة
        if st not in (401, 426):
            bad.append((kind, u"ردّ غيرُ متوقَّع: %s" % st))
        if st == 426 and not isinstance(b, dict):
            bad.append((kind, u"٤٢٦ بجسمٍ غيرِ مفهوم"))
    return bad


@check(u"رمزُ التوثيق يُطلب بتذكرةٍ لا برسالة",
       u"التوثيقُ المعكوس (٢٠٢٦-٠٨-٢٦): المنصّةُ لا ترسل أوّلاً — "
       u"المستخدمُ يرسل نصّاً فيه وسمٌ فيردّ البوت. وهو ما أنقذ الرقمَ "
       u"من التقييد.")
def c_wa_ticket():
    req = urllib.request.Request(BASE + "/api/v1/auth/wa/ticket", method="POST",
                                 data=json.dumps({"phone": "0999000000", "purpose": "verify"}).encode("utf-8"))
    req.add_header("Content-Type", "application/json")
    req.add_header("X-RahalGo-Client", "android-customer")
    try:
        with urllib.request.urlopen(req, timeout=TIMEOUT) as r:
            b = json.loads(r.read().decode("utf-8"))
    except urllib.error.HTTPError as e:
        return [(u"/auth/wa/ticket", u"الحالة %s" % e.code)]
    except Exception as e:
        return [(u"/auth/wa/ticket", str(e))]
    data = b.get("data", b)
    bad = []
    for k in ("tag", "text", "wa_url"):
        if not data.get(k):
            bad.append((u"/auth/wa/ticket", u"حقلٌ ناقص: " + k))
    if data.get("tag") and len(str(data["tag"])) != 7:
        bad.append((u"/auth/wa/ticket", u"طولُ الوسم %d لا ٧" % len(str(data["tag"]))))
    return bad


@check(u"نمطُ الخريطة يُخدَم",
       u"التطبيقُ يفتح الخريطةَ من مضيفنا. ونمطٌ لا يُخدَم يعني شاشةَ "
       u"ملاحةٍ رماديّةً عند السائق في منتصف رحلة.")
def c_map_style():
    st, _b, ms = fastest("/api/v1/public/map-style.json")
    if st != 200:
        return [(u"/public/map-style.json", u"الحالة %s" % st)]
    if ms > 6000:
        return [(u"/public/map-style.json", u"ردَّ في %d ملّي" % ms)]
    return []


# ══════════════════════════════════════════════════════════════════════
#  مساراتُ الموقع — بعد أن صار للإدارة وحدَها
# ══════════════════════════════════════════════════════════════════════

WEB = "https://rahalgo.com"


def web(path, follow=False):
    """يطرق الموقعَ ويعيد الحالة. **ولا يتبع التحويلَ إلّا إن طُلب.**"""
    req = urllib.request.Request(WEB + path, method="GET")
    req.add_header("User-Agent", "rahalgo-testkit")
    opener = urllib.request.build_opener()
    if not follow:
        class NoRedirect(urllib.request.HTTPRedirectHandler):
            def redirect_request(self, *_a, **_k):
                return None
        opener = urllib.request.build_opener(NoRedirect)
    try:
        with opener.open(req, timeout=TIMEOUT) as r:
            return r.status
    except urllib.error.HTTPError as e:
        return e.code
    except Exception:
        return 0


@check(u"بابُ التطبيقات مفتوح",
       u"‏`UpdateGate` في أندرويد يجرّب `market://` ثمّ يسقط إلى "
       u"‏`rahalgo.com/app`. **وكان أربعمئةً وأربعة حتّى ٢٠٢٦-٠٨-٢٦** — "
       u"فمن لا متجرَ على جهازه يُترك بلا مخرج.")
def c_app_page():
    st = web("/app", follow=True)
    return [] if st == 200 else [(u"/app", u"الحالة %s" % st)]


@check(u"بابُ الإدارة في موضعه الجديد",
       u"‏`‎/adminrahalgo` (قرارُ المالك ٢٠٢٦-٠٨-٢٦). **وإن عاد ٤٠٤ فالنشرُ "
       u"لم يصل** — والموظّفون بلا باب.")
def c_admin_door():
    st = web("/adminrahalgo", follow=True)
    return [] if st == 200 else [(u"/adminrahalgo", u"الحالة %s" % st)]


@check(u"أبوابُ الأدوار أُغلقت",
       u"‏`‎/store` كانت لوحةَ المتجر على الويب — **حُذفت ٢٠٢٦-٠٨-٢٦**، "
       u"ولوحةٌ تُصان مرّتين تفترق نسختاها بلا أن يصرخ شيء.")
def c_role_doors():
    bad = []
    st = web("/store", follow=True)
    if st != 404:
        bad.append((u"/store", u"ما زالت تُخدَم — الحالة %s" % st))
    return bad


@check(u"الروابطُ القديمة تحوّل لا تسقط",
       u"‏`‎/login` و`‎/signup` و`‎/forgot` منشورةٌ في رسائلَ قديمةٍ "
       u"وإشعارات. **ومن فتحها لا يُرمى إلى أربعمئةٍ وأربعة** بل إلى "
       u"حيث يجد تطبيقه.")
def c_old_links():
    bad = []
    for p in ["/login", "/signup", "/forgot"]:
        st = web(p, follow=True)
        if st != 200:
            bad.append((p, u"الحالة %s" % st))
    return bad


@check(u"صفحةُ حذف الحساب عامّة",
       u"**غوغل بلاي تشترطها** رابطاً في بطاقة التطبيق، **يُفتح بلا "
       u"تسجيل دخولٍ وبلا تثبيت التطبيق.** وحذفُها يُسقط المراجعة.")
def c_delete_account():
    st = web("/delete-account", follow=True)
    return [] if st == 200 else [(u"/delete-account", u"الحالة %s" % st)]


@check(u"وجهاتُ الإشعارات موجودة",
       u"**أمسك الحارسُ ٢٠٢٦-٠٨-٢٧** سبعَ نقاطٍ تشير إلى `‎/orders` "
       u"و`‎/wallet` و`‎/complaints` — **وقد حُذفت من الموقع.** ومن ضغط "
       u"خبرَه وقع على أربعمئةٍ وأربعة، **وهو يُقرأ أنّ المنصّة معطوبة.**")
def c_notify_targets():
    bad = []
    for p in ["/portal/orders", "/portal/wallet", "/portal/complaints"]:
        st = web(p, follow=True)
        if st not in (200, 302, 307):
            bad.append((p, u"الحالة %s" % st))
    return bad


def run():
    out = []
    for name, why, fn in CHECKS:
        try:
            out.append((name, why, fn()))
        except Exception as e:
            out.append((name, why, [(u"—", u"الفحصُ نفسُه تعطّل: %s" % e)]))
    return out
