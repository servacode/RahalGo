"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُ المتجر — قراءةٌ واحدةٌ ورنّةٌ واحدةٌ للوحة كلِّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `OrdersViewModel.kt` و`MerchantOrderAlert.kt` في تطبيق المتجر.)
 *
 * **تُقرأ هنا لا في صفحة الطلبات** — فصاحبُ المتجر قد يكون في «الأصناف»
 * أو «توصيلة» حين يصل الطلب، **والرنّةُ لا تنتظر أن يفتح الصفحة.**
 *
 * **ومصدران للتحديث**: قناةُ البثّ (تصل فوراً) **وقراءةٌ كلَّ عشر ثوانٍ**
 * — فقناةٌ انقطعت بصمتٍ لا تُضيّع طلباً.
 *
 * **والرنّةُ تتكرّر ما دام طلبٌ ينتظر القبول** — كما في التطبيق: لا تسكت
 * إلّا بـ«استلمت الطلب» أو «اعتذار». **وفي وضع «المنصّة تدير» لا رنّة** —
 * المتجرُ لا يملك القبولَ أصلاً (`orders/modes.go`).
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { playChime, useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useRinger } from "@/lib/ringer";
import { useStore } from "./context";
import type { MerchantOrder, OrdersPage } from "./types";

const REFRESH_MS = 10_000;
/** **مفتاحُ تفضيل الصوت** — نفسُه الذي كانت تقرؤه بوّابةُ المتجر القديمة. */
export const MERCHANT_SOUND_KEY = "rahalgo_merchant_sound";

interface OrdersCtx {
  orders: MerchantOrder[] | null;
  error: unknown;
  pending: MerchantOrder[];
  reload: () => Promise<void>;
  /** **أيسمح المتصفّحُ بالصوت الآن؟** — آيفون يمنعه قبل أوّل لمسة. */
  soundUnlocked: boolean;
  unlockSound: () => void;
}

const Ctx = createContext<OrdersCtx | null>(null);

export function StoreOrdersProvider({ children }: { children: ReactNode }) {
  const { store, selfManage } = useStore();
  const storeId = store?.id ?? "";
  const [orders, setOrders] = useState<MerchantOrder[] | null>(null);
  const [error, setError] = useState<unknown>(null);

  const reload = useCallback(async () => {
    if (!storeId) return;
    try {
      const p = await api<OrdersPage>(`/api/v1/merchant/stores/${storeId}/orders`);
      setOrders(p.orders ?? []);
      setError(null);
    } catch (e) {
      setError(e);
    }
  }, [storeId]);

  useEffect(() => {
    setOrders(null);
    void reload();
    const t = setInterval(() => {
      if (document.visibilityState === "visible" || selfManage) void reload();
    }, REFRESH_MS);
    const onVis = () => document.visibilityState === "visible" && void reload();
    document.addEventListener("visibilitychange", onVis);
    return () => {
      clearInterval(t);
      document.removeEventListener("visibilitychange", onVis);
    };
  }, [reload, selfManage]);
  useLiveRefresh(["order", "merchant"], () => void reload());

  const pending = useMemo(
    () => (selfManage ? (orders ?? []).filter((o) => o.status === "pending") : []),
    [orders, selfManage],
  );

  const ringer = useRinger(pending.length > 0, MERCHANT_SOUND_KEY);

  // ══════════════════════════════════════════════════════════════════
  // **والاهتزازُ حيث يُدعَم** — أندرويد يهتزّ، وآيفون لا يملك الواجهة.
  // ══════════════════════════════════════════════════════════════════
  const ringing = pending.length > 0;
  const vibTimer = useRef<ReturnType<typeof setInterval> | null>(null);
  useEffect(() => {
    const nav = typeof navigator !== "undefined" ? navigator : null;
    if (!ringing || !nav || typeof nav.vibrate !== "function") return;
    const buzz = () => {
      try {
        nav.vibrate([400, 200, 400]);
      } catch {
        /* **متصفّحٌ يرفض الاهتزاز بلا لمسة** — الرنّةُ تكفي. */
      }
    };
    buzz();
    vibTimer.current = setInterval(buzz, 6000);
    return () => {
      if (vibTimer.current) clearInterval(vibTimer.current);
      vibTimer.current = null;
    };
  }, [ringing]);

  const value: OrdersCtx = {
    orders,
    error: orders ? null : error,
    pending,
    reload,
    soundUnlocked: ringer.unlocked,
    unlockSound: () => {
      if (!ringer.enabled) ringer.setEnabled(true);
      ringer.unlock();
      // **ورنّةٌ قصيرةٌ تثبت أنّ الصوتَ وصل** — من ضغط ولم يسمع شيئاً يعرف
      // أنّ صوتَ الجهاز مكتوم، لا أنّ الصفحةَ معطّلة.
      playChime(true);
    },
  };
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useStoreOrders(): OrdersCtx {
  const c = useContext(Ctx);
  if (!c) throw new Error("useStoreOrders outside StoreOrdersProvider");
  return c;
}
