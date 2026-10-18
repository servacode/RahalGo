"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/orders — طلباتُ المتجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `OrdersScreen.kt` في تطبيق المتجر.)
 *
 * **آلةُ الحالات في المحرّك لا هنا** — الشاشةُ تعرض الزرَّ التالي بحسب
 * حال الطلب كما يعرضه التطبيق حرفاً:
 *
 *     جديد      ← «استلمت الطلب» (و«اعتذار عن الطلب» بسبب)
 *     استلمته   ← «بدأت التحضير»
 *     يتحضّر   ← «جاهز للاستلام»
 *
 * **وما بعدها للسائق** — يُعرض حالاً لا زرّاً.
 *
 * **وفي وضع «المنصّة تدير»** لا أزرار: المحرّكُ يردّ أيَّ انتقالٍ من
 * المتجر (`rolesUnderMode`)، **وزرٌّ يُضغط فيُرفض أسوأُ من غيابه.**
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum, fmtSpan } from "@rahalgo/i18n";
import { Alert, Button, EmptyState, IconOrder, LoadingState, Modal, ReloadState, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useStoreOrders } from "@/lib/store/orders";
import { canMarkReady, type MerchantOrder } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const O = m.store.orders;
const STATUS = O.status as Record<string, string>;

/** **ثانيةٌ تمشي** — «عم ينتظر من ٣ د» يتبدّل بلا نداء. */
function useNow(ms = 15_000): number {
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), ms);
    return () => clearInterval(t);
  }, [ms]);
  return now;
}

function RejectDialog({ order, onClose, onPick }: { order: MerchantOrder; onClose: () => void; onPick: (r: string) => void }) {
  const [custom, setCustom] = useState(false);
  const [text, setText] = useState("");
  const canned = [O.reasons.unavailable, O.reasons.busy, O.reasons.cantPrep, O.reasons.closing];
  return (
    <Modal open onClose={onClose} title={`${O.rejectTitle} · #${order.number}`}>
      {!custom ? (
        <div className="flex flex-col gap-2">
          {canned.map((r) => (
            <Button key={r} variant="secondary" className="w-full py-3 text-start" onClick={() => onPick(r)}>
              {r}
            </Button>
          ))}
          <Button variant="ghost" className="w-full py-3 text-start" onClick={() => setCustom(true)}>
            {O.other}
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-3">
          <Textarea id="order-reject-reason" label={O.customHint} value={text} onChange={(e) => setText(e.target.value)} rows={3} />
          <Button variant="danger" className="w-full py-3" disabled={!text.trim()} onClick={() => onPick(text.trim())}>
            {O.send}
          </Button>
        </div>
      )}
    </Modal>
  );
}

function OrderCard({
  order,
  selfManage,
  busy,
  now,
  onAct,
}: {
  order: MerchantOrder;
  selfManage: boolean;
  busy: boolean;
  now: number;
  onAct: (o: MerchantOrder, action: "accept" | "start" | "ready" | "reject", note?: string) => void;
}) {
  const [rejecting, setRejecting] = useState(false);
  const isNew = order.status === "pending";
  const since = Math.max(0, (now - new Date(order.sent_to_merchant_at ?? order.created_at).getTime()) / 1000);
  const ended = !!order.closed_at;
  const tone = isNew
    ? "border-danger-edge bg-danger-tint"
    : ended
      ? "border-line opacity-80"
      : "border-line";

  return (
    <article className={`surface flex flex-col gap-3 rounded-card border p-4 ${tone}`} data-testid="store-order">
      <header className="flex items-center gap-2">
        <span className="heading-card" dir="ltr">
          #{order.number}
        </span>
        {!ended && <span className="text-xs text-ink-muted">{fill(O.waitingFor, { t: fmtSpan(since) })}</span>}
        <span className="ms-auto rounded-badge bg-field px-2 py-1 text-xs font-bold">
          {STATUS[order.status] ?? O.readyDone}
        </span>
      </header>

      <div>
        <p className="text-xs text-ink-muted">{O.items}</p>
        <ul className="mt-1 flex flex-col gap-1">
          {order.items.map((l) => (
            <li key={l.id}>
              <p className="text-sm">
                <span className="font-bold text-accent-text">{fmtNum(l.qty)}×</span> {l.name}
              </p>
              {l.options.length > 0 && (
                <p className="text-xs text-ink-muted">{l.options.map((o) => o.name).join(m.common.listSep)}</p>
              )}
              {l.note.trim() && <p className="text-xs text-ink-muted">{l.note}</p>}
            </li>
          ))}
        </ul>
      </div>

      {order.notes.trim() && (
        <div className="rounded-control border border-warning-edge bg-warning-tint px-3 py-2">
          <p className="text-xs text-ink-muted">{O.note}</p>
          <p className="text-sm">{order.notes}</p>
        </div>
      )}

      <dl className="flex flex-col gap-1 border-t border-line-soft pt-2 text-sm">
        <div className="flex justify-between">
          <dt>{O.sum}</dt>
          <dd>{fmtMoney(order.subtotal)}</dd>
        </div>
        <div className="flex justify-between text-ink-muted">
          <dt>{fill(O.cut, { n: fmtNum(order.commission_percent_exact ?? order.commission_percent) })}</dt>
          <dd>− {fmtMoney(order.platform_commission)}</dd>
        </div>
        <div className="flex justify-between border-t border-line-soft pt-1 font-bold text-accent-text">
          <dt>{O.due}</dt>
          <dd>{fmtMoney(order.merchant_net)}</dd>
        </div>
      </dl>

      {selfManage && order.status === "pending" && (
        <div className="flex flex-col gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={() => onAct(order, "accept")}
            className="w-full rounded-control bg-success px-4 py-4 text-base font-bold text-on-bright transition-opacity hover:opacity-90 disabled:opacity-60"
          >
            {O.accept}
          </button>
          <Button variant="secondary" className="w-full py-3 text-danger" disabled={busy} onClick={() => setRejecting(true)}>
            {O.reject}
          </Button>
        </div>
      )}
      {selfManage && order.status === "accepted" && (
        <Button size="lg" className="w-full" disabled={busy} onClick={() => onAct(order, "start")}>
          {O.start}
        </Button>
      )}
      {/* **و«جاهز» من القبول حتّى وقوف السائق** — لا في «يُحضَّر» وحدَها: في وضع
          المتاجر يقفز القبولُ إلى «البحث عن سائق» فلا يظهر الزرُّ أبداً. */}
      {selfManage && canMarkReady(order) && (
        <Button
          size="lg"
          variant={order.status === "accepted" ? "secondary" : "primary"}
          className="w-full"
          disabled={busy}
          onClick={() => onAct(order, "ready")}
          data-testid="store-order-ready"
        >
          {O.ready}
        </Button>
      )}
      {selfManage && !!order.ready_at && !ended && (
        <p className="text-center text-sm font-bold text-success">{O.readyDone}</p>
      )}
      {rejecting && (
        <RejectDialog
          order={order}
          onClose={() => setRejecting(false)}
          onPick={(r) => {
            setRejecting(false);
            onAct(order, "reject", r);
          }}
        />
      )}
    </article>
  );
}

export default function OrdersPage() {
  const { selfManage } = useStore();
  const { orders, error, reload } = useStoreOrders();
  const toast = useShopToast();
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const now = useNow();

  async function act(o: MerchantOrder, action: "accept" | "start" | "ready" | "reject", note = "") {
    if (busy.has(o.id)) return;
    setBusy((b) => new Set(b).add(o.id));
    try {
      if (action === "ready") {
        await api(`/api/v1/merchant/orders/${o.id}/ready`, { method: "POST", body: "{}" });
      } else {
        const to = action === "accept" ? "accepted" : action === "start" ? "preparing" : "rejected";
        await api(`/api/v1/merchant/orders/${o.id}/transition`, {
          method: "POST",
          body: JSON.stringify({ to, note }),
        });
      }
    } catch (e) {
      toast(errorText(e), "error");
    } finally {
      setBusy((b) => {
        const n = new Set(b);
        n.delete(o.id);
        return n;
      });
      void reload();
    }
  }

  if (!orders && error) return <ReloadState label={errorText(error)} onRetry={() => void reload()} />;
  if (!orders) return <LoadingState variant="list" rows={3} />;

  const active = orders.filter((o) => !o.closed_at);
  const ended = orders.filter((o) => !!o.closed_at);

  return (
    <div className="flex flex-col gap-3">
      <header>
        <h2 className="heading-page">{O.title}</h2>
        {selfManage && <p className="text-sm text-ink-muted">{O.hint}</p>}
      </header>
      {!selfManage && <Alert tone="info">{O.platformMode}</Alert>}
      {orders.length === 0 ? (
        <EmptyState
          icon={IconOrder}
          title={O.empty}
          action={<p className="text-xs text-ink-muted">{O.emptyHint}</p>}
        />
      ) : (
        <>
          {active.length > 0 && (
            <section className="flex flex-col gap-3" aria-label={O.active}>
              {active.map((o) => (
                <OrderCard key={o.id} order={o} selfManage={selfManage} busy={busy.has(o.id)} now={now} onAct={act} />
              ))}
            </section>
          )}
          {ended.length > 0 && (
            <section className="flex flex-col gap-3" aria-label={O.ended}>
              <h3 className="heading-card text-ink-muted">{O.ended}</h3>
              {ended.slice(0, 10).map((o) => (
                <OrderCard key={o.id} order={o} selfManage={selfManage} busy={false} now={now} onAct={act} />
              ))}
            </section>
          )}
        </>
      )}
    </div>
  );
}
