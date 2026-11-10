"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/orders — طلباتُ المتجر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `OrdersScreen.kt` في تطبيق المتجر. والتصميمُ المعتمد ٢٠٢٦-١١-٠٨.)
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
 *
 * **حاسوباً لوحةٌ بأربعة أعمدة** (جديدة · قيد التحضير · جاهزة للاستلام · مع السائق) أو قائمة؛ **وجوّالاً
 * أقسامٌ متتاليةٌ بعناوينها.** وعمودا «جاهزة» و«مع السائق» تقسيمٌ للعرض فقط من حال الطلب
 * (`groupOf` تبقى المرجع)، **ولا أسماءَ زبائن** — المتجرُ لا يراها.
 */

import { useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum, fmtSince } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  EmptyState,
  IconChevronDown,
  IconOrder,
  IconWarning,
  InvoiceLines,
  LoadingState,
  Modal,
  PageHead,
  SectionHeader,
  ReloadState,
  Segmented,
  StatusChip,
  Textarea,
  useIsDesktop,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useStoreOrders } from "@/lib/store/orders";
import { canMarkReady, groupOf, noPayout, type MerchantOrder } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const O = m.store.orders;
const SH = m.store.shell;
const STATUS = O.status as Record<string, string>;

/** دقائقُ الانتظار التي يصير بعدها الطلبُ الجديدُ «متأخّراً» فيُلوَّن عدّادُه. */
const LATE_MIN = 5;

/** أعمدةُ اللوحة — جديدة · قيد التحضير · جاهزة (بانتظار السائق) · مع السائق · (منتهية تحتها). */
type Col = "new" | "preparing" | "ready" | "driver" | "closed";
const COLS: readonly Col[] = ["new", "preparing", "ready", "driver", "closed"];
const WITH_DRIVER = new Set(["picked_up", "on_the_way", "at_dropoff"]);
const DOT: Record<Col, "info" | "warn" | undefined | "off"> = { new: "info", preparing: "warn", ready: undefined, driver: "off", closed: "off" };
const TITLE: Record<Col, string> = {
  new: SH.newOrders,
  preparing: SH.preparing,
  ready: SH.colReady,
  driver: SH.colDriver,
  closed: SH.colClosed,
};

function colOf(o: MerchantOrder): Col {
  const g = groupOf(o);
  if (g === "waiting") return WITH_DRIVER.has(o.status) ? "driver" : "ready";
  return g;
}

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
        <div className="flex flex-col gap-rg-sm">
          {canned.map((r) => (
            <Button key={r} variant="secondary" className="w-full py-rg-md text-start" onClick={() => onPick(r)}>
              {r}
            </Button>
          ))}
          <Button variant="ghost" className="w-full py-rg-md text-start" onClick={() => setCustom(true)}>
            {O.other}
          </Button>
        </div>
      ) : (
        <div className="flex flex-col gap-rg-md">
          <Textarea id="order-reject-reason" label={O.customHint} value={text} onChange={(e) => setText(e.target.value)} rows={3} />
          <Button variant="danger" className="w-full py-rg-md" disabled={!text.trim()} onClick={() => onPick(text.trim())}>
            {O.send}
          </Button>
        </div>
      )}
    </Modal>
  );
}

/** **سطرُ الأصناف المختصر** — «حليب ×2 · أرز ×1». */
function itemsLine(o: MerchantOrder): string {
  return o.items.map((l) => (l.qty > 1 ? `${l.name} ×${fmtNum(l.qty)}` : l.name)).join(" · ");
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
  const [open, setOpen] = useState(false);
  const isNew = order.status === "pending";
  const ended = !!order.closed_at;
  // **عمرُ الطلب بكلماتٍ كاملةٍ كالتطبيق** (`Since.text`): «منذ 5 دقائق» · «منذ ساعتين» — لا «5 د» و«2 س».
  // وفارغٌ إن لم يُفهم الوقت: لا يُكتب «منذ 0».
  const age = ended ? "" : fmtSince(order.sent_to_merchant_at ?? order.created_at, now);
  const late = isNew && now - new Date(order.sent_to_merchant_at ?? order.created_at).getTime() > LATE_MIN * 60_000;
  const state = late ? "late" : isNew ? "new" : undefined;
  /** **هل للمتجر زرٌّ لهذا الحال؟** — وإلّا يُعرض الحالُ شارةً. */
  const hasAction = selfManage && (isNew || order.status === "accepted" || canMarkReady(order));

  return (
    <article className={`order-card ${ended ? "opacity-80" : ""}`} data-state={state} data-testid="store-order">
      <header className="flex items-center gap-rg-sm">
        <b className="text-section-title" dir="ltr">
          #{order.number}
        </b>
        {late && <StatusChip tone="danger">{SH.late}</StatusChip>}
        <span className={`ms-auto text-caption ${late ? "font-bold text-danger" : "text-ink-muted"}`}>{age}</span>
      </header>

      {/* **طلبٌ مشترك**: قسمُك فقط، بلا اسم أيّ محلٍّ آخر (٢٠٢٦-١١-٠٦). */}
      {(order.shared_stops ?? 0) >= 2 && (
        <p className="text-caption font-bold text-accent-text" data-testid="store-order-shared">
          {fill(O.sharedStops, { n: fmtNum(order.shared_stops ?? 0) })}
        </p>
      )}

      <p className="line-clamp-2 text-body leading-relaxed text-ink-muted">{itemsLine(order)}</p>

      {/* **وملاحظةُ الزبون في صندوقٍ بأيقونة تنبيهٍ وخطٍّ عريض** (تدقيقُ ٢٠٢٦-١٠-٢٧) —
          «حساسية» تُقرأ بنظرةٍ أو لا تُقرأ. */}
      {order.notes.trim() && (
        <div className="flex items-start gap-rg-sm rounded-rg-md bg-warning-tint p-rg-md">
          <IconWarning size={20} className="mt-rg-xxs shrink-0 text-warning" aria-hidden />
          <div className="min-w-0 flex-1">
            <p className="text-caption font-bold text-warning">{O.note}</p>
            <p className="text-body font-bold">{order.notes}</p>
          </div>
        </div>
      )}

      {open && (
        <div className="flex flex-col gap-rg-sm border-t border-line-soft pt-rg-sm">
          {/* **سطرٌ لكلّ عنصر** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — «شاورما دبل × 3 ...... 90 ل.س»، كتطبيق المتجر. */}
          <InvoiceLines items={order.items} />
          {/* **وطلبٌ انتهى بلا بيعٍ لا حسبةَ فيه** — انظر `noPayout`. */}
          {noPayout(order.status) ? (
            <p className="text-body text-ink-muted">{O.noPayout}</p>
          ) : (
            <dl className="flex flex-col gap-rg-xs text-body">
              <div className="flex justify-between text-ink-muted">
                <dt>{O.sum}</dt>
                <dd className="tabular-nums">{fmtMoney(order.subtotal)}</dd>
              </div>
              <div className="flex justify-between text-ink-muted">
                <dt>{fill(O.cut, { n: fmtNum(order.commission_percent_exact ?? order.commission_percent) })}</dt>
                <dd className="tabular-nums">− {fmtMoney(order.platform_commission)}</dd>
              </div>
              <div className="flex justify-between font-bold text-accent-text">
                <dt>{O.due}</dt>
                <dd className="tabular-nums">{fmtMoney(order.merchant_net)}</dd>
              </div>
            </dl>
          )}
        </div>
      )}

      <footer className="flex flex-col gap-rg-md border-t border-line-soft pt-rg-md">
        <div className="flex flex-wrap items-center gap-rg-sm">
          <b className="text-price tabular-nums">{fmtMoney(order.subtotal)}</b>
          <button
            type="button"
            onClick={() => setOpen((v) => !v)}
            aria-expanded={open}
            className="flex min-h-rg-touch items-center gap-rg-xxs px-rg-xs text-caption font-bold text-ink-muted hover:text-ink"
          >
            {open ? SH.hideDetails : SH.details}
            <IconChevronDown size={14} className={open ? "rotate-180" : ""} />
          </button>
          {!ended && !hasAction && STATUS[order.status] && (
            <span className="ms-auto">
              <StatusChip tone={order.ready_at ? "success" : "info"}>{STATUS[order.status]}</StatusChip>
            </span>
          )}
        </div>
        {/* **أزرارُ القرار كبيرةٌ بعرض البطاقة** — صاحبُ المتجر مشغولٌ وإبهامُه واحد:
            «اقبل» أعرضُ من «اعتذر»، و«جاهز للاستلام» زرُّ البطاقة الأكبر حين يحين. */}
        {selfManage && order.status === "pending" && (
          <div className="flex items-stretch gap-rg-sm">
            <Button size="lg" variant="danger-outline" className="min-h-rg-touch flex-1" disabled={busy} onClick={() => setRejecting(true)}>
              {O.reject}
            </Button>
            <Button size="lg" className="min-h-rg-touch flex-[2]" disabled={busy} onClick={() => onAct(order, "accept")} data-testid="store-order-accept">
              {O.accept}
            </Button>
          </div>
        )}
        {selfManage && order.status === "accepted" && (
          <Button size="lg" className="min-h-rg-touch w-full" disabled={busy} onClick={() => onAct(order, "start")}>
            {O.start}
          </Button>
        )}
        {/* **و«جاهز» من القبول حتّى وقوف السائق** — لا في «يُحضَّر» وحدَها: في وضع
            المتاجر يقفز القبولُ إلى «البحث عن سائق» فلا يظهر الزرُّ أبداً. */}
        {selfManage && canMarkReady(order) && (
          <Button
            size="lg"
            variant={order.status === "accepted" ? "secondary" : "primary"}
            className="min-h-rg-touch w-full"
            disabled={busy}
            onClick={() => onAct(order, "ready")}
            data-testid="store-order-ready"
          >
            {O.ready}
          </Button>
        )}
      </footer>
      {selfManage && !!order.ready_at && !ended && <p className="text-center text-caption font-bold text-success">{SH.readyNote}</p>}
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
  const desk = useIsDesktop();
  const { orders, error, reload } = useStoreOrders();
  const toast = useShopToast();
  const [busy, setBusy] = useState<Set<string>>(new Set());
  const [mode, setMode] = useState<"board" | "list">("board");
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
  const all = COLS.map((c) => ({ c, list: orders.filter((o) => colOf(o) === c) }));
  const liveCount = all.filter((x) => x.c !== "closed").reduce((a, x) => a + x.list.length, 0);
  const closed = all.find((x) => x.c === "closed")?.list ?? [];
  const live = all.filter((x) => x.c !== "closed");

  const card = (o: MerchantOrder) => <OrderCard key={o.id} order={o} selfManage={selfManage} busy={busy.has(o.id)} now={now} onAct={act} />;
  const head = (c: Col, n: number) => (
    <h3 className="flex items-center gap-rg-sm px-rg-xs pb-rg-xxs pt-rg-xs text-card-title">
      <span className="dot" data-tone={DOT[c]} aria-hidden />
      {TITLE[c]}
      <span className="ms-auto text-caption font-medium text-ink-muted tabular-nums">{fmtNum(n)}</span>
    </h3>
  );

  return (
    <div className="console-page" data-testid="store-orders-page">
      <PageHead
        title={m.store.nav.orders}
        subtitle={fill(SH.ordersSub, { n: fmtNum(liveCount) })}
        actions={
          <>
            <Link href="/store/history" className="flex min-h-rg-touch items-center text-body font-bold text-primary hover:underline">
              {O.historyLink}
            </Link>
            {desk && (
              <Segmented
                ariaLabel={m.store.nav.orders}
                value={mode}
                onChange={setMode}
                items={[
                  { key: "board", label: SH.viewBoard },
                  { key: "list", label: SH.viewList },
                ]}
              />
            )}
          </>
        }
      />
      {!selfManage && (
        <Alert tone="info">{O.platformMode}</Alert>
      )}
      {orders.length === 0 ? (
        <EmptyState
          icon={IconOrder}
          title={O.empty}
          action={
            <div className="flex flex-col items-center gap-rg-md">
              <p className="text-caption text-ink-muted">{O.emptyHint}</p>
              <Button variant="secondary" onClick={() => router.push("/store")}>
                {O.goStore}
              </Button>
            </div>
          }
        />
      ) : desk && mode === "board" ? (
        <>
          <div className="board" data-testid="store-orders-board">
            {live.map(({ c, list }) => (
              <section key={c} className="board-col" aria-label={TITLE[c]} data-col={c}>
                {head(c, list.length)}
                {list.length === 0 && <p className="py-rg-xl text-center text-body text-ink-muted">{O.colEmpty}</p>}
                {list.map(card)}
              </section>
            ))}
          </div>
          {closed.length > 0 && (
            <section className="flex flex-col gap-rg-sm">
              {head("closed", closed.length)}
              <div className="grid grid-cols-3 gap-rg-md 2xl:grid-cols-4">{closed.map(card)}</div>
            </section>
          )}
        </>
      ) : (
        <div className="flex flex-col gap-rg-xs" data-testid="store-orders-board">
          {all
            .filter((x) => x.list.length > 0)
            .map(({ c, list }) => (
              <section key={c} className="flex flex-col gap-rg-sm lg:pb-rg-md" aria-label={TITLE[c]}>
                <SectionHeader title={`${TITLE[c]} · ${fmtNum(list.length)}`} />
                <div className="flex flex-col gap-rg-sm md:grid md:grid-cols-2 xl:grid-cols-3">{list.map(card)}</div>
              </section>
            ))}
        </div>
      )}
    </div>
  );
}
