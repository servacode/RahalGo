"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/complaints/[id] — شكوى بعينها: حالُها وخيطُ ردودها** (٢٠٢٦-١٠-١٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `TicketThreadScreen` في تطبيق الزبون — `MineScreens.kt`.)
 *
 * **الموضوعُ والحالُ ورقمُ الطلب وكلمةُ المنصّة عند الإغلاق**، ثمّ الردودُ
 * بترتيبها — ردُّه في جهةٍ وردُّ المنصّة في الأخرى. **ويردّ ما دامت غيرَ
 * محلولة** (`ticketCanReply` في التطبيق: كلُّ حالٍ عدا `resolved`)، **والمحرّكُ
 * يحرس ذلك** (`ticket_resolved`). **وإشعارُ «ردّت المنصّة» يفتح هنا.**
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtDateTime, fmtRef } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  LoadingState,
  ReloadState,
  Textarea,
  IconPrev,
  useLiveRefresh,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { fill } from "@/lib/customer/texts";
import { NeedLogin } from "@/components/customer/NeedLogin";

const m = getMessages(defaultLocale);
const Q = m.shop.complaint;
const ST = m.admin.tickets.status as Record<string, string>;

interface Reply {
  id: number;
  body: string;
  mine: boolean;
  created_at: string;
}
interface TicketDetail {
  id: string;
  number: number;
  subject: string;
  status: string;
  resolution: string;
  order_number: number | null;
  created_at: string;
  replies: Reply[];
}

function tone(status: string) {
  if (status === "resolved") return "success" as const;
  if (status === "in_progress") return "info" as const;
  return "warning" as const;
}

export default function ComplaintPage() {
  const { id } = useParams<{ id: string }>();
  const { user, loading } = useAuth();
  const [t, setT] = useState<TicketDetail | null>(null);
  const [error, setError] = useState("");
  const [draft, setDraft] = useState("");
  const [sending, setSending] = useState(false);
  const [sendError, setSendError] = useState("");

  const load = useCallback(() => {
    if (!user || !id) return;
    return api<TicketDetail>(`/api/v1/my/tickets/${encodeURIComponent(id)}`)
      .then((d) => {
        setT(d);
        setError("");
      })
      .catch((err) => setError(errorText(err)));
  }, [user, id]);

  useEffect(() => {
    void load();
  }, [load]);
  // **وردُّ المنصّة يصل حيّاً** — إشعارُ التذكرة يوقظ القراءة.
  useLiveRefresh(["ticket"], load);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;

  const back = (
    <Link href="/complaints" className="flex w-fit items-center gap-1 text-sm text-accent-text">
      <IconPrev size={16} />
      {Q.back}
    </Link>
  );

  if (!t) {
    return (
      <div className="flex flex-col gap-3">
        {back}
        {error ? <ReloadState label={error} onRetry={() => void load()} /> : <LoadingState variant="list" rows={3} />}
      </div>
    );
  }

  const canReply = t.status !== "resolved";

  const send = async (e: React.FormEvent) => {
    e.preventDefault();
    const body = draft.trim();
    if (!body) return;
    setSending(true);
    setSendError("");
    try {
      const d = await api<TicketDetail>(`/api/v1/my/tickets/${encodeURIComponent(t.id)}/replies`, {
        method: "POST",
        body: JSON.stringify({ body }),
      });
      setT(d);
      setDraft("");
    } catch (err) {
      setSendError(errorText(err));
    } finally {
      setSending(false);
    }
  };

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-3">
      {back}
      <section className="surface flex flex-col gap-3 p-4">
        <div className="flex items-start justify-between gap-2">
          <div className="min-w-0">
            <h1 className="heading-page">{fill(Q.detailTitle, { n: `#${t.number}` })}</h1>
            <p className="font-bold">{t.subject}</p>
          </div>
          <Badge variant={tone(t.status)}>{ST[t.status] ?? t.status}</Badge>
        </div>
        <dl className="flex flex-col gap-1 text-sm">
          <div className="flex justify-between gap-2">
            <dt className="text-ink-muted">{Q.opened}</dt>
            <dd>{fmtDateTime(t.created_at)}</dd>
          </div>
          {t.order_number != null && (
            <div className="flex justify-between gap-2">
              <dt className="text-ink-muted">{Q.order}</dt>
              <dd className="tabular-nums" dir="ltr">
                #{fmtRef(t.order_number)}
              </dd>
            </div>
          )}
        </dl>
        {t.resolution && (
          <Alert tone="info">
            <span className="font-bold">{Q.resolution}: </span>
            {t.resolution}
          </Alert>
        )}
      </section>

      <section className="surface flex flex-col gap-3 p-4" aria-label={Q.replies}>
        <h2 className="heading-card">{Q.replies}</h2>
        {t.replies.length === 0 ? (
          <p className="text-sm text-ink-muted">{Q.noReplies}</p>
        ) : (
          <ul className="flex flex-col gap-2">
            {t.replies.map((r) => (
              <li
                key={r.id}
                className={`flex max-w-[85%] flex-col gap-1 rounded-card p-3 text-sm ${
                  r.mine ? "self-start bg-accent-tint" : "surface-inset self-end"
                }`}
              >
                <span className="text-xs font-bold text-ink-muted">{r.mine ? Q.you : Q.platform}</span>
                <span className="whitespace-pre-wrap">{r.body}</span>
                <span className="text-xs text-ink-muted">{fmtDateTime(r.created_at)}</span>
              </li>
            ))}
          </ul>
        )}
        {canReply ? (
          <form onSubmit={send} className="flex flex-col gap-2">
            <Textarea
              id="ticket-reply"
              label={Q.replyLabel}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              maxLength={1000}
              autoGrow
            />
            {sendError && <Alert>{sendError}</Alert>}
            <Button type="submit" disabled={sending || !draft.trim()} className="w-fit">
              {Q.replySend}
            </Button>
          </form>
        ) : (
          <p className="text-xs text-ink-muted">{Q.replyClosed}</p>
        )}
      </section>
    </div>
  );
}
