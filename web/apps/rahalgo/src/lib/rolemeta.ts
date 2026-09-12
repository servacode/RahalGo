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

/**
 * RoleClass **صنفُ الدور في السياسة** — لا في وجوده.
 *
 * **والوجودُ من المحرّك دائماً** (`ADG-2`): **جدولُ `roles` هو الحقيقة**،
 * **وهذا تصنيفُ ما وُجد لا قائمةُ ما يوجد.**
 *
 * **ونسخةُ المحرّك في `internal/authz/roleclass.go`** — **ويحرس
 * تطابقَهما `TestRoleClassesMatchWebPolicy`**: **قائمتان لشيءٍ واحدٍ
 * تفترقان يوماً، وافتراقُهما صامت.**
 */
export type RoleClass =
  | "account_type"
  | "staff"
  | "elevated"
  | "protected"
  | "legacy"
  | "custom";

/**
 * ROLE_CLASSES **تصنيفٌ صريحٌ للرموز المعروفة.**
 *
 * **والمجهولُ لا يُصنَّف هنا** — يصير `custom`: **يُسند بمسار المنح
 * المخوَّل، ولا يُخلَق به حسابٌ ولا يكسب سلطةً مرتفعةً صامتة.**
 */
const ROLE_CLASSES: Record<string, RoleClass> = {
  // **المالكُ الأعلى محميّ** (بندُ ز) — للمالك وحدَه منحاً ونزعاً.
  owner_super_admin: "protected",

  // **و`admin` مرتفع** (بندُ و): **يبلغ `roles.manage`** — **فمن منحه
  // منح سلطةَ السلطات**، **ومنحُه للمالك وحدَه.**
  admin: "elevated",

  // **أدوارُ العمل** (بندُ ج) — تخويلٌ يمرّ بمسار المنح المخوَّل.
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

  // **و`ops` إرثٌ يُقرأ ولا يُمنَح** (`OPS-5`، قرارُ المالك بندَ ج):
  // **اسمُه العربيُّ «العمليات» كاسم `operations` وقدراتُهما مختلفة**
  // — **ودمجُهما هجرةٌ بقرارٍ مستقلّ.** **فمن يحمله يبقى ويُنزَع
  // منه، ولا يُحمَّل أحدٌ جديد.**
  ops: "legacy",
};

/** classifyRole **صنفُ رمزٍ** — والمجهولُ `custom` لا `staff`. */
export function classifyRole(code: string): RoleClass {
  return ROLE_CLASSES[code] ?? "custom";
}

const codesOfClass = (c: RoleClass): readonly string[] =>
  Object.keys(ROLE_CLASSES)
    .filter((k) => ROLE_CLASSES[k] === c)
    .sort();

/** STAFF_ASSIGNABLE_CODES **أدوارُ العمل** — للتقرير والحرّاس. */
export const STAFF_ASSIGNABLE_CODES = codesOfClass("staff");
/** PROTECTED_ROLE_CODES **ما لا يُسند من نافذةٍ عاديّة.** */
export const PROTECTED_ROLE_CODES = codesOfClass("protected");
/** ACCOUNT_TYPE_ROLE_CODES **صفةُ الحساب لا وظيفتُه.** */
export const ACCOUNT_TYPE_ROLE_CODES = codesOfClass("account_type");
/** ELEVATED_ROLE_CODES **ما يبلغ `roles.manage`.** */
export const ELEVATED_ROLE_CODES = codesOfClass("elevated");
/** LEGACY_ROLE_CODES **إرثٌ يُقرأ ولا يُمنَح جديداً.** */
export const LEGACY_ROLE_CODES = codesOfClass("legacy");

/**
 * isOwner **أيملك هذا المشغّلُ دورَ المالك؟**
 *
 * **وهو سؤالُ عرضٍ لا سؤالُ تخويل** — **والحدُّ في المحرّك**
 * (`guardGrantAuthority`). **وهذا يمنع أن تُعرَض على المالكِ نقرةٌ
 * يردُّها المحرّك** (بندُ ط: «لا تُعرَض إلّا أفعالٌ يُتمّها العقد»).
 */
export function isOwner(actorRoles: readonly string[] = []): boolean {
  return PROTECTED_ROLE_CODES.some((c) => actorRoles.includes(c));
}

/**
 * canGrantNew **أيُعرَض هذا الدورُ لمنحٍ جديد؟**
 *
 *	protected  ⇒ **لا** — للمالك بمسارٍ مصرَّحٍ به، لا حبّةً عاديّة
 *	elevated   ⇒ للمالك وحدَه
 *	legacy     ⇒ **لا** — قائمُه يعمل، ولا منحَ جديد (`OPS-5`)
 *	غيرُها     ⇒ نعم، بـ`roles.manage`
 */
export function canGrantNew(code: string, actorRoles: readonly string[] = []): boolean {
  switch (classifyRole(code)) {
    case "protected":
      return false;
    case "elevated":
      return isOwner(actorRoles);
    case "legacy":
      return false;
    default:
      return true;
  }
}

/** creatableAtSignup **أيُخلَق حسابٌ بهذا الدور مباشرةً؟** — صفةُ الحساب وحدَها. */
export function creatableAtSignup(code: string): boolean {
  return classifyRole(code) === "account_type";
}

/** RoleOption **حبّةٌ في نافذة الإسناد.** */
export interface RoleOption {
  code: string;
  label: string;
  /** **يُرى ولا يُنقَر** — المحميُّ المملوك: يُعرَف ولا يُبدَّل من هنا. */
  locked: boolean;
}

/** RoleGroup **مجموعةٌ مسمَّاةٌ في النافذة.** */
export interface RoleGroup {
  cls: RoleClass;
  title: string;
  note: string;
  /** **يلزمه منحٌ بعد الإنشاء** — في شاشة «مستخدم جديد» وحدَها. */
  viaGrant?: boolean;
  roles: RoleOption[];
}

const GROUP_ORDER: readonly RoleClass[] = [
  "staff",
  "custom",
  "account_type",
  "elevated",
  "legacy",
  "protected",
];

const GROUP_TITLES: Record<RoleClass, string> = {
  staff: m.terms.roleGroups.staff,
  custom: m.terms.roleGroups.custom,
  account_type: m.terms.roleGroups.accountType,
  elevated: m.terms.roleGroups.elevated,
  legacy: m.terms.roleGroups.legacy,
  protected: m.terms.roleGroups.protected,
};

const GROUP_NOTES: Record<RoleClass, string> = {
  staff: m.terms.roleGroupNotes.staff,
  custom: m.terms.roleGroupNotes.custom,
  account_type: m.terms.roleGroupNotes.accountType,
  elevated: m.terms.roleGroupNotes.elevated,
  legacy: m.terms.roleGroupNotes.legacy,
  protected: m.terms.roleGroupNotes.protected,
};

/**
 * assignmentGroups **أدوارُ المحرّك مرتَّبةً بسياسةٍ واحدة.**
 *
 * `held` **ما يملكه الحسابُ الآن** · `actorRoles` **أدوارُ المشغّل.**
 *
 * **وقاعدةُ الظهور**: **ما لا يُمنَح جديداً لا يُعرَض إلّا مملوكاً** —
 * **فيُرى ليُنزَع.** **وإخفاءُ المملوكِ يمنع النزعَ لا المنحَ، وذاك
 * أسوأ.**
 *
 * **ولا رمزَ يُخترَع**: ما ليس في `roles` لا يظهر أبداً.
 */
export function assignmentGroups(
  roles: readonly { code: string; name_key?: string }[],
  held: readonly string[] = [],
  actorRoles: readonly string[] = [],
): RoleGroup[] {
  const heldSet = new Set(held);
  const buckets = new Map<RoleClass, RoleOption[]>();
  for (const r of roles) {
    const cls = classifyRole(r.code);
    const mine = heldSet.has(r.code);
    // **ونقرةٌ لا يُتمّها المحرّكُ لا تُعرَض** (بندُ ط).
    if (!mine && !canGrantNew(r.code, actorRoles)) continue;
    buckets.set(cls, [
      ...(buckets.get(cls) ?? []),
      // **والمحميُّ يُرى ولا يُنقَر** — نزعُه بمسار المالك لا من هنا.
      { code: r.code, label: roleLabel(r), locked: cls === "protected" },
    ]);
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

/**
 * signupGroups **ما يُعرَض في «مستخدم جديد»** — بثلاث فئاتٍ صريحة.
 *
 * (بندُ ط: «أنواع الحسابات · أدوار الموظفين · الإدارة المرتفعة».)
 *
 * **والفرقُ الجوهريُّ مُعلَنٌ في كلّ مجموعة** (`viaGrant`):
 *
 *	account_type  ⇒ **يُخلَق الحسابُ به مباشرةً**
 *	staff · custom · elevated ⇒ **حسابٌ يُخلَق ثمّ دورٌ يُمنَح**
 *	                            بـ`roles.manage` وتأكيدٍ وقيدِ تدقيق
 *
 * **والمحميُّ والإرثُ لا يُعرَضان هنا إطلاقاً** — **لا يُخلَق بهما
 * حسابٌ ولا يُمنحان جديداً.**
 */
export function signupGroups(
  roles: readonly { code: string; name_key?: string }[],
  actorRoles: readonly string[] = [],
): RoleGroup[] {
  return assignmentGroups(roles, [], actorRoles)
    .filter((g) => g.cls !== "protected" && g.cls !== "legacy")
    .map((g) => ({ ...g, viaGrant: !creatableAtSignup(g.roles[0]?.code ?? "") }));
}
