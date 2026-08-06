"use client";

/**
 * شارة الدور الموحدة: لون + أيقونة مميزان لكل دور في كل المنظومة —
 * تمييز بصري فوري في الجداول والبطاقات (ملاحظة مراجعة قسم الحسابات).
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
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
const ROLE_LABELS: Record<string, string> = m.terms.roleNames;

interface RoleStyle {
  cls: string;
  Icon: React.ComponentType<{ size?: number }>;
}

const FALLBACK: RoleStyle = { cls: "border border-line bg-field text-ink-muted", Icon: IconUser };

export const ROLE_STYLES: Record<string, RoleStyle> = {
  admin: { cls: "bg-danger-tint text-danger", Icon: IconRoles },
  ops: { cls: "bg-info-tint text-info", Icon: IconStatus },
  finance: { cls: "bg-success-tint text-success", Icon: IconWallet },
  merchant: { cls: "bg-accent-tint text-accent-dark", Icon: IconStore },
  driver: { cls: "bg-primary-tint text-primary", Icon: IconDriver },
  sales: { cls: "bg-violet-tint text-violet", Icon: IconUsers },
  customer: { cls: "border border-line bg-field text-ink-muted", Icon: IconUser },
};

export default function RoleBadge({ role }: { role: string }) {
  const s = ROLE_STYLES[role] ?? FALLBACK;
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-badge px-2 py-0.5 text-xs font-medium ${s.cls}`}
    >
      <s.Icon size={12} />
      {ROLE_LABELS[role] ?? role}
    </span>
  );
}
