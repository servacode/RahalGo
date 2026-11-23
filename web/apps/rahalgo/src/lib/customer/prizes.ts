/**
 * ══════════════════════════════════════════════════════════════════════
 * **جوائز رحّال غو — منطقٌ صافٍ بلا شبكةٍ ولا واجهة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك: كلُّ ٥ دعواتٍ في الشهر = فرصةٌ بسحبٍ شهريّ، وللرابحين «رصيدُ جوائز» لا يُصرف إلّا
 * على أصناف الجوائز.) **والخادمُ يقرّر كلَّ شيء** — هنا تنسيقٌ وتقسيمٌ للعرض وحدَه، وكلُّ حقلٍ اختياريّ.
 */

import type { PrizeItem, PrizesMe } from "./types";

/** **الجوائزُ فعّالةٌ** — الخادمُ يقول، وبلا تاريخ إطلاقٍ مقروءٍ لا نخمّن. */
export function prizesLive(me: PrizesMe | null | undefined): boolean {
  return !!me && me.enabled === true;
}

/** **كم دعوةً بعد آخر فرصة** — «٣ من ٥»: تتصفّر عند كلّ فرصةٍ جديدة. */
export function progressInEntry(me: PrizesMe): { done: number; total: number } {
  const total = Math.max(1, me.invites_per_entry ?? 5);
  const left = me.invites_to_next ?? total;
  const done = Math.min(total, Math.max(0, total - left));
  return { done: left <= 0 ? 0 : done, total };
}

/** **شارةُ الرأس «٣/٥»** بأرقامٍ لاتينيّة. */
export function badgeText(me: PrizesMe): string {
  const { done, total } = progressInEntry(me);
  return `${done}/${total}`;
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
