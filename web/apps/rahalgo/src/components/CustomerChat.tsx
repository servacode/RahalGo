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

/** محادثةٌ كما يردّها الخادم — **مع عدّ ما لم يُقرأ.** */
interface Thread {
  order_id: string;
  number: number;
  peer: string;
  open: boolean;
  unread: number;
}

export function CustomerChat() {
  const { user } = useAuth();
  const [threads, setThreads] = useState<ChatThread[]>([]);

  const load = useCallback(() => {
    if (!user) {
      setThreads([]);
      return;
    }
    // **والردُّ كائنٌ لا مصفوفة** — `{orders, page, per_page, total}`.
    //
    // **وقرأتُه مصفوفةً فكان الفلترُ يردّ فراغاً دائماً** — فلم تظهر الفقّاعةُ
    // للزبون قطّ. **ولا خطأ ولا سجلّ**: `Array.isArray` تردّ `false` بهدوء،
    // **والمكوّنُ يُخفي نفسَه عند الفراغ فبدا كأنّه يعمل.**
    // **ومن سجلّ المحادثات لا من الطلبات** — **العدُّ يأتي معه**، وبلاه
    // كانت الشارةُ صفراً أبداً **فلا تُفتح الفقّاعةُ ولا يرنّ تنبيه.**
    api<{ threads: Thread[] }>("/api/v1/my/chats")
      .then((res) => {
        setThreads(
          (res?.threads ?? [])
            // **والمفتوحةُ وحدَها في الفقّاعة** — والمنتهيةُ في السجلّ.
            .filter((t) => t.open)
            .map((t) => ({
              id: t.order_id,
              number: t.number,
              peer: t.peer,
              unread: t.unread,
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
