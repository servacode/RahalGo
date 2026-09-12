/**
 * ══════════════════════════════════════════════════════════════════════
 * **اسمُ الدور للعرض — مصدرٌ واحدٌ لا معجمٌ في كلّ شاشة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما وقع
 *
 * **شاشةُ الأدوار كانت تعرض `analytics` و`customer_support` و
 * `driver_verification` و`marketing_content` بالإنجليزيّة** — رموزَ
 * محرّكٍ لموظّفٍ عربيٍّ يقرأ. **والسببُ أنّ معجمَ الأسماء فيه سبعةُ
 * رموزٍ وفي القاعدة ستّةَ عشر**: هجرةُ `0138` أضافت ثمانيةً ولم يُضف
 * لها اسم.
 *
 * **وخمسُ شاشاتٍ كانت تقرأ `m.terms.roleNames` كلُّ واحدةٍ بنفسها**
 * وتكتب `LABELS[r] ?? r` من جديد — **فخمسُ نسخٍ لمنطقٍ واحد.**
 *
 * # وما لا يجوز
 *
 * **الواجهةُ لا تقول أيُّ أدوارٍ توجد** — **المحرّكُ يقولها.**
 * (`ADG-1`، وقد أُغلق هذا العطبُ في ٧٠ب-و١ حين كانت الشاشاتُ تقرأ
 * `Object.keys(ROLE_LABELS)` فتُخفي أربعةَ أدوارٍ قائمةً في القاعدة.)
 *
 * **فهذه الوحدةُ عرضٌ محض**: اسمٌ ووصفٌ عربيّان. **ولا وجودَ ولا
 * تخويلَ فيها.**
 *
 * # وترتيبُ الاسم — والمحرّكُ يسبق
 *
 *	١ · اسمُ المحرّكِ إن كان اسمَ إنسانٍ لا مفتاحَ ترجمة
 *	٢ · فمعجمُ الأسماء المعروفة بالرمز
 *	٣ · فالرمزُ نفسُه
 *
 * **والمحرّكُ أوّلاً بقصد**: **دورٌ ينشئه المالكُ باسمٍ عربيٍّ يُعرَض
 * باسمه الذي كتبه**، لا باسمٍ خبّأته الواجهةُ له. **والأدوارُ المبذورةُ
 * تحمل مفاتيحَ ترجمة** (`roles.admin`) **فلا تُعرَض أبداً لإنسان**،
 * وعندها يعمل المعجم.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

/** **أسماءُ الأدوار المعروفة** — معينُ عرضٍ لا مصدرَ وجود. */
export const ROLE_LABELS: Record<string, string> = m.terms.roleNames;

/** **وصفٌ موجزٌ لكلّ دور** — ملخّصُ عرضٍ لا قاعدةَ تخويل. */
export const ROLE_DESCRIPTIONS: Record<string, string> = m.terms.roleDescriptions;

/**
 * **أسماءُ الصلاحيّات المعروضة.**
 *
 * **وهو مقصودٌ صغيراً**: **المحرّكُ يرسل وصفاً عربيّاً لكلّ قدرةٍ في
 * معجمه** — ونسخُه هنا معجمٌ ثانٍ للشيء الواحد، **ومعجمان يفترقان
 * يوماً.** فلا يُكتب هنا إلّا اسمٌ أقصرُ أراده المالكُ للعرض.
 */
export const CAPABILITY_LABELS: Record<string, string> = m.terms.capabilityNames;

/** **مفتاحُ ترجمةٍ لا اسمُ إنسان** — `roles.admin` لا يُعرَض. */
function isTranslationKey(s: string): boolean {
  return /^[a-z_]+\.[a-z_.]+$/.test(s);
}

/**
 * roleLabelByCode **الاسمُ العربيُّ لرمزِ دور.**
 *
 * `backendName` **اسمُ المحرّك إن توفّر** — ويسبق المعجمَ إن كان اسمَ
 * إنسان.
 */
export function roleLabelByCode(code: string, backendName?: string): string {
  const given = (backendName ?? "").trim();
  if (given && !isTranslationKey(given)) return given;
  const known = ROLE_LABELS[code];
  if (known) return known;
  return code;
}

/** roleLabel **الاسمُ الذي يُعرَض لدورٍ جاء من المحرّك.** */
export function roleLabel(role: { code: string; name_key?: string }): string {
  return roleLabelByCode(role.code, role.name_key);
}

/**
 * roleDescription **وصفُ الدور** — أو فراغٌ لمن لا وصفَ له.
 *
 * **والفراغُ لا يُعرَض سطراً خالياً** — الشاشةُ تُسقط العنصرَ.
 */
export function roleDescription(code: string): string {
  return ROLE_DESCRIPTIONS[code] ?? "";
}

/**
 * capabilityLabel **الاسمُ العربيُّ لقدرة.**
 *
 * **ثلاثُ محاولاتٍ**: معجمُ العرض · فوصفُ المحرّك · فالرمزُ نفسُه.
 * **والرمزُ يبقى ظاهراً ثانويّاً في الشاشة** — فمن يمنح قدرةً يحتاج
 * أن يرى ما يمنحه بالضبط.
 */
export function capabilityLabel(code: string, backendDescription?: string): string {
  const known = CAPABILITY_LABELS[code];
  if (known) return known;
  const desc = (backendDescription ?? "").trim();
  if (desc) return desc;
  return code;
}

/**
 * ASSIGNABLE_ROLE_CODES **قائمةُ اختيارٍ في شاشة الحسابات.**
 *
 * **وليست مصدرَ وجود** — المحرّكُ يرفض رمزاً لا يعرفه، **وهو الحارس.**
 *
 * **وكانت `Object.keys(ROLE_LABELS)`** — **فكان كلُّ اسمٍ يُضاف للعرض
 * يُغيّر صامتاً ما يُعرَض على من ينشئ حساباً.** فصُرّح بالقائمة كي
 * يبقى هذا التغييرُ قراراً لا أثراً جانبيّاً.
 *
 * **وهي نفسُ السبعةِ المعروضةِ قبل هذه الدورة** — توسيعُها قرارُ
 * مالكٍ لا مسألةُ ترجمة.
 */
export const ASSIGNABLE_ROLE_CODES: readonly string[] = [
  "customer",
  "driver",
  "merchant",
  "sales",
  "ops",
  "finance",
  "admin",
];
