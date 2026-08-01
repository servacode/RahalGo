"use client";

import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtTime } from "@rahalgo/i18n";
import {
  IconNote,
  IconEdit,
  Stars,
  useLiveEvent,
  useLiveStatus,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  FormSection,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconOrder,
  IconSearch,
  IconUser,
  IconStore,
  IconWallet,
  IconDriver,
  IconStatus,
  IconLocation,
  IconStar,
  IconBalance,
} from "@rahalgo/ui";
import { api, ApiError, type AuthUser } from "@/lib/api";

const m = getMessages(defaultLocale);

// ---------- الأنواع ----------

/** رسالةُ المتجر كما يبنيها الخادم — نصّاً ورابطاً معاً، فلا يفترقان. */
type MerchantMessage = {
  text: string;
  phone: string;
  /** رابطُ واتساب جاهزاً — فارغٌ إن كان رقمُ المتجر غيرَ صالح */
  wa_link: string;
  /** أمُهيَّأةٌ بوّابةُ الرسائل النصّية؟ */
  sms_ready: boolean;
};

interface DriverRow {
  id: string;
  full_name: string;
  phone: string;
  status: string;
  on_shift: boolean;
  open_orders: number;
}

interface OrderRow {
  id: string;
  number: number;
  customer_phone: string;
  customer_name: string;
  merchant_name: string;
  driver_phone: string | null;
  driver_name: string | null;
  /** متى حُوِّل الطلب إلى المتجر — فارغٌ يعني لم يُحوَّل بعد */
  sent_to_merchant_at: string | null;
  /** أجرُ السائق — تقديرٌ قبل التسليم وواقعٌ بعده، من مصدر الحساب نفسه */
  driver_fee: number;
  status: string;
  payment_method: "cash" | "wallet";
  subtotal: number;
  delivery_fee: number;
  discount: number;
  total: number;
  wallet_paid: number;
  cash_due: number;
  promo_code: string | null;
  address_text: string;
  notes: string;
  cancel_reason: string;
  created_at: string;
  items?: {
    id: string;
    name: string;
    unit_price: number;
    qty: number;
    note: string;
    options: { group: string; name: string; price_delta: number }[];
  }[];
  events?: { from_status: string; to_status: string; note: string; created_at: string }[];
  rating?: {
    merchant_stars: number;
    driver_stars: number | null;
    comment: string;
    created_at: string;
  };
}

interface OrderPage {
  orders: OrderRow[];
  total: number;
  page: number;
  per_page: number;
}

interface Alert {
  order_id: string;
  number: number;
  status: string;
  merchant_name: string;
  customer_phone: string;
  reason: "no_accept" | "no_driver" | "too_long";
  minutes: number;
}

const STATUS_LABELS: Record<string, string> = m.orders.status;
const ACTION_LABELS: Record<string, string> = m.admin.ordersPage.actions;
const PAYMENT_LABELS: Record<string, string> = m.orders.payment;

const STATUS_VARIANT: Record<string, "warning" | "primary" | "success" | "danger" | "neutral"> = {
  pending: "warning",
  accepted: "primary",
  preparing: "primary",
  dispatching: "warning",
  assigned: "primary",
  at_pickup: "primary",
  picked_up: "primary",
  on_the_way: "primary",
  at_dropoff: "primary",
  delivered: "success",
  rejected: "danger",
  cancelled: "danger",
  failed: "danger",
  refunded: "neutral",
};

/**
 * الأفعالُ الهدّامة — تُطلب لها ضغطةٌ ثانية على البطاقة.
 *
 * كلُّها تُنهي الطلب أو تعكس مالاً: الرفضُ والإلغاءُ يُرجعان ما دُفع، والفشلُ
 * يُغلق بلا تسليم، والاسترجاعُ يعكس تسويةً تمّت. **وما لا يُستدرَك لا يُترك
 * لضغطةٍ واحدة.**
 */
const DESTRUCTIVE = new Set(["rejected", "cancelled", "failed", "refunded"]);

// أزرار الانتقال المتاحة للعمليات/الأدمن حسب الحالة (مرآة لخارطة الخادم)
const OPS_NEXT: Record<string, string[]> = {
  pending: ["accepted", "rejected", "cancelled"],
  accepted: ["preparing", "cancelled"],
  preparing: ["cancelled"], // + إسناد سائق
  dispatching: ["cancelled"], // + إسناد سائق
  assigned: ["at_pickup", "cancelled"],
  at_pickup: ["picked_up", "cancelled"],
  picked_up: ["on_the_way"],
  on_the_way: ["at_dropoff"],
  at_dropoff: ["delivered", "failed"],
  delivered: ["refunded"],
};

/**
 * ما تملكه العملياتُ فعلاً بعد حساب الوضع — مرآةُ `orders/modes.go`.
 *
 * **والخادمُ هو الحَكَم**: هذه الدالة تُخفي ما سيُردّ، فلا يضغط الموظّفُ زرّاً
 * يعتذر. ولو انحرفت عن الخادم لظهر زرٌّ لا يعمل — **وهو أخفُّ ضرراً من زرٍّ
 * يعمل ولا يجب أن يعمل**.
 */
function opsNext(status: string, selfManage: boolean, hasDriver: boolean): string[] {
  let next = OPS_NEXT[status] ?? [];
  // **المتجر يدير — والمنصةُ عينٌ لا يد**: القبولُ والرفضُ اختصاصُه.
  if (selfManage && status === "pending") {
    next = next.filter((t) => t !== "accepted" && t !== "rejected");
  }
  // **وبعد أن يمسكه سائق، لا تُلغيه العمليات**: هو عند الباب يرى ما لا تراه
  // غرفةُ العمليات، وإلغاؤها من بعيدٍ يُلغي طلباً ربّما استُلم فعلاً.
  if (!selfManage && hasDriver) {
    next = next.filter((t) => t !== "cancelled");
  }
  return next;
}

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}
function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

// ---------- الشاشة الرئيسية ----------

export default function OrdersPage() {
  const params = useSearchParams();
  const initialQ = params.get("q") ?? "";
  const [data, setData] = useState<OrderPage | null>(null);
  const [status, setStatus] = useState("");
  const [query, setQuery] = useState(initialQ);
  const [openOnly, setOpenOnly] = useState(initialQ === "");
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [alerts, setAlerts] = useState<Alert[]>([]);
  /**
   * أيُدير المتجرُ طلباته بنفسه؟
   *
   * **`null` تعني «لم نعرف بعد»** لا «المنصة تدير»: لو بدأناها `false` لظهر
   * زرُّ الإرسال لحظةً في كل بطاقة ثم اختفى — **ووميضُ زرٍّ كاذب يُفقد الثقة
   * بكل زرّ**.
   */
  const [selfManage, setSelfManage] = useState<boolean | null>(null);

  useEffect(() => {
    api<{ key: string; value: unknown }[]>("/api/v1/admin/settings")
      .then((all) => {
        const row = all.find((x) => x.key === "merchants.self_manage_orders");
        setSelfManage(row ? row.value === true : true);
      })
      .catch(() => setSelfManage(true));
  }, []);
  const [view, setView] = useViewMode("orders");

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        status,
        query,
        open: openOnly ? "1" : "",
        page: String(page),
        per_page: "12",
      });
      setData(await api<OrderPage>(`/api/v1/admin/orders?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [status, query, openOnly, page]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  // البث الحي: تحديثات الطلبات تعيد التحميل فوراً، والتنبيهات تُستبدل مباشرة
  useLiveEvent((event) => {
    if (event.type === "order") void load();
    if (event.type === "alerts") setAlerts((event.alerts as Alert[]) ?? []);
  });
  const liveConnected = useLiveStatus();
  useEffect(() => {
    api<Alert[]>("/api/v1/admin/orders/alerts").then(setAlerts).catch(() => undefined);
  }, []);

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  const columns: DataColumn<OrderRow>[] = [
    {
      id: "number",
      header: m.admin.ordersPage.number,
      icon: <IconOrder />,
      primary: true,
      cell: (o) => <span className="font-bold">#{o.number}</span>,
    },
    {
      id: "customer",
      header: m.admin.ordersPage.customer,
      icon: <IconUser />,
      primary: true,
      cell: (o) => (
        <span>
          {o.customer_name || "—"}{" "}
          <span dir="ltr" className="text-xs text-ink-muted">
            {o.customer_phone}
          </span>
        </span>
      ),
    },
    {
      id: "merchant",
      header: m.admin.ordersPage.merchant,
      icon: <IconStore />,
      cell: (o) => o.merchant_name,
    },
    {
      // **الأصناف على البطاقة لا خلف «التفاصيل»**: «ماذا طلب؟» أوّلُ ما تسأله
      // غرفةُ العمليات، وكان يلزمها فتحُ نافذةٍ لكل طلب — وهي تنظر إلى عشرين.
      id: "items",
      header: m.admin.ordersPage.itemsSection,
      icon: <IconOrder />,
      // بعرض البطاقة: قائمةٌ تُقرأ سطراً سطراً لا تُحشَر في خانةٍ ضيّقة
      block: true,
      cell: (o) => (
        <ul className="space-y-1.5">
          {(o.items ?? []).map((it) => (
            <li key={it.id} className="flex items-start gap-2">
              <span className="mt-1.5 h-1.5 w-1.5 shrink-0 rounded-badge bg-primary" />
              <span className="min-w-0">
                <span className="font-medium">{it.name}</span>
                <span className="font-bold text-primary-dark"> ×{fmtNum(it.qty)}</span>
                {it.options && it.options.length > 0 && (
                  <span className="block text-xs text-ink-muted">
                    {it.options.map((x) => x.name).join(m.common.listSeparator)}
                  </span>
                )}
                {it.note && (
                  <span className="block text-xs text-accent-dark">
                    <IconEdit size={11} className="inline align-[-1px]" /> {it.note}
                  </span>
                )}
              </span>
            </li>
          ))}
          {(o.items ?? []).length === 0 && <li className="text-ink-muted">—</li>}
        </ul>
      ),
    },
    {
      // **قيمة البضاعة وحدها**: هي ما يخصّ المتجر، ورسمُ التوصيل شأنٌ آخر
      // لصاحبٍ آخر. وجمعُهما في رقمٍ واحد يُخفي أين يذهب المال.
      id: "goods",
      header: m.admin.ordersPage.goodsValue,
      icon: <IconBalance />,
      cell: (o) => (
        <span className="font-medium">
          {fmtNum(o.subtotal)} {m.common.currency}
        </span>
      ),
    },
    {
      // **السائق وأجرُه — أو أجرةُ التوصيل قبل أن يُسنَد أحد.**
      // الطلبُ يولد بلا سائق، والخانةُ الفارغة لا تقول شيئاً: فيُعرض ما يُدفع
      // عن التوصيل حتى يُعرف من سيقبضه.
      id: "driver",
      header: m.admin.ordersPage.driver,
      icon: <IconDriver />,
      cell: (o) =>
        o.driver_name || o.driver_phone ? (
          <span>
            {o.driver_name || o.driver_phone}
            <span className="block text-xs text-ink-muted">
              {m.admin.ordersPage.driverFee}: {fmtNum(o.driver_fee)} {m.common.currency}
            </span>
          </span>
        ) : (
          <span className="text-ink-muted">
            {m.admin.ordersPage.deliveryFee}: {fmtNum(o.delivery_fee)} {m.common.currency}
          </span>
        ),
    },
    {
      id: "note",
      header: m.admin.ordersPage.customerNote,
      icon: <IconNote />,
      block: true,
      cell: (o) =>
        o.notes ? (
          <span className="text-xs text-accent-dark">{o.notes}</span>
        ) : (
          <span className="text-ink-muted">—</span>
        ),
    },
    {
      id: "total",
      header: m.admin.ordersPage.total,
      icon: <IconWallet />,
      cell: (o) => (
        <span>
          <span className="font-bold text-primary-dark">{fmtNum(o.total)}</span>{" "}
          <span className="text-xs text-ink-muted">{PAYMENT_LABELS[o.payment_method]}</span>
        </span>
      ),
    },
    {
      id: "status",
      header: m.admin.ordersPage.statusCol,
      icon: <IconStatus />,
      cell: (o) => (
        <Badge variant={STATUS_VARIANT[o.status] ?? "neutral"}>{STATUS_LABELS[o.status]}</Badge>
      ),
    },
    {
      id: "time",
      header: m.admin.ordersPage.time,
      cell: (o) =>
        fmtTime(o.created_at),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <IconOrder className="text-primary" />
          {m.admin.ordersPage.title}
          <span
            className={`flex items-center gap-1.5 rounded-badge px-2.5 py-1 text-xs font-medium ${
              liveConnected ? "bg-success/10 text-success" : "bg-danger/10 text-danger"
            }`}
          >
            <span
              className={`h-2 w-2 rounded-badge ${liveConnected ? "animate-pulse bg-success" : "bg-danger"}`}
            />
            {liveConnected ? m.admin.ordersPage.live : m.admin.ordersPage.liveOff}
          </span>
        </h1>

      </div>

      {/* تنبيهات التصعيد */}
      {alerts.length > 0 && (
        <div className="mb-4 rounded-card border-2 border-danger/50 bg-danger/5 p-4">
          <p className="mb-2 flex items-center gap-2 font-bold text-danger">
            <span className="h-2.5 w-2.5 animate-pulse rounded-badge bg-danger" />
            {m.admin.ordersPage.alertsTitle} ({alerts.length})
          </p>
          <ul className="space-y-1.5">
            {alerts.map((a) => (
              <li key={a.order_id + a.reason} className="flex flex-wrap items-center gap-2 text-sm">
                {/* الإنذارُ يجلب طلبَه إلى القائمة بدل أن يفتح نافذة:
                    **البطاقة نفسها صارت تحمل كل ما يُقرَّر به** — والنافذة
                    كانت تُخفي بقيّة الطلبات وهي مفتوحة. */}
                <button
                  onClick={() => {
                    setQuery(String(a.number));
                    setOpenOnly(false);
                    setPage(1);
                  }}
                  className="font-bold text-danger underline-offset-2 hover:underline"
                >
                  #{a.number}
                </button>
                <Badge variant="danger">{m.admin.ordersPage.alertReasons[a.reason]}</Badge>
                <span>{a.merchant_name}</span>
                <span dir="ltr" className="text-xs text-ink-muted">{a.customer_phone}</span>
                <span className="text-xs text-ink-muted">
                  {m.admin.ordersPage.sinceMinutes.replace("{m}", String(a.minutes))}
                </span>
                <Badge variant="warning">{STATUS_LABELS[a.status]}</Badge>
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="mb-4 flex flex-wrap items-center gap-3">
        <div className="w-64">
          <Input
            icon={<IconSearch />}
            placeholder={m.admin.ordersPage.searchPlaceholder}
            value={query}
            onChange={(e) => {
              setQuery(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <div className="w-44">
          <Select
            value={status}
            onChange={(e) => {
              setStatus(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{m.admin.ordersPage.allStatuses}</option>
            {Object.entries(STATUS_LABELS).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </Select>
        </div>
        <label className="flex cursor-pointer items-center gap-2 text-sm">
          <input
            type="checkbox"
            checked={openOnly}
            onChange={(e) => {
              setOpenOnly(e.target.checked);
              setPage(1);
            }}
            className="h-4 w-4 accent-primary"
          />
          {m.admin.ordersPage.openOnly}
        </label>
        <div className="ms-auto">
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        </div>
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <DataView
        items={data?.orders ?? []}
        getKey={(o) => o.id}
        columns={columns}
        view={view}
        empty={m.admin.ordersPage.empty}
        actions={(o) => (
          <OrderActions o={o} onChanged={load} selfManage={selfManage !== false} />
        )}
      />

      {data && (
        <div className="mt-4 flex items-center justify-between text-sm text-ink-muted">
          <span>{m.admin.users.totalCount.replace("{count}", String(data.total))}</span>
          <div className="flex items-center gap-2">
            <Button variant="secondary" disabled={page <= 1} onClick={() => setPage(page - 1)}>
              {m.admin.users.prev}
            </Button>
            <span>
              {page} / {totalPages}
            </span>
            <Button
              variant="secondary"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              {m.admin.users.next}
            </Button>
          </div>
        </div>
      )}

    </div>
  );
}

// ---------- تفاصيل الطلب ----------

/**
 * أزرارُ الفعل على البطاقة — لا خلف «التفاصيل».
 *
 * كانت كلُّ حركةٍ تكلّف: فتحُ النافذة ← انتظارُ تحميلها ← الفعل ← الإغلاق ←
 * **البحث عن موضعك في القائمة من جديد**. وفي ساعة ذروةٍ فيها عشرون طلباً هذه
 * عشرون رحلةَ ذهابٍ وإياب — والنافذةُ تُخفي بقيّة الطلبات وهي مفتوحة، فيعمل
 * الموظّف أعمى عمّا يجري.
 *
 * **والهدّامةُ لا تُنفَّذ بضغطةٍ واحدة**: رفضٌ أو إلغاءٌ في صفٍّ مزدحم يُتلف
 * طلبَ زبونٍ بإصبعٍ زلّ. فتُطلب ضغطةٌ ثانية تؤكّد — **تأكيدٌ في مكانه أخفُّ من
 * نافذةٍ تُفتح وتُغلق**، وأصدقُ من ثقةٍ في دقّة الإصبع.
 *
 * **وإسنادُ السائق يبقى في التفاصيل**: يحتاج قائمةَ اختيارٍ لا زرّاً.
 */
function OrderActions({
  o,
  onChanged,
  selfManage,
}: {
  o: OrderRow;
  onChanged: () => void;
  /** حين تكون `false` تُدير المنصةُ الطلبات وتُرسلها للمتجر على واتساب */
  selfManage: boolean;
}) {
  const [busy, setBusy] = useState("");
  /** الفعلُ الهدّام المفتوح الآن — يُطلب سببُه قبل تنفيذه */
  const [asking, setAsking] = useState("");
  /** قائمةُ السائقين مفتوحةٌ للإسناد اليدوي */
  const [assigning, setAssigning] = useState(false);
  const [drivers, setDrivers] = useState<DriverRow[]>([]);
  /** معاينةُ الرسالة قبل تحويلها — ومعها ما يلزم لإرسالها */
  const [preview, setPreview] = useState<MerchantMessage | null>(null);
  const [reason, setReason] = useState("");
  const [err, setErr] = useState("");

  // **ما تملكه العملياتُ بعد حساب الوضع** — لا الخريطةُ الخام.
  const next = opsNext(o.status, selfManage, o.driver_name !== null);

  // **الإسنادُ اليدوي مخرجٌ لا طريق.** السائقون يلتقطون من الطابور بأنفسهم
  // (تطبيق :3005)، وهذا لمن لم يلتقطه أحد. ولذلك يُجلب السائقون **عند فتح
  // القائمة** لا مع كل بطاقة: عشرون بطاقةً تعني عشرين نداءً لقائمةٍ واحدة.
  //
  // ولا يُعرض إلا **من هو على الدوام**: إسنادُ طلبٍ إلى منصرفٍ يُخفيه عن
  // الطابور ولا يوصله أحد.
  const canAssign = o.status === "preparing" || o.status === "dispatching";

  async function openAssign() {
    setAssigning(true);
    try {
      const res = await api<{ drivers: DriverRow[] } | DriverRow[]>("/api/v1/admin/drivers");
      const list = Array.isArray(res) ? res : res.drivers;
      setDrivers(list.filter((x) => x.on_shift && x.status === "active"));
    } catch {
      setDrivers([]);
    }
  }

  // **ما يُرسَل باسم المنصة يُقرأ قبل أن يُرسَل.** والنصُّ والرابطُ من الخادم
  // لا من هنا: لو رُكّبا في الواجهة لأمكن أن يفترقا عمّا يصل المتجر، ولا
  // يكتشفه أحدٌ حتى يشتكي متجر.
  async function openSend() {
    setErr("");
    try {
      setPreview(await api<MerchantMessage>(`/api/v1/admin/orders/${o.id}/message`));
    } catch (e) {
      setErr(e instanceof ApiError ? translateKey(e.body.message_key) : m.errors.internal);
    }
  }

  /**
   * التحويل إلى المتجر.
   *
   * **النافذةُ تُفتح قبل الشبكة لا بعدها.** المتصفّحات تمنع فتحَ نافذةٍ لا
   * ينشأ عن ضغطةٍ مباشرة، و`await` قبلها يقطع هذا النسب — فتُحجب النافذة
   * ويظنّ الموظّفُ أن الزرّ معطوب.
   *
   * **والوسمُ يقول ما نعلمه**: فتحنا المحادثةَ والنصُّ فيها. أمّا أنه ضغط
   * إرسال فلا نعلمه — فاللفظ «حُوِّل» لا «وصل».
   */
  async function send(channel: "whatsapp" | "sms") {
    if (channel === "whatsapp" && preview?.wa_link) {
      window.open(preview.wa_link, "_blank", "noopener");
    }
    setBusy(channel);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${o.id}/whatsapp`, {
        method: "POST",
        body: JSON.stringify({ channel }),
      });
      setPreview(null);
      onChanged();
    } catch (e) {
      setErr(e instanceof ApiError ? translateKey(e.body.message_key) : m.errors.internal);
    } finally {
      setBusy("");
    }
  }

  async function assign(driverID: string) {
    setBusy("assign");
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${o.id}/assign`, {
        method: "POST",
        body: JSON.stringify({ driver_id: driverID, note: "" }),
      });
      setAssigning(false);
      onChanged();
    } catch (e) {
      setErr(e instanceof ApiError ? translateKey(e.body.message_key) : m.errors.internal);
    } finally {
      setBusy("");
    }
  }

  async function go(to: string, note: string) {
    setBusy(to);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${o.id}/transition`, {
        method: "POST",
        body: JSON.stringify({ to, note }),
      });
      setAsking("");
      setReason("");
      onChanged();
    } catch (e) {
      setErr(e instanceof ApiError ? translateKey(e.body.message_key) : m.errors.internal);
    } finally {
      setBusy("");
    }
  }

  // **السببُ بدل الضغطتين العمياوين.**
  //
  // كانت الضغطةُ الثانية تحرس من الإصبع الزالّ وحده. والسببُ يحرس منه **ويُبقي
  // أثراً**: هو ما يُقال للزبون، وما يُقاس به متجرٌ يُكثر الرفض أو موظّفٌ يُكثر
  // الإلغاء. **وطلبٌ يُلغى بلا كلمة يترك الجميع يخمّنون.**
  if (preview !== null) {
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium">
          {m.admin.ordersPage.sendPreview} · {preview.phone}
        </p>
        <pre className="max-h-52 overflow-auto whitespace-pre-wrap rounded-control bg-page p-3 text-xs leading-relaxed">
          {preview.text}
        </pre>
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex flex-wrap gap-2">
          {preview.wa_link && (
            <Button disabled={busy !== ""} onClick={() => void send("whatsapp")}>
              {m.admin.ordersPage.sendViaWhatsApp}
            </Button>
          )}
          {/* **لا يُعرض زرٌّ لا يعمل.** بوّابةٌ غير مضبوطة تردّ خطأً، وزرٌّ
              يُضغط فيعتذر يُعلّم الموظّفَ ألّا يثق بالأزرار. */}
          {preview.sms_ready && (
            <Button
              variant="secondary"
              disabled={busy !== ""}
              onClick={() => void send("sms")}
            >
              {m.admin.ordersPage.sendViaSMS}
            </Button>
          )}
          <Button variant="secondary" onClick={() => setPreview(null)}>
            {m.common.cancel}
          </Button>
        </div>
      </div>
    );
  }

  if (assigning) {
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium">{m.admin.ordersPage.chooseDriver}</p>
        {drivers.length === 0 ? (
          <p className="text-xs text-ink-muted">{m.admin.ordersPage.noDriversOnShift}</p>
        ) : (
          <div className="flex flex-col gap-1.5">
            {drivers.map((dv) => (
              <Button
                key={dv.id}
                variant="secondary"
                disabled={busy !== ""}
                onClick={() => void assign(dv.id)}
              >
                {dv.full_name || dv.phone}
                {dv.open_orders > 0 && ` (${fmtNum(dv.open_orders)})`}
              </Button>
            ))}
          </div>
        )}
        {err && <p className="text-xs text-danger">{err}</p>}
        <Button variant="secondary" onClick={() => setAssigning(false)}>
          {m.common.cancel}
        </Button>
      </div>
    );
  }

  if (asking) {
    return (
      <div className="w-full space-y-2" onClick={(e) => e.stopPropagation()}>
        <p className="text-xs font-medium text-danger">
          {m.admin.ordersPage.reasonTitle.replace("{action}", ACTION_LABELS[asking] ?? asking)}
        </p>
        <Input
          id={`reason-${o.id}`}
          autoFocus
          value={reason}
          onChange={(e) => {
            setReason(e.target.value);
            setErr("");
          }}
          placeholder={m.admin.ordersPage.reasonPlaceholder}
        />
        <p className="text-xs text-ink-muted">{m.admin.ordersPage.reasonHint}</p>
        {err && <p className="text-xs text-danger">{err}</p>}
        <div className="flex gap-2">
          <Button
            variant="danger"
            disabled={!reason.trim() || busy !== ""}
            onClick={() => void go(asking, reason.trim())}
          >
            {m.admin.ordersPage.confirm}
          </Button>
          <Button
            variant="secondary"
            onClick={() => {
              setAsking("");
              setReason("");
              setErr("");
            }}
          >
            {m.common.cancel}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <>
      {next.map((to) => {
        const destructive = DESTRUCTIVE.has(to);
        return (
          <Button
            key={to}
            variant={destructive ? "danger" : "primary"}
            disabled={busy !== ""}
            onClick={() => (destructive ? setAsking(to) : void go(to, ""))}
          >
            {ACTION_LABELS[to] ?? to}
          </Button>
        );
      })}
      {/* **التحويلُ إلى المتجر — في الوضعين لا في وضعٍ واحد.**

          في «المنصة تدير» هو الفعلُ الأساسيّ: تقبل العملياتُ نيابةً عن المتجر
          ثم تُحوّل إليه الطلب، ولا تعلن بدء تحضيرٍ لم تبدأه.

          وفي «المتجر يدير» يبقى بيدها: **المطعمُ لا يردّ على بوّابته أحياناً**
          — والطلبُ الذي في الشاشة لا يُطبخ حتى يراه أحد. فتُحوّله على واتساب
          حيث يعمل صاحبُه أصلاً، وهو ثانويٌّ هنا لا أساسيّ.

          والنافذةُ حتى `preparing`: بعد بدء الطبخ لم يعد المطبخُ يحتاج خبراً. */}
      {(o.status === "accepted" || o.status === "preparing") && (
        <Button
          variant={o.sent_to_merchant_at || selfManage ? "secondary" : "primary"}
          disabled={busy !== ""}
          onClick={() => void openSend()}
        >
          {o.sent_to_merchant_at
            ? m.admin.ordersPage.sentWhatsApp
            : m.admin.ordersPage.sendWhatsApp}
        </Button>
      )}
      {canAssign && (
        <Button variant="secondary" disabled={busy !== ""} onClick={() => void openAssign()}>
          {m.admin.ordersPage.assignHere}
        </Button>
      )}
      {err && <p className="w-full text-xs text-danger">{err}</p>}
    </>
  );
}
