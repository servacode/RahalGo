"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقة رحّال غو وكرت اليوم** (قرارُ المالك ٢٠٢٦-١١-٢٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * مرآةُ `LoyaltyCards.kt` في `mobile/ui` — للزبون (/shop و/offers) وللمتجر (/store و«اطلب توصيلة»).
 * بطاقةٌ بلون العلامة بدوائرَ في صفّين (١٠ ⇒ ٥×٢)، **والمختومةُ بشعار رحّال غو**، وتحتها **كرتُ
 * اليوم**. **والختمُ الجديدُ منذ آخر نظرةٍ يُختم أمامه** (العددُ المرئيُّ محفوظٌ بالمتصفّح بمفتاح
 * `seenKey`). **والمطفأةُ لا تُرسم.** والنصوصُ `loyalty.*` — هي نفسُها في التطبيقين.
 */

import { useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtMoney } from "@rahalgo/i18n";
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
  useEffect(() => {
    if (!path) {
      setCard(null);
      return;
    }
    let live = true;
    api<StampCard>(path)
      .then((c) => live && setCard(c))
      .catch(() => live && setCard(null));
    return () => {
      live = false;
    };
  }, [path]);
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

function Grid({ size, stamps, seen, daily }: { size: number; stamps: number; seen: number; daily: boolean }) {
  const perRow = size <= 10 ? Math.ceil(size / 2) : 5;
  return (
    <div className="loyalty-grid" style={{ gridTemplateColumns: `repeat(${perRow}, minmax(0, 1fr))` }}>
      {Array.from({ length: size }, (_, i) => {
        const stamped = i < stamps;
        const fresh = stamped && seen >= 0 && i >= seen;
        return (
          <span
            key={i}
            className={`loyalty-dot${daily ? " is-daily" : ""}${stamped ? " is-stamped" : ""}${fresh ? " is-fresh" : ""}`}
            style={fresh ? { animationDelay: `${250 + 140 * (i - seen)}ms` } : undefined}
            role="img"
            aria-label={stamped ? L.stamped : L.empty}
          >
            {stamped ? <img src="/pwa/mark-color.png" alt="" /> : <span className="num">{i + 1}</span>}
          </span>
        );
      })}
    </div>
  );
}

function StampBox({ card, seenKey }: { card: StampCard; seenKey: string }) {
  const seen = useSeen(`${seenKey}.stamp`, card.stamps);
  return (
    <section className={`loyalty-card${card.free_ready ? " is-full" : ""}`} aria-label={L.stampTitle}>
      <header className="flex items-center gap-rg-sm">
        <img src="/pwa/mark-color.png" alt="" className="loyalty-logo" />
        <div className="min-w-0 flex-1">
          <h2 className="text-body font-bold">{L.stampTitle}</h2>
          <p className="text-caption opacity-85">{L.stampSub}</p>
        </div>
        <span className="loyalty-count">{fill(L.count, { n: card.stamps, size: card.size })}</span>
      </header>
      <Grid size={card.size} stamps={card.stamps} seen={seen} daily={false} />
      {card.free_ready ? (
        <p className="loyalty-highlight is-accent">{L.stampFull}</p>
      ) : (
        <p className="text-body font-bold">{card.reward_pending ? L.stampPending : leftText(card.size - card.stamps, false)}</p>
      )}
      {card.free_cap > 0 && <p className="text-caption opacity-85">{fill(L.cap, { amount: fmtMoney(card.free_cap) })}</p>}
    </section>
  );
}

function DailyBox({ card, seenKey }: { card: DailyCard; seenKey: string }) {
  const seen = useSeen(`${seenKey}.daily`, card.stamps);
  const full = card.free_ready && card.open;
  return (
    <section className={`loyalty-daily${full ? " is-full" : ""}`} aria-label={L.dailyTitle}>
      <header className="flex items-center gap-rg-sm">
        <span className="loyalty-daily-dot" aria-hidden />
        <h2 className="min-w-0 flex-1 text-body font-bold text-ink">{L.dailyTitle}</h2>
        <span className="loyalty-daily-count">{fill(L.count, { n: card.stamps, size: card.size })}</span>
      </header>
      <Grid size={card.size} stamps={card.stamps} seen={seen} daily />
      {!card.open ? (
        <p className="text-body text-ink-muted">{L.dailyClosed}</p>
      ) : full ? (
        <p className="loyalty-highlight is-success">{L.dailyFull}</p>
      ) : (
        <p className="text-body font-bold text-ink">{card.reward_pending ? L.dailyPending : leftText(card.size - card.stamps, true)}</p>
      )}
      {card.open && <p className="text-caption text-ink-muted">{L.dailyRenews}</p>}
      {card.free_cap > 0 && <p className="text-caption text-ink-muted">{fill(L.cap, { amount: fmtMoney(card.free_cap) })}</p>}
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
