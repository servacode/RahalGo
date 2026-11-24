"use client";

/**
 * **شام كاش عند الزبون — معلوماتُ الحساب وطلباتُ الشحن**
 *
 * `GET /my/shamcash` يردّ المفاتيحَ التي يضبطها المالكُ في الإعدادات (شحنٌ وعند الاستلام وأقلُّ مبلغ
 * واسمُ الحساب ورقمُه وباركودُه) مع طلباتِ شحن الزبون. **والمبالغُ بالليرة الجديدة.**
 */

import { useCallback, useEffect, useState } from "react";
import { api } from "@/lib/api";

export type ShamcashInfo = {
  topup_enabled: boolean;
  pay_on_delivery_enabled: boolean;
  min_topup: number;
  account_name: string;
  account_number: string;
  qr_url: string | null;
};

export type ShamcashRequest = {
  id: string;
  amount: number;
  txn_ref: string;
  status: "pending" | "approved" | "rejected";
  reject_reason: string;
  created_at: string;
};

type Payload = { info: ShamcashInfo; requests: ShamcashRequest[] };

/** **يجلب معلوماتِ شام كاش** — `null` قبل الوصول أو إن تعذّر (فلا خيارَ يظهر ولا شيءَ ينكسر). */
export function useShamcash(enabled = true): {
  info: ShamcashInfo | null;
  requests: ShamcashRequest[];
  failed: boolean;
  reload: () => void;
} {
  const [data, setData] = useState<Payload | null>(null);
  const [failed, setFailed] = useState(false);
  const load = useCallback(() => {
    api<Payload>("/api/v1/my/shamcash")
      .then((r) => {
        setData(r);
        setFailed(false);
      })
      .catch(() => setFailed(true));
  }, []);
  useEffect(() => {
    if (enabled) load();
  }, [enabled, load]);
  return { info: data?.info ?? null, requests: data?.requests ?? [], failed, reload: load };
}
