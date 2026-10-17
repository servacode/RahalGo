"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متاجرُ الحساب والمتجرُ المختار — قراءةٌ واحدةٌ للوحة كلِّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `SelectedStore.kt` و`StoreViewModel.kt` في تطبيق المتجر.)
 *
 * **ولا يُفترض أنّه متجرٌ واحد**: من ملك فرعين ثمّ رأى واحداً ظنّ الآخرَ
 * ضاع. **والمختارُ يُحفظ في المتصفّح** — فإن غاب (حُذف أو نُقل) أُخذ
 * الأوّل، كما يفعل التطبيق.
 *
 * **و`self_manage_orders` يأتي معها** — إعدادُ منصّةٍ يقرّر هل يقبل المتجرُ
 * طلباتِه بيده أم تقبلها المنصّة. **والمحرّكُ هو الحَكَم** (`orders/modes.go`)،
 * والشاشةُ تُخفي ما لا يملكه فقط.
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { readJSON, writeJSON } from "@/lib/customer/storage";
import type { MerchantStore, StoresPage } from "./types";

const PICK_KEY = "rahalgo_merchant_store";
/** **وكلُّ دقيقةٍ تُعاد القراءة** — حالُ الفتح يتبدّل بالساعات وبيد الإدارة. */
const REFRESH_MS = 60_000;

interface StoreCtx {
  loading: boolean;
  /** **خطأُ القراءة الأولى** — كما قاله الخادم (`errorText`). */
  error: unknown;
  stores: MerchantStore[];
  store: MerchantStore | null;
  selfManage: boolean;
  select: (id: string) => void;
  reload: () => Promise<void>;
  /** **تحديثٌ محلّيٌّ بعد ردٍّ ناجح** — لا ينتظر القراءةَ التالية. */
  patch: (fields: Partial<MerchantStore>) => void;
}

const Ctx = createContext<StoreCtx | null>(null);

export function StoreProvider({ children }: { children: ReactNode }) {
  const [page, setPage] = useState<StoresPage | null>(null);
  const [error, setError] = useState<unknown>(null);
  const [loading, setLoading] = useState(true);
  const [picked, setPicked] = useState("");

  useEffect(() => {
    setPicked(readJSON<string>(PICK_KEY, ""));
  }, []);

  const reload = useCallback(async () => {
    try {
      const p = await api<StoresPage>("/api/v1/merchant/stores");
      setPage(p);
      setError(null);
    } catch (e) {
      setError(e);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void reload();
    const t = setInterval(() => void reload(), REFRESH_MS);
    return () => clearInterval(t);
  }, [reload]);
  useLiveRefresh(["account", "merchant", "settings"], () => void reload());

  const stores = useMemo(() => page?.stores ?? [], [page]);
  const store = useMemo(
    () => stores.find((s) => s.id === picked) ?? stores[0] ?? null,
    [stores, picked],
  );

  const select = useCallback((id: string) => {
    setPicked(id);
    writeJSON(PICK_KEY, id);
  }, []);

  const patch = useCallback(
    (fields: Partial<MerchantStore>) => {
      setPage((p) =>
        p && store
          ? { ...p, stores: p.stores.map((s) => (s.id === store.id ? { ...s, ...fields } : s)) }
          : p,
      );
    },
    [store],
  );

  const value: StoreCtx = {
    loading,
    error: page ? null : error,
    stores,
    store,
    selfManage: page?.self_manage_orders ?? false,
    select,
    reload,
    patch,
  };
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useStore(): StoreCtx {
  const c = useContext(Ctx);
  if (!c) throw new Error("useStore outside StoreProvider");
  return c;
}
