"use client";

/**
 * لوحة طلبات المتجر: الجديدة ترنّ باستمرار حتى القبول/الرفض،
 * والباقي أعمدة متابعة حية عبر WebSocket (والتحديث الدوري احتياط).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtTime } from "@rahalgo/i18n";
import {
  Alert,
  EmptyState,
  useLiveEvent,
  useLiveStatus,
  Badge,
  Button,
  Modal,
  Input,
  IconOrder,
  IconSuccess,
  IconWarning,
  IconDriver,
  IconBell,
  IconBellOff,
  IconEdit,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useStore } from "@/lib/store";
import { useRinger } from "@/lib/ringer";

const m = getMessages(defaultLocale);
const MO = m.shared.merchantOps;
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
  prep_minutes: number | null;
  ready_at: string | null;
  accepted_at: string | null;
  // **لا عنوان ولا سعر ولا طريقة دفع**: الخادم يحجبها عن المتجر
  // (`merchant_privacy.go`). وإبقاؤها في النوع يُغري ببنائها في شاشةٍ غداً
  // فتُقرأ أصفاراً — **حقلٌ ميّت أخطرُ من حقلٍ غائب**.
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
  const { store, selfManage } = useStore();
  const [orders, setOrders] = useState<Order[]>([]);
  const [doneToday, setDoneToday] = useState(0);
  const [accepting, setAccepting] = useState<Order | null>(null);
  const [cancelling, setCancelling] = useState<Order | null>(null);
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
  }, [load]);

  // البث الحي المركزي — قناة واحدة للتطبيق كله، بلا استطلاع دوري
  useLiveEvent((event) => {
    if (event.type === "order") void load();
  });
  const connected = useLiveStatus();

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
            {connected ? m.shared.live : m.shared.liveOff}
          </Badge>
        </div>
        <div className="flex items-center gap-3">
          <span className="text-sm text-ink-muted">
            {m.merchant.orders.doneToday}: <b className="text-success">{fmtNum(doneToday)}</b>
          </span>
          <button
            onClick={() => setSoundOn(!soundOn)}
            className={`rounded-control border px-3 py-1.5 text-sm transition-colors ${
              soundOn
                ? "border-primary bg-primary-light text-primary-dark"
                : "border-line text-ink-muted"
            }`}
          >
            {soundOn ? <IconBell size={15} /> : <IconBellOff size={15} />}
            {soundOn ? m.merchant.header.soundOn : m.merchant.header.soundOff}
          </button>
        </div>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      {/* الطلبات الجديدة — شريط بارز نابض مع الرنين */}
      <section
        className={`mb-6 rounded-card border-2 p-4 ${
          pending.length > 0 ? "animate-pulse border-danger bg-danger/5" : "border-line bg-surface"
        }`}
      >
        <h2 className="mb-3 flex items-center gap-2 font-bold">
          <IconWarning size={18} className={pending.length ? "text-danger" : "text-ink-muted"} />
          {m.merchant.orders.newOrders} ({fmtNum(pending.length)})
        </h2>
        {pending.length === 0 ? (
          <p className="text-sm text-ink-muted">{m.merchant.orders.empty}</p>
        ) : (
          /*
            **بطاقاتٌ بعرضٍ محدود لا تمتدّ بامتداد الشاشة.**

            كانت `md:grid-cols-2`، فالبطاقةُ الوحيدة تأخذ نصف الشاشة وزرُّ
            القبول فيها يمتدّ ذراعاً كاملة. **وزرٌّ بعرض الشاشة لا يبدو أهمّ،
            يبدو مكسوراً** — والعينُ تقرأ العرضَ الزائد فوضىً لا تأكيداً.

            وأربعةُ أعمدة على الشاشات الواسعة: هذه بطاقاتُ **مطبخ** تُمسح بالعين
            بسرعة، لا صفحاتُ تفصيل.
          */
          <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
            {pending.map((o) => (
              <OrderCard key={o.id} order={o} highlight>
                {/* **حين تُدير المنصةُ الطلبات لا أزرارَ هنا.**

                    والحكمُ في الخادم لا هنا (`orders/modes.go`): من استدعى
                    الواجهةَ البرمجية مباشرةً يُردّ. **وهذه الشاشةُ تعرض
                    السياسةَ ولا تصنعها.** */}
                {selfManage && (
                  <>
                    <Button onClick={() => setAccepting(o)}>
                      {m.merchant.orders.accept}
                    </Button>
                    <Button variant="danger" onClick={() => setRejecting(o)}>
                      {m.merchant.orders.reject}
                    </Button>
                  </>
                )}
              </OrderCard>
            ))}
          </div>
        )}
      </section>

      <div className="grid gap-4 lg:grid-cols-3">
        <Column title={m.merchant.orders.accepted} icon={<IconSuccess size={16} />}>
          {accepted.map((o) => (
            <OrderCard key={o.id} order={o}>
              {selfManage && (
                <>
                  <Button onClick={() => startPreparing(o)}>
                    {m.merchant.orders.startPreparing}
                  </Button>
                  <Button variant="danger" onClick={() => setCancelling(o)}>
                    {MO.cancelOrder}
                  </Button>
                </>
              )}
            </OrderCard>
          ))}
          {accepted.length === 0 && <EmptyState title={m.merchant.orders.empty} />}
        </Column>

        <Column title={m.merchant.orders.preparing} icon={<IconOrder size={16} />}>
          {preparing.map((o) => (
            <OrderCard key={o.id} order={o}>
              {selfManage && (
                <>
                  <ReadyControl order={o} onDone={load} />
                  <Button variant="danger" onClick={() => setCancelling(o)}>
                    {MO.cancelOrder}
                  </Button>
                </>
              )}
            </OrderCard>
          ))}
          {preparing.length === 0 && <EmptyState title={m.merchant.orders.empty} />}
        </Column>

        <Column title={m.merchant.orders.withDriver} icon={<IconDriver size={16} />}>
          {withDriver.map((o) => (
            <OrderCard key={o.id} order={o}>
              <Badge variant="primary">{STATUS_LABELS[o.status] ?? o.status}</Badge>
              {["dispatching", "assigned", "at_pickup"].includes(o.status) && (
                <ReadyControl order={o} onDone={load} />
              )}
            </OrderCard>
          ))}
          {withDriver.length === 0 && <EmptyState title={m.merchant.orders.empty} />}
        </Column>
      </div>

      {cancelling && (
        <CancelModal
          order={cancelling}
          onClose={() => setCancelling(null)}
          onDone={() => {
            setCancelling(null);
            void load();
          }}
        />
      )}

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
    <section className="surface p-3">
      <h2 className="mb-3 flex items-center gap-1.5 text-sm font-bold text-ink-muted">
        {icon}
        {title}
      </h2>
      <div className="space-y-3">{children}</div>
    </section>
  );
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
        highlight ? "border-danger/40 elev-1" : "border-line"
      }`}
    >
      <button
        type="button"
        onClick={() => setExpanded(!expanded)}
        className="flex w-full items-center justify-between text-start"
      >
        {/* **ورقةُ مطبخٍ لا فاتورة.**

            لا سعرَ ولا عنوانَ ولا اسمَ زبون: السائقُ يأتي إلى المتجر ولا يذهب
            المتجرُ إلى أحد، فلا حاجةَ له بالعنوان. وهاتفُ الزبون وعنوانُه في
            يد مطعمٍ يعني أنه يستطيع الاتصال به مباشرةً في الطلب القادم —
            **فتصير المنصةُ دليلَ زبائنَ يُبنى على ظهرها ثم يُستغنى عنها**.

            والمالُ يراه في محفظته وتقاريره، **وهي أدقّ**: تعرض مستحقّه هو لا
            ما دفعه الزبون (وفيه رسمُ توصيلٍ ليس له).

            والحجبُ في الخادم لا هنا (`merchant_privacy.go`) — وهذا عرضُ ما
            وصل، لا إخفاءُ ما وصل. */}
        <span className="font-bold">#{fmtRef(order.number)}</span>
        <span className="text-xs text-ink-muted" dir="ltr">
          {fmtTime(order.created_at)}
        </span>
      </button>

      {expanded && (
        <div className="mt-2 border-t border-line pt-2">
          {items.length === 0 ? (
            <p className="text-xs text-ink-muted">{m.common.loading}</p>
          ) : (
            <ul className="space-y-1 text-sm">
              {items.map((it, i) => (
                <li key={i}>
                  <span className="font-medium">
                    {it.name} ×{fmtNum(it.qty)}
                  </span>
                  {it.options.length > 0 && (
                    <span className="text-xs text-ink-muted">
                      {" "}
                      — {it.options.map((op) => op.name).join(m.common.listSeparator)}
                    </span>
                  )}
                  {it.note && <p className="text-xs text-accent-dark"><IconEdit size={11} className="inline align-[-1px]" /> {it.note}</p>}
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

      {/* **الأزرار تتقاسم السطر بالتساوي.**

          كان زرُّ القبول `flex-1` وزرُّ الرفض بمقاسه الطبيعي، فيبتلع الأوّل كلَّ
          الفراغ ويبدو الثاني ملصقاً به. **وزرٌّ يمتدّ ذراعاً لا يبدو أهمّ، يبدو
          مكسوراً.** والأهميّةُ تُقال باللون لا بالعرض — والقبولُ ممتلئٌ والرفضُ
          خفيف. */}
      {children && (
        <div className="mt-3 flex items-center gap-2 [&>button]:flex-1 [&>button]:!px-3">
          {children}
        </div>
      )}
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
          <Alert>{error}</Alert>
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
          <Alert>{error}</Alert>
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

/**
 * زرّ الجاهزية وعدّاده — الوصلة بين المطبخ والسائق.
 *
 * قبل الإعلان يعرض ما تبقّى من وقت التحضير الذي قاله المتجر نفسه عند القبول،
 * فيقيسه بوعده لا بتقدير غيره. وبعد الإعلان يصير وسماً ثابتاً: الجاهزية لحظة
 * واحدة لا تتكرّر، وإعادة ضبطها تُفسد قياس زمن التحضير الفعلي.
 */
function ReadyControl({ order, onDone }: { order: Order; onDone: () => void }) {
  const [busy, setBusy] = useState(false);

  if (order.ready_at) {
    return (
      <span className="flex flex-1 items-center justify-center gap-1.5 rounded-control bg-success/10 px-3 py-1.5 text-sm font-medium text-success">
        <IconSuccess size={15} />
        {MO.readyDone}
      </span>
    );
  }

  // العدّ من لحظة القبول لا من الإنشاء: الوعد يبدأ حين يلتزم المتجر
  let hint = "";
  if (order.accepted_at && order.prep_minutes) {
    const dueMs = new Date(order.accepted_at).getTime() + order.prep_minutes * 60_000;
    const left = Math.round((dueMs - Date.now()) / 60_000);
    hint =
      left >= 0
        ? MO.remaining.replace("{n}", fmtNum(left))
        : MO.overdue.replace("{n}", fmtNum(-left));
  }

  return (
    <div className="flex flex-1 items-center gap-2">
      <Button
        disabled={busy}
        onClick={async () => {
          setBusy(true);
          try {
            await api(`/api/v1/merchant/orders/${order.id}/ready`, { method: "POST" });
            onDone();
          } finally {
            setBusy(false);
          }
        }}
        className="flex-1"
      >
        {MO.ready}
      </Button>
      {hint && <span className="shrink-0 text-xs text-ink-muted">{hint}</span>}
    </div>
  );
}

/** إلغاء المتجر — بسببٍ إلزامي يصل الزبون والإدارة. */
function CancelModal({
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
        body: JSON.stringify({ to: "cancelled", note: reason }),
      });
      onDone();
    } catch {
      setError(m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={`${MO.cancelOrder} #${fmtRef(order.number)}`}>
      <form onSubmit={submit} className="space-y-4">
        <div>
          <Input
            id="cancel-reason"
            label={MO.cancelReason}
            required
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <p className="mt-1 text-xs text-ink-muted">{MO.cancelHint}</p>
        </div>
        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" variant="danger" disabled={busy}>
            {MO.cancelOrder}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
