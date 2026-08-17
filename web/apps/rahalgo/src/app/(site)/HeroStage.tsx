"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مشهدُ الافتتاحيّة — يُكتب حرفاً حرفاً ثمّ يُعاد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٧: «اجعلها تُكتب حرفاً حرفاً متلاحقةً إلى أن
 *  تنتهي العبارة، تنبض تومض لثلاث ثوانٍ وتُعاد الحركة من جديد… وأيضاً
 *  المربّعات يظهر مربّعٌ مربّع ثمّ يظهر السهم ثمّ تظهر عبارة… ثمّ ننتقل
 *  إلى المربّعات الأخرى».)
 *
 * # ولماذا مكوّنُ متصفّحٍ وحدَه
 *
 * **الترتيبُ زمنٌ لا شكل** — ولا يُكتب في ورقة أنماط: **ما يظهر بعد ماذا
 * وبكم.** والصفحةُ نفسُها تبقى تُقدَّم من الخادم، **وهذا الجزءُ وحدَه
 * يهبط إلى المتصفّح.**
 *
 * # والنصُّ كاملٌ في أوّل رسمة
 *
 * **الحركةُ تُخفيه ثمّ تُظهره حرفاً حرفاً** — **ولو بُني النصُّ من
 * جافاسكربت لَقرأه غوغلُ فارغاً**، وهي الصفحةُ الوحيدةُ التي يهمّنا أن
 * تُفهرَس. **فما يُرسم أوّلاً هو الجملةُ تامّةً**، ثمّ يتولّاها المشهد.
 *
 * # ومن أطفأ الحركةَ يرى كلَّ شيءٍ ساكناً
 *
 * **لا يُحرَم أحدٌ من المحتوى لأنّه لا يحتمل الحركة** — يُعرض تامّاً بلا
 * كتابةٍ ولا ظهورٍ متتابع.
 */

import { useEffect, useMemo, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  IconStore,
  IconSearch,
  IconCart,
  IconMoto,
  IconWallet,
  IconRoles,
  IconNext,
} from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const H = m.site.homePage;

type Icon = React.ComponentType<{ size?: number; className?: string }>;

/** **قطعةٌ من العنوان** — نصٌّ وصنفُ لونه إن كان له لون. */
interface Piece {
  t: string;
  c?: string;
}

/** **مربّعُ وعدٍ — لا زرّ.** (شكوى المالك ٢٠٢٦-٠٨-١٧.) */
function Chip({ Icon, label, on }: { Icon: Icon; label: string; on: boolean }) {
  return (
    <span
      className={`flex items-center gap-2.5 rounded-card bg-accent-tint px-4 py-3 text-base font-bold text-accent-dark transition-all duration-500 ${
        on ? "translate-y-0 opacity-100" : "translate-y-2 opacity-0"
      }`}
    >
      <Icon size={26} className="shrink-0" />
      {label}
    </span>
  );
}

const ROW1: [Icon, string][] = [
  [IconStore, H.heroChip1],
  [IconSearch, H.heroChip2],
  [IconCart, H.heroChip3],
];
const ROW2: [Icon, string][] = [
  [IconMoto, H.heroChip4],
  [IconWallet, H.heroChip5],
  [IconRoles, H.heroChip6],
];

/** **إيقاعُ المشهد بالملّي** — في مكانٍ واحدٍ يُضبط منه كلُّه. */
const MS = {
  letter: 55,
  letterLead: 32,
  blink: 3000,
  step: 260,
  hold: 2600,
};

export default function HeroStage({ name }: { name: string }) {
  /* **والعنوانُ قطعٌ ليُكتب حرفاً حرفاً وهو ملوّن.**

     **ولو كان نصّاً واحداً لَتعذّر تلوينُ كلمتين فيه** أثناء الكتابة —
     **والقطعُ يُقصّ عبرها بعدّادٍ واحدٍ فتخرج الحروفُ بترتيبها ولونها.** */
  const pieces: Piece[] = useMemo(() => {
    /* @platform-ok — يُحقن أدناه قطعةً ملوّنةً بنفسه. */
    const [head = "", rest = ""] = H.heroTitle.split("{city}");
    const [mid = "", tail = ""] = rest.split("{platform}");
    return [
      { t: head },
      { t: H.cityName, c: "text-gradient-city" },
      { t: mid },
      { t: name, c: "text-gradient-platform" },
      { t: tail },
    ];
  }, [name]);
  const titleLen = pieces.reduce((n, p) => n + p.t.length, 0);

  const [still, setStill] = useState(false);
  const [typed, setTyped] = useState(titleLen);
  const [blink, setBlink] = useState(false);
  /** **كم عنصراً ظهر من الصفّين والسهم** — ٠ لا شيء، ٧ الكلّ. */
  const [shown, setShown] = useState(7);
  const [l1, setL1] = useState(H.heroLine1);
  const [l2, setL2] = useState(H.heroLine2);
  const alive = useRef(true);

  useEffect(() => {
    const mq = window.matchMedia?.("(prefers-reduced-motion: reduce)");
    if (!mq) return;
    const read = () => setStill(mq.matches);
    read();
    mq.addEventListener("change", read);
    return () => mq.removeEventListener("change", read);
  }, []);

  useEffect(() => {
    alive.current = true;
    if (still) {
      // **والساكنُ يرى الجملةَ تامّةً** — لا نصفَ مشهد.
      setTyped(titleLen);
      setShown(7);
      setL1(H.heroLine1);
      setL2(H.heroLine2);
      return;
    }
    const sleep = (ms: number) => new Promise((r) => setTimeout(r, ms));
    const type = async (full: string, put: (s: string) => void, ms: number) => {
      for (let i = 0; i <= full.length; i++) {
        if (!alive.current) return;
        put(full.slice(0, i));
        await sleep(ms);
      }
    };
    (async () => {
      while (alive.current) {
        setTyped(0);
        setShown(0);
        setL1("");
        setL2("");
        setBlink(false);
        for (let i = 0; i <= titleLen; i++) {
          if (!alive.current) return;
          setTyped(i);
          await sleep(MS.letter);
        }
        // **ثمّ تومض ثلاثَ ثوانٍ** — (نصُّ الطلب).
        setBlink(true);
        await sleep(MS.blink);
        if (!alive.current) return;
        setBlink(false);
        // **ثمّ مربّعٌ مربّعٌ ثمّ السهمُ ثمّ العبارة.**
        for (const n of [1, 2, 3, 4]) {
          if (!alive.current) return;
          setShown(n);
          await sleep(MS.step);
        }
        await type(H.heroLine1, setL1, MS.letterLead);
        for (const n of [5, 6, 7]) {
          if (!alive.current) return;
          setShown(n);
          await sleep(MS.step);
        }
        await type(H.heroLine2, setL2, MS.letterLead);
        await sleep(MS.hold);
      }
    })();
    return () => {
      alive.current = false;
    };
  }, [still, titleLen]);

  // **والقصُّ عبر القطع** — حرفٌ واحدٌ يعبر حدَّ اللون فلا ينقطع.
  let left = typed;
  const parts = pieces.map((p, i) => {
    const take = Math.max(0, Math.min(p.t.length, left));
    left -= take;
    const text = p.t.slice(0, take);
    return p.c ? (
      <span key={i} className={p.c}>
        {text}
      </span>
    ) : (
      <span key={i}>{text}</span>
    );
  });

  return (
    <div className="flex flex-col items-start text-start">
      <h1 className={`heading-hero ${blink ? "title-blink" : ""}`}>
        {parts}
        {/* **ومؤشّرٌ ما دامت تُكتب** — يقول «لم تنتهِ بعد». */}
        {!still && typed < titleLen && <span className="caret">|</span>}
      </h1>

      <div className="hero-lead mt-7 flex max-w-prose flex-col gap-4 text-ink-muted">
        <div className="flex flex-wrap items-center gap-2.5">
          {ROW1.map(([Ico, label], i) => (
            <Chip key={label} Icon={Ico} label={label} on={shown > i} />
          ))}
          <IconNext
            size={26}
            className={`arrow-nudge shrink-0 text-accent-text transition-opacity duration-500 ${
              shown > 3 ? "opacity-100" : "opacity-0"
            }`}
          />
          <span className="text-base font-bold text-ink">{l1}</span>
        </div>

        <div className="flex flex-wrap items-center gap-2.5">
          {ROW2.map(([Ico, label], i) => (
            <Chip key={label} Icon={Ico} label={label} on={shown > i + 4} />
          ))}
        </div>

        <p>{l2}</p>
      </div>
    </div>
  );
}
