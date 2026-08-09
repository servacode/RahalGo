"use client";

/**
 * **حديثُ الطلب — مكوّنٌ واحدٌ لطرفيه.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «لازم الاثنان لا يقدران يوصلان لبعض إلّا عن طريق
 *  المنصّة فقط».)
 *
 * # ولماذا واحدٌ لا اثنان
 *
 * **الشاشتان تعرضان الحديثَ نفسَه** — ونسختان منه تفترقان: يُصلَح شيءٌ في
 * لوحة السائق ويبقى عند الزبون، **فيرى أحدُهما ما لا يراه الآخر من محادثةٍ
 * بينهما.**
 *
 * **والخادمُ يعرف من هو من جلسته** — فلا يأخذ هذا المكوّنُ دوراً ولا معرّفاً،
 * **وكلُّ رسالةٍ تحمل `mine` من الخادم.**
 *
 * # ولا رقمَ ولا اسمَ شخصٍ يُطلب
 *
 * **يُعرَض اسمُ الطرف الآخر كما يردّه الخادم** (`peer_name`)، **ولا رقمَ معه
 * أبداً** — لا في الحمولة ولا في الشاشة.
 *
 * # والقناةُ تُقفل للكتابة لا للقراءة
 *
 * **حديثٌ يختفي بانتهاء الطلب يمحو ما يُحتجّ به** عند شكوى. فيبقى مقروءاً،
 * **ويُقال متى أُغلق** — ومن كتب فرُدّ بلا سببٍ يعيدها ثمّ يظنّ التطبيق معطوباً.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, fmtTime } from "@rahalgo/i18n";
import { Alert } from "./feedback";
import { Button, Textarea } from "./components";
import { IconChat, IconSend } from "./icons";
import { useLiveRefresh } from "./Notifications";

const m = getMessages(defaultLocale);
const C = m.chat;

type ApiFn = <T = unknown>(path: string, init?: RequestInit) => Promise<T>;

interface ChatMessage {
  id: string;
  body: string;
  role: "customer" | "driver";
  mine: boolean;
  created_at: string;
  read_at: string | null;
}

interface Thread {
  messages: ChatMessage[];
  peer_name: string;
  open: boolean;
  closes_at: string | null;
}

export function OrderChat({ api, orderId }: { api: ApiFn; orderId: string }) {
  const [thread, setThread] = useState<Thread | null>(null);
  const [draft, setDraft] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const endRef = useRef<HTMLDivElement | null>(null);

  const load = useCallback(async () => {
    try {
      const t = await api<Thread>(`/api/v1/orders/${orderId}/messages`);
      setThread(t);
    } catch {
      // **وقناةٌ لا تُفتح لا تُسقط الشاشة** — الطلبُ يُقرأ بلا حديثه.
      setThread(null);
    }
  }, [api, orderId]);

  useEffect(() => {
    void load();
  }, [load]);

  // **حيٌّ**: الرسالةُ تصل وصاحبُها ينظر — **وشاشةٌ لا تتحدّث تجعله يكتب
  // مرّتين** ظنّاً أنّ الأولى لم تصل.
  useLiveRefresh(["order"], load);

  // **وينزل إلى آخره** — الجديدُ في الأسفل، **ومن فتح فوجد قديماً ظنّ أنّ
  // شيئاً لم يصل.**
  useEffect(() => {
    endRef.current?.scrollIntoView({ block: "nearest" });
  }, [thread?.messages.length]);

  if (!thread) return null;

  async function send(e: React.FormEvent) {
    e.preventDefault();
    const body = draft.trim();
    if (!body || busy) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/orders/${orderId}/messages`, {
        method: "POST",
        body: JSON.stringify({ body }),
      });
      setDraft("");
      await load();
    } catch {
      setError(C.sendFailed);
    } finally {
      setBusy(false);
    }
  }

  return (
    <section className="surface p-4">
      <h3 className="mb-3 flex items-center gap-2 heading-card text-ink">
        <IconChat size={18} className="text-ink-muted" />
        {C.title.replace("{name}", thread.peer_name)}
      </h3>

      {/* **ولا رقمَ يُعرض** — الاسمُ وحدَه، وهو ما يحتاجه من يكتب. */}
      <div className="mb-3 max-h-72 space-y-2 overflow-y-auto">
        {thread.messages.length === 0 ? (
          <p className="py-6 text-center text-sm text-ink-muted">{C.empty}</p>
        ) : (
          thread.messages.map((x) => (
            <div key={x.id} className={`flex ${x.mine ? "justify-end" : "justify-start"}`}>
              <div
                className={`max-w-[78%] rounded-card px-3 py-2 text-sm ${
                  x.mine ? "bg-accent-tint text-ink" : "bg-field text-ink"
                }`}
              >
                <p className="whitespace-pre-wrap break-words">{x.body}</p>
                <p className="mt-1 text-2xs text-ink-muted">
                  {fmtTime(x.created_at)}
                  {/* **و«قُرئت» لمن أرسل وحدَه** — ولا معنى لها عند المتلقّي. */}
                  {x.mine && x.read_at ? ` · ${C.seen}` : ""}
                </p>
              </div>
            </div>
          ))
        )}
        <div ref={endRef} />
      </div>

      {error && <Alert>{error}</Alert>}

      {thread.open ? (
        <form onSubmit={send} className="flex items-end gap-2">
          <Textarea
            rows={2}
            value={draft}
            onChange={(e) => setDraft(e.target.value)}
            placeholder={C.placeholder}
            className="min-h-0 flex-1"
            maxLength={500}
          />
          <Button type="submit" disabled={busy || !draft.trim()} aria-label={C.send}>
            <IconSend size={18} />
          </Button>
        </form>
      ) : (
        /* **ويُقال إنّها أُغلقت** — لا يُخفى الحقلُ بلا كلمة. */
        <p className="rounded-control bg-field px-3 py-2 text-center text-xs text-ink-muted">
          {C.closed}
        </p>
      )}
    </section>
  );
}
