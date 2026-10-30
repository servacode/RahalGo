/**
 * **نصُّ سطر الفاتورة وترتيبُ بنودها** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — دوالُّ صافيةٌ بلا واجهة،
 * **وتوأمُها في أندرويد** `mobile/ui/.../Invoice.kt` (`invoiceLineText`). تفصيلُ القاعدة في `InvoiceLines.tsx`.
 *
 * **وفي `.ts` لا `.tsx` بقصد**: كلماتُ المطابقة («كيلو»، «غ»، «وزن»، «نمرة») قاعدةٌ لا نصُّ
 * شاشة، والفاصلُ من المعجم.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

export interface InvoiceLineItem {
  id?: string;
  name: string;
  qty: number;
  unit_price: number;
  note?: string;
  options?: { group?: string; name: string }[] | null;
  /** **مشروبُ بندٍ آخر** — معرّفُ الصنف الذي طُلب معه. */
  drink_for?: string | null;
}

const WEIGHT = /كيلو|كغ|غرام|(^|\s|[0-9٠-٩])غ$/;
const NUMBER_ONLY = /^[0-9٠-٩]+([.,٫][0-9٠-٩]+)?$/;

/**
 * **نصُّ سطر الفاتورة — القاعدةُ في مكانٍ واحد.**
 *
 * - **الوزنُ** (مجموعةٌ فيها «وزن» أو قيمةٌ بالكيلو/الغرام) بعد العدد كما هو: «× 1 كيلو».
 * - **الرقمُ المجرّد** (نمرةُ حذاء) بعد العدد: «× 1 نمرة 40» — ولمجموعةٍ رقميّةٍ غيرِها
 *   اسمُها مختصراً بلا «ال».
 * - **وكلُّ قيمةٍ غيرِها** (دبل، أسود، L) تُلحق بالاسم.
 */
export function invoiceLineText(
  name: string,
  qty: number,
  options: { group?: string; name: string }[] | null | undefined,
): string {
  const plain: string[] = [];
  const after: string[] = [];
  for (const o of options ?? []) {
    const group = (o.group ?? "").trim();
    const value = (o.name ?? "").trim();
    if (!value) continue;
    if (group.includes("وزن") || WEIGHT.test(value)) {
      after.push(value);
    } else if (NUMBER_ONLY.test(value)) {
      const short = group.replace(/^ال/, "").trim();
      after.push(short.startsWith("نمرة") ? `نمرة ${value}` : short ? `${short} ${value}` : value);
    } else {
      plain.push(value);
    }
  }
  const head = plain.length ? `${name.trim()} ${plain.join(m.common.listSeparator)}` : name.trim();
  return `${head} × ${qty}${after.length ? " " + after.join(" ") : ""}`;
}

/**
 * **كلُّ صنفٍ ثمّ مشروباتُه** — المحرّكُ يرسلها هكذا (`ORDER BY COALESCE(drink_for, id)`)،
 * **والترتيبُ هنا حارسٌ.** ومشروبٌ لا صنفَ له في القائمة يُلحق آخراً ولا يضيع.
 */
export function invoiceOrder<T extends InvoiceLineItem>(lines: T[]): T[] {
  const out: T[] = [];
  const drinks = lines.filter((l) => l.drink_for);
  for (const item of lines.filter((l) => !l.drink_for)) {
    out.push(item);
    if (item.id) out.push(...drinks.filter((d) => d.drink_for === item.id));
  }
  for (const d of drinks) if (!out.includes(d)) out.push(d);
  return out;
}
