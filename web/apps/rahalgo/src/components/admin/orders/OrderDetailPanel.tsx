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
  fmtNum,
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
  IconUser,
  IconStore,
  IconMoto,
  IconWallet,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { roleLabelByCode } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const H = m.admin.ordersPage.history;
const OFFICE_STATUS: Record<string, string> = m.admin.ordersPage.board.status;
const STATUS_LABELS: Record<string, string> = m.orders.status;
const TX_KINDS: Record<string, string> = m.shared.txKinds;
const PARTIES: Record<string, string> = H.parties;
/** **ترتيبُ الأطراف في القيود** — الزبونُ يدفع، ثمّ من يقبض، ثمّ المنصّة، ثمّ ما سواها. */
const PARTY_ORDER = ["customer", "merchant", "driver", "sales", "platform", "other"];

/** **لونُ الحالة** — أخضرُ للمسلَّم، أحمرُ للفاشل، برتقاليٌّ للملغى والمرفوض (تدقيقُ السجلّ ٢٠٢٦-١٠-٢٧). */
const STATUS_VARIANT: Record<string, "success" | "danger" | "warning" | "violet" | "accent"> = {
  delivered: "success",
  failed: "danger",
  cancelled: "warning",
  rejected: "warning",
  refunded: "violet",
};

/** **اسمُ المرسِل** — وحين يغيب يُقال دورُه بالعربيّ لا رمزُه (`driver`). */
function senderLabel(l: { sender: string; role: string }): string {
  return l.sender || PARTIES[l.role] || roleLabelByCode(l.role);
}

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
  /**
   * **موقعُ الزبون المباشر إن شاركه وهو حديث** (قرارُ المالك ٢٠٢٦-١٠-١٦) — يصل لمن يرى
   * العنوانَ وحدَه (`admin_orders_handlers.go`).
   */
  customer_live?: {
    pin_gap_m: number;
    at: string;
    accuracy_m: number | null;
    precise: boolean;
  } | null;
}

/**
 * **سطرُ موقع الزبون المباشر** — «يبعد X م عن العنوان · قبل Y ث»، **والتقريبيُّ يُقال عنه**:
 * يُرى ولا يتحوّل إليه طريقُ الكابتن.
 */
function liveLine(live: NonNullable<Detail["customer_live"]>, nowMs: number): string {
  const ageSec = Math.max(0, Math.round((nowMs - new Date(live.at).getTime()) / 1000));
  const age =
    ageSec < 60
      ? H.liveAgeSec.replace("{n}", fmtNum(ageSec))
      : H.liveAgeMin.replace("{n}", fmtNum(Math.floor(ageSec / 60)));
  const gap = live.pin_gap_m < 0 ? "—" : fmtNum(Math.round(live.pin_gap_m));
  const line = H.liveLine.replace("{m}", gap).replace("{age}", age);
  return live.precise ? line : line + H.liveApprox;
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
          {/* **كلُّ سطرٍ بعنوانه وأيقونته** (تدقيقُ السجلّ ٢٠٢٦-١٠-٢٧) — كانت أسماءٌ متتاليةٌ بلا عنوان
              فلا يُعرف من المتجرُ ومن السائق، ومبلغٌ لا يقول ما هو. */}
          <div className="space-y-2 text-sm">
            <p className="flex flex-wrap items-center gap-2">
              <Badge variant={STATUS_VARIANT[order.status] ?? "accent"}>
                {OFFICE_STATUS[order.status] ?? order.status}
              </Badge>
              <span className="text-ink-muted" dir="ltr">
                {fmtDateTime(order.created_at)}
              </span>
            </p>
            <InfoRow icon={<IconUser size={15} />} label={H.infoCustomer}>
              {order.customer_name || "—"}
              <span dir="ltr" className="ms-2 text-xs text-ink-muted">
                {order.customer_phone || order.customer_phone_masked || ""}
              </span>
            </InfoRow>
            {order.merchant_name && (
              <InfoRow icon={<IconStore size={15} />} label={H.infoStore}>
                {order.merchant_name}
              </InfoRow>
            )}
            {order.driver_name && (
              <InfoRow icon={<IconMoto size={15} />} label={H.infoDriver}>
                {order.driver_name}
              </InfoRow>
            )}
            {order.customer_live && (
              <p className="font-bold text-success">{liveLine(order.customer_live, Date.now())}</p>
            )}
            <InfoRow icon={<IconWallet size={15} />} label={H.infoTotal}>
              <Money value={order.total} className="font-bold" />
            </InfoRow>
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
                        {senderLabel(l)}
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
              {/* **مجمّعةٌ بالطرف، و«إلو/عليه» بدل إشارة الناقص** (تدقيقُ السجلّ ٢٠٢٦-١٠-٢٧) — كانت أسماءَ
                  حساباتٍ داخليّةٍ ومبالغَ بناقصٍ بلا شرح. **والأرقامُ من الدفتر كما هي — عرضٌ لا حساب.** */}
              {money && money !== "failed" && money.lines.length > 0 && (
                <div className="space-y-3 text-sm">
                  <p className="text-2xs text-ink-muted">{H.moneyHint}</p>
                  {PARTY_ORDER.map((party) => {
                    const lines = money.lines.filter(
                      (l) => (PARTY_ORDER.includes(l.party) ? l.party : "other") === party,
                    );
                    if (lines.length === 0) return null;
                    return (
                      <div key={party} className="rounded-control border border-line-soft p-2.5">
                        <p className="mb-1.5 text-xs font-bold">{PARTIES[party] ?? party}</p>
                        <ul className="space-y-1.5">
                          {lines.map((l, i) => (
                            <li key={i} className="flex items-start justify-between gap-2">
                              <span className="min-w-0">
                                {TX_KINDS[l.kind] ?? l.kind}
                                {/* **واسمُ صاحب المحفظة للأطراف** — والحسابُ الداخليُّ لا يُسمّى. */}
                                {party !== "platform" && party !== "other" && l.name && (
                                  <span className="text-ink-muted"> — {l.name}</span>
                                )}
                                {l.note && <span className="block text-2xs text-ink-muted">{l.note}</span>}
                              </span>
                              <span
                                className={`flex shrink-0 items-center gap-1.5 ${
                                  l.amount < 0 ? "text-danger" : "text-success"
                                }`}
                              >
                                <span className="text-2xs">{l.amount < 0 ? H.debit : H.credit}</span>
                                <Money value={Math.abs(l.amount)} small />
                              </span>
                            </li>
                          ))}
                        </ul>
                      </div>
                    );
                  })}
                </div>
              )}
            </section>
          )}
        </div>
      )}
    </Drawer>
  );
}

/** **سطرٌ بعنوانه وأيقونته** — العنوانُ يميناً والقيمةُ يساراً. */
function InfoRow({ icon, label, children }: { icon: React.ReactNode; label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3">
      <span className="inline-flex shrink-0 items-center gap-1.5 text-ink-muted">
        {icon}
        {label}
      </span>
      <span className="min-w-0 text-end">{children}</span>
    </div>
  );
}
