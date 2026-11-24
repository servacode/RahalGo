"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **جوائز رحّال غو** (`/shop/prizes`) — مرآةُ شاشة التطبيق
 * ══════════════════════════════════════════════════════════════════════
 *
 * كلُّ ٥ دعواتٍ في الشهر = فرصةٌ بسحبٍ تلقائيّ، وللرابحين «رصيدُ جوائز» لا يُصرف إلّا على أصناف الجوائز.
 * **والخادمُ يقرّر كلَّ شيء** (الفرصُ والسحبُ والرصيدُ والتوصيلُ المجانيّ) — هنا عرضٌ، والطلبُ بالسلّة
 * المعتادة من متجر `store_id` (نافذةُ الصنف نفسُها فتعمل بوّابةُ العنوان و«سلّةٌ من متجرٍ آخر»).
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDate, fmtMoney, fmtNum, withPlatform } from "@rahalgo/i18n";
import { Button, IconCheck, IconCopy, IconShare, LoadingState, Money, ReloadState, usePlatform } from "@rahalgo/ui";
import { ItemSheet } from "@/components/customer/items";
import { useReferral } from "@/components/customer/InviteNudge";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { OUTLINE_CARD } from "@/components/customer/kit";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";
import { ShopPic } from "@/components/customer/ShopPic";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { difference, launchLabel, prizesLive, progressInEntry, splitByBalance } from "@/lib/customer/prizes";
import { fill } from "@/lib/customer/texts";
import type { Item, PrizeItem, PrizeItems, PrizeWinner, PrizeWinners, PrizesMe } from "@/lib/customer/types";
import { usePrizesMe } from "@/lib/customer/usePrizes";

const m = getMessages(defaultLocale);
const P = m.shop.prizes;
const V = m.customer.invite;

/** **صنفُ جوائز كصنفٍ عاديّ** — ليفتح نافذةَ الصنف والسلّةَ المعتادتين (التاجرُ = متجرُ الجوائز). */
function asItem(p: PrizeItem, storeId: string): Item {
  return {
    id: p.id,
    name: p.name,
    description: p.description ?? "",
    price: p.price,
    merchant_id: storeId,
    image_url: p.image_url ?? null,
    image_thumb_url: p.thumb_url ?? null,
    available: p.available !== false,
    source_closed: false,
    source_opens_at: null,
    section_id: "",
    section_name: "",
    price_before: null,
    discount_percent: null,
    has_options: false,
  };
}

/** **من اليمين**: الرابع، الثاني، الأوّل، الثالث، الخامس — والثاني جنبَ الأوّل حيث تبدأ العين. */
const PODIUM_ORDER = [4, 2, 1, 3, 5];

/**
 * **منصّةُ التتويج** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤: «٥ مربعات، الأول مرتفع بوسطهن… وفيه كاس») —
 * مرآةُ `PrizesPodium.kt`.
 */
function Podium({ amounts }: { amounts: number[] }) {
  if (amounts.length === 0) return null;
  const ranks = PODIUM_ORDER.filter((r) => r <= amounts.length);
  return (
    <section className={`${OUTLINE_CARD} flex flex-col gap-3`}>
      <h2 className="text-section-title text-center">{P.ladderTitle}</h2>
      <div className="flex items-end gap-1">
        {ranks.map((r) => (
          <div key={r} className={`flex min-w-0 flex-col items-center gap-1 ${r === 1 ? "flex-[1.25]" : "flex-1"}`}>
            {r === 1 && <img src="/prizes/trophy.svg" alt="" width={64} height={64} />}
            <Money value={amounts[r - 1] ?? 0} className={`font-bold ${r === 1 ? "text-section-title" : "text-sm"}`} />
            <div className={`prize-step prize-step-${r} flex w-full items-center justify-center text-center`}>
              <span className={`font-black ${r === 1 ? "text-3xl" : "text-xl"}`}>{fmtNum(r)}</span>
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}

/** **شروطُ المسابقة** — بنودٌ مرقّمةٌ بدوائر، وأرقامُها من المحرّك (مرآةُ `PrizeRules`). */
function Rules({ per, winners }: { per: number; winners: number }) {
  const rules = [
    /* **أربعةُ سطورٍ قصار** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤، مرآةُ `PrizeRules`). */
    fill(P.ruleInvite, { per: fmtNum(per) }),
    fill(P.ruleDraw, { n: fmtNum(winners) }),
    P.ruleUse,
    P.ruleExpire,
  ];
  return (
    <section className={`${OUTLINE_CARD} flex flex-col gap-2`}>
      <h2 className="text-section-title">{P.rulesTitle}</h2>
      <ol className="flex flex-col gap-2">
        {rules.map((t, i) => (
          <li key={i} className="flex items-start gap-2 text-sm">
            <span className="grid size-6 shrink-0 place-items-center rounded-full bg-accent text-xs font-bold text-on-bright">
              {fmtNum(i + 1)}
            </span>
            <span className="min-w-0 flex-1">{t}</span>
          </li>
        ))}
      </ol>
    </section>
  );
}

/** **كم يوم باقي** لتاريخٍ — صفرٌ اليومَ وما فات، و`null` إن لم يُقرأ. */
function daysUntil(at?: string | null): number | null {
  if (!at) return null;
  const t = Date.parse(at.length === 10 ? `${at}T00:00:00+03:00` : at);
  if (!Number.isFinite(t)) return null;
  return Math.max(0, Math.ceil((t - Date.now()) / 86_400_000));
}

/**
 * **رأسُ صفحة الجوائز** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤، مرآةُ `PrizeHero`) — كرتٌ واحد: متى وكم يوم باقي،
 * و«ضمن المسابقة» أو لا، وتقدّمُه. بدل العنوان والأيقونة و«انتظرونا» والمربّعات.
 */
function Hero({
  label,
  date,
  days,
  inside,
  title,
  body,
  progress,
  progressLine,
}: {
  label: string;
  date: string;
  days: number | null;
  inside: boolean;
  title: string;
  body: string;
  progress?: number;
  progressLine?: string;
}) {
  return (
    <section className="prize-hero flex flex-col gap-4 rounded-rg-lg p-rg-lg" data-testid="prize-entry-status">
      <div className="flex items-center gap-3">
        <div className="flex min-w-0 flex-1 flex-col gap-1">
          <span className="text-sm opacity-80">{label}</span>
          <span className="text-3xl font-black">{date}</span>
          {days != null && (
            <span className="prize-hero-chip w-fit rounded-full px-3 py-0.5 text-xs font-bold">
              {days === 0 ? P.heroToday : fill(P.heroDays, { n: fmtNum(days) })}
            </span>
          )}
        </div>
        <img src="/prizes/trophy.svg" alt="" width={84} height={84} />
      </div>
      <div className="prize-hero-line h-px w-full" />
      <div className="flex items-center gap-3">
        <span
          className={`grid size-8 shrink-0 place-items-center rounded-full font-black ${inside ? "prize-hero-ok" : "prize-hero-soft"}`}
        >
          {inside ? <IconCheck size={16} /> : "!"}
        </span>
        <div className="flex min-w-0 flex-col">
          <span className="font-bold">{title}</span>
          <span className="text-caption opacity-85">{body}</span>
        </div>
      </div>
      {progress != null && (
        <div className="flex flex-col gap-1">
          <div className="prize-hero-line h-1.5 overflow-hidden rounded-full">
            <div className="prize-hero-bar h-full" style={{ width: `${Math.min(100, Math.max(0, progress * 100))}%` }} />
          </div>
          {progressLine && <span className="text-caption opacity-85">{progressLine}</span>}
        </div>
      )}
    </section>
  );
}

function Share() {
  const r = useReferral();
  const { name } = usePlatform();
  const [copied, setCopied] = useState(false);
  if (!r) return null;
  const text = fill(withPlatform(V.shareText, name), { code: r.code, link: r.link });
  const copy = async () => {
    try {
      await navigator.clipboard.writeText(r.link);
      setCopied(true);
      setTimeout(() => setCopied(false), 2000);
    } catch {
      /* متصفّحٌ يمنع النسخ — الرابطُ ظاهرٌ. */
    }
  };
  return (
    <section className={`${OUTLINE_CARD} flex flex-col gap-3`}>
      <h2 className="text-section-title">{P.shareTitle}</h2>
      <p className="text-caption text-ink-muted">{P.shareHint}</p>
      <p dir="ltr" className="break-all rounded-rg-md bg-field px-rg-md py-rg-sm text-body">
        {r.link}
      </p>
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
        <Button variant="secondary" size="sm" className="flex min-h-12 items-center justify-center gap-2" onClick={copy}>
          <IconCopy size={16} />
          {copied ? P.copied : P.copy}
        </Button>
        <a
          href={`https://wa.me/?text=${encodeURIComponent(text)}`}
          target="_blank"
          rel="noreferrer"
          className="flex min-h-12 items-center justify-center gap-2 rounded-control bg-success px-4 py-2 text-sm font-bold text-on-bright hover:opacity-90"
        >
          <IconShare size={16} />
          {P.shareAction}
        </a>
      </div>
    </section>
  );
}

function PrizeCard({ item, balance, onOrder }: { item: PrizeItem; balance: number; onOrder: (p: PrizeItem) => void }) {
  const diff = difference(item.price, balance);
  const canOrder = balance > 0 && item.available !== false;
  return (
    /* **هديّتان بكلّ سطر متل كروت السوق** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤، مرآةُ `prizeGrid`) — صورةٌ فوق،
       الاسمُ والسعرُ تحتها، وزرُّ «اطلب» بعرض الكرت. */
    <li className="flex min-w-0 flex-col overflow-hidden rounded-rg-lg border border-line bg-surface">
      <div className="relative aspect-square w-full">
        <ShopPic src={item.image_url ?? item.thumb_url} name={item.name} icon="gifts" iconSize={36} />
      </div>
      <div className="flex flex-col gap-1 p-rg-sm">
        <h3 className="line-clamp-2 min-h-[2.5em] text-sm font-bold" title={item.name}>
          {item.name}
        </h3>
        <Money value={item.price} className="font-bold" />
        {balance > 0 ? (
          <span className="min-h-[1.25em] text-caption text-warning">
            {diff > 0 ? fill(P.payDiff, { amount: fmtMoney(diff) }) : ""}
          </span>
        ) : (
          /* **وبلا رصيدٍ يُقال لماذا الزرُّ باهت** (المالك ٢٠٢٦-١١-٢٥، مرآةُ `prizes_win_first`). */
          <span className="min-h-[1.25em] text-caption text-ink-muted">{P.winFirst}</span>
        )}
        <Button size="sm" className="w-full" disabled={!canOrder} onClick={() => onOrder(item)}>
          {P.order}
        </Button>
      </div>
    </li>
  );
}

function Items({ me }: { me: PrizesMe }) {
  const [data, setData] = useState<PrizeItems | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Item | null>(null);
  const load = () => {
    setError("");
    api<PrizeItems>("/api/v1/prizes/items")
      .then((x) => setData(x ?? {}))
      .catch((e) => setError(errorText(e)));
  };
  useEffect(load, []);

  if (!data) return error ? <ReloadState label={error} onRetry={load} /> : <LoadingState variant="list" rows={2} />;
  const items = data.items ?? [];
  const storeId = data.store_id ?? me.store_id ?? "";
  const balance = me.prize_balance ?? 0;
  if (items.length === 0 || !storeId) return <p className="text-sm text-ink-muted">{P.noItems}</p>;

  const { fits, over } = splitByBalance(items, balance);
  const order = (p: PrizeItem) => setOpen(asItem(p, storeId));
  const group = (title: string, list: PrizeItem[]) =>
    list.length === 0 ? null : (
      <section className="flex flex-col gap-2">
        <h2 className="text-section-title">{title}</h2>
        <ul className="grid grid-cols-2 gap-rg-sm">
          {list.map((p) => (
            <PrizeCard key={p.id} item={p} balance={balance} onOrder={order} />
          ))}
        </ul>
      </section>
    );

  return (
    <>
      {balance > 0 ? (
        <>
          {group(P.itemsFit, fits)}
          {group(P.itemsOver, over)}
        </>
      ) : (
        <>
          {group(P.itemsShowcase, [...fits, ...over])}
          <p className="text-sm font-bold text-accent-text">{P.needBalance}</p>
        </>
      )}
      <ItemSheet item={open} onClose={() => setOpen(null)} storeLink={false} />
    </>
  );
}

function Winners() {
  const [list, setList] = useState<PrizeWinner[]>([]);
  useEffect(() => {
    api<PrizeWinners>("/api/v1/prizes/winners")
      .then((x) => setList(x?.winners ?? []))
      .catch(() => undefined);
  }, []);
  if (list.length === 0) return null;
  return (
    <section className={`${OUTLINE_CARD} flex flex-col gap-2`}>
      <h2 className="text-section-title">{P.winnersTitle}</h2>
      <ul className="flex flex-col divide-y divide-line-soft text-sm">
        {[...list]
          .sort((a, b) => a.rank - b.rank)
          .map((w) => (
            <li key={w.rank} className="flex items-center justify-between gap-2 py-2">
              <span className="min-w-0 truncate">
                {fmtNum(w.rank)}. {w.name}
              </span>
              <Money value={w.amount} className="font-bold" />
            </li>
          ))}
      </ul>
    </section>
  );
}

export default function PrizesPage() {
  const { user, loading: authLoading } = useAuth();
  const { me, error, reload } = usePrizesMe();

  if (authLoading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;
  if (!me) return error ? <ReloadState label={P.loadError} onRetry={reload} /> : <LoadingState variant="list" rows={3} />;

  const amounts = me.amounts ?? [];

  if (!prizesLive(me)) {
    const date = launchLabel(me.launch_on);
    return (
      <div className="mx-auto flex w-full max-w-none flex-col gap-4">
        <ShopPageHeader title={P.title} href="/shop" />
        <Hero
          label={P.heroStarts}
          date={date}
          days={daysUntil(me.launch_on)}
          inside={false}
          title={P.outTitle}
          body={fill(P.outBefore, { date })}
        />
        <Podium amounts={amounts} />
        {/* **والهدايا تحتها** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤) — عرضٌ فقط قبل الافتتاح. */}
        <Items me={{ ...me, prize_balance: 0 }} />
        <Rules per={me.invites_per_entry ?? 5} winners={me.winners_count ?? amounts.length} />
      </div>
    );
  }

  const { done, total } = progressInEntry(me);
  const left = me.invites_to_next ?? total - done;
  const balance = me.prize_balance ?? 0;

  return (
    <div className="mx-auto flex w-full max-w-none flex-col gap-4">
      <ShopPageHeader title={P.title} href="/shop" />

      <Hero
        label={P.heroNext}
        date={me.draw_at ? fmtDate(me.draw_at) : "—"}
        days={daysUntil(me.draw_at)}
        inside={(me.entries ?? 0) > 0}
        title={(me.entries ?? 0) > 0 ? P.inTitle : P.outTitle}
        body={
          (me.entries ?? 0) > 0
            ? fill(P.inBody, { n: fmtNum(me.entries ?? 0) })
            : fill(P.outBody, { n: fmtNum(Math.max(1, left)) })
        }
        progress={total > 0 ? done / total : 0}
        progressLine={fill(P.invitesOf, { n: fmtNum(done), t: fmtNum(total) })}
      />

      <Share />
      <Podium amounts={amounts} />

      {balance > 0 && (
        <section className={`${OUTLINE_CARD} flex flex-col gap-1`}>
          <h2 className="text-section-title">{P.balanceTitle}</h2>
          <Money value={balance} className="text-price-lg font-bold text-accent-text" />
          {me.prize_expires_at && (
            <p className="text-caption text-ink-muted">{fill(P.balanceExpires, { date: fmtDate(me.prize_expires_at) })}</p>
          )}
          <p className="text-caption text-ink-muted">{P.balanceNote}</p>
        </section>
      )}

      <Items me={me} />
      <Winners />
      <Rules per={me.invites_per_entry ?? 5} winners={me.winners_count ?? amounts.length} />
    </div>
  );
}
