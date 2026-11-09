"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **فقاعةُ الشرح وسهمُها — مرآةُ `GuideBubble` في التطبيق** (قرارُ المالك ٢٠٢٦-١١-٠٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بلون زرّ «إنشاء حساب» نفسِه** (`cta-orange`) بخطٍّ أبيض، وسهمٌ إلى تحت
 * نحو الحقل. **وتُوضع في سطرها فوق الحقل** لا طبقةً فوقه — فلا تحجب الكتابة.
 * **ومن أطفأ الحركةَ في نظامه** يرى سهماً ساكناً (`motion-safe:`، والثيمُ
 * يسكّن كلَّ حركةٍ مركزيّاً).
 */

import { IconChevronDown } from "./icons";

export function GuideBubble({
  text,
  skipLabel,
  onSkip,
}: {
  text: string;
  skipLabel: string;
  onSkip: () => void;
}) {
  return (
    <div className="flex w-full flex-col items-center" role="note">
      <div className="flex w-full items-center gap-2 rounded-control bg-cta-orange ps-3 pe-1 text-on-solid">
        <p className="flex-1 py-2 text-sm font-bold">{text}</p>
        <button
          type="button"
          onClick={onSkip}
          className="min-h-11 shrink-0 rounded-control px-3 py-2 text-xs text-on-solid underline-offset-2 hover:underline"
        >
          {skipLabel}
        </button>
      </div>
      <IconChevronDown aria-hidden size={22} className="text-cta-orange motion-safe:animate-bounce" />
    </div>
  );
}

/** **إضاءةٌ خفيفةٌ حول الزرّ المقصود** — تنبض، أو تسكن لمن أطفأ الحركة. */
export function GuideSpotlight({ active, children }: { active: boolean; children: React.ReactNode }) {
  if (!active) return <>{children}</>;
  return (
    <div className="relative w-full rounded-card p-1">
      <span
        aria-hidden
        className="pointer-events-none absolute inset-0 rounded-card bg-cta-orange opacity-25 motion-safe:animate-pulse"
      />
      <div className="relative">{children}</div>
    </div>
  );
}
