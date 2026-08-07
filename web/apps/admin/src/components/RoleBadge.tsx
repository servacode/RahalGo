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
import {
  Badge,
  IconRoles,
  IconStatus,
  IconWallet,
  IconStore,
  IconDriver,
  IconUsers,
  IconUser,
} from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const ROLE_LABELS: Record<string, string> = m.terms.roleNames;

interface RoleStyle {
  /** نغمةُ الشارة المركزيّة. */
  variant: "neutral" | "primary" | "accent" | "success" | "warning" | "danger" | "info" | "violet";
  /** لونُ النصِّ وحدَه — لمن أراد الرقمَ بلون الدور بلا شارة. */
  text: string;
  Icon: React.ComponentType<{ size?: number }>;
}

const FALLBACK: RoleStyle = { variant: "neutral", text: "text-ink-muted", Icon: IconUser };

export const ROLE_STYLES: Record<string, RoleStyle> = {
  admin: { variant: "danger", text: "text-danger", Icon: IconRoles },
  ops: { variant: "info", text: "text-info", Icon: IconStatus },
  finance: { variant: "success", text: "text-success", Icon: IconWallet },
  merchant: { variant: "accent", text: "text-accent", Icon: IconStore },
  driver: { variant: "primary", text: "text-primary", Icon: IconDriver },
  sales: { variant: "violet", text: "text-violet", Icon: IconUsers },
  customer: { variant: "neutral", text: "text-ink-muted", Icon: IconUser },
};

export default function RoleBadge({ role }: { role: string }) {
  const s = ROLE_STYLES[role] ?? FALLBACK;
  return (
    <Badge variant={s.variant} className="gap-1 px-2">
      <s.Icon size={12} />
      {ROLE_LABELS[role] ?? role}
    </Badge>
  );
}
