"use client";

/**
 * **الأهداف والمكافآت** — من وصل لأيّ مرحلة، وكم انصرف، ومن ينتظر قراراً.
 *
 * # مكافأةُ الهدف تنصرف لحالها
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٩ ثمّ ٢٠٢٦-٠٨-٣١: ثلاثُ مراحل تُدفع آليّاً عند
 * بلوغها.) **كان العنوانُ يقول «والمكافأةُ بيدك»** فيفهم الموظّفُ أنّه يدفع
 * بيده فيدفع مرّتين. هذه الصفحةُ لا تدفع مكافأةَ الهدف — تُريها.
 *
 * # واليدويُّ بموافقة (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
 *
 *  ١ · المكافأةُ والعقوبةُ اليدويّة طلبٌ يوافق عليه موظّفٌ آخر — لا مالَ قبلها
 *  ٣ و٤ · طلبٌ مسترجَعٌ أو متجرٌ حُذف/تجريبيّ تحت مرحلةٍ قُبضت ⇒ تنبيهٌ هنا
 *  ٥ · اختيارُ شهرٍ وتصدير
 *  ٦ · اليدويُّ «تقدير» دائماً — لا «عن الهدف»
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Chips,
  Alert,
  PageContainer,
  Button,
  ButtonLink,
  Badge,
  Modal,
  Drawer,
  Confirm,
  Input,
  Textarea,
  EmptyState,
  LoadingState,
  ReloadState,
  IconStar,
  IconDriver,
  IconUser,
  IconTarget,
  IconWallet,
  IconWarning,
  IconSettings,
  IconArrowOut,
  IconSearch,
  FormActions,
  useLiveRefresh,
} from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall, useCanOpen } from "@/lib/policy";
import { PageHero, type HeroTab } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const P = m.admin.incentives;
const S = m.shared.incentives;

interface Standing {
  user_id: string;
  name: string;
  phone: string;
  done: number;
  target: number;
  reached: boolean;
  level: number;
  levels: number;
  auto_paid: number;
  manual_paid: number;
  penalized: number;
  balance: number;
  /** **متاجرُ فتحها المندوبُ ولم تكتمل شروطُها** — لا تُعدّ بعد (٢٠٢٦-١٠-١٣). */
  pending?: number;
}

interface Level {
  n: number;
  target: number;
  reward: number;
}

interface IncAlert {
  kind: "count_dropped" | "grant_failed";
  incentive_id?: string;
  failure_id?: string;
  user_id: string;
  name: string;
  month: string;
  level: number;
  target_count: number;
  current: number;
  amount: number;
  refunded: number;
  fake_stores: string[];
  attempts?: number;
}

interface PageData {
  month: string;
  standings: Standing[];
  summary: { reached_per_level: number[]; auto_paid: number; manual_paid: number; penalties: number };
  levels: Level[];
  alerts: IncAlert[];
  cap: number;
}

interface IncRequest {
  id: string;
  user_id: string;
  user_name: string;
  kind: "reward" | "penalty";
  amount: number;
  note: string;
  source: string;
  proposed_by: string;
  proposer_name: string;
  created_at: string;
  /** **الدورُ الذي اقتُرح له** — ويُعرض في تبويبه وحدَه (قرارُ المالك ٢٠٢٦-١٠-١٥). */
  role?: Role | "";
}

interface Entry {
  id: string;
  kind: "reward" | "penalty";
  amount: number;
  reason: string;
  for_target: boolean;
  by: string;
  created_at: string;
}

type Role = "driver" | "sales";
type Kind = "reward" | "penalty";

const fill = (s: string, v: Record<string, string | number>) =>
  Object.entries(v).reduce((acc, [k, x]) => acc.split(`{${k}}`).join(String(x)), s);

/** الشهرُ الجاري بتوقيت دمشق — «2026-10». */
function thisMonth(): string {
  const d = new Date(Date.now() + 3 * 3600 * 1000);
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, "0")}`;
}

const kindLabel = (k: Kind) => (k === "reward" ? S.reward : S.penalty);

export default function IncentivesPage() {
  const [role, setRole] = useState<Role>("driver");
  const [month, setMonth] = useState(thisMonth());
  const [query, setQuery] = useState("");
  const { user: me } = useAuth();
  const canCall = useCanCall();
  const canGrant = canCall("POST", "/users/{id}/incentive");
  const canDecide = canCall("POST", "/incentive-requests/{id}/approve");
  const canSettings = useCanOpen().settings;

  /** **والفشلُ ليس فراغاً** — «لا أحدَ بلغ» على قراءةٍ فشلت قرارُ مالٍ على زور. */
  const [data, setData] = useState<PageData | null | "failed">(null);
  const [requests, setRequests] = useState<IncRequest[]>([]);
  const [pageError, setPageError] = useState("");
  const [notice, setNotice] = useState("");

  const load = useCallback(() => {
    setPageError("");
    api<PageData>(`/api/v1/admin/incentives/${role}?month=${month}`)
      .then((r) => setData(r))
      .catch((err) => {
        setData("failed");
        setPageError(errorText(err));
      });
    api<{ requests: IncRequest[] }>(`/api/v1/admin/incentive-requests?status=pending`)
      .then((r) => setRequests(r.requests ?? []))
      .catch((err) => setPageError(errorText(err)));
  }, [role, month]);

  useEffect(() => {
    setData(null);
    load();
  }, [load]);
  useLiveRefresh(["incentive", "wallet"], load);

  // ── نافذةُ الطلب ─────────────────────────────────────────────────
  const [granting, setGranting] = useState<Standing | null>(null);
  const [kind, setKind] = useState<Kind>("reward");
  const [amount, setAmount] = useState("");
  const [reason, setReason] = useState("");
  const [formError, setFormError] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);

  const cap = data && data !== "failed" ? data.cap : 0;

  function openGrant(x: Standing) {
    setKind("reward");
    setAmount("");
    setReason("");
    setFormError("");
    setNotice("");
    setGranting(x);
  }

  /** **التحقّقُ قبل الإرسال** — كان الحقلُ يقبل كسوراً وسالباً ويردّه الخادمُ برسالةٍ عامّة. */
  function validate(): string {
    const n = Number(amount);
    if (!/^\d+$/.test(amount.trim()) || !Number.isSafeInteger(n) || n <= 0) return P.errAmount;
    if (cap > 0 && n > cap) return fill(P.errOverCap, { cap: fmtMoney(cap) });
    if (!reason.trim()) return P.errReason;
    if (kind === "penalty" && granting && n > granting.balance)
      return fill(P.errBalance, { b: fmtMoney(granting.balance) });
    return "";
  }

  function askConfirm() {
    const err = validate();
    setFormError(err);
    if (!err) setConfirming(true);
  }

  async function submit() {
    const who = granting;
    if (!who) return;
    setConfirming(false);
    setBusy(true);
    setFormError("");
    try {
      await api(`/api/v1/admin/users/${who.user_id}/incentive`, {
        method: "POST",
        // **والدورُ مع الطلب** — من له الدوران يُعدّ في التبويب الذي اقتُرح منه وحدَه.
        body: JSON.stringify({ kind, amount: Number(amount), reason: reason.trim(), role }),
      });
      setGranting(null);
      setNotice(P.requestSent);
      load();
    } catch (e) {
      // **والخطأُ داخلَ النافذة** — كان يُكتب خلفها فلا يُرى.
      setFormError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  // ── الطلباتُ المعلّقة ─────────────────────────────────────────────
  const [approving, setApproving] = useState<IncRequest | null>(null);
  const [rejecting, setRejecting] = useState<IncRequest | null>(null);
  const [rejectNote, setRejectNote] = useState("");

  async function decide(r: IncRequest, ok: boolean) {
    setApproving(null);
    setRejecting(null);
    setBusy(true);
    try {
      await api(`/api/v1/admin/incentive-requests/${r.id}/${ok ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify({ note: ok ? "" : rejectNote.trim() }),
      });
      setRejectNote("");
      setNotice(P.decided);
      load();
    } catch (err) {
      setPageError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  // ── التنبيهات ─────────────────────────────────────────────────────
  const [deciding, setDeciding] = useState<{ a: IncAlert; d: "keep" | "clawback" } | null>(null);
  const [alertNote, setAlertNote] = useState("");

  async function decideAlert() {
    const x = deciding;
    if (!x?.a.incentive_id) return;
    setDeciding(null);
    setBusy(true);
    try {
      await api(`/api/v1/admin/incentive-alerts/${x.a.incentive_id}/decide`, {
        method: "POST",
        body: JSON.stringify({ decision: x.d, note: alertNote.trim() }),
      });
      setAlertNote("");
      setNotice(P.decided);
      load();
    } catch (err) {
      setPageError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  async function retry(a: IncAlert) {
    setBusy(true);
    try {
      await api(`/api/v1/admin/incentive-failures/${a.failure_id}/retry`, { method: "POST", body: "{}" });
      setNotice(P.decided);
      load();
    } catch (err) {
      setPageError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  // ── التصدير ───────────────────────────────────────────────────────
  async function exportCsv() {
    try {
      const res = await apiFile(`/api/v1/admin/incentives/${role}/export?month=${month}`);
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `incentives-${role}-${month}.csv`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (err) {
      setPageError(errorText(err));
    }
  }

  // ── كشفُ الشخص ────────────────────────────────────────────────────
  const [ledgerOf, setLedgerOf] = useState<Standing | null>(null);
  const [ledger, setLedger] = useState<Entry[] | null | "failed">(null);
  useEffect(() => {
    if (!ledgerOf) return;
    setLedger(null);
    api<{ entries: Entry[] }>(`/api/v1/admin/users/${ledgerOf.user_id}/incentives?limit=200`)
      .then((r) => setLedger(r.entries ?? []))
      .catch(() => setLedger("failed"));
  }, [ledgerOf]);

  const TABS: HeroTab<Role>[] = [
    { key: "driver", label: P.tabDrivers, icon: IconDriver },
    { key: "sales", label: P.tabSales, icon: IconUser },
  ];
  const unit = role === "driver" ? P.unitDriver : P.unitSales;

  const rows = useMemo(() => {
    if (!data || data === "failed") return [];
    const q = query.trim();
    if (!q) return data.standings;
    return data.standings.filter((x) => x.name.includes(q) || x.phone.includes(q));
  }, [data, query]);

  const roleRequests = useMemo(() => {
    if (!data || data === "failed") return [];
    const ids = new Set(data.standings.map((x) => x.user_id));
    return requests.filter((r) => ids.has(r.user_id) && (!r.role || r.role === role));
  }, [data, requests, role]);

  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) — رأسٌ موحّدٌ بتبويبَي الدور، أرقامٌ بمربّعات،
  // وكرتٌ لكلّ شخصٍ بدل الجدول.
  return (
    <PageContainer>
      <PageHero<Role>
        icon={IconStar}
        title={P.title}
        subtitle={P.subtitle}
        actions={
          <Button variant="secondary" onClick={() => void exportCsv()} className="flex items-center gap-1.5">
            <IconArrowOut size={16} />
            {P.export}
          </Button>
        }
        tabs={TABS}
        value={role}
        onChange={setRole}
      />

      {/* ── الفلاتر ببطاقةٍ واحدة: الشهر والبحث ── */}
      <section className="surface flex flex-wrap items-end gap-3 p-3">
        <Input
          id="incentives-month"
          label={P.month}
          type="month"
          value={month}
          max={thisMonth()}
          onChange={(e) => e.target.value && setMonth(e.target.value)}
          wrapperClassName="w-44"
        />
        <Input
          id="incentives-search"
          label={P.search}
          icon={<IconSearch size={16} />}
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          wrapperClassName="min-w-0 flex-1"
        />
      </section>

      {pageError && <Alert>{pageError}</Alert>}
      {notice && <Alert tone="success">{notice}</Alert>}

      {data === null ? (
        <LoadingState variant="stats" />
      ) : data === "failed" ? (
        <ReloadState onRetry={load} />
      ) : (
        <>
          {/* ── الأرقام: كلُّ مرحلة · الآليّ · اليدويّ · العقوبات ── */}
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-3 2xl:grid-cols-6">
            {data.summary.reached_per_level.map((n, i) => (
              <Tile
                key={i}
                icon={IconTarget}
                label={fill(P.cardStage, { n: i + 1 })}
                value={fmtNum(n)}
                tone={n > 0 ? "accent" : "muted"}
              />
            ))}
            <Tile
              icon={IconWallet}
              tone={data.summary.auto_paid > 0 ? "success" : "muted"}
              label={P.cardAuto}
              value={fmtMoney(data.summary.auto_paid)}
            />
            <Tile
              icon={IconStar}
              tone={data.summary.manual_paid > 0 ? "success" : "muted"}
              label={P.cardManual}
              value={fmtMoney(data.summary.manual_paid)}
            />
            <Tile
              icon={IconWarning}
              tone={data.summary.penalties > 0 ? "danger" : "muted"}
              label={P.cardPenalties}
              value={fmtMoney(data.summary.penalties)}
            />
          </div>

          {/* ── مراحلُ الهدف — الوعدُ كما هو، وتعديلُه من الإعدادات ── */}
          <section className="surface p-4">
            <div className="mb-3 flex flex-wrap items-center justify-between gap-2">
              <div className="min-w-0">
                <h2 className="heading-card">{P.stagesTitle}</h2>
                <p className="text-xs text-ink-muted">{P.stagesHint}</p>
              </div>
              {/* **ولمن يفتح الإعدادات وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٥): الماليّةُ لا تراها. */}
              {canSettings && (
                <ButtonLink href="/dashboard/settings" variant="secondary">
                  <IconSettings size={14} />
                  {P.editInSettings}
                </ButtonLink>
              )}
            </div>
            {data.levels.length === 0 ? (
              <p className="text-sm text-ink-muted">{P.noStages}</p>
            ) : (
              <ol className="grid grid-cols-1 gap-2 sm:grid-cols-3">
                {data.levels.map((l) => (
                  <li key={l.n} className="flex items-center gap-3 rounded-control bg-field px-3 py-2.5">
                    <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-accent text-sm font-bold text-on-bright tabular-nums">
                      {fmtNum(l.n)}
                    </span>
                    <span className="min-w-0">
                      <span className="block text-2xs text-ink-muted">{fill(P.stageN, { n: fmtNum(l.n) })}</span>
                      <span className="block font-bold tabular-nums">
                        {fmtNum(l.target)} {unit}
                      </span>
                      <span className="block text-xs text-success">
                        {fill(P.stageReward, { reward: fmtMoney(l.reward) })}
                      </span>
                    </span>
                  </li>
                ))}
              </ol>
            )}
          </section>

          {/* ── التنبيهات — الماليّةُ تقرّر ── */}
          {data.alerts.length > 0 && (
            <section className="surface p-4">
              <h2 className="heading-card mb-3 flex items-center gap-2 text-danger">
                <IconWarning size={18} />
                {P.alertsTitle}
              </h2>
              <ul className="space-y-2">
                {data.alerts.map((a) => (
                  <li
                    key={a.incentive_id ?? a.failure_id}
                    className="flex flex-col gap-2 rounded-control bg-field p-3 text-sm sm:flex-row sm:items-center"
                  >
                    <span className="min-w-0 flex-1">
                      {a.kind === "grant_failed"
                        ? fill(P.alertFailed, { name: a.name, month: a.month, attempts: fmtNum(a.attempts ?? 1) })
                        : fill(P.alertDropped, {
                            name: a.name,
                            level: a.level,
                            month: a.month,
                            target: fmtNum(a.target_count),
                            current: fmtNum(a.current),
                          })}
                      {a.kind === "count_dropped" && (
                        <span className="block text-xs text-ink-muted">
                          {a.refunded > 0
                            ? fill(P.alertRefunded, { n: fmtNum(a.refunded) })
                            : a.fake_stores.length > 0
                              ? fill(P.alertFake, { names: a.fake_stores.join(P.listSep) })
                              : P.alertRemoved}
                          {" · "}
                          {fmtMoney(a.amount)}
                        </span>
                      )}
                    </span>
                    {canDecide && a.kind === "count_dropped" && (
                      <span className="grid shrink-0 grid-cols-2 gap-2">
                        <Button variant="secondary" disabled={busy} onClick={() => setDeciding({ a, d: "keep" })}>
                          {P.keep}
                        </Button>
                        <Button variant="danger" disabled={busy} onClick={() => setDeciding({ a, d: "clawback" })}>
                          {P.clawback}
                        </Button>
                      </span>
                    )}
                    {canDecide && a.kind === "grant_failed" && (
                      <Button disabled={busy} onClick={() => void retry(a)}>
                        {P.retry}
                      </Button>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          )}

          {/* ── الطلباتُ المعلّقة — يوافق عليها غيرُ من اقترحها ── */}
          {roleRequests.length > 0 && (
            <section className="surface p-4">
              <h2 className="heading-card mb-3 flex items-center gap-2">
                <IconWallet size={18} />
                {P.pendingTitle}
                <Badge variant="warning">{fmtNum(roleRequests.length)}</Badge>
              </h2>
              <ul className="grid grid-cols-1 gap-2 lg:grid-cols-2">
                {roleRequests.map((r) => (
                  <li key={r.id} className="flex flex-col gap-2 rounded-control bg-field p-3 text-sm">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge variant={r.kind === "penalty" ? "danger" : "success"}>{kindLabel(r.kind)}</Badge>
                      <span className="font-medium">{r.user_name}</span>
                      <span
                        className={`ms-auto font-bold tabular-nums ${r.kind === "penalty" ? "text-danger" : "text-success"}`}
                        dir="ltr"
                      >
                        {fmtMoney(r.amount)}
                      </span>
                    </div>
                    {r.note && <p className="border-s-2 border-line ps-2">{r.note}</p>}
                    <p className="text-2xs text-ink-muted">
                      {P.proposedBy}: {r.proposer_name} · <span dir="ltr">{fmtDateTime(r.created_at)}</span>
                    </p>
                    {canDecide && (
                      <div className="grid grid-cols-2 gap-2">
                        {me?.id !== r.proposed_by && (
                          <Button className="w-full" disabled={busy} onClick={() => setApproving(r)}>
                            {P.approve}
                          </Button>
                        )}
                        <Button
                          variant="secondary"
                          className={`w-full ${me?.id === r.proposed_by ? "col-span-2" : ""}`}
                          disabled={busy}
                          onClick={() => setRejecting(r)}
                        >
                          {P.reject}
                        </Button>
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </section>
          )}

          {/* ── الأشخاص — كرتٌ لكلّ واحد ── */}
          {data.standings.length === 0 || rows.length === 0 ? (
            <section className="surface flex flex-col items-center gap-3 p-8 text-center">
              <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-field text-ink-muted">
                {data.standings.length === 0 ? <IconUser size={28} /> : <IconSearch size={28} />}
              </span>
              <h2 className="heading-section">{data.standings.length === 0 ? P.empty : P.noResults}</h2>
              <p className="max-w-md text-sm text-ink-muted">
                {data.standings.length === 0 ? P.emptyHint : P.noResultsHint}
              </p>
            </section>
          ) : (
            <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
              {rows.map((x) => (
                <PersonCard
                  key={x.user_id}
                  x={x}
                  levels={data.levels}
                  unit={unit}
                  canGrant={canGrant}
                  onLedger={() => setLedgerOf(x)}
                  onGrant={() => openGrant(x)}
                />
              ))}
            </ul>
          )}
        </>
      )}

      {/* ── نافذةُ الطلب ── */}
      <Modal
        open={granting !== null}
        onClose={() => setGranting(null)}
        title={`${P.grantTitle} — ${granting?.name || granting?.phone || ""}`}
      >
        <div className="space-y-3">
          {formError && <Alert>{formError}</Alert>}
          <p className="text-sm font-medium">{P.kind}</p>
          <Chips
            items={[
              { id: "reward" as Kind, label: S.reward },
              { id: "penalty" as Kind, label: S.penalty },
            ]}
            value={kind}
            onChange={(k) => {
              setKind(k);
              setFormError("");
            }}
          />
          {kind === "penalty" && granting && (
            <p className="text-sm text-ink-muted">
              {P.balance}:{" "}
              <span className="font-bold text-ink" dir="ltr">
                {fmtMoney(granting.balance)}
              </span>
            </p>
          )}
          <Input
            id="incentive-amount"
            label={P.amount}
            type="number"
            inputMode="numeric"
            min={1}
            step={1}
            max={cap > 0 ? cap : undefined}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          {cap > 0 && <p className="text-xs text-ink-muted">{fill(P.amountHint, { cap: fmtMoney(cap) })}</p>}
          {/* **والسببُ إلزاميّ ويصل صاحبَه في الإشعار** — فمن عوقب يعرف بماذا. */}
          <Textarea
            id="incentive-reason"
            label={P.reason}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
          <FormActions
            onSave={askConfirm}
            onCancel={() => setGranting(null)}
            busy={busy}
            saveLabel={P.sendForApproval}
          />
        </div>
      </Modal>

      <Confirm
        open={confirming && granting !== null}
        tone={kind === "penalty" ? "danger" : "primary"}
        title={P.confirmTitle}
        body={
          granting &&
          fill(P.confirmBody, {
            kind: kindLabel(kind),
            amount: fmtMoney(Number(amount) || 0),
            name: granting.name || granting.phone,
          })
        }
        confirmLabel={P.sendForApproval}
        busy={busy}
        onConfirm={() => void submit()}
        onCancel={() => setConfirming(false)}
      />

      <Confirm
        open={approving !== null}
        tone="primary"
        title={P.approveTitle}
        body={
          approving &&
          fill(P.approveBody, {
            kind: kindLabel(approving.kind),
            amount: fmtMoney(approving.amount),
            name: approving.user_name,
            reason: approving.note,
          })
        }
        confirmLabel={P.approve}
        busy={busy}
        onConfirm={() => approving && void decide(approving, true)}
        onCancel={() => setApproving(null)}
      />
      <Confirm
        open={rejecting !== null}
        title={P.rejectTitle}
        body={
          <Input id="inc-reject-note" label={P.rejectNote} value={rejectNote} onChange={(e) => setRejectNote(e.target.value)} />
        }
        confirmLabel={P.reject}
        busy={busy}
        onConfirm={() => rejecting && void decide(rejecting, false)}
        onCancel={() => setRejecting(null)}
      />
      <Confirm
        open={deciding !== null}
        tone={deciding?.d === "clawback" ? "danger" : "primary"}
        title={P.decideTitle}
        body={
          <div className="space-y-2">
            {deciding?.d === "clawback" && <p className="text-sm">{P.clawbackHint}</p>}
            <Input id="inc-alert-note" label={P.decideNote} value={alertNote} onChange={(e) => setAlertNote(e.target.value)} />
          </div>
        }
        confirmLabel={deciding?.d === "clawback" ? P.clawback : P.keep}
        busy={busy || !alertNote.trim()}
        onConfirm={() => void decideAlert()}
        onCancel={() => setDeciding(null)}
      />

      {/* ── كشفُ الشخص — مَن ومتى ولماذا، آليّاً أو يدويّاً ── */}
      <Drawer
        open={ledgerOf !== null}
        onClose={() => setLedgerOf(null)}
        title={`${P.ledgerTitle} — ${ledgerOf?.name || ledgerOf?.phone || ""}`}
      >
        {ledger === null ? (
          <LoadingState variant="text" />
        ) : ledger === "failed" ? (
          <ReloadState onRetry={() => ledgerOf && setLedgerOf({ ...ledgerOf })} />
        ) : ledger.length === 0 ? (
          <EmptyState icon={IconWallet} title={P.ledgerEmpty} />
        ) : (
          <ul className="divide-y divide-line-soft">
            {ledger.map((e) => (
              <li key={e.id} className="space-y-1 py-2 text-sm">
                <div className="flex flex-wrap items-center gap-2">
                  <Badge variant={e.kind === "penalty" ? "danger" : "success"}>{kindLabel(e.kind)}</Badge>
                  {e.kind === "reward" && (
                    <Badge variant={e.for_target ? "primary" : "accent"}>
                      {e.for_target ? P.targetBonus : P.appreciation}
                    </Badge>
                  )}
                  <span className="font-bold" dir="ltr">
                    {fmtMoney(e.amount)}
                  </span>
                  <span className="ms-auto text-xs text-ink-muted" dir="ltr">
                    {fmtDateTime(e.created_at)}
                  </span>
                </div>
                <p>{e.reason}</p>
                <p className="text-xs text-ink-muted">{e.by || P.system}</p>
              </li>
            ))}
          </ul>
        )}
      </Drawer>
    </PageContainer>
  );
}

/**
 * **كرتُ الشخص** — تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) بدل صفّ الجدول: الاسمُ والمرحلة،
 * والإنجازُ بشريطٍ وكم باقي للمرحلة الجاية، والآليُّ واليدويُّ والعقوباتُ بثلاث مربّعات، والزرّان.
 */
function PersonCard({
  x,
  levels,
  unit,
  canGrant,
  onLedger,
  onGrant,
}: {
  x: Standing;
  levels: Level[];
  unit: string;
  canGrant: boolean;
  onLedger: () => void;
  onGrant: () => void;
}) {
  const name = x.name || x.phone;
  const pct = x.target > 0 ? Math.min(100, Math.round((x.done / x.target) * 100)) : 0;
  // **المرحلةُ الجاية** — للعرض وحدَه؛ المكافأةُ يحسبها الخادم.
  const next = levels.find((l) => l.n > x.level);
  const remaining = next ? Math.max(0, next.target - x.done) : 0;
  return (
    <li className="surface flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <span
          className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full font-bold ${
            x.level > 0 ? "bg-success-solid text-on-solid" : "bg-field text-ink"
          }`}
          aria-hidden
        >
          {name.trim().charAt(0)}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold">{name}</span>
          {x.name && (
            <span className="block text-2xs text-ink-muted" dir="ltr">
              {x.phone}
            </span>
          )}
        </span>
        {x.levels > 0 && (
          <Badge variant={x.level > 0 ? "success" : "neutral"} className="shrink-0">
            {x.level > 0 ? fill(P.stageReached, { n: x.level, total: x.levels }) : P.stageNone}
          </Badge>
        )}
      </div>

      <div className="rounded-control bg-field px-3 py-2.5">
        {x.target > 0 ? (
          <>
            <div className="flex items-baseline justify-between gap-2">
              <span className="text-2xs text-ink-muted">{P.progress}</span>
              {/* **والوحدةُ بوحدة الدور** — عملاءُ للمندوب لا متاجر (٢٠٢٦-١٠-١٥). */}
              <span className="font-bold tabular-nums">
                {fill(P.progressOf, { done: fmtNum(x.done), target: fmtNum(x.target), unit })}
              </span>
            </div>
            <div className="mt-2 h-2 overflow-hidden rounded-full bg-surface" aria-hidden>
              <div
                className={`h-full rounded-full ${pct >= 100 ? "bg-success-solid" : "bg-accent"}`}
                style={{ width: `${pct}%` }}
              />
            </div>
            {x.levels > 0 && (
              <p className="mt-1.5 text-2xs text-ink-muted">
                {next
                  ? remaining > 0
                    ? fill(P.nextStage, { n: fmtNum(remaining), unit, stage: fmtNum(next.n) })
                    : ""
                  : P.allStages}
              </p>
            )}
          </>
        ) : (
          <span className="text-xs text-ink-muted">{P.noTarget}</span>
        )}
        {(x.pending ?? 0) > 0 && (
          <p className="mt-1 text-2xs text-warning">{fill(P.pendingStores, { n: x.pending ?? 0 })}</p>
        )}
      </div>

      <div className="flex gap-2 text-center">
        <span className="min-w-0 flex-1 rounded-control border border-line-soft px-2 py-2">
          <span className="block text-2xs text-ink-muted">{P.auto}</span>
          <span className="block font-bold tabular-nums text-success" dir="ltr">
            {fmtMoney(x.auto_paid)}
          </span>
        </span>
        <span className="min-w-0 flex-1 rounded-control border border-line-soft px-2 py-2">
          <span className="block text-2xs text-ink-muted">{P.manual}</span>
          <span className="block font-bold tabular-nums text-success" dir="ltr">
            {fmtMoney(x.manual_paid)}
          </span>
        </span>
        <span className="min-w-0 flex-1 rounded-control border border-line-soft px-2 py-2">
          <span className="block text-2xs text-ink-muted">{P.penalized}</span>
          <span className="block font-bold tabular-nums text-danger" dir="ltr">
            {fmtMoney(x.penalized)}
          </span>
        </span>
      </div>

      <div className={`mt-auto grid gap-2 border-t border-line-soft pt-3 ${canGrant ? "grid-cols-2" : "grid-cols-1"}`}>
        <Button variant="secondary" className="w-full" onClick={onLedger}>
          {P.openLedger}
        </Button>
        {canGrant && (
          <Button className="w-full" onClick={onGrant}>
            {P.grant}
          </Button>
        )}
      </div>
    </li>
  );
}
