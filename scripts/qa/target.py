# -*- coding: utf-8 -*-
"""**بوّابةُ الهدف — التجهيزُ افتراضاً، والإنتاجُ لا يُبلَغ بالخطأ.**

(قرارُ المالك ٢٠٢٦-٠٩-٢٩، شرطٌ حاجبٌ قبل أيّ تنفيذ: «لا سكربتَ يُستعمل في
 هذه المرحلة يقصد الإنتاجَ افتراضاً أو صامتاً».)

# ما كان

**ثلاثةُ سكربتاتٍ كانت تحمل عنوانَ الإنتاج حرفيّاً في أعلاها**:
`scripts/e2e-cycle.py` (ومعه ثلاثةُ حساباتٍ حقيقيّةٍ بكلماتها ورمزِ أدمن)،
و`scripts/perf-baseline.py`، و`scripts/qa/media.py`. **ولا واحدَ منها يقصد
التجهيز.**

**وأخطرُ ما فيها أنّها تعمل بلا سؤال**: من شغّل «الدورةَ الشاملة» ليجرّب
على التجهيز **أنشأ طلباتٍ حقيقيّةً في بيانات الزبائن** — ولا سطرَ يمنعه
ولا سطرَ يُنبّهه.

# والقاعدةُ الآن

	لا متغيّرَ بيئة            ⇒ التجهيز
	RAHALGO_TARGET=staging    ⇒ التجهيز
	RAHALGO_TARGET=production ⇒ **يُرفض** إلّا بعلَمَي سلامةٍ صريحين

**وعلَمان لا واحد**: `RAHALGO_ALLOW_PRODUCTION=1` **و**
`RAHALGO_CONFIRM=I-KNOW-THIS-IS-PRODUCTION` — **فعلَمٌ واحدٌ يُنسى في
سجلّ أوامر، واثنان لا يقعان سهواً.**

# ولا كلمةَ مرورٍ في المستودع

**والحساباتُ تُقرأ من البيئة لا من الملفّ** — [creds]. **وكلمةٌ في ملفٍّ
مُتابَعٍ بـgit هي كلمةٌ منشورة**، ومن نسخ المستودعَ نسخها.
"""
import os
import sys

STAGING = "https://staging-api.rahalgo.com"
PRODUCTION = "https://api.rahalgo.com"

STAGING_WEB = "https://staging.rahalgo.com"
PRODUCTION_WEB = "https://rahalgo.com"

_CONFIRM = "I-KNOW-THIS-IS-PRODUCTION"


class ProductionRefused(SystemExit):
    """**يُرفع فيقتل العمليّة** — لا يُلتقط فيُتجاهل."""


def target() -> str:
    """**يردّ `staging` أو `production`** — والافتراضُ التجهيز."""
    raw = (os.environ.get("RAHALGO_TARGET") or "staging").strip().lower()
    if raw in ("", "staging", "stg"):
        return "staging"
    if raw in ("production", "prod"):
        return "production"
    raise SystemExit(
        "RAHALGO_TARGET غيرُ معروف: %r — المسموحُ staging أو production" % raw)


def base_url(api_suffix: str = "") -> str:
    """**عنوانُ الهدف** — ويرفض الإنتاجَ بلا علَمَي السلامة.

    @param api_suffix مثل `"/api/v1"`، أو فارغٌ للجذر.
    """
    t = target()
    if t == "staging":
        return STAGING + api_suffix

    if not _allowed_production():
        raise ProductionRefused(
            "\n".join([
                "",
                "  !!! REFUSED: production needs an explicit permission !!!",
                "  *** رُفض: لا يُشغَّل على الإنتاج بلا إذنٍ صريح ***",
                "",
                "  الهدفُ المطلوب : production (%s)" % PRODUCTION,
                "  وينقص          : RAHALGO_ALLOW_PRODUCTION=1",
                "                   RAHALGO_CONFIRM=%s" % _CONFIRM,
                "",
                "  **والإنتاجُ بياناتُ زبائنٍ حقيقيّة** — طلبٌ يُنشأ فيه طلبٌ حقيقيّ,",
                "  وقيدٌ ماليٌّ يُكتب فيه دفترٌ حقيقيّ. **فإن لم تكن تقصده فلا تُمرّر",
                "  شيئاً**: بلا متغيّرٍ أصلاً يذهب السكربتُ إلى التجهيز.",
                "",
            ]))
    sys.stderr.write(
        "WARNING target=production (%s) - explicit permission given\n" % PRODUCTION)
    return PRODUCTION + api_suffix


def _allowed_production() -> bool:
    return (os.environ.get("RAHALGO_ALLOW_PRODUCTION", "").strip() == "1"
            and os.environ.get("RAHALGO_CONFIRM", "").strip() == _CONFIRM)


def web_url(path: str = "") -> str:
    """**عنوانُ الويب** — التجهيزُ افتراضاً، وبالعلَمَين نفسِهما للإنتاج.

    **وقراءةٌ ليست إذناً**: `mobile/testkit/api.py` كانت تطرق الإنتاجَ
    قراءةً فقط — **ولا ضررَ في `GET`**. لكنّ القاعدةَ في الافتراضِ لا في
    الأثر: **من رأى سكربتاً يذهب إلى الإنتاج بلا سؤالٍ ظنَّ الباقيَ كذلك.**
    """
    if target() == "staging":
        return STAGING_WEB + path
    if not _allowed_production():
        raise ProductionRefused(
            "web target=production مرفوضٌ بلا RAHALGO_ALLOW_PRODUCTION=1 "
            "و RAHALGO_CONFIRM=%s" % _CONFIRM)
    return PRODUCTION_WEB + path


def creds(name: str):
    """**حسابٌ من البيئة** — `(phone, password, pin|None)`.

    **ولا قيمةَ افتراضيّة**: من لم يُمرّر الحسابَ لم يقصد تشغيلَ هذا الشوط،
    **وحسابٌ مكتوبٌ في المستودع حسابٌ مكشوف.**

    المتغيّرات: `RAHALGO_<NAME>_PHONE` · `_PASSWORD` · `_PIN` (اختياريّ).
    """
    up = name.upper()
    phone = os.environ.get("RAHALGO_%s_PHONE" % up, "").strip()
    password = os.environ.get("RAHALGO_%s_PASSWORD" % up, "").strip()
    pin = os.environ.get("RAHALGO_%s_PIN" % up, "").strip() or None
    if not phone or not password:
        raise SystemExit(
            "ينقص حسابُ %s — مرّر RAHALGO_%s_PHONE و RAHALGO_%s_PASSWORD "
            "(ولا كلمةَ مرورٍ تُكتب في المستودع)" % (name, up, up))
    return (phone, password, pin)


def banner() -> str:
    """**سطرٌ يُطبع في أوّل كلّ سكربت** — فلا يُقرأ تقريرٌ بلا هدفه."""
    return "target=%s  url=%s" % (target(), base_url())
