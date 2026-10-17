/**
 * ══════════════════════════════════════════════════════════════════════
 * **الدولارُ في لوحة المتجر — قواعدُ التطبيق نفسُها حرفاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `ItemDraft.kt` في `:ui` — `parseUsd` و`usdText` و`usdToSyp`.)
 *
 * **والمحرّكُ هو الحَكَم** (`catalog.USDToSYP`): ما يُعرض هنا معاينةٌ قبل
 * الحفظ، **والليرةُ التي تُكتب يحسبها الخادمُ من الدولار وسعرِ الصرف.**
 */

/** **مطابقٌ لـ`usdToSyp` في التطبيق ولـ`USDToSYP` في المحرّك** — لأقرب خمس ليرات. */
export function usdToSyp(usd: number, rate: number): number {
  return Math.round((usd * 100 * rate) / 100 / 5) * 5;
}

/** **الفاصلةُ العشريّةُ العربيّة والفاصلةُ والنقطةُ سواء** — مرآةُ `parseUsd`. */
const ARABIC_DECIMAL = "٫";
export function parseUsd(text: string): number | null {
  const t = text.trim();
  if (!t) return null;
  const v = Number(t.replace(ARABIC_DECIMAL, ".").replace(",", "."));
  return Number.isFinite(v) && v >= 0 ? v : null;
}

/** **نصُّ الدولار بلا أصفارٍ زائدة** — `1.5` لا `1.50`، و`2` لا `2.00`. */
export function usdText(v: number | null | undefined): string {
  if (v == null) return "";
  return String(Math.round(v * 100) / 100);
}

/** **ما يُكتب في حقل الدولار** — أرقامٌ وفاصلةٌ عشريّةٌ واحدة، بحدٍّ كالتطبيق. */
export function usdInput(v: string, max = 10): string {
  return v.replace(/[^\d.,٫]/g, "").slice(0, max);
}
