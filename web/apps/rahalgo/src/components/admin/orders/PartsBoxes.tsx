"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صناديقُ المتاجر في طلبٍ من أكثر من متجر** (قرارُ المالك ٢٠٢٦-١١-٠٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * طلبٌ واحدٌ وسائقٌ واحد، **وصندوقٌ لكلّ متجر**: اسمٌ واتّصال · العنوانُ والخريطة · أصنافُه ·
 * حالُه · علاماتُ «انبعتله / جاهز / انستلم» · السببُ ومن الغلطان. **وأزرارُ كلِّ قسمٍ لحاله**
 * على مسارات `/parts/{متجر}/…` — قبولٌ بوقت تحضير، رفضٌ بسبب، شيلٌ بسببٍ وغلط، تحويلٌ لمتجرٍ
 * تاني، وإرسالُ واتساب لهالمتجر وحدَه.
 *
 * **طلبُ المتجر الواحد لا يمرّ من هنا** (`parts` غائب) فيبقى صندوقُه `StoreSection` كما كان.
 * **والخادمُ هو الحَكَم** في كلّ فعل؛ والشاشةُ تعرض ما ردّه.
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { Badge, Button, FormActions, IconStore, Input, Select } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Box, MapLink, PartyLine, type OrderPart } from "./CardSections";
import { toWhatsAppWeb } from "./BoardWidgets";
import { TransferPanel } from "./TransferPanel";

const m = getMessages(defaultLocale);
const P = m.admin.ordersPage.parts;
const OP = m.admin.ordersPage;

/** **لونُ حالة القسم** — دالةٌ لا خريطة (حارسُ `statusColor`: القاموسُ الواحد للحالات). */
function statusTone(s: OrderPart["status"]): "warning" | "accent" | "danger" | "success" {
  if (s === "pending") return "warning";
  if (s === "accepted") return "accent";
  if (s === "picked_up") return "success";
  return "danger";
}

/** رسالةُ المتجر كما يبنيها الخادم لقسمٍ واحد. */
type PartMessage = { text: string; phone: string; wa_link: string };

/** **ما يخصّ الطلبَ** — الحالُ والإغلاقُ يحكمان أيَّ الأزرار تُعرض. */
export interface PartsOrder {
  id: string;
  status: string;
  closed_at?: string | null;
  parts?: OrderPart[];
}

type Ask = "" | "accept" | "reject" | "remove";

function PartBox({
  order,
  part,
  live,
  compact,
  canAct,
  onChanged,
}: {
  order: PartsOrder;
  part: OrderPart;
  live: boolean;
  /** **سجلُّ الطلب**: النتيجةُ وحدَها بلا أزرار. */
  compact: boolean;
  canAct: boolean;
  onChanged: () => void;
}) {
  const [ask, setAsk] = useState<Ask>("");
  const [reason, setReason] = useState("");
  const [fault, setFault] = useState("");
  const [prep, setPrep] = useState("");
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const [transferring, setTransferring] = useState(false);

  const open = !order.closed_at;
  const collected = !!part.collected || part.status === "picked_up";
  const dead = part.status === "rejected" || part.status === "removed";
  const orderPending = order.status === "pending";
  const base = `/api/v1/admin/orders/${order.id}/parts/${part.merchant_id}`;

  const canAnswer = canAct && open && orderPending && part.status === "pending";
  const canRemove =
    canAct && open && !collected && (part.status === "accepted" || (part.status === "pending" && !orderPending));
  const canTransfer = canAct && open && !collected && !dead;
  const canSend =
    canAct && open && !orderPending && part.status === "accepted" && !part.sent_at && !collected;

  async function post(action: string, path: string, body: Record<string, unknown>) {
    setBusy(action);
    setErr("");
    try {
      await api(`${base}/${path}`, { method: "POST", body: JSON.stringify(body) });
      setAsk("");
      setReason("");
      setFault("");
      setPrep("");
      onChanged();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy("");
    }
  }

  function submitAsk() {
    if (ask === "accept") {
      const n = Number(prep);
      void post("accept", "accept", prep.trim() !== "" && Number.isFinite(n) && n >= 0 ? { prep_minutes: Math.round(n) } : {});
      return;
    }
    if (!reason.trim()) {
      setErr(P.reasonNeeded);
      return;
    }
    if (ask === "reject") {
      void post("reject", "reject", { reason: reason.trim() });
      return;
    }
    if (!fault) {
      setErr(P.faultNeeded);
      return;
    }
    void post("remove", "remove", { reason: reason.trim(), fault });
  }

  /** **واتساب هالمتجر وحدَه** — نافذةٌ تُفتح بالضغطة ثمّ يُوسَم القسمُ مرسَلاً (كزرّ الطلب كلِّه). */
  async function sendWhatsApp() {
    setErr("");
    const win = window.open("", "rahalgo-whatsapp");
    setBusy("wa");
    try {
      const msg = await api<PartMessage>(`/api/v1/admin/orders/${order.id}/message?merchant_id=${part.merchant_id}`);
      if (!msg.wa_link) {
        win?.close();
        setErr(OP.noWhatsApp);
        return;
      }
      if (win) win.location.href = toWhatsAppWeb(msg.wa_link);
      await api(`/api/v1/admin/orders/${order.id}/whatsapp`, {
        method: "POST",
        body: JSON.stringify({ channel: "whatsapp", merchant_id: part.merchant_id }),
      });
      if (!win) setErr(OP.popupBlocked);
      onChanged();
    } catch (e) {
      win?.close();
      setErr(errorText(e));
    } finally {
      setBusy("");
    }
  }

  async function sendManual() {
    setBusy("manual");
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${order.id}/whatsapp`, {
        method: "POST",
        body: JSON.stringify({ channel: "manual", merchant_id: part.merchant_id }),
      });
      onChanged();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy("");
    }
  }

  const items = part.items ?? [];
  const title = P.box.replace("{seq}", fmtNum(part.seq));
  const roleName = part.answered_role ? P.roles[part.answered_role] : "";

  return (
    <Box look="store" icon={<IconStore size={13} />} title={title}>
      <PartyLine name={part.name} phone={part.phone} callLabel={OP.callStoreShort} live={live} extra={<Badge variant={statusTone(part.status)}>{P.status[part.status]}</Badge>} />
      {(part.address || part.lat != null) && (
        <span className="flex items-start justify-between gap-2">
          <span className="min-w-0 flex-1 whitespace-pre-line break-words text-ink-muted">{part.address || "—"}</span>
          <MapLink lat={part.lat} lng={part.lng} />
        </span>
      )}

      {!compact && (
        <div className="flex flex-col gap-0.5 text-sm">
          <span className="text-xs font-bold text-ink-muted">{dead ? P.removedLines : P.lines}</span>
          {items.length === 0 && <span className="text-xs text-ink-muted">{P.noLines}</span>}
          {items.map((it, i) => (
            <span key={`${it.name}-${i}`} className={`flex flex-col ${dead ? "text-ink-muted line-through" : ""}`}>
              <span className="flex items-center justify-between gap-2">
                <span className="min-w-0 break-words">
                  {it.name}
                  {it.qty > 1 && ` × ${fmtNum(it.qty)}`}
                </span>
                <span className="shrink-0 text-xs text-ink-muted">{fmtMoney(it.unit_price * it.qty)}</span>
              </span>
              {it.options && <span className="text-xs text-ink-muted">{it.options}</span>}
              {it.note && <span className="text-xs text-accent-dark">{it.note}</span>}
            </span>
          ))}
        </div>
      )}

      {/* ── العلامات: انبعتله · جاهز · انستلم ── */}
      {!compact && !dead && (
        <span className="flex flex-wrap items-center gap-1.5 text-xs">
          <Badge variant={part.sent_at ? "success" : "neutral"}>
            {part.sent_at ? (part.sent_channel === "manual" ? P.sentManual : P.sent) : P.notSent}
          </Badge>
          <Badge variant={part.ready_at ? "success" : "neutral"}>{part.ready_at ? P.ready : P.notReady}</Badge>
          <Badge variant={collected ? "success" : "neutral"}>
            {collected ? (part.picked_up_by_name ? P.collectedBy.replace("{name}", part.picked_up_by_name) : P.collected) : P.notCollected}
          </Badge>
          {part.prep_minutes != null && part.prep_minutes > 0 && (
            <span className="text-ink-muted">{P.prep.replace("{n}", fmtNum(part.prep_minutes))}</span>
          )}
        </span>
      )}

      {/* ── السببُ ومن الغلطان ── */}
      {(part.reason || part.removed_fault) && (
        <span className="flex flex-col gap-0.5 text-xs">
          {part.reason && <span className="text-danger">{P.reason.replace("{reason}", part.reason)}</span>}
          {part.removed_fault && <span className="font-bold text-ink">{P.fault[part.removed_fault]}</span>}
        </span>
      )}
      {roleName && !compact && part.status !== "pending" && (
        <span className="text-xs text-ink-muted">{P.answeredBy.replace("{role}", roleName)}</span>
      )}

      {/* ── مالُ القسم ── */}
      {!compact && !dead && (part.subtotal != null || part.pay_amount != null) && (
        <span className="flex flex-wrap items-center gap-x-3 gap-y-0.5 text-xs text-ink-muted">
          {part.subtotal != null && (
            <span>
              {P.subtotal}: {fmtMoney(part.subtotal)}
            </span>
          )}
          {part.cost != null && (
            <span>
              {P.cost}: {fmtMoney(part.cost)}
            </span>
          )}
          {part.settlement_method === "wallet" ? (
            <span>{P.payWallet}</span>
          ) : (
            part.pay_amount != null &&
            part.pay_amount > 0 && <span>{P.pay.replace("{amount}", fmtMoney(part.pay_amount))}</span>
          )}
        </span>
      )}

      {/* ── أزرارُ القسم ── */}
      {!compact && ask === "" && (canAnswer || canRemove || canTransfer || canSend) && (
        <span className="flex flex-wrap items-center gap-1.5">
          {canAnswer && (
            <>
              <Button size="sm" disabled={busy !== ""} onClick={() => setAsk("accept")}>
                {P.accept}
              </Button>
              <Button size="sm" variant="danger-outline" disabled={busy !== ""} onClick={() => setAsk("reject")}>
                {P.reject}
              </Button>
            </>
          )}
          {canSend && (
            <>
              <Button size="sm" variant="secondary" disabled={busy !== "" || !part.phone} onClick={() => void sendWhatsApp()}>
                {P.send}
              </Button>
              <Button size="sm" variant="secondary" disabled={busy !== ""} onClick={() => void sendManual()}>
                {P.sendManual}
              </Button>
            </>
          )}
          {canTransfer && (
            <Button size="sm" variant="secondary" disabled={busy !== ""} onClick={() => setTransferring(true)}>
              {P.transfer}
            </Button>
          )}
          {canRemove && (
            <Button size="sm" variant="danger-outline" disabled={busy !== ""} onClick={() => setAsk("remove")}>
              {P.remove}
            </Button>
          )}
        </span>
      )}

      {/* ── السؤالُ قبل الفعل ── */}
      {!compact && ask !== "" && (
        <div className="flex flex-col gap-2" onClick={(e) => e.stopPropagation()}>
          {ask === "accept" ? (
            <Input
              id={`part-prep-${part.merchant_id}`}
              type="number"
              inputMode="numeric"
              min={0}
              dir="ltr"
              value={prep}
              onChange={(e) => setPrep(e.target.value)}
              placeholder={P.acceptPrep}
            />
          ) : (
            <Input
              id={`part-reason-${part.merchant_id}`}
              value={reason}
              onChange={(e) => {
                setReason(e.target.value);
                setErr("");
              }}
              placeholder={ask === "reject" ? P.rejectReason : P.removeReason}
            />
          )}
          {ask === "remove" && (
            <Select
              id={`part-fault-${part.merchant_id}`}
              label={P.faultLabel}
              value={fault}
              onChange={(e) => {
                setFault(e.target.value);
                setErr("");
              }}
            >
              <option value="">{P.faultLabel}</option>
              <option value="merchant">{P.fault.merchant}</option>
              <option value="platform">{P.fault.platform}</option>
            </Select>
          )}
          <FormActions
            busy={busy !== ""}
            tone={ask === "accept" ? undefined : "danger"}
            saveLabel={P.confirm}
            onSave={submitAsk}
            onCancel={() => {
              setAsk("");
              setReason("");
              setFault("");
              setPrep("");
              setErr("");
            }}
          />
        </div>
      )}
      {err && <p className="text-xs text-danger">{err}</p>}
      {transferring && (
        <TransferPanel
          orderId={order.id}
          part={{ merchantId: part.merchant_id, name: part.name }}
          onClose={() => setTransferring(false)}
          onDone={() => {
            setTransferring(false);
            onChanged();
          }}
        />
      )}
    </Box>
  );
}

/**
 * **قسمُ المتاجر في البطاقة** — صندوقٌ لكلّ متجر، **مرتّبةً بـ`seq`.**
 * `compact` لكرت السجلّ: كلُّ متجرٍ ونتيجتُه بلا أصنافٍ ولا أزرار.
 */
export function PartsSection({
  order,
  live,
  compact = false,
  onChanged,
}: {
  order: PartsOrder;
  live: boolean;
  compact?: boolean;
  onChanged?: () => void;
}) {
  const { can } = useAuth();
  const parts = [...(order.parts ?? [])].sort((a, b) => a.seq - b.seq);
  if (parts.length === 0) return null;
  const canAct = can("orders.intervene") && !!onChanged;
  return (
    <div className="flex min-w-0 flex-col gap-2" data-testid="order-parts">
      <span className="text-xs font-bold text-accent-text">
        {compact ? P.history : P.title.replace("{n}", fmtNum(parts.length))}
      </span>
      {parts.map((p) => (
        <PartBox
          key={p.merchant_id}
          order={order}
          part={p}
          live={live}
          compact={compact}
          canAct={canAct}
          onChanged={onChanged ?? (() => undefined)}
        />
      ))}
    </div>
  );
}
