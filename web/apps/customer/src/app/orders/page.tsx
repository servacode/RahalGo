"use client";

/** طلباتي: السجل الكامل مع حالة كل طلب. */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime, fmtClock } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Card,
  Modal,
  Invoice,
  OrderTrack,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  useLiveRefresh,
  IconOrder,
  IconStar,
  Stars,
  IconPrint,
  IconMoto,
  IconLocation,
  IconSupport,
  IconCheck,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";
import RatingModal from "@/components/RatingModal";
import ComplaintModal from "@/components/ComplaintModal";
import { etaText, hasEta } from "@/lib/eta";

const m = getMessages(defaultLocale);
const STATUS_LABELS: Record<string, string> = m.orders.status;

/** رسالةُ الخطأ من مفتاح الخادم — **لا نصَّ إنجليزيّ يصل الزبون.** */
function errText(e: unknown): string {
  const key = e instanceof ApiError ? ((e.body.message_key ?? "").split(".").pop() ?? "") : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

const VARIANT: Record<string, "warning" | "primary" | "success" | "danger" | "neutral"> = {
  pending: "warning",
  delivered: "success",
  rejected: "danger",
  cancelled: "danger",
  failed: "danger",
  refunded: "neutral",
};

interface OrderLineOption {
  id?: string;
  group: string;
  name: string;
  price_delta: number;
}

interface Order {
  id: string;
  number: number;
  // **ولا حقلَ متجرٍ هنا** — الخادمُ يمسح الاسمَ والمعرّفَ والشعار قبل الإرسال
  // (`customer_privacy.go`). **وحقلٌ في النوع يُغري بعرضه يوماً.**
  items_count: number;
  items_preview: string;
  items?: {
    menu_item_id: string | null;
    name: string;
    unit_price: number;
    qty: number;
    note: string;
    options: OrderLineOption[];
  }[];
  status: string;
  total: number;
  created_at: string;
  /** حقولُ الوقت المتوقَّع — **يرسلها الخادمُ أصلاً وكان النوعُ يتجاهلها.** */
  accepted_at?: string | null;
  ready_at?: string | null;
  prep_minutes?: number | null;
  delivery_estimate_min?: number;
  closed_at?: string | null;
  /** ما بقي من مهلة الإلغاء بالثواني — و`-1` تعني «بلا مهلة» (قبل القبول). */
  cancel_seconds_left?: number;
  /** ما يحتاجه **الطلبُ السريع**: العنوانُ نفسُه وطريقةُ الدفع نفسُها. */
  address_text?: string;
  lat?: number;
  lng?: number;
  payment_method?: string;
  /**
   * **لماذا انتهى قبل أن يصل.**
   *
   * الخادمُ يُلزم بالسبب في الرفض والإلغاء والفشل والاسترجاع
   * (`requiresReason` في `admin_orders_handlers.go`) **ويحفظه ويرسله** —
   * **وكانت البطاقةُ وحدَها لا تقرؤه.** فيقرأ الزبونُ «مرفوض» بلا كلمة،
   * ويبقى السببُ مكتوباً في قاعدةٍ لا يراها.
   *
   * (ملاحظةُ المالك ٢٠٢٦-٠٨-٠٣: «لا يوجد سبب واضح للرفض».)
   *
   * **وحقلٌ واحدٌ يكفي للأربع**: `transitions.go` يكتب التعليلَ في
   * `cancel_reason` عند كلّ نهايةٍ غير التسليم — رفضاً وإلغاءً وفشلاً
   * واسترجاعاً.
   */
  cancel_reason?: string;
  /**
   * **ورمزُ التعذّر يسدّ ما يتركه النصّ.**
   *
   * صار التفصيلُ الحرُّ اختيارياً للسائق — «القائمةُ تُصنّف والنصُّ يشرح»
   * (`failreasons.go`) — **فطلبٌ تعذّر بلا تفصيلٍ يبقى `cancel_reason` فيه
   * فارغاً**، فيقرأ الزبونُ «تعذّر التسليم» بلا كلمة وقد عاد بلا طعامه.
   *
   * **والرمزُ مصنَّفٌ فيُترجَم** — ولا يُعرض خاماً.
   */
  fail_reason?: string;
}

interface RateInfo {
  order_id: string;
  number: number;
  items_preview: string;
  has_driver: boolean;
  rated: boolean;
  platform_stars: number;
  driver_stars: number | null;
  comment: string;
}

/**
 * إعادة الطلب: تُبنى السلّة من أصناف طلبٍ سابق.
 *
 * وتُستثنى الأصناف التي **حُذفت من القائمة** (`menu_item_id = null`) أو التي
 * تحمل خياراً بلا معرّف — وهي طلباتٌ سُجّلت قبل أن نحفظ معرّفات الخيارات. إضافتها
 * ناقصةً تُنتج طلباً يُرفض عند الإنشاء بـ«أصناف غير صالحة»، وهو أسوأ من إخبار
 * الزبون أن صنفاً لم يعد متاحاً.
 */
function reorderLines(o: Order) {
  const lines = [];
  let skipped = 0;
  for (const it of o.items ?? []) {
    const opts = it.options ?? [];
    if (!it.menu_item_id || opts.some((x) => !x.id)) {
      skipped++;
      continue;
    }
    lines.push({
      menu_item_id: it.menu_item_id,
      name: it.name,
      price: it.unit_price,
      qty: it.qty,
      note: it.note ?? "",
      option_ids: opts.map((x) => x.id!),
      option_names: opts.map((x) => x.name),
      options_delta: opts.reduce((a, x) => a + x.price_delta, 0),
    });
  }
  return { lines, skipped };
}

export default function MyOrdersPage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [orders, setOrders] = useState<Order[] | null>(null);
  const [rateMap, setRateMap] = useState<Record<string, RateInfo>>({});
  const [rating, setRating] = useState<RateInfo | null>(null);
  /** الطلبُ الذي تُعرض فاتورتُه — **نافذةٌ لا صفحة**. */
  const [invoice, setInvoice] = useState<Order | null>(null);
  const [notice, setNotice] = useState("");
  /** الطلبُ الذي يُسأل عن تكراره — **سؤالٌ واحدٌ لا خطوات.** */
  const [again, setAgain] = useState<Order | null>(null);
  const [busy, setBusy] = useState(false);

  /**
   * **إعادةُ الطلب: طلبٌ مباشرٌ لا سلّة.**
   *
   * كانت تملأ السلّةَ وتنقل الزبونَ إلى `/cart` — **فيُعيد الخطواتِ التي أراد
   * أن يتخطّاها**: يراجع، ويختار عنواناً، ويختار دفعاً، ويضغط. **وأربعُ ضغطاتٍ
   * لطلبٍ سبق أن طلبه.**
   *
   * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «ليس المقصود ع سلّة، يجب أن يقوم بطلب سريع
   * مباشر وليس إعادة الخطوات».
   *
   * **والعنوانُ والدفعُ من الطلب السابق** — هما ما اختاره حين طلبه أوّلَ مرّة.
   *
   * **ويُسأل مرّةً واحدة قبل الإنشاء.** لا لأنّها خطوة، **بل لأنّ ضغطةً
   * خاطئةً في قائمةٍ تُنشئ طلباً حقيقياً يُطبخ ويُوصَّل** — والزبونُ يدفع نقداً
   * عند الباب. **وسؤالٌ ثمنُه ثانية، وخطؤه ثمنُه طلب.**
   */
  async function orderAgain(o: Order) {
    const { lines, skipped } = reorderLines(o);
    if (lines.length === 0) {
      setNotice(m.site.orders.reorderNone);
      setAgain(null);
      return;
    }
    setBusy(true);
    setNotice("");
    try {
      const created = await api<{ number: number }>("/api/v1/orders", {
        method: "POST",
        body: JSON.stringify({
          items: lines.map((l) => ({
            menu_item_id: l.menu_item_id,
            qty: l.qty,
            note: l.note,
            option_ids: l.option_ids,
          })),
          address_text: o.address_text ?? "",
          lat: o.lat ?? 0,
          lng: o.lng ?? 0,
          payment_method: o.payment_method ?? "cash",
        }),
      });
      setAgain(null);
      setNotice(
        (skipped > 0 ? m.site.orders.reorderPartial.replace("{n}", fmtNum(skipped)) + " " : "") +
          m.site.orders.againDone.replace("{n}", fmtNum(created.number)),
      );
      load();
    } catch (e) {
      // **وسببُ الرفض يُقال**: متجرٌ أغلق، أو صنفٌ نفد، أو رصيدٌ لا يكفي —
      // **و«تعذّر» وحدَها تترك الزبونَ يعيد المحاولة على ما لن ينجح.**
      setNotice(errText(e));
      setAgain(null);
    } finally {
      setBusy(false);
    }
  }

  const loadRatings = useCallback(() => {
    api<RateInfo[]>("/api/v1/my/ratings")
      .then((rs) => setRateMap(Object.fromEntries(rs.map((r) => [r.order_id, r]))))
      .catch(() => undefined);
  }, []);

  /**
   * **وفشلُ الجلب لا يُعرض «لا طلبات».**
   *
   * كان `.catch(() => setOrders([]))` — **فيرى الزبونُ «لا طلباتِ لك»** حين
   * تنقطع الشبكة، **وله طلبٌ في الطريق الآن.** فيظنّ أنّه ضاع أو أنّ المنصةَ
   * ألغته، **ويتّصل بالمكتب أو يطلب من جديد.**
   *
   * **وهي العائلةُ نفسُها** التي أُصلحت في صفحات الخادم — وهذه في أكثر ما
   * يُفتح قلقاً.
   */
  const [failed, setFailed] = useState(false);
  const load = useCallback(() => {
    setFailed(false);
    api<{ orders: Order[] }>("/api/v1/my/orders?per_page=50")
      .then((d) => setOrders(d.orders))
      .catch(() => setFailed(true));
    loadRatings();
  }, [loadRatings]);

  useEffect(() => {
    if (loading) return;
    if (!isLoggedIn(user)) {
      router.replace("/login?next=/orders");
      return;
    }
    load();
  }, [user, loading, router, load]);

  useLiveRefresh(["order", "rating"], load);

  if (failed) {
    return (
      <PageContainer>
        <PageHeader icon={IconOrder} title={m.terms.orders} />
        <Alert tone="warning" title={m.errors.offline}>
          {m.errors.offlineHint}
        </Alert>
        <Button variant="secondary" onClick={load}>
          {m.common.retry}
        </Button>
      </PageContainer>
    );
  }
  if (!orders) return <LoadingState />;

  return (
    <PageContainer>
      <PageHeader icon={IconOrder} title={m.terms.orders} />
      {notice && (
        <Alert tone="warning" className="mb-3">{notice}</Alert>
      )}

      {orders.length === 0 ? (
        <EmptyState icon={IconOrder} title={m.site.orders.empty} />
      ) : (
        <>
        {/* **الجاري أوّلاً وبعنوانه — والمنتهي تحته.**

            كانت القائمةُ واحدةً مرتّبةً بالتاريخ. **ومن له طلبٌ في الطريق
            يفتح الشاشةَ ليتتبّعه** — لا ليقرأ سجلَّه، **وطلبٌ من الشهر
            الماضي في الصفّ الأوّل يجعله يبحث عن طلبه بين طلباته.**

            **والفصلُ بعنوانين لا بترتيبٍ وحدَه**: ترتيبٌ بلا عنوانٍ يُقرأ
            صدفةً، **وعنوانٌ يقول «هذا يجري الآن».**

            **ولا يُعرض العنوانان على فراغ**: من لا طلبَ جارياً له لا يرى
            «الجاري» فارغةً — وقسمٌ فارغٌ يُقرأ عطباً. */}
        {[
          { key: "live", rows: orders.filter((o) => !o.closed_at), title: m.site.orders.live },
          { key: "past", rows: orders.filter((o) => o.closed_at), title: m.site.orders.past },
        ]
          .filter((g) => g.rows.length > 0)
          .map((g) => (
        <section key={g.key} className="mb-6 last:mb-0">
          <h2 className="mb-3 text-sm font-bold text-ink-muted">{g.title}</h2>
        <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
          {g.rows.map((o) => (
            <OrderCard
              key={o.id}
              o={o}
              rate={rateMap[o.id]}
              onReorder={() => setAgain(o)}
              onRate={() => setRating(rateMap[o.id] ?? null)}
              onInvoice={() => setInvoice(o)}
              onChanged={load}
            />
          ))}
        </div>
        </section>
          ))}
        </>
      )}

      {again && (
        <Modal open title={m.site.orders.againTitle} onClose={() => setAgain(null)}>
          <p className="mb-3 text-sm text-ink-muted">{m.site.orders.againBody}</p>
          <ul className="mb-3 divide-y divide-line rounded-control bg-page/60">
            {(again.items ?? []).map((it, i) => (
              <li key={i} className="flex items-center justify-between px-3 py-2 text-sm">
                <span className="min-w-0 flex-1">
                  {it.name}
                  {it.qty > 1 && <span className="text-ink-muted" dir="ltr"> ×{fmtNum(it.qty)}</span>}
                </span>
                <span dir="ltr" className="shrink-0 tabular-nums text-ink-muted">
                  {fmtNum(it.unit_price * it.qty)}
                </span>
              </li>
            ))}
          </ul>
          {/* **العنوانُ يُعرض لا يُفترض** — من طلب إلى بيته أمسِ قد يكون اليومَ
              في عمله، **وطلبٌ يصل إلى عنوانٍ خاطئٍ خسارةٌ للجميع.** */}
          {again.address_text && (
            <p className="mb-4 flex items-start gap-2 rounded-control bg-page px-3 py-2 text-sm">
              <IconLocation size={15} className="mt-0.5 shrink-0 text-ink-muted" />
              <span className="min-w-0 flex-1">{again.address_text}</span>
            </p>
          )}
          <div className="flex gap-2">
            <Button variant="secondary" onClick={() => setAgain(null)}>
              {m.common.cancel}
            </Button>
            <Button disabled={busy} onClick={() => void orderAgain(again)}>
              {busy ? m.common.loading : m.site.orders.againConfirm}
            </Button>
          </div>
        </Modal>
      )}

      {invoice && (
        <Modal open title={m.site.orders.invoice} onClose={() => setInvoice(null)} size="lg">
          <Invoice order={invoice as never} />
        </Modal>
      )}

      {rating && (
        <RatingModal
          order={rating}
          onClose={() => setRating(null)}
          onRated={() => {
            setRating(null);
            loadRatings();
          }}
        />
      )}
    </PageContainer>
  );
}

/**
 * **مراحلُ الطلب كما يراها الزبون** — لا كما تُسمّى في المحرّك.
 *
 * المحرّكُ يعرف `assigned` و`at_pickup` و`picked_up` و`on_the_way` و
 * `at_dropoff` — **وهي شؤونٌ داخلية لا تخصّ من ينتظر طعامه.** فتُطوى في مرحلةٍ
 * واحدة: «في الطريق».
 *
 * **وخمسُ عُقَدٍ لا تسع**: بطاقةٌ في قائمة، وأسماءٌ تحت العُقَد. **وأربعٌ تُقرأ
 * بلمحة.**
 */
const TRACK = ["pending", "accepted", "preparing", "onway", "delivered"] as const;

/** الحالةُ الخام ← موضعُها على المسار. */
/**
 * **لماذا انتهى — بأوّل ما يُقال، لا بأدقّه.**
 *
 * التفصيلُ الحرُّ أدلُّ حين يُكتب («الباب مغلق ولا أحد يردّ»)، **لكنّه اختياريّ
 * للسائق**. فإن غاب بقي الرمزُ المصنَّف — **وهو يقول شيئاً ولو أقلّ**، وسطرٌ
 * ناقصٌ خيرٌ من صمتٍ أمام زبونٍ عاد بلا طعامه.
 *
 * **والرمزُ يُترجَم ولا يُعرض خاماً** — و`customer_absent` في شاشةِ زبونٍ
 * عطبٌ يُقرأ، لا معلومة.
 */
function endedReason(o: Order): string {
  const free = o.cancel_reason?.trim();
  if (free) return free;
  const code = o.fail_reason?.trim();
  if (!code) return "";
  return m.common.failReasons[code as keyof typeof m.common.failReasons] ?? "";
}

function trackIndex(status: string): number {
  switch (status) {
    case "pending":
      return 0;
    case "accepted":
      return 1;
    case "preparing":
    case "dispatching":
      return 2;
    case "assigned":
    case "at_pickup":
    case "picked_up":
    case "on_the_way":
    case "at_dropoff":
      return 3;
    case "delivered":
      return 4;
    default:
      return -1; // ملغى · مرفوض · مُخفق · مُسترجَع — **لا مسارَ لهم**
  }
}

/**
 * **بطاقةُ الطلب — قائمةٌ بذاتها.**
 *
 * قرارُ المالك (٢٠٢٦-٠٨-٠٣): «لا يوجد داعي لزرّ تفاصيل الطلب · حالة الطلب
 * تكون بنفس الكرت · بالكرت كلّ صنف وسعره في حقلٍ خاصّ مرتّب · الفاتورة تصير
 * أيقونة».
 *
 * **وكلُّ ما كان خلف ضغطةٍ صار في وجه البطاقة**: ماذا طلبتُ، وبكم، وأين هو
 * الآن. **والانتقالُ إلى صفحةٍ لقراءة سطرين ضريبةٌ تُدفع في كلّ مرّة.**
 */
function OrderCard({
  o,
  rate,
  onReorder,
  onRate,
  onInvoice,
  onChanged,
}: {
  o: Order;
  rate?: RateInfo;
  onReorder: () => void;
  onRate: () => void;
  onInvoice: () => void;
  /** يُعاد التحميلُ بعد إلغاءٍ ناجح — **البطاقةُ تُظهر النتيجة بلا تحديث.** */
  onChanged: () => void;
}) {
  /** نافذةُ الشكوى — **في البطاقة لا في صفحةٍ أخرى.** */
  const [complaining, setComplaining] = useState(false);
  /** نافذةُ تأكيد الإلغاء — **والإلغاءُ لا يُتراجع عنه.** */
  const [confirming, setConfirming] = useState(false);
  const [cancelBusy, setCancelBusy] = useState(false);
  const [cancelErr, setCancelErr] = useState("");
  /**
   * **العدّادُ يعدّ فعلاً.**
   *
   * الرقمُ من الخادم نسبيٌّ عند وصوله، **فيُثبَّت مرساه بساعة الجهاز نفسِه**
   * ويُطرح منه ما مضى — **فلا تدخل ساعةُ الخادم في الحساب**، وفارقُ الساعتين
   * لا يجعل زرّاً حيّاً يبدو منقضياً.
   */
  const [left, setLeft] = useState(0);
  useEffect(() => {
    const n = o.cancel_seconds_left ?? 0;
    if (n <= 0) {
      setLeft(0);
      return;
    }
    const anchor = Date.now();
    const tick = () => setLeft(Math.max(0, n - Math.floor((Date.now() - anchor) / 1000)));
    tick();
    const id = setInterval(tick, 1000);
    return () => clearInterval(id);
  }, [o.cancel_seconds_left]);
  const [ticketNo, setTicketNo] = useState(0);
  const at = trackIndex(o.status);
  const closed = at < 0;
  const live = !closed && o.status !== "delivered";
  const canRate = o.status === "delivered" && rate && !rate.rated;
  /** **انتهى أمرُه** — سُلّم أو أُغلق. وعندها وحدَها تُفتح الشكوى. */
  const finished = closed || o.status === "delivered";
  const items = o.items ?? [];

  return (
    <Card className="flex flex-col gap-4">
      {/* ── الترويسة: علامةُ المنصة · الرقم · الحالة ─────────────────── */}
      <div className="flex items-start gap-3">
        {/* **علامةُ المنصة لا أيقونةُ متجر.**

            الزبونُ اشترى من «رحّال غو» — **واسمُ المتجر محجوبٌ عمداً**، فأيقونةُ
            متجرٍ عامّة تقول شيئاً لا نقوله. والعلامةُ هي نفسُها في الشريط
            العلويّ وفي الفاتورة: **حرفٌ أبيضُ على أزرق وطريقٌ برتقاليّ تحته.** */}
        <span className="relative flex h-12 w-12 shrink-0 items-center justify-center overflow-hidden rounded-control bg-primary text-xl font-bold text-on-solid">
          {m.terms.brandInitial}
          <span aria-hidden className="absolute inset-x-0 bottom-0 h-1.5 bg-accent" />
        </span>

        <div className="min-w-0 flex-1">
          <p className="truncate font-bold leading-tight">{m.site.orders.fromPlatform}</p>
          {/* **الوقتُ تحت الاسم** — سطرٌ خافتٌ يُقرأ حين يُبحث عنه ولا يزاحم. */}
          <p className="mt-0.5 text-xs text-ink-muted" dir="ltr">
            {fmtDateTime(o.created_at)}
          </p>
        </div>

        <div className="flex shrink-0 flex-col items-end gap-1.5">
          {/* **الرقمُ بحقلٍ خاصّ** — هو ما يُقال في الهاتف حين يُسأل عن طلب،
              **ورقمٌ خافتٌ بجانب عنوانٍ يضيع.** */}
          <span
            dir="ltr"
            /* **رقمُ الطلب برتقاليّ** — هو ما يُقال في الهاتف حين يُسأل عنه،
               **فيُلمح في البطاقة قبل أن يُبحث عنه.** */
            className="rounded-control bg-accent px-2.5 py-1 text-sm font-bold tabular-nums text-on-accent"
          >
            #{fmtRef(o.number)}
          </span>
          <Badge variant={VARIANT[o.status] ?? "primary"}>
            {STATUS_LABELS[o.status] ?? o.status}
          </Badge>
        </div>
      </div>

      {/* ── الأصناف: كلٌّ في سطره وسعرُه أمامه ────────────────────────── */}
      {items.length > 0 ? (
        <ul className="divide-y divide-line rounded-control bg-page/60">
          {items.map((it, i) => (
            <li key={i} className="flex items-start gap-3 px-3 py-2 text-sm">
              <span className="min-w-0 flex-1">
                {it.name}
                {it.qty > 1 && (
                  <span className="text-ink-muted" dir="ltr">
                    {" "}
                    ×{fmtNum(it.qty)}
                  </span>
                )}
                {/* الخياراتُ تحت الاسم — **هي ما يُميّز طلباً عن طلب.** */}
                {it.options?.length > 0 && (
                  <span className="block text-xs text-ink-muted">
                    {it.options.map((x) => x.name).join(m.common.listSep)}
                  </span>
                )}
              </span>
              <span dir="ltr" className="shrink-0 tabular-nums text-ink-muted">
                {fmtNum(it.unit_price * it.qty)}
              </span>
            </li>
          ))}
        </ul>
      ) : (
        <p className="text-sm text-ink-muted">{o.items_preview || m.site.orders.noItems}</p>
      )}

      {/* ── الإجمالي ──────────────────────────────────────────────────── */}
      <div className="flex items-center justify-between border-t border-line pt-3">
        <span className="text-sm text-ink-muted">{m.site.orders.statTotal}</span>
        <span dir="ltr" className="text-lg font-bold tabular-nums">
          {fmtNum(o.total)}{" "}
          <span className="text-xs font-normal text-ink-muted">{m.common.currency}</span>
        </span>
      </div>

      {/* ── أين هو الآن ───────────────────────────────────────────────── */}
      {closed ? (
        /* **ولا مسارَ لما انتهى قبل أن يصل.** شريطٌ يقف في منتصفه يُقرأ
           «عالق» لا «انتهى» — فيُقال بالحرف. */
        <div className="space-y-1.5 rounded-control bg-page px-3 py-2 text-center">
          <p className="text-sm text-ink-muted">
            {m.site.orders.trackClosed.replace("{s}", STATUS_LABELS[o.status] ?? o.status)}
          </p>
          {/* **والسببُ يُقال لصاحبه.**

              الخادمُ يمنع الرفضَ والإلغاءَ والفشلَ بلا تعليل (`requiresReason`)،
              **ثمّ كانت البطاقةُ تبتلع ما كُتب**: يقرأ الزبونُ «مرفوض» ولا يعرف
              أنفدت الأصنافُ أم أُغلق المتجر أم أخطأ عنوانُه — **فيعيد الطلبَ
              نفسَه فيُرفض مرّةً ثانية.**

              وهو ظاهرٌ للإدارة وللمندوب منذ البداية — **وللزبون وحدَه لا.**
              (ملاحظةُ المالك ٢٠٢٦-٠٨-٠٣: «لا يوجد سبب واضح للرفض».) */}
          {endedReason(o) && (
            <p className="text-sm">
              <span className="text-ink-muted">{m.site.orders.endedReason}: </span>
              <span className="font-medium text-danger">{endedReason(o)}</span>
            </p>
          )}
        </div>
      ) : (
        <div className="space-y-2">
          <OrderTrack
            stages={TRACK.map((id) => ({ id, label: m.site.orders.track[id] }))}
            current={at}
            vehicle={IconMoto}
            live={live}
          />
          {/* **الوقتُ المتوقَّع تحت المسار مباشرةً.**

              «قيد التحضير» وحدَها لا تقول عشرَ دقائقَ أم ساعة. **والمسارُ يقول
              أين، والوقتُ يقول متى** — ولا يُقرأ أحدُهما بلا الآخر.

              وحسابُه مشتركٌ مع صفحة التتبّع (`lib/eta.ts`) — **ولو نُسخ لَافترقا
              يوماً، فتقول البطاقةُ عشرين وتقول الصفحةُ خمساً.** */}
          {hasEta(o, o.status, closed) && (
            <p className="flex items-center justify-center gap-1.5 text-xs font-medium text-primary-dark">
              <IconCheck size={13} strokeWidth={3} />
              {m.site.orders.eta}: {etaText(o)}
            </p>
          )}
        </div>
      )}

      {/* ── الإلغاء: نافذةٌ تُرى وهي تنقضي ───────────────────────────── */}
      {(o.status === "pending" || (o.status === "accepted" && left > 0)) && (
        <div>
          <Button
            variant="danger"
            disabled={cancelBusy}
            onClick={() => setConfirming(true)}
            className="w-full !py-2"
          >
            {m.site.orders.cancel}
          </Button>
          {o.status === "accepted" && (
            <p className="mt-1 text-center text-xs text-ink-muted">
              {m.site.orders.cancelWindow.replace("{t}", fmtClock(left))}
            </p>
          )}
          {cancelErr && <p className="mt-1 text-sm text-danger">{cancelErr}</p>}

          <Modal
            open={confirming}
            onClose={() => setConfirming(false)}
            title={m.site.orders.cancelConfirmTitle}
          >
            <p className="mb-4 text-sm text-ink-muted">{m.site.orders.cancelConfirmBody}</p>
            <div className="flex gap-2">
              {/* **والتراجعُ أوّلاً** — من فتح النافذةَ بالخطأ يجد المخرجَ في
                  موضع الزرّ الذي اعتاده. */}
              <Button variant="secondary" onClick={() => setConfirming(false)}>
                {m.site.orders.cancelConfirmNo}
              </Button>
              <Button
                variant="danger"
                disabled={cancelBusy}
                onClick={async () => {
                  setCancelBusy(true);
                  setCancelErr("");
                  try {
                    await api(`/api/v1/orders/${o.id}/cancel`, {
                      method: "POST",
                      body: JSON.stringify({ note: "" }),
                    });
                    setConfirming(false);
                    onChanged();
                  } catch (e) {
                    setConfirming(false);
                    setCancelErr(errText(e));
                  } finally {
                    setCancelBusy(false);
                  }
                }}
              >
                {m.site.orders.cancelConfirmYes}
              </Button>
            </div>
          </Modal>
        </div>
      )}

      {/* ── الأفعال ───────────────────────────────────────────────────── */}
      <div className="flex items-center gap-2">
        {items.length > 0 && (
          /* **الفعلُ الأكثرُ تكراراً يلبس لونَ العلامة** — ومن طلب مرّةً
             يطلب ثانية، **وزرٌّ باهتٌ لأكثر ما يُضغط يُبطئ ما يجب أن يسرع.** */
          <Button
            onClick={onReorder}
            className="flex flex-1 items-center justify-center gap-1.5 !py-1.5"
          >
            <IconOrder size={15} />
            {m.site.orders.reorder}
          </Button>
        )}
        {canRate && (
          <Button onClick={onRate} className="flex flex-1 items-center justify-center gap-1.5 !py-1.5">
            <IconStar size={15} />
            {m.site.rating.rateOrder}
          </Button>
        )}
        {o.status === "delivered" && rate?.rated && (
          <span className="flex flex-1 items-center justify-center gap-2 rounded-control bg-page px-3 py-1.5">
            <span className="text-xs text-ink-muted">{m.site.rating.merchant}</span>
            <Stars value={rate.platform_stars} size="sm" />
          </span>
        )}
        {/* **الفاتورةُ أيقونة** — يحتاجها من يطبع، ولا يحتاجها الباقون.
            **وزرٌّ بعرض الثلث لفعلٍ نادرٍ يزاحم فعلاً يوميّاً.** */}
        <button
          type="button"
          onClick={onInvoice}
          aria-label={m.site.orders.invoice}
          title={m.site.orders.invoice}
          /* **الطباعةُ خضراء والشكوى حمراء** — فعلان متجاوران ومعناهما متضادّ:
             أحدُهما يحفظ والآخر يشتكي. **ولونٌ واحدٌ لهما يجعل الضغطةَ قرعةً**،
             والعينُ تفرّق بالألوان قبل أن تقرأ الأيقونات.
             (قرارُ المالك ٢٠٢٦-٠٨-٠٣) */
          className="flex h-9 w-9 shrink-0 items-center justify-center rounded-control bg-success-solid text-on-solid transition-opacity hover:opacity-90"
        >
          <IconPrint size={16} />
        </button>

        {/* **وبابُ الشكوى في البطاقة.**

            كان خلف صفحة التفاصيل — **وقد أُغلق بابُها.** ومن وقع له خطأٌ في
            طلبه لا يبحث عن مكانِ الشكوى، **ومن لم يجدها في ثانيتين يتّصل أو
            يسكت** — والسكوتُ أسوأ: نخسر الزبونَ ولا نعرف لماذا.

            **ولا يظهر إلّا بعد أن ينتهي الطلب**: شكوى على طلبٍ في الطريق
            شكوى على ما لم يقع بعد. */}
        {finished &&
          (ticketNo > 0 ? (
            <span
              title={m.site.complaint.opened.replace("{n}", fmtNum(ticketNo))}
              className="flex h-9 shrink-0 items-center rounded-control bg-page px-2 text-xs text-ink-muted"
              dir="ltr"
            >
              #{fmtRef(ticketNo)}
            </span>
          ) : (
            <button
              type="button"
              onClick={() => setComplaining(true)}
              aria-label={m.site.complaint.open}
              title={m.site.complaint.open}
              /* **أيقونةُ الشكوى هي أيقونتُها في اللوحة كلِّها** — طوقُ
                 النجاة: قسمُ التذاكر، ورأسُ صفحتها، وبطاقاتُها.

                 وكانت مثلّثَ تحذير — **وهو يُقرأ «في هذه البطاقة خطأ» لا
                 «اشتكِ من هذا الطلب»**، فيقلق من لا شكوى له.

                 **وحدودٌ حمراءُ وخلفيةٌ خفيفةٌ تجعله يُرى** بجانب أيقونة
                 الطباعة الرمادية: الفعلان مختلفان، **فلا يُلبسان لباساً
                 واحداً.** */
              className="flex h-9 shrink-0 items-center gap-1.5 rounded-control bg-danger-solid px-2.5 text-xs font-medium text-on-solid transition-opacity hover:opacity-90"
            >
              <IconSupport size={16} />
              {m.site.complaint.short}
            </button>
          ))}
      </div>

      {complaining && (
        <ComplaintModal
          orderId={o.id}
          orderNumber={o.number}
          onClose={() => setComplaining(false)}
          onOpened={(n) => {
            setTicketNo(n);
            setComplaining(false);
          }}
        />
      )}
    </Card>
  );
}
