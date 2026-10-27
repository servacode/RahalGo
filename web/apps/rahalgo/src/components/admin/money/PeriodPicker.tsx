"use client";

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Chips, Input } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const L = m.admin.losses;

/**
 * **أزرارُ مدّةٍ جاهزةٌ وتاريخان باسمهما** (فحصُ ٢٠٢٦-١٠-٢٧) — للخسائر والتعويضات.
 *
 * **و«اليوم» يومُ الحساب من الخادم** (`today`، يبدأ بساعة فتح الدوام) — لا منتصفُ ليل المتصفّح:
 * كانت الواجهةُ تبعث بين ١٢ الليل و٩ الصبح يوماً لم يبدأ بعد، فتنزلق الفترةُ يوماً.
 */

/** **يومٌ ± أيّام** — حسابٌ على التاريخ وحدَه (غرينتش) فلا تُزحزحه منطقةُ الجهاز. */
export function shiftDay(iso: string, days: number): string {
  const d = new Date(`${iso}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

export type Preset = "today" | "7" | "30" | "month";

export function presetRange(p: Preset, today: string): { from: string; to: string } {
  switch (p) {
    case "today":
      return { from: today, to: today };
    case "7":
      return { from: shiftDay(today, -6), to: today };
    case "30":
      return { from: shiftDay(today, -29), to: today };
    case "month":
      return { from: `${today.slice(0, 8)}01`, to: today };
  }
}

const PRESETS: { id: Preset; label: string }[] = [
  { id: "today", label: L.presetToday },
  { id: "7", label: L.preset7 },
  { id: "30", label: L.preset30 },
  { id: "month", label: L.presetMonth },
];

export function PeriodPicker({
  from,
  to,
  today,
  onChange,
  allLabel,
}: {
  from: string;
  to: string;
  /** **يومُ الحساب من الخادم** — فارغٌ حتّى يصل أوّلُ ردّ، فتُخفى الأزرارُ الجاهزة. */
  today: string;
  onChange: (from: string, to: string) => void;
  /** **حبّةُ «أيّ وقت»** — لفلترٍ يجوز أن يكون بلا تاريخ (التعويضات). */
  allLabel?: string;
}) {
  const active =
    !from && !to ? "all" : today ? (PRESETS.find((p) => {
      const r = presetRange(p.id, today);
      return r.from === from && r.to === to;
    })?.id ?? "") : "";
  const items = [
    ...(allLabel ? [{ id: "all", label: allLabel }] : []),
    ...(today ? PRESETS : []),
  ];
  return (
    <div className="space-y-3">
      {items.length > 0 && (
        <Chips
          wrap
          items={items}
          value={active}
          onChange={(k) => {
            if (k === "all") onChange("", "");
            else if (today) {
              const r = presetRange(k as Preset, today);
              onChange(r.from, r.to);
            }
          }}
        />
      )}
      <div className="grid grid-cols-2 gap-2">
        <Input
          label={m.admin.compensations.filterFrom}
          type="date"
          dir="ltr"
          value={from}
          onChange={(e) => onChange(e.target.value, to)}
        />
        <Input
          label={m.admin.compensations.filterTo}
          type="date"
          dir="ltr"
          value={to}
          onChange={(e) => onChange(from, e.target.value)}
        />
      </div>
    </div>
  );
}
