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
  /** **ضبطُ المتجر الذي يملكه بيده** — يُقرأ في صفحة إعداداته. */
  default_prep_minutes: number;
  min_order: number;
}

interface StoreState {
  stores: Store[];
  /** تعذّرت قراءةُ المتاجر — **غيرُ «لا متاجرَ له».** */
  failed: boolean;
  store: Store | null;
  loading: boolean;
  /**
   * أيدير المتجرُ طلباته بنفسه؟ إعدادُ منصّةٍ يصل مع المتاجر.
   *
   * **الافتراضُ `true` عند الجهل**: من يدير طلباته يرى أزراره، ومن لا يديرها
   * تديرها المنصةُ عنه. **وإخفاءُ الأزرار خطأً يُجمّد متجراً**، وإظهارُها خطأً
   * يُظهر زرّاً يعمل — والأوّل أسوأ.
   */
  selfManage: boolean;
  refresh: () => Promise<void>;
  select: (id: string) => void;
}

const StoreContext = createContext<StoreState | null>(null);

const SELECTED_KEY = "rahalgo_merchant_store";

export function StoreProvider({ children }: { children: React.ReactNode }) {
  const [stores, setStores] = useState<Store[]>([]);
  const [selected, setSelected] = useState<string>("");
  const [loading, setLoading] = useState(true);
  /**
   * **وفشلُ القراءة ليس «لا متاجرَ لك».**
   *
   * كان `.catch(() => setStores([]))` — **فتصير اللوحةُ كلُّها فارغة**:
   * لا طلباتٍ ولا قائمةٍ ولا محفظة، **ويظنّ صاحبُ المطعم أنّ متجرَه حُذف.**
   * وهو يقف خلف الكاشير وطلباتُه تنتظر.
   */
  const [failed, setFailed] = useState(false);
  const [selfManage, setSelfManage] = useState(true);

  const refresh = useCallback(async () => {
    const res = await api<{ stores: Store[]; self_manage_orders: boolean } | Store[]>(
      "/api/v1/merchant/stores",
    );
    /* **وردٌّ بلا الحقل ليس ردّاً بمصفوفةٍ فارغة — لكنّه ليس سقوطاً.**

       (كُشف ٢٠٢٦-٠٨-٠٧ بمسبار كروت البلاغات: بوّابةُ المتجر كلُّها تسقط
       بـ`Cannot read properties of undefined (reading 'some')`.)

       **كان `res.stores` يُقرأ بلا حارس** — فأيُّ ردٍّ لا يحمل الحقلَ
       (نداءٌ رُدّ بجسمٍ ناقص، أو نسخةُ خادمٍ أقدم) **يرمي في `‎.some` فتبيضّ
       الشاشةُ كلُّها**: لا طلباتٍ ولا قائمةٍ ولا محفظة.

       **وسطرٌ واحدٌ في مزوّدٍ يعلو الشجرةَ يُسقط ما تحته كلَّه.**

       @empty-ok — الفراغُ هنا قرارٌ: صاحبُ المطعم يرى «لا متاجر» ويبقى
       باقي اللوحة يعمل، **وهو أهونُ من شاشةٍ بيضاء.** */
    const list = Array.isArray(res) ? res : (res?.stores ?? []);
    if (!Array.isArray(res)) setSelfManage(res.self_manage_orders !== false);
    setStores(list);
    setSelected((cur) => {
      const saved = cur || localStorage.getItem(SELECTED_KEY) || "";
      return list.some((s) => s.id === saved) ? saved : (list[0]?.id ?? "");
    });
  }, []);

  useEffect(() => {
    refresh()
      .catch(() => setFailed(true))
      .finally(() => setLoading(false));
  }, [refresh]);

  const select = useCallback((id: string) => {
    setSelected(id);
    localStorage.setItem(SELECTED_KEY, id);
  }, []);

  const store = stores.find((s) => s.id === selected) ?? null;

  return (
    <StoreContext.Provider value={{ stores, store, loading, failed, selfManage, select, refresh }}>
      {children}
    </StoreContext.Provider>
  );
}

export function useStore(): StoreState {
  const ctx = useContext(StoreContext);
  if (!ctx) throw new Error("useStore must be used within StoreProvider");
  return ctx;
}
