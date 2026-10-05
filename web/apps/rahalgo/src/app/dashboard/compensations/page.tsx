"use client";

/**
 * **«التعويضات» — صفحةٌ واحدةٌ وطريقُ موافقةٍ واحدٌ لكلّ تعويض** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 *
 * كانت «تعويضاتٌ بانتظار الموافقة» تعرض تعويضَ السائق المعلَّقَ وحدَه. **والآن كلُّ
 * تعويضٍ في المنصّة هنا**: السائق (المحرّكُ أو المكتبُ يقترح)، والمتجرُ عن بضاعةٍ
 * رُدّت (الماليّةُ تكتب المبلغ من بطاقة الطلب)، وصاحبُ الشكوى (الدعمُ يقترح).
 *
 * - **تبويبات**: بانتظار القرار · تمت الموافقة · مرفوضة · الكل — فالقرارُ يُقرأ بعد شهر.
 * - **بطاقات**: ما ينتظر (عدداً ومجموعاً) · ما قُبل ورُفض هذا الشهر · ما تأخّر.
 * - **فلاتر في الرابط**: النوع · الجهة المذنبة · المستفيد · التاريخ.
 * - **الأزرارُ بمعرّف طلب التعويض لا برقم الطلب** — كان الزرُّ يدفع لسائقٍ غيرِ المعروض.
 * - **ولا يوافق أحدٌ على ما اقترحه، وفوق السقف مديرُ المنصّة وحدَه** — المحرّكُ يحرس
 *   ذلك، والتأكيدُ بكلمة المرور يلتقطه العميلُ المركزيّ (`StepUpGate`).
 */

import { useEffect, useMemo, useState } from "react";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtRef,
  fmtDateTime,
  fmtMoney,
  fmtSpan,
  errorText,
} from "@rahalgo/i18n";
import { WalletRequestsList } from "@/components/admin/WalletRequests";
import {
  FormSection,
  Alert,
  Badge,
  Button,
  Input,
  Select,
  Tabs,
  Textarea,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  DataView,
  Pagination,
  StatGrid,
  StatCard,
  ViewToggle,
  useViewMode,
  type DataColumn,
  useLiveData,
  IconWallet,
  IconDriver,
  IconOrder,
  IconStatus,
  IconDate,
  IconWarning,
  Money,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall, useCanOpen } from "@/lib/policy";
import { OpenLink } from "@/components/admin/OpenLink";

const m = getMessages(defaultLocale);
const C = m.admin.compensations;
const FAULTS = C.faults as Record<string, string>;
const KINDS = C.kinds as Record<string, string>;
const ROLES = C.roles as Record<string, string>;
const STATUSES = C.statuses as Record<string, string>;

type Tab = "pending" | "approved" | "rejected" | "all";

interface Compensation {
  id: string;
  kind: string;
  order_id: string;
  order_number: number;
  order_status: string;
  ticket_id: string;
  ticket_number: number;
  driver_id: string;
  driver_name: string;
  beneficiary_role: string;
  fault: string;
  fail_reason: string;
  reason_label: string;
  expected_fault: string;
  fault_mismatch: boolean;
  suggested_amount: number;
  status: string;
  amount: number | null;
  note: string;
  proposed_by: string | null;
  proposed_by_name: string;
  decided_by_name: string;
  decided_at: string | null;
  decision_note: string;
  self_approved: boolean;
  created_at: string;
  overdue: boolean;
  cap: number;
}

interface Summary {
  pending_count: number;
  pending_sum: number;
  approved_month_count: number;
  approved_month_sum: number;
  rejected_month_count: number;
  overdue_count: number;
  overdue_hours: number;
}

interface Filters {
  status: Tab;
  kind: string;
  fault: string;
  person: string;
  from: string;
  to: string;
}

const EMPTY: Filters = { status: "pending", kind: "", fault: "", person: "", from: "", to: "" };

function toQuery(f: Filters, page: number): string {
  const q = new URLSearchParams();
  (Object.keys(f) as (keyof Filters)[]).forEach((k) => {
    if (f[k]) q.set(k, f[k]);
  });
  if (page > 1) q.set("page", String(page));
  return q.toString();
}

export default function CompensationsPage() {
  const { can } = useAuth();
  const canDecide = can("finance.manage");
  // **والحسابُ والطلبُ والشكوى لمن تُفتح له** (قرارُ المالك ٢٠٢٦-١٠-٠٥): الماليّةُ
  // لا ترى الطلباتِ ولا الحساباتِ ولا الشكاوى — فتُقرأ نصّاً لا رابطاً.
  const open = useCanOpen();
  const canTickets = useCanCall()("GET", "/tickets");
  const [view, setView] = useViewMode("compensations");
  const [page, setPage] = useState(1);
  const [filters, setFilters] = useState<Filters>(EMPTY);
  const [ready, setReady] = useState(false);
  const [deciding, setDeciding] = useState<{ c: Compensation; approve: boolean } | null>(null);

  // **الفلاترُ في الرابط** — تُقرأ عند الفتح وتُكتب عند التغيير.
  useEffect(() => {
    try {
      const q = new URLSearchParams(window.location.search);
      const st = q.get("status");
      setFilters({
        status: st === "approved" || st === "rejected" || st === "all" ? st : "pending",
        kind: q.get("kind") ?? "",
        fault: q.get("fault") ?? "",
        person: q.get("person") ?? "",
        from: q.get("from") ?? "",
        to: q.get("to") ?? "",
      });
      setPage(Math.max(1, Number(q.get("page")) || 1));
    } catch {
      /* الرابطُ راحةٌ لا شرط */
    }
    setReady(true);
  }, []);

  const qs = useMemo(() => toQuery(filters, page), [filters, page]);
  useEffect(() => {
    if (!ready) return;
    try {
      window.history.replaceState(null, "", `${window.location.pathname}${qs ? `?${qs}` : ""}`);
    } catch {
      /* الرابطُ راحةٌ لا شرط */
    }
  }, [qs, ready]);

  const { data, loading, error, reload } = useLiveData<{
    compensations: Compensation[];
    total: number;
    page: number;
    per_page: number;
    summary: Summary;
  }>(
    () => api(`/api/v1/admin/compensations?${qs}`),
    ["order", "wallet"],
    [qs],
  );

  function set<K extends keyof Filters>(k: K, v: Filters[K]) {
    setFilters((f) => ({ ...f, [k]: v }));
    setPage(1);
  }

  if (loading && !data) return <LoadingState />;

  const rows = data?.compensations ?? [];
  const sum = data?.summary;
  const pendingTab = filters.status === "pending";

  const columns: DataColumn<Compensation>[] = [
    {
      id: "kind",
      header: C.colKind,
      icon: <IconStatus />,
      cell: (c) => <Badge variant={c.kind === "driver" ? "primary" : c.kind === "complaint" ? "violet" : "info"}>{KINDS[c.kind] ?? c.kind}</Badge>,
    },
    {
      id: "beneficiary",
      header: C.colBeneficiary,
      icon: <IconDriver />,
      primary: true,
      cell: (c) => (
        <span>
          <OpenLink allowed={open.user} href={`/dashboard/users/${c.driver_id}`} className="text-primary-dark hover:underline">
            {c.driver_name}
          </OpenLink>{" "}
          <span className="text-xs text-ink-muted">{ROLES[c.beneficiary_role] ?? ""}</span>
        </span>
      ),
    },
    {
      id: "ref",
      header: C.colRef,
      icon: <IconOrder />,
      primary: true,
      cell: (c) => (
        <span className="flex flex-col">
          {c.order_number > 0 && (
            <OpenLink
              allowed={open.order}
              href={`/dashboard/orders?q=${c.order_number}`}
              className="font-bold text-primary-dark hover:underline"
              dir="ltr"
            >
              #{fmtRef(c.order_number)}
            </OpenLink>
          )}
          {c.ticket_number > 0 && (
            <OpenLink allowed={canTickets} href="/dashboard/tickets" className="text-xs text-primary-dark hover:underline">
              {C.ticketRef.replace("{n}", fmtNum(c.ticket_number))}
            </OpenLink>
          )}
        </span>
      ),
    },
    {
      id: "reason",
      header: C.reason,
      icon: <IconStatus />,
      cell: (c) => (
        <span className="flex flex-col">
          <span>{c.reason_label}</span>
          {c.fault_mismatch && (
            <span className="text-xs text-warning">
              <IconWarning className="inline" size={12} /> {FAULTS[c.expected_fault] ?? ""}
            </span>
          )}
        </span>
      ),
    },
    {
      id: "fault",
      header: C.fault,
      icon: <IconWarning />,
      cell: (c) => (c.fault ? FAULTS[c.fault] ?? c.fault : "—"),
    },
    {
      id: "amount",
      header: pendingTab ? C.suggested : C.amount,
      icon: <IconWallet />,
      primary: true,
      cell: (c) => <Money value={c.amount ?? c.suggested_amount} className="font-bold" />,
    },
    {
      id: "proposed",
      header: C.colProposedBy,
      icon: <IconDriver />,
      cell: (c) => <span className="text-sm">{c.proposed_by ? c.proposed_by_name || "—" : C.engine}</span>,
    },
    pendingTab
      ? {
          id: "waiting",
          header: C.colWaiting,
          icon: <IconDate />,
          cell: (c) => (
            <span className={`text-xs ${c.overdue ? "font-bold text-danger" : "text-ink-muted"}`}>
              {fmtSpan((Date.now() - new Date(c.created_at).getTime()) / 1000)}
              {c.overdue && (
                <>
                  {" "}
                  <Badge variant="danger">{C.overdueBadge}</Badge>
                </>
              )}
            </span>
          ),
        }
      : {
          id: "decision",
          header: C.colDecision,
          icon: <IconStatus />,
          cell: (c) => (
            <span className="flex flex-col text-xs">
              <Badge variant={c.status === "approved" ? "success" : c.status === "rejected" ? "danger" : "warning"}>
                {STATUSES[c.status] ?? c.status}
              </Badge>
              {c.decided_by_name && (
                <span className="text-ink-muted">
                  {C.decidedBy}: {c.decided_by_name}
                  {c.decided_at && (
                    <span dir="ltr"> · {fmtDateTime(c.decided_at)}</span>
                  )}
                </span>
              )}
              {c.decision_note && <span className="text-ink-muted">{c.decision_note}</span>}
              {c.self_approved && <span className="text-warning">{C.selfApproved}</span>}
            </span>
          ),
        },
    {
      id: "created",
      header: C.created,
      icon: <IconDate />,
      cell: (c) => (
        <span dir="ltr" className="text-xs text-ink-muted">
          {fmtDateTime(c.created_at)}
        </span>
      ),
    },
  ];

  const tabs = [
    { key: "pending" as const, label: C.tabs.pending, count: sum?.pending_count },
    { key: "approved" as const, label: C.tabs.approved },
    { key: "rejected" as const, label: C.tabs.rejected },
    { key: "all" as const, label: C.tabs.all },
  ];
  const filtered = filters.kind || filters.fault || filters.person || filters.from || filters.to;

  return (
    <PageContainer width="full">
      <PageHeader
        icon={IconWallet}
        title={C.titleAll}
        subtitle={C.hintAll}
        actions={
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
        }
      />

      {error && (
        <Alert className="mb-3">
          <span>{C.loadFailed}</span>{" "}
          <button type="button" onClick={() => reload()} className="font-bold underline">
            {m.common.retry}
          </button>
        </Alert>
      )}

      {sum && (
        <StatGrid>
          <StatCard
            label={C.cardPending}
            value={fmtNum(sum.pending_count)}
            sub={C.cardPendingSub.replace("{amount}", fmtMoney(sum.pending_sum))}
            icon={IconWallet}
            tone={sum.overdue_count > 0 ? "danger" : sum.pending_count > 0 ? "accent" : "muted"}
            onClick={() => set("status", "pending")}
          />
          <StatCard
            label={C.cardApproved}
            value={fmtMoney(sum.approved_month_sum)}
            sub={C.cardApprovedSub.replace("{n}", fmtNum(sum.approved_month_count))}
            icon={IconStatus}
            tone="success"
            onClick={() => set("status", "approved")}
          />
          <StatCard
            label={C.cardRejected}
            value={fmtNum(sum.rejected_month_count)}
            icon={IconStatus}
            tone="muted"
            onClick={() => set("status", "rejected")}
          />
          <StatCard
            label={C.cardOverdue.replace("{h}", fmtNum(sum.overdue_hours))}
            value={fmtNum(sum.overdue_count)}
            icon={IconWarning}
            tone={sum.overdue_count > 0 ? "danger" : "muted"}
          />
        </StatGrid>
      )}

      <Tabs items={tabs} value={filters.status} onChange={(k) => set("status", k)} className="mt-4" />

      <div className="mt-3 flex flex-wrap items-end gap-2">
        <Select value={filters.kind} onChange={(e) => set("kind", e.target.value)} aria-label={C.colKind}>
          <option value="">{C.filterKind}</option>
          {Object.keys(KINDS).map((k) => (
            <option key={k} value={k}>
              {KINDS[k]}
            </option>
          ))}
        </Select>
        <Select value={filters.fault} onChange={(e) => set("fault", e.target.value)} aria-label={C.fault}>
          <option value="">{C.filterFault}</option>
          {Object.keys(FAULTS).map((k) => (
            <option key={k} value={k}>
              {FAULTS[k]}
            </option>
          ))}
        </Select>
        <Input
          value={filters.person}
          onChange={(e) => set("person", e.target.value)}
          placeholder={C.filterPerson}
          aria-label={C.filterPerson}
        />
        <Input
          type="date"
          dir="ltr"
          value={filters.from}
          onChange={(e) => set("from", e.target.value)}
          aria-label={C.filterFrom}
          title={C.filterFrom}
        />
        <Input
          type="date"
          dir="ltr"
          value={filters.to}
          onChange={(e) => set("to", e.target.value)}
          aria-label={C.filterTo}
          title={C.filterTo}
        />
        {filtered && (
          <Button variant="secondary" onClick={() => setFilters((f) => ({ ...EMPTY, status: f.status }))}>
            {C.clearFilters}
          </Button>
        )}
      </div>

      <div className="mt-3">
        {data && rows.length === 0 ? (
          <EmptyState
            icon={IconStatus}
            title={pendingTab && !filtered ? C.empty : C.emptyHistory}
            tone={pendingTab ? "success" : "muted"}
          />
        ) : (
          rows.length > 0 && (
            <DataView
              items={rows}
              getKey={(c) => c.id}
              columns={columns}
              view={view}
              empty={C.empty}
              actions={
                canDecide
                  ? (c) =>
                      c.status === "pending" ? (
                        <>
                          <Button onClick={() => setDeciding({ c, approve: true })}>{C.approve}</Button>
                          <Button variant="danger" onClick={() => setDeciding({ c, approve: false })}>
                            {C.reject}
                          </Button>
                        </>
                      ) : null
                  : undefined
              }
            />
          )
        )}
      </div>

      {data && data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}

      {/* **وطلباتُ حركة المحفظة اليدويّة هنا أيضاً** (قرارُ المالك ٢٠٢٦-١٠-٠٤). */}
      <div className="mt-6">
        <FormSection title={m.admin.acc.requestsTitle} icon={<IconWallet />}>
          <p className="mb-2 text-xs text-ink-muted">{m.admin.acc.requestsHint}</p>
          <WalletRequestsList />
        </FormSection>
      </div>

      {deciding && (
        <DecideModal
          item={deciding.c}
          approve={deciding.approve}
          onClose={() => setDeciding(null)}
          onDone={() => {
            setDeciding(null);
            reload();
          }}
        />
      )}
    </PageContainer>
  );
}

/**
 * **نافذةُ القرار** — بمعرّف طلب التعويض نفسِه، لا برقم الطلب.
 *
 * ملخّصُ الحالة · رابطُ سجلّ الطلب · تحذيرٌ إن خالف الذنبُ السبب · تحذيرٌ فوق
 * السقف · **وملاحظةٌ داخليّةٌ لا يراها المستفيد** (السائقُ يرى «تسوية من الإدارة»).
 */
function DecideModal({
  item,
  approve,
  onClose,
  onDone,
}: {
  item: Compensation;
  approve: boolean;
  onClose: () => void;
  onDone: () => void;
}) {
  const [amount, setAmount] = useState(String(item.suggested_amount > 0 ? item.suggested_amount : ""));
  const canOrder = useCanOpen().order;
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const value = Number(amount);
  const amountOk = Number.isInteger(value) && value > 0;
  const aboveCap = item.cap > 0 && amountOk && value > item.cap;

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    const text = note.trim();
    if (text === "") return;
    if (approve && !amountOk) {
      setError(C.amountInvalid);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/compensations/${item.id}/${approve ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify(approve ? { amount: value, note: text } : { note: text }),
      });
      onDone();
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={approve ? C.approveTitleAll : C.rejectTitle}>
      <form onSubmit={submit} className="space-y-4">
        <div className="rounded-control border border-line bg-field px-3 py-2 text-sm">
          <p className="font-bold">
            {item.driver_name}{" "}
            <span className="font-normal text-ink-muted">· {KINDS[item.kind] ?? ""}</span>
          </p>
          <p className="text-ink-muted">
            {item.reason_label}
            {item.fault ? ` · ${FAULTS[item.fault] ?? item.fault}` : ""}
          </p>
          <p className="text-ink-muted">
            {C.suggested}: <Money value={item.suggested_amount} />
          </p>
          {item.note && (
            <p className="text-ink-muted">
              {C.proposalNote}: {item.note}
            </p>
          )}
          {item.order_number > 0 && (
            <OpenLink
              allowed={canOrder}
              href={`/dashboard/orders?q=${item.order_number}`}
              className="text-xs text-primary-dark hover:underline"
              dir="ltr"
            >
              #{fmtRef(item.order_number)}
            </OpenLink>
          )}
        </div>
        {item.fault_mismatch && (
          <Alert tone="warning">
            {C.mismatch
              .replace("{reason}", item.reason_label)
              .replace("{expected}", FAULTS[item.expected_fault] ?? "")
              .replace("{fault}", FAULTS[item.fault] ?? "")}
          </Alert>
        )}
        {approve && (
          <>
            <Input
              id="comp-amount"
              label={`${C.amount} (${m.common.currency})`}
              type="number"
              inputMode="numeric"
              min={1}
              step={1}
              dir="ltr"
              required
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
              error={amount !== "" && !amountOk ? C.amountInvalid : undefined}
            />
            <p className="text-xs text-ink-muted">
              {C.amountHint.replace("{n}", fmtNum(item.suggested_amount))}
            </p>
            {aboveCap && <Alert tone="warning">{C.aboveCap.replace("{cap}", fmtMoney(item.cap))}</Alert>}
            {item.kind === "driver" && item.fault === "merchant" && (
              <Alert tone="info">{C.merchantClaimHint}</Alert>
            )}
          </>
        )}
        <Textarea
          id="comp-note"
          label={`${approve ? C.approveNote : C.rejectNote} — ${C.internalNote}`}
          required
          maxLength={300}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={approve ? C.approveNotePlaceholderAll : C.rejectNotePlaceholder}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions
          submit
          onCancel={onClose}
          busy={busy}
          saveLabel={busy ? m.common.loading : approve ? C.approve : C.reject}
        />
      </form>
    </Modal>
  );
}
