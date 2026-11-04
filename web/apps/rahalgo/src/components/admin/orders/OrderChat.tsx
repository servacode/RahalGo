"use client";

/**
 * **محادثةُ الطلب قراءةً فقط** — مكوّنٌ واحدٌ تستعمله لوحةُ «ملف الطلب» ونافذةُ زرّ «الدردشة»
 * على بطاقة السجلّ (قرارُ المالك ٢٠٢٦-١١-٠٤). **والصلاحيةُ يحرسها المستدعي**
 * (`orders.communications.read`) — والخادمُ هو الحَكَم.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDateTime } from "@rahalgo/i18n";
import { Alert, Badge, LoadingState, Modal } from "@rahalgo/ui";
import { api } from "@/lib/api";
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
}

/** **اسمُ المرسِل** — وحين يغيب يُقال دورُه بالعربيّ لا رمزُه (`driver`). */
function senderLabel(l: { sender: string; role: string }): string {
  return l.sender || PARTIES[l.role] || roleLabelByCode(l.role);
}

export function OrderChatView({ orderId }: { orderId: string }) {
  const [chat, setChat] = useState<{ lines: ChatLine[] } | null | "failed">(null);
  useEffect(() => {
    let alive = true;
    setChat(null);
    api<{ lines: ChatLine[] }>(`/api/v1/admin/orders/${orderId}/chat`)
      .then((r) => alive && setChat(r))
      .catch(() => alive && setChat("failed"));
    return () => {
      alive = false;
    };
  }, [orderId]);

  return (
    <>
      {chat === null && <LoadingState variant="text" />}
      {chat === "failed" && <Alert tone="warning">{m.errors.offline}</Alert>}
      {chat && chat !== "failed" && chat.lines.length === 0 && (
        <p className="text-sm text-ink-muted">{H.chatEmpty}</p>
      )}
      {chat && chat !== "failed" && chat.lines.length > 0 && (
        <ul className="space-y-1.5">
          {chat.lines.map((l) => (
            <li key={l.id} className="flex items-start gap-2 text-sm">
              <Badge variant={l.role === "driver" ? "primary" : "neutral"}>{senderLabel(l)}</Badge>
              <span className="min-w-0 flex-1">{l.body}</span>
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

/** **نافذةُ الدردشة من بطاقة السجلّ** — تفتح المحادثةَ مباشرةً بلا لوحةِ الملف. */
export function OrderChatModal({
  orderId,
  orderNumber,
  onClose,
}: {
  orderId: string;
  orderNumber: number;
  onClose: () => void;
}) {
  return (
    <Modal open onClose={onClose} size="lg" title={`${H.chat} · #${orderNumber}`}>
      <OrderChatView orderId={orderId} />
    </Modal>
  );
}
