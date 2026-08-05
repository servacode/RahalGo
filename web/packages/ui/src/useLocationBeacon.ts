"use client";

/**
 * نبضةُ الموضع — **يُرسلها من يعمل وحدَه.**
 *
 * # لماذا
 *
 * الترتيبُ يوزّع بالدور: من طال انتظارُه يأخذ. **وهو عدلٌ في الوقت وأعمى في
 * المكان** — فيُسنَد طلبٌ من مطعمٍ في المنصور إلى سائقٍ في الرميلة، وآخرُ
 * يقف أمام المطعم نفسِه لا يُعرض عليه شيء.
 *
 * **والمسافةُ لا تُقاس بلا نقطة.**
 *
 * # ولا تُرسَل إلّا في الدوام
 *
 * **موضعُ من ليس على الدوام لا يُسأل عنه أحد** — ومتابعتُه استنزافُ بطّاريةٍ
 * وتعقّبٌ بلا سبب. **ومن أُغلق دوامُه أُغلقت نبضتُه.**
 *
 * # والإخفاقُ يمرّ صامتاً
 *
 * رفضُ إذن الموقع، أو جهازٌ بلا GPS، أو نفقٌ تحت الأرض — **ولا شيءَ من ذلك
 * يمنعه من العمل**: تُقرأ المسافةُ «لا تُعرف»، والبطاقاتُ تظهر كما هي.
 * **وشاشةٌ تسقط لأنّ القمرَ الصناعيَّ لم يُجب شاشةٌ لا تُحتمَل.**
 */

import { useEffect } from "react";

type Sender = (path: string, init?: RequestInit) => Promise<unknown>;

/**
 * @param api   عميلُ الطلبات المُصادَق.
 * @param on    ارفعه مع الدوام وأنزله معه.
 * @param everyMs كم بين نبضةٍ وأخرى — **دقيقةٌ تكفي**: السائقُ على درّاجةٍ
 *                يقطع في الدقيقة نصفَ كيلومتر، **وأدقُّ من ذلك يستنزف بلا
 *                فائدةٍ في قرارٍ يُقاس بالمئات.**
 */
export function useLocationBeacon(api: Sender, on: boolean, everyMs = 60_000): void {
  useEffect(() => {
    if (!on || typeof navigator === "undefined" || !navigator.geolocation) return;

    let alive = true;
    const send = () => {
      navigator.geolocation.getCurrentPosition(
        (pos) => {
          if (!alive) return;
          void api("/api/v1/driver/location", {
            method: "POST",
            body: JSON.stringify({
              lat: pos.coords.latitude,
              lng: pos.coords.longitude,
              // **وما يقوله الجهازُ عن نفسِه يُرسَل ولا يُخمَّن** — نقطةٌ
              // بدقّةِ خمسِمئة مترٍ ليست نقطة، **والوقوفُ يُقاس بالأمتار.**
              speed_mps: pos.coords.speed,
              accuracy_m: pos.coords.accuracy,
            }),
          }).catch(() => undefined);
        },
        // **ورفضُ الإذن لا يُسقط شيئاً** — يبقى الموضعُ مجهولاً وتُقرأ
        // المسافةُ «لا تُعرف».
        () => undefined,
        { enableHighAccuracy: true, maximumAge: 30_000, timeout: 15_000 },
      );
    };

    send();
    const id = setInterval(send, everyMs);
    return () => {
      alive = false;
      clearInterval(id);
    };
  }, [api, on, everyMs]);
}

/** مسافةٌ مقروءة — **بالمتر تحت الكيلو وبالكيلو فوقه**، وفراغٌ لما لا يُعرف. */
export function fmtDistance(meters: number, unitM: string, unitKm: string): string {
  if (!(meters >= 0)) return "";
  if (meters < 1000) return `${Math.round(meters)} ${unitM}`;
  return `${(meters / 1000).toFixed(1)} ${unitKm}`;
}
