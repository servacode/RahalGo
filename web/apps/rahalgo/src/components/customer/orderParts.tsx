"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أجزاءُ الطلب — المراحلُ وكودُ الاستلام وتأكيدُ السعر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مرآةُ `OrderCard.kt`**: المراحلُ يحسبها المحرّك (`stages` و`stage_at`)
 * **والشاشةُ تترجمها وحدَها** — أربعَ عشرةَ حالاً تُطوى في ستّ مراحل،
 * **والطيُّ في موضعٍ واحدٍ لا في كلّ شاشة.**
 *
 * **ونافذةُ تأكيد السعر بلا «لاحقاً»** (قرارُ المالك): التسليمُ موقوفٌ على
 * جوابه — **قبولٌ أخضرُ أو رفضٌ أحمر**، ولا بابَ ثالث.
 */

import { useEffect, useState } from "react";
import { createPortal } from "react-dom";
import { getMessages, defaultLocale, errorText, fmtRef, fmtSpan } from "@rahalgo/i18n";
import { Alert, Button, Chips, Confirm, Money } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { fill, stageLabel } from "@/lib/customer/texts";
import { ENDED, type MyOrder } from "@/lib/customer/types";

const m = getMessages(defaultLocale);
const O = m.shop.orders;

/** **أينتظر هذا الطلبُ تأكيدَ سعرٍ منه؟** — مرآةُ `quotePending`. */
export function quotePending(o: MyOrder): boolean {
  if (o.kind !== "custom" || o.custom_goods_amount == null) return false;
  if (o.quote_confirmed_at != null && o.quote_confirmed_version === o.quote_version) return false;
  if (ENDED.has(o.status)) return false;
  return o.quote_rejected_at == null;
}

/** **شريطُ المراحل** — المنتهي مُضاء، والجاري بارز، والآتي باهت. */
export function OrderStages({ order }: { order: MyOrder }) {
  if (order.stage_at < 0 || order.stages.length === 0) return null;
  const amanah = order.custom_mode === "amanah";
  return (
    <ol className="flex w-full items-start gap-1" aria-label={O.live}>
      {order.stages.map((s, i) => {
        const done = i < order.stage_at;
        const now = i === order.stage_at;
        return (
          <li key={s} className="flex min-w-0 flex-1 flex-col items-center gap-1.5">
            <span
              className={`h-1.5 w-full rounded-badge ${
                done ? "bg-accent" : now ? "bg-accent motion-safe:animate-pulse" : "bg-line-soft"
              }`}
            />
            <span
              className={`w-full truncate text-center text-2xs ${
                now ? "font-bold text-accent-text" : done ? "text-ink" : "text-ink-dim"
              }`}
            >
              {stageLabel(s, amanah)}
            </span>
          </li>
        );
      })}
    </ol>
  );
}

/**
 * **كم يأخذ تقريباً — من المحرّك** (مرآةُ `OrderCard.kt` ٢٠٢٦-١٠-٢٧).
 *
 * **وبالطريق زمنُ الطريق من المتجر إلى بابه** (`to_door_eta_sec`، يُلتقط عند الاستلام)،
 * **وقبله مدّةُ التوصيل المتوقّعة** (`delivery_estimate_min`). **ولا يُخترَع رقم.**
 */
export function etaText(o: MyOrder): string {
  if (ENDED.has(o.status)) return "";
  const door = o.to_door_eta_sec ?? 0;
  if (door > 0 && (o.status === "picked_up" || o.status === "on_the_way")) {
    return fill(O.etaDoor, { t: fmtSpan(Math.ceil(door / 60) * 60) });
  }
  const est = o.delivery_estimate_min ?? 0;
  if (est > 0 && o.status !== "at_dropoff") return fill(O.etaEstimate, { t: fmtSpan(est * 60) });
  return "";
}

export function OrderEta({ order }: { order: MyOrder }) {
  const text = etaText(order);
  if (!text) return null;
  return <p className="text-sm font-bold text-primary">{text}</p>;
}

/**
 * **كودُ الاستلام** — حقلٌ في الطلب نفسِه (`delivery_code`، ٢٠٢٦-١٠-١٧)، يرسله المحرّكُ
 * لصاحب الطلب وحدَه من الاستلام حتّى الباب. **ولا يُقرأ من نصّ إشعار.**
 */
export function DeliveryCode({ order }: { order: MyOrder }) {
  const code = order.delivery_code ?? "";
  if (!code) return null;
  return (
    <div className="surface-inset flex items-center justify-between gap-3 p-3">
      <div className="min-w-0">
        <p className="text-sm font-bold">{O.code}</p>
        <p className="text-xs text-ink-muted">{O.codeHint}</p>
      </div>
      <span dir="ltr" className="figure tracking-[0.3em] text-accent-text">
        {code}
      </span>
    </div>
  );
}

/**
 * **مهلةُ الإلغاء بكلامٍ مفهوم** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — مرآةُ `CancelCountdown.within`:
 * «46 ثانية» · «دقيقة و10 ثوانٍ» · «دقيقتين»، **بأرقامٍ غربيّة.**
 */
export function cancelWithin(secs: number): string {
  if (secs <= 0) return "";
  const m = Math.floor(secs / 60);
  const s = secs % 60;
  const W = O.within;
  const unit = (n: number, one: string, two: string, few: string, many: string) =>
    n === 0 ? "" : n === 1 ? one : n === 2 ? two : fill(n <= 10 ? few : many, { n: String(n) });
  const min = unit(m, W.minOne, W.minTwo, W.minFew, W.minMany);
  const sec = unit(s, W.secOne, W.secTwo, W.secFew, W.secMany);
  if (!min) return sec;
  if (!sec) return min;
  return fill(W.and, { a: min, b: sec });
}

/** **الثواني الباقية من مهلة الإلغاء — تمشي والصفحةُ مفتوحة.** و`-1` «بلا مهلة». */
function useCancelLeft(order: MyOrder): number {
  const [left, setLeft] = useState(order.cancel_seconds_left);
  useEffect(() => setLeft(order.cancel_seconds_left), [order.cancel_seconds_left]);
  useEffect(() => {
    if (left <= 0) return;
    const t = setInterval(() => setLeft((s) => s - 1), 1000);
    return () => clearInterval(t);
  }, [left > 0]); // eslint-disable-line react-hooks/exhaustive-deps
  return left;
}

/**
 * **«يمكنك إلغاء الطلب خلال 46 ثانية» جنبَ الحال** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — والزرُّ تحت
 * بلا عدّاد. **ويغيب مع الزرّ حين تنقضي المهلة.**
 */
export function CancelNote({ order }: { order: MyOrder }) {
  const left = useCancelLeft(order);
  if (ENDED.has(order.status) || left <= 0) return null;
  return <span className="text-xs text-danger tabular-nums">{fill(O.cancelWithin, { t: cancelWithin(left) })}</span>;
}

/** **زرُّ الإلغاء ما دامت المهلة** — والمهلةُ من الخادم لا من حسبةِ الشاشة. */
export function CancelOrder({ order, onDone }: { order: MyOrder; onDone: () => void }) {
  const left = useCancelLeft(order);
  const [ask, setAsk] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  if (ENDED.has(order.status) || left === 0 || left < -1) return null;
  return (
    <div className="space-y-2">
      <Button variant="danger" className="w-full" onClick={() => setAsk(true)} disabled={busy}>
        {O.cancel}
      </Button>
      {error && <Alert>{error}</Alert>}
      <Confirm
        open={ask}
        title={O.cancelTitle}
        body={O.cancelAsk}
        confirmLabel={O.cancelYes}
        busy={busy}
        onCancel={() => setAsk(false)}
        onConfirm={async () => {
          setBusy(true);
          setError("");
          try {
            await api(`/api/v1/orders/${order.id}/cancel`, { method: "POST", body: JSON.stringify({ note: "" }) });
            setAsk(false);
            onDone();
          } catch (err) {
            setError(errorText(err));
            setAsk(false);
          } finally {
            setBusy(false);
          }
        }}
      />
    </div>
  );
}

/** **مبالغُ العرض** — ثمنُ البضاعة والأجرةُ والمجموع. */
export function QuoteAmounts({ order }: { order: MyOrder }) {
  const goods = order.custom_goods_amount;
  if (goods == null) return null;
  const fee = order.custom_fee ?? 0;
  const waived = order.free_waived ?? 0;
  return (
    <dl className="space-y-1 text-sm">
      {order.custom_mode !== "amanah" && (
        <div className="flex justify-between gap-2">
          <dt className="text-ink-muted">{O.goods}</dt>
          <dd>{order.custom_goods_pending ? O.goodsAfter : <Money value={goods} />}</dd>
        </div>
      )}
      <div className="flex justify-between gap-2">
        <dt className="text-ink-muted">{O.delivery}</dt>
        {/* **والأجرةُ المُعفاةُ تُقال «مجاني 🎁» والأصلُ مشطوب** — كالطلب العاديّ (`DeliveryFeeRow`). */}
        <dd>
          {waived > 0 ? (
            <span className="flex items-center gap-2">
              <s className="text-xs text-ink-muted">
                <Money value={fee + waived} />
              </s>
              {fee > 0 ? <Money value={fee} /> : <span className="font-bold text-success">{m.shop.free.badge}</span>}
            </span>
          ) : (
            <Money value={fee} />
          )}
        </dd>
      </div>
      <div className="flex justify-between gap-2 font-bold">
        <dt>{O.total}</dt>
        <dd className="text-accent-text">
          <Money value={order.total} />
        </dd>
      </div>
    </dl>
  );
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تأكيدُ السعر — يُرسَل ما رآه** (`expected_total` و`expected_quote_version`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **فإن تبدّل العرضُ بين الرؤية والضغطة ردّ المحرّكُ `quote_changed`** —
 * فيُعاد جلبُه ويُسأل من جديد. **والمحفظةُ لا تُعرض إلّا إن غطّت المبلغ.**
 */
export function QuotePanel({
  order,
  available,
  onDone,
  onMessage,
}: {
  order: MyOrder;
  available: number;
  onDone: () => void;
  onMessage: (text: string) => void;
}) {
  const ownReserved = order.payment_method === "wallet" ? order.custom_reserved_amount : 0;
  const walletOk = available + ownReserved >= order.total;
  const [method, setMethod] = useState<"cash" | "wallet">("cash");
  const [busy, setBusy] = useState(false);
  const [askReject, setAskReject] = useState(false);
  const [error, setError] = useState("");
  const amanah = order.custom_mode === "amanah";
  const pay = method === "wallet" && !walletOk ? "cash" : method;

  const confirm = async () => {
    const slot = "confirm_" + order.id;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/orders/${order.id}/confirm-quote`, {
        method: "POST",
        headers: { "Idempotency-Key": attemptKey(slot) },
        body: JSON.stringify({
          payment_method: pay,
          expected_total: order.total,
          expected_quote_version: order.quote_version,
        }),
      });
      clearAttempt(slot);
      onMessage(pay === "wallet" ? O.confirmedWallet : O.confirmedCash);
      onDone();
    } catch (err) {
      const code = (err as { body?: { code?: string } })?.body?.code;
      if (code === "quote_changed") {
        clearAttempt(slot);
        onMessage(O.quoteStale);
        onDone();
      } else {
        if (isDecided(err)) clearAttempt(slot);
        setError(errorText(err));
      }
    } finally {
      setBusy(false);
    }
  };

  const reject = async () => {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/orders/${order.id}/reject-quote`, { method: "POST", body: "{}" });
      setAskReject(false);
      onMessage(O.rejectedSent);
      onDone();
    } catch (err) {
      setAskReject(false);
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="space-y-3">
      {order.quote_confirmed_version != null && <p className="text-sm">{O.raisedNote}</p>}
      <p className="text-sm font-bold">{walletOk ? O.confirmPrompt : O.confirmAmount}</p>
      {/* **والمحفظةُ خيارٌ فقط إن غطّت المبلغ** — وإلّا سطرٌ واحدٌ «الدفع: نقداً عند الاستلام»
          (قرارُ المالك ٢٠٢٦-١٠-٣٠)، والطريقةُ تبقى نقداً. مرآةُ `QuoteConfirmPanel`. */}
      {walletOk ? (
        <Chips<"cash" | "wallet">
          items={[
            { id: "cash", label: m.shop.cart.cash },
            { id: "wallet", label: m.shop.cart.wallet },
          ]}
          value={pay}
          onChange={setMethod}
        />
      ) : (
        <p className="text-sm">{O.payCashOnly}</p>
      )}
      {error && <Alert>{error}</Alert>}
      <button
        type="button"
        onClick={confirm}
        disabled={busy}
        className="w-full rounded-control bg-success px-4 py-3 text-sm font-bold text-on-bright transition-opacity hover:opacity-90 disabled:opacity-60"
      >
        {busy ? m.common.loading : amanah || order.custom_goods_pending ? O.confirmFee : O.confirmGoods}
      </button>
      <Button variant="danger" className="w-full py-3" onClick={() => setAskReject(true)} disabled={busy}>
        {O.reject}
      </Button>
      <Confirm
        open={askReject}
        title={O.rejectTitle}
        body={O.rejectBody}
        confirmLabel={O.rejectYes}
        busy={busy}
        onCancel={() => setAskReject(false)}
        onConfirm={reject}
      />
    </div>
  );
}

/**
 * **نافذةُ السعر — تُفتح من أيّ شاشة ولا تُغلق إلّا بجواب.**
 *
 * **ولا `Modal`**: تلك تحمل ✕ وتُغلق بالضغط خارجها — **وهو «لاحقاً» بلا
 * اسم.** فتُرسم هنا بحجاب المنصّة ولوحها (`scrim` · `surface-modal`).
 */
export function QuoteDialog({
  order,
  available,
  onDone,
  onMessage,
}: {
  order: MyOrder;
  available: number;
  onDone: () => void;
  onMessage: (text: string) => void;
}) {
  if (typeof document === "undefined") return null;
  return createPortal(
    <div className="fixed inset-0 z-[80] flex items-end justify-center scrim p-0 sm:items-center sm:p-4">
      <div
        role="alertdialog"
        aria-modal="true"
        aria-labelledby="quote-t"
        className="max-h-[90vh] w-full overflow-y-auto surface-modal p-5 sm:max-w-md"
      >
        <h2 id="quote-t" className="heading-card mb-3">
          {fill(O.dialogTitle, { n: fmtRef(order.number) })}
        </h2>
        {order.custom_request && (
          <p className="mb-3 line-clamp-2 text-sm">
            {O.wanted} {order.custom_request}
          </p>
        )}
        <div className="mb-4">
          <QuoteAmounts order={order} />
        </div>
        <QuotePanel order={order} available={available} onDone={onDone} onMessage={onMessage} />
      </div>
    </div>,
    document.body,
  );
}
