"use client";

/**
 * **مسارُ الطلب أفقياً** — الحالةُ في البطاقة لا في صفحةٍ أخرى.
 *
 * # لماذا أفقيّ ولماذا هنا
 *
 * الخطُّ الزمنيُّ الرأسيّ (`Timeline`) يشرح **ما وقع ومتى** — وهو صفحةُ تفصيل.
 * **والزبونُ في قائمة طلباته يسأل سؤالاً واحداً: أين طلبي الآن؟** وجوابُه لا
 * يستحقّ انتقالاً إلى صفحة.
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «حالة الطلب تكون بنفس الكرت، لا يوجد داعي
 * للانتقال إلى صفحة التفاصيل · نكبّر الكرت شوي والحالة تصير بشكل أفقيّ».
 *
 * # والدرّاجةُ تمشي على المراحل
 *
 * «شو رأيك نخليه أيقونة موتسكل هي تتحرك وتمشي المراحل» — **وهي أصدقُ ما يقوله
 * تطبيقُ توصيل**: الشريطُ يقول «كم بقي» والدرّاجةُ تقول «أنا في الطريق».
 *
 * **وتتحرّك في الجاري وحدَه.** طلبٌ سُلّم أو أُلغي **درّاجتُه ساكنة** — وحركةٌ
 * في عشر بطاقاتٍ مغلقةٍ ضجيجٌ يُتعب العين ولا يدلّ على شيء.
 *
 * **وتسكن لمن أطفأ الحركة في نظامه** (`prefers-reduced-motion`) — فمن طلب
 * السكون طلبه لسببٍ يخصّه.
 *
 * # وما ينتهي قبل أن يصل
 *
 * الملغى والمرفوض والمُخفق **لا مسارَ لهم**: شريطٌ يتوقّف في منتصفه يُقرأ
 * «عالق» لا «انتهى». **فيُعرض سطرٌ واحدٌ يقول أين توقّف ولماذا** — والحقيقةُ
 * أوضحُ من رسمٍ جميلٍ يكذب.
 */

import type { ComponentType } from "react";

export interface TrackStage {
  id: string;
  label: string;
  icon?: ComponentType<{ size?: number; className?: string; strokeWidth?: number }>;
}

export function OrderTrack({
  stages,
  /** رقمُ المرحلة الحالية — و`-1` لما لم يبدأ */
  current,
  /** أيقونةُ المركبة التي تمشي على المسار */
  vehicle: Vehicle,
  /** **الحركةُ للجاري وحدَه** — المغلقُ يسكن */
  live = false,
  className = "",
}: {
  stages: TrackStage[];
  current: number;
  vehicle: ComponentType<{ size?: number; className?: string; strokeWidth?: number }>;
  live?: boolean;
  className?: string;
}) {
  const last = Math.max(1, stages.length - 1);
  const at = Math.min(Math.max(current, 0), stages.length - 1);
  // **النسبةُ من الطرف** — والصفحةُ من اليمين إلى اليسار، فالبدايةُ يمين.
  const pct = (at / last) * 100;

  return (
    <div className={`w-full ${className}`}>
      {/* الدرّاجةُ فوق الشريط — تنزلق إلى موضع المرحلة */}
      <div className="relative mb-1 h-6">
        <span
          className={`absolute top-0 -translate-x-1/2 text-primary ${
            live ? "motion-safe:animate-[rahalgo-ride_1.6s_ease-in-out_infinite]" : ""
          }`}
          style={{
            // **`right` لا `left`**: المسارُ يبدأ من اليمين في واجهةٍ عربية،
            // ولو استُعمل `left` لَمشت الدرّاجةُ عكسَ القراءة.
            right: `${pct}%`,
            transform: "translateX(50%)",
            transition: "right 600ms cubic-bezier(0.4, 0, 0.2, 1)",
          }}
          aria-hidden
        >
          <Vehicle size={20} />
        </span>
      </div>

      {/* القضيبُ والعُقَد */}
      <div className="relative h-1.5" aria-hidden>
        <div className="absolute inset-0 rounded-badge bg-page" />
        <div
          className="absolute inset-y-0 right-0 rounded-badge bg-primary transition-[width] duration-700 ease-out"
          style={{ width: `${pct}%` }}
        />
        {stages.map((s, i) => (
          <span
            key={s.id}
            className={`absolute top-1/2 h-3 w-3 -translate-y-1/2 translate-x-1/2 rounded-full border-2 transition-colors ${
              i <= at ? "border-primary bg-primary" : "border-line bg-surface"
            }`}
            style={{ right: `${(i / last) * 100}%` }}
          />
        ))}
      </div>

      {/* الأسماءُ تحت العُقَد — **والحاليّةُ وحدَها داكنة** */}
      <div className="mt-1.5 flex justify-between text-[10px] leading-tight">
        {stages.map((s, i) => (
          <span
            key={s.id}
            className={`flex-1 text-center ${
              i === at
                ? "font-bold text-primary-dark"
                : i < at
                  ? "text-ink-muted"
                  : "text-ink-muted opacity-50"
            }`}
          >
            {s.label}
          </span>
        ))}
      </div>
    </div>
  );
}
