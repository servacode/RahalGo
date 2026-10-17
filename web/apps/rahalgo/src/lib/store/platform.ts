"use client";

/**
 * **رابطُ «شرح التطبيق بالفيديو»** — من إعدادات المنصّة
 * (`accounts.merchant_tutorial_url` يردّه `/public/platform`)، **كما يقرؤه
 * تطبيقُ المتجر** (`openTutorial` في `Menu.kt`).
 *
 * **وفراغُه يعود إلى الفيديو الذي يعود إليه التطبيق** — بندٌ في القائمة لا
 * يفتح شيئاً أسوأُ من رابطٍ قديم.
 */

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

/** **مطابقٌ لـ`TUTORIAL_URL_DEFAULT` في `Menu.kt`.** */
const TUTORIAL_FALLBACK = "https://youtu.be/cIQ_dXUjZ_g";

export function useTutorialUrl(): string {
  const [url, setUrl] = useState(TUTORIAL_FALLBACK);
  useEffect(() => {
    let alive = true;
    api<{ merchant_tutorial_url?: string }>("/api/v1/public/platform")
      .then((p) => {
        const u = (p?.merchant_tutorial_url ?? "").trim();
        if (alive && u) setUrl(u);
      })
      .catch(() => undefined);
    return () => {
      alive = false;
    };
  }, []);
  return url;
}
