"use client";

/**
 * **الخصوماتُ على الأصناف** — تبويبٌ في صفحة العروض.
 *
 * (قراراتُ المالك ٢٠٢٦-١٠-٠٤:)
 *  - الخصمُ من الإدارة على حساب المنصّة وحدَها — المتجرُ يعمل خصمَه من تطبيقه.
 *  - فوق ٢٠٪ ينتظر موافقةَ الماليّة.
 *  - الإشعارُ خيارٌ في النافذة بعدد مستلميه، لزبائن منطقة المتجر وحدَهم،
 *    وإعادةُ التفعيل لا تُبلّغ إلّا إن اختير.
 *  - الحالةُ الحقيقيّة، ومين عمله، والسعران بعملةٍ، والخطأُ داخل النافذة.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Button,
  Badge,
  Modal,
  Input,
  Alert,
  Chips,
  Checkbox,
  Pagination,
  LoadingState,
  IconStore,
  IconSearch,
  IconAdd,
  IconDate,
  Money,
  FormActions,
  Confirm,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const P = m.admin.offersPage;
const O = m.admin.promosOwner;

type OfferStatus = "active" | "scheduled" | "stopped" | "expired" | "pending" | "rejected";

interface Offer {
  id: string;
  title: string;
  body: string;
  menu_item_id: string | null;
  item_name: string;
  merchant_name: string;
  price_before: number;
  price_after: number;
  discount_percent: number | null;
  discount_amount: number | null;
  borne_by: "platform" | "merchant" | null;
  ends_at: string | null;
  active: boolean;
  live: boolean;
  status: "active" | "scheduled" | "stopped" | "expired";
  approval_state?: "ok" | "pending" | "rejected";
  created_by_name?: string;
  created_by_kind?: "admin" | "merchant" | "rep";
  item_available: boolean;
}

interface Item {
  id: string;
  name: string;
  merchant_name: string;
}

const STATUS: Record<OfferStatus, { label: string; tone: "success" | "neutral" | "warning" | "danger" | "info" }> = {
  active: { label: O.stActive, tone: "success" },
  scheduled: { label: O.stScheduled, tone: "info" },
  stopped: { label: O.stPaused, tone: "warning" },
  expired: { label: O.stExpired, tone: "neutral" },
  pending: { label: O.stPending, tone: "info" },
  rejected: { label: O.stRejected, tone: "danger" },
};

const BY: Record<string, string> = { admin: O.byAdmin, merchant: O.byMerchant, rep: O.byRep };

function statusOf(o: Offer): OfferStatus {
  if (o.approval_state === "pending") return "pending";
  if (o.approval_state === "rejected") return "rejected";
  return o.status;
}

function cutText(o: Offer): string {
  if (o.discount_percent != null) return `−${o.discount_percent}%`;
  if (o.discount_amount != null) return `−${fmtMoney(o.discount_amount)}`;
  return "";
}

const PER_PAGE = 20;

export default function DiscountsTab({ onChanged }: { onChanged?: () => void }) {
  const [rows, setRows] = useState<Offer[] | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState("");
  const [status, setStatus] = useState<OfferStatus | "all">("all");
  const [page, setPage] = useState(1);
  const [activating, setActivating] = useState<Offer | null>(null);
  const [busyId, setBusyId] = useState("");

  const load = useCallback(() => {
    api<{ offers: Offer[] }>("/api/v1/admin/offers")
      .then((r) => {
        setRows(r.offers ?? []);
        setError("");
      })
      .catch((err) => {
        setRows([]);
        setError(errorText(err));
      });
  }, []);
  useEffect(load, [load]);

  const filtered = useMemo(() => {
    const needle = q.trim();
    return (rows ?? []).filter(
      (o) =>
        (status === "all" || statusOf(o) === status) &&
        (!needle ||
          o.title.includes(needle) ||
          o.item_name.includes(needle) ||
          o.merchant_name.includes(needle)),
    );
  }, [rows, q, status]);
  const shown = filtered.slice((page - 1) * PER_PAGE, page * PER_PAGE);

  async function deactivate(o: Offer) {
    if (busyId) return;
    setBusyId(o.id);
    try {
      await api(`/api/v1/admin/offers/${o.id}/active`, {
        method: "POST",
        body: JSON.stringify({ active: false }),
      });
      load();
      onChanged?.();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusyId("");
    }
  }

  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) — فلاترُ ببطاقةٍ واحدة، وكرتٌ لكلّ خصمٍ بدل الجدول.
  return (
    <div className="space-y-4">
      <section className="surface space-y-3 p-3">
        <div className="flex flex-wrap items-end gap-2">
          <Input
            id="offer-search"
            label={O.search}
            icon={<IconSearch size={16} />}
            placeholder={O.searchDiscounts}
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setPage(1);
            }}
            wrapperClassName="min-w-0 flex-1"
          />
          <Button onClick={() => setOpen(true)} className="flex items-center gap-1.5">
            <IconAdd size={16} />
            {P.create}
          </Button>
        </div>
        <Chips
          wrap
          items={[
            { id: "all" as const, label: O.fAll },
            ...(Object.keys(STATUS) as OfferStatus[]).map((k) => ({ id: k, label: STATUS[k].label })),
          ]}
          value={status}
          onChange={(k) => {
            setStatus(k);
            setPage(1);
          }}
        />
      </section>

      {error && <Alert>{error}</Alert>}

      {rows === null ? (
        <LoadingState variant="list" />
      ) : shown.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-field text-ink-muted">
            <IconStore size={28} />
          </span>
          <h2 className="heading-section">{rows.length === 0 ? P.empty : O.noMatch}</h2>
          <p className="max-w-md text-sm text-ink-muted">{rows.length === 0 ? O.discountsEmptyHint : O.noMatchHint}</p>
        </section>
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {shown.map((o) => (
            <OfferCard
              key={o.id}
              o={o}
              busy={busyId === o.id}
              onDeactivate={() => void deactivate(o)}
              onActivate={() => setActivating(o)}
            />
          ))}
        </ul>
      )}

      {filtered.length > PER_PAGE && (
        <div className="flex justify-center">
          <Pagination page={page} total={filtered.length} perPage={PER_PAGE} onChange={setPage} />
        </div>
      )}

      {open && (
        <CreateOfferModal
          onClose={() => setOpen(false)}
          onSaved={() => {
            setOpen(false);
            load();
            onChanged?.();
          }}
        />
      )}
      {activating && (
        <ActivateModal
          offer={activating}
          onClose={() => setActivating(null)}
          onDone={() => {
            setActivating(null);
            load();
            onChanged?.();
          }}
        />
      )}
    </div>
  );
}

/**
 * **كرتُ الخصم** — تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦): العنوانُ والصنفُ والمتجر، والسعران
 * (قبل مشطوب وبعد كبير)، ونسبةُ الخصم، ومين بيتحمّل ومين عمله، والانتهاء — والفعل.
 */
function OfferCard({
  o,
  busy,
  onDeactivate,
  onActivate,
}: {
  o: Offer;
  busy: boolean;
  onDeactivate: () => void;
  onActivate: () => void;
}) {
  const s = STATUS[statusOf(o)];
  const st = statusOf(o);
  const canAct = st !== "pending" && st !== "rejected" && st !== "expired";
  const cut = cutText(o);
  return (
    <li className="surface flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
          <IconStore size={20} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold">{o.title}</span>
          <span className="block truncate text-xs text-ink-muted">
            {o.item_name} — {o.merchant_name}
          </span>
        </span>
        <Badge variant={s.tone} className="shrink-0">
          {s.label}
        </Badge>
      </div>

      {!o.item_available && (
        <Badge variant="warning" className="self-start">
          {O.itemUnavailable}
        </Badge>
      )}

      <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
        <span className="min-w-0">
          <span className="block text-2xs text-ink-muted">{O.priceBefore}</span>
          <span className="block text-sm text-ink-muted line-through">
            <Money value={o.price_before} />
          </span>
        </span>
        {cut && (
          <span className="rounded-control border-2 border-dashed border-success px-2 py-0.5 text-sm font-bold tabular-nums text-success" dir="ltr">
            {cut}
          </span>
        )}
        <span className="text-end">
          <span className="block text-2xs text-ink-muted">{O.priceAfter}</span>
          <span className="figure block text-success">
            <Money value={o.price_after} />
          </span>
        </span>
      </div>

      <div className="grid grid-cols-1 gap-x-3 gap-y-1 text-2xs text-ink-muted sm:grid-cols-2">
        <span>
          {O.borneCol}: <span className="text-ink">{o.borne_by === "platform" ? P.byPlatform : P.byMerchant}</span>
        </span>
        <span className="flex items-center gap-1">
          <IconDate size={12} className="shrink-0" />
          {O.endsCol}: {o.ends_at ? fmtDate(o.ends_at) : P.noEnd}
        </span>
        <span className="sm:col-span-2">
          {O.createdBy}: {BY[o.created_by_kind ?? "admin"] ?? ""}
          {o.created_by_name ? ` · ${o.created_by_name}` : ""}
        </span>
      </div>

      {canAct && (
        <div className="mt-auto border-t border-line-soft pt-3">
          {o.active ? (
            <Button variant="secondary" className="w-full" disabled={busy} onClick={onDeactivate}>
              {busy ? O.busy : P.deactivate}
            </Button>
          ) : (
            <Button className="w-full" onClick={onActivate}>
              {P.activate}
            </Button>
          )}
        </div>
      )}
    </li>
  );
}

/** **عددُ من يصله الإشعار** — زبائنُ منطقة المتجر. */
function useAudience(itemID: string): number | null {
  const [n, setN] = useState<number | null>(null);
  useEffect(() => {
    if (!itemID) {
      setN(null);
      return;
    }
    api<{ count: number }>(`/api/v1/admin/offers/audience?menu_item_id=${encodeURIComponent(itemID)}`)
      .then((r) => setN(r.count))
      // @empty-ok — **العددُ إخبارٌ قبل الإرسال**؛ إن سقط يبقى الخيارُ بلا رقم.
      .catch(() => setN(null));
  }, [itemID]);
  return n;
}

function NotifyChoice({
  count,
  checked,
  onChange,
  id,
}: {
  count: number | null;
  checked: boolean;
  onChange: (v: boolean) => void;
  id: string;
}) {
  if (count === 0) return <p className="text-sm text-ink-muted">{O.notifyNone}</p>;
  return (
    <Checkbox
      id={id}
      checked={checked}
      onChange={(e) => onChange(e.target.checked)}
      label={O.notify.replace("{n}", count === null ? "…" : fmtNum(count))}
    />
  );
}

function CreateOfferModal({ onClose, onSaved }: { onClose: () => void; onSaved: () => void }) {
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [itemQuery, setItemQuery] = useState("");
  const [items, setItems] = useState<Item[]>([]);
  const [itemID, setItemID] = useState("");
  const [method, setMethod] = useState<"percent" | "amount">("percent");
  const [value, setValue] = useState("");
  const [endsOn, setEndsOn] = useState("");
  const [notify, setNotify] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  /** **على الصنف عرضٌ من المتجر** — يُسأل قبل استبداله (قرارُ المالك ٢٠٢٦-١٠-١٥). */
  const [askReplace, setAskReplace] = useState(false);
  const audience = useAudience(itemID);

  /** **والصنفُ يُبحث لا يُكتب معرّفُه** — لا أحدَ يحفظ `uuid`. */
  useEffect(() => {
    if (itemQuery.trim().length < 2 || itemID) return;
    const t = setTimeout(() => {
      api<{ items: Item[] }>(
        `/api/v1/public/search/items?q=${encodeURIComponent(itemQuery.trim())}`,
      )
        .then((r) => setItems(r.items ?? []))
        // @empty-ok — **بحثٌ يُعاد بحرفٍ واحد**: من كتب فلم يجد يُضيف حرفاً
        // فيُعاد النداء، **ولا قرارَ يُبنى على فراغه.**
        .catch(() => setItems([]));
    }, 250);
    return () => clearTimeout(t);
  }, [itemQuery, itemID]);

  async function submit(replace = false) {
    // **والضغطُ الثاني لا يُرسل ثانيةً.**
    if (busy) return;
    setBusy(true);
    setError("");
    setAskReplace(false);
    try {
      const o = await api<Offer>("/api/v1/admin/offers", {
        method: "POST",
        body: JSON.stringify({
          title: title.trim(),
          body: body.trim(),
          menu_item_id: itemID,
          ...(method === "percent"
            ? { discount_percent: Number(value) }
            : { discount_amount: Number(value) }),
          borne_by: "platform",
          ends_on: endsOn || undefined,
          notify,
          ...(replace ? { replace: true } : {}),
        }),
      });
      if (o.approval_state === "pending") {
        setNotice(O.pendingSaved);
        setTimeout(onSaved, 1500);
        return;
      }
      onSaved();
    } catch (e) {
      // **وعرضُ المتجر لا يُستبدل إلّا بتأكيد** — الخادمُ يردّ `offer_merchant_live`.
      if (e instanceof ApiError && e.body.code === "offer_merchant_live") {
        setAskReplace(true);
        return;
      }
      // **والخطأُ داخل النافذة** — كان تحتها فلا يُرى.
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.create}>
      <Confirm
        open={askReplace}
        title={O.replaceTitle}
        body={O.replaceBody}
        confirmLabel={O.replaceConfirm}
        tone="primary"
        busy={busy}
        onConfirm={() => void submit(true)}
        onCancel={() => setAskReplace(false)}
      />
      <div className="space-y-3">
        <Input id="offer-title" label={P.fTitle} value={title} onChange={(e) => setTitle(e.target.value)} />
        <Input id="offer-body" label={P.fBody} value={body} onChange={(e) => setBody(e.target.value)} />
        <Input
          id="offer-item"
          label={P.fItem}
          value={itemQuery}
          onChange={(e) => {
            setItemQuery(e.target.value);
            setItemID("");
          }}
        />
        {items.length > 0 && !itemID && (
          <ul className="max-h-40 space-y-1 overflow-y-auto">
            {items.map((it) => (
              <li key={it.id}>
                <button
                  type="button"
                  onClick={() => {
                    setItemID(it.id);
                    setItemQuery(`${it.name} — ${it.merchant_name}`);
                  }}
                  className="w-full rounded-control border border-line px-3 py-2 text-start text-sm hover:border-accent"
                >
                  {it.name} — {it.merchant_name}
                </button>
              </li>
            ))}
          </ul>
        )}

        <p className="text-sm font-medium">{O.method}</p>
        <Chips
          items={[
            { id: "percent" as const, label: O.methodPercent },
            { id: "amount" as const, label: O.methodAmount },
          ]}
          value={method}
          onChange={setMethod}
        />
        <Input
          id="offer-value"
          label={method === "percent" ? P.fPercent : O.fAmount}
          type="number"
          min="1"
          max={method === "percent" ? "90" : undefined}
          value={value}
          onChange={(e) => setValue(e.target.value)}
        />
        <Input
          id="offer-ends"
          label={O.endsOn}
          type="date"
          value={endsOn}
          onChange={(e) => setEndsOn(e.target.value)}
        />
        <p className="text-xs text-ink-muted">{O.platformOnly}</p>
        <p className="text-xs text-ink-muted">{O.offerApprovalHint}</p>
        {itemID && (
          <NotifyChoice id="offer-notify" count={audience} checked={notify} onChange={setNotify} />
        )}
        {notice && <Alert tone="info">{notice}</Alert>}
        {error && <Alert>{error}</Alert>}
        <FormActions onSave={() => void submit()} onCancel={onClose} busy={busy} saveLabel={P.publish} />
      </div>
    </Modal>
  );
}

function ActivateModal({
  offer,
  onClose,
  onDone,
}: {
  offer: Offer;
  onClose: () => void;
  onDone: () => void;
}) {
  const [notify, setNotify] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const audience = useAudience(offer.menu_item_id ?? "");

  async function submit() {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/offers/${offer.id}/active`, {
        method: "POST",
        body: JSON.stringify({ active: true, notify }),
      });
      onDone();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={O.reactivateTitle}>
      <div className="space-y-3">
        <p className="font-bold">
          {offer.title} — {offer.item_name}
        </p>
        <NotifyChoice id="offer-renotify" count={audience} checked={notify} onChange={setNotify} />
        {error && <Alert>{error}</Alert>}
        <FormActions onSave={() => void submit()} onCancel={onClose} busy={busy} saveLabel={O.reactivate} />
      </div>
    </Modal>
  );
}
