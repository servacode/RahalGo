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
 *
 * ═══════════════════════════════════════════════════════════════════════
 * # وخمسةُ نواقصَ سُدَّت (٢٠٢٦-٠٨-٠٦، بطلب المالك: «السلايدر الاحترافيّ»)
 * ═══════════════════════════════════════════════════════════════════════
 *
 * **١ · كان يقفز ولا ينتقل.** لافتةٌ تحلّ محلَّ أختها في إطارٍ واحد **بلا
 * انتقالٍ يُرى** — والعينُ تقرأ القفزةَ خطأً في الصفحة لا تبدُّلَ لافتة.
 * **وهذا وحدَه الفرقُ بين سلايدرَ ومصفوفةِ صور.**
 *
 * **٢ · وكان يرسم واحدةً فقط.** فصورةُ التالية تبدأ تحميلَها **لحظةَ ظهورها**
 * — فيقع إطارٌ رماديٌّ ثمّ تظهر. **والآن كلُّها مرسومةٌ فوق بعضها** والظاهرةُ
 * وحدَها معتمة: **المتصفّحُ حمّلها قبل دورها.**
 *
 * **٣ · ولا سهمَين على الحاسب.** السحبُ يعمل بالإصبع، **ومن يجلس أمام شاشةٍ
 * بفأرةٍ لا يسحب** — فلا يملك إلّا انتظارَ الدورة أو ضغطَ نقطةٍ بحجم ستّة
 * بكسلات.
 *
 * **٤ · ولا لوحةَ مفاتيح.** من ينتقل بالتاب يقف على النقاط **ولا يستطيع
 * التنقّلَ بالأسهم** — وهو أوّلُ ما يُجرَّب.
 *
 * **٥ · ولا يُعرَّف لقارئ الشاشة.** كتلةٌ تتبدّل وحدَها بلا `aria-live`
 * **تُقرأ على الأعمى مرّةً واحدةً ثمّ تكذب عليه** بقيّةَ الوقت.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconNext, IconPrev } from "./icons";

const m = getMessages(defaultLocale);

export interface SlideItem {
  id: string;
  title: string;
  imageUrl: string | null;
  href?: string;
}

/** @param Link رابطُ الإطار — يختلف بين `next/link` وغيره. */
export function BannerSlider({
  items,
  Link,
  everyMs = 5000,
  className = "",
}: {
  items: SlideItem[];
  Link?: React.ComponentType<{ href: string; className?: string; children: React.ReactNode }>;
  everyMs?: number;
  className?: string;
}) {
  const [at, setAt] = useState(0);
  const [paused, setPaused] = useState(false);
  const many = items.length > 1;

  /**
   * **وإطفاءُ الحركة يُقرأ حيّاً لا مرّةً عند الإقلاع.**
   *
   * كان يُفحص داخل المؤقّت **فيُقرأ عند أوّل رسمٍ وحدَه** — ومن غيّر الإعدادَ
   * في نظامه وهو يتصفّح **يبقى السلايدرُ يدور عنده حتّى يُحدِّث الصفحة.**
   */
  const [still, setStill] = useState(false);
  useEffect(() => {
    const mq = window.matchMedia?.("(prefers-reduced-motion: reduce)");
    if (!mq) return;
    const read = () => setStill(mq.matches);
    read();
    mq.addEventListener("change", read);
    return () => mq.removeEventListener("change", read);
  }, []);

  const go = useCallback(
    (step: number) => setAt((i) => (i + step + items.length) % items.length),
    [items.length],
  );

  useEffect(() => {
    if (!many || paused || still) return;
    const id = setInterval(() => go(1), everyMs);
    return () => clearInterval(id);
  }, [many, paused, still, everyMs, go]);

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
      if (Math.abs(dx) > 40 && many) go(dx < 0 ? 1 : -1);
      setPaused(false);
    },
    [many, go],
  );

  // **ولا إطارَ بلا لافتة** — وحارسٌ بعد القراءة يُرضي المدقّق أيضاً.
  if (items.length === 0) return null;
  const cur = Math.min(at, items.length - 1);

  /*
    كان `h-40 sm:h-52` — **رقمان ثابتان بالبكسل**، والإطارُ يتمدّد بعرض
    الشاشة والارتفاعُ لا يتحرّك. **فالصورةُ الواحدةُ تُقصّ قصّاً مختلفاً في
    كلّ شاشة**: على الجوّال إطارٌ نسبتُه ٢:١، وعلى اللابتوب شريطٌ نسبتُه
    ٦:١ **يبتلع أعلى اللافتة وأسفلَها** — فيضيع العنوانُ المرسومُ فيها
    ويبقى وسطُها وحدَه. (شهده المالك ٢٠٢٦-٠٨-٠٥.)

    وتضيق النسبةُ كلّما اتّسعت الشاشة — **لأنّ الملء ليس هدفاً**: لافتةٌ
    بنسبة ١٦:٩ على شاشةٍ عريضة تصير سبعمئة بكسلٍ من الطول **تملأ الشاشة
    وحدَها** ولا يُرى تحتها شيء.

        جوّال   ٣١١ بكسلاً عرضاً · ١٦:٩ ← ١٧٥ طولاً
        لوحيّ   ٥٧٦ ·············· ٥:٢ ← ٢٣٠
        لابتوب  ٩٦٠ ·············· ١٦:٥ ← ٣٠٠
        أعرض    السقفُ يمسك عند ٣٦٠ **فتزداد عرضاً لا طولاً**
  */
  const frame =
    "relative aspect-[16/9] max-h-[360px] w-full overflow-hidden rounded-card bg-page sm:aspect-[5/2] lg:aspect-[16/5]";

  const slide = (it: SlideItem, i: number) => {
    const on = i === cur;
    const body = (
      <>
        {it.imageUrl ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={it.imageUrl}
            alt={it.title}
            draggable={false}
            /* **والأولى تُحمَّل فوراً وما بعدها كسولاً**: الأولى هي ما يُرى في
               أوّل رسمٍ — **وتأجيلُها يترك فراغاً في أهمّ موضعٍ من الصفحة.** */
            loading={i === 0 ? "eager" : "lazy"}
            className="h-full w-full select-none object-cover"
          />
        ) : (
          <div className="h-full w-full bg-gradient-to-l from-primary/20 to-accent/20" />
        )}
        {/* **والعنوانُ على حجابٍ متدرّج** — لا يُقرأ على صورةٍ لا يُعرف لونُها. */}
        <span className="absolute bottom-0 start-0 end-0 bg-gradient-to-t from-ink/70 to-transparent p-3 text-sm font-bold text-on-solid sm:p-4 sm:text-base">
          {it.title}
        </span>
      </>
    );
    return (
      <div
        key={it.id}
        aria-hidden={!on}
        /* **والمخفيّةُ لا تُلتقط بالتاب**: `inert` تمنع تركيزَ ما تحتها،
           **فلا يقف المتنقّلُ بالكيبورد على رابطٍ لا يراه.** */
        {...(!on ? { inert: "" as unknown as boolean } : {})}
        className={`absolute inset-0 transition-opacity ease-[--ease-out] ${
          still ? "duration-0" : "duration-[--duration-slow]"
        } ${on ? "opacity-100" : "pointer-events-none opacity-0"}`}
      >
        {it.href && Link ? (
          <Link href={it.href} className="block h-full w-full">
            {body}
          </Link>
        ) : (
          body
        )}
      </div>
    );
  };

  /** سهمٌ على الحاسب — **ويُخفى على الجوّال حيث السحبُ أطوعُ من زرٍّ صغير.** */
  const arrow = (dir: -1 | 1) => (
    <button
      type="button"
      onClick={() => go(dir)}
      aria-label={dir === 1 ? m.common.next : m.common.back}
      className={`taparea absolute inset-block-0 my-auto hidden h-10 w-10 items-center justify-center rounded-badge bg-ink/40 text-on-solid backdrop-blur-sm transition-colors hover:bg-ink/60 sm:flex ${
        dir === 1 ? "start-2" : "end-2"
      }`}
    >
      {/* **والأيقونتان من المركز** (`IconNext`/`IconPrev`) — وهما تنقلبان مع
          اتّجاه الصفحة كسائر أيقونات المنصة، **فلا يُقلَبان باليد هنا.** */}
      {dir === 1 ? <IconNext size={18} /> : <IconPrev size={18} />}
    </button>
  );

  return (
    <div
      className={className}
      /* **ويُعرَّف كتلةً تتبدّل** — قارئُ الشاشة يقول «لافتة ٢ من ٣» بدل أن
         يقرأ نصّاً يتغيّر تحته بلا خبر. */
      role="region"
      aria-roledescription="carousel"
      aria-label={m.site.home.bannersLabel}
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
      onTouchStart={(e) => onStart(e.touches[0]?.clientX ?? 0)}
      onTouchEnd={(e) => onEnd(e.changedTouches[0]?.clientX ?? 0)}
      /* **والأسهمُ تنقل حين يقع التركيزُ داخله** — وهي أوّلُ ما يُجرَّب. */
      onKeyDown={(e) => {
        if (!many) return;
        if (e.key === "ArrowLeft") go(1);
        else if (e.key === "ArrowRight") go(-1);
      }}
    >
      <div className={frame} aria-live={paused ? "polite" : "off"}>
        {items.map(slide)}
        {many && arrow(1)}
        {many && arrow(-1)}
      </div>

      {/* **ونقاطٌ تُضغط لا تُرى فقط** — من رأى الثالثةَ وأراد العودةَ إلى
          الأولى لا ينتظر دورةً كاملة. */}
      {many && (
        <div className="mt-2 flex justify-center gap-1.5">
          {items.map((it, i) => (
            <button
              key={it.id}
              type="button"
              aria-label={it.title}
              aria-current={i === cur}
              onClick={() => setAt(i)}
              className={`taparea h-1.5 rounded-badge transition-all duration-[--duration-base] ease-[--ease-out] ${
                i === cur ? "w-5 bg-accent" : "w-1.5 bg-line hover:bg-ink-muted"
              }`}
            />
          ))}
        </div>
      )}
    </div>
  );
}
