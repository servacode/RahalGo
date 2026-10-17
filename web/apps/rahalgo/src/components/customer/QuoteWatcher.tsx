"use client";

/**
 * **مراقبُ السعر — يسأل الزبونَ من أيّ شاشةٍ كان.**
 *
 * (مرآةُ `pendingQuote` في `MainActivity.kt`.) **وطلبٌ خاصٌّ عدّل المكتبُ
 * سعرَه بعد الشراء لا يُسلَّم حتّى يُؤكَّد** — **ومن لم يفتح «طلباتي» لا
 * يعلم.** فتُقرأ الطلباتُ الجاريةُ كلَّ نصف دقيقة، **والنافذةُ لا تُغلق
 * إلّا بجواب.**
 */

import { useCallback, useEffect, useState } from "react";
import { usePathname } from "next/navigation";
import { api } from "@/lib/api";
import { useCustomer } from "@/lib/customer/context";
import type { MyOrder, MyOrdersPage } from "@/lib/customer/types";
import { QuoteDialog, quotePending } from "./orderParts";
import { useShopToast } from "./toast";

const EVERY_MS = 30_000;

export default function QuoteWatcher() {
  const { available, reloadWallet } = useCustomer();
  const toast = useShopToast();
  const path = usePathname();
  const [pending, setPending] = useState<MyOrder | null>(null);

  const check = useCallback(() => {
    if (typeof document !== "undefined" && document.visibilityState === "hidden") return;
    api<MyOrdersPage>("/api/v1/my/orders?open_only=true&page=1&per_page=30")
      .then((p) => setPending((p.orders ?? []).find(quotePending) ?? null))
      .catch(() => undefined); // **وقراءةٌ فشلت تُعاد بعد نصف دقيقة** — ولا تُسقط الشاشةَ التي هو فيها.
  }, []);

  useEffect(() => {
    check();
    const t = setInterval(check, EVERY_MS);
    return () => clearInterval(t);
  }, [check, path]);

  if (!pending) return null;
  return (
    <QuoteDialog
      key={`${pending.id}:${pending.quote_version}`}
      order={pending}
      available={available}
      onMessage={(t) => toast(t)}
      onDone={() => {
        setPending(null);
        reloadWallet();
        check();
      }}
    />
  );
}
