"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عدّاداتُ اللوحة — نداءٌ واحدٌ يغذّي القائمةَ والرئيسيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إعادةُ تصميم اللوحة ٢٠٢٦-١١-٠٣.) `GET /admin/menu-counts` يردّ لكلّ موظّفٍ **ما يملك قدراتِه
 * فقط** (`null` لما سواه). والهيكلُ يجلبه مرّةً ويمرّره بسياقٍ، **فالقائمةُ وشريطُ الانتباه في
 * الرئيسيّة يقرآن الرقمَ نفسَه بنداءٍ واحد** — لا نداءٌ للقائمة ونداءٌ للرئيسيّة يفترقان.
 *
 * **حيٌّ**: يُعاد الجلبُ مع الأحداث (بتأخيرٍ يدمج المتلاحق) وكلَّ دقيقة احتياطاً.
 */

import { createContext, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";

type N = number | null | undefined;

export interface MenuCountsData {
  generated_at: string;
  today: string;
  day_start_hour: number;
  missing: string[];
  attention: {
    stuck_orders: N;
    unassigned_orders: N;
    reports_waiting: N;
    stores_silent: N;
    stores_silent_names: string[] | null;
    expansion_waiting: N;
    cash_over: N;
    cash_over_names: string[] | null;
    cash_overdue: N;
    payouts_pending: N;
    obligations_overdue: N;
    obligation_requests: N;
    compensations_pending: N;
    tickets_open: N;
    tickets_late: N;
    emergencies_open: N;
    emergencies_unacked: N;
    leads_new: N;
  };
  pulse: {
    orders_open: N;
    stages: Record<string, N>;
    drivers_on_shift: N;
    drivers_busy: N;
    drivers_free: N;
    stores_open_now: N;
    stores_active: N;
    avg_delivery_min: number | null;
  } | null;
  money: { cash_held: N; treasury_balance: N } | null;
}

interface Value {
  data: MenuCountsData | null;
  loading: boolean;
  error: boolean;
  reload: () => void;
  /** متى وصلت آخرُ أرقام (مللي ثانية) — للسطر الحيّ «تحدّث قبل…». */
  loadedAt: number | null;
}

const Ctx = createContext<Value>({ data: null, loading: true, error: false, reload: () => {}, loadedAt: null });

/** **الأحداثُ التي تغيّر أرقامَ الانتباه** — كما تسمعها الرئيسيّةُ القديمة. */
export const LIVE_KINDS = [
  "order", "orders", "account", "lead", "wallet", "alerts", "ticket", "emergency",
  "driver", "merchant", "dispute", "settings", "user", "payout", "menu",
];

export function MenuCountsProvider({ enabled, children }: { enabled: boolean; children: ReactNode }) {
  // **وكلَّ دقيقة** — العالقُ والمتاجرُ الفاتحةُ تتبدّل مع الوقت ولو لم يقع حدث.
  const [tick, setTick] = useState(0);
  const [loadedAt, setLoadedAt] = useState<number | null>(null);
  useEffect(() => {
    if (!enabled) return;
    const t = window.setInterval(() => {
      if (document.visibilityState === "visible") setTick((x) => x + 1);
    }, 60_000);
    return () => window.clearInterval(t);
  }, [enabled]);
  const { data, loading, error, reload } = useLiveData<MenuCountsData | null>(
    async () => {
      if (!enabled) return null;
      const d = await api<MenuCountsData>("/api/v1/admin/menu-counts");
      setLoadedAt(Date.now());
      return d;
    },
    LIVE_KINDS,
    [enabled, tick],
    800,
  );
  const value = useMemo<Value>(
    () => ({ data, loading, error, reload: () => void reload(), loadedAt }),
    [data, loading, error, reload, loadedAt],
  );
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useMenuCounts(): Value {
  return useContext(Ctx);
}
