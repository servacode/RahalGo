/**
 * **أشكالُ ردود أبواب المتجر** — كما يردّها المحرّك اليوم (`/api/v1/merchant/…`)،
 * **وكما يقرؤها تطبيقُ المتجر لأندرويد** (`shared/merchant/MerchantApi.kt` و
 * `DeliveriesApi.kt`) حقلاً بحقل.
 *
 * **ولا منطقَ هنا** — أنواعٌ لا غير. القرارُ في المحرّك والعرضُ في الشاشة.
 */

export interface MerchantStore {
  id: string;
  name: string;
  status: string;
  category_icon: string;
  logo_thumb_url: string | null;
  emergency_closed: boolean;
  default_prep_minutes: number;
  min_order: number;
  address_text: string;
  lat: number | null;
  lng: number | null;
  open_now: boolean;
  next_open: string | null;
  settlement_method: string;
  unpaid_cash_due: number;
  settlement_confirmed: boolean;
  accepts_returns: boolean;
  returns_confirmed: boolean;
  has_hours: boolean;
  exchange_rate: number | null;
  display_currency: string;
}

export interface StoresPage {
  stores: MerchantStore[];
  self_manage_orders: boolean;
  item_name_max: number;
}

export interface OrderOption {
  group: string;
  name: string;
  price_delta: number;
}

export interface OrderLine {
  id: string;
  name: string;
  qty: number;
  unit_price: number;
  note: string;
  options: OrderOption[];
}

export interface MerchantOrder {
  id: string;
  number: number;
  status: string;
  subtotal: number;
  platform_commission: number;
  merchant_net: number;
  commission_percent: number;
  /** **النسبةُ بمنزلتين** (١٫٥) — و`commission_percent` مقرَّبةٌ للتطبيق القديم (٢٠٢٦-١٠-١٨). */
  commission_percent_exact?: number;
  items: OrderLine[];
  notes: string;
  created_at: string;
  sent_to_merchant_at: string | null;
  closed_at: string | null;
  cancel_reason: string;
  /** **لحظةُ «جاهز للاستلام»** — وفارغةٌ ما لم يُعلنها المتجر. */
  ready_at?: string | null;
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أيُعرض «جاهز للاستلام»؟** (٢٠٢٦-١٠-١٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **في وضع المتاجر القبولُ يدخل التحضيرَ ويُرسل للسائقين بنفسه** (قرارُ
 * المالك ٢٠٢٦-٠٨-٢٩) — **فلا تُرى «يُحضَّر» لحظةً** وكان الزرُّ لا يظهر
 * أبداً. **والجاهزيّةُ وسمٌ لا حال**: المحرّكُ يقبلها من القبول حتّى وقوف
 * السائق عند الباب (`readyableStatuses`)، **وتُعلَن مرّةً** (`ready_at`).
 */
const READYABLE = new Set(["accepted", "preparing", "dispatching", "assigned", "at_pickup"]);
export function canMarkReady(o: Pick<MerchantOrder, "status" | "ready_at">): boolean {
  return READYABLE.has(o.status) && !o.ready_at;
}

/**
 * **طلبٌ انتهى بلا بيع — فلا مجموعَ ولا عمولةَ ولا مستحقَّ يُعرض** (مرآةُ `noPayout`
 * في `Readiness.kt`، تدقيقُ ٢٠٢٦-١٠-٢٧). **المحرّكُ يحسبها لكلّ طلبٍ مهما كان حالُه**،
 * فطلبٌ اعتُذر عنه كان يُري «المستحق لك» **ولن يقبض منه شيئاً.**
 */
const NO_PAYOUT = new Set(["rejected", "cancelled", "failed", "refunded"]);
export function noPayout(status: string): boolean {
  return NO_PAYOUT.has(status);
}

/**
 * **مجموعةُ الطلب في صفحة الطلبات** — مرآةُ `groupOf`: جديدٌ ينتظر قراراً، ثمّ ما
 * يُحضَّر، ثمّ ما ينتظر السائق، **ثمّ ما انتهى** (لا يصل مع `open_only` إلّا من خادمٍ أقدم).
 */
export type OrderGroup = "new" | "preparing" | "waiting" | "closed";
export const ORDER_GROUPS: readonly OrderGroup[] = ["new", "preparing", "waiting", "closed"];
export function groupOf(o: Pick<MerchantOrder, "status" | "ready_at">): OrderGroup {
  if (o.status === "pending") return "new";
  if (o.status === "delivered" || noPayout(o.status)) return "closed";
  if (o.status === "accepted" || o.status === "preparing" || canMarkReady(o)) return "preparing";
  return "waiting";
}

export interface OrdersPage {
  orders: MerchantOrder[];
  total: number;
  page?: number;
  per_page?: number;
  status_counts?: Record<string, number>;
  /** **ما يُبلَّغ عن سائقه** — معرّفاتُ الطلبات. */
  reportable?: string[];
}

export interface MenuItem {
  id: string;
  name: string;
  description: string;
  price: number;
  available: boolean;
  approved?: boolean;
  review_note?: string;
  image_url: string | null;
  image_thumb_url: string | null;
  platform_section_id: string | null;
  platform_section_name: string;
  price_currency: string;
  merchant_price_usd: number | null;
  modifiers?: ModifierGroup[];
}

/** **مجموعةُ إضافات** — `catalog.ModifierGroup`، **وتُستبدل كلُّها عند الحفظ.** */
export interface ModifierOption {
  id?: string;
  name: string;
  price_delta: number;
  price_delta_usd?: number | null;
  /** **خيارٌ خلصان** (مقاس ٤٢ مثلاً) — والصنفُ يبقى متوفّراً. غيابُه يعني متوفّر. */
  available?: boolean;
}

export interface ModifierGroup {
  id?: string;
  name: string;
  min_select: number;
  max_select: number;
  options: ModifierOption[];
}

/**
 * **مجموعةُ خياراتٍ جاهزة** (قرارُ المالك ٢٠٢٦-١٠-٢٧) — من
 * `GET /merchant/stores/{id}/option-presets`. و`chosen` «تظهر لهذا المتجر».
 */
export interface OptionPreset {
  id: string;
  name: string;
  min_select: number;
  max_select: number;
  options: { name: string; price_delta: number }[];
  sort_order: number;
  active: boolean;
  category_ids?: string[];
  chosen?: boolean;
}

/** **ردُّ خيارات المتجر** — و`custom=false` يعني «ماشي على تصنيفه». */
export interface StorePresets {
  presets: OptionPreset[];
  custom: boolean;
}

/** **قسمُ سوقٍ** — من `/merchant/platform-sections` و`/stores/{id}/sections`. */
export interface SectionRef {
  id: string;
  name: string;
  image_url?: string | null;
  image_thumb_url?: string | null;
}

/** **يومٌ من ساعات العمل** — `0` الأحد … `6` السبت، والوقتُ `HH:MM`. */
export interface DayHours {
  day_of_week: number;
  closed: boolean;
  open_time: string;
  close_time: string;
  /** **«مفتوح ٢٤ ساعة»** — ٠٠:٠٠ ← ٠٠:٠٠ (٢٠٢٦-١٠-١٨). */
  open_24h?: boolean;
}

/** **عرضٌ على صنف** — مرآةُ `StoreOffer` في التطبيق. */
export interface StoreOffer {
  id: string;
  title: string;
  menu_item_id: string | null;
  item_name: string;
  discount_percent: number | null;
  discount_amount: number | null;
  price_before: number;
  price_after: number;
  starts_at: string | null;
  ends_at: string | null;
  status: string;
}

export interface Warning {
  id: string;
  reason: string;
  note: string;
  order_number?: number | null;
  created_at: string;
}

export interface MerchantReport {
  id: string;
  number: number;
  order_number: number | null;
  reason: string;
  status: string;
  resolution: string;
  created_at: string;
}

export interface MenuSection {
  id: string;
  name: string;
  image_thumb_url?: string | null;
  items: MenuItem[];
}

export interface DeliveryQuote {
  fee: number;
  zone_name: string;
  wallet_balance: number;
  available_balance: number;
  merchant_can_pay: boolean;
  fee_basis: string;
  distance_m: number;
  mode: string;
}

export interface Delivery {
  id: string;
  number: number;
  status: string;
  customer_name: string;
  customer_phone: string;
  address_text: string;
  parcel_note: string;
  fee_payer: string;
  delivery_fee: number;
  delivery_mode: string;
  collect_amount: number | null;
  fetch_amount?: number | null;
  fetch_amount_version?: number;
  fetch_amount_status?: string;
  /** **عملةُ المبلغين** — «SYP» أو «USD»، والدولارُ بالسنتات (٢٠٢٦-١٠-٢٠). */
  amount_currency?: string;
  delivery_code?: string | null;
  dropoff_known: boolean;
  picked_up_at: string | null;
  delivered_at: string | null;
  cancel_reason: string | null;
  created_at: string;
}

export interface StoreReport {
  summary: {
    orders: number;
    delivered: number;
    cancelled: number;
    sales: number;
    platform_commission: number;
  };
}
