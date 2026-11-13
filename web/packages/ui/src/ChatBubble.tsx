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
  onOpen,
}: {
  api: ApiFn;
  /** الطلباتُ المفتوحةُ التي لها طرفٌ ثانٍ — يبنيها من يعرف شاشتَه. */
  threads: ChatThread[];
  onRead?: () => void;
  /** حديثٌ فُتح أو بُدِّل إليه — **صاحبُ `threads` يُصفِّر عدَّه الآن** ثمّ تأتي الحقيقةُ من `onRead`. */
  onOpen?: (orderId: string) => void;
}) {
  const [pos, setPos] = useState({ x: 0, y: 0 });
  const drag = useRef({ sx: 0, sy: 0, ox: 0, oy: 0, on: false, moved: false, endedAt: 0 });
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
   * **ومن فتح حديثاً أو بدّل إليه قرأه** — يسقط عدُّه (شارةُ مربّعه وشارةُ الفقّاعة) الآن.
   *
   * **والمحرّكُ يَسِمُ الرسائلَ مقروءةً مع قراءة الحديث لكنّه لا يبعث للقارئ شيئاً**: بلا هذا
   * يبقى العددُ القديمُ على مربّعٍ قُرئ، وتبقى الفقّاعةُ على مجموعها القديم حتّى تأتي قراءةٌ
   * أخرى. **وما يصل والحديثُ مفتوحٌ يُقرأ حيّاً** — فيسقط كذلك.
   */
  const pickedUnread = threads.find((t) => t.id === picked)?.unread ?? 0;
  useEffect(() => {
    if (open && picked && pickedUnread > 0) onOpen?.(picked);
  }, [open, picked, pickedUnread, onOpen]);

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
   *
   * **والحديثُ المعروضُ في اللوح المفتوح لا يدخل العدّ**: صاحبُه يقرؤه الآن، **وعدُّه يرجع قديماً
   * مرّةً بعد مرّة** — جلبٌ تأخّر عن وسم القراءة، أو تبديلٌ سريعٌ بين المربّعات، أو قناةٌ حيّةٌ
   * مقطوعةٌ لا يُعاد معها جلبُ الحديث — **ورنّةٌ عن رسالةٍ يقرؤها كاذبة.** وكذا في التطبيق: الرنّةُ
   * ساكتةٌ ما دام اللوحُ مفتوحاً. **أمّا رسالةٌ على مربّعٍ آخرَ فتبقى تُنبّه.**
   */
  const seen = useRef<number | null>(null);
  const chime = useChime();
  useEffect(() => {
    const total = threads.reduce((n, t) => n + (open && t.id === picked ? 0 : t.unread), 0);
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
  }, [threads, chime, open, picked]);

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
      {/* **يُسحب بالإصبع أو الفأرة** (قرارُ المالك ٢٠٢٦-١١-١٣: «نقدر نحركها لنبعده عن مكانه») — ويبقى حيث تُرك
          طولَ الجلسة، **ورفعةُ الإصبع بعد السحب لا تفتح الحديث.** مرآةُ قرص التطبيق. */}
      {!open && (
        <button
          type="button"
          aria-label={C.openChat}
          onClick={() => {
            if (Date.now() - drag.current.endedAt > 350) setOpen(true);
          }}
          onPointerDown={(e) => {
            drag.current = { ...drag.current, sx: e.clientX, sy: e.clientY, ox: pos.x, oy: pos.y, on: true, moved: false };
            e.currentTarget.setPointerCapture(e.pointerId);
          }}
          onPointerMove={(e) => {
            const d = drag.current;
            if (!d.on) return;
            const dx = e.clientX - d.sx;
            const dy = e.clientY - d.sy;
            if (!d.moved && Math.hypot(dx, dy) < 6) return;
            d.moved = true;
            const maxX = window.innerWidth - 56 - 32;
            const maxY = window.innerHeight - 56 - 160;
            const rtl = getComputedStyle(e.currentTarget).direction === "rtl";
            const nx = d.ox + dx;
            setPos({ x: rtl ? Math.max(-maxX, Math.min(0, nx)) : Math.max(0, Math.min(maxX, nx)), y: Math.max(-maxY, Math.min(0, d.oy + dy)) });
          }}
          onPointerUp={() => {
            if (drag.current.moved) drag.current.endedAt = Date.now();
            drag.current.on = false;
          }}
          style={{ transform: `translate(${pos.x}px, ${pos.y}px)`, touchAction: "none" }}
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
        /* ══════════════════════════════════════════════════════════════
           **ولوحُ الحديث من لغة الأسطح**
           ══════════════════════════════════════════════════════════════

           (شكوى المالك ٢٠٢٦-٠٨-١٠: «شكلُ شاشة الدردشة مختلفٌ عن الثيم».)

           **كان `bg-paper` — وهو الأبيضُ الصريح** (‎#ffffff): لوحُ الطباعة
           والفواتير. **في منصّةٍ داكنةٍ يُقرأ ورقةً من عالمٍ آخرَ تطفو
           فوقها**، لا سطحاً منها.

           **و`surface-sheet` هو أخوه في اللغة**: قشرةٌ معتمةٌ لِما يقف
           **فوق** محتوًى لا على خلفيّة صفحة — **وهو ما تفعله هذه الفقّاعة
           بالضبط.**

           **وارتفاعٌ محدودٌ لا مفتوح**: حديثٌ بعشرين سطراً كان يمدّ اللوحَ
           حتّى يخرج رأسُه من الشاشة، **فيختفي زرُّ الإغلاق ولا تُغلق.** */
        <div className="fixed inset-x-3 bottom-20 z-40 flex max-h-[70vh] max-w-md flex-col surface-sheet p-3 sm:inset-x-auto sm:start-4 sm:w-96">
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
                    {/* **نقطةٌ بعدّادها على كلّ مربّعٍ فيه جديد** — لا يفتح الزبونُ كلَّ طلبٍ ليعرف. */}
                    {t.unread > 0 && t.id !== picked && (
                      <span className="ms-1.5 inline-flex h-4 min-w-4 items-center justify-center rounded-full bg-danger px-1 text-2xs font-bold text-on-bright">
                        {t.unread}
                      </span>
                    )}
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
          {/* **وبعد وصول الحديث تُقرأ الحقيقةُ من المحرّك** — فوصولُه هو وسمُ القراءة. */}
          {picked && <OrderChat api={api} orderId={picked} bare streamClass="min-h-40" onLoaded={refresh} />}
        </div>
      )}
    </>
  );
}
