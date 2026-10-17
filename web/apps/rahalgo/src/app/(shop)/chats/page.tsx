"use client";

/**
 * **دردشاتي — ما دار مع السائق في طلباتٍ أُغلقت** (مرآةُ «دردشاتي» في أندرويد).
 *
 * **والجاريةُ في الفقّاعة العائمة** فوق كلّ شاشة — **ولا تُفتح من موضعين.**
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { ChatArchive, LoadingState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { NeedLogin } from "@/components/customer/NeedLogin";

const m = getMessages(defaultLocale);
const C = m.shop.chats;

export default function ChatsPage() {
  const { user, loading } = useAuth();
  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;
  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      <div>
        <h1 className="heading-page">{C.title}</h1>
        <p className="text-sm text-ink-muted">{C.hint}</p>
      </div>
      <ChatArchive api={api} bare />
    </div>
  );
}
