"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **محرّرُ جدولِ أسبوعٍ — يُكتب مرّةً ويُستعمل مرّتين** (`ZH`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ودوامُ المنصّة وأوقاتُ المناطق جدولان بالاصطلاح عينِه**: **يومٌ بلا
 * فتراتٍ مغلقٌ بذاته، وفترةٌ تعبر منتصفَ الليل، والبدءُ داخلٌ والانتهاءُ
 * خارج.**
 *
 * **ونسختان من الشاشة تفترقان يوماً** — **فيصلح أحدُهما ويبقى الآخر،
 * فتقبل شاشةٌ ما تردّه الأخرى.** **وهو العطبُ عينُه الذي مُنع في
 * المحرّك** حين استعملت المناطقُ `Schedule` عينَها.
 *
 * **ولا حكمَ هنا** — **والتحقّقُ في المحرّك**: `Validate` تردّ التداخلَ
 * والفترةَ الفارغة، **ونداءٌ مباشرٌ يتجاوز هذه الشاشة.**
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, Badge, IconAdd, IconDelete } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const P = m.admin.platformHours;

export interface Win {
  day_of_week: number;
  start: string;
  end: string;
}

/** **أتعبر الفترةُ منتصفَ الليل؟** — الاصطلاحُ عينُه الذي في المحرّك. */
export function crossesMidnight(w: Win): boolean {
  return w.end <= w.start;
}

interface Props {
  windows: Win[];
  onChange: (next: Win[]) => void;
  disabled?: boolean;
}

export default function WeekHours({ windows, onChange, disabled }: Props) {
  function add(day: number) {
    onChange([...windows, { day_of_week: day, start: "09:00", end: "17:00" }]);
  }
  function remove(idx: number) {
    onChange(windows.filter((_, i) => i !== idx));
  }
  function edit(idx: number, patch: Partial<Win>) {
    onChange(windows.map((w, i) => (i === idx ? { ...w, ...patch } : w)));
  }

  // ── موعدٌ واحدٌ لعدّة أيّام (طلبُ المالك ٢٠٢٦-١٠-٠٥) ────────────
  //
  // «لازم نقدر نحدد موعد نطبقه على جميع الأيام أو نحدد الأيام» — **وسبعةُ
  // أيّامٍ تُعبّأ واحداً واحداً سبعةُ فرصٍ للخطأ.** **والتطبيقُ يستبدل فتراتِ
  // الأيّام المختارة** ولا يُضيف إليها: من طبّق مرّتين لا يجد فترتين متداخلتين.
  const [qStart, setQStart] = useState("09:00");
  const [qEnd, setQEnd] = useState("23:00");
  const [qDays, setQDays] = useState<number[]>(() => P.days.map((_: string, i: number) => i));
  const allPicked = qDays.length === P.days.length;
  function toggleDay(day: number) {
    setQDays((d) => (d.includes(day) ? d.filter((x) => x !== day) : [...d, day].sort()));
  }
  function applyQuick() {
    const kept = windows.filter((w) => !qDays.includes(w.day_of_week));
    onChange([...kept, ...qDays.map((day) => ({ day_of_week: day, start: qStart, end: qEnd }))]);
  }

  return (
    <div className="grid grid-cols-1 gap-3">
      {!disabled && (
        <div className="space-y-2 rounded border border-accent p-3">
          <span className="text-sm font-medium">{P.quickTitle}</span>
          <p className="text-xs text-ink-muted">{P.quickHint}</p>
          <div className="flex flex-wrap items-center gap-2">
            <span className="text-xs text-ink-muted">{P.from}</span>
            <Input type="time" value={qStart} onChange={(e) => setQStart(e.target.value)} />
            <span className="text-xs text-ink-muted">{P.to}</span>
            <Input type="time" value={qEnd} onChange={(e) => setQEnd(e.target.value)} />
          </div>
          <div className="flex flex-wrap gap-2">
            <Button
              variant={allPicked ? "primary" : "secondary"}
              onClick={() => setQDays(allPicked ? [] : P.days.map((_: string, i: number) => i))}
            >
              {P.allDays}
            </Button>
            {P.days.map((label: string, day: number) => (
              <Button
                key={day}
                variant={qDays.includes(day) ? "primary" : "ghost"}
                onClick={() => toggleDay(day)}
                aria-pressed={qDays.includes(day)}
              >
                {label}
              </Button>
            ))}
          </div>
          <Button onClick={applyQuick} disabled={qDays.length === 0 || !qStart || !qEnd}>
            {P.apply}
          </Button>
        </div>
      )}
      {P.days.map((label: string, day: number) => {
        const rows = windows
          .map((w, i) => ({ w, i }))
          .filter((x) => x.w.day_of_week === day);
        return (
          <div key={day} className="rounded border border-line p-3">
            <div className="mb-2 flex items-center justify-between">
              <span className="text-sm font-medium">{label}</span>
              {rows.length === 0 && <Badge>{P.closedDay}</Badge>}
            </div>
            <div className="space-y-2">
              {rows.map(({ w, i }) => (
                <div key={i} className="flex flex-wrap items-center gap-2">
                  <span className="text-xs text-ink-muted">{P.from}</span>
                  <Input
                    type="time"
                    value={w.start}
                    disabled={disabled}
                    onChange={(e) => edit(i, { start: e.target.value })}
                  />
                  <span className="text-xs text-ink-muted">{P.to}</span>
                  <Input
                    type="time"
                    value={w.end}
                    disabled={disabled}
                    onChange={(e) => edit(i, { end: e.target.value })}
                  />
                  {crossesMidnight(w) && <Badge>{P.crosses}</Badge>}
                  {!disabled && (
                    <Button variant="ghost" onClick={() => remove(i)} aria-label={P.closedDay}>
                      <IconDelete />
                    </Button>
                  )}
                </div>
              ))}
            </div>
            {!disabled && (
              <Button variant="ghost" onClick={() => add(day)}>
                <IconAdd />
                {P.addWindow}
              </Button>
            )}
          </div>
        );
      })}
    </div>
  );
}
