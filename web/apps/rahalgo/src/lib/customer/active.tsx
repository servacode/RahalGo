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
  /**
   * **حديثٌ فتحه صاحبُه فقرأه** — يسقط عدُّه الآن لا مع النبضة التالية.
   *
   * **والمحرّكُ يَسِمُ الرسائلَ مقروءةً مع قراءة الحديث لكنّه لا يبعث للقارئ شيئاً**،
   * فيبقى العددُ القديمُ على المربّع وعلى الفقّاعة حتّى تأتي قراءةٌ أخرى. **وتأتي
   * الحقيقةُ من `reload` بعد وصول الحديث** فإن اختلفت ربحت.
   */
  markRead: (orderId: string) => void;
}

const Ctx = createContext<ActiveState>({ open: null, chats: [], reload: () => undefined, markRead: () => undefined });

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

  const markRead = useCallback((orderId: string) => {
    // **وإن لم يكن فيه ما يُصفَّر تعود القائمةُ نفسُها** — فلا إعادةَ رسمٍ بلا سبب.
    setChats((rows) =>
      rows.some((r) => r.order_id === orderId && r.unread > 0)
        ? rows.map((r) => (r.order_id === orderId ? { ...r, unread: 0 } : r))
        : rows,
    );
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

  const value = useMemo(() => ({ open, chats, reload, markRead }), [open, chats, reload, markRead]);
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useActiveOrders(): ActiveState {
  return useContext(Ctx);
}
