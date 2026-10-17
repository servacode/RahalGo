"use client";

/**
 * **مراقبُ السعر — يسأل الزبونَ من أيّ شاشةٍ كان.**
 *
 * (مرآةُ `pendingQuote` في `MainActivity.kt`.) **وطلبٌ خاصٌّ عدّل المكتبُ
 * سعرَه بعد الشراء لا يُسلَّم حتّى يُؤكَّد** — **ومن لم يفتح «طلباتي» لا
 * يعلم.** فتُقرأ الطلباتُ الجاريةُ من القشرة (`ActiveOrdersProvider`)،
 * **والنافذةُ لا تُغلق إلّا بجواب.**
 */

import { useState } from "react";
import { useActiveOrders } from "@/lib/customer/active";
import { useCustomer } from "@/lib/customer/context";
import { QuoteDialog, quotePending } from "./orderParts";
import { useShopToast } from "./toast";

export default function QuoteWatcher() {
  const { available, reloadWallet } = useCustomer();
  const { open, reload } = useActiveOrders();
  const toast = useShopToast();
  const [done, setDone] = useState("");
  const pending = (open ?? []).find((o) => quotePending(o) && `${o.id}:${o.quote_version}` !== done) ?? null;

  if (!pending) return null;
  return (
    <QuoteDialog
      key={`${pending.id}:${pending.quote_version}`}
      order={pending}
      available={available}
      onMessage={(t) => toast(t)}
      onDone={() => {
        setDone(`${pending.id}:${pending.quote_version}`);
        reloadWallet();
        reload();
      }}
    />
  );
}
