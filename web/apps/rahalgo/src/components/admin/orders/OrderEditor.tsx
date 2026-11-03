"use client";

/**
 * **«تعديل الطلب» — الطلبُ العاديّ لحتّى التسليم** (قرارُ المالك ٢٠٢٦-١١-٠٣)
 *
 * كمّيّةٌ تزيد أو تنقص، وصنفٌ يُحذف، وصنفٌ يُضاف من متجرٍ في الطلب، وسعرُ متجرٍ تغيّر —
 * **بسببٍ إجباريّ.** خطوتان: يُملأ التعديل، ثمّ **معاينةٌ يحسبها الخادمُ كلَّها**
 * (`dry_run`) فيُقرأ المجموعُ الجديدُ وما يُردّ أو يُقبض قبل أن يقع. **والحسبةُ كلُّها
 * في المحرّك** (`orders/order_edit.go`) — الشاشةُ تسأل وتُظهر ما ردّه.
 */
import { useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Checkbox,
  FormActions,
  IconAdd,
  IconDelete,
  IconMinus,
  Input,
  LoadingState,
  Modal,
  Money,
  Select,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const E = getMessages(defaultLocale).admin.ordersPage.orderEdit;

/** **بابا المحرّك** — يملكهما من يملك التدخّل، وبعد الاستلام قدرةٌ أضيق. */
export const ORDER_EDIT_ROUTE = ["POST", "/orders/{id}/edit"] as const;

interface CatalogLine {
  id: string;
  name: string;
  qty: number;
  unit_price: number;
  merchant_price: number;
  merchant_id: string;
  drink_for?: string | null;
}
interface CatalogGroup {
  id: string;
  name: string;
  min: number;
  max: number;
  options: { id: string; name: string; price_delta: number }[];
}
interface CatalogItem {
  id: string;
  name: string;
  price: number;
  groups: CatalogGroup[];
}
interface CatalogStore {
  merchant_id: string;
  name: string;
  items: CatalogItem[];
}
interface EditChange {
  name: string;
  old_qty: number;
  new_qty: number;
  old_price: number;
  new_price: number;
  /** **ما رجّعه الزبونُ والمنصّةُ تتحمّله** — يبقى للمتجر حقُّه. */
  returned?: number;
}
interface StoreDelta {
  merchant_id: string;
  name: string;
  method: string;
  old_due: number;
  new_due: number;
  driver_pays: number;
  debt: number;
}
export interface EditResult {
  after_pickup: boolean;
  old_total: number;
  new_total: number;
  refunded: number;
  charged: number;
  old_cash_due: number;
  cash_due: number;
  changes: EditChange[];
  stores?: StoreDelta[];
  reason: string;
}
interface AddRow {
  key: number;
  menu_item_id: string;
  name: string;
  price: number;
  qty: number;
  option_ids: string[];
}

export function OrderEditor({
  orderId,
  orderNumber,
  afterPickup,
  onClose,
  onDone,
}: {
  orderId: string;
  orderNumber: number;
  /** **بعد الاستلام** — يُسأل مَن يتحمّل النقص (قرارُ المالك، البند ٢). */
  afterPickup: boolean;
  onClose: () => void;
  onDone: (res: EditResult) => void;
}) {
  const [lines, setLines] = useState<CatalogLine[] | null>(null);
  const [stores, setStores] = useState<CatalogStore[]>([]);
  const [qty, setQty] = useState<Record<string, number>>({});
  const [price, setPrice] = useState<Record<string, string>>({});
  const [adds, setAdds] = useState<AddRow[]>([]);
  const [pick, setPick] = useState("");
  const [pickOpts, setPickOpts] = useState<Record<string, string>>({});
  const [reason, setReason] = useState("");
  const [storeRefunded, setStoreRefunded] = useState(false);
  const [review, setReview] = useState<EditResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  useEffect(() => {
    let live = true;
    api<{ lines: CatalogLine[]; stores: CatalogStore[] }>(
      `/api/v1/admin/orders/${orderId}/edit-catalog`,
    )
      .then((c) => {
        if (!live) return;
        setLines(c.lines);
        setStores(c.stores);
        setQty(Object.fromEntries(c.lines.map((l) => [l.id, l.qty])));
        setPrice(
          Object.fromEntries(
            c.lines.map((l) => [l.id, String(l.merchant_price)]),
          ),
        );
      })
      .catch((e) => live && setErr(errorText(e)));
    return () => {
      live = false;
    };
  }, [orderId]);

  const items = useMemo(
    () =>
      stores.flatMap((s) =>
        s.items.map((it) => ({
          ...it,
          store: s.name,
          multi: stores.length > 1,
        })),
      ),
    [stores],
  );
  const picked = items.find((it) => it.id === pick);

  function body(dryRun: boolean) {
    const changed = (lines ?? [])
      .map((l) => {
        const q = qty[l.id] ?? l.qty;
        const p = Math.round(Number(price[l.id]));
        const priceChanged =
          Number.isFinite(p) && p >= 0 && p !== l.merchant_price;
        if (q === l.qty && !priceChanged) return null;
        return {
          item_id: l.id,
          qty: q,
          ...(priceChanged ? { merchant_price: p } : {}),
        };
      })
      .filter(Boolean);
    return JSON.stringify({
      reason: reason.trim(),
      lines: changed,
      add: adds.map((a) => ({
        menu_item_id: a.menu_item_id,
        qty: a.qty,
        option_ids: a.option_ids,
      })),
      store_refunded: afterPickup && storeRefunded,
      dry_run: dryRun,
    });
  }

  async function send(dryRun: boolean) {
    setBusy(true);
    setErr("");
    try {
      const r = await api<{ result: EditResult }>(
        `/api/v1/admin/orders/${orderId}/edit`,
        {
          method: "POST",
          body: body(dryRun),
        },
      );
      if (dryRun) setReview(r.result);
      else onDone(r.result);
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  function addPicked() {
    if (!picked) return;
    const groups = picked.groups;
    if (groups.some((g) => !pickOpts[g.id])) {
      setErr(E.chooseOption);
      return;
    }
    const extra = groups.reduce(
      (sum, g) =>
        sum +
        (g.options.find((op) => op.id === pickOpts[g.id])?.price_delta ?? 0),
      0,
    );
    const optNames = groups
      .map((g) => g.options.find((op) => op.id === pickOpts[g.id])?.name)
      .filter(Boolean)
      .join(E.optionSep);
    setAdds((a) => [
      ...a,
      {
        key: Date.now(),
        menu_item_id: picked.id,
        name: optNames ? `${picked.name} (${optNames})` : picked.name,
        price: picked.price + extra,
        qty: 1,
        option_ids: groups.map((g) => pickOpts[g.id] ?? ""),
      },
    ]);
    setPick("");
    setPickOpts({});
    setErr("");
  }

  const stepper = (value: number, set: (n: number) => void, min: number) => (
    <span className="flex shrink-0 items-center gap-1">
      <Button
        variant="secondary"
        size="sm"
        aria-label={E.less}
        disabled={value <= min}
        onClick={() => set(value - 1)}
      >
        <IconMinus size={14} />
      </Button>
      <span className="w-6 text-center text-sm font-bold">{fmtNum(value)}</span>
      <Button
        variant="secondary"
        size="sm"
        aria-label={E.more}
        disabled={value >= 50}
        onClick={() => set(value + 1)}
      >
        <IconAdd size={14} />
      </Button>
    </span>
  );

  // ── المراجعة: ما سيقع بالضبط، كما حسبه الخادم ─────────────────────────
  if (review) {
    const row = "flex items-center justify-between gap-2";
    return (
      <Modal
        open
        onClose={onClose}
        title={E.reviewTitle.replace("{n}", String(orderNumber))}
      >
        <div className="space-y-3 text-sm">
          <ul className="space-y-1">
            {review.changes.map((c, i) => (
              <li key={i} className={row}>
                <span className="min-w-0 flex-1">{c.name}</span>
                <span className="shrink-0 text-ink-muted">
                  {c.returned
                    ? `${E.returnedLabel} ×${fmtNum(c.returned)}`
                    : c.new_qty === 0
                      ? E.removed
                      : c.old_qty === 0
                        ? `${E.added} ×${fmtNum(c.new_qty)}`
                        : c.old_qty !== c.new_qty
                          ? `×${fmtNum(c.old_qty)} ← ×${fmtNum(c.new_qty)}`
                          : E.priceChanged}
                </span>
              </li>
            ))}
          </ul>
          <div className={`${row} border-t border-line-soft pt-2 font-bold`}>
            <span>{E.total}</span>
            <span className="flex items-center gap-1.5">
              <Money value={review.old_total} small />
              <span aria-hidden>←</span>
              <Money value={review.new_total} small />
            </span>
          </div>
          {review.refunded > 0 && (
            <div className={row}>
              <span className="text-ink-muted">{E.refunded}</span>
              <Money value={review.refunded} small />
            </div>
          )}
          {review.charged > 0 && (
            <div className={row}>
              <span className="text-ink-muted">{E.charged}</span>
              <Money value={review.charged} small />
            </div>
          )}
          <div className={row}>
            <span className="text-ink-muted">{E.cashDue}</span>
            <span className="flex items-center gap-1.5">
              <Money value={review.old_cash_due} small />
              <span aria-hidden>←</span>
              <Money value={review.cash_due} small />
            </span>
          </div>
          {(review.stores ?? []).map((s) => (
            <div
              key={s.merchant_id}
              className="space-y-1 rounded-control bg-field px-3 py-2"
            >
              <div className={row}>
                <span className="text-ink-muted">
                  {E.storeDue.replace("{name}", s.name)}
                </span>
                <span className="flex items-center gap-1.5">
                  <Money value={s.old_due} small />
                  <span aria-hidden>←</span>
                  <Money value={s.new_due} small />
                </span>
              </div>
              {s.driver_pays > 0 && (
                <div className={row}>
                  <span className="text-ink-muted">{E.driverPays}</span>
                  <Money value={s.driver_pays} small />
                </div>
              )}
              {s.driver_pays < 0 && (
                <div className={row}>
                  <span className="text-ink-muted">{E.driverTakes}</span>
                  <Money value={-s.driver_pays} small />
                </div>
              )}
              {s.debt > 0 && (
                <div className={row}>
                  <span className="text-ink-muted">{E.debt}</span>
                  <Money value={s.debt} small />
                </div>
              )}
            </div>
          ))}
          <div className="flex items-start justify-between gap-2">
            <span className="shrink-0 text-ink-muted">{E.reason}</span>
            <span className="text-end">{review.reason}</span>
          </div>
          {err && <Alert tone="error">{err}</Alert>}
          <FormActions
            busy={busy}
            onSave={() => void send(false)}
            onCancel={() => setReview(null)}
            saveLabel={E.confirm}
            cancelLabel={E.back}
          />
        </div>
      </Modal>
    );
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={E.title.replace("{n}", String(orderNumber))}
      size="lg"
    >
      {lines == null ? (
        err ? (
          <Alert tone="error">{err}</Alert>
        ) : (
          <LoadingState />
        )
      ) : (
        <div className="space-y-4 text-sm">
          <ul className="space-y-2">
            {lines.map((l) => {
              const q = qty[l.id] ?? l.qty;
              return (
                <li
                  key={l.id}
                  className={`flex flex-wrap items-center gap-2 ${q === 0 ? "opacity-50" : ""}`}
                >
                  <span
                    className={`min-w-0 flex-1 font-medium ${q === 0 ? "line-through" : ""}`}
                  >
                    {l.name}
                  </span>
                  {q > 0 &&
                    stepper(q, (n) => setQty((s) => ({ ...s, [l.id]: n })), 1)}
                  <label className="flex shrink-0 items-center gap-1 text-xs text-ink-muted">
                    {E.storePrice}
                    <Input
                      type="number"
                      inputMode="numeric"
                      className="w-24"
                      value={price[l.id] ?? ""}
                      disabled={q === 0}
                      onChange={(e) =>
                        setPrice((s) => ({ ...s, [l.id]: e.target.value }))
                      }
                    />
                  </label>
                  <Button
                    variant={q === 0 ? "secondary" : "danger-outline"}
                    size="sm"
                    onClick={() =>
                      setQty((s) => ({ ...s, [l.id]: q === 0 ? l.qty : 0 }))
                    }
                  >
                    {q === 0 ? E.undo : <IconDelete size={14} />}
                  </Button>
                </li>
              );
            })}
            {adds.map((a) => (
              <li key={a.key} className="flex flex-wrap items-center gap-2">
                <span className="min-w-0 flex-1 font-medium text-success">
                  {a.name}
                </span>
                {stepper(
                  a.qty,
                  (n) =>
                    setAdds((s) =>
                      s.map((x) => (x.key === a.key ? { ...x, qty: n } : x)),
                    ),
                  1,
                )}
                <Money value={a.price * a.qty} small className="shrink-0" />
                <Button
                  variant="danger-outline"
                  size="sm"
                  aria-label={E.remove}
                  onClick={() =>
                    setAdds((s) => s.filter((x) => x.key !== a.key))
                  }
                >
                  <IconDelete size={14} />
                </Button>
              </li>
            ))}
          </ul>

          <div className="space-y-2 rounded-control bg-field p-3">
            <p className="text-xs font-medium">{E.add}</p>
            <Select
              value={pick}
              onChange={(e) => {
                setPick(e.target.value);
                setPickOpts({});
              }}
            >
              <option value="">{E.pickItem}</option>
              {items.map((it) => (
                <option key={it.id} value={it.id}>
                  {it.multi ? `${it.store} — ` : ""}
                  {it.name} · {fmtNum(it.price)}
                </option>
              ))}
            </Select>
            {picked?.groups.map((g) => (
              <Select
                key={g.id}
                label={g.name}
                value={pickOpts[g.id] ?? ""}
                onChange={(e) =>
                  setPickOpts((s) => ({ ...s, [g.id]: e.target.value }))
                }
              >
                <option value="">{E.pickOption}</option>
                {g.options.map((op) => (
                  <option key={op.id} value={op.id}>
                    {op.name}
                    {op.price_delta ? ` +${fmtNum(op.price_delta)}` : ""}
                  </option>
                ))}
              </Select>
            ))}
            <Button
              variant="secondary"
              size="sm"
              disabled={!picked}
              onClick={addPicked}
            >
              {E.addButton}
            </Button>
          </div>

          {afterPickup && (
            <Checkbox
              id="order-edit-store-refunded"
              label={E.storeRefunded}
              checked={storeRefunded}
              onChange={(e) => setStoreRefunded(e.target.checked)}
            />
          )}
          <Input
            placeholder={E.reason}
            value={reason}
            onChange={(e) => {
              setReason(e.target.value);
              setErr("");
            }}
          />
          {err && <Alert tone="error">{err}</Alert>}
          <FormActions
            busy={busy || reason.trim() === ""}
            onSave={() => void send(true)}
            onCancel={onClose}
            saveLabel={E.next}
          />
        </div>
      )}
    </Modal>
  );
}
