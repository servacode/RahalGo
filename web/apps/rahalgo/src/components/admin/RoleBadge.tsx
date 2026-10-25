"use client";

/**
 * شارة الدور الموحدة: لون + أيقونة مميزان لكل دور في كل المنظومة —
 * تمييز بصري فوري في الجداول والبطاقات (ملاحظة مراجعة قسم الحسابات).
 *
 * **والنغمةُ اسمٌ لا سلسلةُ أصناف.** (طلبُ المالك ٢٠٢٦-٠٨-٠٧: المركزيّة.)
 *
 * كانت `cls` سلسلةَ أصنافٍ مكتوبةً هنا — **فبنى هذا الملفُّ شارتَه بيده
 * بجانب `Badge` المركزيّة**، وافترقا في الحشوة والنغمة (`text-accent-dark`
 * حيث تقول الشارةُ `text-accent`).
 *
 * **وأسوأُ منه أنّ من أراد اللونَ وحدَه نشر السلسلة**: كانت شاشةُ الحسابات
 * تكتب `style.cls.split(" ").filter(c => c.startsWith("text-"))` — **جراحةُ
 * نصٍّ على أصنافٍ لتستخرج لوناً**، تنكسر بأوّلِ صنفٍ يُزاد.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { roleLabelByCode } from "@/lib/rolemeta";
import {
  IconRoles,
  IconStatus,
  IconWallet,
  IconStore,
  IconDriver,
  IconUsers,
  IconUser,
} from "@rahalgo/ui";

const m = getMessages(defaultLocale);
// **والاسمُ من `rolemeta` لا من معجمٍ محلّيّ** — مصدرٌ واحد.

interface RoleStyle {
  /** نغمةُ الشارة المركزيّة. */
  variant: "neutral" | "primary" | "accent" | "success" | "warning" | "danger" | "info" | "violet";
  /** لونُ النصِّ وحدَه — لمن أراد الرقمَ بلون الدور بلا شارة. */
  text: string;
  /** **تعبئةٌ صلبةٌ بلون الدور** — والنصُّ والأيقونةُ أبيضان (قرارُ المالك ٢٠٢٦-١٠-٢٥). */
  solid: string;
  Icon: React.ComponentType<{ size?: number }>;
}

const FALLBACK: RoleStyle = { variant: "neutral", text: "text-ink-muted", solid: "bg-info-solid", Icon: IconUser };

export const ROLE_STYLES: Record<string, RoleStyle> = {
  admin: { variant: "danger", text: "text-danger", solid: "bg-danger-solid", Icon: IconRoles },
  ops: { variant: "info", text: "text-info", solid: "bg-info-solid", Icon: IconStatus },
  finance: { variant: "success", text: "text-success", solid: "bg-success-solid", Icon: IconWallet },
  merchant: { variant: "accent", text: "text-accent", solid: "bg-warning-solid", Icon: IconStore },
  driver: { variant: "primary", text: "text-primary", solid: "bg-primary-solid", Icon: IconDriver },
  sales: { variant: "violet", text: "text-violet", solid: "bg-violet-solid", Icon: IconUsers },
  customer: { variant: "neutral", text: "text-ink-muted", solid: "bg-sky-solid", Icon: IconUser },
};

export default function RoleBadge({ role }: { role: string }) {
  const s = ROLE_STYLES[role] ?? FALLBACK;
  return (
    <span className={`inline-flex items-center gap-1 rounded-badge px-2.5 py-1 text-xs font-bold text-on-solid ${s.solid}`}>
      <s.Icon size={13} />
      {roleLabelByCode(role)}
    </span>
  );
}
