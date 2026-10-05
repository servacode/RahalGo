"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تعويضُ البضاعة الراجعة — في «الخسائر والنزاعات»** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **العمليّاتُ تقول أين البضاعة** (`POST /orders/{id}/goods`)، **والماليّةُ تكتب
 * كم** — وكان الزرُّ في لوح الطلبات وحدَه، **والماليّةُ لا تراه** (هجرة `0422`).
 * فالقائمةُ هنا (`GET /losses/goods-compensations`)، **والتعويضُ من بابه القائم**
 * (`POST /orders/{id}/goods/compensation`، `finance.manage`): سقفُه سعرُ شراء البضاعة،
 * ويذهب إلى صفحة «التعويضات» **فيوافق عليه غيرُ كاتبه.**
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtMoney, fmtRef, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  EmptyState,
  FormActions,
  Input,
  LoadingState,
  PageHeader,
  ReloadState,
  IconBalance,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

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
  const { data, error, reload } = useLiveData<{ orders: Row[] }>(
    () => api("/api/v1/admin/losses/goods-compensations"),
    ["order", "wallet"],
  );
  const [open, setOpen] = useState<string | null>(null);
  const [amount, setAmount] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [sent, setSent] = useState("");

  async function submit(r: Row) {
    const v = Math.round(Number(amount));
    if (!Number.isFinite(v) || v <= 0 || v > r.goods_cost) {
      setErr(G.compBad);
      return;
    }
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${r.order_id}/goods/compensation`, {
        method: "POST",
        body: JSON.stringify({ amount: v }),
      });
      setOpen(null);
      setAmount("");
      setSent(L.goodsSent);
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

  return (
    <div className="space-y-3">
      <PageHeader icon={IconBalance} title={L.goodsTitle} subtitle={L.goodsHint} />
      {sent && (
        <Alert tone="success" onDismiss={() => setSent("")}>
          {sent}
        </Alert>
      )}
      {rows.length === 0 ? (
        <EmptyState icon={IconBalance} title={L.goodsEmpty} />
      ) : (
        <ul className="space-y-2">
          {rows.map((r) => (
            <li key={r.order_id} className="surface space-y-2 p-3">
              <div className="flex flex-wrap items-center gap-3">
                <span className="font-bold" dir="ltr">
                  #{fmtRef(r.order_number)}
                </span>
                <span className="min-w-0 flex-1 font-medium">{r.merchant_name}</span>
                <span className="text-sm">
                  {L.goodsCost}: {fmtMoney(r.goods_cost)}
                </span>
                {r.closed_at && (
                  <span className="text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(r.closed_at)}
                  </span>
                )}
                {r.request_status === "pending" && (
                  <Badge variant="info">
                    {L.goodsPending} · {fmtMoney(r.request_amount)}
                  </Badge>
                )}
                {r.request_status === "rejected" && <Badge variant="danger">{L.goodsRejected}</Badge>}
                {canWrite && r.request_status === "" && open !== r.order_id && (
                  <Button
                    variant="secondary"
                    onClick={() => {
                      setOpen(r.order_id);
                      setAmount("");
                      setErr("");
                    }}
                  >
                    {G.compensate}
                  </Button>
                )}
              </div>
              {open === r.order_id && (
                <div className="space-y-2">
                  <p className="text-xs text-ink-muted">{G.compHintQueue.replace("{max}", fmtMoney(r.goods_cost))}</p>
                  <Input
                    type="number"
                    inputMode="numeric"
                    placeholder={G.compLabel}
                    value={amount}
                    onChange={(e) => {
                      setAmount(e.target.value);
                      setErr("");
                    }}
                  />
                  {err && <p className="text-xs text-danger">{err}</p>}
                  <FormActions
                    busy={busy}
                    onSave={() => void submit(r)}
                    onCancel={() => {
                      setOpen(null);
                      setErr("");
                    }}
                    saveLabel={G.compSendQueue}
                  />
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
