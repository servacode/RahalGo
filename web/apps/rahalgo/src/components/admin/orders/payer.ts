/**
 * **مين دفع وكيف** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — كانت كروتُ السجلّ تقول «الدفع عند الاستلام» على كلّ
 * طلب، فطلبٌ دفعه المتجرُ من محفظته ظهر نقداً وظنّه المالكُ ضائعاً من كشف النقد.
 *
 * مصدرٌ واحدٌ لكرت السجلّ ولكشف نقد السائق — **عرضٌ فقط، لا يحسب مالاً.** القواعدُ من المحرّك:
 * «لدي توصيلة» يقرّرها `fee_payer` (`merchant` من محفظته · `merchant_cash` نقداً عند الاستلام ·
 * `recipient` نقداً عند التسليم — `orders/merchant_delivery.go`)، وغيرُها `wallet_paid` و`cash_due`.
 */
import { getMessages, defaultLocale, fmtMoney } from "@rahalgo/i18n";

const P = getMessages(defaultLocale).admin.ordersPage.payer;

export type PayerKey =
  | "cashCustomer"
  | "walletCustomer"
  | "mixedCustomer"
  | "merchantWallet"
  | "merchantCash"
  | "recipientCash"
  | "free";

export interface PayerInput {
  kind?: string;
  fee_payer?: string;
  payment_method?: string;
  wallet_paid?: number;
  cash_due?: number;
  free_waived?: number;
}

export function payerOf(o: PayerInput): PayerKey {
  const w = o.wallet_paid ?? 0;
  const c = o.cash_due ?? 0;
  const free = (o.free_waived ?? 0) > 0;
  if (o.kind === "merchant_delivery") {
    if (free && w === 0 && c === 0) return "free";
    if (o.fee_payer === "recipient") return "recipientCash";
    if (o.fee_payer === "merchant_cash") return "merchantCash";
    return "merchantWallet";
  }
  if (w > 0 && c > 0) return "mixedCustomer";
  if (c > 0) return "cashCustomer";
  if (w > 0) return "walletCustomer";
  if (free) return "free";
  return o.payment_method === "wallet" ? "walletCustomer" : "cashCustomer";
}

/** **أيُّ الأربعة في ملخّص اليوم**: نقدٌ (أيًّا كان دافعُه) · محفظةُ الزبون · محفظةُ المتجر · مجاني. */
export type PayerGroup = "cash" | "customerWallet" | "merchantWallet" | "free";

export function payerGroup(k: PayerKey): PayerGroup {
  if (k === "walletCustomer") return "customerWallet";
  if (k === "merchantWallet") return "merchantWallet";
  if (k === "free") return "free";
  return "cash";
}

export function payerLabel(o: PayerInput): string {
  const k = payerOf(o);
  if (k === "mixedCustomer") {
    return P.mixedCustomer.replace("{w}", fmtMoney(o.wallet_paid ?? 0)).replace("{c}", fmtMoney(o.cash_due ?? 0));
  }
  return P[k];
}

/** نقدٌ أم محفظة — للّون: النقدُ تحذيريّ، والمحفظةُ والمجانيّ هادئان. */
export function payerIsCash(k: PayerKey): boolean {
  return payerGroup(k) === "cash";
}

/** **نوعُ الطلب بلفظ اللوحة** — عاديّ · توصيل من المتجر · إحضار للمتجر · أمانة · مشتريات. */
export function orderKindLabel(o: { kind?: string; delivery_mode?: string; custom_mode?: string }): string {
  const OP = getMessages(defaultLocale).admin.ordersPage;
  if (o.kind === "merchant_delivery") return o.delivery_mode === "fetch" ? OP.kindMdFetch : OP.kindMdSend;
  if (o.kind === "custom") return o.custom_mode === "amanah" ? OP.kindAmanah : OP.kindPurchase;
  return OP.kindStd;
}
