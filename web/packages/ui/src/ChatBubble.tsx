"use client";

/**
 * **فقّاعةُ المحادثة — تطفو ولا تسكن بطاقة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «الأفضل الدردشة بدل ما تكون داخل الكرت تظهر
 *  أيقونة دردشة عائمة تحصل فيها الدردشة».)
 *
 * # ولماذا تطفو
 *
 * **الحديثُ يجري والعملُ يجري**: السائقُ يقرأ العنوانَ ويضغط الملاحةَ ويعود
 * ليكتب. **ومحادثةٌ داخلَ بطاقةٍ تدفع ما تحتها** — فيبحث عن زرّه بين سطورٍ
 * تتحرّك.
 *
 * **وهي واحدةٌ لكلّ الطلبات المفتوحة**: لا فقّاعةٌ لكلّ بطاقة — **وثلاثُ
 * فقّاعاتٍ في زاويةٍ واحدةٍ تُخفي بعضَها.**
 *
 * # وشارةٌ تقول كم ينتظر
 *
 * **ومحادثةٌ لا تُعلِم بجديدها لا تُفتح** — يكتب أحدُهما ويظنّ أنّ الآخر يقرأ.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "./components";
import { IconChat, IconClose } from "./icons";
import { useLiveRefresh } from "./Notifications";
import { useChime } from "./chime";
import { OrderChat } from "./OrderChat";

const m = getMessages(defaultLocale);
const C = m.chat;

type ApiFn = <T = unknown>(path: string, init?: RequestInit) => Promise<T>;

/** طلبٌ له محادثةٌ مفتوحة — **الاسمُ ليُعرَف مع من يتكلّم.** */
export interface ChatThread {
  id: string;
  number: number;
  peer: string;
  unread: number;
}

export function ChatBubble({
  api,
  threads,
  onRead,
}: {
  api: ApiFn;
  /** الطلباتُ المفتوحةُ التي لها طرفٌ ثانٍ — يبنيها من يعرف شاشتَه. */
  threads: ChatThread[];
  onRead?: () => void;
}) {
  const [open, setOpen] = useState(false);
  const [picked, setPicked] = useState<string>("");

  // **وتُغلق حين لا يبقى ما يُحادَث** — طلبٌ سُلّم تُغلق قناتُه، **ونافذةٌ
  // مفتوحةٌ على قناةٍ مغلقةٍ تُقرأ عطباً.**
  useEffect(() => {
    if (threads.length === 0) {
      setOpen(false);
      setPicked("");
    } else if (!threads.some((t) => t.id === picked)) {
      setPicked(threads[0]?.id ?? "");
    }
  }, [threads, picked]);

  const refresh = useCallback(() => onRead?.(), [onRead]);
  useLiveRefresh(["order"], refresh);

  /**
   * **وتُفتح وحدَها عند أوّل رسالةٍ ويُنبَّه بصوت.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «وفتح الدردشة بشكلٍ تلقائيٍّ مع صوتٍ للتنبيه».)
   *
   * **ورسالةٌ تصل ولا تُرى لا تُجيب**: السائقُ على درّاجته والزبونُ في شغله،
   * **وفقّاعةٌ صامتةٌ في زاويةٍ تُقرأ بعد أن يمضي وقتُها.**
   *
   * **ولا تُفتح إلّا على جديدٍ حقيقيّ**: تُقارَن بما كان لا بما هو —
   * **ونافذةٌ تفتح نفسَها كلَّ تحديثٍ تُغلَق بغضبٍ ثمّ لا تُفتح.**
   */
  const seen = useRef<number | null>(null);
  const chime = useChime();
  useEffect(() => {
    const total = threads.reduce((n, t) => n + t.unread, 0);
    // **وأوّلُ قراءةٍ تُسجَّل ولا تُنبّه** — من فتح شاشتَه لا يُفاجَأ برنّةٍ
    // عن رسالةٍ قرأها أمس.
    if (seen.current === null) {
      seen.current = total;
      return;
    }
    if (total > seen.current) {
      setOpen(true);
      chime();
    }
    seen.current = total;
  }, [threads, chime]);

  if (threads.length === 0) return null;
  const unread = threads.reduce((n, t) => n + t.unread, 0);

  return (
    <>
      {/* **والفقّاعةُ فوق الشريط السفليّ** — لا تحته فتُقصّ، ولا في وسط
          الشاشة فتغطّي ما يُقرأ.

          **وفي جهة البداية — عكسِ السلّة.** (قرارُ المالك ٢٠٢٦-٠٨-١٠:
          «انقل أيقونة الدردشة على اليمين لتكون عكس السلّة، بكلّ الواجهات».)

          **وكانتا في الجهة نفسِها**: السلّةُ في `end-8` والفقّاعةُ في
          `end-4` — **قرصان مستديران متراكبان في زاويةٍ واحدة**، والإبهامُ
          يقصد أحدَهما فيصيب الآخر.

          **و`start` لا `right`**: في عربيّةٍ هي اليمين، **وفي لغةٍ تُكتب
          يساراً هي اليسار** — فتبقى عكسَ السلّة أبداً. **ورقمٌ مكتوبٌ
          صراحةً ينقلب على الوجه الآخر ويصير فوقها.** */}
      {!open && (
        <button
          type="button"
          aria-label={C.openChat}
          onClick={() => setOpen(true)}
          className="fixed bottom-20 start-4 z-40 flex h-14 w-14 items-center justify-center rounded-full bg-accent text-on-bright elev-2"
        >
          <IconChat size={24} />
          {unread > 0 && (
            <span className="absolute -top-1 -start-1 flex h-6 min-w-6 items-center justify-center rounded-full bg-danger px-1.5 text-xs font-bold text-on-bright">
              {unread}
            </span>
          )}
        </button>
      )}

      {open && (
        <div className="fixed inset-x-3 bottom-20 z-40 max-w-md rounded-card bg-paper p-3 elev-2 sm:inset-x-auto sm:start-4 sm:w-96">
          <div className="mb-2 flex items-center justify-between gap-2">
            {/* **وتبويبُ الطلبات إن كانت أكثرَ من واحد** — ولا يظهر لواحد. */}
            {threads.length > 1 ? (
              <div className="flex min-w-0 flex-1 gap-1 overflow-x-auto">
                {threads.map((t) => (
                  <button
                    key={t.id}
                    type="button"
                    onClick={() => setPicked(t.id)}
                    className={`shrink-0 rounded-control px-2.5 py-1 text-xs ${
                      t.id === picked ? "bg-accent-tint text-ink" : "bg-field text-ink-muted"
                    }`}
                  >
                    #{t.number}
                    {t.unread > 0 ? ` (${t.unread})` : ""}
                  </button>
                ))}
              </div>
            ) : (
              <span className="truncate text-sm font-medium">{threads[0]?.peer ?? ""}</span>
            )}
            <Button variant="ghost" aria-label={m.common.close} onClick={() => setOpen(false)}>
              <IconClose size={18} />
            </Button>
          </div>
          {picked && <OrderChat api={api} orderId={picked} />}
        </div>
      )}
    </>
  );
}
