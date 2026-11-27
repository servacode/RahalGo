"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقة رحّال غو وكرت اليوم** (قرارُ المالك ٢٠٢٦-١١-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * مرآةُ `LoyaltyCards.kt` في `mobile/ui` — للزبون والمتجر، **في صفحة «بطاقاتي»** (‏/cards و‏/store/cards)
 * تُفتح من أيقونةٍ صغيرةٍ في الشريط العلويّ (`LoyaltyCardsButton`) — لا كتلةً كبيرةً في الرئيسيّة (قرارُ المالك ٢٠٢٦-١١-٢٧).
 * بطاقةٌ بلون العلامة بدوائرَ في صفّين (١٠ ⇒ ٥×٢)، **والمختومةُ بشعار رحّال غو**، وتحتها **كرتُ
 * اليوم**. **والختمُ الجديدُ منذ آخر نظرةٍ يُختم أمامه** (العددُ المرئيُّ محفوظٌ بالمتصفّح بمفتاح
 * `seenKey`). **والمطفأةُ لا تُرسم.** والنصوصُ `loyalty.*` — هي نفسُها في التطبيقين.
 */

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useLiveRefresh } from "@rahalgo/ui";
import { getMessages, defaultLocale, fmtDamascusTime, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { api } from "@/lib/api";

const L = getMessages(defaultLocale).loyalty;

export interface DailyCard {
  enabled: boolean;
  size: number;
  stamps: number;
  free_ready: boolean;
  free_cap: number;
  reward_pending: boolean;
  open: boolean;
  closes_at: string;
  renews_at: string;
}

export interface StampCard {
  enabled: boolean;
  size: number;
  stamps: number;
  free_ready: boolean;
  free_cap: number;
  reward_pending: boolean;
  recent: { order_id: string; number: number; at: string }[];
  daily: DailyCard;
}

const fill = (s: string, v: Record<string, string | number>) =>
  s.replace(/\{(\w+)\}/g, (_, k: string) => String(v[k] ?? ""));

/** **كم بقي** — بالعدد العربيّ: واحدة · اثنتان · ٣–١٠ جمع · ١١+ مفرد (كالتطبيق). */
function leftText(n: number, daily: boolean): string {
  if (n <= 1) return daily ? L.dailyLeft1 : L.stampLeft1;
  if (n === 2) return daily ? L.dailyLeft2 : L.stampLeft2;
  return fill(n <= 10 ? (daily ? L.dailyLeftFew : L.stampLeftFew) : daily ? L.dailyLeftMany : L.stampLeftMany, { n });
}

/** **سطرُ «مجاناً» ومصدرُه** — للأجرة حين تُعفى بجائزة بطاقة أو بالعرض. */
export function freeSourceText(source?: string): string {
  if (source === "stamp") return L.freeStamp;
  if (source === "daily") return L.freeDaily;
  return L.freeCampaign;
}

/** **يقرأ البطاقتين** — من `path` (`/api/v1/my/stamp-card` أو بطاقةُ المتجر)، وخطأٌ يعني لا بطاقة. */
export function useStampCard(path: string | null): StampCard | null {
  const [card, setCard] = useState<StampCard | null>(null);
  const [tick, setTick] = useState(0);
  // **والختمُ يصل حيّاً** (قرارُ المالك ٢٠٢٦-١١-٢٧): إشارةُ `loyalty` عند التسليم، وتحديثُ الطلب احتياطاً.
  const bump = useCallback(() => setTick((n) => n + 1), []);
  useLiveRefresh(["loyalty", "order"], bump);
  useEffect(() => {
    if (!path) {
      setCard(null);
      return;
    }
    let live = true;
    api<StampCard>(path)
      .then((c) => live && setCard(c))
      .catch(() => live && setCard((old) => (tick > 0 ? old : null)));
    return () => {
      live = false;
    };
  }, [path, tick]);
  return card;
}

/** **كم ختماً رآه آخرَ مرّة** — ويُحفظ ما يراه الآن. `-1` أوّلَ مرّة (فلا يُختم شيءٌ أمامه). */
function useSeen(key: string, stamps: number): number {
  const seen = useMemo(() => {
    try {
      const v = window.localStorage.getItem(`loyalty:${key}`);
      return v === null ? -1 : Number(v);
    } catch {
      return -1;
    }
  }, [key, stamps]);
  useEffect(() => {
    try {
      window.localStorage.setItem(`loyalty:${key}`, String(stamps));
    } catch {
      /* متصفّحٌ بلا تخزين — بلا نبضة، والبطاقةُ تُرسم */
    }
  }, [key, stamps]);
  return seen;
}

/** **«١١:٠٠ م»** بتوقيت دمشق من `closes_at`، أو `null` إن لم يصل وقتٌ صالح (مرآةُ `closesText` في التطبيق). */
function closesText(iso: string): string | null {
  const d = new Date(iso);
  if (!iso || Number.isNaN(d.getTime()) || d.getUTCFullYear() < 2000) return null;
  const m = /^(\d{1,2}):(\d{2})$/.exec(fmtDamascusTime(d));
  if (!m) return null;
  const h24 = Number(m[1]) % 24;
  const h12 = h24 % 12 === 0 ? 12 : h24 % 12;
  return fill(h24 < 12 ? L.timeAm : L.timePm, { t: `${h12}:${m[2]}` });
}

/** **كم بقي في التذييل** — «باقي ٣ توصيلات» بلا بقيّة الجملة. */
function footLeft(n: number): string {
  if (n <= 1) return L.footLeft1;
  if (n === 2) return L.footLeft2;
  return fill(n <= 10 ? L.footLeftFew : L.footLeftMany, { n });
}

/** **الخاناتُ** — `n` خانةً، وخانةُ `giftIndex` هديّةٌ 🎁، والمختومةُ بشعار رحّال غو. */
function Grid({
  n,
  stamps,
  seen,
  daily,
  perRow,
  giftIndex,
}: {
  n: number;
  stamps: number;
  seen: number;
  daily: boolean;
  perRow: number;
  giftIndex: number;
}) {
  return (
    <div className={`loyalty-grid${daily ? " is-daily" : ""}`} style={{ gridTemplateColumns: `repeat(${perRow}, minmax(0, 1fr))` }}>
      {Array.from({ length: n }, (_, i) => {
        const gift = i === giftIndex && i >= stamps; // العاشرةُ تُختم أيضاً
        const stamped = !gift && i < stamps;
        const fresh = stamped && seen >= 0 && i >= seen;
        return (
          <span
            key={i}
            className={`loyalty-dot${daily ? " is-daily" : ""}${stamped ? " is-stamped" : ""}${gift ? " is-gift" : ""}${fresh ? " is-fresh" : ""}`}
            style={fresh ? { animationDelay: `${250 + 140 * (i - seen)}ms` } : undefined}
            role="img"
            aria-label={stamped ? L.stamped : L.empty}
          >
            {gift ? <span aria-hidden>{L.gift}</span> : stamped ? <img src="/pwa/mark-color.png" alt="" /> : <span>{i + 1}</span>}
          </span>
        );
      })}
    </div>
  );
}

function StampBox({ card, seenKey }: { card: StampCard; seenKey: string }) {
  const seen = useSeen(`${seenKey}.stamp`, card.stamps);
  const frac = card.size > 0 ? Math.min(1, Math.max(0, card.stamps / card.size)) : 0;
  return (
    <section className={`loyalty-card${card.free_ready ? " is-full" : ""}`} aria-label={L.stampTitle}>
      <img src="/pwa/mark-color.png" alt="" className="loyalty-wm" aria-hidden />
      <header className="flex items-start justify-between gap-rg-sm">
        <div className="min-w-0">
          <h2 className="loyalty-title">{L.stampTitle}</h2>
          <p className="loyalty-sub">{L.stampSub}</p>
        </div>
        <span className="loyalty-count">{`${card.stamps} / ${card.size}`}</span>
      </header>
      <div className="mt-rg-md">
        <Grid
          n={card.size}
          stamps={card.stamps}
          seen={seen}
          daily={false}
          perRow={card.size <= 10 ? Math.ceil(card.size / 2) : 5}
          giftIndex={card.size - 1}
        />
      </div>
      <div className="loyalty-prog" aria-hidden>
        <i style={{ width: `${frac * 100}%` }} />
      </div>
      {card.free_ready ? (
        <p className="loyalty-highlight is-accent mt-rg-sm">{L.stampFull}</p>
      ) : card.reward_pending ? (
        <p className="text-body font-bold mt-rg-sm">{L.stampPending}</p>
      ) : (
        <div className="loyalty-foot">
          <span>{footLeft(card.size - card.stamps)}</span>
          <b>{L.footNextFree}</b>
        </div>
      )}
      {card.free_cap > 0 && <p className="text-caption opacity-85 mt-rg-xs">{fill(L.cap, { amount: fmtMoney(card.free_cap) })}</p>}
    </section>
  );
}

function DailyBox({ card, seenKey }: { card: DailyCard; seenKey: string }) {
  const seen = useSeen(`${seenKey}.daily`, card.stamps);
  const full = card.free_ready && card.open;
  const until = card.open ? closesText(card.closes_at) : null;
  return (
    <section className={`loyalty-daily${full ? " is-full" : ""}`} aria-label={L.dailyTitle}>
      <div className="loyalty-daily-top">
        <header className="flex items-center justify-between gap-rg-sm">
          <h2 className="loyalty-title min-w-0">{L.dailyTitle}</h2>
          {card.open && <span className="loyalty-timer">{until ? fill(L.dailyUntil, { time: until }) : L.dailyRenews}</span>}
        </header>
        <p className="loyalty-sub">{fill(L.dailySub, { size: card.size })}</p>
      </div>
      <div className="loyalty-tear" aria-hidden>
        <i />
        <i />
      </div>
      <div className="loyalty-daily-body">
        <Grid n={card.size + 1} stamps={card.stamps} seen={seen} daily perRow={card.size + 1} giftIndex={card.size} />
        {!card.open ? (
          <p className="text-caption font-bold">{L.dailyClosed}</p>
        ) : full ? (
          <p className="loyalty-highlight is-success">{L.dailyFull}</p>
        ) : (
          <p className="text-caption font-bold">{card.reward_pending ? L.dailyPending : leftText(card.size - card.stamps, true)}</p>
        )}
        {card.free_cap > 0 && <p className="text-caption opacity-75">{fill(L.cap, { amount: fmtMoney(card.free_cap) })}</p>}
      </div>
    </section>
  );
}

/** **البطاقتان** — ولا شيءَ إن كانتا مطفأتين أو لم تُقرآ. */
export function LoyaltyCards({ card, seenKey, className = "" }: { card: StampCard | null; seenKey: string; className?: string }) {
  if (!card || (!card.enabled && !card.daily?.enabled)) return null;
  return (
    <div className={`flex flex-col gap-rg-sm ${className}`} data-testid="loyalty-cards">
      {card.enabled && card.size > 0 && <StampBox card={card} seenKey={seenKey} />}
      {card.daily?.enabled && card.daily.size > 0 && <DailyBox card={card.daily} seenKey={seenKey} />}
    </div>
  );
}

/** **شارةُ تقدّم الختم** («٧/١٠» بأرقام غربيّة كالرصيد) — فارغةٌ إن لم تُفعَّل بطاقةٌ بعدد. */
export function cardsBadge(card: StampCard): string {
  if (card.enabled && card.size > 0) return `${fmtNum(card.stamps)}/${fmtNum(card.size)}`;
  if (card.daily?.enabled && card.daily.size > 0) return `${fmtNum(card.daily.stamps)}/${fmtNum(card.daily.size)}`;
  return "";
}

/**
 * **أيقونةُ «بطاقاتي» في الشريط العلويّ** (قرارُ المالك ٢٠٢٦-١١-٢٧) — تذكرةٌ صغيرةٌ بشارة التقدّم، تفتح `href`.
 * **لا تُرسم** إلّا إن فُعّلت بطاقةٌ واحدةٌ على الأقلّ، ولا للضيف (`path = null`). وارتفاعُ الشريط كما كان
 * (الشارةُ مطلقةٌ فوق الزرّ). و`className` يختاره الشريطُ: `shop-iconbtn` للزبون و`icon-btn` للمتجر.
 */
export function LoyaltyCardsButton({
  path,
  href,
  className = "shop-iconbtn",
  size = 20,
}: {
  path: string | null;
  href: string;
  className?: string;
  size?: number;
}) {
  const card = useStampCard(path);
  if (!card || (!card.enabled && !card.daily?.enabled)) return null;
  // **دائرةُ «اربح مجاناً» النابضة** (اختيارُ المالك ٢٠٢٦-١١-٢٧، النموذج ١) — بدل أيقونة التذكرة.
  void className;
  void size;
  return (
    <Link href={href} className="loyalty-win" aria-label={L.cardsTitle} title={L.cardsTitle} data-testid="loyalty-cards-button">
      <span>{L.winFree}</span>
    </Link>
  );
}
