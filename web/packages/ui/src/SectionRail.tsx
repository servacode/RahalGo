"use client";

/**
 * **شريطُ الأقسام** — صورٌ دائريّةٌ تمشي وحدَها وتُساق باليد.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «شريطٌ أفقيٌّ لصورٍ دائريّةٍ فيه الأصناف، يذهب
 * يميناً ويساراً تلقائيّاً ويدويّاً، ومن ضغط الصورةَ ظهر ما بها من عناصر».)
 *
 * # لماذا دائريّةٌ لا مربّعة
 *
 * **الدائرةُ تقول «اختَرني» والمربّعُ يقول «اقرأني».** والقسمُ ليس بضاعةً
 * تُشترى — **هو مِصفاةٌ تُضغط**، وشكلُه يجب أن يفرّقه عن البطاقات تحته.
 * **وشبكةُ مربّعاتٍ فوق شبكةِ مربّعاتٍ تُقرأ كتلةً واحدة.**
 *
 * # والمشيُ ذهاباً وإياباً لا دورةً لا تنتهي
 *
 * **الشريطُ اللانهائيُّ يحتاج نسخَ القائمة** — فيُرسم القسمُ مرّتين، **ومن
 * ضغط النسخةَ الثانيةَ يظنّ أنّه رأى قسمين متشابهين.** والذهابُ والإيابُ يُري
 * كلَّ الأقسام ولا يكذب على العين، **وهو ما وصفه المالك: «يمين ويسار».**
 *
 * # ويقف عند أوّل لمسة
 *
 * **شريطٌ يمشي تحت الإصبع لا يُضغط**: يزيح الهدفَ في اللحظة التي يُقصد فيها.
 * **فيقف عند التحويم واللمس والتركيز** — ولا يعود حتّى تُرفع اليد.
 *
 * # واتّجاهُ التمرير يُتحسَّس ولا يُفترض
 *
 * `scrollLeft` في صفحةٍ عربيّةٍ **يعدّ من الصفر إلى السالب** في المتصفّحات
 * التي تتبع المعيار، **ومن الصفر إلى الموجب في غيرها.** ومن افترض واحداً منهما
 * **رأى الشريطَ واقفاً في نصف المتصفّحات** — والعطبُ لا يظهر عنده.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { CategoryIcon } from "./CategoryIcon";
import { IconNext, IconPrev } from "./icons";

const m = getMessages(defaultLocale);

export interface RailItem {
  id: string;
  name: string;
  /** صورةُ القسم — **وأيقونتُه بديلاً حين لا صورة.** */
  imageUrl: string | null;
  icon: string;
  /** **عددُ المتاح الآن** — وقسمٌ فارغٌ يبهت ولا يُخفى. */
  count: number;
}

/** بكسلاتٌ في كلّ إطار — **ما يُحسّ مشياً ولا يُقرأ هرباً.** */
const STEP = 0.4;

export function SectionRail({
  items,
  activeID,
  onSelect,
  className = "",
}: {
  items: RailItem[];
  activeID?: string;
  onSelect: (id: string) => void;
  className?: string;
}) {
  const box = useRef<HTMLDivElement>(null);
  const [held, setHeld] = useState(false);
  const [still, setStill] = useState(false);

  /** **وإطفاءُ الحركة يُقرأ حيّاً** — من غيّره وهو يتصفّح لا ينتظر تحديثَ صفحة. */
  useEffect(() => {
    const mq = window.matchMedia?.("(prefers-reduced-motion: reduce)");
    if (!mq) return;
    const read = () => setStill(mq.matches);
    read();
    mq.addEventListener("change", read);
    return () => mq.removeEventListener("change", read);
  }, []);

  /**
   * **إشارةُ العدّ تُتحسَّس مرّةً.**
   *
   * نضع واحداً موجباً ونقرأ: **إن قُصّ إلى الصفر فالعدُّ سالب** (المعيار في
   * العربيّة)، وإلّا فموجب. **وهو فحصٌ لا افتراض.**
   */
  const sign = useRef(1);
  useEffect(() => {
    const el = box.current;
    if (!el) return;
    const keep = el.scrollLeft;
    el.scrollLeft = 1;
    sign.current = el.scrollLeft === 0 ? -1 : 1;
    el.scrollLeft = keep;
  }, [items.length]);

  /** يمشي ذهاباً ثمّ إياباً — **والاتّجاه ينقلب عند الطرف لا يقفز إلى البداية.** */
  const way = useRef(1);
  useEffect(() => {
    if (held || still) return;
    const el = box.current;
    if (!el) return;
    const id = setInterval(() => {
      const span = el.scrollWidth - el.clientWidth;
      // **ولا مشيَ حيث لا فائض** — أقسامٌ قليلةٌ تملأ العرضَ ولا تنزلق.
      if (span < 8) return;
      const now = Math.abs(el.scrollLeft) + way.current * STEP;
      if (now >= span) way.current = -1;
      else if (now <= 0) way.current = 1;
      el.scrollLeft = sign.current * Math.min(Math.max(now, 0), span);
    }, 16);
    return () => clearInterval(id);
  }, [held, still, items.length]);

  /** سهمٌ ينقل صفحةً — **وعلى الحاسب وحدَه**: الإصبعُ على الجوّال أطوعُ منه. */
  const nudge = useCallback((dir: -1 | 1) => {
    const el = box.current;
    if (!el) return;
    el.scrollBy({ left: sign.current * dir * el.clientWidth * 0.8, behavior: "smooth" });
  }, []);

  if (items.length === 0) return null;

  const arrow = (dir: -1 | 1) => (
    <button
      type="button"
      onClick={() => nudge(dir)}
      aria-label={dir === 1 ? m.common.next : m.common.back}
      className={`taparea absolute inset-block-0 my-auto hidden h-9 w-9 items-center justify-center rounded-badge border border-line bg-surface text-ink-muted transition-colors hover:text-ink sm:flex ${
        dir === 1 ? "start-0" : "end-0"
      }`}
    >
      {dir === 1 ? <IconNext size={16} /> : <IconPrev size={16} />}
    </button>
  );

  return (
    <div
      className={`relative ${className}`}
      onMouseEnter={() => setHeld(true)}
      onMouseLeave={() => setHeld(false)}
      onTouchStart={() => setHeld(true)}
      onTouchEnd={() => setHeld(false)}
      onFocusCapture={() => setHeld(true)}
      onBlurCapture={() => setHeld(false)}
    >
      <div
        ref={box}
        /* **وشريطُ التمرير يُخفى ولا يُمنع التمرير** — مقبضٌ رماديٌّ تحت صفٍّ
           من الصور يُقرأ عطباً، **والانزلاقُ بالإصبع لا يحتاج مقبضاً يُرى.** */
        className="flex gap-3 overflow-x-auto scroll-smooth px-1 py-1 [-ms-overflow-style:none] [scrollbar-width:none] sm:px-10 [&::-webkit-scrollbar]:hidden"
      >
        {items.map((it) => {
          const on = it.id === activeID;
          return (
            <button
              key={it.id}
              type="button"
              onClick={() => onSelect(it.id)}
              aria-pressed={on}
              /* **والعرضُ ثابتٌ لا يتقلّص** — بلا `shrink-0` يضغط `flex`
                 الدوائرَ حتّى تصير بيضاً، **فيختلف مقاسُها بعدد الأقسام.** */
              className="flex w-[76px] shrink-0 flex-col items-center gap-1.5 sm:w-[88px]"
            >
              <span
                /* **والحلقةُ تقول المختار** — لا لونُ النصّ وحدَه: صفٌّ من
                   تسعِ دوائرَ لا يُميَّز فيه اسمٌ أغمقُ من أخيه. */
                className={`relative flex aspect-square w-full items-center justify-center overflow-hidden rounded-full bg-page transition-all duration-[--duration-base] ease-[--ease-out] ${
                  on
                    ? "ring-2 ring-accent ring-offset-2 ring-offset-shell"
                    : "ring-1 ring-line hover:ring-primary/50"
                } ${it.count === 0 ? "opacity-50" : ""}`}
              >
                {it.imageUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={it.imageUrl}
                    alt=""
                    loading="lazy"
                    draggable={false}
                    /* **والدائرةُ تقصّ عمداً** — بخلاف اللافتة: صورةُ القسم
                       مشهدٌ يُقتطع منه، **واللافتةُ تصميمٌ يُقرأ كاملاً.** */
                    className="h-full w-full select-none object-cover"
                  />
                ) : (
                  <CategoryIcon name={it.icon} size={26} />
                )}
              </span>

              <span className={`w-full truncate text-center text-2xs sm:text-xs ${on ? "font-bold text-ink" : "text-ink-muted"}`}>
                {it.name}
              </span>
              <span className="text-2xs text-ink-muted/70">{fmtNum(it.count)}</span>
            </button>
          );
        })}
      </div>

      {arrow(1)}
      {arrow(-1)}
    </div>
  );
}
