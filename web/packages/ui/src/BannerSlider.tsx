"use client";

/**
 * **سلايدرُ اللافتات** — لافتةٌ واحدةٌ في المرّة تتبدّل وحدَها.
 *
 * # لماذا لا شريطٌ يُسحب
 *
 * كانت اللافتاتُ شريطاً أفقياً يُسحب باليد. **ومن لا يسحب لا يرى إلّا الأولى**
 * — والثانيةُ والثالثةُ تُنشَران ولا يراهما أحد، **فيُقال «اللافتاتُ لا تنفع»
 * والحقيقةُ أنّها لم تُعرَض.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٥: «لا تنسَ بناء سلايدر بالموقع لعرض البانرات».)
 *
 * # وثلاثةُ تفاصيلَ لولاها لصار إزعاجاً
 *
 *  1. **يتوقّف تحت الإصبع**: من مدّ يدَه ليقرأ لا تُسحب اللافتةُ من تحته.
 *  2. **ولا يدور على واحدة**: لافتةٌ وحيدةٌ تُعرض ساكنةً بلا نقاطٍ ولا مؤقّت.
 *  3. **ويحترم من أطفأ الحركة** (`prefers-reduced-motion`) — لمن يدوخه
 *     المتحرّك، **وهو إعدادٌ في نظامه لا رأيٌ نُصدره نحن.**
 */

import { useCallback, useEffect, useRef, useState } from "react";

export interface SlideItem {
  id: string;
  title: string;
  imageUrl: string | null;
  href?: string;
}

/** @param LinkComp رابطُ الإطار — يختلف بين `next/link` وغيره. */
export function BannerSlider({
  items,
  Link,
  everyMs = 5000,
  className = "",
}: {
  items: SlideItem[];
  /** رابطُ الإطار — **نوعٌ فضفاضٌ عمداً**: `next/link` وغيرُه لا يتطابقان. */
  Link?: React.ComponentType<{ href: string; className?: string; children: React.ReactNode }>;
  everyMs?: number;
  className?: string;
}) {
  const [at, setAt] = useState(0);
  const [paused, setPaused] = useState(false);

  const many = items.length > 1;

  useEffect(() => {
    if (!many || paused) return;
    if (
      typeof window !== "undefined" &&
      window.matchMedia?.("(prefers-reduced-motion: reduce)").matches
    ) {
      return;
    }
    const id = setInterval(() => setAt((i) => (i + 1) % items.length), everyMs);
    return () => clearInterval(id);
  }, [many, paused, items.length, everyMs]);

  // **والفهرسُ يُقلَّم حين تنقص اللافتات** — عرضٌ يُنزَل والسلايدرُ واقفٌ
  // عليه يترك إطاراً فارغاً.
  useEffect(() => {
    if (at >= items.length) setAt(0);
  }, [at, items.length]);

  const touch = useRef(0);
  const onStart = useCallback((x: number) => {
    touch.current = x;
    setPaused(true);
  }, []);
  const onEnd = useCallback(
    (x: number) => {
      const dx = x - touch.current;
      // **وسحبةٌ قصيرةٌ ليست سحبة** — إصبعٌ يرتجف على شاشةٍ ليس أمراً.
      if (Math.abs(dx) > 40 && many) {
        setAt((i) => (i + (dx < 0 ? 1 : items.length - 1)) % items.length);
      }
      setPaused(false);
    },
    [many, items.length],
  );

  const cur = items[Math.min(at, items.length - 1)];
  // **ولا إطارَ بلا لافتة** — وحارسٌ بعد القراءة يُرضي المدقّق أيضاً.
  if (!cur) return null;

  const inner = (
    <div className="relative h-40 w-full overflow-hidden rounded-card bg-page sm:h-52">
      {cur.imageUrl ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={cur.imageUrl} alt={cur.title} className="h-full w-full object-cover" />
      ) : (
        <div className="h-full w-full bg-gradient-to-l from-primary/20 to-accent/20" />
      )}
      <span className="absolute bottom-0 start-0 end-0 bg-gradient-to-t from-ink/70 to-transparent p-3 text-sm font-bold text-white">
        {cur.title}
      </span>
    </div>
  );

  return (
    <div
      className={className}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onTouchStart={(e) => onStart(e.touches[0]?.clientX ?? 0)}
      onTouchEnd={(e) => onEnd(e.changedTouches[0]?.clientX ?? 0)}
    >
      {cur.href && Link ? <Link href={cur.href}>{inner}</Link> : inner}

      {/* **ونقاطٌ تُضغط لا تُرى فقط** — من رأى الثالثةَ وأراد العودةَ إلى
          الأولى لا ينتظر دورةً كاملة. */}
      {many && (
        <div className="mt-2 flex justify-center gap-1.5">
          {items.map((it, i) => (
            <button
              key={it.id}
              type="button"
              aria-label={it.title}
              onClick={() => setAt(i)}
              className={`h-1.5 rounded-badge transition-all ${
                i === at ? "w-5 bg-accent" : "w-1.5 bg-line"
              }`}
            />
          ))}
        </div>
      )}
    </div>
  );
}
