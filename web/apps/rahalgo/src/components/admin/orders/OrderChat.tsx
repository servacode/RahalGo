"use client";

/**
 * **محادثةُ الطلب قراءةً فقط** — مكوّنٌ واحدٌ تستعمله لوحةُ «ملف الطلب» ونافذةُ زرّ «الدردشة»
 * على بطاقة السجلّ (قرارُ المالك ٢٠٢٦-١١-٠٤). **والصلاحيةُ يحرسها المستدعي**
 * (`orders.communications.read`) — والخادمُ هو الحَكَم.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDateTime, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, ChatAttachmentView, Confirm, LoadingState, Modal, type ChatAttachment } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { roleLabelByCode } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const H = m.admin.ordersPage.history;
const PARTIES: Record<string, string> = H.parties;

interface ChatLine {
  id: string;
  body: string;
  role: string;
  sender: string;
  created_at: string;
  /** **سطرٌ موسوم** — لفظٌ مسيءٌ أو رقمُ تواصلٍ أُخفي عن الطرف الآخر. */
  flagged?: boolean;
  /** **ما أُخفي بنصّه الأصليّ** — للإدارة وحدَها (قرارُ المالك ٢٠٢٦-١١-٠٥). */
  flag_word?: string;
  /** **أُخفي منه رقمٌ أو واتساب** — حمايةُ الزبون من سحبه برّا المنصّة. */
  contact_leak?: boolean;
  /** صورةٌ أو رسالةٌ صوتيّة — تُعرض هي وشرحُها بدل `body` (نصٌّ احتياطيٌّ للتطبيقات القديمة). */
  attachment?: ChatAttachment;
  /** **صورةٌ محجوزةٌ** — فيها رقمٌ فلم تصل الطرفَ الآخر حتّى تُمرَّر. */
  held?: boolean;
  released_at?: string | null;
  /** **ما قُرئ من الصورة أو سُمع من التسجيل** — للإدارة وحدَها. */
  scan_text?: string;
}

/** **اسمُ المرسِل** — وحين يغيب يُقال دورُه بالعربيّ لا رمزُه (`driver`). */
function senderLabel(l: { sender: string; role: string }): string {
  return l.sender || PARTIES[l.role] || roleLabelByCode(l.role);
}

/** **كلّ كم تُقرأ دردشةُ طلبٍ ماشٍ** (م ث) — الرسالةُ الجديدةُ تظهر بلا سكّرٍ وفتح. */
const LIVE_POLL_MS = 5000;

/**
 * `live` **للطلب الماشي** (المالك ٢٠٢٦-١١-١٨): تُعاد القراءةُ كلَّ خمس ثوانٍ والصفحةُ ظاهرة، **وخطأُ
 * قراءةٍ عابرٍ لا يمحو ما ظهر** — يبقى آخرُ ما قُرئ حتّى تنجح القراءةُ التالية.
 */
export function OrderChatView({ orderId, live = false }: { orderId: string; live?: boolean }) {
  const [chat, setChat] = useState<{ lines: ChatLine[] } | null | "failed">(null);
  const canRelease = useCanCall()("POST", "/order-messages/{id}/release");
  const [reloadKey, setReloadKey] = useState(0);
  const [releaseId, setReleaseId] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [releaseErr, setReleaseErr] = useState("");

  const release = async () => {
    if (!releaseId) return;
    setBusy(true);
    setReleaseErr("");
    try {
      await api(`/api/v1/admin/order-messages/${releaseId}/release`, { method: "POST" });
      setReleaseId(null);
      setReloadKey((k) => k + 1);
    } catch (err) {
      setReleaseErr(errorText(err));
      setReleaseId(null);
      setReloadKey((k) => k + 1);
    } finally {
      setBusy(false);
    }
  };

  useEffect(() => {
    let alive = true;
    setChat(null);
    const load = () =>
      api<{ lines: ChatLine[] }>(`/api/v1/admin/orders/${orderId}/chat`)
        .then((r) => alive && setChat(r))
        .catch(() => alive && setChat((prev) => (prev && prev !== "failed" ? prev : "failed")));
    void load();
    const timer = live
      ? setInterval(() => {
          if (typeof document === "undefined" || document.visibilityState === "visible") void load();
        }, LIVE_POLL_MS)
      : undefined;
    return () => {
      alive = false;
      if (timer) clearInterval(timer);
    };
  }, [orderId, live, reloadKey]);

  return (
    <>
      {chat === null && <LoadingState variant="text" />}
      {chat === "failed" && <Alert tone="warning">{m.errors.offline}</Alert>}
      {chat && chat !== "failed" && chat.lines.length === 0 && (
        <p className="text-sm text-ink-muted">{H.chatEmpty}</p>
      )}
      {releaseErr && <Alert tone="warning">{releaseErr}</Alert>}
      <Confirm
        open={releaseId !== null}
        title={H.releaseTitle}
        body={H.releaseBody}
        confirmLabel={H.releaseConfirm}
        tone="primary"
        busy={busy}
        onConfirm={() => void release()}
        onCancel={() => setReleaseId(null)}
      />
      {chat && chat !== "failed" && chat.lines.length > 0 && (
        <ul className="space-y-1.5">
          {chat.lines.map((l) => (
            <li key={l.id} className="flex items-start gap-2 text-sm">
              <Badge variant={l.role === "driver" ? "primary" : "neutral"}>{senderLabel(l)}</Badge>
              <span className="min-w-0 flex-1">
                {l.attachment ? (
                  <span className="flex flex-col items-start gap-1">
                    <ChatAttachmentView attachment={l.attachment} native />
                    {l.attachment.caption && (
                      <span className={l.flagged ? "text-danger" : ""}>{l.attachment.caption}</span>
                    )}
                  </span>
                ) : (
                  <span className={l.flagged ? "text-danger" : ""}>{l.body}</span>
                )}
                {/* **وما أُخفي يُقال للإدارة بنصّه** — الرقمُ أو اللفظ — ليُحسم البلاغ. */}
                {l.flagged && l.flag_word && (
                  <span className="ms-2 text-xs text-danger">({l.flag_word})</span>
                )}
                {l.attachment && l.scan_text && (
                  <span className="mt-1 block text-sm text-ink-muted">
                    <span className="font-bold">
                      {l.attachment.kind === "audio" ? H.audioText : H.photoText}
                    </span>{" "}
                    {l.scan_text}
                  </span>
                )}
                {l.held && !l.released_at && (
                  <span className="mt-1 flex flex-wrap items-center gap-2">
                    <Badge variant="danger">{H.photoHeld}</Badge>
                    {canRelease && (
                      <Button size="sm" variant="secondary" onClick={() => setReleaseId(l.id)}>
                        {H.releasePhoto}
                      </Button>
                    )}
                  </span>
                )}
                {l.held && l.released_at && (
                  <span className="ms-2 text-sm text-ink-muted">({H.photoReleased})</span>
                )}
                {l.contact_leak && (
                  <span className="ms-2 inline-block">
                    <Badge variant="danger">{H.contactLeak}</Badge>
                  </span>
                )}
              </span>
              <span dir="ltr" className="shrink-0 text-2xs text-ink-muted">
                {fmtDateTime(l.created_at)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </>
  );
}

/**
 * **نافذةُ الدردشة** — من بطاقة السجلّ، **ومن بطاقة الطلب الماشي** (`live`: تتحدّث لحالها).
 */
export function OrderChatModal({
  orderId,
  orderNumber,
  onClose,
  live = false,
}: {
  orderId: string;
  orderNumber: number;
  onClose: () => void;
  live?: boolean;
}) {
  return (
    <Modal open onClose={onClose} size="lg" title={`${H.chat} · #${orderNumber}`}>
      <OrderChatView orderId={orderId} live={live} />
    </Modal>
  );
}
