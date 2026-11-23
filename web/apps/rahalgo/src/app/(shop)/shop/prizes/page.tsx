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
import { Button, IconCopy, IconShare, LoadingState, Money, ReloadState, usePlatform } from "@rahalgo/ui";
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
const PLACES = [P.place1, P.place2, P.place3, P.place4, P.place5];

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
            <div className={`prize-step prize-step-${r} flex w-full flex-col items-center pt-2 text-center`}>
              <span className={`font-black ${r === 1 ? "text-3xl" : "text-xl"}`}>{fmtNum(r)}</span>
              <span className="px-0.5 text-[11px] leading-tight">{PLACES[r - 1]}</span>
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
    fill(P.ruleInvite, { per: fmtNum(per) }),
    fill(P.ruleMore, { per: fmtNum(per) }),
    P.ruleNew,
    P.ruleDraw,
    fill(P.ruleWinners, { n: fmtNum(winners) }),
    P.ruleUse,
    P.ruleDiff,
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
    <li className="flex items-center gap-3 rounded-rg-lg border border-line bg-surface p-rg-sm">
      <div className="relative h-16 w-16 shrink-0 overflow-hidden rounded-rg-md">
        <ShopPic src={item.thumb_url ?? item.image_url} name={item.name} icon="gifts" iconSize={24} />
      </div>
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <h3 className="truncate text-sm font-bold" title={item.name}>
          {item.name}
        </h3>
        <Money value={item.price} className="text-sm" />
        {balance > 0 && diff > 0 && (
          <span className="text-caption text-warning">{fill(P.payDiff, { amount: fmtMoney(diff) })}</span>
        )}
      </div>
      <Button size="sm" disabled={!canOrder} onClick={() => onOrder(item)}>
        {P.order}
      </Button>
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
        <ul className="flex flex-col gap-2">
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
        <section className={`${OUTLINE_CARD} flex flex-col gap-2 text-center`}>
          {date && <p className="text-price-lg font-bold text-accent-text">{fill(P.teaserSoon, { date })}</p>}
          <p className="text-sm text-ink-muted">{fill(P.teaserBody, { per: fmtNum(me.invites_per_entry ?? 5) })}</p>
        </section>
        <Podium amounts={amounts} />
        <Rules per={me.invites_per_entry ?? 5} winners={me.winners_count ?? amounts.length} />
        {/* **والهدايا تحتها** (ملاحظةُ المالك ٢٠٢٦-١١-٢٤) — عرضٌ فقط قبل الافتتاح. */}
        <Items me={{ ...me, prize_balance: 0 }} />
      </div>
    );
  }

  const { done, total } = progressInEntry(me);
  const left = me.invites_to_next ?? total - done;
  const balance = me.prize_balance ?? 0;

  return (
    <div className="mx-auto flex w-full max-w-none flex-col gap-4">
      <ShopPageHeader title={P.title} href="/shop" />

      <section className={`${OUTLINE_CARD} flex flex-col gap-2`}>
        <h2 className="text-section-title">{P.progressTitle}</h2>
        <div
          role="progressbar"
          aria-valuemin={0}
          aria-valuemax={total}
          aria-valuenow={done}
          className="h-2 overflow-hidden rounded-full bg-field"
        >
          <div className="h-full bg-accent" style={{ width: `${(done / total) * 100}%` }} />
        </div>
        <div className="flex items-center justify-between text-sm">
          <span>{fill(P.invitesOf, { n: fmtNum(done), t: fmtNum(total) })}</span>
          <span className="font-bold">{fill(P.entries, { n: fmtNum(me.entries ?? 0) })}</span>
        </div>
        <p className="text-sm text-accent-text">
          {(me.month_invites ?? 0) === 0 ? P.noInvitesYet : fill(P.toNext, { n: fmtNum(left) })}
        </p>
        {me.draw_at && <p className="text-caption text-ink-muted">{fill(P.drawAt, { date: fmtDate(me.draw_at) })}</p>}
        <p className="text-caption text-ink-muted">{fill(P.drawNote, { n: fmtNum(me.winners_count ?? amounts.length) })}</p>
      </section>

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
