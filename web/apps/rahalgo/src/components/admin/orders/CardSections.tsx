"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أقسامُ بطاقة الطلب في لوحة العمل** (قرارُ المالك ٢٠٢٦-١١-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 *	١ · الطريق     سطرٌ واحدٌ بالأسماء — من أين ولمن، حسب نوع الطلب
 *	٢ · المتجر     اسمٌ واتّصال · العنوانُ المفصّلُ وزرُّ خريطة · بُعدُ السائق · بُعدُ الزبون
 *	٣ · الوقت      المقدّرُ للوصول للمتجر وللزبون
 *	٤ · الزبون     اسمٌ واتّصالٌ وخريطة · العنوانُ المفصّل · الملاحظة أو «لا يوجد ملاحظات»
 *	٦ · العمولة    من التوصيل ومن المتجر، ثمّ حسابُ السائق كما كان
 *
 * (والفاتورةُ ٥ في `OrdersScreen.tsx` — تُعدَّل من البطاقة.) **والأرقامُ كلُّها من الخادم**،
 * والشاشةُ تعرضها: المسافاتُ وزمنا الطريق والعمولة.
 */

import { getMessages, defaultLocale, fmtNum, fmtSpan } from "@rahalgo/i18n";
import { IconHourglass, IconLocation, IconRoute, IconStore, IconUser, Money } from "@rahalgo/ui";
import { ContactLinks } from "./BoardWidgets";
import { DriverMoneyBox } from "./DriverMoneyBox";

const m = getMessages(defaultLocale);
const OP = m.admin.ordersPage;
const C = OP.card;
const SEC = OP.sections;

/** **سطرُ صنفٍ في قسم متجر** (طلبٌ من أكثر من متجر). */
export interface PartItem {
  name: string;
  qty: number;
  unit_price: number;
  merchant_price: number;
  options?: string;
  note?: string;
}

/**
 * **قسمُ متجرٍ في طلبٍ من أكثر من متجر** (٢٠٢٦-١١-٠٦) — مرآةُ `parts[]` في ردّ المكتب.
 * غائبٌ في طلب المتجر الواحد.
 */
export interface OrderPart {
  merchant_id: string;
  seq: number;
  name: string;
  phone?: string;
  address?: string;
  lat?: number | null;
  lng?: number | null;
  status: "pending" | "accepted" | "rejected" | "removed" | "picked_up";
  answered_role?: "" | "merchant" | "ops" | "admin" | "system";
  answered_at?: string | null;
  reason?: string;
  removed_fault?: "" | "merchant" | "platform";
  prep_minutes?: number | null;
  ready_at?: string | null;
  sent_at?: string | null;
  sent_channel?: "" | "whatsapp" | "manual";
  arrived_at?: string | null;
  picked_up_at?: string | null;
  picked_up_by_name?: string;
  collected?: boolean;
  items?: PartItem[];
  subtotal?: number;
  cost?: number;
  settlement_method?: "cash" | "wallet";
  pay_amount?: number;
  rep_name?: string;
  rep_commission?: number;
}

/** **ما تقرؤه الأقسامُ من الطلب** — حقولُ ردّ المكتب كما هي. */
export interface CardOrder {
  /** أقسامُ المتاجر — غائبٌ في طلب المتجر الواحد. */
  parts?: OrderPart[];
  kind?: string;
  delivery_mode?: string;
  custom_mode?: string;
  status: string;
  closed_at?: string | null;
  picked_up_at?: string | null;
  customer_name: string;
  customer_phone: string;
  customer_phone_masked?: string;
  merchant_name: string;
  merchant_phone?: string;
  partner_merchant_name?: string;
  holder_name?: string;
  holder_phone?: string;
  driver_name: string | null;
  address_text: string;
  zone_name?: string | null;
  notes: string;
  lat?: number | null;
  lng?: number | null;
  pickup_lat?: number | null;
  pickup_lng?: number | null;
  pickup_address?: string;
  leg_m?: number;
  driver_to_pickup_m?: number;
  driver_to_drop_m?: number;
  to_store_eta_sec?: number | null;
  to_door_eta_sec?: number | null;
  payment_method: string;
  delivery_fee: number;
  free_waived?: number;
  driver_fee?: number;
  custom_fee?: number | null;
  custom_fee_snapshot?: number | null;
  cash_due?: number;
  platform_commission?: number;
  commission_taken?: number;
  commission_percent_exact?: number;
  pay_merchant?: number;
  /** **ما يدفعه السائقُ لكلّ متجر** — والدولارُ (`currency: "USD"`) بالسنتات ولا يدخل `pay_merchant` بالليرة. */
  pay_merchants?: { merchant_name: string; amount: number; currency?: string }[];
  /** **المسافةُ كما قادها السائق** (قرارُ المالك ٢٠٢٦-١١-٠٥) — صفٌّ لكلّ سائقٍ حمله. */
  drive?: DriveRow[];
}

/**
 * **مسافةُ الطلب لسائقٍ حمله** — «من بداية استلام الطلب الى انتهاء الطلب»: للمتجر (من القبول
 * حتى الاستلام) وللزبون (حتى خرج من يده، ومعه مشوارُ إرجاع البضاعة). **من القيادة الفعليّة**
 * لا من رجفة الموقع، ويحسبها الخادم بعد كلّ دقيقة.
 */
export interface DriveRow {
  driver_id: string;
  driver_name: string;
  started_at: string;
  picked_up_at?: string | null;
  /** فارغٌ = الطلبُ لسّا معه، والرقمُ «لحد هلق». */
  ended_at?: string | null;
  end_kind: string;
  to_pickup_m: number;
  with_order_m: number;
  /** تقاسم مقطعاً مع طلبٍ آخر بيده. */
  shared: boolean;
  /** دقائقُ بلا إشارةٍ أثناء الطلب — رقمٌ قليلٌ سببُه جوّالٌ صامت. */
  gap_min: number;
  /** ثبت الرقم (بعد ساعتين وعشر دقائق من سكّر الدوام). */
  final: boolean;
}

/** **مسافةٌ مقروءة** — «٤٥٠ م» أو «٢٫٣ كم»، **وسالبٌ لا يُعرف فلا يُكتب.** */
export function distText(v?: number | null): string {
  if (v == null || v < 0) return "";
  return v < 1000
    ? OP.meters.replace("{n}", fmtNum(Math.round(v / 10) * 10))
    : OP.km.replace("{n}", fmtNum(Math.round(v / 100) / 10));
}

const linkCls =
  "inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-control border border-line-soft bg-surface text-accent-text hover:bg-row-hover";

/** **زرُّ الخريطة** — غوغل ماب على النقطة نفسِها، في تبويبٍ جديد. */
export function MapLink({ lat, lng }: { lat?: number | null; lng?: number | null }) {
  if (lat == null || lng == null) return null;
  return (
    <a
      // **خريطتُنا لا غوغل** (قرارُ المالك ٢٠٢٦-١١-٠٣) — خريطةُ العمليّات على النقطة، والمتجرُ والزبونُ والسائقُ عليها.
      href={`/dashboard/opsmap?focus=${lat},${lng}`}
      target="_blank"
      rel="noreferrer"
      aria-label={C.map}
      title={C.map}
      className={linkCls}
      onClick={(e) => e.stopPropagation()}
    >
      <IconLocation size={15} />
    </a>
  );
}

/** **صندوقٌ واحدُ الشكل لكلّ الأقسام** (قرارُ المالك ٢٠٢٦-١١-٠٣: «وحّد شكل الكرت») — نفسُ اللون والإطار والحشوة والعنوان. */
const SAME = { box: "bg-ink-faint border-line-soft", tone: "text-accent-text" };
const BOX = { store: SAME, customer: SAME, plain: SAME } as const;

export function Box({
  look,
  icon,
  title,
  children,
}: {
  look: keyof typeof BOX;
  icon: React.ReactNode;
  title: string;
  children: React.ReactNode;
}) {
  return (
    <div className={`flex h-full flex-col gap-2 rounded-control border p-3 ${BOX[look].box}`}>
      <span className={`flex items-center gap-1.5 text-xs font-bold ${BOX[look].tone}`}>
        {icon}
        {title}
      </span>
      {children}
    </div>
  );
}

/** **سطرُ الطرف** — الاسمُ يميناً، والرقمُ وأزرارُه يساراً، **ولا يفيض على شاشة جوّال.** */
export function PartyLine({
  name,
  phone,
  masked,
  callLabel,
  live,
  extra,
}: {
  name: string;
  phone?: string | null;
  masked?: string;
  callLabel: string;
  live: boolean;
  extra?: React.ReactNode;
}) {
  const shown = phone || masked;
  // **والرقمُ وأزرارُه ينزلان سطراً إن ضاق العرض** — لا يفيضان عن بطاقة جوّالٍ ٣٦٠ بكسلاً.
  return (
    <span className="flex flex-wrap items-center justify-between gap-x-2 gap-y-1 font-medium">
      <span className="min-w-0 flex-1 basis-28 truncate">{name || "—"}</span>
      <span className="ms-auto flex shrink-0 items-center gap-1.5">
        {shown && (
          <span dir="ltr" className="text-xs text-ink-muted">
            {shown}
          </span>
        )}
        {live && <ContactLinks phone={phone} callLabel={callLabel} />}
        {extra}
      </span>
    </span>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ١ · الطريق
// ══════════════════════════════════════════════════════════════════════

/** **من أين ولمن** — سطرٌ واحدٌ بالأسماء حسب نوع الطلب. */
export function routeText(o: CardOrder): string {
  const customer = o.customer_name || "—";
  const store = o.merchant_name || "—";
  if (o.kind === "merchant_delivery" && o.delivery_mode === "fetch") {
    return C.routeFromTo.replace("{from}", customer).replace("{to}", store);
  }
  if (o.kind === "custom") {
    return o.custom_mode === "amanah"
      ? C.routeAmanah.replace("{from}", o.holder_name || "—").replace("{to}", customer)
      : C.routePurchase.replace("{to}", customer);
  }
  return C.routeFromTo.replace("{from}", store).replace("{to}", customer);
}

export function RouteLine({ o }: { o: CardOrder }) {
  return (
    <span className="flex items-start gap-1.5 font-bold">
      <IconRoute size={16} className="mt-0.5 shrink-0 text-accent-text" aria-hidden />
      <span className="min-w-0 break-words">{routeText(o)}</span>
    </span>
  );
}

/** **المسافةُ المقطوعة على البطاقة** (قرارُ المالك ٢٠٢٦-١١-٠٥) — مثلاً «المسافة المقطوعة: ٣ كم». */
export function DriveLine({ o }: { o: CardOrder }) {
  const rows = o.drive ?? [];
  if (rows.length === 0) return null;
  const D = C.drive;
  const total = rows.reduce((s, r) => s + r.to_pickup_m + r.with_order_m, 0);
  const open = rows.some((r) => !r.ended_at);
  const one = rows.length === 1 ? rows[0] : null;
  return (
    <span className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-ink-muted">
      <span className="font-bold text-ink">{(open ? D.soFar : D.total).replace("{d}", distText(total))}</span>
      {one ? (
        <span>{D.parts.replace("{a}", distText(one.to_pickup_m)).replace("{b}", distText(one.with_order_m))}</span>
      ) : (
        <span>{D.drivers.replace("{n}", fmtNum(rows.length))}</span>
      )}
    </span>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ٢ · المتجر — أو مكانُ الاستلام في الخاصّ
// ══════════════════════════════════════════════════════════════════════

/** **أيُّ طرفٍ يُعرض في قسم المتجر** — وفارغٌ: لا قسم. */
function pickupParty(o: CardOrder): { title: string; name: string; phone?: string; store: boolean } | null {
  if (o.kind === "custom") {
    if (o.custom_mode === "amanah") {
      if (!o.holder_name && o.pickup_lat == null) return null;
      return { title: C.holder, name: o.holder_name || "—", phone: o.holder_phone, store: false };
    }
    if (o.partner_merchant_name) return { title: C.pickupParty, name: o.partner_merchant_name, store: false };
    // **متجرٌ غيرُ تابعٍ سمّاه المكتبُ عند القلب** (٢٠٢٦-١١-١٢) — اسمُه وهاتفُه وعنوانُه ونقطتُه.
    if (o.merchant_name) return { title: C.buyFrom, name: o.merchant_name, phone: o.merchant_phone, store: false };
    return null;
  }
  const md = o.kind === "merchant_delivery";
  const title = md ? (o.delivery_mode === "fetch" ? SEC.mdStoreTo : SEC.mdStoreFrom) : C.store;
  return { title, name: o.merchant_name, phone: o.merchant_phone, store: true };
}

export function hasPickupSection(o: CardOrder): boolean {
  return (o.parts?.length ?? 0) > 0 || pickupParty(o) != null;
}

export function StoreSection({ o, live }: { o: CardOrder; live: boolean }) {
  const p = pickupParty(o);
  if (!p) return null;
  const open = !o.closed_at;
  const fromDriver = open && o.driver_name && !o.picked_up_at ? distText(o.driver_to_pickup_m) : "";
  const fromCustomer = open && p.store ? distText(o.leg_m) : "";
  return (
    <Box look="store" icon={<IconStore size={13} />} title={p.title}>
      <PartyLine name={p.name} phone={p.phone} callLabel={OP.callStoreShort} live={live} />
      {(o.pickup_address || o.pickup_lat != null) && (
        <span className="flex items-start justify-between gap-2">
          <span className="min-w-0 flex-1 whitespace-pre-line break-words text-ink-muted">
            {o.pickup_address || "—"}
          </span>
          <MapLink lat={o.pickup_lat} lng={o.pickup_lng} />
        </span>
      )}
      {fromDriver && <span className="text-xs text-ink-muted">{C.storeFromDriver.replace("{d}", fromDriver)}</span>}
      {fromCustomer && <span className="text-xs text-ink-muted">{C.storeFromCustomer.replace("{d}", fromCustomer)}</span>}
    </Box>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ٣ · الوقتُ المقدّر — زمنا الطريق كما قالتهما الخريطةُ لحظةَ الإسناد والاستلام
// ══════════════════════════════════════════════════════════════════════

export function EtaSection({ o }: { o: CardOrder }) {
  const eta = (s?: number | null) => (s != null && s > 0 ? fmtSpan(s) : "—");
  const row = "flex items-center justify-between gap-2";
  return (
    <Box look="plain" icon={<IconHourglass size={13} />} title={C.eta}>
      <span className={row}>
        <span className="text-ink-muted">{C.etaStore}</span>
        <span className="shrink-0 font-medium">{eta(o.to_store_eta_sec)}</span>
      </span>
      <span className={row}>
        <span className="text-ink-muted">{C.etaDoor}</span>
        <span className="shrink-0 font-medium">{eta(o.to_door_eta_sec)}</span>
      </span>
    </Box>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ٤ · الزبون
// ══════════════════════════════════════════════════════════════════════

export function CustomerSection({ o, live }: { o: CardOrder; live: boolean }) {
  const md = o.kind === "merchant_delivery";
  const title = md ? (o.delivery_mode === "fetch" ? SEC.mdSender : C.recipient) : C.customer;
  // **وبُعدُ السائق عن الزبون** — بعد الاستلام، أو حين لا نقطةَ استلامٍ معروفة.
  const fromDriver =
    !o.closed_at && o.driver_name && (o.picked_up_at || !distText(o.driver_to_pickup_m))
      ? distText(o.driver_to_drop_m)
      : "";
  return (
    <Box look="customer" icon={<IconUser size={13} />} title={title}>
      <PartyLine
        name={o.customer_name}
        phone={o.customer_phone}
        masked={o.customer_phone_masked}
        callLabel={OP.callCustomerShort}
        live={live}
        extra={<MapLink lat={o.lat} lng={o.lng} />}
      />
      <span className="whitespace-pre-line break-words text-ink-muted">{o.address_text || "—"}</span>
      {o.zone_name && <span className="text-xs font-medium text-accent-text">{o.zone_name}</span>}
      {fromDriver && <span className="text-xs text-ink-muted">{C.driverFromCustomer.replace("{d}", fromDriver)}</span>}
      <span className={`text-xs ${o.notes ? "text-accent-dark" : "text-ink-muted"}`}>
        {o.notes || C.noNote}
      </span>
    </Box>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ٦ · عمولةُ المنصّة — من التوصيل ومن المتجر، ثمّ حسابُ السائق كما كان
// ══════════════════════════════════════════════════════════════════════

/** **أجرةُ التوصيل كاملةً** — قبل إعفاء العرض، ومن موضعها في كلّ نوع. */
function feeOf(o: CardOrder): number | null {
  if (o.kind === "custom") return o.custom_fee ?? o.custom_fee_snapshot ?? null;
  if (o.kind === "merchant_delivery") return o.delivery_fee;
  return o.delivery_fee + (o.free_waived ?? 0);
}

export function CommissionSection({ o }: { o: CardOrder }) {
  const fee = feeOf(o);
  // **وقبل الإسناد لا عمولةَ توصيل** — أجرُ السائق لم يُعرف بعد (المالك ٢٠٢٦-١٠-٢٢).
  const delivery =
    o.driver_name && o.driver_fee != null && fee != null && fee > 0 ? Math.max(0, fee - o.driver_fee) : null;
  const standard = (o.kind ?? "standard") === "standard";
  const store = standard ? (o.commission_taken ?? o.platform_commission ?? null) : null;
  const row = "flex items-center justify-between gap-2";
  return (
    <ul className="space-y-1.5 text-sm">
      <li className={row}>
        <span className="text-ink-muted">{C.deliveryCommission}</span>
        {delivery != null ? <Money value={delivery} small /> : <span className="text-ink-muted">—</span>}
      </li>
      {standard && (
        <li className={row}>
          <span className="text-ink-muted">{C.storeCommission}</span>
          {store != null ? <Money value={store} small /> : <span className="text-ink-muted">—</span>}
        </li>
      )}
      {standard && <DriverMoneyBox o={o} hideCommission />}
    </ul>
  );
}
