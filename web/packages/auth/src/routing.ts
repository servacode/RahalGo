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

export const APP_URLS = {
  admin: () => process.env.NEXT_PUBLIC_ADMIN_URL ?? "http://localhost:3001",
  merchant: () => process.env.NEXT_PUBLIC_MERCHANT_URL ?? "http://localhost:3002",
  customer: () => process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003",
  rep: () => process.env.NEXT_PUBLIC_REP_URL ?? "http://localhost:3004",
  driver: () => process.env.NEXT_PUBLIC_DRIVER_URL ?? "http://localhost:3005",
};

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
  if (has("admin") || has("ops") || has("finance"))
    return { origin: APP_URLS.admin(), path: "/dashboard" };
  if (has("merchant")) return { origin: APP_URLS.merchant(), path: "/portal" };
  if (has("sales")) return { origin: APP_URLS.rep(), path: "/portal" };
  if (has("driver")) return { origin: APP_URLS.driver(), path: "/portal" };
  return { origin: APP_URLS.customer(), path: "/" };
}

/** هل للمستخدم لوحة تحكم خاصة (غير واجهة الزبون)؟ */
export function portalFor(roles: string[]): string | null {
  const dest = homeFor(roles);
  return dest.origin === APP_URLS.customer() ? null : dest.origin;
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
