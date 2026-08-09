"use client";

/**
 * **فقّاعةُ محادثة الزبون — كأختِها عند السائق.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «يجب أن تظهر أيقونة المحادثة عند الطرفين ليبقى
 *  كرت الطلبات نظيفاً عند السائق وعند الزبون».)
 *
 * # وتُخفي نفسَها حتّى يكون لها معنى
 *
 * **لا تظهر لزائرٍ ولا لمن لا طلبَ له بسائق** — **وأيقونةٌ تفتح فراغاً تُقرأ
 * عطباً**، ثمّ لا تُضغط حين يصير لها ما تقول.
 *
 * # ولا تُجلَب إلّا لمن دخل
 *
 * **ونداءٌ لطلباتٍ من زائرٍ يردّ ٤٠١ في كلّ صفحةٍ يفتحها** — يملأ السجلَّ بما
 * ليس خطأً.
 */

import { useCallback, useEffect, useState } from "react";
import { ChatBubble, type ChatThread, useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

interface MyOrder {
  id: string;
  number: number;
  status: string;
  closed_at: string | null;
  driver_id?: string | null;
  driver_name?: string | null;
}

export function CustomerChat() {
  const { user } = useAuth();
  const [threads, setThreads] = useState<ChatThread[]>([]);

  const load = useCallback(() => {
    if (!user) {
      setThreads([]);
      return;
    }
    api<MyOrder[]>("/api/v1/my/orders")
      .then((list) => {
        setThreads(
          (Array.isArray(list) ? list : [])
            // **وما دام في يد سائق** — القناةُ تُغلق بانتهاء الطلب،
            // **وتبويبٌ لطلبٍ مغلقٍ يفتح شاشةً لا تُكتب.**
            .filter((o) => !o.closed_at && o.driver_id)
            .map((o) => ({
              id: o.id,
              number: o.number,
              peer: o.driver_name ?? "",
              unread: 0,
            })),
        );
      })
      // **الفقّاعةُ عنصرٌ عائمٌ فوق كلّ صفحة** — وإنذارٌ يطفو على كلّ شاشةٍ
      // لأنّ نداءً تعثّر يُقرأ عطباً في المنصّة كلِّها. والطلباتُ تُعرض في
      // صفحتها مع خطئها إن وقع.
      // @empty-ok **الفراغُ صوابٌ** — الفقّاعةُ تُخفي نفسَها ولا تُنذر.
      .catch(() => setThreads([]));
  }, [user]);

  useEffect(load, [load]);
  useLiveRefresh(["order"], load);

  if (!user) return null;
  return <ChatBubble api={api} threads={threads} onRead={load} />;
}
