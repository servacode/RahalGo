"use client";

/**
 * **التبويبُ في الرابط** — صفحاتُ الملفّ (٣٦٠) تُشارَك برابطٍ يفتح التبويبَ نفسَه، والرجوعُ يعيدك إليه.
 *
 * يقرأ `?tab=` ويكتبه بـ`replace` (لا يملأ سجلَّ المتصفّح). وقيمةٌ مجهولةٌ تعود للافتراضيّ.
 */

import { useCallback } from "react";
import { usePathname, useRouter, useSearchParams } from "next/navigation";

export function useUrlTab<T extends string>(keys: readonly T[], fallback: T): [T, (t: T) => void] {
  const router = useRouter();
  const pathname = usePathname();
  const params = useSearchParams();
  const raw = params.get("tab");
  const tab = (keys as readonly string[]).includes(raw ?? "") ? (raw as T) : fallback;
  const set = useCallback(
    (t: T) => {
      const next = new URLSearchParams(params.toString());
      if (t === fallback) next.delete("tab");
      else next.set("tab", t);
      const qs = next.toString();
      router.replace(qs ? `${pathname}?${qs}` : pathname, { scroll: false });
    },
    [params, pathname, router, fallback],
  );
  return [tab, set];
}
