"use client";

/**
 * **«طلبات البوت بانتظار الموافقة»** — طلباتُ قائمة بوت الواتساب (`wa_bot`، حالتها `pending`).
 * الموظّفُ يحدّد الأجرة ويوافق فيُرسَل الطلبُ للكباتن، أو يرفض بسببٍ إلزاميّ.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDamascusTime, fmtNum } from "@rahalgo/i18n";
import {
  AdminCard,
  AdminLoading,
  AdminReload,
  Alert,
  Button,
  FormActions,
  Input,
  Modal,
  Textarea,
  useLiveEvent,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const B = m.admin.whatsappOrders.bot;

interface BotRow {
  id: string;
  number: number;
  status: string;
  created_at: string;
  wa_bot?: boolean;
  parcel_note?: string;
  pickup_name?: string;
  merchant_name?: string;
  recipient_phone?: string;
  customer_phone?: string;
  dropoff_known?: boolean;
  lat?: number | null;
  lng?: number | null;
}

interface WaQuote {
  fee: number;
  fee_min: number;
  fee_max: number;
}

export function BotOrdersPending({ refresh, onChanged }: { refresh: number; onChanged: () => void }) {
  const [rows, setRows] = useState<BotRow[] | null>(null);
  const [failed, setFailed] = useState(false);
  const [quote, setQuote] = useState<WaQuote | null>(null);
  const [fees, setFees] = useState<Record<string, string>>({});
  const [names, setNames] = useState<Record<string, string>>({});
  const [busyId, setBusyId] = useState("");
  const [msg, setMsg] = useState<{ tone: "success" | "error"; text: string } | null>(null);
  const [rejectFor, setRejectFor] = useState<BotRow | null>(null);
  const [reason, setReason] = useState("");
  const [err, setErr] = useState("");

  const load = useCallback(async () => {
    try {
      const r = await api<{ orders: BotRow[] }>("/api/v1/admin/whatsapp-orders");
      setRows((r.orders ?? []).filter((o) => o.wa_bot && o.status === "pending"));
      setFailed(false);
    } catch {
      setFailed(true);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load, refresh]);

  useEffect(() => {
    api<WaQuote>("/api/v1/admin/whatsapp-orders/quote")
      .then(setQuote)
      .catch(() => setQuote(null));
  }, []);

  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  useLiveEvent((event) => {
    const origin = (event.order as { origin?: string } | undefined)?.origin;
    if ((event.type === "order" && origin === "whatsapp") || event.type === "orders") {
      if (timer.current) clearTimeout(timer.current);
      timer.current = setTimeout(() => void load(), 500);
    }
  });
  useEffect(
    () => () => {
      if (timer.current) clearTimeout(timer.current);
    },
    [],
  );

  async function approve(o: BotRow) {
    if (busyId) return;
    const raw = (fees[o.id] ?? "").replace(/\D/g, "");
    const fee = raw === "" ? NaN : Number(raw);
    if (!Number.isFinite(fee) || fee <= 0) {
      setMsg({ tone: "error", text: B.feeBad });
      return;
    }
    const body: { fee: number; store_name?: string } = { fee };
    const nm = (names[o.id] ?? "").trim();
    if (nm) body.store_name = nm;
    setBusyId(o.id);
    setMsg(null);
    try {
      await api(`/api/v1/admin/whatsapp-orders/${o.id}/approve`, { method: "POST", body: JSON.stringify(body) });
      setMsg({ tone: "success", text: fill(B.approved, { n: o.number }) });
      onChanged();
    } catch (e) {
      setMsg({ tone: "error", text: errorText(e) });
    } finally {
      setBusyId("");
      void load();
    }
  }

  async function reject() {
    if (!rejectFor || busyId) return;
    if (!reason.trim()) {
      setErr(B.rejectReason);
      return;
    }
    setBusyId(rejectFor.id);
    setErr("");
    try {
      await api(`/api/v1/admin/whatsapp-orders/${rejectFor.id}/reject`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim() }),
      });
      setMsg({ tone: "success", text: fill(B.rejected, { n: rejectFor.number }) });
      setRejectFor(null);
      setReason("");
      onChanged();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusyId("");
      void load();
    }
  }

  if (rows !== null && rows.length === 0 && !failed && !msg) return null;

  return (
    <AdminCard title={`${B.title}${rows ? ` (${fmtNum(rows.length)})` : ""}`}>
      {msg && (
        <Alert tone={msg.tone} className="mb-3" onDismiss={() => setMsg(null)}>
          {msg.text}
        </Alert>
      )}
      {rows === null && !failed && <AdminLoading rows={2} />}
      {failed && <AdminReload onRetry={() => void load()} />}
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        {rows?.map((o) => {
          const store = o.pickup_name || o.merchant_name || "";
          const phone = o.recipient_phone || o.customer_phone || "";
          const hasLoc = o.dropoff_known ?? (o.lat != null && o.lng != null);
          return (
            <div key={o.id} className="flex flex-col gap-3 rounded-card border border-line p-3">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="font-bold">#{o.number}</span>
                <span className="text-sm text-ink-muted">{fmtDamascusTime(o.created_at)}</span>
              </div>
              <div className="text-sm">
                {B.customerPhone}: <span dir="ltr">{phone}</span>
              </div>
              <div className="text-sm">
                {B.storeName}: {store}
              </div>
              <div className="text-sm">
                <div className="text-ink-muted">{B.text}</div>
                <p className="whitespace-pre-wrap">{o.parcel_note}</p>
              </div>
              <div className="text-sm">
                {hasLoc ? (
                  o.lat != null && o.lng != null ? (
                    <a
                      className="underline"
                      href={`https://www.google.com/maps?q=${o.lat},${o.lng}`}
                      target="_blank"
                      rel="noreferrer"
                    >
                      {B.locationReceived}
                    </a>
                  ) : (
                    B.locationReceived
                  )
                ) : (
                  <span className="text-warning">{B.noLocation}</span>
                )}
              </div>
              <Input
                label={B.storeNameEdit}
                value={names[o.id] ?? ""}
                maxLength={120}
                onChange={(e) => setNames((s) => ({ ...s, [o.id]: e.target.value }))}
              />
              <Input
                label={B.fee}
                value={fees[o.id] ?? ""}
                dir="ltr"
                inputMode="numeric"
                placeholder={quote ? String(quote.fee) : undefined}
                onChange={(e) => setFees((s) => ({ ...s, [o.id]: e.target.value }))}
              />
              {quote && (
                <p className="text-xs text-ink-muted">
                  {fill(B.feeRange, { min: fmtNum(quote.fee_min), max: fmtNum(quote.fee_max) })}
                </p>
              )}
              <div className="flex flex-wrap gap-2">
                <Button size="sm" disabled={busyId === o.id} onClick={() => void approve(o)}>
                  {B.approve}
                </Button>
                <Button
                  size="sm"
                  variant="danger-outline"
                  disabled={busyId === o.id}
                  onClick={() => {
                    setRejectFor(o);
                    setReason("");
                    setErr("");
                  }}
                >
                  {B.reject}
                </Button>
              </div>
            </div>
          );
        })}
      </div>

      <Modal
        open={rejectFor !== null}
        onClose={() => setRejectFor(null)}
        title={rejectFor ? fill(B.rejectTitle, { n: rejectFor.number }) : B.reject}
      >
        <div className="space-y-3">
          <Textarea label={B.rejectReason} value={reason} maxLength={300} autoGrow onChange={(e) => setReason(e.target.value)} />
          {err && <Alert tone="error">{err}</Alert>}
          <FormActions
            onSave={() => void reject()}
            onCancel={() => setRejectFor(null)}
            busy={busyId !== ""}
            tone="danger"
            saveLabel={B.rejectConfirm}
          />
        </div>
      </Modal>
    </AdminCard>
  );
}
