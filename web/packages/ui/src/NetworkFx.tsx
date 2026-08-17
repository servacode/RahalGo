"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحةُ الشبكة — مركزٌ يصل مواقعَه واحداً بعد واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-١٧: «Motion Graphic برمجيٌّ حقيقيٌّ… هادئ +
 *  هندسيّ + ذكيّ + فاخر + دقيق… النظامُ يبدأ من مركزٍ واحد، ثمّ يربط
 *  المواقعَ واحداً تلوَ الآخر، ثمّ تصبح الشبكةُ كلُّها فعّالةً ومتّصلة».)
 *
 * # ولماذا لا GSAP
 *
 * **قِيس المشروعُ فلا مكتبةَ حركةٍ فيه أصلاً** — لا GSAP ولا Framer.
 * **وإذنُه صريحٌ**: «لا تضف مكتبةً جديدةً إذا كانت هناك أداةٌ مناسبةٌ
 * موجودة… وإلّا نفّذها بطريقة SVG/CSS/GSAP بسيطةٍ دون إضافة Dependencies
 * غيرِ ضروريّة».
 *
 * **وكلُّ ما وصفه يقع في CSS بلا نقصان**: الرسمُ بـ`stroke-dashoffset`،
 * والتتابعُ بـ`animation-delay`، والنبضُ والإشاراتُ بـ`transform`،
 * والانسيابُ بمنحنياتِ تسارعٍ هي `power2.out` نفسُها.
 *
 * **وثلاثةُ فروقٍ في صالح CSS هنا**:
 *
 *   **١ · الدورةُ تُغلق بنفسها.** كلُّ عنصرٍ دورتُه ستُّ ثوانٍ بالضبط،
 *       **وأوّلُ إطارٍ فيها هو آخرُها** — فلا وصلةَ تُرى بين دورتين.
 *       **وخطُّ زمنٍ في جافاسكربت يتراكم فيه الانحرافُ** ما لم يُضبط.
 *
 *   **٢ · لا خيطَ يعمل ولا ذاكرةَ تُترك.** لا مؤقّتٌ ولا `requestAnimationFrame`
 *       **ولا خطُّ زمنٍ يُنظَّف عند التفكيك** — فلا تسرّبَ أصلاً.
 *
 *   **٣ · الحركةُ تُدار في خيط التركيب لا في جافاسكربت** — `transform`
 *       و`opacity` وحدَهما، **فتبقى ستّين إطاراً وإن انشغل الخيطُ الرئيسيّ.**
 *
 * # وما يحتاج جافاسكربت فعلاً — سطران
 *
 * **الإيقافُ خارج الشاشة** (`IntersectionObserver`): لوحةٌ تدور والمستخدمُ
 * في أسفل الصفحة **تحرق بطاريّةً بلا أن تُرى.**
 *
 * # والهندسة
 *
 * **مركزٌ في (٣٠٠، ٣٠٠) وثمانيةُ مواقع** — بزوايا ونصفِ أقطارٍ فيها
 * اختلافٌ محسوب: **دائرةٌ كاملةُ الانتظام تُقرأ رسماً آليّاً**، والاختلافُ
 * الطفيفُ يجعلها تُقرأ خريطةً.
 *
 * **والترتيبُ مع عقارب الساعة من الأعلى** — كما نصّت المواصفة، والرقمُ
 * في `--i` هو موضعُه في الدور.
 *
 * # والخطُّ يُدار لا يُحسب طرفاه
 *
 * **كلُّ وصلةٍ مجموعةٌ مُدارةٌ حول المركز** فيها خطٌّ أفقيٌّ وإشارةٌ تمشي
 * عليه. **ولو رُسم كلُّ خطٍّ بنقطتيه لَاحتاجت الإشارةُ مساراً منحنياً
 * تُحسب نقاطُه** — والإدارةُ تجعل المشيَ نحو الطرف انزلاقاً في محورٍ واحد.
 */

import { useEffect, useRef } from "react";

/** موقعٌ خارجيّ — زاويتُه بالدرجات ونصفُ قطره في إحداثيّات اللوحة. */
interface Node {
  a: number;
  r: number;
}

/**
 * **ثمانيةُ مواقعَ بترتيب عقارب الساعة من الأعلى.**
 *
 * **والزوايا ليست مضاعفاتِ خمسٍ وأربعين** — فرقُ درجتين أو ثلاثٍ في كلٍّ
 * منها، **ونصفُ القطر يتبدّل بين المئتين والمئتين والخمسين.**
 */
const NODES: Node[] = [
  { a: -88, r: 198 }, // أعلى
  { a: -43, r: 226 }, // أعلى اليمين
  { a: 4, r: 243 }, // يمين
  { a: 47, r: 209 }, // أسفل اليمين
  { a: 91, r: 231 }, // أسفل
  { a: 136, r: 217 }, // أسفل اليسار
  { a: 179, r: 238 }, // يسار
  { a: -134, r: 212 }, // أعلى اليسار
];

const C = 300;

export function NetworkFx({ className = "" }: { className?: string }) {
  const box = useRef<SVGSVGElement>(null);

  /* **وتقف حين تخرج من الشاشة.**

     **ولا تُهدم ثمّ تُبنى**: `animation-play-state` تُجمّد الإطارَ الحاليَّ
     وتُكمل منه، **وإعادةُ البناء تُرجع الدورةَ إلى أوّلها** فيراها العائدُ
     تبدأ من الصفر. */
  useEffect(() => {
    const el = box.current;
    if (!el || typeof IntersectionObserver === "undefined") return;
    const io = new IntersectionObserver(
      ([e]) => el.setAttribute("data-run", e?.isIntersecting ? "on" : "off"),
      { rootMargin: "120px" },
    );
    io.observe(el);
    return () => io.disconnect();
  }, []);

  return (
    <svg
      ref={box}
      data-run="on"
      viewBox="0 0 600 600"
      preserveAspectRatio="xMidYMid meet"
      role="presentation"
      aria-hidden="true"
      className={`netfx block h-auto w-full ${className}`}
    >
      {/* **وضوءٌ خافتٌ خلف المركز** — يعمّق اللوحةَ ولا يُرى بنفسه. */}
      <defs>
        <radialGradient id="netfx-core" cx="50%" cy="50%" r="50%">
          <stop offset="0%" className="netfx-halo-in" />
          <stop offset="100%" className="netfx-halo-out" />
        </radialGradient>
      </defs>
      <circle cx={C} cy={C} r={250} fill="url(#netfx-core)" className="netfx-glow" />

      {/* ══════════════════════════════════════════════════════════════
          **الوصلاتُ — خطٌّ يُرسم من المركز ثمّ إشارةٌ تمشي عليه**
          ══════════════════════════════════════════════════════════════

          **وطولُ الخطّ يُمرَّر إلى الأنماط** (`--len`): الرسمُ يحتاجه في
          `stroke-dasharray`، **والإشارةُ تحتاجه لتعرف أين تقف.** */}
      {NODES.map((n, i) => (
        <g key={`link-${i}`} transform={`rotate(${n.a} ${C} ${C})`}>
          <line
            x1={C}
            y1={C}
            x2={C + n.r}
            y2={C}
            className="netfx-line"
            style={
              {
                "--len": n.r,
                "--i": i,
              } as React.CSSProperties
            }
          />
          <circle
            cx={C}
            cy={C}
            r={4}
            className="netfx-signal"
            style={
              {
                "--len": `${n.r}px`,
                "--i": i,
              } as React.CSSProperties
            }
          />
        </g>
      ))}

      {/* **والمواقعُ ثابتةٌ لا تدور ولا تطفو** — تُضيء حين يصلها خطُّها. */}
      {NODES.map((n, i) => {
        const rad = (n.a * Math.PI) / 180;
        const x = C + n.r * Math.cos(rad);
        const y = C + n.r * Math.sin(rad);
        return (
          <g
            key={`node-${i}`}
            className="netfx-node"
            style={{ "--i": i, transformOrigin: `${x}px ${y}px` } as React.CSSProperties}
          >
            <circle cx={x} cy={y} r={13} className="netfx-node-halo" />
            <circle cx={x} cy={y} r={7.5} className="netfx-node-ring" />
            <circle cx={x} cy={y} r={2.6} className="netfx-node-dot" />
          </g>
        );
      })}

      {/* ══════════════════════════════════════════════════════════════
          **حلقاتُ النبض — ثلاثٌ في الافتتاح وواحدةٌ واسعةٌ عند الاكتمال**
          ══════════════════════════════════════════════════════════════

          **وتخرج من المركز وتخفت** — لا ذراعَ تمسح ولا دورة. */}
      {[0, 1, 2].map((i) => (
        <circle
          key={`ring-${i}`}
          cx={C}
          cy={C}
          r={78}
          className="netfx-ring"
          style={{ "--i": i } as React.CSSProperties}
        />
      ))}
      <circle cx={C} cy={C} r={78} className="netfx-ring netfx-ring-wide" />

      {/* **والمركزُ في مكانه دائماً** — يسطع ويهدأ ولا يتحرّك. */}
      <g className="netfx-core" style={{ transformOrigin: `${C}px ${C}px` }}>
        <circle cx={C} cy={C} r={34} className="netfx-core-halo" />
        <circle cx={C} cy={C} r={21} className="netfx-core-ring" />
        <circle cx={C} cy={C} r={7} className="netfx-core-dot" />
      </g>
    </svg>
  );
}
