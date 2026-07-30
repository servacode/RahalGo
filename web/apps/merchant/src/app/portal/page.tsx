"use client";

/**
 * لوحة طلبات المتجر: الجديدة ترنّ باستمرار حتى القبول/الرفض،
 * والباقي أعمدة متابعة حية عبر WebSocket (والتحديث الدوري احتياط).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Modal,
  Input,
  IconOrder,
  IconLocation,
  IconSuccess,
  IconWarning,
  IconDriver,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useLiveEvents } from "@/lib/ws";
import { useStore } from "@/lib/store";
import { useRinger } from "@/lib/ringer";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");
const STATUS_LABELS: Record<string, string> = m.orders.status;

interface OrderItem {
  name: string;
  qty: number;
  unit_price: number;
  note: string;
  options: { group: string; name: string; price_delta: number }[];
}

interface Order {
  id: string;
  number: number;
  status: string;
  address_text: string;
  payment_method: string;
  subtotal: number;
  total: number;
  notes: string;
  created_at: string;
  items?: OrderItem[];
}

interface OrderPage {
  orders: Order[];
  total: number;
}

function errText(err: unknown): string {
  if (err instanceof ApiError) {
    const key = err.body.message_key.split(".").pop() ?? "";
    const known = (m.errors as Record<string, string>)[key];
    if (known) return known;
  }
  return m.errors.internal;
}

const WITH_DRIVER = ["dispatching", "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"];

export default function OrdersBoard() {
  const { store } = useStore();
  const [orders, setOrders] = useState<Order[]>([]);
  const [doneToday, setDoneToday] = useState(0);
  const [accepting, setAccepting] = useState<Order | null>(null);
  const [rejecting, setRejecting] = useState<Order | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (!store) return;
    try {
      const open = await api<OrderPage>(
        `/api/v1/merchant/stores/${store.id}/orders?open_only=true&per_page=100`
      );
      setOrders(open.orders);
      const delivered = await api<OrderPage>(
        `/api/v1/merchant/stores/${store.id}/orders?status=delivered&per_page=1`
      );
      setDoneToday(delivered.total);
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [store]);

  useEffect(() => {
    void load();
    const t = setInterval(load, 60000);
    return () => clearInterval(t);
  }, [load]);

  const connected = useLiveEvents(
    useCallback(
      (event) => {
        if (event.type === "order") void load();
      },
      [load]
    )
  );

  const pending = orders.filter((o) => o.status === "pending");
  const accepted = orders.filter((o) => o.status === "accepted");
  const preparing = orders.filter((o) => o.status === "preparing");
  const withDriver = orders.filter((o) => WITH_DRIVER.includes(o.status));

  const { enabled: soundOn, setEnabled: setSoundOn } = useRinger(pending.length > 0);

  async function startPreparing(o: Order) {
    try {
      await api(`/api/v1/merchant/orders/${o.id}/transition`, {
        method: "POST",
        body: JSON.stringify({ to: "preparing", note: "" }),
      });
      await load();
    } catch (err) {
      setError(errText(err));
    }
  }

  return (
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <h1 className="flex items-center gap-2 text-xl font-bold">
            <IconOrder className="text-primary" />
            {m.merchant.nav.orders}
          </h1>
          <Badge variant={connected ? "success" : "danger"}>
            {connected ? m.admin.ordersPage.live : m.admin.ordersPage.liveOff}
          </Badge>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-ink-muted">
            {m.merchant.orders.doneToday}: <b className="text-success">{fmt.format(doneToday)}</b>
          </span>
          <button
            onClick={() => setSoundOn(!soundOn)}
            className={`rounded-control border px-3 py-1.5 text-sm transition-colors ${
              soundOn
                ? "border-primary bg-primary-light text-primary-dark"
                : "border-line text-ink-muted"
            }`}
          >
            {soundOn ? `🔔 ${m.merchant.header.soundOn}` : `🔕 ${m.merchant.header.soundOff}`}
          </button>
        </div>
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      {/* الطلبات الجديدة — شريط بارز نابض مع الرنين */}
      <section
        className={`mb-6 rounded-card border-2 p-4 ${
          pending.length > 0 ? "animate-pulse border-danger bg-danger/5" : "border-line bg-surface"
        }`}
      >
        <h2 className="mb-3 flex items-center gap-2 font-bold">
          <IconWarning size={18} className={pending.length ? "text-danger" : "text-ink-muted"} />
          {m.merchant.orders.newOrders} ({fmt.format(pending.length)})
        </h2>
        {pending.length === 0 ? (
          <p className="text-sm text-ink-muted">{m.merchant.orders.empty}</p>
        ) : (
          <div className="grid gap-3 md:grid-cols-2">
            {pending.map((o) => (
              <OrderCard key={o.id} order={o} highlight>
                <Button onClick={() => setAccepting(o)} className="flex-1">
                  {m.merchant.orders.accept}
                </Button>
                <Button variant="danger" onClick={() => setRejecting(o)}>
                  {m.merchant.orders.reject}
                </Button>
              </OrderCard>
            ))}
          </div>
        )}
      </section>

      <div className="grid gap-4 lg:grid-cols-3">
        <Column title={m.merchant.orders.accepted} icon={<IconSuccess size={16} />}>
          {accepted.map((o) => (
            <OrderCard key={o.id} order={o}>
              <Button onClick={() => startPreparing(o)} className="flex-1">
                {m.merchant.orders.startPreparing}
              </Button>
            </OrderCard>
          ))}
          {accepted.length === 0 && <Empty />}
        </Column>

        <Column title={m.merchant.orders.preparing} icon={<IconOrder size={16} />}>
          {preparing.map((o) => (
            <OrderCard key={o.id} order={o}>
              <span className="text-xs text-ink-muted">{m.merchant.orders.waitingDriver}</span>
            </OrderCard>
          ))}
          {preparing.length === 0 && <Empty />}
        </Column>

        <Column title={m.merchant.orders.withDriver} icon={<IconDriver size={16} />}>
          {withDriver.map((o) => (
            <OrderCard key={o.id} order={o}>
              <Badge variant="primary">{STATUS_LABELS[o.status] ?? o.status}</Badge>
            </OrderCard>
          ))}
          {withDriver.length === 0 && <Empty />}
        </Column>
      </div>

      {accepting && (
        <AcceptModal
          order={accepting}
          onClose={() => setAccepting(null)}
          onDone={() => {
            setAccepting(null);
            void load();
          }}
        />
      )}
      {rejecting && (
        <RejectModal
          order={rejecting}
          onClose={() => setRejecting(null)}
          onDone={() => {
            setRejecting(null);
            void load();
          }}
        />
      )}
    </div>
  );
}

function Column({
  title,
  icon,
  children,
}: {
  title: string;
  icon: React.ReactNode;
  children: React.ReactNode;
}) {
  return (
    <section className="rounded-card border border-line bg-surface p-3">
      <h2 className="mb-3 flex items-center gap-1.5 text-sm font-bold text-ink-muted">
        {icon}
        {title}
      </h2>
      <div className="space-y-3">{children}</div>
    </section>
  );
}

function Empty() {
  return <p className="py-4 text-center text-xs text-ink-muted">{m.merchant.orders.empty}</p>;
}

function OrderCard({
  order,
  highlight,
  children,
}: {
  order: Order;
  highlight?: boolean;
  children?: React.ReactNode;
}) {
  const [expanded, setExpanded] = useState(!!highlight);
  const [detail, setDetail] = useState<Order | null>(null);

  useEffect(() => {
    if (!expanded || detail) return;
    api<Order>(`/api/v1/merchant/orders/${order.id}`)
      .then(setDetail)
      .catch(() => undefined);
  }, [expanded, detail, order.id]);

  const items = detail?.items ?? [];

  return (
    <div
      className={`rounded-control border bg-surface p-3 ${
        highlight ? "border-danger/40 shadow-sm" : "border-line"
      }`}
    >
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center justify-between text-start"
      >
        <span className="font-bold">#{fmt.format(order.number)}</span>
        <span className="text-sm font-bold text-primary-dark">
          {fmt.format(order.subtotal)} {m.common.currency}
        </span>
      </button>
      <p className="mt-1 flex items-center gap-1 text-xs text-ink-muted">
        <IconLocation size={12} className="shrink-0" />
        <span className="truncate">{order.address_text}</span>
        <span className="ms-auto shrink-0" dir="ltr">
          {new Date(order.created_at).toLocaleTimeString("ar-SY", {
            hour: "2-digit",
            minute: "2-digit",
          })}
        </span>
      </p>

      {expanded && (
        <div className="mt-2 border-t border-line pt-2">
          {items.length === 0 ? (
            <p className="text-xs text-ink-muted">{m.common.loading}</p>
          ) : (
            <ul className="space-y-1 text-sm">
              {items.map((it, i) => (
                <li key={i}>
                  <span className="font-medium">
                    {it.name} ×{fmt.format(it.qty)}
                  </span>
                  {it.options.length > 0 && (
                    <span className="text-xs text-ink-muted">
                      {" "}
                      — {it.options.map((op) => op.name).join("، ")}
                    </span>
                  )}
                  {it.note && <p className="text-xs text-accent-dark">✎ {it.note}</p>}
                </li>
              ))}
            </ul>
          )}
          {order.notes && (
            <p className="mt-2 rounded-control bg-page px-2 py-1 text-xs">
              <span className="text-ink-muted">{m.merchant.orders.customerNote}:</span> {order.notes}
            </p>
          )}
        </div>
      )}

      {children && <div className="mt-3 flex items-center gap-2">{children}</div>}
    </div>
  );
}

function AcceptModal({
  order,
  onClose,
  onDone,
}: {
  order: Order;
  onClose: () => void;
  onDone: () => void;
}) {
  const [minutes, setMinutes] = useState("20");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/merchant/orders/${order.id}/transition`, {
        method: "POST",
        body: JSON.stringify({
          to: "accepted",
          note: m.merchant.orders.prepNote.replace("{m}", minutes || "20"),
        }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={m.merchant.orders.prepTitle.replace("{number}", String(order.number))}
    >
      <form onSubmit={submit} className="space-y-4">
        <Input
          id="prep"
          label={m.merchant.orders.prepMinutes}
          type="number"
          min="5"
          max="120"
          dir="ltr"
          required
          autoFocus
          value={minutes}
          onChange={(e) => setMinutes(e.target.value)}
          className="text-center text-lg font-bold"
        />
        <div className="flex flex-wrap gap-2">
          {["10", "15", "20", "30", "45"].map((v) => (
            <button
              key={v}
              type="button"
              onClick={() => setMinutes(v)}
              className={`rounded-control border px-3 py-1 text-sm ${
                minutes === v
                  ? "border-primary bg-primary-light font-bold text-primary-dark"
                  : "border-line text-ink-muted"
              }`}
            >
              {v}
            </button>
          ))}
        </div>
        {error && (
          <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {m.merchant.orders.confirmAccept}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

function RejectModal({
  order,
  onClose,
  onDone,
}: {
  order: Order;
  onClose: () => void;
  onDone: () => void;
}) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/merchant/orders/${order.id}/transition`, {
        method: "POST",
        body: JSON.stringify({ to: "rejected", note: reason }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      title={m.merchant.orders.rejectTitle.replace("{number}", String(order.number))}
    >
      <form onSubmit={submit} className="space-y-4">
        <Input
          id="reason"
          label={m.merchant.orders.rejectReason}
          required
          autoFocus
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && (
          <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" variant="danger" disabled={busy}>
            {m.merchant.orders.confirmReject}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
