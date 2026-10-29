"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/offers — «العروض»: القائمة · عرضٌ جديد · إيقاف** (تصميمُ ٢٠٢٦-١٠-٢٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `OffersWizard.kt` بوضع المتجر: متجرٌ واحدٌ مختار، **ولا اختيارَ قسم**.)
 *
 * - **رأسٌ بسهم رجوعٍ وعنوان** — في القائمة يعود إلى الأصناف، **وفي النموذج
 *   يعود إلى القائمة.**
 * - **القائمة**: «أنشئ عرضاً» بارزٌ ثمّ البطاقات، **والإيقافُ يُسأل عنه قبل
 *   أن يقع.** والفراغُ رسمٌ وزرٌّ كبير.
 * - **النموذج**: نوعُ الخصم مربّعان متساويان، والصنفُ مربّعاتٌ بعلامة ✓،
 *   والقيمةُ حقلٌ بوحدته ونسبٌ سريعة، والمدّةُ مربّعات، ثمّ ملخّصٌ و«حفظ / إلغاء».
 *   **ولا سطرَ شرح.**
 *
 * **والإنشاءُ مرّةً مهما أُعيد** (`Attempt.OFFER`) — **والمحرّكُ هو الحَكَم**
 * على النافذة والقيمة (`bad_offer_window` …).
 */

import { useCallback, useMemo, useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText, fmtDateTime, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Confirm,
  FormActions,
  IconAdd,
  IconCheck,
  IconClose,
  IconPromos,
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
import { CheckBadge, ToolHeader } from "@/components/store/ToolParts";

const m = getMessages(defaultLocale);
const O = m.store.offers;

const SLOT = "merchant_offer";
/** **مرآةُ `OfferDuration.PRESET_HOURS`.** */
const PRESET_HOURS = [1, 3, 6, 12, 24, 48, 72, 168] as const;
/** **نسبٌ تُضغط ضغطةً** — والحقلُ باقٍ لما سواها. */
const QUICK_PERCENTS = [10, 15, 20, 25, 30, 50] as const;

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

/** **بطاقةُ قسمٍ من النموذج** — بحدٍّ رفيعٍ وعنوانٍ عريض. */
function FormCard({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="surface flex flex-col gap-2 rounded-card border border-line p-3">
      <h3 className="heading-card">{title}</h3>
      {children}
    </section>
  );
}

/** **مربّعٌ يُختار** — والمختارُ بحدّ المنصّة وأرضِها و✓ في زاويته. */
function Tile({
  on,
  onClick,
  children,
  testId,
  className = "",
}: {
  on: boolean;
  onClick: () => void;
  children: ReactNode;
  testId?: string;
  className?: string;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={on}
      onClick={onClick}
      data-testid={testId}
      className={`relative flex min-h-14 w-full flex-col justify-center rounded-control border p-2 text-start transition-colors ${
        on ? "border-2 border-accent bg-accent-tint" : "border-line bg-surface hover:bg-row-hover"
      } ${className}`}
    >
      {children}
      {on && <CheckBadge className="absolute end-1.5 top-1.5" />}
    </button>
  );
}

function OfferCard({ o, stopping, onStop }: { o: StoreOffer; stopping: boolean; onStop: () => void }) {
  const live = o.status === "active";
  const saved = Math.max(0, o.price_before - o.price_after);
  const badge =
    o.discount_percent != null
      ? fill(O.pctBadge, { n: fmtNum(o.discount_percent) })
      : o.discount_amount != null
        ? `−${fmtMoney(o.discount_amount)}`
        : "";
  const status = (O.status as Record<string, string>)[o.status] ?? O.status.unknown;
  const tone =
    o.status === "active"
      ? "bg-success-tint text-success"
      : o.status === "scheduled"
        ? "bg-accent-tint text-accent-text"
        : "bg-field text-ink-muted";
  return (
    <article
      className={`surface flex flex-col gap-2 rounded-card border p-3 ${live ? "border-success-edge" : "border-line"}`}
      data-testid="store-offer"
      data-offer-id={o.id}
      data-status={o.status}
    >
      <header className="flex items-center gap-2">
        {badge && (
          <span
            className={`flex h-12 min-w-12 shrink-0 items-center justify-center rounded-control px-2 text-sm font-bold ${
              live ? "bg-accent text-on-bright" : "bg-field text-ink-muted"
            }`}
          >
            {badge}
          </span>
        )}
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate font-bold">{o.item_name}</span>
          <span className="flex items-baseline gap-2">
            <span className={`font-bold ${live ? "text-success" : "text-ink"}`}>{fmtMoney(o.price_after)}</span>
            <span className="text-xs text-ink-muted line-through">{fmtMoney(o.price_before)}</span>
          </span>
        </span>
        <span className={`shrink-0 rounded-full px-2 py-0.5 text-2xs font-bold ${tone}`}>{status}</span>
      </header>
      {(saved > 0 && LIVE.has(o.status)) || (live && o.ends_at) ? (
        <p className="flex flex-wrap gap-x-3 text-xs text-ink-muted">
          {saved > 0 && LIVE.has(o.status) && <span className={live ? "text-success" : ""}>{fill(O.saves, { amount: fmtMoney(saved) })}</span>}
          {live && o.ends_at && <span>{fill(O.endsAt, { when: fmtDateTime(o.ends_at) })}</span>}
        </p>
      ) : null}
      {LIVE.has(o.status) && (
        <Button
          variant="secondary"
          className="inline-flex w-fit items-center gap-1.5 border-danger-edge text-danger"
          disabled={stopping}
          onClick={onStop}
          data-testid="offer-stop"
        >
          <IconClose size={16} />
          {O.stopDo}
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
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-3" data-testid="offer-form">
      <ToolHeader title={O.newTitle} onBack={onCancel} />

      <FormCard title={O.fType}>
        <div role="radiogroup" aria-label={O.fType} className="grid grid-cols-2 gap-2">
          {(
            [
              [true, O.unitPct, O.modePercent],
              [false, O.unitSyp, O.modeFixed],
            ] as const
          ).map(([pct, glyph, label]) => {
            const on = percentMode === pct;
            return (
              <Tile
                key={label}
                on={on}
                testId={pct ? "offer-type-percent" : "offer-type-fixed"}
                className="min-h-24 items-center"
                onClick={() => {
                  if (on) return;
                  setPercentMode(pct);
                  setValue("");
                }}
              >
                <span
                  className={`flex h-12 w-12 items-center justify-center self-center rounded-full text-lg font-medium ${
                    on ? "bg-accent text-on-bright" : "bg-field text-ink-muted"
                  }`}
                  aria-hidden
                >
                  {glyph}
                </span>
                <span className={`mt-1 self-center font-bold ${on ? "text-accent-text" : "text-ink"}`}>{label}</span>
              </Tile>
            );
          })}
        </div>
      </FormCard>

      <FormCard title={O.item}>
        <Input
          type="search"
          aria-label={O.search}
          placeholder={O.search}
          icon={<IconSearch size={16} />}
          value={query}
          onChange={(e) => setQuery(e.target.value.slice(0, 40))}
        />
        {shown.length === 0 && <p className="text-sm text-ink-muted">{query.trim() ? O.noMatch : O.noItems}</p>}
        <div role="radiogroup" aria-label={O.item} className="grid max-h-96 grid-cols-2 gap-2 overflow-y-auto sm:grid-cols-3">
          {shown.map((i) => {
            const on = picked === i.id;
            const note = taken.get(i.id);
            return (
              <Tile key={i.id} on={on} onClick={() => setPicked(i.id)} testId="offer-item">
                <span className={`line-clamp-2 pe-5 text-sm font-bold ${on ? "text-accent-text" : "text-ink"}`}>{i.name}</span>
                <span className="text-xs text-ink-muted">{fmtMoney(i.price)}</span>
                {note && (
                  <span className="mt-1 w-fit rounded-control bg-warning-tint px-1.5 py-0.5 text-2xs text-warning">
                    {note === "scheduled" ? O.itemScheduled : O.itemTaken}
                  </span>
                )}
              </Tile>
            );
          })}
        </div>
      </FormCard>

      <FormCard title={O.fValue}>
        <label className="flex flex-col gap-1">
          <span className="text-sm font-medium">{percentMode ? O.percent : O.amount}</span>
          <span
            className={`flex h-12 items-center gap-2 rounded-control border bg-field px-3 focus-within:border-accent ${
              value && !valueOK ? "border-danger" : "border-line"
            }`}
          >
            <input
              id="offer-value"
              value={value}
              onChange={(e) => setValue(e.target.value.replace(/\D/g, "").slice(0, percentMode ? 2 : 9))}
              inputMode="numeric"
              dir="ltr"
              className="min-w-0 flex-1 bg-transparent text-lg font-medium text-ink outline-none"
            />
            <span className="shrink-0 font-bold text-ink-muted">{percentMode ? O.unitPct : O.unitSyp}</span>
          </span>
        </label>
        {value && !valueOK && (
          <p className="text-xs text-danger">{percentMode ? O.badPercent : fill(O.badAmount, { price: fmtNum(price) })}</p>
        )}
        {percentMode && (
          <div className="flex flex-wrap gap-2">
            {QUICK_PERCENTS.map((p) => (
              <button
                key={p}
                type="button"
                aria-pressed={n === p}
                onClick={() => setValue(String(p))}
                className={`min-h-10 rounded-full border px-4 text-sm ${
                  n === p ? "border-accent bg-accent font-bold text-on-bright" : "border-line bg-surface text-ink hover:bg-row-hover"
                }`}
              >
                {fill(O.pct, { n: fmtNum(p) })}
              </button>
            ))}
          </div>
        )}
      </FormCard>

      <FormCard title={O.fPeriod}>
        <div role="radiogroup" aria-label={O.fPeriod} className="grid grid-cols-2 gap-2 sm:grid-cols-4">
          {PRESET_HOURS.map((h) => (
            <button
              key={h}
              type="button"
              role="radio"
              aria-checked={h === hours}
              onClick={() => setHours(h)}
              className={`min-h-14 rounded-control border px-1 text-center text-sm font-bold ${
                h === hours ? "border-accent bg-accent text-on-bright" : "border-line bg-surface text-ink hover:bg-row-hover"
              }`}
            >
              {durationLabel(h)}
            </button>
          ))}
        </div>
      </FormCard>

      {item && (
        <FormCard title={O.previewTitle}>
          <p className="font-bold">{item.name}</p>
          <p className="text-sm">{fill(O.previewPrice, { price: fmtMoney(price) })}</p>
          {valueOK && (
            <p className="font-bold text-success">
              {fill(O.previewDiscount, { value: percentMode ? fill(O.pct, { n: fmtNum(n) }) : fmtMoney(n) })}
            </p>
          )}
          <p className="text-xs text-ink-muted">{fill(O.previewFor, { d: durationLabel(hours) })}</p>
        </FormCard>
      )}

      {error && <Alert>{error}</Alert>}
      <FormActions
        onSave={() => void create()}
        onCancel={onCancel}
        busy={busy || !item || !valueOK}
        saveIcon={<IconCheck size={18} />}
        cancelIcon={<IconClose size={18} />}
      />
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
  const [askStop, setAskStop] = useState<StoreOffer | null>(null);

  const items = useMemo<PickItem[]>(
    () =>
      (data?.menu ?? [])
        .flatMap((s) => s.items)
        .filter((i) => i.approved !== false && i.available)
        .map((i) => ({ id: i.id, name: i.name, price: i.price })),
    [data],
  );

  async function stop(o: StoreOffer) {
    setAskStop(null);
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

  const start = () => {
    setCreated(false);
    setComposing(true);
  };

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-3">
      <ToolHeader title={O.title} href="/store/menu" />
      {created && <Alert tone="success">{O.createdTop}</Alert>}
      {rows.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 rounded-card border border-line px-4 py-8 text-center">
          <span className="flex h-22 w-22 items-center justify-center rounded-full bg-accent-tint text-accent-text" aria-hidden>
            <IconPromos size={40} />
          </span>
          <p className="heading-card">{O.emptyTitle}</p>
          <Button className="inline-flex w-full items-center justify-center gap-1.5 font-bold" data-testid="offer-new" onClick={start}>
            <IconAdd size={18} />
            {O.add}
          </Button>
        </section>
      ) : (
        <>
          <Button className="inline-flex w-full items-center justify-center gap-1.5 font-bold" data-testid="offer-new" onClick={start}>
            <IconAdd size={18} />
            {O.add}
          </Button>
          {rows.map((o) => (
            <OfferCard key={o.id} o={o} stopping={stopping === o.id} onStop={() => setAskStop(o)} />
          ))}
        </>
      )}
      <Confirm
        open={askStop != null}
        title={O.stopQ}
        body={O.stopBody}
        confirmLabel={O.stopDo}
        busy={stopping !== ""}
        onConfirm={() => askStop && void stop(askStop)}
        onCancel={() => setAskStop(null)}
      />
    </div>
  );
}
