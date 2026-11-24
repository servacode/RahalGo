/**
 * ══════════════════════════════════════════════════════════════════════
 * **جوائز رحّال غو — منطقٌ صافٍ بلا شبكةٍ ولا واجهة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١١-٢٥: مسابقةُ نقاطٍ شهريّة، والأعلى نقاطاً ممّن بلغوا الحدّ يربحون «رصيدَ جوائز» لا
 * يُصرف إلّا على أصناف الجوائز.) **والخادمُ يقرّر كلَّ شيء** — هنا تنسيقٌ للعرض وحدَه، وكلُّ حقلٍ اختياريّ.
 */

import type { PrizeItem, PrizesMe } from "./types";

/** **الجوائزُ فعّالةٌ** — الخادمُ يقول، وبلا تاريخ إطلاقٍ مقروءٍ لا نخمّن. */
export function prizesLive(me: PrizesMe | null | undefined): boolean {
  return !!me && me.enabled === true;
}

/** **تقدّمُه نحو حدّ التأهّل** — من صفرٍ إلى واحد؛ والمتأهّلُ واحد. */
export function qualifyProgress(me: PrizesMe): number {
  const min = Math.max(0, me.min_points ?? 0);
  if (min === 0) return 1;
  return Math.min(1, Math.max(0, (me.points ?? 0) / min));
}

/** **شارةُ الزرّ العائم** — نقاطُه هذا الشهر بأرقامٍ لاتينيّة. */
export function badgeText(me: PrizesMe): string {
  return String(Math.max(0, me.points ?? 0));
}

/** **رابطُ صفحةٍ يُفتح** — بلا بروتوكولٍ يُضاف `https://`؛ والفارغُ فارغ. */
export function pageUrl(raw: string | undefined): string {
  const v = (raw ?? "").trim();
  if (!v) return "";
  return /^https?:\/\//i.test(v) ? v : `https://${v}`;
}

/** **«١/١٢/٢٠٢٦»** من `2026-12-01` — ترتيبُ اليوم/الشهر/السنة بلا منطقةٍ زمنيّةٍ تقلب اليوم. */
export function launchLabel(launchOn: string | undefined): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(launchOn ?? "");
  if (!m) return "";
  return `${Number(m[3])}/${Number(m[2])}/${m[1]}`;
}

/** **صنفٌ على قدّ الرصيد؟** — السعرُ لا يزيد عنه. */
export function fitsBalance(price: number, balance: number): boolean {
  return price <= balance;
}

/** **الفرقُ الذي يدفعه الزبون** — صفرٌ إن كفى الرصيد. */
export function difference(price: number, balance: number): number {
  return Math.max(0, price - Math.max(0, balance));
}

/** **الأصنافُ فئتان**: «على قدّ جائزتك» ثمّ «بتدفع الفرق» — كلٌّ مرتّبٌ بالسعر. */
export function splitByBalance(items: PrizeItem[], balance: number): { fits: PrizeItem[]; over: PrizeItem[] } {
  const sorted = [...items].sort((a, b) => a.price - b.price);
  return {
    fits: sorted.filter((i) => fitsBalance(i.price, balance)),
    over: sorted.filter((i) => !fitsBalance(i.price, balance)),
  };
}
