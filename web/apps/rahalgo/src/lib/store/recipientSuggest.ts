/**
 * ══════════════════════════════════════════════════════════════════════
 * **اقتراحُ المستلِم وقتَ الكتابة — قواعدُه بلا شاشة** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «بمجرد ما يكتب اسم الزبون يتعبى باقي المعلومات بشكل تلقائي عنوان موقع رقم وكلشي».
 *
 * **مرآةُ `RecipientSuggest.kt`** في تطبيق المتجر — حرفان فأكثر، ٣٠٠ مللي ثانية بعد آخر حرف،
 * والرقمُ يُملأ بصيغة `09…`. **ويحرسها `check-store-parity.mjs`** فلا يفترق الموقعُ عن التطبيق.
 * **وبلا React ولا شبكة** ليُحمَّل في الحارس كما هو.
 */

import type { DeliveryRecipient } from "./types";

/** **انتظارُ ما بعد آخر حرف** — لا نداءَ مع كلّ ضغطة. */
export const RECIPIENT_DEBOUNCE_MS = 300;

/** **أقلُّ ما يُبحث به** — حرفٌ واحدٌ يطابق نصفَ الدفتر. */
export const RECIPIENT_MIN_CHARS = 2;

/** **أيُسأل الخادمُ عن هذا النصّ؟** — حرفان غيرُ الفراغ فأكثر. */
export function shouldSearchRecipients(text: string): boolean {
  return text.replace(/\s/g, "").length >= RECIPIENT_MIN_CHARS;
}

/** **الرقمُ كما يكتبه الناس** — `+9639XXXXXXXX` ⇒ `09XXXXXXXX`، وغيرُه كما هو. */
export function localPhone(phone: string): string {
  return phone.startsWith("+963") && phone.length === 13 ? "0" + phone.slice(4) : phone;
}

/** **ما يُملأ في النموذج** — والنقطةُ `null` إن أُرسلت آخرُ توصيلةٍ بلا نقطة. */
export function recipientFill(r: DeliveryRecipient): {
  name: string;
  phone: string;
  address: string;
  point: { lat: number; lng: number } | null;
} {
  return {
    name: r.name,
    phone: localPhone(r.phone),
    address: r.address_text,
    point: r.lat != null && r.lng != null ? { lat: r.lat, lng: r.lng } : null,
  };
}
