"use client";

/** تتبع الطلب الحي: خط تقدم بالحالات، تحديث لحظي عبر WebSocket، والتقييم بعد التسليم. */

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtTime, fmtClock } from "@rahalgo/i18n";
import {
  IconCheck,
  IconLocation,
  Badge,
  Button,
  useLiveEvent,
  IconStar,
  IconSuccess,
  IconPrint,
  Invoice,
  Timeline,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const STATUS_LABELS: Record<string, string> = m.orders.status;

// مسار التقدم الطبيعي المعروض للزبون
const FLOW = ["pending", "accepted", "preparing", "assigned", "on_the_way", "delivered"];

/**
 * وقتُ كل خطوةٍ من الحقل الذي يحملها.
 *
 * **وما لا وقتَ له يبقى بلا وقت** — لا يُخمَّن ولا يُملأ بوقتٍ قريب. وقتٌ
 * مُخمَّنٌ في شاشةٍ يُقرأ حقيقةً، ثم يُبنى عليه اتّهامٌ لمن لم يتأخّر.
 */
const STEP_TIME: Record<string, (o: Order) => string | undefined> = {
  pending: (o) => fmtTime(o.created_at),
  accepted: (o) => (o.accepted_at ? fmtTime(o.accepted_at) : undefined),
  preparing: (o) => (o.ready_at ? fmtTime(o.ready_at) : undefined),
  delivered: (o) => (o.delivered_at ? fmtTime(o.delivered_at) : undefined),
};
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
  // بيانات الفاتورة — يرسلها الخادم أصلاً وكان النوع يتجاهلها
  payment_method: string;
  merchant_id: string;
  prep_minutes?: number | null;
  accepted_at?: string | null;
  /** ما بقي من مهلة الإلغاء بالثواني — و`-1` تعني «بلا مهلة» (قبل القبول). */
  cancel_seconds_left?: number;
  ready_at?: string | null;
  subtotal: number;
  delivery_fee: number;
  discount: number;
  created_at: string;
  delivered_at?: string | null;
  rating?: Rating;
  items?: {
    id: string;
    name: string;
    qty: number;
    unit_price: number;
    note?: string;
    options?: { group: string; name: string; price_delta: number }[];
  }[];
}

/**
 * الوقت المتوقّع = لحظة القبول + وقت تحضير المتجر + تقدير التوصيل.
 *
 * ويُعرض **بالمتبقّي لا بالساعة**: «خلال ١٢ دقيقة» أقرب إلى ذهن المنتظِر من
 * «يصل ٨:٤٧». وإن أعلن المتجر الجاهزية سقط وقت التحضير من الحساب — صار الطلب
 * ينتظر السائق لا المطبخ.
 */
const DELIVERY_ESTIMATE_MIN = 15;

/** رسالة الخطأ من مفتاح الخادم — لا نصّ إنجليزي يصل المستخدم. */
function errText(err: unknown): string {
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key.split(".").pop() ?? "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

function etaText(o: Order): string {
  const accepted = new Date(o.accepted_at!).getTime();
  const prepDone = o.ready_at ? new Date(o.ready_at).getTime() : accepted + (o.prep_minutes ?? 0) * 60_000;
  const left = Math.round((prepDone + DELIVERY_ESTIMATE_MIN * 60_000 - Date.now()) / 60_000);
  return m.site.orders.etaValue.replace("{n}", fmtNum(Math.max(1, left)));
}

export default function OrderTrackingPage() {
  const { id } = useParams<{ id: string }>();
  const [order, setOrder] = useState<Order | null>(null);
  const [error, setError] = useState("");
  const [showInvoice, setShowInvoice] = useState(false);
  const [cancelBusy, setCancelBusy] = useState(false);
  /**
   * ما بقي من نافذة الإلغاء بالثواني.
   *
   * **يُعاد حسابُه كلَّ ثانية** — فالعدّادُ الذي لا يعدّ ليس عدّاداً. ويُقرأ
   * من `accepted_at` والمهلةِ الآتية من الخادم، **فلا رقمَ مكتوبٌ في الشاشة
   * يخالف رقماً في الإعدادات.**
   */
  const [cancelLeft, setCancelLeft] = useState(0);

  const [cancelError, setCancelError] = useState("");

  const load = useCallback(() => {
    api<Order>(`/api/v1/my/orders/${id}`)
      .then(setOrder)
      .catch(() => setError(m.errors.not_found));
  }, [id]);

  useEffect(() => {
    load();
  }, [load]);

  /**
   * العدّادُ ينطلق من رقم الخادم **لا من حسابٍ محليّ**.
   *
   * الرقمُ نسبيٌّ عند وصوله، **فيُثبَّت مرساه بساعة الجهاز نفسِه** ويُطرح منه
   * ما مضى. وبهذا لا تدخل ساعةُ الخادم في الحساب أصلاً — **وفارقُ الساعتين
   * لا يجعل زرّاً حيّاً يبدو منقضياً.**
   */
  useEffect(() => {
    const left = order?.cancel_seconds_left ?? 0;
    if (left <= 0) {
      setCancelLeft(0);
      return;
    }
    const anchor = Date.now();
    const tick = () => setCancelLeft(Math.max(0, left - Math.floor((Date.now() - anchor) / 1000)));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [order?.cancel_seconds_left]);

  // البث الحي المركزي — قناة واحدة للتطبيق كله، بلا استطلاع دوري
  useLiveEvent((event) => {
    const o = event.order as { id?: string } | undefined;
    if (event.type === "order" && o?.id === id) load();
  });

  if (error) return <p className="py-10 text-center text-ink-muted">{error}</p>;
  if (!order) return <p className="py-10 text-center text-ink-muted">{m.common.loading}</p>;

  const norm = NORMALIZE[order.status] ?? order.status;
  const stepIdx = FLOW.indexOf(norm);
  const failed = ["rejected", "cancelled", "failed", "refunded"].includes(order.status);

  return (
    <div>
      {/* **ترويسةٌ واحدة تجيب ثلاثة أسئلة معاً**: أيُّ طلبٍ هذا، وأين وصل،
          ومتى يصل. كانت ثلاثةَ أسطرٍ متفرّقة — **والعينُ تقفز بينها لتجمع
          خبراً واحداً.** */}
      <header
        className={`mb-5 overflow-hidden rounded-card border ${
          failed ? "border-danger/30" : "border-line"
        } bg-surface`}
      >
        <div
          className={`flex flex-wrap items-center justify-between gap-3 px-5 py-4 ${
            failed
              ? "bg-danger/5"
              : order.status === "delivered"
                ? "bg-success/5"
                : "bg-primary-light"
          }`}
        >
          <div className="min-w-0">
            <p className="text-xs text-ink-muted">{order.merchant_name}</p>
            <h1 className="mt-0.5 text-2xl font-bold tabular-nums" dir="ltr">
              #{fmtNum(order.number)}
            </h1>
          </div>
          <div className="flex flex-wrap items-center gap-2">
            <Badge
              variant={failed ? "danger" : order.status === "delivered" ? "success" : "primary"}
            >
              {/* **الشارةُ تقرأ المُطبَّعة كما يقرؤها الخطّ.**

                  كانت تقرأ الحالةَ الخام — ففي `dispatching` تقول «جارٍ إسناد
                  سائق» **والخطُّ تحتها يُضيء «قيد التحضير»: جوابان في شاشةٍ
                  واحدة.**

                  وأسوأُ من التناقض معناه: **«جارٍ إسناد سائق» تُخبر الزبونَ
                  بشؤوننا الداخلية** فيقرؤها قلقاً — ولا يحتاج أن يعرف أن لنا
                  طابوراً. */}
              {STATUS_LABELS[norm] ?? order.status}
            </Badge>
            <Button
              variant="secondary"
              onClick={() => setShowInvoice((v) => !v)}
              className="flex items-center gap-1.5"
            >
              <IconPrint size={15} />
              {m.shared.invoice.open}
            </Button>
          </div>
        </div>

        {/* الوقت المتوقّع: «قيد التحضير» وحدها لا تقول عشر دقائق أم ساعة */}
        {!failed && order.status !== "delivered" && order.accepted_at && order.prep_minutes && (
          <p className="flex items-center gap-2 border-t border-line px-5 py-2.5 text-sm text-primary-dark">
            <IconCheck size={16} strokeWidth={3} />
            <span className="font-medium">{m.site.orders.eta}:</span>
            {etaText(order)}
          </p>
        )}
      </header>

      {/* **الإلغاء: نافذةٌ تُرى وهي تنقضي.**

          كان الزرُّ يبقى طوالَ حالة «مقبول» **بلا فحصٍ للوقت** — فيُضغط بعد
          ساعةٍ فيعتذر. **وزرٌّ يَعِد بما لا يفعله الخادم** هو ما نطارده منذ
          يومين.

          والنصُّ كان يقول «خلال دقيقتين» **مكتوبةً بالحرف والمهلةُ إعدادٌ
          يملك المالكُ تغييره** — **ورقمٌ في نصٍّ يخالف رقماً في إعدادٍ أخطرُ
          من غياب الرقم: غيابُه يُسأل عنه، وخلافُه يُصدَّق.**

          **والعدّادُ يحلّهما معاً**: يقرأ المهلةَ من الخادم فلا نصَّ يُكتب،
          ويختفي بانقضائها فلا زرَّ يعتذر. */}
      {(order.status === "pending" || (order.status === "accepted" && cancelLeft > 0)) && (
        <div className="mb-4">
          <Button
            variant="danger"
            disabled={cancelBusy}
            onClick={async () => {
              setCancelBusy(true);
              setCancelError("");
              try {
                await api(`/api/v1/orders/${order.id}/cancel`, {
                  method: "POST",
                  body: JSON.stringify({ note: "" }),
                });
                load();
              } catch (err) {
                setCancelError(errText(err));
              } finally {
                setCancelBusy(false);
              }
            }}
          >
            {m.site.orders.cancel}
          </Button>
          {order.status === "accepted" && (
            <p className="mt-1 text-xs text-ink-muted">
              {m.site.orders.cancelWindow.replace("{t}", fmtClock(cancelLeft))}
            </p>
          )}
          {cancelError && <p className="mt-1 text-sm text-danger">{cancelError}</p>}
        </div>
      )}

      {/* الفاتورة: سجلُّ الواقعة — يفتحها الزبون ويطبعها متى شاء */}
      {showInvoice && (
        <div className="mb-6">
          <Invoice order={order} />
        </div>
      )}

      {/* خط التقدم الحي */}
      {/* **الرحلةُ بأوقاتها لا بترتيبها وحده.**

          «قُبل الطلب» تقول أنه وقع، **و«قُبل ٠٩:٤٠» تقول متى** — ومنها يعرف
          الزبونُ أين طال الانتظار: أعند المطعم أم في الطريق. وهو أوّلُ ما
          يسأل عنه حين يتأخّر، وأوّلُ ما كانت الشاشةُ تسكت عنه. */}
      {!failed && (
        <section className="mb-6 rounded-card border border-line bg-surface p-4">
          <Timeline
            nodes={FLOW.map((st, i) => ({
              id: st,
              state:
                stepIdx > i || order.status === "delivered"
                  ? ("done" as const)
                  : stepIdx === i
                    ? ("current" as const)
                    : ("todo" as const),
              icon: stepIdx > i || order.status === "delivered" ? IconCheck : undefined,
              tone: st === "delivered" ? ("success" as const) : ("primary" as const),
              title: STATUS_LABELS[st],
              trailing: STEP_TIME[st]?.(order) ?? undefined,
            }))}
          />
        </section>
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
                {it.name} ×{fmtNum(it.qty)}
              </span>
              <span className="text-ink-muted">{fmtNum(it.unit_price * it.qty)}</span>
            </li>
          ))}
        </ul>
        <div className="mt-2 flex justify-between border-t border-line pt-2 font-bold">
          <span>{m.site.cart.total}</span>
          <span className="text-primary-dark">
            {fmtNum(order.total)} {m.common.currency}
          </span>
        </div>
        {order.wallet_paid > 0 && (
          <p className="mt-1 text-xs text-ink-muted">
            {m.orders.payment.wallet}: {fmtNum(order.wallet_paid)} — {m.orders.payment.cash}:{" "}
            {fmtNum(order.cash_due)}
          </p>
        )}
        <p className="mt-2 flex items-center gap-1.5 text-xs text-ink-muted"><IconLocation size={13} />{order.address_text}</p>
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
