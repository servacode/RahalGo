"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **توصيلاتُ المتجر — قائمةٌ واحدةٌ للوحة كلِّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `DeliveryViewModel.list` في تطبيق المتجر.)
 *
 * **تُقرأ هنا لا في صفحة «توصيلة»** — لأنّ نافذةَ «السائق دفع مبلغ بمكان
 * الاستلام» تُسأل من أيّ شاشة (`FetchAmountApproveDialog` في التطبيق
 * مركَّبةٌ فوق الكلّ). **والسائقُ واقفٌ عند المورّد ينتظر الجواب.**
 */

import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "./context";
import type { Delivery } from "./types";

const REFRESH_MS = 10_000;

/** **مطابقةٌ حرفيّةٌ لـ`DeliveryTexts.awaitsAmount`** — متى يُسأل صاحبُ المتجر. */
export function awaitsAmount(d: Delivery): boolean {
  return (
    d.delivery_mode === "fetch" &&
    d.status === "at_pickup" &&
    d.fetch_amount_status === "pending" &&
    d.fetch_amount != null
  );
}

/** **ما يُلغى** — مطابقٌ لـ`cancellable` في `DeliveryUi.kt`، **والمحرّكُ يحكم.** */
export const CANCELLABLE = new Set(["pending", "accepted", "dispatching", "assigned", "at_pickup"]);

interface DeliveriesCtx {
  list: Delivery[] | null;
  error: unknown;
  reload: () => Promise<void>;
}

const Ctx = createContext<DeliveriesCtx | null>(null);

export function StoreDeliveriesProvider({ children }: { children: ReactNode }) {
  const { store } = useStore();
  const storeId = store?.id ?? "";
  const [list, setList] = useState<Delivery[] | null>(null);
  const [error, setError] = useState<unknown>(null);

  const reload = useCallback(async () => {
    if (!storeId) return;
    try {
      const p = await api<{ deliveries: Delivery[] }>(`/api/v1/merchant/stores/${storeId}/deliveries`);
      setList(p.deliveries ?? []);
      setError(null);
    } catch (e) {
      setError(e);
    }
  }, [storeId]);

  useEffect(() => {
    setList(null);
    void reload();
    const t = setInterval(() => {
      if (document.visibilityState === "visible") void reload();
    }, REFRESH_MS);
    return () => clearInterval(t);
  }, [reload]);
  useLiveRefresh(["order", "delivery", "wallet"], () => void reload());

  return <Ctx.Provider value={{ list, error, reload }}>{children}</Ctx.Provider>;
}

export function useStoreDeliveries(): DeliveriesCtx {
  const c = useContext(Ctx);
  if (!c) throw new Error("useStoreDeliveries outside StoreDeliveriesProvider");
  return c;
}
