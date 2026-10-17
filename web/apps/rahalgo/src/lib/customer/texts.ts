/**
 * ══════════════════════════════════════════════════════════════════════
 * **نصوصُ الزبون المركّبة — في موضعٍ واحد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مرآةٌ لـ`ServiceReason.kt` و`CartChanges.kt` و`Money.kt`** في أندرويد
 * — فالسببُ الواحدُ يُقال باللفظ نفسِه في الهاتف والمتصفّح.
 *
 * **ولا قرارَ هنا**: الخادمُ يقول السببَ رمزاً، **وهذه تختار اللفظ.**
 */

import { getMessages, defaultLocale, fmtMoney, fmtTime, withPlatform } from "@rahalgo/i18n";
import type { CartChange } from "./types";

const m = getMessages(defaultLocale);
const S = m.shop;

/** **يملأ `{x}` في نصٍّ من المعجم.** */
export function fill(t: string, vars: Record<string, string | number>): string {
  return t.replace(/\{(\w+)\}/g, (all, k: string) => (k in vars ? String(vars[k]) : all));
}

/** **السعرُ بالدولار** — بلا أصفارٍ زائدة: ‎$1.8 لا ‎$1.80. */
export function dollars(v: number): string {
  const cents = Math.round(v * 100);
  const text = cents % 100 === 0 ? String(cents / 100) : String(cents / 100).replace(/0+$/, "");
  return `$${text}`;
}

/** **«نعود الساعة…»** — من وقتٍ يرسله المحرّك، وفارغٌ إن لم يُرسل. */
export function backAt(raw: string | undefined | null): string {
  const v = (raw ?? "").trim();
  if (!v) return "";
  const d = new Date(v);
  if (Number.isNaN(d.getTime())) return "";
  return fmtTime(d);
}

export const REASON = {
  AVAILABLE: "service_available",
  LAUNCH_CLOSED: "launch_closed",
  PLATFORM_CLOSED_NOW: "platform_closed_now",
  ZONE_CLOSED_NOW: "zone_closed_now",
  ADDRESS_OUTSIDE_COVERAGE: "address_outside_coverage",
  CITY_NOT_SUPPORTED: "city_not_supported",
  PROVINCE_NOT_SUPPORTED: "province_not_supported",
  AREA_NOT_SUPPORTED: "area_not_supported",
} as const;

/**
 * **سببُ تعذّر الخدمة بلفظ الزبون** — مرآةُ `ServiceReason.text`.
 *
 * **ورسالةُ المالك تسبق كلَّ لفظ**: من كتب في اللوحة «مسكّرين للعيد»
 * قُرئت كما كتبها.
 */
export function serviceText(
  reason: string,
  message = "",
  nextAvailableAt = "",
): string {
  const own = message.trim();
  const errs = m.errors as unknown as Record<string, string>;
  const base = own || errs[reason] || S.service.unexpected;
  const time = backAt(nextAvailableAt);
  return time ? fill(S.service.backAt, { text: base, time }) : base;
}

/** **أيُعرض زرُّ «أضِف منطقتي» أو «أخبرني»؟** — مرآةُ `ctaKind`. */
export function demandKind(reason: string): "" | "coverage_request" | "service_interest" {
  if (reason === REASON.ADDRESS_OUTSIDE_COVERAGE) return "coverage_request";
  if (
    reason === REASON.CITY_NOT_SUPPORTED ||
    reason === REASON.PROVINCE_NOT_SUPPORTED ||
    reason === REASON.AREA_NOT_SUPPORTED
  )
    return "service_interest";
  return "";
}

export function demandCta(kind: string, platform: string): string {
  return kind === "coverage_request"
    ? S.service.requestCoverage
    : withPlatform(S.service.notifyMe, platform);
}

export function demandDone(kind: string, platform: string): string {
  return kind === "coverage_request"
    ? S.service.coverageDone
    : withPlatform(S.service.notifyDone, platform);
}

/** **ما تبدّل في السلّة منذ عُرضت** — مرآةُ `CartChanges.text`. */
export function changeText(c: CartChange): string {
  const C = S.changes;
  const name = c.name.trim() || C.unnamed;
  switch (c.type) {
    case "product_removed":
      return fill(C.removed, { name });
    case "product_unavailable":
      return fill(C.unavailable, { name });
    case "quantity_invalid":
      return fill(C.qty, { name });
    case "product_price_changed":
      return fill(C.price, { name, old: fmtMoney(c.old_value), new: fmtMoney(c.new_value) });
    case "delivery_fee_changed":
      return fill(C.fee, { old: fmtMoney(c.old_value), new: fmtMoney(c.new_value) });
    case "promo_or_discount_changed":
      return c.new_value === 0
        ? C.promoGone
        : fill(C.promo, { old: fmtMoney(c.old_value), new: fmtMoney(c.new_value) });
    default:
      return S.cart.reviewTitle;
  }
}

/** **اسمُ المرحلة** — والأمانةُ تُستلَم لا تُشترى. */
export function stageLabel(key: string, amanah: boolean): string {
  const G = S.orders.stages;
  switch (key) {
    case "buying":
      return amanah ? G.buyingAmanah : G.buying;
    case "bought":
      return amanah ? G.boughtAmanah : G.bought;
    case "waiting":
    case "accepted":
    case "preparing":
    case "seeking":
    case "on_the_way":
    case "arrived":
    case "delivered":
      return G[key];
    default:
      return key;
  }
}

/** **اسمُ الحال** — من المعجم المشترك، وما لا اسمَ له يُقال رمزاً لا فراغاً. */
export function statusLabel(status: string): string {
  const st = m.orders.status as Record<string, string>;
  return st[status] ?? status;
}

/**
 * **كودُ الاستلام من إشعاره** — حتّى يُرسله المحرّكُ حقلاً في الطلب.
 *
 * **المحرّكُ يُرسل الكودَ إشعاراً للزبون** (`orders/delivery_code.go`):
 * «كود استلام طلبك #١٠٨١ هو ٠٤٢٧ — …» **ولا يضعه في `/my/orders`.**
 * **فيُقرأ من صندوقه**: إشعارٌ على هذا الطلب بعينه، **وأربعةُ أرقامٍ
 * ليست رقمَ الطلب.**
 */
export function codeFromNotice(body: string, orderNumber: number): string {
  const hits = [...body.matchAll(/(?<![#\d])(\d{4})(?!\d)/g)].map((x) => x[1] ?? "");
  const own = String(orderNumber);
  return hits.find((h) => h && h !== own) ?? "";
}
