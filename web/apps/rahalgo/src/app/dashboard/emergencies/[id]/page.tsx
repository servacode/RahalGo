"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صفحةُ الطارئ** — قراراتُ المالك ٢٠٢٦-١٠-٠٤ (غرفةُ الطوارئ)
 * ══════════════════════════════════════════════════════════════════════
 *
 * الطلبُ وحالُه والسائقُ الجديد · البضاعةُ (استُلمت أم لا) والمالُ مع السائق وطلباتُه
 * الأخرى · اتّصالٌ بالسائق والزبون والمتجر · خريطةٌ بموقع السائق **الحيّ**.
 *
 * **وخطواتُ الحلّ بترتيبها**: استلمتها ← السائقُ بخير؟ ← مصيرُ الطلب ← التعويض ← تمّ.
 * **وسجلُّ ملاحظاتٍ** يُضاف إليه ولا يُعدَّل. **ومتجرٌ أُغلق طارئاً** تُعرض طلباتُه
 * المفتوحة، **والمكتبُ يحوّلها أو يلغيها.**
 *
 * # والتصميمُ الاحترافيّ (٢٠٢٦-١٠-٢٧)
 *
 * - رأسٌ موحّد: النوعُ والاسمُ والمدّةُ ورقمُ الطلب، وشاراتُ الحال فيه.
 * - **كرتُ خطواتٍ عموديّ**: المقطوعةُ خضراءُ بمن قطعها ومتى، **والحاليّةُ مفتوحةٌ بزرٍّ
 *   بعرضٍ كامل**، والجاية رماديّة. ومصائرُ الطلب الأربعة كروتٌ بشرحها.
 * - **التعويضُ يقول ما سيقع فعلاً** — المالكُ والمديرُ ينفُذ تعويضُهما فوراً (قرارُ
 *   ٢٠٢٦-١٠-٢٧، `autoApproveCompensationTx`)، وغيرُهما طلبٌ للماليّة. **ويُقرأ `applied`.**
 * - **توقّفُ المنصّة لا يُسكَّر والمنصّةُ واقفة** (`halt_active`، والمحرّكُ يردّه أيضاً).
 * - **والخريطةُ تقول ما النقطة** (موقعُ السائق أم مكانُ الضغطة) **وتلحق الدبّوس.**
 */

import { useEffect, useState, type ReactNode } from "react";
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
  fmtNum,
  fmtTime,
  errorText,
} from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  FormActions,
  Input,
  LoadingState,
  Modal,
  PageContainer,
  ReloadState,
  Textarea,
  useLiveData,
  IconWarning,
  IconPhone,
  IconCheck,
  IconLocation,
  IconNote,
  IconDriver,
  IconOrder,
  IconStore,
  IconSteps,
  IconRoute,
  IconSwap,
  IconBlock,
  IconSuccess,
  IconPrev,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PageHero } from "@/components/admin/PageHero";
import { kindVariant, kindTint } from "@/components/admin/emergency/kinds";
import { TransferPanel } from "@/components/admin/orders/TransferPanel";

const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
});

const m = getMessages(defaultLocale);
const R = m.admin.emergencyRoom;
const statusText = (s: string) => (m.orders.status as Record<string, string>)[s] ?? s;
const paymentText = (s: string) => (m.orders.payment as Record<string, string>)[s] ?? s;
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
type Outcome = keyof typeof R.outcomes;
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
  outcome: "" | Outcome;
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
  halt_active: boolean;
  money_auto_applies: boolean;
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

const OUTCOME_ICON: Record<Outcome, typeof IconRoute> = {
  redispatch: IconRoute,
  return_store: IconStore,
  cancel: IconBlock,
  continue: IconSuccess,
};

/** **زرُّ اتّصالٍ بعرضٍ كامل** — ولا يُرسم بلا رقم. */
function Call({ phone, label }: { phone: string | undefined; label: string }) {
  if (!phone) return null;
  return (
    <ButtonLink href={`tel:${phone}`} variant="secondary" className="w-full">
      <IconPhone size={15} />
      {label}
    </ButtonLink>
  );
}

function KV({ k, v }: { k: string; v: ReactNode }) {
  return (
    <div className="flex items-center justify-between gap-3 text-sm">
      <span className="text-ink-muted">{k}</span>
      <span className="text-end font-medium">{v}</span>
    </div>
  );
}

/** **كرتٌ بعنوانٍ وأيقونةٍ في دائرة** — نمطُ الأقسام المكتملة. */
function Section({ icon: Icon, title, extra, children }: {
  icon: typeof IconDriver;
  title: string;
  extra?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="surface flex flex-col gap-3 p-4">
      <div className="flex items-center gap-3">
        <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-field text-ink">
          <Icon size={18} />
        </span>
        <h2 className="heading-card min-w-0 flex-1 truncate">{title}</h2>
        {extra}
      </div>
      {children}
    </section>
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
  const [outcome, setOutcome] = useState<"" | Outcome>("");
  const [outcomeNote, setOutcomeNote] = useState("");
  const [amount, setAmount] = useState("");
  const [moneyNote, setMoneyNote] = useState("");
  const [moneyResult, setMoneyResult] = useState<"" | "applied" | "sent">("");
  const [resolution, setResolution] = useState("");
  const [note, setNote] = useState("");
  const [transfer, setTransfer] = useState<string | null>(null);
  const [cancelling, setCancelling] = useState<Row | null>(null);
  const [cancelReason, setCancelReason] = useState("");

  if (loadErr && !data) {
    return (
      <PageContainer>
        <section className="surface">
          <ReloadState onRetry={reload} label={R.loadFailed} />
        </section>
      </PageContainer>
    );
  }
  if (!data) return <LoadingState />;
  const d = data;
  const open = d.status === "open";
  // **تعويضُ المالك والمدير ينفُذ فوراً** — علَمٌ من المحرّك (`approval.AutoApplies`)، لا حكمٌ بالدور هنا.
  const autoPays = d.money_auto_applies;

  async function post<T = unknown>(key: string, path: string, body: unknown, after?: (r: T) => void) {
    setBusy(key);
    setErr("");
    try {
      const r = await api<T>(`/api/v1/admin/emergencies/${id}/${path}`, {
        method: "POST",
        body: JSON.stringify(body ?? {}),
      });
      after?.(r);
      reload();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy("");
    }
  }

  /** **وسببُ الإلغاء يُمسح مع كلّ إغلاقٍ للنافذة** — كان يبقى فيُرسَل لطلبٍ آخر. */
  function closeCancel() {
    setCancelling(null);
    setCancelReason("");
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
      closeCancel();
      reload();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy("");
    }
  }

  const byAt = (name: string, at: string | null) =>
    R.stepBy.replace("{name}", name || "—").replace("{at}", at ? fmtDateTime(at) : "");

  const driverKind = !!d.driver;
  const halt = d.kind === "platform_halt";
  const now = d.next_step;
  const amountN = Math.round(Number(amount));

  // ── الخطوات: لكلٍّ ما قُطع منها ومن قطعها، وما يُفعل فيها الآن ──────────
  const steps: { key: string; label: string; done: boolean; doneText?: ReactNode; body?: ReactNode }[] = [
    {
      key: "ack",
      label: R.steps.ack,
      done: !!d.acknowledged_at,
      doneText: byAt(d.acknowledged_by, d.acknowledged_at),
      body: (
        <>
          <p className="text-sm text-ink-muted">{R.ackHint}</p>
          <Button variant="danger" className="w-full" disabled={busy !== ""} onClick={() => void post("ack", "ack", {})}>
            {R.ack}
          </Button>
        </>
      ),
    },
  ];
  if (driverKind) {
    steps.push({
      key: "driver_ok",
      label: R.steps.driverOk,
      done: !!d.driver_ok_at,
      doneText: byAt(d.driver_ok_by, d.driver_ok_at),
      body: (
        <>
          <p className="text-sm text-ink-muted">
            {R.driverOkHint}
            {d.kind === "accident" && ` ${R.driverOkAccidentHint}`}
          </p>
          <Call phone={d.driver?.phone} label={R.callDriver} />
          <Input
            id="emg-driver-note"
            label={R.note}
            value={driverNote}
            onChange={(e) => setDriverNote(e.target.value)}
          />
          <Button
            className="w-full"
            disabled={busy !== ""}
            onClick={() => void post("driver_ok", "driver-ok", { note: driverNote.trim() }, () => setDriverNote(""))}
          >
            {R.driverOkButton}
          </Button>
        </>
      ),
    });
    if (d.order) {
      steps.push({
        key: "outcome",
        label: R.steps.outcome,
        done: d.outcome !== "",
        doneText: d.outcome ? (
          <>
            <b>{R.outcomes[d.outcome]}</b> · {byAt(d.outcome_by, d.outcome_at)}
          </>
        ) : undefined,
        body: (
          <>
            <p className="text-sm text-ink-muted">{R.outcomeHint}</p>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {(Object.keys(R.outcomes) as Outcome[]).map((k) => {
                const Icon = OUTCOME_ICON[k];
                const on = outcome === k;
                return (
                  <button
                    key={k}
                    type="button"
                    aria-pressed={on}
                    onClick={() => setOutcome(k)}
                    className={`flex items-start gap-2 rounded-card border p-3 text-start transition-colors hover:bg-row-hover ${
                      on ? (k === "cancel" ? "border-danger" : "border-accent") : "border-line-soft"
                    }`}
                  >
                    <span
                      className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${
                        on ? (k === "cancel" ? "bg-danger-solid text-on-solid" : "bg-accent text-on-bright") : "bg-field text-ink"
                      }`}
                    >
                      <Icon size={16} />
                    </span>
                    <span className="min-w-0">
                      <span className="block text-sm font-bold">{R.outcomes[k]}</span>
                      <span className="block text-2xs text-ink-muted">{R.outcomeHelp[k]}</span>
                    </span>
                  </button>
                );
              })}
            </div>
            {outcome && (
              <>
                <Input
                  id="emg-outcome-note"
                  label={R.outcomeReason}
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
          </>
        ),
      });
      steps.push({
        key: "money",
        label: R.steps.money,
        done: !!d.money_request || d.money_skipped,
        doneText: d.money_request ? (
          <>
            <b>{R.moneyRequested.replace("{amount}", fmtMoney(d.money_request.amount ?? d.money_request.suggested_amount))}</b>
            {" · "}
            <Badge
              variant={
                d.money_request.status === "approved"
                  ? "success"
                  : d.money_request.status === "rejected"
                    ? "danger"
                    : "warning"
              }
            >
              {R.moneyStatus[d.money_request.status] ?? d.money_request.status}
            </Badge>
          </>
        ) : d.money_skipped ? (
          R.moneySkipped
        ) : undefined,
        body: (
          <>
            <p className="text-sm text-ink-muted">{autoPays ? R.moneyHintAuto : R.moneyHintStaff}</p>
            <Input
              id="emg-money-amount"
              type="number"
              inputMode="numeric"
              label={R.moneyAmount}
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
            {amountN > 0 && (
              <p className="text-xs text-ink-muted">{R.moneyPreview.replace("{amount}", fmtMoney(amountN))}</p>
            )}
            <Input
              id="emg-money-note"
              label={R.moneyNote}
              value={moneyNote}
              onChange={(e) => setMoneyNote(e.target.value)}
            />
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <Button
                className="w-full"
                disabled={busy !== "" || !(amountN > 0) || !moneyNote.trim()}
                onClick={() =>
                  void post<{ applied?: boolean }>(
                    "money",
                    "money",
                    { amount: amountN, note: moneyNote.trim() },
                    (r) => {
                      setMoneyResult(r?.applied ? "applied" : "sent");
                      setAmount("");
                      setMoneyNote("");
                    },
                  )
                }
              >
                {autoPays ? R.moneySendAuto : R.moneySendStaff}
              </Button>
              <Button
                variant="secondary"
                className="w-full"
                disabled={busy !== ""}
                onClick={() =>
                  void post("money", "money", { skip: true, note: moneyNote.trim() }, () => setMoneyNote(""))
                }
              >
                {R.moneySkip}
              </Button>
            </div>
          </>
        ),
      });
    }
  }
  steps.push({
    key: "resolve",
    label: R.steps.done,
    done: !open,
    doneText: d.resolved_at ? (
      <>
        <b>{R.resolved.replace("{text}", d.resolution || "—")}</b> · {byAt(d.resolved_by, d.resolved_at)}
      </>
    ) : undefined,
    body:
      halt && d.halt_active ? (
        // **والمنصّةُ واقفة: لا «تمّ»** — يُغلق وحدَه بعودتها، والمفتاحُ في الإعدادات.
        <>
          <p className="text-sm text-ink-muted">{R.haltActive}</p>
          <ButtonLink href="/dashboard/settings?topic=launch" variant="secondary" className="w-full">
            {R.haltSettings}
          </ButtonLink>
        </>
      ) : (
        <>
          {halt && <p className="text-sm text-ink-muted">{R.haltEnded}</p>}
          <Textarea
            id="emg-resolution"
            label={R.resolution}
            rows={2}
            value={resolution}
            onChange={(e) => setResolution(e.target.value)}
          />
          <p className="text-2xs text-ink-muted">{R.resolutionHint}</p>
          <Button
            className="w-full"
            disabled={busy !== "" || !resolution.trim()}
            onClick={() => void post("resolve", "resolve", { resolution: resolution.trim() })}
          >
            {R.resolveButton}
          </Button>
        </>
      ),
  });

  // ── الخريطة: موقعُ السائق الحيّ، وإلّا مكانُ الضغطة — **وتقول أيَّهما.** ──
  const live = d.driver?.lat != null && d.driver?.lng != null;
  const mapLat = live ? d.driver!.lat : d.lat;
  const mapLng = live ? d.driver!.lng : d.lng;
  const hasMap = mapLat != null && mapLng != null;
  const mapCaption = live
    ? d.driver?.location_at
      ? R.mapDriver.replace("{t}", ago(d.driver.location_at))
      : R.mapPress
    : hasMap
      ? R.mapPress
      : R.noLocation;

  const who = d.driver?.name || d.merchant?.name || "";
  const title = who ? `${R.kinds[d.kind] ?? d.kind} — ${who}` : (R.kinds[d.kind] ?? d.kind);

  return (
    <PageContainer>
      <PageHero
        icon={IconWarning}
        title={title}
        titleExtra={
          <>
            {!open && <Badge variant="success">{R.statusResolved}</Badge>}
            {open && !d.acknowledged_at && (
              <Badge variant={d.stale ? "danger" : "warning"}>
                {d.stale ? R.stale.replace("{t}", ago(d.created_at)) : R.unacked}
              </Badge>
            )}
          </>
        }
        subtitle={
          <>
            {R.since.replace("{t}", ago(d.created_at))}
            {d.order && (
              <>
                {m.common.listSeparator}
                {R.order} <span dir="ltr">#{fmtRef(d.order.number)}</span>
              </>
            )}
            {d.stage && `${m.common.listSeparator}${R.stages[d.stage]}`}
          </>
        }
        actions={
          <ButtonLink href="/dashboard/emergencies" variant="secondary">
            <IconPrev size={15} className="rtl:rotate-180" />
            {R.back}
          </ButtonLink>
        }
      >
        <div className="flex flex-wrap items-center gap-2">
          <span className={`inline-flex h-7 w-7 items-center justify-center rounded-full ${kindTint(d.kind)}`}>
            <IconWarning size={14} />
          </span>
          <Badge variant={kindVariant(d.kind)}>{R.kinds[d.kind] ?? d.kind}</Badge>
          <span className="text-2xs text-ink-muted" dir="ltr">
            {fmtDateTime(d.created_at)}
          </span>
        </div>
        {d.note && (
          <p className="mt-3 rounded-control bg-field px-3 py-2 text-sm">
            <span className="block text-2xs text-ink-muted">{R.noteTitle}</span>
            {d.note}
          </p>
        )}
      </PageHero>

      {err && <Alert className="mb-3">{err}</Alert>}
      {moneyResult && (
        <Alert tone="success" className="mb-3">
          {moneyResult === "applied" ? R.moneyApplied : R.moneySent}
        </Alert>
      )}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        {/* ── خطواتُ الحلّ — عموديّة، والحاليّةُ مفتوحة ───────────────── */}
        <section className={`surface p-4 ${d.stale ? "border border-danger" : ""}`}>
          <div className="mb-3 flex items-center gap-3">
            <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-field text-ink">
              <IconSteps size={18} />
            </span>
            <h2 className="heading-card">{R.steps.title}</h2>
          </div>
          <ol className="space-y-0">
            {steps.map((s, i) => {
              const current = open && !s.done && now === s.key;
              const last = i === steps.length - 1;
              return (
                <li key={s.key} className="flex gap-3">
                  <div className="flex flex-col items-center">
                    <span
                      className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-sm font-bold tabular-nums ${
                        s.done
                          ? "bg-success-solid text-on-solid"
                          : current
                            ? "bg-danger-solid text-on-solid"
                            : "bg-field text-ink-muted"
                      }`}
                    >
                      {s.done ? <IconCheck size={15} /> : fmtNum(i + 1)}
                    </span>
                    {!last && <span className="my-1 w-0.5 flex-1 rounded-full bg-line-soft" />}
                  </div>
                  <div className={`min-w-0 flex-1 ${last ? "" : "pb-4"}`}>
                    <p className={`pt-1 text-sm font-bold ${s.done || current ? "" : "text-ink-muted"}`}>{s.label}</p>
                    {s.done && s.doneText && <p className="text-xs text-ink-muted">{s.doneText}</p>}
                    {!s.done && !current && open && <p className="text-2xs text-ink-muted">{R.stepLater}</p>}
                    {current && <div className="mt-2 space-y-2">{s.body}</div>}
                  </div>
                </li>
              );
            })}
          </ol>
        </section>

        <div className="space-y-4">
          {/* ── السائقُ وموقعُه الحيّ ───────────────────────────────── */}
          {d.driver && (
            <Section icon={IconDriver} title={d.driver.name || d.driver.phone}>
              <div className="rounded-control bg-field px-3 py-2.5">
                <span className="block text-2xs text-ink-muted">{R.cashHeld}</span>
                <span className="figure block">{fmtMoney(d.driver.cash_held)}</span>
              </div>
              <KV
                k={R.shift}
                v={
                  <Badge variant={d.driver.on_shift ? "success" : "neutral"}>
                    {d.driver.on_shift ? R.onShift : R.offShift}
                  </Badge>
                }
              />
              {d.driver.accident_locked && <Alert>{R.accidentLocked}</Alert>}
              <Call phone={d.driver.phone} label={R.callDriver} />
              <p className="flex items-center gap-1 text-xs text-ink-muted">
                <IconLocation size={13} />
                {mapCaption}
              </p>
              {hasMap && (
                <>
                  {/* **والمفتاحُ بالموضع** — فتلحق الكاميرا السائقَ إذا تحرّك. */}
                  <PickMap
                    key={`${mapLat!.toFixed(4)},${mapLng!.toFixed(4)}`}
                    lat={mapLat}
                    lng={mapLng}
                    onPick={() => undefined}
                    hideLocate
                    height="h-56"
                  />
                  <ButtonLink
                    href={`https://www.google.com/maps?q=${mapLat},${mapLng}`}
                    target="_blank"
                    rel="noreferrer"
                    variant="secondary"
                    className="w-full"
                  >
                    <IconLocation size={15} />
                    {R.openInMaps}
                  </ButtonLink>
                </>
              )}
              <h3 className="pt-1 text-sm font-bold">{R.otherOrders}</h3>
              {d.other_orders.length === 0 ? (
                <p className="text-xs text-ink-muted">{R.noOtherOrders}</p>
              ) : (
                <ul className="space-y-2">
                  {d.other_orders.map((o) => (
                    <li key={o.id} className="flex flex-wrap items-center gap-2 rounded-control border border-line-soft p-2 text-sm">
                      <Link href={`/dashboard/orders?id=${o.id}`} dir="ltr" className="font-bold underline">
                        #{fmtRef(o.number)}
                      </Link>
                      <span className="min-w-0 flex-1 truncate text-ink-muted">{statusText(o.status)}</span>
                      <Badge variant={o.picked_up ? "warning" : "neutral"}>
                        {o.picked_up ? R.picked : R.notPicked}
                      </Badge>
                    </li>
                  ))}
                </ul>
              )}
            </Section>
          )}

          {/* ── الطلب ───────────────────────────────────────────────── */}
          {d.order && (
            <Section
              icon={IconOrder}
              title={R.order}
              extra={
                <span className="text-sm font-bold" dir="ltr">
                  #{fmtRef(d.order.number)}
                </span>
              }
            >
              <div className="grid grid-cols-2 gap-2">
                <div className="rounded-control bg-field px-3 py-2.5">
                  <span className="block text-2xs text-ink-muted">{R.orderTotal}</span>
                  <span className="figure block">{fmtMoney(d.order.total)}</span>
                </div>
                <div className="rounded-control bg-field px-3 py-2.5">
                  <span className="block text-2xs text-ink-muted">{R.cashDue}</span>
                  <span className="figure block">{fmtMoney(d.order.cash_due)}</span>
                </div>
              </div>
              <KV k={R.orderStatus} v={statusText(d.order.status)} />
              <KV
                k={R.goods}
                v={
                  <Badge variant={d.order.picked_up ? "warning" : "neutral"}>
                    {d.order.picked_up ? R.goodsPicked : R.goodsNotPicked}
                  </Badge>
                }
              />
              {d.order.picked_up_at && (
                <KV k={R.pickedAt} v={<span dir="ltr">{fmtTime(d.order.picked_up_at)}</span>} />
              )}
              {d.order.payment_method && <KV k={R.payment} v={paymentText(d.order.payment_method)} />}
              {d.order.address_text && <KV k={R.address} v={d.order.address_text} />}
              {d.order.customer && <KV k={R.customer} v={d.order.customer.name} />}
              {d.order.store && <KV k={R.store} v={d.order.store.name} />}
              <KV k={R.newDriver} v={d.order.new_driver ? d.order.new_driver.name : R.noNewDriver} />
              <div className="grid grid-cols-1 gap-2 border-t border-line-soft pt-3 sm:grid-cols-2">
                <ButtonLink href={`/dashboard/orders?id=${d.order.id}`} className="w-full sm:col-span-2">
                  <IconOrder size={15} />
                  {R.openOrder}
                </ButtonLink>
                <Call phone={d.order.customer?.phone} label={R.callCustomer} />
                <Call phone={d.order.store?.phone} label={R.callStore} />
                <Call phone={d.order.new_driver?.phone} label={R.callNewDriver} />
              </div>
            </Section>
          )}

          {/* ── المتجرُ المغلقُ طارئاً وطلباتُه ─────────────────────── */}
          {d.merchant && (
            <Section
              icon={IconStore}
              title={d.merchant.name}
              extra={
                <Badge variant={d.merchant.emergency_closed ? "danger" : "success"}>
                  {d.merchant.emergency_closed ? R.storeStillClosed : R.storeReopened}
                </Badge>
              }
            >
              <Call phone={d.merchant.phone} label={R.callStore} />
              <h3 className="pt-1 text-sm font-bold">{R.storeOrders}</h3>
              {d.store_orders.length === 0 ? (
                <p className="text-xs text-ink-muted">{R.noStoreOrders}</p>
              ) : (
                <ul className="space-y-2">
                  {d.store_orders.map((o) => (
                    <li key={o.id} className="space-y-2 rounded-card border border-line-soft p-3">
                      <div className="flex flex-wrap items-center gap-2 text-sm">
                        <Link href={`/dashboard/orders?id=${o.id}`} dir="ltr" className="font-bold underline">
                          #{fmtRef(o.number)}
                        </Link>
                        <span className="min-w-0 flex-1 truncate">{o.customer}</span>
                        <Badge>{statusText(o.status)}</Badge>
                      </div>
                      {canIntervene && (
                        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                          <Button
                            variant="secondary"
                            className="w-full"
                            disabled={busy !== ""}
                            onClick={() => setTransfer(o.id)}
                          >
                            <IconSwap size={15} />
                            {R.transfer}
                          </Button>
                          <Button
                            variant="danger"
                            className="w-full"
                            disabled={busy !== ""}
                            onClick={() => {
                              setCancelReason("");
                              setCancelling(o);
                            }}
                          >
                            {R.cancelOrder}
                          </Button>
                        </div>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </Section>
          )}

          {/* ── سجلُّ الملاحظات — خطٌّ زمنيّ ────────────────────────── */}
          <Section icon={IconNote} title={R.notes}>
            {d.notes.length === 0 ? (
              <p className="text-xs text-ink-muted">{R.noNotes}</p>
            ) : (
              <ol className="space-y-3 border-s-2 border-line-soft ps-4">
                {d.notes.map((n) => (
                  <li key={n.id} className="relative">
                    <span className="absolute -start-[1.4rem] top-1.5 h-2.5 w-2.5 rounded-full bg-accent" />
                    <p className="text-2xs text-ink-muted">
                      <b className="text-ink">{n.author || "—"}</b> · <span dir="ltr">{fmtDateTime(n.created_at)}</span>
                    </p>
                    <p className="whitespace-pre-wrap text-sm">{n.body}</p>
                  </li>
                ))}
              </ol>
            )}
            <div className="space-y-2 border-t border-line-soft pt-3">
              <Textarea
                id="emg-note"
                rows={2}
                placeholder={R.notePlaceholder}
                value={note}
                onChange={(e) => setNote(e.target.value)}
              />
              <Button
                variant="secondary"
                className="w-full"
                disabled={busy !== "" || !note.trim()}
                onClick={() => void post("note", "notes", { body: note.trim() }, () => setNote(""))}
              >
                {R.addNote}
              </Button>
            </div>
          </Section>
        </div>
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
        <Modal open onClose={closeCancel} title={R.cancelOrder}>
          <div className="space-y-3">
            <p className="text-sm">
              <span dir="ltr" className="font-bold">
                #{fmtRef(cancelling.number)}
              </span>
              {cancelling.customer && `${m.common.listSeparator}${cancelling.customer}`}
            </p>
            <p className="text-sm text-ink-muted">{R.cancelHint}</p>
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
              onCancel={closeCancel}
            />
          </div>
        </Modal>
      )}
    </PageContainer>
  );
}
