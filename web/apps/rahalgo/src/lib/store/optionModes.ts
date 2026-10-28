/**
 * ══════════════════════════════════════════════════════════════════════
 * **طريقةُ سعر الخيارات** (قرارُ المالك ٢٠٢٦-١٠-٢٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 *   priced  «بسعر»       فرقُ سعرٍ لكلّ خيار (الحجم · العبوة · الإضافات).
 *   none    «بدون سعر»   لا سعر (المقاس · اللون · النمرة · بدون).
 *   weight  «حسب الوزن»  سعرُ الصنف لوزنٍ أساسيّ، ولكلّ خيارٍ غراماته.
 *
 * **والخادمُ يحسب السعرَ ويحفظه** (`weight_option_delta`، هجرة 0482) — وهذه
 * معاينةٌ بالمعادلة نفسِها، ومرآتُها `OptionPresets.kt` في التطبيقات.
 */

export type PriceMode = "priced" | "none" | "weight";

export const DEFAULT_BASE_GRAMS = 100;

/** **أوزانٌ سريعة** — ٢٥٠ غ ونصف كيلو وكيلو. */
export const QUICK_GRAMS = [250, 500, 1000];

export function priceModeOf(v: string | null | undefined): PriceMode {
  return v === "none" || v === "weight" ? v : "priced";
}

/** **اسمُ الوزن بالفصحى** — «250 غ»، «نصف كيلو»، «كيلو»، «1.5 كيلو». */
export function weightName(g: number): string {
  if (g === 500) return "نصف كيلو";
  if (g === 1000) return "كيلو";
  if (g > 1000 && g % 500 === 0) return `${g / 1000} كيلو`;
  return `${g} غ`;
}

/** **سعرُ الوزن كاملاً** = سعرُ الصنف × غراماته ÷ الأساس، لأقرب ٥ ل.س. */
export function weightPrice(
  itemPrice: number,
  grams: number,
  baseGrams: number | null | undefined,
): number {
  const base = baseGrams && baseGrams > 0 ? baseGrams : DEFAULT_BASE_GRAMS;
  return Math.round((itemPrice * grams) / base / 5) * 5;
}
