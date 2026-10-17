"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/offers — «عروض متجري»: القائمة · عرضٌ جديد · إيقاف**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `OffersWizard.kt` و`OffersWizardViewModel.kt` بوضع المتجر:
 *  متجرٌ واحدٌ مختار، **ولا اختيارَ قسم** — `picksStore = false`،
 *  `picksSection = false`.)
 *
 * **القائمةُ أوّلاً** بزرّ «＋ عرض جديد» فوقها، **والنموذجُ** بثلاث بطاقات:
 * الصنف (ببحث، **وما عليه عرضٌ جارٍ أو مجدولٌ يُوسَم** — الجديدُ يحلّ
 * محلّه) · الخصم (نسبةٌ ١–٩٠ أو مبلغٌ أقلُّ من سعر الصنف) · المدّة (مددٌ
 * تُضغط ولا يُكتب تاريخ) · **ثمّ المعاينة وزرُّ الإنشاء.**
 *
 * **والإنشاءُ مرّةً مهما أُعيد** (`Attempt.OFFER`) — **والمحرّكُ هو الحَكَم**
 * على النافذة والقيمة (`bad_offer_window` …).
 */

import { useCallback, useMemo, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDateTime, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Chips,
  IconCheck,
  IconSearch,
  Input,
  LoadingState,
  ReloadState,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { MenuSection, StoreOffer } from "@/lib/store/types";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const O = m.store.offers;

const SLOT = "merchant_offer";
/** **مرآةُ `OfferDuration.PRESET_HOURS`.** */
const PRESET_HOURS = [1, 3, 6, 12, 24, 48, 72, 168] as const;

/** **«٣ ساعات» و«يومان» لا «٤٨ ساعة»** — مرآةُ `OfferDuration.label` بجمع العربيّة. */
function plural(n: number, f: { one: string; two: string; few: string; many: string }): string {
  if (n === 1) return f.one;
  if (n === 2) return f.two;
  const r = n % 100;
  return fill(r >= 3 && r <= 10 ? f.few : f.many, { n: fmtNum(n) });
}
function durationLabel(hours: number): string {
  return hours >= 24 && hours % 24 === 0 ? plural(hours / 24, O.days) : plural(hours, O.hours);
}

const LIVE = new Set(["active", "scheduled"]);

function OfferCard({ o, stopping, onStop }: { o: StoreOffer; stopping: boolean; onStop: () => void }) {
  const live = o.status === "active";
  const saved = Math.max(0, o.price_before - o.price_after);
  const badge =
    o.discount_percent != null ? fill(O.pctBadge, { n: fmtNum(o.discount_percent) }) : o.discount_amount != null ? `−${fmtMoney(o.discount_amount)}` : "";
  const status = (O.status as Record<string, string>)[o.status] ?? O.status.unknown;
  return (
    <article
      className={`surface flex flex-col gap-2 rounded-card border p-3 ${live ? "border-accent-edge" : "border-line"}`}
      data-testid="store-offer"
      data-offer-id={o.id}
      data-status={o.status}
    >
      <header className="flex items-center gap-2">
        {badge && (
          <span className={`rounded-control px-2 py-0.5 text-sm font-bold ${live ? "bg-accent-tint text-accent-text" : "bg-field text-ink-muted"}`}>
            {badge}
          </span>
        )}
        <span className="min-w-0 flex-1 font-bold">{o.item_name}</span>
        <span className={`text-xs ${live ? "text-accent-text" : "text-ink-muted"}`}>{status}</span>
      </header>
      <p className="flex items-baseline gap-2">
        <span className={`font-bold ${live ? "text-success" : "text-ink"}`}>{fmtMoney(o.price_after)}</span>
        <span className="text-xs text-ink-muted line-through">{fmtMoney(o.price_before)}</span>
      </p>
      {saved > 0 && LIVE.has(o.status) && (
        <p className={`text-xs ${live ? "text-success" : "text-ink-muted"}`}>{fill(O.saves, { amount: fmtMoney(saved) })}</p>
      )}
      {live && o.ends_at && <p className="text-xs text-ink-muted">{fill(O.endsAt, { when: fmtDateTime(o.ends_at) })}</p>}
      {LIVE.has(o.status) && (
        <Button variant="ghost" className="w-fit text-danger" disabled={stopping} onClick={onStop} data-testid="offer-stop">
          {O.stop}
        </Button>
      )}
    </article>
  );
}

interface PickItem {
  id: string;
  name: string;
  price: number;
}

function NewOffer({
  items,
  rows,
  onCreated,
  onCancel,
}: {
  items: PickItem[];
  rows: StoreOffer[];
  onCreated: () => void;
  onCancel: () => void;
}) {
  const { store } = useStore();
  const [query, setQuery] = useState("");
  const [picked, setPicked] = useState("");
  const [percentMode, setPercentMode] = useState(true);
  const [value, setValue] = useState("");
  const [hours, setHours] = useState(24);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const item = items.find((i) => i.id === picked) ?? null;
  const n = Number(value) || 0;
  const price = item?.price ?? 0;
  const valueOK = percentMode ? n >= 1 && n <= 90 : n > 0 && (price <= 0 || n < price);
  const taken = useMemo(() => {
    const t = new Map<string, string>();
    for (const o of rows) if (LIVE.has(o.status) && o.menu_item_id) t.set(o.menu_item_id, o.status);
    return t;
  }, [rows]);
  const shown = query.trim() ? items.filter((i) => i.name.includes(query.trim())) : items;

  async function create() {
    if (!store) return;
    if (!item || !valueOK) {
      setError(O.badInput);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/merchant/stores/${store.id}/offers`, {
        method: "POST",
        headers: { "Idempotency-Key": attemptKey(SLOT) },
        body: JSON.stringify({
          title: O.defaultTitle,
          menu_item_id: item.id,
          ...(percentMode ? { discount_percent: n } : { discount_amount: n }),
          ends_at: new Date(Date.now() + hours * 3_600_000).toISOString(),
        }),
      });
      clearAttempt(SLOT);
      onCreated();
    } catch (e) {
      if (isDecided(e)) clearAttempt(SLOT);
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex flex-col gap-3" data-testid="offer-form">
      <header className="flex items-center gap-2">
        <h2 className="heading-page min-w-0 flex-1">{O.newTitle}</h2>
        <Button variant="ghost" onClick={onCancel}>
          {O.backToList}
        </Button>
      </header>

      <section className="surface flex flex-col gap-2 rounded-card p-3">
        <h3 className="heading-card">{O.item}</h3>
        <Input
          type="search"
          aria-label={O.search}
          placeholder={O.search}
          icon={<IconSearch size={16} />}
          value={query}
          onChange={(e) => setQuery(e.target.value.slice(0, 40))}
        />
        {shown.length === 0 && <p className="text-xs text-ink-muted">{query.trim() ? O.noMatch : O.noItems}</p>}
        <ul className="flex max-h-80 flex-col gap-1.5 overflow-y-auto">
          {shown.map((i) => {
            const on = picked === i.id;
            const note = taken.get(i.id);
            return (
              <li key={i.id}>
                <button
                  type="button"
                  aria-pressed={on}
                  onClick={() => setPicked(i.id)}
                  data-testid="offer-item"
                  className={`flex w-full items-center gap-2 rounded-control border px-3 py-2.5 text-start text-sm ${
                    on ? "border-accent bg-accent-tint font-bold text-accent-text" : "border-line bg-surface hover:bg-row-hover"
                  }`}
                >
                  <span className="min-w-0 flex-1 truncate">{`${i.name}  ·  ${fmtMoney(i.price)}`}</span>
                  {note ? (
                    <span className="shrink-0 rounded-control bg-warning-tint px-1.5 py-0.5 text-2xs text-warning">
                      {note === "scheduled" ? O.itemScheduled : O.itemTaken}
                    </span>
                  ) : (
                    on && <IconCheck size={16} className="shrink-0" />
                  )}
                </button>
              </li>
            );
          })}
        </ul>
      </section>

      <section className="surface flex flex-col gap-2 rounded-card p-3">
        <h3 className="heading-card">{O.discount}</h3>
        <Chips
          items={[
            { id: "percent", label: O.modePercent },
            { id: "fixed", label: O.modeFixed },
          ]}
          value={percentMode ? "percent" : "fixed"}
          onChange={(id) => {
            setPercentMode(id === "percent");
            setValue("");
          }}
        />
        <Input
          id="offer-value"
          label={percentMode ? O.percent : O.amount}
          inputMode="numeric"
          dir="ltr"
          value={value}
          onChange={(e) => setValue(e.target.value.replace(/\D/g, "").slice(0, percentMode ? 2 : 9))}
          error={
            value && !valueOK
              ? percentMode
                ? O.badPercent
                : fill(O.badAmount, { price: fmtNum(price) })
              : undefined
          }
        />
        {!(value && !valueOK) && <p className="text-xs text-ink-muted">{percentMode ? O.percentHint : O.amountHint}</p>}
      </section>

      <section className="surface flex flex-col gap-2 rounded-card p-3">
        <h3 className="heading-card">{O.duration}</h3>
        <Chips
          wrap
          items={PRESET_HOURS.map((h) => ({ id: String(h), label: durationLabel(h) }))}
          value={String(hours)}
          onChange={(id) => setHours(Number(id))}
        />
      </section>

      <section className="flex flex-col gap-1.5 rounded-card border border-success-edge bg-success-tint p-3">
        <p className="text-xs text-ink-muted">{O.previewTitle}</p>
        {item ? (
          <>
            <p className="font-bold">{item.name}</p>
            <p className="text-sm">{fill(O.previewPrice, { price: fmtMoney(price) })}</p>
            {valueOK && (
              <p className="font-bold text-success">
                {fill(O.previewDiscount, { value: percentMode ? fill(O.pct, { n: fmtNum(n) }) : fmtMoney(n) })}
              </p>
            )}
            <p className="text-xs text-ink-muted">{fill(O.previewFor, { d: durationLabel(hours) })}</p>
            <p className="text-xs text-ink-muted">{O.previewFinal}</p>
          </>
        ) : (
          <p className="text-sm text-ink-muted">{O.previewPick}</p>
        )}
        <p className="text-xs text-ink-muted">{O.borneMe}</p>
        {error && <Alert>{error}</Alert>}
        <Button className="mt-1 w-full" disabled={busy || !item || !valueOK} onClick={() => void create()} data-testid="offer-create">
          {O.create}
        </Button>
      </section>
    </div>
  );
}

export default function OffersPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const toast = useShopToast();
  const load = useCallback(async () => {
    const [offers, menu] = await Promise.all([
      api<{ offers: StoreOffer[] }>(`/api/v1/merchant/stores/${id}/offers`),
      api<MenuSection[]>(`/api/v1/merchant/stores/${id}/menu`),
    ]);
    return { offers: offers.offers ?? [], menu };
  }, [id]);
  const { data, loading, error, reload, setData } = useLiveData(load, ["menu", "offers"], [id]);
  const [composing, setComposing] = useState(false);
  const [created, setCreated] = useState(false);
  const [stopping, setStopping] = useState("");

  const items = useMemo<PickItem[]>(
    () =>
      (data?.menu ?? [])
        .flatMap((s) => s.items)
        .filter((i) => i.approved !== false && i.available)
        .map((i) => ({ id: i.id, name: i.name, price: i.price })),
    [data],
  );

  async function stop(o: StoreOffer) {
    setStopping(o.id);
    try {
      const updated = await api<StoreOffer>(`/api/v1/merchant/stores/${id}/offers/${o.id}/stop`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      setData((d) => (d ? { ...d, offers: d.offers.map((x) => (x.id === updated.id ? updated : x)) } : d));
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setStopping("");
    }
  }

  if (loading && !data) return <LoadingState variant="list" rows={3} label={O.loading} />;
  if (error && !data) return <ReloadState onRetry={() => void reload()} />;
  const rows = data?.offers ?? [];

  if (composing) {
    return (
      <NewOffer
        items={items}
        rows={rows}
        onCancel={() => setComposing(false)}
        onCreated={() => {
          setComposing(false);
          setCreated(true);
          void reload();
        }}
      />
    );
  }

  return (
    <div className="flex flex-col gap-3">
      <h2 className="heading-page">{O.title}</h2>
      {created && <Alert tone="success">{O.createdTop}</Alert>}
      <Button
        className="w-full"
        data-testid="offer-new"
        onClick={() => {
          setCreated(false);
          setComposing(true);
        }}
      >
        {O.newButton}
      </Button>
      {rows.length === 0 && <p className="text-sm text-ink-muted">{O.empty}</p>}
      {rows.map((o) => (
        <OfferCard key={o.id} o={o} stopping={stopping === o.id} onStop={() => void stop(o)} />
      ))}
    </div>
  );
}
