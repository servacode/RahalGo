"use client";

/** **أدواتٌ مشتركةٌ لتبويبات الخزينة** — رقمٌ بإشارته ولونه، وشارةُ حال. */

import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);
export const T = m.admin.treasury;

/** **مبلغٌ بإشارته** — الداخلُ أخضر والخارجُ أحمر، وبأرقامٍ غربيّة. */
export function Signed({ value, className = "" }: { value: number; className?: string }) {
  const tone = value > 0 ? "text-success" : value < 0 ? "text-danger" : "text-ink-muted";
  return (
    <span dir="ltr" className={`font-bold tabular-nums ${tone} ${className}`}>
      {value > 0 ? "+" : value < 0 ? "−" : ""}
      {fmtNum(Math.abs(value))}
    </span>
  );
}

/** **مبلغٌ بلا إشارة.** */
export function Amount({ value }: { value: number }) {
  return (
    <span dir="ltr" className="font-bold tabular-nums">
      {fmtNum(value)}
    </span>
  );
}

/** **رسالةُ الخادم أو نصٌّ احتياطيّ.** */
export function sectionLabel(key: string): string {
  const s = T.approvals.sections as Record<string, string>;
  return s[key] ?? s.other ?? key;
}
