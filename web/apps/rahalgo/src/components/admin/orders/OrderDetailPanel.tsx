"use client";

/**
 * **تفاصيلُ الطلب في لوحةٍ جانبيّة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (سجلُّ الطلبات، البند ٤).
 *
 * **«مين ألغى ولّا مين أخّر؟»** — كان المحرّكُ يردّ المسارَ كاملاً بمن فعل كلَّ
 * خطوةٍ ومتى (`GET /admin/orders/{id}` ← `timeline`) **ولا شاشةَ تقرؤه.**
 *
 *   - **المسارُ بأوقاته ومن فعل كلَّ خطوة.**
 *   - **والمحادثةُ قراءةً فقط** — لمن يملك `orders.communications.read`.
 *   - **والقيودُ الماليّةُ من الدفتر** — لمن يملك `finance.read`.
 *   - **ورابطٌ خاصٌّ `?order=`** يُرسَل لزميلٍ فيفتح عليه.
 *
 * **وكلُّ قسمٍ يُحمَّل وحدَه ويفشل وحدَه** — المحادثةُ الساقطةُ لا تُخفي المسار.
 */

import { useEffect, useState } from "react";
import {
  getMessages,
  defaultLocale,
  fmtDateTime,
  errorText,
} from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Drawer,
  LoadingState,
  Money,
  Timeline,
  type TimelineNode,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const H = m.admin.ordersPage.history;
const OFFICE_STATUS: Record<string, string> = m.admin.ordersPage.board.status;
const STATUS_LABELS: Record<string, string> = m.orders.status;
const TX_KINDS: Record<string, string> = m.shared.txKinds;

interface Step {
  status: string;
  at: string;
  actor?: string | null;
  note?: string;
}

interface Detail {
  id: string;
  number: number;
  status: string;
  customer_name: string;
  customer_phone?: string;
  customer_phone_masked?: string;
  merchant_name: string;
  driver_name: string | null;
  total: number;
  created_at: string;
  timeline: Step[];
}

interface ChatLine {
  id: string;
  body: string;
  role: string;
  sender: string;
  created_at: string;
}

interface MoneyBreakdown {
  lines: { party: string; name: string; kind: string; amount: number; note: string }[];
}

type Load<T> = T | null | "failed";

function useLoad<T>(path: string | null): Load<T> {
  const [v, setV] = useState<Load<T>>(null);
  useEffect(() => {
    if (!path) return;
    let alive = true;
    setV(null);
    api<T>(path)
      .then((r) => alive && setV(r))
      .catch(() => alive && setV("failed"));
    return () => {
      alive = false;
    };
  }, [path]);
  return v;
}

export function OrderDetailPanel({
  orderId,
  onClose,
}: {
  orderId: string;
  onClose: () => void;
}) {
  const { can } = useAuth();
  const order = useLoad<Detail>(`/api/v1/admin/orders/${orderId}`);
  const chat = useLoad<{ lines: ChatLine[] }>(
    can("orders.communications.read") ? `/api/v1/admin/orders/${orderId}/chat` : null,
  );
  const money = useLoad<MoneyBreakdown>(
    can("finance.read") ? `/api/v1/admin/orders/${orderId}/breakdown` : null,
  );
  const [copied, setCopied] = useState(false);
  const [copyErr, setCopyErr] = useState("");

  const title =
    order && order !== "failed" ? H.panelTitle.replace("{n}", `#${order.number}`) : H.openDetails;

  const copy = () => {
    void navigator.clipboard
      .writeText(window.location.href)
      .then(() => {
        setCopied(true);
        setCopyErr("");
      })
      .catch((err) => setCopyErr(errorText(err)));
  };

  return (
    <Drawer
      open
      onClose={onClose}
      title={title}
      actions={
        <Button variant="secondary" onClick={copy}>
          {copied ? H.copied : H.copyLink}
        </Button>
      }
    >
      {copyErr && <Alert tone="warning" className="mb-3">{copyErr}</Alert>}
      {order === null && <LoadingState variant="text" />}
      {order === "failed" && <Alert tone="warning">{H.panelFailed}</Alert>}
      {order && order !== "failed" && (
        <div className="space-y-5">
          <div className="space-y-1 text-sm">
            <p className="flex flex-wrap items-center gap-2">
              <Badge variant="neutral">{OFFICE_STATUS[order.status] ?? order.status}</Badge>
              <span className="text-ink-muted" dir="ltr">
                {fmtDateTime(order.created_at)}
              </span>
            </p>
            <p>
              {order.customer_name || "—"}
              {" · "}
              <span dir="ltr" className="text-ink-muted">
                {order.customer_phone || order.customer_phone_masked || "—"}
              </span>
            </p>
            {order.merchant_name && <p>{order.merchant_name}</p>}
            {order.driver_name && <p>{order.driver_name}</p>}
            <p className="font-bold">
              <Money value={order.total} />
            </p>
          </div>

          <section>
            <h3 className="mb-2 text-sm font-bold">{H.timeline}</h3>
            <Timeline
              nodes={order.timeline.map(
                (s, i): TimelineNode => ({
                  id: `${i}`,
                  state: "done",
                  title: OFFICE_STATUS[s.status] ?? STATUS_LABELS[s.status] ?? s.status,
                  detail: (
                    <span className="text-xs text-ink-muted">
                      {s.actor ? H.by.replace("{name}", s.actor) : H.bySystem}
                      {s.note ? ` — ${s.note}` : ""}
                    </span>
                  ),
                  trailing: (
                    <span dir="ltr" className="text-xs">
                      {fmtDateTime(s.at)}
                    </span>
                  ),
                }),
              )}
            />
          </section>

          {can("orders.communications.read") && (
            <section>
              <h3 className="mb-2 text-sm font-bold">{H.chat}</h3>
              {chat === null && <LoadingState variant="text" />}
              {chat === "failed" && <Alert tone="warning">{m.errors.offline}</Alert>}
              {chat && chat !== "failed" && chat.lines.length === 0 && (
                <p className="text-sm text-ink-muted">{H.chatEmpty}</p>
              )}
              {chat && chat !== "failed" && chat.lines.length > 0 && (
                <ul className="space-y-1.5">
                  {chat.lines.map((l) => (
                    <li key={l.id} className="flex items-start gap-2 text-sm">
                      <Badge variant={l.role === "driver" ? "primary" : "neutral"}>
                        {l.sender || l.role}
                      </Badge>
                      <span className="min-w-0 flex-1">{l.body}</span>
                      <span dir="ltr" className="shrink-0 text-2xs text-ink-muted">
                        {fmtDateTime(l.created_at)}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          )}

          {can("finance.read") && (
            <section>
              <h3 className="mb-2 text-sm font-bold">{H.money}</h3>
              {money === null && <LoadingState variant="text" />}
              {money === "failed" && <Alert tone="warning">{m.errors.offline}</Alert>}
              {money && money !== "failed" && money.lines.length === 0 && (
                <p className="text-sm text-ink-muted">{H.moneyEmpty}</p>
              )}
              {money && money !== "failed" && money.lines.length > 0 && (
                <ul className="space-y-1.5 text-sm">
                  {money.lines.map((l, i) => (
                    <li key={i} className="flex items-center justify-between gap-2">
                      <span className="min-w-0">
                        {TX_KINDS[l.kind] ?? l.kind}
                        <span className="text-ink-muted"> — {l.name}</span>
                      </span>
                      <Money value={l.amount} />
                    </li>
                  ))}
                </ul>
              )}
            </section>
          )}
        </div>
      )}
    </Drawer>
  );
}
