"use client";

import type { ComponentType } from "react";
import type { TileTone } from "@/components/admin/DashParts";

/**
 * **مربّعُ رقمٍ — يُضغط فقط لمن له فعل** (فحصُ ٢٠٢٦-١٠-٢٧).
 *
 * شكلُ `Tile` المشترك نفسُه، **لكنّه بلا `onClick` يُرسم صندوقاً لا زرّاً**: كان كلُّ مربّعٍ يتلوّن
 * تحت الفأرة ويُضغط فلا يفعل شيئاً، فيُقرأ القسمُ معطّلاً. و`note` سطرٌ يقول عن أيّ مدّةٍ هو الرقم.
 */
const ICON_TONE: Record<TileTone, string> = {
  default: "bg-field text-ink",
  muted: "bg-field text-ink-muted",
  danger: "bg-danger-solid text-on-solid",
  warning: "bg-warning-solid text-on-solid",
  accent: "bg-accent text-on-bright",
  success: "bg-success-solid text-on-solid",
};

export function StatTile({
  icon: Icon,
  label,
  value,
  sub,
  tone = "default",
  onClick,
  highlight = false,
}: {
  icon: ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: string;
  sub?: string;
  tone?: TileTone;
  onClick?: () => void;
  /** **مربّعٌ مختلفُ المدّة** — «الآن» بين أرقام الفترة: إطارٌ متقطّع يميّزه. */
  highlight?: boolean;
}) {
  const border = tone === "danger" ? "border-danger" : tone === "warning" ? "border-warning" : "border-line-soft";
  const cls = `flex w-full items-center gap-3 rounded-card border p-3 text-start ${border} ${
    highlight ? "border-dashed bg-field" : ""
  }`;
  const body = (
    <>
      <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${ICON_TONE[tone]}`}>
        <Icon size={18} />
      </span>
      <span className="min-w-0">
        <span className={`figure block leading-none ${tone === "muted" ? "text-ink-muted" : ""}`}>{value}</span>
        <span className="mt-1 block truncate text-xs text-ink-muted">{label}</span>
        {sub && <span className="block truncate text-2xs text-ink-muted">{sub}</span>}
      </span>
    </>
  );
  if (!onClick) return <div className={cls}>{body}</div>;
  return (
    <button type="button" onClick={onClick} className={`${cls} transition-colors hover:bg-row-hover`}>
      {body}
    </button>
  );
}
