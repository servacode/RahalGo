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

import { useEffect, useRef, useState } from "react";
import { CategoryIcon } from "./CategoryIcon";

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


  if (items.length === 0) return null;

  /* **ولا سهمين.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «ألغِ السهم اليمين واليسار
     تبع الأقسام ما تلزم».)

     **والشريطُ يمشي وحدَه ذهاباً وإياباً** — فيُري كلَّ الأقسام بلا أن يُطلب
     منه. **والسهمُ يخدم من يريد تجاوزَ الانتظار**، وهو ما لا يقع في تسعة
     أقسامٍ يمرّ عليها الشريطُ في ثوان.

     **وزرّان معلَّقان فوق الصور يزاحمانها** — والدائرتان الأوّلى والأخيرةُ
     تختفيان تحتهما. **والسحبُ بالإصبع والتمريرُ بالفأرة يعملان بلا زرّ.** */

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
        className="flex gap-3 overflow-x-auto scroll-smooth px-1 py-1 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
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
              /* **وكبرت الدائرةُ بقرار المالك** (٢٠٢٦-٠٨-٠٦: «خلّي الدوائر
                 أكبر وأوضح»): من ٧٦ إلى ١٠٤، ومن ٨٨ إلى ١٢٨ على المتّسع.

                 **وصورةُ طعامٍ في ٧٦ بكسلاً تُقرأ لطخةً ملوّنة** — لا يُميَّز
                 فيها الصنفُ إلّا بالاسم تحتها، **فتضيع فائدةُ الصورة أصلاً.** */
              className="flex w-[104px] shrink-0 flex-col items-center gap-2 sm:w-[128px]"
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

              {/* **ولا عددَ تحت الاسم.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «عدد
                  المنتجات بالأصناف لا داعي لكتابتها».)

                  **والرقمُ كان يزاحم الاسمَ ولا يُفيد**: من يتصفّح لا يختار
                  قسماً لأنّ فيه اثني عشر صنفاً بدل خمسة، **إنّما يختار ما
                  يشتهيه.** والعددُ خبرُ إدارةٍ لا خبرُ زبون.

                  **والفارغُ يبقى يُقال بالبهتان** — لا برقمِ صفرٍ مكتوب. */}
              <span className={`w-full truncate text-center text-xs sm:text-sm ${on ? "font-bold text-ink" : "text-ink-muted"}`}>
                {it.name}
              </span>
            </button>
          );
        })}
      </div>

    </div>
  );
}
