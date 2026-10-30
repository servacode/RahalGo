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
  /** **سطرُ مشروبٍ مدموج** — معرّفاتُ الأسطر التي جُمعت فيه (`invoiceOrder`). */
  merged_ids?: string[];
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

/** **مفتاحُ دمج المشروب** — اسمُه وقيمُ خياراته؛ فـ«كولا» مع صنفين سطرٌ واحد. */
function drinkKey(l: InvoiceLineItem): string {
  const opts = (l.options ?? []).map((o) => (o.name ?? "").trim()).filter(Boolean).sort();
  return [l.name.trim(), ...opts].join("|");
}

/**
 * **الأصنافُ كلُّها أوّلاً، ثمّ المشروباتُ كلُّها تحتها** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — لا تحت كلّ صنف.
 *
 * **والمشروبُ نفسُه مع أكثرَ من صنفٍ سطرٌ واحد**: عددُه مجموعٌ ومجموعُه مجموع
 * (`unit_price` = المجموع ÷ العدد إن اختلف سعرُه)، **و`merged_ids` معرّفاتُ أسطره**.
 * وترتيبُ المشروبات بأوّل ظهورها. **وطلبٌ قديمٌ مشروبُه داخل خيارات الصنف لا يتغيّر.**
 */
export function invoiceOrder<T extends InvoiceLineItem>(lines: T[]): (T & { merged_ids?: string[] })[] {
  const items = lines.filter((l) => !l.drink_for);
  const drinks = new Map<string, { first: T; qty: number; total: number; ids: string[]; n: number; samePrice: boolean }>();
  for (const d of lines) {
    if (!d.drink_for) continue;
    const k = drinkKey(d);
    const was = drinks.get(k);
    if (!was) {
      drinks.set(k, { first: d, qty: d.qty, total: d.unit_price * d.qty, ids: d.id ? [d.id] : [], n: 1, samePrice: true });
    } else {
      was.n += 1;
      was.qty += d.qty;
      was.total += d.unit_price * d.qty;
      if (d.id) was.ids.push(d.id);
      if (d.unit_price !== was.first.unit_price) was.samePrice = false;
    }
  }
  const merged: (T & { merged_ids?: string[] })[] = [];
  for (const g of drinks.values()) {
    if (g.n === 1) {
      merged.push(g.first);
      continue;
    }
    merged.push({
      ...g.first,
      qty: g.qty,
      unit_price: g.samePrice || g.qty === 0 ? g.first.unit_price : g.total / g.qty,
      merged_ids: g.ids,
    });
  }
  return [...items, ...merged];
}
