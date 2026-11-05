/**
 * **سطرُ الدين تحت رصيد المحفظة** — «عليك X، بسبب كذا، ويُخصم من أول أرباح قادمة».
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «الدين اللي ما بينفهم بيصير شكوى»، ومدّه ٢٠٢٦-١٠-١٥ إلى
 *  المتجر والمندوب والسائق.) دالّتان صافيتان بلا واجهة، **وتوأمُهما في أندرويد**
 *  `DebtLineText` و`debtCause` في `mobile/ui/.../WalletScreen.kt` — **بالنصّ نفسِه والأسباب
 *  نفسِها**، ومجهولُ السبب «دين سابق» كالتطبيق. يحرسهما `scripts/check-store-parity.mjs`:
 *  سببٌ يُضاف في التطبيق ويُنسى هنا يُسقط الفحص.
 *
 * **وفي `.ts` لا `.tsx` بقصد**: لتُفحص بالنداء بلا واجهة.
 */

import { getMessages, defaultLocale, fmtMoney } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);
const D = m.shared.walletDebt;

/** **دينٌ مفتوحٌ بسببه** — كما يرسله `GET /api/v1/my/wallet` (`debts`). */
export interface DebtLine {
  amount: number;
  cause: string;
}

/** **سببُ الدين بكلمات المكتب** — والسببُ رمزٌ من المحرّك (`obligations.Cause*`). */
export function debtCauseText(cause: string): string {
  switch (cause) {
    case "refund_merchant_earning":
    case "refund_rep_commission":
      return D.refund;
    case "returned_goods":
      return D.returned;
    case "merchant_delivery_fee":
      return D.delivery;
    case "delivery_failed_recipient":
      return D.deliveryFailed;
    case "penalty_remainder":
      return D.penalty;
    case "dispute_remainder":
      return D.dispute;
    case "waived_rep_commission":
      return D.waivedRep;
    case "undelivered_order":
      return D.undelivered;
    // **نقصُ حقّ المتجر بتعديل الطلب بعد الاستلام** (قرارُ المالك ٢٠٢٦-١١-٠٣، هجرة 0497).
    case "order_edit":
      return D.orderEdit;
    default:
      return D.old;
  }
}

/**
 * **السطرُ كاملاً** — وفارغٌ لمن لا دينَ عليه (المحرّكُ لا يرسل `debt_total` إلّا لصاحب دين).
 * والأسبابُ المتكرّرةُ تُكتب مرّةً وتُفصل بالفاصلة العربية.
 */
export function debtLineText(total: number, debts: DebtLine[]): string {
  if (!(total > 0)) return "";
  const causes = Array.from(new Set(debts.map((d) => debtCauseText(d.cause)))).join(m.common.listSep);
  return D.line.replace("{amount}", fmtMoney(total)).replace("{causes}", causes);
}
