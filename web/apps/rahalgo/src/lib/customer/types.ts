/**
 * ══════════════════════════════════════════════════════════════════════
 * **أشكالُ ردود الزبون — كما يرسلها المحرّكُ اليوم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: زبونُ آيفون يطلب من المتصفّح.)
 *
 * **مرآةٌ لنماذج تطبيق أندرويد** (`mobile/shared/.../model/Shop.kt`
 * و`customer/CustomerApi.kt`) — **وقِيست على التجهيز بنداءٍ حيّ قبل أن
 * تُكتب.** **وما كان اختياريّاً في Go يكون اختياريّاً هنا**: حقلٌ يصل
 * `null` لا يُسقط الشاشة.
 *
 * **ولا منطقَ عملٍ فيها** — السعرُ والتوصيلُ والخصمُ يحسبها المحرّك.
 */

export interface Banner {
  id: string;
  title: string;
  image_url: string | null;
  image_thumb_url?: string | null;
  target?: string;
  sizes?: boolean;
}

export interface Category {
  id: string;
  name: string;
  icon: string;
}

export interface Section {
  id: string;
  name: string;
  icon: string;
  image_url: string | null;
  image_thumb_url: string | null;
  count: number;
  orderable_now: number;
}

/** **مربّعٌ في شريط «متاجر مميّزة»** (قرارُ المالك ٢٠٢٦-١٠-١٨). */
export interface FeaturedStore {
  id: string;
  name: string;
  logo_url: string | null;
  logo_thumb_url: string | null;
  open_now: boolean;
  opens_at: string | null;
}

export interface HomePage {
  banners: Banner[];
  categories: Category[];
  sections: Section[];
  /** **وغيابُه من خادمٍ أقدم = لا شريط.** */
  featured_stores?: FeaturedStore[];
  banner_auto: boolean;
  banner_every_ms: number;
}

/** **رأسُ صفحة المتجر.** */
export interface StoreHeader {
  id: string;
  name: string;
  logo_url: string | null;
  logo_thumb_url: string | null;
  open_now: boolean;
  opens_at: string | null;
  prep_minutes: number;
  /** **إغلاقٌ طارئ** — «مسكّر — إغلاق طارئ» بدل الدوام (٢٠٢٦-١٠-٢٨). */
  emergency_closed?: boolean;
  /** **دوامُ اليوم** — بدل «مفتوح هلّق» (٢٠٢٦-١٠-١٨). */
  today?: { closed: boolean; open_24h: boolean; open: string; close: string };
}

/** **قسمُ منصّةٍ وأصنافُ المتجر فيه** — الصنفُ بشكل القسم حرفاً. */
export interface StoreSection {
  id: string;
  name: string;
  icon: string;
  image_url: string | null;
  image_thumb_url: string | null;
  items: Item[];
}

export interface StorePage {
  store: StoreHeader;
  sections: StoreSection[];
}

export interface Item {
  id: string;
  name: string;
  description: string;
  price: number;
  merchant_id?: string;
  merchant_name?: string | null;
  image_url: string | null;
  image_thumb_url: string | null;
  available: boolean;
  source_closed: boolean;
  source_opens_at: string | null;
  section_id: string;
  section_name: string;
  price_before: number | null;
  discount_percent: number | null;
  price_usd?: number | null;
  price_before_usd?: number | null;
  exchange_rate?: number | null;
  has_options: boolean;
}

export interface ModifierOption {
  id: string;
  name: string;
  price_delta: number;
  price_delta_usd?: number | null;
  available: boolean;
  /** **وزنُ الخيار بالغرام** لمجموعة «حسب الوزن» (٢٠٢٦-١٠-٢٨). */
  grams?: number | null;
}

export interface ModifierGroup {
  id: string;
  name: string;
  min_select: number;
  max_select: number;
  options: ModifierOption[];
  /** **طريقةُ السعر** «priced» «none» «weight» — و«weight» يُعرض سعرُها كاملاً. */
  price_mode?: string;
  base_grams?: number | null;
  /** **مجموعةُ مشروبات المتجر** — إن أرسلها المحرّك؛ وإلّا تُعرف باسمها (`isDrinksGroup`). */
  store_drinks?: boolean;
}

/** **اسمُ مجموعة مشروبات المتجر** — مرآةُ `DrinksGroupName` في المحرّك. */
export const DRINKS_GROUP = "مشروبات";

/**
 * **مشروباتُ المتجر** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — كلُّ مشروبٍ بعدّاده، ولا يُضرب بعدد الصنف.
 * مرآةُ `ModifierGroup.isDrinks` في التطبيق.
 */
export function isDrinksGroup(g: ModifierGroup): boolean {
  return !!g.store_drinks || g.name.trim() === DRINKS_GROUP;
}

export interface ItemDetail {
  item: Item;
  modifiers: ModifierGroup[];
}

export interface Availability {
  available: boolean;
  reason: string;
  message?: string;
  next_available_at?: string;
  place_name?: string;
}

export interface CartChange {
  type: string;
  menu_item_id: string;
  name: string;
  /** **اسمُ الخيار الذي نفد** — مع `product_unavailable` وحدَه، والصنفُ قائم. */
  option?: string;
  qty: number;
  old_value: number;
  new_value: number;
}

export interface Quote {
  changes?: CartChange[];
  blocked?: boolean;
  subtotal: number;
  delivery_fee: number;
  total: number;
  sources: number;
  max_sources: number;
  too_many_sources: boolean;
  base_fee: number;
  sources_fee: number;
  /** **ما أُعفي منه بعرض التوصيل المجانيّ** — و`delivery_fee` بعده (٢٠٢٦-١٠-٢٦). */
  free_waived?: number;
  out_of_zone: boolean;
  zone_closed: boolean;
  next_available_at?: string;
  availability?: Availability | null;
}

export interface PromoPreview {
  valid: boolean;
  discount: number;
  delivery_fee: number;
  reason: string;
  min_order: number;
  kind: string;
  changes?: CartChange[];
}

export interface Address {
  id: string;
  area_building: string;
  street: string;
  floor: string;
  kind: string;
  address_text: string;
  lat: number;
  lng: number;
  is_default: boolean;
}

export interface Ordering {
  ordering_available: boolean;
  reason: string;
  message: string;
  next_available_at?: string;
}

export interface Launch {
  customer_signup: boolean;
  customer_browse: boolean;
  customer_orders: boolean;
  notice: string;
}

/**
 * **سياسةُ أجرة الطلب الخاصّ** — مرآةُ `CustomDelivery` في أندرويد.
 *
 * **`admin_defined` تعني أنّ المنصّةَ حدّدتها** فتُعرض قبل الإرسال، **وغيرُها
 * يحدّدها السائقُ عند الاتّفاق.** **وأجرةُ الأمانة صفرُها «مثل المشتريات».**
 */
export interface CustomDelivery {
  source: string;
  fee: number;
  fee_amanah: number;
}

/** **ما تقوله المنصّةُ عن أبوابها** — من `/public/platform`. */
export interface CustomerPlatform {
  ordering: Ordering;
  launch: Launch & { customer_custom_orders: boolean };
  signup_verify: boolean;
  password_min_length: number;
  social: { whatsapp: string };
  /** **رقمُ الدعم** — وعليه واتساب حين لا يُضبط رقمُ واتسابٍ منفصل (قرارُ المالك: رقمٌ واحد). */
  support_phone: string;
  custom_delivery: CustomDelivery;
  /** **الدخولُ برمزٍ على واتساب** — مفتاحُ `auth.otp_login` في اللوحة. */
  otp_login: boolean;
}

export interface MyOrder {
  /** **إعفاءُ عرض التوصيل المجانيّ** — و`delivery_fee` بعده (٢٠٢٦-١٠-٢٦). */
  free_waived?: number;
  /** **كودُ الاستلام** — لصاحب الطلب من الاستلام حتّى الباب، وإلّا غائب. */
  delivery_code?: string;
  id: string;
  number: number;
  kind: string;
  status: string;
  subtotal: number;
  delivery_fee: number;
  discount: number;
  total: number;
  payment_method: string;
  address_text: string;
  notes: string;
  custom_request: string;
  custom_goods_amount: number | null;
  custom_mode: string;
  custom_goods_pending: boolean;
  custom_fee: number | null;
  quote_version: number;
  quote_confirmed_at: string | null;
  quote_confirmed_version: number | null;
  quote_rejected_at: string | null;
  custom_reserved_amount: number;
  created_at: string;
  delivered_at: string | null;
  cancel_reason: string;
  cancel_seconds_left: number;
  driver_name: string | null;
  /** **أُسند له سائقٌ يوماً** — ومنه يُعرف أيُسأل عن نجوم السائق. */
  driver_assigned?: boolean;
  items_count: number;
  items_preview: string;
  /** **بنودُ الطلب للفاتورة** (٢٠٢٦-١٠-٣٠) — والمشروبُ بندٌ مستقلٌّ بعد صنفه (`drink_for`). */
  items?: MyOrderLine[] | null;
  wallet_paid: number;
  cash_due: number;
  stages: string[];
  stage_at: number;
  /** **زمنُ الطريق من المتجر إلى الباب بالثواني** — يُلتقط عند الاستلام، و`null` لم يُقَس. */
  to_door_eta_sec?: number | null;
  /** **مدّةُ التوصيل المتوقّعة بالدقائق** — من المحرّك، وصفرٌ لا تقدير. */
  delivery_estimate_min?: number;
}

/**
 * **بندٌ في طلب الزبون** — صنفٌ أو مشروبٌ طُلب معه. مرآةُ `MyOrderLine` في التطبيق.
 * **و`drink_for` يعني «مشروبٌ لبند كذا»**؛ وطلبٌ قديمٌ مشروبُه داخل خيارات الصنف.
 */
export interface MyOrderLine {
  id: string;
  menu_item_id: string | null;
  name: string;
  unit_price: number;
  qty: number;
  note?: string;
  options?: { id?: string; group?: string; name: string; price_delta: number }[] | null;
  drink_for?: string | null;
  drink_option_id?: string | null;
}

export interface MyOrdersPage {
  orders: MyOrder[];
  total: number;
  page: number;
}

export interface Wallet {
  balance: number;
  available?: number | null;
  reserved?: number | null;
}

export interface MeSummary {
  full_name: string;
  phone: string;
  balance: number;
  avatar_thumb_url: string | null;
  whatsapp_phone?: string | null;
  whatsapp_verified?: boolean;
}

/** **عرضٌ سارٍ** — من `/public/offers`، **والسريانُ يقوله الخادم** (`live`). */
export interface Offer {
  id: string;
  kind: string;
  title: string;
  body: string;
  image_url: string | null;
  menu_item_id: string | null;
  item_name: string;
  item_image_url: string | null;
  price_before: number;
  price_after: number;
  discount_percent: number | null;
  ends_at: string | null;
  live: boolean;
  has_options: boolean;
  merchant_name?: string | null;
}

/** **صفٌّ من «ما قيّمه»** — `/my/ratings`. */
export interface RatingRow {
  order_id: string;
  has_driver: boolean;
  rated: boolean;
}

/** **سببُ شكوى — رمزٌ لا نصّ**، والاسمُ العربيُّ في المعجم. */
export interface ComplaintReason {
  code: string;
  against: string;
}

/** **حديثٌ في «محادثاتي»** — ومنه عدّادُ ما لم يُقرأ. */
export interface ChatRow {
  order_id: string;
  number: number;
  peer: string;
  open: boolean;
  unread: number;
}

/** **حالُ دعواته** — `/auth/referral`، **والرابطُ يبنيه الخادم.** */
export interface Referral {
  code: string;
  link: string;
  invited: number;
  rewarded: number;
  earned: number;
  next_reward: number;
  tiers: number[];
  rest: number;
  reward_on: string;
}

/** **حالاتُ المشاركة بالموقع** — كما يقبلها المحرّك (`orders.LiveStatuses`). */
export const LIVE_STATUSES = new Set(["assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"]);

export interface Notice {
  id: string;
  kind: string;
  title: string;
  body: string;
  entity: string;
  entity_id: string;
  created_at: string;
}

/** **حالاتُ الطلب المنتهية** — مرآةُ `ENDED` في تطبيق أندرويد. */
export const ENDED = new Set(["delivered", "cancelled", "failed", "rejected", "refunded"]);

/** **أللزبون توصيلةٌ مجانيّةٌ الآن — ولأيّ نوع** (`/my/free-delivery`، ٢٠٢٦-١٠-٢٦). */
export interface FreeDelivery {
  standard: boolean;
  purchase: boolean;
  amanah: boolean;
  max_fee: number | null;
  ends_at: string | null;
}
