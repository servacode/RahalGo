"use client";

/**
 * **دردشاتي السابقة — سجلٌّ يُحتجّ به.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «يجب أن يكون هناك دردشاتي السابقة… مشان إثبات».)
 *
 * # ولماذا المنتهيةُ تبقى
 *
 * **الحديثُ حجّةٌ عند الخلاف** — ومن اتُّفق معه على سعرٍ ثمّ أُنكر يرجع إليه.
 * **ومحادثةٌ تختفي بانتهاء الطلب تمحو الدليلَ في اللحظة التي يُحتاج فيها**:
 * لا يُختلَف أثناء الطلب، **إنّما بعده.**
 *
 * # وتُقرأ ولا تُكتب
 *
 * **القناةُ تُقفل للكتابة بانتهاء الطلب** — و`OrderChat` يعرف ذلك من الخادم
 * ويُخفي الحقل، **فلا شرطَ يُكتب هنا** ولا اثنان يفترقان.
 *
 * # ومكوّنٌ واحدٌ للوحتين
 *
 * **السائقُ والزبونُ يريان السجلَّ نفسَه** — كلٌّ محادثاتِه، **ونسختان
 * تفترقان يوماً.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import { Badge } from "./components";
import { PageContainer, PageHeader, EmptyState, LoadingState } from "./layout";
import { IconChat } from "./icons";
import { OrderChat } from "./OrderChat";

const m = getMessages(defaultLocale);
const C = m.chat;

type ApiFn = <T = unknown>(path: string, init?: RequestInit) => Promise<T>;

interface Thread {
  order_id: string;
  number: number;
  peer: string;
  open: boolean;
  unread: number;
  last_body: string;
  last_at: string | null;
}

export function ChatArchive({
  api,
  /**
   * **عارياً — بلا حاويةٍ ولا عنوان.**
   *
   * **حين يسكن تبويباً** لصفحةٍ لها عنوانُها: **عنوانان فوق بعضهما يُقرآن
   * عطباً**، وحاويةٌ داخل حاويةٍ تضاعف الحشوة.
   */
  bare = false,
}: {
  api: ApiFn;
  bare?: boolean;
}) {
  const [rows, setRows] = useState<Thread[] | null>(null);
  const [picked, setPicked] = useState<string>("");

  const load = useCallback(() => {
    api<{ threads: Thread[] }>("/api/v1/my/chats")
      // **والمنتهيةُ وحدَها هنا** — (قرارُ المالك ٢٠٢٦-٠٨-١٠: «الدردشةُ
      // عندما تُغلق فقط تظهر بالدردشات السابقة، وليس عندما تكون مفتوحة»).
      //
      // **وحديثٌ يجري في «السابقة» تناقضٌ في الاسم**: يُفتح من موضعين
      // فيُقرأ مرّتين، **وشارةُ ما لم يُقرأ تنطفئ في أحدهما** فيظنّ صاحبُها
      // أنّه ردّ وهو لم يفعل.
      .then((r) => setRows((r?.threads ?? []).filter((t) => !t.open)))
      // @empty-ok **قائمةٌ فارغةٌ حالٌ لا خطأ** — من لم يُحادث أحداً بعد.
      .catch(() => setRows([]));
  }, [api]);

  useEffect(load, [load]);

  if (rows === null) return <LoadingState />;

  const body = (
    <>
      {rows.length === 0 ? (
        <EmptyState icon={IconChat} title={C.archiveEmpty} />
      ) : (
        <ul className="space-y-2">
          {rows.map((t) => (
            <li key={t.order_id} className="surface p-3">
              <button
                type="button"
                onClick={() => setPicked(picked === t.order_id ? "" : t.order_id)}
                className="flex w-full items-center gap-2 text-start"
              >
                <span className="font-bold">#{fmtNum(t.number)}</span>
                <span className="min-w-0 flex-1 truncate text-sm text-ink-muted">
                  {t.peer} — {t.last_body}
                </span>
                {t.unread > 0 && <Badge variant="danger">{fmtNum(t.unread)}</Badge>}
                {t.last_at && (
                  <span className="shrink-0 text-2xs text-ink-muted">
                    {fmtDateTime(t.last_at)}
                  </span>
                )}
              </button>
              {picked === t.order_id && (
                <div className="mt-3">
                  <OrderChat api={api} orderId={t.order_id} />
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </>
  );

  if (bare) return body;
  return (
    <PageContainer>
      <PageHeader icon={IconChat} title={C.archiveTitle} subtitle={C.archiveHint} />
      {body}
    </PageContainer>
  );
}
