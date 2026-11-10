"use client";

/**
 * **«طلب تعديل من الكابتن» على كرت الطلب** (قرارُ المالك ٢٠٢٦-١١-١٠)
 *
 * الكابتن بعت: سطر «ناقص» أو «الزبون لغاه» أو «انزاد صنف». **والمكتب يقرّر هون**: السطور قبل وبعد،
 * **ومعاينةُ المحرّك نفسِه** (المجموع والمستحقّ النقديّ والمحفظة وحق المتجر) — الرقم اللي رح يصير
 * بالضبط، لا حسبة بالشاشة. «وافق» بتمشي من باب تعديل الطلب، و«ارفض» بسبب بيوصل الكابتن.
 * **وبعد الاستلام الموافقة للأدمن والمالك بس** (البند ٤) — والشاشة بتقولها بدل زرّ بيرد ٤٠٣.
 */
import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import { Alert, Button, Input, LoadingState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import type { EditResult } from "./OrderEditor";

const R = getMessages(defaultLocale).admin.ordersPage.editRequest;

/** **بابا القرار** — بقدرة التدخّل، والموافقة بعد الاستلام يحرسها المحرّك بقدرةٍ أضيق. */
export const EDIT_REQUEST_APPROVE_ROUTE = ["POST", "/orders/{id}/edit-request/approve"] as const;

interface ReqLine {
  item_id: string;
  name: string;
  action: "missing" | "cancelled";
  old_qty: number;
  new_qty: number;
}
interface ReqAdd {
  name: string;
  qty: number;
  price: number;
}
interface EditRequestView {
  id: string;
  status: string;
  lines: ReqLine[];
  adds: ReqAdd[];
  note: string;
  old_total: number;
  new_total: number;
  after_pickup: boolean;
  stale: boolean;
  preview?: EditResult;
  preview_error?: string;
}

const fill = (s: string, v: Record<string, string | number>) =>
  s.replace(/\{(\w+)\}/g, (_, k: string) => String(v[k] ?? ""));

export function EditRequestBox({ orderId, onChanged }: { orderId: string; onChanged: () => void }) {
  const canCall = useCanCall();
  const canDecide = canCall(...EDIT_REQUEST_APPROVE_ROUTE);
  const [req, setReq] = useState<EditRequestView | null | undefined>(undefined);
  const [canAfterPickup, setCanAfterPickup] = useState(false);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState("");

  useEffect(() => {
    let live = true;
    api<{ request: EditRequestView | null; can_approve_after_pickup: boolean }>(
      `/api/v1/admin/orders/${orderId}/edit-request`,
    )
      .then((r) => {
        if (!live) return;
        setReq(r.request);
        setCanAfterPickup(r.can_approve_after_pickup);
      })
      .catch((e) => live && setErr(errorText(e)));
    return () => {
      live = false;
    };
  }, [orderId]);

  if (req === undefined) return err ? <Alert tone="error">{err}</Alert> : <LoadingState />;
  if (req === null) return null;

  async function decide(path: "approve" | "reject") {
    if (!req) return;
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${orderId}/edit-request/${path}`, {
        method: "POST",
        body: JSON.stringify(path === "approve" ? { request_id: req.id } : { request_id: req.id, reason }),
      });
      onChanged();
      setReq(null);
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const p = req.preview;
  const blockedAfterPickup = req.after_pickup && !canAfterPickup;
  return (
    <div className="flex w-full flex-col gap-1.5 rounded-control border border-warning-edge bg-warning-tint p-3 text-sm">
      <p className="font-bold text-ink">{R.title}</p>
      <ul className="flex flex-col gap-0.5">
        {req.lines.map((l) => (
          <li key={l.item_id}>
            <span className="font-medium">{l.action === "missing" ? R.missing : R.cancelled}</span>
            {" — "}
            {fill(R.line, { name: l.name, old: l.old_qty, new: l.new_qty })}
          </li>
        ))}
        {req.adds.map((a, i) => (
          <li key={`add-${i}`}>
            <span className="font-medium">{R.added}</span>
            {" — "}
            {fill(R.addLine, { name: a.name, qty: a.qty, price: fmtMoney(a.price) })}
          </li>
        ))}
      </ul>
      {req.note && <p className="text-ink-muted">{fill(R.note, { note: req.note })}</p>}
      <p className="font-bold">
        {fill(R.totals, { old: fmtMoney(p?.old_total ?? req.old_total), new: fmtMoney(p?.new_total ?? req.new_total) })}
      </p>
      {p && (
        <div className="flex flex-col gap-0.5 text-ink-muted">
          <span>{fill(R.cashDue, { amount: fmtMoney(p.cash_due) })}</span>
          {p.refunded > 0 && <span>{fill(R.refunded, { amount: fmtMoney(p.refunded) })}</span>}
          {p.charged > 0 && <span>{fill(R.charged, { amount: fmtMoney(p.charged) })}</span>}
          {(p.stores ?? []).map((s) => (
            <span key={s.merchant_id}>
              {fill(R.storeDue, { name: s.name, old: fmtMoney(s.old_due), new: fmtMoney(s.new_due) })}
            </span>
          ))}
        </div>
      )}
      {req.stale && <Alert tone="warning">{R.stale}</Alert>}
      {req.preview_error && <Alert tone="error">{errorText({ body: { message_key: `errors.${req.preview_error}` } })}</Alert>}
      {blockedAfterPickup && <Alert tone="warning">{R.afterPickup}</Alert>}
      {err && <Alert tone="error">{err}</Alert>}
      {canDecide && !rejecting && (
        <div className="flex flex-wrap gap-2">
          <Button
            disabled={busy || req.stale || blockedAfterPickup || !!req.preview_error}
            onClick={() => void decide("approve")}
          >
            {R.approve}
          </Button>
          <Button variant="secondary" disabled={busy} onClick={() => setRejecting(true)}>
            {R.reject}
          </Button>
        </div>
      )}
      {canDecide && rejecting && (
        <div className="flex flex-col gap-1.5">
          <Input value={reason} onChange={(e) => setReason(e.target.value)} placeholder={R.rejectReason} />
          <div className="flex flex-wrap gap-2">
            <Button variant="danger" disabled={busy || reason.trim() === ""} onClick={() => void decide("reject")}>
              {R.confirmReject}
            </Button>
            <Button variant="secondary" disabled={busy} onClick={() => setRejecting(false)}>
              {getMessages(defaultLocale).common.back}
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}
