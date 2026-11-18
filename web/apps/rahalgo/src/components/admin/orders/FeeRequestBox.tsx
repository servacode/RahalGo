"use client";

/**
 * **«الكابتن طلب تعديل الأجرة» على كرت الطلب** (قرارُ المالك ٢٠٢٦-١١-١٨)
 *
 * طلب الواتساب: الكابتن بيكتب الأجرة الجديدة والسبب، **والمكتب يقرّر هون** بمعاينة المحرّك نفسِه
 * (`dry_run` على باب الموافقة): الأجرة قبل وبعد، وما ياخده الكابتن، وحصّة المنصّة، والنقد اللي بينقبض.
 * «وافق» بتمشي من باب «تعديل الأجرة»، و«ارفض» بسبب بيوصل الكابتن.
 */
import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import { Alert, Button, Input } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import type { FeeEditResult } from "./FeeEditor";

const R = getMessages(defaultLocale).admin.ordersPage.feeRequest;

/** **بابا القرار** — بقدرة التدخّل. */
export const FEE_REQUEST_APPROVE_ROUTE = ["POST", "/orders/{id}/fee-request/approve"] as const;

export interface FeeRequestBrief {
  id: string;
  old_fee: number;
  fee: number;
  reason: string;
  status: string;
  created_at: string;
  driver_name?: string;
}

const fill = (s: string, v: Record<string, string | number>) =>
  s.replace(/\{(\w+)\}/g, (_, k: string) => String(v[k] ?? ""));

export function FeeRequestBox({
  orderId,
  request,
  onChanged,
}: {
  orderId: string;
  request: FeeRequestBrief;
  onChanged: () => void;
}) {
  const canCall = useCanCall();
  const canDecide = canCall(...FEE_REQUEST_APPROVE_ROUTE);
  const [preview, setPreview] = useState<FeeEditResult | null>(null);
  const [previewErr, setPreviewErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [rejecting, setRejecting] = useState(false);
  const [reason, setReason] = useState("");
  const [gone, setGone] = useState(false);

  useEffect(() => {
    if (!canDecide) return;
    let live = true;
    api<{ result: FeeEditResult }>(`/api/v1/admin/orders/${orderId}/fee-request/approve`, {
      method: "POST",
      body: JSON.stringify({ request_id: request.id, dry_run: true }),
    })
      .then((r) => live && setPreview(r.result))
      .catch((e) => live && setPreviewErr(errorText(e)));
    return () => {
      live = false;
    };
  }, [orderId, request.id, canDecide]);

  if (gone) return null;

  async function decide(path: "approve" | "reject") {
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${orderId}/fee-request/${path}`, {
        method: "POST",
        body: JSON.stringify(path === "approve" ? { request_id: request.id } : { request_id: request.id, reason }),
      });
      setGone(true);
      onChanged();
    } catch (e) {
      setErr(errorText(e));
      onChanged();
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="flex w-full flex-col gap-1.5 rounded-control border border-warning-edge bg-warning-tint p-3 text-sm">
      <p className="font-bold text-ink">
        {request.driver_name ? fill(R.titleBy, { name: request.driver_name }) : R.title}
      </p>
      <p className="font-bold">{fill(R.fees, { old: fmtMoney(request.old_fee), new: fmtMoney(request.fee) })}</p>
      <p className="text-ink-muted">{fill(R.reason, { reason: request.reason })}</p>
      {preview && (
        <div className="flex flex-col gap-0.5 text-ink-muted">
          <span>{fill(R.driverGets, { old: fmtMoney(preview.old_driver_fee), new: fmtMoney(preview.driver_fee) })}</span>
          <span>
            {fill(R.platformGets, { old: fmtMoney(preview.old_platform_share), new: fmtMoney(preview.platform_share) })}
          </span>
          <span>{fill(R.cashDue, { old: fmtMoney(preview.old_cash_due), new: fmtMoney(preview.cash_due) })}</span>
        </div>
      )}
      {previewErr && <Alert tone="error">{previewErr}</Alert>}
      {err && <Alert tone="error">{err}</Alert>}
      {canDecide && !rejecting && (
        <div className="flex flex-wrap gap-2">
          <Button disabled={busy || !preview} onClick={() => void decide("approve")}>
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
