"use client";

/**
 * **«خلّي الشاشة صاحية» ما دامت لوحةُ الطلبات مفتوحة** (`navigator.wakeLock`).
 *
 * **وصفحةٌ نامت شاشتُها لا ترنّ** — آيفون يجمّد الصفحةَ حين تنطفئ الشاشة،
 * **ولا إشعاراتِ دفعٍ للمتصفّح عندنا.** فالقفلُ هو ما يُبقي الرنّةَ حيّة.
 *
 * **والمتصفّحُ يفكّه وحدَه** حين تغيب الصفحة (تبويبٌ آخر، قفلُ الجهاز) —
 * **فيُطلب من جديد عند عودتها** (`visibilitychange`). **والرفضُ يُحتمَل**:
 * متصفّحٌ قديمٌ أو بطاريّةٌ ضعيفة — يبقى التلميحُ «خلّي الصفحة مفتوحة».
 */

import { useEffect, useState } from "react";

interface Sentinel {
  release: () => Promise<void>;
  addEventListener: (t: "release", fn: () => void) => void;
}
interface WakeNav {
  wakeLock?: { request: (t: "screen") => Promise<Sentinel> };
}

export function useWakeLock(active: boolean): { held: boolean } {
  const [held, setHeld] = useState(false);

  useEffect(() => {
    if (!active || typeof navigator === "undefined") return;
    const wl = (navigator as unknown as WakeNav).wakeLock;
    if (!wl) return;
    let sentinel: Sentinel | null = null;
    let alive = true;

    const acquire = async () => {
      if (!alive || document.visibilityState !== "visible" || sentinel) return;
      try {
        const s = await wl.request("screen");
        if (!alive) {
          void s.release().catch(() => undefined);
          return;
        }
        sentinel = s;
        setHeld(true);
        s.addEventListener("release", () => {
          sentinel = null;
          setHeld(false);
        });
      } catch {
        // **رُفض — يُحتمَل ولا يُعرض خطأ.** التلميحُ ظاهرٌ على كلّ حال.
        setHeld(false);
      }
    };
    const onVis = () => {
      if (document.visibilityState === "visible") void acquire();
    };

    void acquire();
    document.addEventListener("visibilitychange", onVis);
    // **وبعض المتصفّحات لا تمنحه إلّا بعد لمسة** — فتُعاد المحاولةُ عندها.
    window.addEventListener("pointerdown", onVis, { passive: true });
    return () => {
      alive = false;
      document.removeEventListener("visibilitychange", onVis);
      window.removeEventListener("pointerdown", onVis);
      if (sentinel) void sentinel.release().catch(() => undefined);
      sentinel = null;
      setHeld(false);
    };
  }, [active]);

  return { held };
}
