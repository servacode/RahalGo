"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قاموسُ الحالات الواحد — اسمٌ ونبرةٌ وأيقونة** (دفعةُ تصميم اللوحة ١، ٢٠٢٦-١٠-٣١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * كانت كلُّ صفحةٍ تكتب خريطتَها: «معلَّق» أصفرُ هنا وأزرقُ هناك، و«مرفوض» رماديٌّ في السحوبات
 * وأحمرُ في الطلبات. **فصار للحالة لونٌ واحدٌ وأيقونةٌ واحدة** — والأيقونةُ مع النصّ دائماً،
 * فلا يُقرأ المعنى باللون وحدَه.
 *
 * والحارسُ (`check-admin-design.mjs`) يمنع خريطةَ ألوانٍ للحالات خارجَ هذا الملفّ.
 */

import type { ComponentType } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge } from "./components";
import { IconError, IconHourglass, IconMinus, IconReturn, IconStatus, IconSuccess } from "./icons";

const m = getMessages(defaultLocale);

/** **النبرةُ معنًى لا لون** — وكلُّ نبرةٍ لها شارةٌ وأيقونة. */
export type StatusTone = "attention" | "progress" | "good" | "bad" | "neutral" | "special";
export type StatusDomain =
  | "order"
  | "payout"
  | "debt"
  | "complaint"
  | "emergency"
  /** **حضورُ السائق** — متصل · مشغول · فاضي · غير متصل. */
  | "presence"
  /** **حالةُ المتجر** — فاتح · سكرّان · موقوف · محظور. */
  | "store"
  /** **طلبُ انضمام**. */
  | "lead";

type IconT = ComponentType<{ size?: number; className?: string }>;

const TONE_BADGE: Record<StatusTone, "warning" | "info" | "success" | "danger" | "neutral" | "violet"> = {
  attention: "warning",
  progress: "info",
  good: "success",
  bad: "danger",
  neutral: "neutral",
  special: "violet",
};

const TONE_ICON: Record<StatusTone, IconT> = {
  attention: IconHourglass,
  progress: IconStatus,
  good: IconSuccess,
  bad: IconError,
  neutral: IconMinus,
  special: IconReturn,
};

const TONES: Record<StatusDomain, Record<string, StatusTone>> = {
  order: {
    pending: "attention",
    accepted: "progress",
    preparing: "progress",
    dispatching: "attention",
    assigned: "progress",
    at_pickup: "progress",
    picked_up: "progress",
    on_the_way: "progress",
    at_dropoff: "progress",
    delivered: "good",
    rejected: "bad",
    cancelled: "neutral",
    failed: "bad",
    refunded: "special",
  },
  payout: {
    pending: "attention",
    processing: "progress",
    paid: "good",
    rejected: "neutral",
    failed: "bad",
    reversed: "special",
  },
  debt: {
    open: "attention",
    paid: "good",
    voided: "neutral",
    written_off: "special",
  },
  complaint: {
    open: "attention",
    in_progress: "progress",
    awaiting_finance: "attention",
    resolved: "good",
  },
  emergency: {
    open: "bad",
    acked: "attention",
    resolved: "good",
  },
  // **الأخضرُ تمام · الأصفرُ ناطر · الأحمرُ مشكلة · الأزرقُ شغلٌ جارٍ · الرماديُّ لا يعنينا.**
  presence: { online: "good", busy: "progress", free: "good", offline: "neutral" },
  store: { open: "good", closed: "neutral", paused: "attention", banned: "bad" },
  lead: { new: "attention", contacted: "progress", approved: "good", rejected: "neutral" },
};

const LABELS: Record<StatusDomain, Record<string, string>> = {
  order: m.orders.status as Record<string, string>,
  payout: m.shared.payoutAdmin.status as Record<string, string>,
  debt: m.admin.obligations.states as Record<string, string>,
  complaint: m.admin.tickets.status as Record<string, string>,
  emergency: m.admin.kit.emergency as Record<string, string>,
  presence: m.admin.kit.presence as Record<string, string>,
  store: m.admin.kit.storeState as Record<string, string>,
  lead: m.admin.kit.lead as Record<string, string>,
};

export interface StatusMeta {
  label: string;
  tone: StatusTone;
  icon: IconT;
  badge: (typeof TONE_BADGE)[StatusTone];
}

/** **اسمُ الحالة ونبرتُها وأيقونتُها** — وحالةٌ مجهولةٌ تُعرض محايدةً باسمها كما هو. */
export function statusMeta(domain: StatusDomain, status: string): StatusMeta {
  const tone = TONES[domain][status] ?? "neutral";
  return {
    label: LABELS[domain][status] ?? status,
    tone,
    icon: TONE_ICON[tone],
    badge: TONE_BADGE[tone],
  };
}

/** **شارةُ الحالة** — نصٌّ وأيقونةٌ ولونٌ من القاموس الواحد. */
export function StatusBadge({
  domain,
  status,
  className = "",
}: {
  domain: StatusDomain;
  status: string;
  className?: string;
}) {
  const s = statusMeta(domain, status);
  const Icon = s.icon;
  return (
    <Badge variant={s.badge} className={`gap-1 ${className}`}>
      <Icon size={12} className="shrink-0" />
      {s.label}
    </Badge>
  );
}


/**
 * **لونُ النبرة نصّاً وحافّةً** — للأرقام والحواف، حتى لا تكتب صفحةٌ لوناً بيدها.
 * (`good` أخضر · `attention` أصفر · `bad` أحمر · `progress` أزرق · الباقي هادئ.)
 */
export const TONE_TEXT: Record<StatusTone, string> = {
  good: "text-success",
  attention: "text-warning",
  bad: "text-danger",
  progress: "text-info",
  special: "text-violet",
  neutral: "text-ink-muted",
};
