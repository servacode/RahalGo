/**
 * توجيه المستخدم حسب دوره — مصدر واحد يستعمله تسجيل الدخول وزر "العودة إلى لوحتي".
 * كل اللوحات متطابقة، والفرق الوحيد بينها الصلاحيات والأقسام؛ فالتوجيه مركزي.
 */

import { api, type AuthUser } from "./client";

/**
 * هل يحمل أصحابُ الأدوار الأخرى دورَ الزبون أيضاً؟
 *
 * **نعم — بقرار المالك ٢٠٢٦-٠٨-١٠**: «أيُّ دورٍ لا يمكن أن يظهر له إلّا
 * كزبون… طبعاً الدورُ يتم تفعيله بشكلٍ تلقائيٍّ كزبون، لا يحتاج تدخّلاً
 * يدويّاً».
 *
 * **ومرآةٌ لثابت الخادم `identity.FieldRolesAreCustomers`** — والخادمُ هو
 * الذي يمنح، **وهذه تقرّر ما يُعرض.** فلو افترقا لَظهر زرٌّ لا يعمل أو
 * غاب بابٌ مفتوح.
 *
 * **وكانت مُطفأةً لأجل تجربةٍ انتهت** (٢٠٢٦-٠٨-٠١): كان السائقُ يطلب طلباً
 * ثمّ يوصّله بنفسه فتختلط الأطراف. **ولم يعد ممكناً** — الطلبُ يُعرض
 * بالدور لا على من يشاء.
 *
 * وعليها يتوقّف زرّ «تسوّق» في اللوحات وزرُّ «لوحتي» في الموقع: **وزرٌّ
 * يقود إلى بابٍ لا يُفتح أسوأ من غيابه** — من غاب زرُّه يعرف أنه لا يملك
 * الأمر، ومن تبعه ظنّ أن النظام انكسر.
 */
export const FIELD_ROLES_ARE_CUSTOMERS = true;

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بيتٌ واحدٌ — فالوجهاتُ مساراتٌ لا عناوين**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «بابٌ واحدٌ للجميع وشاشةُ تسجيلٍ واحدةٌ
 *  للجميع بدون استثناء».)
 *
 * **كانت خمسةَ عناوينَ لخمسة تطبيقات** — ولكلٍّ منفذُه ومتغيّرُ بيئته.
 * **وصارت اللوحاتُ أقساماً في تطبيقٍ واحد**، فالانتقالُ بينها تنقّلٌ داخليّ.
 *
 * **والأصلُ يبقى حقلاً في `Destination`** ولا يُحذف: **تطبيقُ أندرويد
 * سيفتح لوحاتِه من أصلٍ آخر**، ونقطتا `handoff`/`sso` في المحرّك تنتظرانه.
 * **وحقلٌ فارغٌ يعني «هنا»** — فتقصر `goTo` الطريقَ بلا تسليمِ جلسة.
 */
/**
 * **ولوحتا السائق والمندوب حُذفتا من الويب** (٢٠٢٦-٠٨-٢٣، بأمر المالك).
 *
 * **لكلٍّ منهما تطبيقُ أندرويد قائمٌ ومُختبَر** — **ولوحتان تُصانان لدورٍ
 * له تطبيقٌ ضريبةٌ بلا مقابل**: كلُّ ميزةٍ تُبنى مرّتين، وكلُّ عطبٍ
 * يُصلَح في موضعين، **وأحدُهما لا يفتحه أحد.**
 *
 * **وصاحبُ الدور يُردّ إلى واجهة الزبون** — فهو زبونٌ أيضاً
 * (`FIELD_ROLES_ARE_CUSTOMERS`)، **ولا يقع على بابٍ مغلق.**
 */
export const PANEL_PATHS = {
  admin: "/dashboard",
  merchant: "/store",
  customer: "/",
} as const;

export interface Destination {
  /** أصل التطبيق الهدف (فارغ = التطبيق الحالي) */
  origin: string;
  /** المسار داخل التطبيق الهدف */
  path: string;
}

/**
 * homeFor يحدد وجهة المستخدم بعد الدخول حسب دوره.
 * الأولوية: موظفو المنصة ← المتجر ← المندوب ← السائق ← الزبون.
 *
 * والسائق له تطبيقه منذ الآن: كان يُردّ إلى واجهة الزبون لأنه بلا بيت، فيرى
 * متاجر ولا يرى طلباته.
 */
export function homeFor(roles: string[]): Destination {
  const has = (r: string) => roles.includes(r);
  // **والأصلُ فارغٌ — أي «هنا»**: اللوحاتُ أقسامٌ في التطبيق نفسِه.
  if (has("admin") || has("ops") || has("finance"))
    return { origin: "", path: PANEL_PATHS.admin };
  if (has("merchant")) return { origin: "", path: PANEL_PATHS.merchant };
  // **والسائقُ والمندوبُ إلى واجهة الزبون** — لوحتاهما في تطبيقيهما.
  return { origin: "", path: PANEL_PATHS.customer };
}

/**
 * portalFor **مسارُ لوحةِ صاحبِ الحساب — أو لا شيءَ إن كان زبوناً وحدَه.**
 *
 * **وكانت تعيد أصلاً (عنواناً كاملاً)** حين كانت اللوحاتُ تطبيقاتٍ منفصلة.
 * **وصارت مساراً** — ومن قرأها شرطاً («أله لوحة؟») لا يتبدّل عنده شيء.
 */
export function portalFor(roles: string[]): string | null {
  const path = homeFor(roles).path;
  return path === PANEL_PATHS.customer ? null : path;
}

/**
 * goTo ينقل المستخدم إلى وجهته. عبر أصل مختلف يستعمل تسليم الجلسة لمرّة واحدة
 * (SSO) لأن التخزين المحلي لا يُشارَك بين الأصول — فيصل مسجّلاً بلا كلمة مرور.
 */
export async function goTo(dest: Destination, currentOrigin?: string): Promise<void> {
  const here = currentOrigin ?? (typeof window !== "undefined" ? window.location.origin : "");
  if (!dest.origin || dest.origin === here) {
    window.location.href = dest.path;
    return;
  }
  const { code } = await api<{ code: string }>("/api/v1/auth/handoff", { method: "POST" });
  window.location.href = `${dest.origin}/sso?code=${encodeURIComponent(code)}&next=${encodeURIComponent(dest.path)}`;
}

/** يوجّه المستخدم لوجهته حسب دوره بعد تسجيل دخول ناجح. */
export async function routeByRole(user: AuthUser, next?: string): Promise<void> {
  const dest = homeFor(user.roles);
  // `next` يُحترم فقط كمسار نسبي داخل الموقع الحالي (منعاً لإعادة توجيه مفتوحة)
  if (next && next.startsWith("/") && !next.startsWith("//")) {
    window.location.href = next;
    return;
  }
  await goTo(dest);
}

/** مسار نسبي آمن فقط — يُستعمل مع معامل next القادم من العنوان. */
export function safeNext(raw: string | null): string | null {
  if (!raw || !raw.startsWith("/") || raw.startsWith("//")) return null;
  return raw;
}
