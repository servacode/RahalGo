"use client";

/**
 * **شريطُ الأقسام** — جولةٌ تعرض المنصةَ على نفسها.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «عندما تتحرّك الدوائر يجب أن يكون هناك انتقال،
 * وليس فقط حركة للأقسام. يعني القسم الذي يكون عليه الاختيار يجب أن يُظهر
 * المنتجات الخاصّة به — وهكذا تُعرض الأقسام والمنتجات بشكلٍ تلقائيّ، مع
 * هوفر جميلٍ للقسم الذي يُظهر المنتجات ليكون واضحاً».)
 *
 * # الحركةُ تحمل الاختيار — ولا تنزلق وحدَها
 *
 * **كان الشريطُ ينزلق بكسلاتٍ في الثانية** والاختيارُ ثابتٌ عند أوّل قسم.
 * **فتمشي الدوائرُ ولا يتغيّر شيءٌ تحتها** — حركةٌ بلا معنًى، تُقرأ زخرفةً
 * ويطفئها الناظرُ بعينه بعد ثوان.
 *
 * **والآن ينتقل الاختيارُ قسماً قسماً**: يُختار «مشاوي» فتظهر مشاويه، ثمّ
 * «حلويات» فتظهر حلوياته. **فالشريطُ يعرض المنصةَ على من لم يضغط شيئاً** —
 * وهو أنفعُ ما يفعله سوقٌ في أوّل عشر ثوانٍ من الزيارة.
 *
 * # والشريطُ يتبع المختار لا العكس
 *
 * **الدائرةُ المختارةُ تُساق إلى وسط الشاشة** — فلا يقع الاختيارُ على دائرةٍ
 * خارج النظر. **وهذا هو «الانتقال» المطلوب**: المسافةُ تُقطع بانزلاقٍ ناعمٍ
 * لا بقفزة.
 *
 * **وذهبت النسخةُ الثانيةُ من القائمة**: كانت شرطَ الدورة اللانهائيّة حين كان
 * الانزلاقُ بالبكسل. **والجولةُ الآن بالاختيار** — تصل آخرَ قسمٍ فتعود إلى
 * الأوّل، **ولا حاجةَ لرسم القائمة مرّتين.**
 *
 * # ويقف تحت اليد
 *
 * **من مدّ يدَه ليقرأ لا يُسحب منه ما يقرؤه.** فيقف عند التحويم واللمس
 * والتركيز — **ولا يعود حتّى تُرفع اليد.**
 *
 * # ومن أطفأ الحركة لا تُبدَّل الشاشةُ تحته
 *
 * **وتبديلُ المحتوى حركةٌ وإن لم ينزلق شيء** — فمن اختار السكونَ في نظامه
 * يُعطى شريطاً ساكناً يختار منه بيده.
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

/**
 * **مهلةُ القسم قبل أن ينتقل إلى التالي.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «يجب أن يكون الانتقال سريعاً للملاحظة».)
 *
 * **كانت سبعاً فبدت ساكنة**: من يفتح الصفحةَ وينظر خمسَ ثوانٍ **لا يرى
 * انتقالاً واحداً**، فيظنّ الشريطَ صورةً ثابتة.
 *
 * **وأربعٌ ونصف تُرى ولا تُلاحق**: انتقالان في أوّل عشر ثوانٍ — يكفيان ليفهم
 * الزائرُ أنّ الشريطَ يعرض عليه، **ويكفيان لتمرّ العينُ على صفٍّ من المنتجات.**
 *
 * **والسرعةُ وحدَها تنقلب ضرراً** لولا الشرطان تحتها: **تقف عند أوّل ضغطةٍ
 * يدويّة، وتقف حين يخرج الشريطُ من النظر.**
 */
const EVERY_MS = 4500;

export function SectionRail({
  items,
  activeID,
  onSelect,
  /**
   * **أتدور الجولةُ أصلاً؟** — (`shop.rail_auto` في إعدادات اللوحة.)
   *
   * **جولةٌ تعرض المنصةَ نافعةٌ لسوقٍ فيه ستّةٌ وعشرون قسماً**، وقد تصير
   * إزعاجاً في سوقٍ فيه أربعة. **ومن يملك المنصةَ يقرّر، لا من كتبها.**
   */
  auto = true,
  everyMs = EVERY_MS,
  className = "",
}: {
  items: RailItem[];
  activeID?: string;
  onSelect: (id: string) => void;
  auto?: boolean;
  everyMs?: number;
  className?: string;
}) {
  const box = useRef<HTMLDivElement>(null);
  const [held, setHeld] = useState(false);
  const [still, setStill] = useState(false);

  /**
   * **ومن اختار بيده انتهت الجولةُ عنده.**
   *
   * **الجولةُ تخدم من لم يقرّر بعد** — تعرض عليه ما لا يعرف أنّه موجود. **ومن
   * ضغط قسماً فقد قرّر**، فتبديلُ الشاشة تحته بعد ثوانٍ **يسرق منه ما اختاره.**
   *
   * **ولا تعود**: عودةٌ مفاجئةٌ بعد صمتٍ أسوأُ من ألّا تقف — لأنّها تقع حين
   * لا يتوقّعها.
   */
  const [taken, setTaken] = useState(false);

  /**
   * **وتقف حين يخرج الشريطُ من النظر.**
   *
   * من نزل يقرأ المنتجات **لا يرى الشريطَ ولا يعلم أنّه ينتقل** — فتتبدّل
   * الشبكةُ تحت عينيه وهو يقرأ سعراً. **والحركةُ التي لا تُرى لا تُفهم، إنّما
   * تُقرأ عطباً.**
   *
   * **وهي كذلك توفيرٌ**: جولةٌ تدور في شريطٍ خارج الشاشة تُحمّل وتُعيد الرسمَ
   * بلا أن يراها أحد.
   */
  const [seen, setSeen] = useState(true);
  useEffect(() => {
    const el = box.current;
    if (!el) return;
    const io = new IntersectionObserver((e) => setSeen(e[0]?.isIntersecting ?? true), {
      threshold: 0.3,
    });
    io.observe(el);
    return () => io.disconnect();
  }, []);

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
   * **الجولةُ: قسمٌ ثمّ الذي يليه، ثمّ تعود من الأوّل.**
   *
   * **والمؤقّتُ يُعاد من الصفر عند كلّ تبديل** — لأنّ `activeID` في تبعيّاته.
   * فمن ضغط قسماً بيده **يُعطى مهلتَه كاملةً قبل أن ينتقل عنه**، ولا يُخطف منه
   * ما اختاره بعد ثانية.
   */
  useEffect(() => {
    if (!auto || held || still || taken || !seen || items.length < 2) return;
    const t = setInterval(() => {
      const at = items.findIndex((x) => x.id === activeID);
      // **ومختارٌ لا وجودَ له يُعيد الجولةَ إلى أوّلها** — يقع حين يُحذف قسمٌ
      // من اللوحة والصفحةُ مفتوحة: `findIndex` تردّ ‎−١، **فيقع التالي على
      // الصفر** وهو ما نريد. والحارسُ هنا للحدّ الأعلى لا للسالب.
      const next = items[(at + 1) % items.length] ?? items[0];
      if (next) onSelect(next.id);
    }, everyMs);
    return () => clearInterval(t);
  }, [auto, held, still, taken, seen, items, activeID, onSelect, everyMs]);

  /**
   * **والشريطُ يسوق المختارَ إلى وسط النظر.**
   *
   * `block: "nearest"` **شرطٌ لا زينة**: بدونها يجرّ `scrollIntoView` الصفحةَ
   * كلَّها عموديّاً إلى الشريط — **فيقفز المتصفّحُ إلى أعلى الصفحة كلَّ سبع
   * ثوانٍ** بينما القارئُ في المنتجات تحته.
   */
  useEffect(() => {
    if (!activeID) return;
    const el = box.current?.querySelector<HTMLElement>(`[data-rail="${activeID}"]`);
    el?.scrollIntoView({
      behavior: still ? "auto" : "smooth",
      inline: "center",
      block: "nearest",
    });
  }, [activeID, still]);

  if (items.length === 0) return null;

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
           من الصور يُقرأ عطباً، **والانزلاقُ بالإصبع لا يحتاج مقبضاً يُرى.**

           **وحشوةٌ خفيفةٌ في الجانبين**: حلقةُ المختار تخرج عن حدّ الدائرة
           بأربعة بكسلاتٍ **فتُقصّ عند حافّة الصندوق الذي يُمرَّر.** وفي
           الجانبين لا في جهةٍ بعينها — **فالصفحةُ تنقلب مع اللغة.** */
        className="flex gap-3 overflow-x-auto px-2 py-3 [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden"
      >
        {items.map((it) => {
          const on = it.id === activeID;
          return (
            <button
              key={it.id}
              data-rail={it.id}
              type="button"
              onClick={() => {
                setTaken(true);
                onSelect(it.id);
              }}
              aria-pressed={on}
              /* **والعرضُ ثابتٌ لا يتقلّص** — بلا `shrink-0` يضغط `flex`
                 الدوائرَ حتّى تصير بيضاً، **فيختلف مقاسُها بعدد الأقسام.** */
              className="group flex w-[104px] shrink-0 flex-col items-center gap-2 sm:w-[128px]"
            >
              <span
                /* ══════════════════════════════════════════════════════════
                   **المختارُ يُرى من طرف الشاشة**
                   ══════════════════════════════════════════════════════════

                   (طلبُ المالك: «هوفر جميلٌ للقسم الذي يُظهر المنتجات ليكون
                   واضحاً».)

                   **وثلاثُ إشاراتٍ لا واحدة**: حلقةٌ جمريّةٌ بإزاحة، وكِبَرٌ
                   ستّةٌ بالمئة، وارتفاعٌ بظلّ. **وإشارةٌ واحدةٌ في صفٍّ من
                   ستٍّ وعشرين دائرةً تُخطَأ** — والعينُ تمسح ولا تدقّق.

                   **والكِبَرُ هو ما يُقرأ من بعيد**: اللونُ يحتاج تركيزاً،
                   **والحجمُ يُلتقط بطرف العين.**

                   **والحلقةُ لها إزاحة** — بلاها تلتصق بحافّة الصورة فتُقرأ
                   جزءاً منها لا علامةً عليها. */
                className={`relative flex aspect-square w-full items-center justify-center overflow-hidden rounded-full bg-field transition-all duration-[--duration-base] ease-[--ease-out] ${
                  on
                    ? "scale-[1.06] ring-2 ring-accent ring-offset-2 ring-offset-shell elev-3"
                    : "ring-1 ring-line group-hover:scale-[1.03] group-hover:ring-primary-edge"
                } ${it.count === 0 ? "opacity-50" : ""}`}
              >
                {it.imageUrl ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img
                    src={it.imageUrl}
                    alt=""
                    loading="lazy"
                    draggable={false}
                    /* **والدائرةُ تقصّ عمداً** — صورةُ القسم مشهدٌ يُقتطع منه،
                       **بخلاف اللافتة: تصميمٌ يُقرأ كاملاً.** */
                    className="h-full w-full select-none object-cover"
                  />
                ) : (
                  <CategoryIcon name={it.icon} size={26} />
                )}
              </span>

              {/* **ولا عددَ تحت الاسم** — من يتصفّح لا يختار قسماً لأنّ فيه
                  اثني عشر صنفاً بدل خمسة، **إنّما يختار ما يشتهيه.**

                  **والاسمُ يشتدّ مع المختار** — رابعةُ الإشارات، ولأنّ الاسمَ
                  هو ما يُقرأ حين تُشبه الصورُ بعضَها. */}
              <span
                className={`w-full truncate text-center text-xs transition-colors duration-[--duration-base] sm:text-sm ${
                  on ? "font-bold text-accent-text" : "text-ink-muted group-hover:text-ink"
                }`}
              >
                {it.name}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
