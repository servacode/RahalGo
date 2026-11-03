"use client";

import { IconCheck, IconClose } from "./icons";
import type { RailStage } from "./ordertrackv";

/**
 * **مسارُ الطلب شريطاً أفقيّاً تحت رقم الطلب** (قرارُ المالك ٢٠٢٦-١١-٠٣: «هذا يصير شريط
 * افقي تحت رقم الطلب») — بدل العمود في طرف البطاقة.
 *
 * كلُّ مرحلةٍ نقطةٌ على خطٍّ واحد واسمُها تحتها، **والحاليّةُ مملوءةٌ بلونها**، وما مضى ملوّن،
 * وما بقي باهت. **والمدّةُ تحت اسم المرحلة التي وصلها** — والساعةُ الدقيقةُ في التلميح.
 * والمنتهي قبل أن يصل (`current < 0`) رماديٌّ كلُّه.
 */
export function OrderTrackH({
  stages,
  current,
  className = "",
}: {
  stages: RailStage[];
  current: number;
  className?: string;
}) {
  const ended = current < 0;
  const last = stages.length - 1;
  return (
    <ol className={`flex w-full items-start ${className}`}>
      {stages.map((s, i) => {
        const reached = !ended && i <= current;
        const now = !ended && i === current;
        const passed = !ended && i < current;
        return (
          <li key={s.id} className="relative flex min-w-0 flex-1 flex-col items-center text-center" title={s.at ?? undefined}>
            {/* نصفُ الخطّ قبل النقطة ونصفُه بعدها — فالأوّلُ والأخيرُ لا يخرجان عن البطاقة. */}
            {i > 0 && (
              <span aria-hidden className={`absolute top-[7px] h-0.5 w-1/2 start-0 ${reached ? "bg-success" : "bg-line"}`} />
            )}
            {i < last && (
              <span aria-hidden className={`absolute top-[7px] h-0.5 w-1/2 end-0 ${passed ? "bg-success" : "bg-line"}`} />
            )}
            {/* **والمرحلةُ اللي خلصت إشارةُ صحٍّ خضرا** (قرارُ المالك ٢٠٢٦-١١-٠٣). */}
            <span
              className={`relative z-10 flex h-4 w-4 items-center justify-center rounded-badge border-2 ${
                now
                  ? "border-accent bg-accent ring-4 ring-accent-tint"
                  : passed
                    ? "border-success bg-success text-on-bright"
                    : "border-line bg-surface"
              }`}
            >
              {passed && <IconCheck size={10} strokeWidth={3} />}
            </span>
            <span
              className={`mt-1.5 px-0.5 text-2xs leading-tight ${
                now ? "font-bold text-ink" : reached ? "text-ink-muted" : "text-ink-dim opacity-70"
              }`}
            >
              {s.label}
            </span>
            {reached && i > 0 && s.span && (
              <span className="mt-0.5 inline-flex items-center gap-0.5 text-2xs text-ink-dim">
                <span className="tabular-nums">{s.span}</span>
                {s.late === true && <IconClose size={11} className="text-danger" />}
                {s.late === false && <IconCheck size={11} className="text-success" />}
              </span>
            )}
          </li>
        );
      })}
    </ol>
  );
}
