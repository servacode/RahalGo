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
  items: OrderLine[];
  notes: string;
  created_at: string;
  sent_to_merchant_at: string | null;
  closed_at: string | null;
  cancel_reason: string;
}

export interface OrdersPage {
  orders: MerchantOrder[];
  total: number;
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
