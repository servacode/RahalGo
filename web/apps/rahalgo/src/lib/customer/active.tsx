"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُه الجارية — قراءةٌ واحدةٌ تخدم القشرةَ كلَّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ثلاثةٌ يحتاجونها في أيّ شاشة**: نافذةُ تأكيد السعر، وسؤالُ مشاركة
 * الموقع («الكابتن بالطريق إليك»)، وفقّاعةُ المحادثة مع السائق. **ومن قرأها
 * كلٌّ بنفسه قرأها ثلاثَ مرّاتٍ كلَّ ربع دقيقة على شبكةِ الرقّة.**
 *
 * **وتُعاد قراءتُها بالنبضة** (`order` من قناة البثّ) **وكلَّ ربع دقيقةٍ
 * احتياطاً** ما دامت الصفحةُ ظاهرة — **فمن أُغلقت قناتُه لا يفوته شيء.**
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { ChatRow, MyOrder, MyOrdersPage } from "./types";

const EVERY_MS = 15_000;

interface ActiveState {
  /** **الجاريةُ** — و`null` قبل أوّل قراءة. */
  open: MyOrder[] | null;
  /** **عدّادُ ما لم يُقرأ في كلّ حديث** — من `/my/chats`. */
  chats: ChatRow[];
  reload: () => void;
}

const Ctx = createContext<ActiveState>({ open: null, chats: [], reload: () => undefined });

export function ActiveOrdersProvider({ children }: { children: ReactNode }) {
  const [open, setOpen] = useState<MyOrder[] | null>(null);
  const [chats, setChats] = useState<ChatRow[]>([]);

  const reload = useCallback(() => {
    if (typeof document !== "undefined" && document.visibilityState === "hidden") return;
    api<MyOrdersPage>("/api/v1/my/orders?open_only=true&page=1&per_page=30")
      .then((p) => setOpen(p.orders ?? []))
      .catch(() => undefined); // **وقراءةٌ فشلت تُعاد بعد ربع دقيقة** — ولا تُسقط الشاشة.
    api<{ threads: ChatRow[] }>("/api/v1/my/chats")
      .then((r) => setChats(r.threads ?? []))
      .catch(() => undefined);
  }, []);

  useEffect(() => {
    reload();
    const t = setInterval(reload, EVERY_MS);
    const onShow = () => document.visibilityState === "visible" && reload();
    document.addEventListener("visibilitychange", onShow);
    return () => {
      clearInterval(t);
      document.removeEventListener("visibilitychange", onShow);
    };
  }, [reload]);

  useLiveRefresh(["order"], reload);

  const value = useMemo(() => ({ open, chats, reload }), [open, chats, reload]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useActiveOrders(): ActiveState {
  return useContext(Ctx);
}
