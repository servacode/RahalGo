"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تعويضُ البضاعة الراجعة — في «الخسائر والنزاعات»** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **العمليّاتُ تقول أين البضاعة** (`POST /orders/{id}/goods`)، **والماليّةُ تكتب
 * كم** — وكان الزرُّ في لوح الطلبات وحدَه، **والماليّةُ لا تراه** (هجرة `0422`).
 * فالقائمةُ هنا (`GET /losses/goods-compensations`)، **والتعويضُ من بابه القائم**
 * (`POST /orders/{id}/goods/compensation`، `finance.manage`): سقفُه سعرُ شراء البضاعة.
 * **وحركةُ المالك أو المدير تنفذ فوراً، وطلبُ الموظّف ينتظر في «التعويضات»** (قرارُ المالك
 * ٢٠٢٦-١٠-٢٧) — والرسالةُ تُقرأ من الردّ (`applied`).
 *
 * **والمرفوضُ يُقترح من جديد** (فحصُ ٢٠٢٦-١٠-٢٧) — كان يبقى في القائمة بلا فعل؛ والمحرّكُ يقبل
 * طلباً جديداً بعد الرفض (فهرسُ «الحيّ» معلَّقٌ أو مقبولٌ فقط).
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtMoney, fmtNum, fmtRef, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  FormActions,
  Input,
  LoadingState,
  ReloadState,
  IconStore,
  IconOrder,
  IconDate,
  IconSwap,
  IconHourglass,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanOpen } from "@/lib/policy";
import { OpenLink } from "@/components/admin/OpenLink";
import { CaseEmpty } from "./caseTable";
import { StatTile } from "./StatTile";

import { SectionBar } from "@/components/admin/support/SectionBar";
const m = getMessages(defaultLocale);
const L = m.admin.losses;
const G = m.admin.ordersPage.board.goods;

interface Row {
  order_id: string;
  order_number: number;
  merchant_name: string;
  goods_cost: number;
  closed_at: string | null;
  request_status: "" | "pending" | "rejected";
  request_amount: number;
}

export function GoodsCompensationView() {
  const { can } = useAuth();
  const canWrite = can("finance.manage");
  const canOrder = useCanOpen().order;
  const { data, error, reload } = useLiveData<{ orders: Row[] }>(
    () => api("/api/v1/admin/losses/goods-compensations"),
    ["order", "wallet"],
  );
  const [open, setOpen] = useState<string | null>(null);
  const [amount, setAmount] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [sent, setSent] = useState("");
  /**
   * **«ما في تعويض»** (قرارُ المالك ٢٠٢٦-١٠-٢٧) — بتأكيدٍ وسببٍ اختياريّ، فيخرج الطلبُ من
   * القائمة نهائياً بلا مال (`POST /orders/{id}/goods/compensation/none`، `finance.manage`).
   */
  const [noneFor, setNoneFor] = useState<string | null>(null);
  const [noneReason, setNoneReason] = useState("");

  async function markNone(r: Row) {
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${r.order_id}/goods/compensation/none`, {
        method: "POST",
        body: JSON.stringify({ reason: noneReason.trim() }),
      });
      setNoneFor(null);
      setNoneReason("");
      setSent(L.goodsNoneDone);
      reload();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function submit(r: Row) {
    const v = Math.round(Number(amount));
    if (!Number.isFinite(v) || v <= 0 || v > r.goods_cost) {
      setErr(G.compBad);
      return;
    }
    setBusy(true);
    setErr("");
    try {
      const res = await api<{ applied?: boolean; compensated?: number }>(
        `/api/v1/admin/orders/${r.order_id}/goods/compensation`,
        { method: "POST", body: JSON.stringify({ amount: v }) },
      );
      setOpen(null);
      setAmount("");
      setSent(res?.applied ? L.goodsApplied.replace("{amount}", fmtMoney(res.compensated ?? v)) : L.goodsSent);
      reload();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!data && error) return <ReloadState label={errorText(error)} onRetry={reload} />;
  if (!data) return <LoadingState />;
  const rows = data.orders ?? [];
  const waiting = rows.filter((r) => r.request_status !== "pending");
  const pending = rows.filter((r) => r.request_status === "pending");

  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) — رقمان فوق، وكرتٌ لكلّ طلبٍ رجعت بضاعتُه: المتجرُ والطلبُ
  // وسعرُ البضاعة كبيراً، وزرُّ التعويض بعرض البطاقة.
  return (
    <div className="space-y-4">
      <SectionBar hint={L.goodsHint} />
      {sent && (
        <Alert tone="success" onDismiss={() => setSent("")}>
          {sent}
        </Alert>
      )}
      {rows.length > 0 && (
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <StatTile
            icon={IconSwap}
            label={L.goodsWaitingTile}
            value={fmtNum(waiting.length)}
            sub={L.goodsWaitingSub.replace("{amount}", fmtMoney(waiting.reduce((a, r) => a + r.goods_cost, 0)))}
            tone={waiting.length > 0 ? "warning" : "muted"}
          />
          <StatTile
            icon={IconHourglass}
            label={L.goodsPendingTile}
            value={fmtNum(pending.length)}
            sub={L.goodsPendingSub.replace("{amount}", fmtMoney(pending.reduce((a, r) => a + r.request_amount, 0)))}
            tone={pending.length > 0 ? "accent" : "muted"}
          />
        </div>
      )}
      {rows.length === 0 ? (
        <CaseEmpty icon={IconSwap} title={L.goodsEmpty} hint={L.goodsEmptyHint} />
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {rows.map((r) => (
            <li key={r.order_id} className="surface flex flex-col gap-3 p-4">
              <div className="flex items-start gap-3">
                <span
                  className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${
                    r.request_status === "pending"
                      ? "bg-accent text-on-bright"
                      : r.request_status === "rejected"
                        ? "bg-danger-solid text-on-solid"
                        : "bg-warning-solid text-on-solid"
                  }`}
                >
                  <IconStore size={20} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-bold">{r.merchant_name}</span>
                  <span className="flex items-center gap-1 text-2xs text-ink-muted">
                    <IconOrder size={12} />
                    <OpenLink
                      allowed={canOrder}
                      href={`/dashboard/orders?id=${r.order_id}`}
                      dir="ltr"
                      className="font-bold text-primary-dark hover:underline"
                    >
                      #{fmtRef(r.order_number)}
                    </OpenLink>
                  </span>
                </span>
                {r.request_status === "rejected" && (
                  <span className="shrink-0">
                    <Badge variant="danger">{L.goodsRejected}</Badge>
                  </span>
                )}
              </div>

              <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
                <span>
                  <span className="block text-2xs text-ink-muted">{L.goodsCost}</span>
                  <span className="figure block">{fmtMoney(r.goods_cost)}</span>
                </span>
                {r.closed_at && (
                  <span className="flex items-center gap-1 text-2xs text-ink-muted">
                    <IconDate size={12} />
                    <span dir="ltr">{fmtDateTime(r.closed_at)}</span>
                  </span>
                )}
              </div>

              {r.request_status === "pending" && (
                <p className="rounded-control bg-accent px-2.5 py-1 text-xs font-bold text-on-bright">
                  {L.goodsPending} · {fmtMoney(r.request_amount)}
                </p>
              )}
              {r.request_status === "rejected" && (
                <p className="text-xs text-ink-muted">{L.goodsRejectedHint}</p>
              )}

              {open === r.order_id ? (
                <div className="space-y-2 border-t border-line-soft pt-3">
                  <p className="text-xs text-ink-muted">{L.goodsFormHint.replace("{max}", fmtMoney(r.goods_cost))}</p>
                  <div className="flex items-end gap-2">
                    <div className="min-w-0 flex-1">
                      <Input
                        id={`goods-amount-${r.order_id}`}
                        label={`${L.goodsAmount} (${m.common.currency})`}
                        type="number"
                        inputMode="numeric"
                        dir="ltr"
                        value={amount}
                        onChange={(e) => {
                          setAmount(e.target.value);
                          setErr("");
                        }}
                      />
                    </div>
                    <Button
                      variant="secondary"
                      onClick={() => {
                        setAmount(String(r.goods_cost));
                        setErr("");
                      }}
                    >
                      {L.goodsFull}
                    </Button>
                  </div>
                  {err && <p className="text-xs text-danger">{err}</p>}
                  <FormActions
                    busy={busy}
                    onSave={() => void submit(r)}
                    onCancel={() => {
                      setOpen(null);
                      setErr("");
                    }}
                    saveLabel={L.goodsSend}
                  />
                </div>
              ) : noneFor === r.order_id ? (
                <div className="space-y-2 border-t border-line-soft pt-3">
                  <p className="text-xs text-ink-muted">{L.goodsNoneConfirm}</p>
                  <Input
                    id={`goods-none-reason-${r.order_id}`}
                    label={L.goodsNoneReason}
                    value={noneReason}
                    maxLength={500}
                    onChange={(e) => {
                      setNoneReason(e.target.value);
                      setErr("");
                    }}
                  />
                  {err && <p className="text-xs text-danger">{err}</p>}
                  <FormActions
                    busy={busy}
                    tone="danger"
                    onSave={() => void markNone(r)}
                    onCancel={() => {
                      setNoneFor(null);
                      setErr("");
                    }}
                    saveLabel={L.goodsNoneYes}
                  />
                </div>
              ) : (
                canWrite &&
                r.request_status !== "pending" && (
                  <div className="mt-auto flex flex-col gap-2 border-t border-line-soft pt-3">
                    <Button
                      className="w-full"
                      variant={r.request_status === "rejected" ? "secondary" : "primary"}
                      onClick={() => {
                        setOpen(r.order_id);
                        setNoneFor(null);
                        setAmount("");
                        setErr("");
                      }}
                    >
                      {r.request_status === "rejected" ? L.goodsRetry : G.compensate}
                    </Button>
                    <Button
                      className="w-full"
                      variant="ghost"
                      onClick={() => {
                        setNoneFor(r.order_id);
                        setOpen(null);
                        setNoneReason("");
                        setErr("");
                      }}
                    >
                      {L.goodsNone}
                    </Button>
                  </div>
                )
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
