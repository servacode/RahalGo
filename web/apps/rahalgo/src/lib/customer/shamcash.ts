"use client";

/**
 * **الدفع بشام كاش — جهةُ الزبون** (مشتركٌ بين المحفظة والسلّة وصفحة الحال).
 *
 * **والخيارُ مخفيٌّ لا مطفأٌ** حين `enabled=false`: مفاتيحُ شام كاش لم تُضبط بعد على السيرفر.
 *
 * **ونافذةُ الدفع تُفتح داخل الضغطة نفسِها** (`openPayTab`) — فحاجبُ النوافذ المنبثقة يمنع
 * ما يُفتح بعد انتظار الشبكة. تُفتح فارغةً ثمّ يُوجَّه عنوانُها حين يردّ الخادم (`goPay`).
 */

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

export interface ShamConfig {
  enabled: boolean;
  min: number;
  max: number;
  expires_minutes: number;
}

export type ShamStatus = "pending" | "paid" | "expired" | "failed" | "refunded" | "partly_refunded";

export interface ShamBill {
  id: string;
  number: string;
  purpose: "topup" | "order";
  amount: number;
  status: ShamStatus;
  created_at: string;
  expires_at: string;
  paid_at: string | null;
  order_id: string | null;
  order_number: number | null;
  /** غير فارغ = وصل الدفع وصار المبلغ بالمحفظة لكنّ الطلب لم يُنشأ. */
  order_error: string;
  payment_url: string;
}

export interface ShamStart {
  bill: ShamBill;
  payment_url: string;
}

/** مفتاحُ حفظ «هذه الفاتورة من السلّة» — لتُمسح السلّةُ حين يُنشأ الطلب لا قبله. */
export const SHAM_CART_KEY = "rahalgo_shamcash_cart";

/** **إعدادُ شام كاش للزبون** — `null` حتّى يردّ الخادم، وأيُّ فشلٍ = غيرُ مفعّل. */
export function useShamConfig(enabledFor: boolean): ShamConfig | null {
  const [cfg, setCfg] = useState<ShamConfig | null>(null);
  useEffect(() => {
    if (!enabledFor) return;
    let alive = true;
    api<ShamConfig>("/api/v1/my/shamcash")
      .then((c) => alive && setCfg(c))
      .catch(() => alive && setCfg(null));
    return () => {
      alive = false;
    };
  }, [enabledFor]);
  return cfg?.enabled ? cfg : null;
}

/** **يُستدعى متزامناً داخل معالج الضغطة.** */
export function openPayTab(): Window | null {
  try {
    return window.open("", "_blank");
  } catch {
    return null;
  }
}

/**
 * يوجّه نافذةَ الدفع. **ومحجوبةٌ؟ تنتقل هذه النافذةُ نفسُها** — وتعود من شام كاش إلى صفحة الحال.
 * @returns أبقيتَ على هذه النافذة (`true`) أم غادرتها (`false`)؟
 */
export function goPay(w: Window | null, url: string): boolean {
  if (w && !w.closed) {
    w.location.href = url;
    return true;
  }
  window.location.href = url;
  return false;
}

export function abortPay(w: Window | null): void {
  try {
    w?.close();
  } catch {
    /* نافذةٌ أُغلقت */
  }
}

export function statusPath(id: string): string {
  return `/wallet/shamcash/${id}`;
}
