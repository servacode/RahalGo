"use client";

/** سياق متجر البوابة: متاجر صاحب الحساب والمتجر النشط (مع مبدّل عند التعدد). */

import { createContext, useCallback, useContext, useEffect, useState } from "react";
import { api } from "./api";

export interface Store {
  id: string;
  name: string;
  category_icon: string;
  logo_thumb_url: string | null;
  status: string;
  emergency_closed: boolean;
}

interface StoreState {
  stores: Store[];
  store: Store | null;
  loading: boolean;
  select: (id: string) => void;
  refresh: () => Promise<void>;
}

const StoreContext = createContext<StoreState | null>(null);

const SELECTED_KEY = "rahalgo_merchant_store";

export function StoreProvider({ children }: { children: React.ReactNode }) {
  const [stores, setStores] = useState<Store[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [loading, setLoading] = useState(true);

  const refresh = useCallback(async () => {
    const list = await api<Store[]>("/api/v1/merchant/stores");
    setStores(list);
    setSelected((cur) => {
      const saved = cur || localStorage.getItem(SELECTED_KEY) || "";
      return list.some((s) => s.id === saved) ? saved : (list[0]?.id ?? "");
    });
  }, []);

  useEffect(() => {
    refresh()
      .catch(() => setStores([]))
      .finally(() => setLoading(false));
  }, [refresh]);

  const select = useCallback((id: string) => {
    setSelected(id);
    localStorage.setItem(SELECTED_KEY, id);
  }, []);

  const store = stores.find((s) => s.id === selected) ?? null;

  return (
    <StoreContext.Provider value={{ stores, store, loading, select, refresh }}>
      {children}
    </StoreContext.Provider>
  );
}

export function useStore(): StoreState {
  const ctx = useContext(StoreContext);
  if (!ctx) throw new Error("useStore must be used within StoreProvider");
  return ctx;
}
