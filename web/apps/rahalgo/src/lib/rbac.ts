/**
 * ══════════════════════════════════════════════════════════════════════
 * **الأدوارُ والصلاحيّاتُ — من المحرّك لا من الواجهة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (دورةُ ٧٠ب-و١.)
 *
 * # العطبُ الذي أُغلق
 *
 * **وكانت الشاشاتُ تقرأ `Object.keys(ROLE_LABELS)`** — **معجمَ نصوصٍ
 * في الواجهة**، **فتعرض سبعةَ أدوارٍ والقاعدةُ فيها أحدَ عشر.**
 * **فلا يُرى `owner_super_admin` ولا `trust_safety` ولا `analytics`
 * في أيّ قائمة** — **ودورٌ يُنشَأ اليومَ لا يظهر حتّى تُبنى الواجهةُ
 * من جديد.**
 *
 * **وذاك حقيقةٌ ثانيةٌ في العميل** — **ومصدران للشيء الواحد يفترقان
 * يوماً**، وقد افترقا.
 *
 * # والمعجمُ يبقى — معيناً للعرض لا مصدراً
 *
 * **والأدوارُ المبذورةُ تحمل مفاتيحَ ترجمة** (`roles.admin`)،
 * **ودورٌ يُنشئه الأدمنُ يحمل اسمَه نصّاً.** **فيُعرَض ما في المعجم إن
 * وُجد، وإلّا عُرض ما جاء من المحرّك.** **ومن فرض مفتاحاً على اسمٍ
 * حرٍّ أظهر `roles.xyz` لإنسانٍ يقرأ.**
 *
 * # ولا فحصَ تخويلٍ هنا
 *
 * **وإخفاءُ زرٍّ لطفٌ بالعين لا حراسة** — **والمنعُ في المحرّك**
 * (`roles.manage` في جدول السياسة). **ومن حرس بالواجهة وحدَها حرس
 * بابَ بيتٍ بستارة.**
 */

import { api } from "@/lib/api";

// **واسمُ العرضِ في `rolemeta` وحدَه** — **وخمسُ شاشاتٍ كانت تقرأ
// المعجمَ كلُّ واحدةٍ بنفسها فتكتب `LABELS[r] ?? r` من جديد.**
export { roleLabel, roleLabelByCode, roleDescription, capabilityLabel } from "@/lib/rolemeta";

/** Role **دورٌ كما يقوله المحرّك.** */
export interface Role {
  code: string;
  /** **مفتاحُ ترجمةٍ للمبذور، واسمٌ نصّيٌّ لما يُنشَأ اليوم.** */
  name_key: string;
  capabilities: string[];
  members: number;
  /** **صنفُ الدور من المحرّك** (`authz.ClassOf`) — يرتّب الأقسام. */
  class?: "protected" | "elevated" | "staff" | "custom" | "account_type" | "legacy";
  /** **أتُبدَّل قدراتُه؟** — لا للمالك الأعلى ولا لأنواع الحسابات. */
  editable?: boolean;
  /** **أيُحذَف؟** — مخصَّصٌ أو قديمٌ بلا حامل. */
  deletable?: boolean;
}

/** Capability **قدرةٌ من معجم الشيفرة** — تُسنَد ولا تُخترَع. */
export interface Capability {
  code: string;
  description?: string;
  /** مجموعتُها في المصفوفة — سبعٌ بترتيبٍ ثابت. */
  group?: string;
  /** خطرُها: `money` أو `power` — يُعلَّم بالأحمر. */
  risk?: "" | "money" | "power";
}

/** مجموعاتُ القدرات بترتيب المالك (قرارُ ٢٠٢٦-١٠-٠٤). */
export const CAPABILITY_GROUPS = [
  "orders",
  "accounts",
  "money",
  "stores_drivers",
  "settings",
  "content",
  "security",
] as const;

/** RoleMember **حاملُ الدور** — بلا هاتف (الرقمُ قدرةٌ لحالها). */
export interface RoleMember {
  id: string;
  full_name: string;
  status: string;
  granted_at: string;
}

/** RoleImpact **أثرُ المنح قبل وقوعه.** */
export interface RoleImpact {
  active_members: number;
  would_gain: number;
  risk: "" | "money" | "power";
  block: string;
}

/** listRoles **كلُّ الأدوار وقدراتُها وعددُ أصحابها.** */
export async function listRoles(): Promise<Role[]> {
  const out = await api<Role[] | { roles?: Role[] }>("/api/v1/admin/roles");
  return Array.isArray(out) ? out : (out.roles ?? []);
}

/** listCapabilities **معجمُ القدرات كما تعرفه الشيفرة.** */
export async function listCapabilities(): Promise<Capability[]> {
  const out = await api<
    Capability[] | string[] | { capabilities?: Array<Capability | string> }
  >("/api/v1/admin/capabilities");
  const raw = Array.isArray(out) ? out : (out.capabilities ?? []);
  return raw.map((c) => (typeof c === "string" ? { code: c } : c));
}

/**
 * createRole **دورٌ جديدٌ بلا قدرة.**
 *
 * **ويُنشَأ فارغاً عمداً** — **وهو الافتراضُ في `ADG-1`**: ثمّ تُمنَح
 * قدراتُه واحدةً واحدةً بفعلٍ مؤكَّدٍ مستقلّ، **فيُقرأ في السجلّ ما
 * مُنح ومتى.**
 */
export function createRole(code: string, name: string): Promise<Role> {
  return api<Role>("/api/v1/admin/roles", {
    method: "POST",
    body: JSON.stringify({ code, name }),
  });
}

/**
 * grantCapability **منحُ قدرةٍ لدور** — فعلٌ يحتاج تأكيداً بكلمة السرّ،
 * **وسببٌ مكتوبٌ إلزاميٌّ يُحفظ في السجلّ** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
 */
export function grantCapability(
  role: string,
  capability: string,
  reason: string,
): Promise<{ granted: boolean; changed: boolean }> {
  return api(`/api/v1/admin/roles/${encodeURIComponent(role)}/capabilities`, {
    method: "POST",
    body: JSON.stringify({ capability, reason }),
  });
}

/** revokeCapability **نزعُها** — وفعلٌ يحتاج تأكيداً كذلك. */
export function revokeCapability(
  role: string,
  capability: string,
  reason = "",
): Promise<{ revoked: boolean; changed: boolean }> {
  const q = reason ? `?reason=${encodeURIComponent(reason)}` : "";
  return api(
    `/api/v1/admin/roles/${encodeURIComponent(role)}/capabilities/${encodeURIComponent(capability)}${q}`,
    { method: "DELETE" },
  );
}

/** deleteRole **حذفُ دورٍ فارغ** — مخصَّصٌ أو قديمٌ بلا حامل. */
export function deleteRole(role: string): Promise<unknown> {
  return api(`/api/v1/admin/roles/${encodeURIComponent(role)}`, { method: "DELETE" });
}

/** roleMembers **مَن يحمل الدور.** */
export function roleMembers(role: string): Promise<RoleMember[]> {
  return api<RoleMember[]>(`/api/v1/admin/roles/${encodeURIComponent(role)}/members`);
}

/** roleImpact **كم حساباً سيكسب القدرةَ فعلاً لو مُنحت.** */
export function roleImpact(role: string, capability: string): Promise<RoleImpact> {
  return api<RoleImpact>(
    `/api/v1/admin/roles/${encodeURIComponent(role)}/impact?capability=${encodeURIComponent(capability)}`,
  );
}
