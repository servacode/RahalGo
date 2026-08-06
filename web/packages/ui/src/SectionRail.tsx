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
  /**
   * **والقائمةُ تُرسم مرّتين حين يفيض الشريط.**
   *
   * وهي شرطُ الدورة لا زينة: **ما يخرج من جهةٍ يجب أن يكون قد دخل من الأخرى.**
   *
   * **ولا تُنسخ إن ملأت العرضَ ولم تفض**: أربعةُ أقسامٍ على شاشةٍ عريضةٍ
   * **تُرسم ثمانيةً بلا حركةٍ تبرّر التكرار** — فيُقرأ خطأً في البيانات.
   */
  const [twice, setTwice] = useState(false);
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

  useEffect(() => {
    const el = box.current;
    if (!el) return;
    const measure = () => {
      // **عرضُ نسخةٍ واحدة** — يُقسَم على عدد النسخ المرسومة الآن.
      const one = el.scrollWidth / (twice ? 2 : 1);
      setTwice(one > el.clientWidth + 8);
    };
    measure();
    const ro = new ResizeObserver(measure);
    ro.observe(el);
    return () => ro.disconnect();
  }, [items.length, twice]);

  /**
   * **دورةٌ لا تنتهي — يخرج من جهةٍ فيعود من الأخرى.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «فتستمرّ بالتحرّك إلى اليمين لتظهر مرّةً أخرى
   * من اليسار».)
   *
   * # وكنتُ بنيتُها ذهاباً وإياباً وخالفتُ
   *
   * حجّتي كانت: **الدورةُ تحتاج نسخَ القائمة، فيُرسم القسمُ مرّتين ومن ضغط
   * النسخةَ الثانيةَ يظنّ أنّه رأى قسمين متشابهين.**
   *
   * **والحجّةُ سقطت بالتنفيذ**: النسخةُ الثانيةُ **تعمل كالأولى تماماً** —
   * تُضغط فتفتح القسمَ نفسَه وتأخذ الحلقةَ نفسَها إن كان مختاراً. **فلا فرقَ
   * يُدركه الناظر.**
   *
   * **وذهاباً وإياباً يُقرأ ترنّحاً** حين يطول الشريط: يمشي إلى طرفٍ ثمّ يرجع
   * على عقبيه — **والعينُ تسأل لماذا رجع.**
   *
   * # والقفزةُ لا تُرى لأنّ ما تحتها نسخةٌ طبقُ الأصل
   *
   * حين يجتاز عرضَ النسخة الأولى **يُطرح ذلك العرضُ من موضعه** — فيقف على
   * البكسل نفسِه من صورةٍ مطابقة. **ولا `scroll-smooth` هنا**: انتقالٌ ناعمٌ
   * للقفزة يجعلها تُرى شريطاً يرجع.
   */
  useEffect(() => {
    if (held || still) return;
    const el = box.current;
    if (!el) return;
    const id = setInterval(() => {
      // **طولُ الدورة = عرضُ نسخةٍ واحدة.**
      const lap = el.scrollWidth / (twice ? 2 : 1);
      // **ولا مشيَ حيث لا فائض** — أقسامٌ قليلةٌ تملأ العرضَ ولا تنزلق.
      if (lap - el.clientWidth < 8) return;
      let now = Math.abs(el.scrollLeft) + STEP;
      if (now >= lap) now -= lap;
      el.scrollLeft = sign.current * now;
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
        /* **وحشوةٌ خفيفةٌ حول الصفّ.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «خلّي بادينك خفيف
           من اليسار كي لا تقصّ الدائرة».)

           **والحلقةُ حول المختار تخرج عن حدّ الدائرة بأربعة بكسلات** (حلقةٌ
           باثنين وإزاحةٌ باثنين)، **فتُقصّ عند حافّة الصندوق الذي يُمرَّر.**

           **والحشوةُ في الجانبين لا في اليسار وحدَه** — فالصفحةُ تنقلب مع
           اللغة، **ومن حشا جهةً بعينها كتب عربيّةً في الشيفرة.** */
        className="flex gap-3 overflow-x-auto px-2 py-2 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        {(twice ? [0, 1] : [0]).flatMap((copy) =>
          items.map((it) => {
            const on = it.id === activeID;
          return (
            <button
              key={`${copy}-${it.id}`}
              /* **والنسخةُ الثانيةُ تعمل كالأولى** — تُضغط فتفتح القسمَ نفسَه
                 وتأخذ الحلقةَ نفسَها إن كان مختاراً. **فلا فرقَ يُدركه
                 الناظر**، وهو ما أسقط حجّتي على الدورة. */
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
          }),
        )}
      </div>

    </div>
  );
}
