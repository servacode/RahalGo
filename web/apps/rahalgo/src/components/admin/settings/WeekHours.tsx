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

  return (
    <div className="grid grid-cols-1 gap-3">
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
