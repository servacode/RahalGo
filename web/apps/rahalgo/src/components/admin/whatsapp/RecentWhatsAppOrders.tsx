"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«آخر طلبات الواتساب»** — قرارُ المالك ٢٠٢٦-١١-٠٥
 * ══════════════════════════════════════════════════════════════════════
 *
 * آخرُ عشرين طلباً بحالها وسائقها وأجرتها (`GET /admin/whatsapp-orders`)، **و«إلغاء الطلب»
 * قبل الاستلام بسببٍ إلزاميّ** (`POST /admin/whatsapp-orders/{id}/cancel`) — والمحرّكُ يحكم:
 * بعد الاستلام يُقال «الحلّ من بطاقة الطلب».
 *
 * **ومن هنا إلى لوحة الطلبات وسجلّها** بفلتر «طلبات الواتساب» — الطلبُ طلبٌ نظاميٌّ هناك.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDamascusTime, fmtMoney } from "@rahalgo/i18n";
import {
  AdminCard,
  AdminEmpty,
  AdminLoading,
  AdminReload,
  Alert,
  Button,
  FormActions,
  IconWhatsApp,
  Modal,
  Row,
  RowList,
  StatusBadge,
  Textarea,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const W = m.admin.whatsappOrders;

/** **طلبٌ في القائمة** — من حمولة المكتب. */
interface WaRow {
  id: string;
  number: number;
  status: string;
  created_at: string;
  merchant_name: string;
  customer_name: string;
  driver_name?: string | null;
  delivery_fee: number;
}

/** **قبل الاستلام وحدَه** — والمحرّكُ يحرسه (`wa_cancel_after_pickup`). */
const CANCELLABLE = new Set(["dispatching", "assigned", "at_pickup"]);

export function RecentWhatsAppOrders({ refresh }: { refresh: number }) {
  const [rows, setRows] = useState<WaRow[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [target, setTarget] = useState<WaRow | null>(null);
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [done, setDone] = useState("");

  const load = useCallback(async () => {
    try {
      const r = await api<{ orders: WaRow[] }>("/api/v1/admin/whatsapp-orders");
      setRows(r.orders ?? []);
      setFailed(false);
    } catch {
      setFailed(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, refresh]);

  async function cancel() {
    if (!target || busy) return;
    if (!reason.trim()) {
      setErr(W.cancelReason);
      return;
    }
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/whatsapp-orders/${target.id}/cancel`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim() }),
      });
      setDone(fill(W.cancelled, { n: target.number }));
      setTarget(null);
      setReason("");
      void load();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <AdminCard
      title={W.recent}
      padded={false}
      actions={
        <span className="flex flex-wrap gap-3 text-sm">
          <Link className="underline" href="/dashboard/orders?kind=whatsapp">
            {W.allOpen}
          </Link>
          <Link className="underline" href="/dashboard/history?kind=whatsapp">
            {W.history}
          </Link>
        </span>
      }
    >
      {done && (
        <div className="px-4 pb-3 lg:px-5">
          <Alert tone="success">{done}</Alert>
        </div>
      )}
      {rows === null && !failed && <AdminLoading rows={3} />}
      {failed && <AdminReload onRetry={() => void load()} />}
      {rows && rows.length === 0 && <AdminEmpty icon={IconWhatsApp} title={W.recentEmpty} />}
      {rows && rows.length > 0 && (
        <RowList className="rounded-none border-0">
          {rows.map((o) => (
            <Row
              key={o.id}
              title={
                <span className="flex flex-wrap items-center gap-2">
                  <Link className="underline" href={`/dashboard/orders?id=${encodeURIComponent(o.id)}`}>
                    #{o.number}
                  </Link>
                  <StatusBadge domain="order" status={o.status} />
                </span>
              }
              meta={
                <>
                  {o.merchant_name} ← {o.customer_name} · {fmtDamascusTime(o.created_at)} ·{" "}
                  {o.driver_name || W.noDriver}
                </>
              }
              trailing={
                <span className="flex flex-col items-end gap-1">
                  <span className="font-bold">{fmtMoney(o.delivery_fee)}</span>
                  {CANCELLABLE.has(o.status) && (
                    <Button
                      size="sm"
                      variant="danger-outline"
                      onClick={() => {
                        setTarget(o);
                        setReason("");
                        setErr("");
                        setDone("");
                      }}
                    >
                      {W.cancel}
                    </Button>
                  )}
                </span>
              }
            />
          ))}
        </RowList>
      )}

      <Modal open={target !== null} onClose={() => setTarget(null)} title={target ? fill(W.cancelTitle, { n: target.number }) : W.cancel}>
        <div className="space-y-3">
          <Textarea label={W.cancelReason} value={reason} maxLength={300} autoGrow onChange={(e) => setReason(e.target.value)} />
          {err && <Alert tone="error">{err}</Alert>}
          <FormActions
            onSave={() => void cancel()}
            onCancel={() => setTarget(null)}
            busy={busy}
            tone="danger"
            saveLabel={W.cancelConfirm}
          />
        </div>
      </Modal>
    </AdminCard>
  );
}
