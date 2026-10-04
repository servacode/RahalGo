"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صفحةُ الطارئ** — قراراتُ المالك ٢٠٢٦-١٠-٠٤ (غرفةُ الطوارئ)
 * ══════════════════════════════════════════════════════════════════════
 *
 * الطلبُ وحالُه والسائقُ الجديد · البضاعةُ (استُلمت أم لا) والمالُ مع السائق وطلباتُه
 * الأخرى · اتّصالٌ بالسائق والزبون والمتجر · خريطةٌ بموقع السائق **الحيّ**.
 *
 * **وخطواتُ الحلّ بترتيبها**: استلمتها ← السائقُ بخير؟ ← مصيرُ الطلب ← المال ← تمّ —
 * **والمالُ طلبُ تعويضٍ إلى الماليّة لا دفع.** **وسجلُّ ملاحظاتٍ** يُضاف إليه ولا يُعدَّل.
 *
 * **ومتجرٌ أُغلق طارئاً** تُعرض طلباتُه المفتوحة، **والمكتبُ يحوّلها أو يلغيها.**
 */

import { useEffect, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import { useParams } from "next/navigation";
import {
  getMessages,
  defaultLocale,
  fmtRef,
  fmtDateTime,
  fmtSpan,
  fmtMoney,
  errorText,
} from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Card,
  FormActions,
  Input,
  LoadingState,
  Modal,
  PageContainer,
  PageHeader,
  Textarea,
  useLiveData,
  IconWarning,
  IconPhone,
  IconCheck,
  IconLocation,
  IconNote,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { kindVariant } from "@/components/admin/emergency/kinds";
import { TransferPanel } from "@/components/admin/orders/TransferPanel";

const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
});

const m = getMessages(defaultLocale);
const R = m.admin.emergencyRoom;
const statusText = (s: string) => (m.orders.status as Record<string, string>)[s] ?? s;
const ago = (iso: string) => fmtSpan(Math.max(0, (Date.now() - Date.parse(iso)) / 1000));

interface Person {
  id: string;
  name: string;
  phone: string;
}
interface Row {
  id: string;
  number: number;
  status: string;
  picked_up: boolean;
  customer: string;
}
interface Detail {
  id: string;
  kind: keyof typeof R.kinds;
  stage: "" | keyof typeof R.stages;
  status: "open" | "resolved";
  note: string;
  lat: number | null;
  lng: number | null;
  created_at: string;
  stale: boolean;
  acknowledged_at: string | null;
  acknowledged_by: string;
  driver_ok_at: string | null;
  driver_ok_by: string;
  outcome: "" | keyof typeof R.outcomes;
  outcome_at: string | null;
  outcome_by: string;
  money_skipped: boolean;
  money_at: string | null;
  money_request: {
    id: string;
    status: keyof typeof R.moneyStatus;
    suggested_amount: number;
    amount: number | null;
  } | null;
  resolution: string;
  resolved_at: string | null;
  resolved_by: string;
  next_step: "" | "ack" | "driver_ok" | "outcome" | "money" | "resolve";
  driver: (Person & {
    lat: number | null;
    lng: number | null;
    location_at: string | null;
    on_shift: boolean;
    accident_locked: boolean;
    cash_held: number;
  }) | null;
  order: {
    id: string;
    number: number;
    status: string;
    picked_up: boolean;
    picked_up_at: string | null;
    total: number;
    cash_due: number;
    payment_method: string;
    address_text: string;
    customer: Person | null;
    store: Person | null;
    new_driver: Person | null;
  } | null;
  other_orders: Row[];
  merchant: (Person & { emergency_closed: boolean }) | null;
  store_orders: Row[];
  notes: { id: number; author: string; body: string; created_at: string }[];
}

function Call({ phone, label }: { phone: string | undefined; label: string }) {
  if (!phone) return null;
  return (
    <a
      href={`tel:${phone}`}
      className="inline-flex items-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm"
    >
      <IconPhone size={15} />
      {label}
      <span dir="ltr" className="text-ink-muted">
        {phone}
      </span>
    </a>
  );
}

function KV({ k, v }: { k: string; v: React.ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 text-sm">
      <span className="text-ink-muted">{k}</span>
      <span className="text-end">{v}</span>
    </div>
  );
}

export default function EmergencyPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const { capabilities } = useAuth();
  const canIntervene = capabilities.includes("orders.intervene");
  // **وموقعُ السائق حيٌّ** — كاتبُ الموضع لا يبثّ، فيُعاد السؤالُ كلَّ نصف دقيقة.
  const [tick, setTick] = useState(0);
  useEffect(() => {
    const t = window.setInterval(() => setTick((x) => x + 1), 30_000);
    return () => window.clearInterval(t);
  }, []);
  const { data, error: loadErr, reload } = useLiveData<Detail>(
    () => api<Detail>(`/api/v1/admin/emergencies/${id}`),
    ["emergency", "order", "driver"],
    [id, tick],
  );
  const [busy, setBusy] = useState("");
  const [err, setErr] = useState("");
  const [driverNote, setDriverNote] = useState("");
  const [outcome, setOutcome] = useState<"" | keyof typeof R.outcomes>("");
  const [outcomeNote, setOutcomeNote] = useState("");
  const [amount, setAmount] = useState("");
  const [moneyNote, setMoneyNote] = useState("");
  const [resolution, setResolution] = useState("");
  const [note, setNote] = useState("");
  const [transfer, setTransfer] = useState<string | null>(null);
  const [cancelling, setCancelling] = useState<Row | null>(null);
  const [cancelReason, setCancelReason] = useState("");

  if (loadErr && !data) {
    return (
      <PageContainer>
        <Alert>{R.loadFailed}</Alert>
      </PageContainer>
    );
  }
  if (!data) return <LoadingState />;
  const d = data;
  const open = d.status === "open";

  async function post(key: string, path: string, body: unknown, after?: () => void) {
    setBusy(key);
    setErr("");
    try {
      await api(`/api/v1/admin/emergencies/${id}/${path}`, {
        method: "POST",
        body: JSON.stringify(body ?? {}),
      });
      after?.();
      reload();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy("");
    }
  }

  async function cancelStoreOrder() {
    if (!cancelling || !cancelReason.trim()) return;
    setBusy("cancel-order");
    setErr("");
    try {
      await api(`/api/v1/admin/orders/${cancelling.id}/transition`, {
        method: "POST",
        body: JSON.stringify({ to: "cancelled", note: cancelReason.trim() }),
      });
      setCancelling(null);
      setCancelReason("");
      reload();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy("");
    }
  }

  const driverKind = !!d.driver;
  const steps: { key: string; label: string; done: boolean }[] = [
    { key: "ack", label: R.steps.ack, done: !!d.acknowledged_at },
    ...(driverKind
      ? [
          { key: "driver_ok", label: R.steps.driverOk, done: !!d.driver_ok_at },
          ...(d.order
            ? [
                { key: "outcome", label: R.steps.outcome, done: d.outcome !== "" },
                { key: "money", label: R.steps.money, done: !!d.money_request || d.money_skipped },
              ]
            : []),
        ]
      : []),
    { key: "resolve", label: R.steps.done, done: !open },
  ];
  const now = d.next_step;
  const mapLat = d.driver?.lat ?? d.lat;
  const mapLng = d.driver?.lng ?? d.lng;

  return (
    <PageContainer>
      <PageHeader
        icon={IconWarning}
        title={R.kinds[d.kind] ?? d.kind}
        subtitle={fmtDateTime(d.created_at)}
      />
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <Link href="/dashboard/emergencies" className="text-sm underline">
          {R.back}
        </Link>
        <Badge variant={kindVariant(d.kind)}>{R.kinds[d.kind] ?? d.kind}</Badge>
        {d.stage && <Badge>{R.stages[d.stage]}</Badge>}
        {open && !d.acknowledged_at && (
          <Badge variant={d.stale ? "danger" : "warning"}>
            {d.stale ? R.stale.replace("{t}", ago(d.created_at)) : R.unacked}
          </Badge>
        )}
        {d.kind === "platform_halt" && <span className="text-sm text-ink-muted">{R.halt}</span>}
      </div>
      {d.note && <p className="mb-3 text-sm">{d.note}</p>}
      {err && <Alert className="mb-3">{err}</Alert>}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* ── خطواتُ الحلّ ─────────────────────────────────────────── */}
        <Card title={R.steps.title} tone={d.stale ? "danger" : "default"}>
          <ol className="mb-4 flex flex-wrap gap-2">
            {steps.map((s, i) => (
              <li key={s.key} className="flex items-center gap-1 text-sm">
                <Badge variant={s.done ? "success" : now === s.key ? "danger" : "neutral"}>
                  {s.done && <IconCheck size={13} className="me-1 inline" />}
                  {`${i + 1}. ${s.label}`}
                </Badge>
              </li>
            ))}
          </ol>

          {open && now === "ack" && (
            <Button variant="danger" disabled={busy !== ""} onClick={() => void post("ack", "ack", {})}>
              {R.ack}
            </Button>
          )}
          {d.acknowledged_at && (
            <p className="text-xs text-success">{R.ackedBy.replace("{name}", d.acknowledged_by || "—")}</p>
          )}

          {open && now === "driver_ok" && (
            <div className="mt-3 space-y-2">
              <p className="text-sm text-ink-muted">{R.driverOkHint}</p>
              <Input
                id="emg-driver-note"
                placeholder={R.note}
                value={driverNote}
                onChange={(e) => setDriverNote(e.target.value)}
              />
              <Button
                disabled={busy !== ""}
                onClick={() =>
                  void post("driver_ok", "driver-ok", { note: driverNote.trim() }, () => setDriverNote(""))
                }
              >
                {R.driverOkButton}
              </Button>
            </div>
          )}

          {open && now === "outcome" && (
            <div className="mt-3 space-y-2">
              <p className="text-sm text-ink-muted">{R.outcomeHint}</p>
              <div className="flex flex-wrap gap-2">
                {(Object.keys(R.outcomes) as (keyof typeof R.outcomes)[]).map((k) => (
                  <Button
                    key={k}
                    variant={outcome === k ? "primary" : "secondary"}
                    onClick={() => setOutcome(k)}
                  >
                    {R.outcomes[k]}
                  </Button>
                ))}
              </div>
              {outcome && (
                <>
                  <p className="text-xs text-ink-muted">{R.outcomeHelp[outcome]}</p>
                  <Input
                    id="emg-outcome-note"
                    placeholder={R.outcomeReason}
                    value={outcomeNote}
                    onChange={(e) => setOutcomeNote(e.target.value)}
                  />
                  {outcome !== "continue" && <p className="text-xs text-ink-muted">{R.customerTold}</p>}
                  <FormActions
                    busy={busy !== "" || (outcome !== "continue" && !outcomeNote.trim())}
                    saveLabel={R.outcomeSend}
                    tone={outcome === "cancel" ? "danger" : undefined}
                    onSave={() =>
                      void post("outcome", "outcome", { outcome, note: outcomeNote.trim() }, () => {
                        setOutcome("");
                        setOutcomeNote("");
                      })
                    }
                    onCancel={() => {
                      setOutcome("");
                      setOutcomeNote("");
                    }}
                  />
                </>
              )}
            </div>
          )}
          {d.outcome && (
            <p className="mt-2 text-xs text-success">
              {R.steps.outcome}: {R.outcomes[d.outcome]} · {d.outcome_by}
            </p>
          )}

          {open && now === "money" && (
            <div className="mt-3 space-y-2">
              <p className="text-sm text-ink-muted">{R.moneyHint}</p>
              <Input
                id="emg-money-amount"
                type="number"
                inputMode="numeric"
                placeholder={R.moneyAmount}
                value={amount}
                onChange={(e) => setAmount(e.target.value)}
              />
              <Input
                id="emg-money-note"
                placeholder={R.moneyNote}
                value={moneyNote}
                onChange={(e) => setMoneyNote(e.target.value)}
              />
              <div className="flex flex-wrap gap-2">
                <Button
                  disabled={busy !== "" || !(Math.round(Number(amount)) > 0) || !moneyNote.trim()}
                  onClick={() =>
                    void post("money", "money", {
                      amount: Math.round(Number(amount)),
                      note: moneyNote.trim(),
                    })
                  }
                >
                  {R.moneySend}
                </Button>
                <Button
                  variant="secondary"
                  disabled={busy !== ""}
                  onClick={() => void post("money", "money", { skip: true, note: moneyNote.trim() })}
                >
                  {R.moneySkip}
                </Button>
              </div>
            </div>
          )}
          {d.money_request && (
            <p className="mt-2 text-xs">
              {R.moneyRequested.replace("{amount}", fmtMoney(d.money_request.amount ?? d.money_request.suggested_amount))}
              {" · "}
              {R.moneyStatus[d.money_request.status] ?? d.money_request.status}
            </p>
          )}
          {d.money_skipped && <p className="mt-2 text-xs text-ink-muted">{R.moneySkipped}</p>}

          {open && now === "resolve" && (
            <div className="mt-3 space-y-2">
              <Textarea
                id="emg-resolution"
                label={R.resolution}
                rows={2}
                value={resolution}
                onChange={(e) => setResolution(e.target.value)}
              />
              <Button
                disabled={busy !== "" || !resolution.trim()}
                onClick={() => void post("resolve", "resolve", { resolution: resolution.trim() })}
              >
                {R.resolveButton}
              </Button>
            </div>
          )}
          {d.resolved_at && (
            <p className="mt-2 text-xs text-success">
              {R.resolved.replace("{text}", d.resolution || "—")} · {d.resolved_by} ·{" "}
              {fmtDateTime(d.resolved_at)}
            </p>
          )}
        </Card>

        {/* ── السائقُ وموقعُه الحيّ ───────────────────────────────── */}
        {d.driver && (
          <Card title={R.driver}>
            <div className="space-y-2">
              <KV k={R.driver} v={d.driver.name} />
              <KV k={R.cashHeld} v={fmtMoney(d.driver.cash_held)} />
              {d.driver.accident_locked && <p className="text-xs text-danger">{R.accidentLocked}</p>}
              <div className="flex flex-wrap gap-2">
                <Call phone={d.driver.phone} label={R.callDriver} />
              </div>
              <p className="flex items-center gap-1 text-xs text-ink-muted">
                <IconLocation size={13} />
                {d.driver.location_at
                  ? R.lastSeen.replace("{t}", ago(d.driver.location_at))
                  : R.noLocation}
              </p>
              {mapLat != null && mapLng != null && (
                <PickMap lat={mapLat} lng={mapLng} onPick={() => undefined} hideLocate height="h-56" />
              )}
              <h4 className="pt-2 text-sm font-bold">{R.otherOrders}</h4>
              {d.other_orders.length === 0 ? (
                <p className="text-xs text-ink-muted">{R.noOtherOrders}</p>
              ) : (
                <ul className="space-y-1 text-sm">
                  {d.other_orders.map((o) => (
                    <li key={o.id} className="flex flex-wrap items-center justify-between gap-2">
                      <Link href={`/dashboard/orders?id=${o.id}`} dir="ltr" className="underline">
                        #{fmtRef(o.number)}
                      </Link>
                      <span>{statusText(o.status)}</span>
                      <Badge variant={o.picked_up ? "warning" : "neutral"}>
                        {o.picked_up ? R.picked : R.notPicked}
                      </Badge>
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </Card>
        )}

        {/* ── الطلب ───────────────────────────────────────────────── */}
        {d.order && (
          <Card title={R.order}>
            <div className="space-y-2">
              <KV
                k={R.order}
                v={
                  <Link href={`/dashboard/orders?id=${d.order.id}`} dir="ltr" className="underline">
                    #{fmtRef(d.order.number)}
                  </Link>
                }
              />
              <KV k={R.orderStatus} v={statusText(d.order.status)} />
              <KV k={R.goods} v={d.order.picked_up ? R.goodsPicked : R.goodsNotPicked} />
              <KV
                k={R.newDriver}
                v={d.order.new_driver ? d.order.new_driver.name : R.noNewDriver}
              />
              {d.order.customer && <KV k={R.customer} v={d.order.customer.name} />}
              {d.order.store && <KV k={R.store} v={d.order.store.name} />}
              <div className="flex flex-wrap gap-2 pt-1">
                <Call phone={d.order.customer?.phone} label={R.callCustomer} />
                <Call phone={d.order.store?.phone} label={R.callStore} />
                <Call phone={d.order.new_driver?.phone} label={R.newDriver} />
              </div>
            </div>
          </Card>
        )}

        {/* ── المتجرُ المغلقُ طارئاً وطلباتُه ─────────────────────── */}
        {d.merchant && (
          <Card title={R.store}>
            <div className="space-y-2">
              <KV k={R.store} v={d.merchant.name} />
              <p className={`text-xs ${d.merchant.emergency_closed ? "text-danger" : "text-success"}`}>
                {d.merchant.emergency_closed ? R.storeStillClosed : R.storeReopened}
              </p>
              <Call phone={d.merchant.phone} label={R.callStore} />
              <h4 className="pt-2 text-sm font-bold">{R.storeOrders}</h4>
              {d.store_orders.length === 0 ? (
                <p className="text-xs text-ink-muted">{R.noStoreOrders}</p>
              ) : (
                <ul className="space-y-2 text-sm">
                  {d.store_orders.map((o) => (
                    <li key={o.id} className="flex flex-wrap items-center gap-2">
                      <Link href={`/dashboard/orders?id=${o.id}`} dir="ltr" className="underline">
                        #{fmtRef(o.number)}
                      </Link>
                      <span className="text-ink-muted">{statusText(o.status)}</span>
                      <span className="min-w-0 flex-1 truncate">{o.customer}</span>
                      {canIntervene && (
                        <>
                          <Button variant="secondary" onClick={() => setTransfer(o.id)}>
                            {R.transfer}
                          </Button>
                          <Button variant="danger" onClick={() => setCancelling(o)}>
                            {R.cancelOrder}
                          </Button>
                        </>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </div>
          </Card>
        )}

        {/* ── سجلُّ الملاحظات ─────────────────────────────────────── */}
        <Card title={R.notes} icon={IconNote}>
          {d.notes.length === 0 ? (
            <p className="text-xs text-ink-muted">{R.noNotes}</p>
          ) : (
            <ul className="mb-3 space-y-2">
              {d.notes.map((n) => (
                <li key={n.id} className="text-sm">
                  <p className="whitespace-pre-wrap">{n.body}</p>
                  <p className="text-xs text-ink-muted">
                    {n.author || "—"} · <span dir="ltr">{fmtDateTime(n.created_at)}</span>
                  </p>
                </li>
              ))}
            </ul>
          )}
          <div className="space-y-2">
            <Textarea
              id="emg-note"
              rows={2}
              placeholder={R.notePlaceholder}
              value={note}
              onChange={(e) => setNote(e.target.value)}
            />
            <Button
              variant="secondary"
              disabled={busy !== "" || !note.trim()}
              onClick={() => void post("note", "notes", { body: note.trim() }, () => setNote(""))}
            >
              {R.addNote}
            </Button>
          </div>
        </Card>
      </div>

      {transfer && (
        <Modal open onClose={() => setTransfer(null)} title={R.transfer} size="lg">
          <TransferPanel
            orderId={transfer}
            onClose={() => setTransfer(null)}
            onDone={() => {
              setTransfer(null);
              reload();
            }}
          />
        </Modal>
      )}
      {cancelling && (
        <Modal open onClose={() => setCancelling(null)} title={R.cancelOrder}>
          <div className="space-y-3">
            <p className="text-sm" dir="ltr">
              #{fmtRef(cancelling.number)}
            </p>
            <Input
              id="emg-cancel-reason"
              label={R.cancelReason}
              value={cancelReason}
              onChange={(e) => setCancelReason(e.target.value)}
            />
            <FormActions
              busy={busy !== "" || !cancelReason.trim()}
              saveLabel={R.cancelOrder}
              tone="danger"
              onSave={() => void cancelStoreOrder()}
              onCancel={() => setCancelling(null)}
            />
          </div>
        </Modal>
      )}
    </PageContainer>
  );
}
