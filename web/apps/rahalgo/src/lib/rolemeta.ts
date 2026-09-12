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

// ══════════════════════════════════════════════════════════════════════
//  **سياسةُ الإسناد — أهليّةٌ صريحةٌ لا وجودٌ مخترَع**
// ══════════════════════════════════════════════════════════════════════
//
// # ما وقع (٢٠٢٦-٠٩-١٢)
//
// **أنشأ المالكُ `observability` في الإنتاج فلم يجده في نافذة أدوار
// الحساب.** **والنافذةُ لم تكن تُرشِّح** — **كانت تقصّ**: `Chips`
// افتراضُها `nowrap` مع `overflow-x-auto` **وشريطُ التمرير مخفيٌّ
// بالأنماط**. **وقِيست المستطيلاتُ في متصفّحٍ حقيقيٍّ على التجهيز**:
//
//	scrollWidth = 2002  ·  clientWidth = 398
//	حبّاتٌ = 18  ·  **مرئيّةٌ = 3**
//	«مراقبة التشغيل» على **-820px** — خارجَ الإطار
//
// **والتعليقُ فوق `wrap` في `@rahalgo/ui` مكتوبٌ بهذا الدرس بعينه**:
// «ومن لم يرَ «مدير المنصة» لأنّها خارج الإطار لا يعرف أنّها موجودة» —
// **كُتب لنافذة «مستخدم جديد» ٢٠٢٦-٠٨-٠٨، ونافذةُ الأدوار لم تأخذه.**
//
// **وعطبٌ ثانٍ في الاتّجاه المقابل**: **النافذةُ كانت تعرض
// `owner_super_admin` حبّةً كبقيّتها** — نقرةٌ واحدةٌ تمنح كلَّ قدرةٍ
// في المعجم.
//
// # والفصلُ الذي تحرسه هذه الوحدة
//
//	**الوجودُ** — من المحرّك وحدَه، ولا `Object.keys` على معجمِ عرض
//	**الأهليّة** — سياسةٌ صريحةٌ هنا، **ومصدرُها واحدٌ لكلّ شاشة**
//	**المنعُ** — في المحرّك: `roles.manage` وتأكيدٌ وقيدُ تدقيق
//
// **وإخفاءُ حبّةٍ لطفٌ بالعين لا حراسة** — ومن حرس بالواجهة وحدَها
// حرس بابَ بيتٍ بستارة. **فهذه ترتيبُ عرضٍ يمنع الزلّة، لا سلطة.**

/** RoleClass **صنفُ الدور في سياسة الإسناد** — لا في وجوده. */
export type RoleClass = "staff" | "elevated" | "account_type" | "protected" | "custom";

/**
 * ROLE_CLASSES **تصنيفٌ صريحٌ للرموز المعروفة.**
 *
 * **والمجهولُ لا يُصنَّف هنا** — يصير `custom`: **يُعرَض في مجموعةٍ
 * مسمَّاةٍ برمزه ظاهراً، فلا يكسب أهليّةً صامتة** ولا يُخفى فيعود
 * العطبُ نفسُه. (شرطُ المالك بندَي ٢ و٩.)
 */
const ROLE_CLASSES: Record<string, RoleClass> = {
  // **المالكُ الأعلى محميّ** — **ولا يُسند من نافذةٍ عاديّة** (بندُ ٤).
  owner_super_admin: "protected",

  // **و`admin` عليا لا محميّة**: **تبلغ كلَّ شيءٍ تقريباً** — فتُفرَد
  // في مجموعتها بتحذيرها، **ولا تُنزع قدرةٌ قائمةٌ للمالك اليوم.**
  admin: "elevated",

  // **أدوارُ العمل** — تخويلٌ داخليٌّ يُسند ويُنزع.
  ops: "staff",
  operations: "staff",
  finance: "staff",
  customer_support: "staff",
  analytics: "staff",
  driver_verification: "staff",
  merchant_verification: "staff",
  marketing_content: "staff",
  trust_safety: "staff",
  observability: "staff",

  // **وصفةُ الحساب ليست وظيفة** — والمحرّكُ يرفض أكثرَها بيدٍ
  // (`ErrRoleConflict` · `ErrMerchantNeedsStore`).
  customer: "account_type",
  driver: "account_type",
  merchant: "account_type",
  sales: "account_type",
};

/** classifyRole **صنفُ رمزٍ** — والمجهولُ `custom` لا `staff`. */
export function classifyRole(code: string): RoleClass {
  return ROLE_CLASSES[code] ?? "custom";
}

/** STAFF_ASSIGNABLE_CODES **الرموزُ المصنَّفةُ أدوارَ عمل** — للتقرير والحرّاس. */
export const STAFF_ASSIGNABLE_CODES: readonly string[] = Object.keys(ROLE_CLASSES)
  .filter((c) => ROLE_CLASSES[c] === "staff")
  .sort();

/** PROTECTED_ROLE_CODES **ما لا يُسند من نافذةٍ عاديّة.** */
export const PROTECTED_ROLE_CODES: readonly string[] = Object.keys(ROLE_CLASSES)
  .filter((c) => ROLE_CLASSES[c] === "protected")
  .sort();

/** ACCOUNT_TYPE_ROLE_CODES **صفةُ الحساب لا وظيفتُه.** */
export const ACCOUNT_TYPE_ROLE_CODES: readonly string[] = Object.keys(ROLE_CLASSES)
  .filter((c) => ROLE_CLASSES[c] === "account_type")
  .sort();

/** RoleOption **حبّةٌ في نافذة الإسناد.** */
export interface RoleOption {
  code: string;
  label: string;
  /** **محميٌّ يملكه الحسابُ فعلاً** — يُرى ليُعرَف، ولا يُنقَر. */
  locked: boolean;
}

/** RoleGroup **مجموعةٌ مسمَّاةٌ في النافذة.** */
export interface RoleGroup {
  cls: RoleClass;
  title: string;
  note: string;
  roles: RoleOption[];
}

const GROUP_ORDER: readonly RoleClass[] = [
  "staff",
  "custom",
  "account_type",
  "elevated",
  "protected",
];

const GROUP_TITLES: Record<RoleClass, string> = {
  staff: m.terms.roleGroups.staff,
  custom: m.terms.roleGroups.custom,
  account_type: m.terms.roleGroups.accountType,
  elevated: m.terms.roleGroups.elevated,
  protected: m.terms.roleGroups.protected,
};

const GROUP_NOTES: Record<RoleClass, string> = {
  staff: m.terms.roleGroupNotes.staff,
  custom: m.terms.roleGroupNotes.custom,
  account_type: m.terms.roleGroupNotes.accountType,
  elevated: m.terms.roleGroupNotes.elevated,
  protected: m.terms.roleGroupNotes.protected,
};

/**
 * assignmentGroups **الأدوارُ التي جاءت من المحرّك، مرتَّبةً بسياسةٍ واحدة.**
 *
 * `held` **ما يملكه الحسابُ الآن** — **ويُقرَّر به ظهورُ المحميّ**:
 * **من ملك `owner_super_admin` يجب أن يُرى ليُنزَع** — **وإخفاؤه يمنع
 * النزعَ لا المنحَ**، وذاك أسوأُ.
 *
 * **ولا رمزَ يُخترَع**: ما ليس في `roles` لا يظهر أبداً.
 */
export function assignmentGroups(
  roles: readonly { code: string; name_key?: string }[],
  held: readonly string[] = [],
): RoleGroup[] {
  const heldSet = new Set(held);
  const buckets = new Map<RoleClass, RoleOption[]>();
  for (const r of roles) {
    const cls = classifyRole(r.code);
    // **والمحميُّ لا يُعرَض إلّا مملوكاً** — ومقفلاً.
    if (cls === "protected" && !heldSet.has(r.code)) continue;
    const list = buckets.get(cls) ?? [];
    list.push({ code: r.code, label: roleLabel(r), locked: cls === "protected" });
    buckets.set(cls, list);
  }
  const out: RoleGroup[] = [];
  for (const cls of GROUP_ORDER) {
    const list = buckets.get(cls);
    if (!list || list.length === 0) continue;
    list.sort((a, b) => a.label.localeCompare(b.label, "ar"));
    out.push({ cls, title: GROUP_TITLES[cls], note: GROUP_NOTES[cls], roles: list });
  }
  return out;
}

// **وشاشةُ إنشاء الحساب تنادي `assignmentGroups(roles)` نفسَها بلا
// `held`** — **ولا دالّةَ ثانيةً تُشبهها فتفترق يوماً** (بندُ ٧).
