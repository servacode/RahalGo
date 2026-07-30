"use client";

/** تتبع الطلب الحي: خط تقدم بالحالات، تحديث لحظي عبر WebSocket، والتقييم بعد التسليم. */

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge, Button, IconStar, IconSuccess } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useLiveEvents } from "@/lib/ws";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");
const STATUS_LABELS: Record<string, string> = m.orders.status;

// مسار التقدم الطبيعي المعروض للزبون
const FLOW = ["pending", "accepted", "preparing", "assigned", "on_the_way", "delivered"];
// تطبيع الحالات الوسيطة لخط التقدم
const NORMALIZE: Record<string, string> = {
  dispatching: "preparing",
  at_pickup: "assigned",
  picked_up: "assigned",
  at_dropoff: "on_the_way",
};

interface Rating {
  merchant_stars: number;
  driver_stars: number | null;
  comment: string;
}
interface Order {
  id: string;
  number: number;
  merchant_name: string;
  driver_phone: string | null;
  status: string;
  total: number;
  wallet_paid: number;
  cash_due: number;
  address_text: string;
  cancel_reason: string;
  rating?: Rating;
  items?: { name: string; qty: number; unit_price: number }[];
}

export default function OrderTrackingPage() {
  const { id } = useParams<{ id: string }>();
  const [order, setOrder] = useState<Order | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api<Order>(`/api/v1/my/orders/${id}`)
      .then(setOrder)
      .catch(() => setError(m.errors.not_found));
  }, [id]);

  useEffect(() => {
    load();
    const t = setInterval(load, 30000);
    return () => clearInterval(t);
  }, [load]);

  useLiveEvents(
    useCallback(
      (event) => {
        const o = event.order as { id?: string } | undefined;
        if (event.type === "order" && o?.id === id) load();
      },
      [id, load]
    )
  );

  if (error) return <p className="py-10 text-center text-ink-muted">{error}</p>;
  if (!order) return <p className="py-10 text-center text-ink-muted">{m.common.loading}</p>;

  const norm = NORMALIZE[order.status] ?? order.status;
  const stepIdx = FLOW.indexOf(norm);
  const failed = ["rejected", "cancelled", "failed", "refunded"].includes(order.status);

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-5 flex flex-wrap items-center justify-between gap-2">
        <h1 className="text-xl font-bold">
          {m.site.orders.orderTitle.replace("{n}", fmt.format(order.number))}
        </h1>
        <Badge variant={failed ? "danger" : order.status === "delivered" ? "success" : "primary"}>
          {STATUS_LABELS[order.status] ?? order.status}
        </Badge>
      </div>

      {/* خط التقدم الحي */}
      {!failed && (
        <ol className="mb-6 space-y-0">
          {FLOW.map((st, i) => {
            const done = stepIdx >= i;
            const current = stepIdx === i && order.status !== "delivered";
            return (
              <li key={st} className="flex gap-3">
                <div className="flex flex-col items-center">
                  <span
                    className={`flex h-7 w-7 items-center justify-center rounded-badge text-xs font-bold ${
                      done ? "bg-primary text-white" : "border border-line text-ink-muted"
                    } ${current ? "animate-pulse" : ""}`}
                  >
                    {done && !current ? "✓" : i + 1}
                  </span>
                  {i < FLOW.length - 1 && (
                    <span className={`h-6 w-0.5 ${stepIdx > i ? "bg-primary" : "bg-line"}`} />
                  )}
                </div>
                <span
                  className={`pt-1 text-sm ${done ? "font-medium" : "text-ink-muted"} ${
                    current ? "text-primary-dark" : ""
                  }`}
                >
                  {STATUS_LABELS[st]}
                </span>
              </li>
            );
          })}
        </ol>
      )}
      {failed && order.cancel_reason && (
        <p className="mb-6 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">
          {order.cancel_reason}
        </p>
      )}

      <section className="mb-5 rounded-card border border-line bg-surface p-4">
        <p className="mb-2 font-bold">{order.merchant_name}</p>
        <ul className="space-y-1 text-sm">
          {order.items?.map((it, i) => (
            <li key={i} className="flex justify-between">
              <span>
                {it.name} ×{fmt.format(it.qty)}
              </span>
              <span className="text-ink-muted">{fmt.format(it.unit_price * it.qty)}</span>
            </li>
          ))}
        </ul>
        <div className="mt-2 flex justify-between border-t border-line pt-2 font-bold">
          <span>{m.site.cart.total}</span>
          <span className="text-primary-dark">
            {fmt.format(order.total)} {m.common.currency}
          </span>
        </div>
        {order.wallet_paid > 0 && (
          <p className="mt-1 text-xs text-ink-muted">
            {m.orders.payment.wallet}: {fmt.format(order.wallet_paid)} — {m.orders.payment.cash}:{" "}
            {fmt.format(order.cash_due)}
          </p>
        )}
        <p className="mt-2 text-xs text-ink-muted">📍 {order.address_text}</p>
      </section>

      {order.status === "delivered" &&
        (order.rating ? (
          <p className="flex items-center gap-2 rounded-card border border-line bg-surface p-4 text-sm text-success">
            <IconSuccess size={18} />
            {m.site.orders.rated}
          </p>
        ) : (
          <RatingForm orderID={order.id} hasDriver={!!order.driver_phone} onRated={load} />
        ))}
    </div>
  );
}

function Stars({ value, onChange }: { value: number; onChange: (v: number) => void }) {
  return (
    <div className="flex gap-1" dir="ltr">
      {[1, 2, 3, 4, 5].map((i) => (
        <button key={i} type="button" onClick={() => onChange(i)} aria-label={String(i)}>
          <IconStar
            size={26}
            className={i <= value ? "fill-accent text-accent" : "text-line"}
          />
        </button>
      ))}
    </div>
  );
}

function RatingForm({
  orderID,
  hasDriver,
  onRated,
}: {
  orderID: string;
  hasDriver: boolean;
  onRated: () => void;
}) {
  const [merchantStars, setMerchantStars] = useState(0);
  const [driverStars, setDriverStars] = useState(0);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/orders/${orderID}/rating`, {
        method: "POST",
        body: JSON.stringify({
          merchant_stars: merchantStars,
          driver_stars: hasDriver && driverStars > 0 ? driverStars : null,
          comment,
        }),
      });
      onRated();
    } catch (err) {
      setError(err instanceof ApiError ? m.errors.internal : m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <section className="rounded-card border border-line bg-surface p-4">
      <h2 className="mb-3 font-bold">{m.site.orders.rateOrder}</h2>
      <div className="mb-3 space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-sm">{m.site.orders.merchantStars}</span>
          <Stars value={merchantStars} onChange={setMerchantStars} />
        </div>
        {hasDriver && (
          <div className="flex items-center justify-between">
            <span className="text-sm">{m.site.orders.driverStars}</span>
            <Stars value={driverStars} onChange={setDriverStars} />
          </div>
        )}
      </div>
      <input
        value={comment}
        onChange={(e) => setComment(e.target.value)}
        placeholder={m.site.orders.commentPlaceholder}
        className="mb-3 w-full rounded-control border border-line bg-surface px-3 py-2 text-sm outline-none focus:border-primary"
      />
      {error && (
        <p className="mb-3 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}
      <Button onClick={submit} disabled={busy || merchantStars === 0} className="w-full">
        {m.site.orders.submitRating}
      </Button>
    </section>
  );
}
