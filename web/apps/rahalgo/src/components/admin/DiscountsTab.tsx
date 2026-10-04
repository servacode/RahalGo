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
  DataView,
  ViewToggle,
  useViewMode,
  Pagination,
  type DataColumn,
  IconStore,
  IconStatus,
  Money,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

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
  const [view, setView] = useViewMode("offers");
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

  const columns: DataColumn<Offer>[] = [
    {
      id: "title",
      header: P.fTitle,
      primary: true,
      cell: (o) => <span className="font-bold">{o.title}</span>,
    },
    {
      id: "item",
      header: O.itemCol,
      icon: <IconStore />,
      cell: (o) => (
        <span>
          {o.item_name} — {o.merchant_name}
          {!o.item_available && (
            <Badge variant="warning" className="ms-1">
              {O.itemUnavailable}
            </Badge>
          )}
        </span>
      ),
    },
    {
      id: "before",
      header: O.priceBefore,
      cell: (o) => (
        <span className="text-ink-muted line-through">
          <Money value={o.price_before} />
        </span>
      ),
    },
    {
      id: "after",
      header: O.priceAfter,
      cell: (o) => (
        <span className="font-bold text-success">
          <Money value={o.price_after} />
        </span>
      ),
    },
    { id: "cut", header: O.discountCol, cell: cutText },
    {
      id: "borne",
      header: O.borneCol,
      cell: (o) => (o.borne_by === "platform" ? P.byPlatform : P.byMerchant),
    },
    {
      id: "by",
      header: O.createdBy,
      cell: (o) =>
        `${BY[o.created_by_kind ?? "admin"] ?? ""}${o.created_by_name ? ` · ${o.created_by_name}` : ""}`,
    },
    {
      id: "status",
      header: O.statusCol,
      icon: <IconStatus />,
      cell: (o) => {
        const s = STATUS[statusOf(o)];
        return <Badge variant={s.tone}>{s.label}</Badge>;
      },
    },
    {
      id: "ends",
      header: O.endsCol,
      cell: (o) => (o.ends_at ? fmtDate(o.ends_at) : P.noEnd),
    },
  ];

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <Button onClick={() => setOpen(true)}>{P.create}</Button>
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>
      <Input
        id="offer-search"
        label={O.search}
        placeholder={O.searchDiscounts}
        value={q}
        onChange={(e) => {
          setQ(e.target.value);
          setPage(1);
        }}
      />
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

      {error && <Alert>{error}</Alert>}

      <DataView
        items={shown}
        loading={rows === null}
        getKey={(o) => o.id}
        columns={columns}
        view={view}
        empty={P.empty}
        actions={(o) => {
          const s = statusOf(o);
          if (s === "pending" || s === "rejected" || s === "expired") return null;
          return o.active ? (
            <Button
              variant="secondary"
              disabled={busyId === o.id}
              onClick={() => void deactivate(o)}
            >
              {busyId === o.id ? O.busy : P.deactivate}
            </Button>
          ) : (
            <Button variant="secondary" onClick={() => setActivating(o)}>
              {P.activate}
            </Button>
          );
        }}
      />
      <Pagination page={page} total={filtered.length} perPage={PER_PAGE} onChange={setPage} />

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

  async function submit() {
    // **والضغطُ الثاني لا يُرسل ثانيةً.**
    if (busy) return;
    setBusy(true);
    setError("");
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
        }),
      });
      if (o.approval_state === "pending") {
        setNotice(O.pendingSaved);
        setTimeout(onSaved, 1500);
        return;
      }
      onSaved();
    } catch (e) {
      // **والخطأُ داخل النافذة** — كان تحتها فلا يُرى.
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.create}>
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
