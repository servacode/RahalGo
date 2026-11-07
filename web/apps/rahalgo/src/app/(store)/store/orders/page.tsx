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
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum, fmtSince } from "@rahalgo/i18n";
import { Alert, Button, EmptyState, IconHourglass, IconOrder, IconWarning, InvoiceLines, LoadingState, Modal, PageHead, Pill, ReloadState, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useStoreOrders } from "@/lib/store/orders";
import { canMarkReady, groupOf, noPayout, ORDER_GROUPS, type MerchantOrder, type OrderGroup } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const O = m.store.orders;
const STATUS = O.status as Record<string, string>;

/** **نبرةُ شارة الحال بمجموعة الطلب** — والنصُّ يقول الحالَ، فلا يعتمد المعنى على اللون وحدَه. */
const TONE: Record<OrderGroup, "danger" | "attention" | "info" | undefined> = {
  new: "danger",
  preparing: "attention",
  waiting: "info",
  closed: undefined,
};
/** دقائقُ الانتظار التي يصير بعدها الطلبُ الجديدُ «متأخّراً» فيُلوَّن عدّادُه. */
const LATE_MIN = 5;
const DOT: Record<OrderGroup, string> = {
  new: "bg-danger-solid",
  preparing: "bg-warning-solid",
  waiting: "bg-primary-solid",
  closed: "bg-ink-dim",
};

/** **ثانيةٌ تمشي** — «منذ 3 دقائق» تتبدّل بلا نداء. */
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
  const ended = !!order.closed_at;
  // **عمرُ الطلب بكلماتٍ كاملةٍ كالتطبيق** (`Since.text`): «منذ 5 دقائق» · «منذ ساعتين» — لا «5 د» و«2 س».
  // وفارغٌ إن لم يُفهم الوقت: لا يُكتب «منذ 0».
  const age = ended ? "" : fmtSince(order.sent_to_merchant_at ?? order.created_at, now);
  const late = isNew && now - new Date(order.sent_to_merchant_at ?? order.created_at).getTime() > LATE_MIN * 60_000;
  /* **الجديدُ يُرى قبل أن يُقرأ**: حدُّ بدايته أحمرُ سميكٌ وأرضيّتُه محمرّة، والمنتهي باهت. */
  const tone = isNew
    ? "border-danger-edge bg-danger-tint shadow-[inset_0_3px_0_var(--color-danger-solid)]"
    : ended
      ? "opacity-80"
      : "";

  return (
    <article className={`panel-lite flex flex-col gap-3.5 p-4 ${tone}`} data-testid="store-order">
      <header className="flex items-center gap-2.5">
        <span className="icon-chip" data-tone={TONE[groupOf(order)]} aria-hidden>
          <IconOrder size={18} />
        </span>
        <span className="text-xl font-extrabold tabular-nums" dir="ltr">
          #{order.number}
        </span>
        {age && (
          <Pill tone={late ? "danger" : undefined} dot={false}>
            <IconHourglass size={13} />
            {age}
          </Pill>
        )}
        <span className="ms-auto">
          <Pill tone={TONE[groupOf(order)]}>{STATUS[order.status] ?? O.readyDone}</Pill>
        </span>
      </header>

      {/* **طلبٌ مشترك**: قسمُك فقط، بلا اسم أيّ محلٍّ آخر (٢٠٢٦-١١-٠٦). */}
      {(order.shared_stops ?? 0) >= 2 && (
        <p className="text-xs font-bold text-accent-text" data-testid="store-order-shared">
          {fill(O.sharedStops, { n: fmtNum(order.shared_stops ?? 0) })}
        </p>
      )}

      <div>
        <p className="text-xs text-ink-muted">{O.items}</p>
        {/* **سطرٌ لكلّ عنصر** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — «شاورما دبل × 3 ...... 90 ل.س»، كتطبيق المتجر. */}
        <InvoiceLines items={order.items} className="mt-1" />
      </div>

      {/* **وملاحظةُ الزبون في صندوقٍ بأيقونة تنبيهٍ وخطٍّ عريض** (تدقيقُ ٢٠٢٦-١٠-٢٧) —
          «حساسية» تُقرأ بنظرةٍ أو لا تُقرأ. */}
      {order.notes.trim() && (
        <div className="flex items-start gap-2 rounded-control border border-warning-edge bg-warning-tint px-3 py-2">
          <IconWarning size={18} className="mt-0.5 shrink-0 text-accent-text" aria-hidden />
          <div className="min-w-0 flex-1">
            <p className="text-xs font-bold text-accent-text">{O.note}</p>
            <p className="text-sm font-bold">{order.notes}</p>
          </div>
        </div>
      )}

      {/* **وطلبٌ انتهى بلا بيعٍ لا حسبةَ فيه** — انظر `noPayout`. */}
      {noPayout(order.status) ? (
        <p className="border-t border-line-soft pt-2 text-sm text-ink-muted">{O.noPayout}</p>
      ) : (
        <dl className="flex flex-col gap-1.5 border-t border-line-soft pt-3 text-sm">
          <div className="flex justify-between text-ink-muted">
            <dt>{O.sum}</dt>
            <dd className="tabular-nums">{fmtMoney(order.subtotal)}</dd>
          </div>
          <div className="flex justify-between text-ink-muted">
            <dt>{fill(O.cut, { n: fmtNum(order.commission_percent_exact ?? order.commission_percent) })}</dt>
            <dd className="tabular-nums">− {fmtMoney(order.platform_commission)}</dd>
          </div>
          <div className="mt-1 flex items-baseline justify-between rounded-tile bg-field px-3 py-2 font-bold text-accent-text">
            <dt>{O.due}</dt>
            <dd className="text-lg font-extrabold tabular-nums">{fmtMoney(order.merchant_net)}</dd>
          </div>
        </dl>
      )}

      {selfManage && order.status === "pending" && (
        <div className="flex flex-col gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={() => onAct(order, "accept")}
            className="min-h-12 w-full rounded-2xl bg-success px-4 py-3 text-base font-extrabold text-on-bright transition-opacity hover:opacity-90 disabled:opacity-60"
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
  const router = useRouter();
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

  // ══════════════════════════════════════════════════════════════════
  // **والطلباتُ بمجموعاتها — الجديدُ أوّلاً** (مرآةُ `groupOf`، تدقيقُ ٢٠٢٦-١٠-٢٧)
  // ══════════════════════════════════════════════════════════════════
  //
  // **كانت قائمةً واحدةً بترتيب الخادم** — فطلبٌ جديدٌ ينتظر قراراً قد يقع
  // تحت طلبٍ يُحضَّر، **ومهلتُه تمضي وهو لا يُرى.** والمنتهي في «سجلّ الطلبات».
  const all = ORDER_GROUPS.map((g) => ({ g, list: orders.filter((o) => groupOf(o) === g) }));
  /* **على الجوّال الفارغُ يُخفى كما كان؛ وعلى `lg` ثلاثةُ أعمدةٍ ثابتةٌ** (جديدة | قيد التحضير | بانتظار
     السائق) ولو فارغةً — فالعينُ تتعلّم موضعَ كلّ حال. والمنتهي صفٌّ كاملٌ تحتها. */
  const groups = all.filter((x) => x.list.length > 0 || x.g !== "closed");

  return (
    <div className="console-page">
      <PageHead
        title={O.title}
        subtitle={fill(O.liveCount, { n: fmtNum(all.filter((x) => x.g !== "closed").reduce((a, x) => a + x.list.length, 0)) })}
        actions={
          <Link href="/store/history" className="flex min-h-11 items-center gap-1 text-sm font-bold text-primary hover:underline">
            {O.historyLink}
          </Link>
        }
      />
      {!selfManage && <Alert tone="info">{O.platformMode}</Alert>}
      {orders.length === 0 ? (
        <EmptyState
          icon={IconOrder}
          title={O.empty}
          action={
            <div className="flex flex-col items-center gap-3">
              <p className="text-xs text-ink-muted">{O.emptyHint}</p>
              <Button variant="secondary" onClick={() => router.push("/store")}>
                {O.goStore}
              </Button>
            </div>
          }
        />
      ) : (
        <div className="flex flex-col gap-4 lg:grid lg:grid-cols-3 lg:items-start lg:gap-4" data-testid="store-orders-board">
          {groups.map(({ g, list }) => (
            <section
              key={g}
              className={`flex min-w-0 flex-col gap-3 ${list.length === 0 ? "hidden lg:flex" : ""} ${
                g === "closed" ? "lg:col-span-3" : "lg:rounded-3xl lg:border lg:border-line-soft lg:bg-field lg:p-3"
              }`}
              aria-label={fill(O.groups[g], { n: fmtNum(list.length) })}
            >
              <h3 className={`console-h2 flex items-center gap-2 pt-2 lg:pt-0 ${g === "new" ? "text-ink" : "text-ink-muted"}`}>
                <span className={`size-2.5 shrink-0 rounded-badge ${DOT[g]} ${g === "new" && list.length > 0 ? "motion-safe:animate-pulse" : ""}`} aria-hidden />
                {fill(O.groups[g], { n: fmtNum(list.length) })}
              </h3>
              {list.length === 0 && <p className="py-6 text-center text-sm text-ink-muted">{O.colEmpty}</p>}
              <div className={g === "closed" ? "flex flex-col gap-3 lg:grid lg:grid-cols-3 2xl:grid-cols-4" : "contents"}>
                {list.map((o) => (
                  <OrderCard key={o.id} order={o} selfManage={selfManage} busy={busy.has(o.id)} now={now} onAct={act} />
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
    </div>
  );
}
