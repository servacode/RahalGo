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
 * - **حركةُ المالك أو المدير تنفذ فوراً، وطلبُ الموظّف ينتظر** (قرارُ المالك ٢٠٢٦-١٠-٢٧)،
 *   **وفوق السقف المالكُ وحدَه** — المحرّكُ يحرس ذلك، والتأكيدُ بكلمة المرور يلتقطه العميلُ المركزيّ (`StepUpGate`).
 *
 * # قراراتُ المالك ٢٠٢٦-١٠-١٥
 *
 * - **«موافقة/رفض» من الخادم** (`can_decide`) — لا تظهر لمقترحها.
 * - **البطاقاتُ تتبع الفلاتر**، و**بطاقةُ التأخّر تفتح المتأخّرَ وحدَه** (`overdue=1`).
 * - **ولا «السائق» في الجهة المذنبة حين يكون النوعُ «سائق»** — لا تعويضَ لذنبه.
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
import { ApproveButton, RejectButton, WalletRequestsList } from "@/components/admin/WalletRequests";
import {
  Alert,
  Badge,
  Chips,
  Button,
  Input,
  Textarea,
  Modal,
  PageContainer,
  LoadingState,
  Pagination,
  useLiveData,
  IconWallet,
  IconOrder,
  IconStatus,
  IconDate,
  IconWarning,
  IconCheck,
  IconBlock,
  IconSupport,
  IconHourglass,
  Money,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall, useCanOpen } from "@/lib/policy";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";
import { OpenLink } from "@/components/admin/OpenLink";
import { PeriodPicker } from "./PeriodPicker";

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
  /** **يظهر «موافقة/رفض» لهذا الموظّف؟** — من الخادم بحكم الموافقة نفسِه. */
  can_decide?: boolean;
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
  /** **المتأخّرُ وحدَه** — «1» من بطاقة التأخّر. */
  overdue: string;
}

const EMPTY: Filters = { status: "pending", kind: "", fault: "", person: "", from: "", to: "", overdue: "" };

function toQuery(f: Filters, page: number): string {
  const q = new URLSearchParams();
  (Object.keys(f) as (keyof Filters)[]).forEach((k) => {
    if (f[k]) q.set(k, f[k]);
  });
  if (page > 1) q.set("page", String(page));
  return q.toString();
}

/**
 * **التعويضات — تبويبٌ في «المشاكل والتعويضات»** (قرارُ المالك ٢٠٢٦-١٠-٢٧: «خليهن بس بقسم المشاكل
 * والتعويضات»). و`embedded` يُخفي الرأسَ لأنّ الصفحةَ الأمَّ لها رأسُها، **ويُبقي `tab=compensations`
 * في الرابط** فلا يرجع التحديثُ إلى تبويبٍ آخر. والرابطُ القديمُ `/dashboard/compensations` باقٍ.
 */
export function CompensationsView({ embedded = false }: { embedded?: boolean }) {
  const { can } = useAuth();
  const canDecide = can("finance.manage");
  // **والحسابُ والطلبُ والشكوى لمن تُفتح له** (قرارُ المالك ٢٠٢٦-١٠-٠٥): الماليّةُ
  // لا ترى الطلباتِ ولا الحساباتِ ولا الشكاوى — فتُقرأ نصّاً لا رابطاً.
  const open = useCanOpen();
  const canTickets = useCanCall()("GET", "/tickets");
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
        overdue: q.get("overdue") === "1" ? "1" : "",
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
      const full = embedded ? `tab=compensations${qs ? `&${qs}` : ""}` : qs;
      window.history.replaceState(null, "", `${window.location.pathname}${full ? `?${full}` : ""}`);
    } catch {
      /* الرابطُ راحةٌ لا شرط */
    }
  }, [qs, ready, embedded]);

  const { data, loading, error, reload } = useLiveData<{
    compensations: Compensation[];
    total: number;
    page: number;
    per_page: number;
    summary: Summary;
    /** **يومُ الحساب** — أزرارُ المدّة الجاهزة تُبنى عليه. */
    today?: string;
  }>(
    () => api(`/api/v1/admin/compensations?${qs}`),
    ["order", "wallet"],
    [qs],
  );

  function set<K extends keyof Filters>(k: K, v: Filters[K]) {
    setFilters((f) => {
      const next = { ...f, [k]: v };
      // **ولا «السائق» جهةً مذنبةً لتعويض السائق** — يُمسح إن كان مختاراً.
      if (k === "kind" && v === "driver" && next.fault === "driver") next.fault = "";
      // **والتبويبُ يُنهي «المتأخّر وحدَه»** — ذاك فلترُ البطاقة لا التبويب.
      if (k === "status") next.overdue = "";
      return next;
    });
    setPage(1);
  }

  if (loading && !data) return <LoadingState />;

  const rows = data?.compensations ?? [];
  const sum = data?.summary;
  const pendingTab = filters.status === "pending";

  const tabs = [
    { key: "pending" as const, label: C.tabs.pending, icon: IconHourglass, count: sum?.pending_count, warn: true },
    { key: "approved" as const, label: C.tabs.approved, icon: IconCheck },
    { key: "rejected" as const, label: C.tabs.rejected, icon: IconBlock },
    { key: "all" as const, label: C.tabs.all, icon: IconStatus },
  ];
  const filtered = filters.kind || filters.fault || filters.person || filters.from || filters.to;

  // تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦: «ألغِ عرض الجداول») — رأسٌ موحّدٌ بتبويبات الحالة، أربعةُ أرقامٍ
  // تُضغط، الفلاترُ بكرتٍ واحد، وكرتُ موافقةٍ لكلّ تعويضٍ ولكلّ طلبِ حركة محفظة.
  return (
    <Wrap embedded={embedded}>
      {embedded ? (
        <p className="text-sm text-ink-muted">{C.hintAll}</p>
      ) : (
        <PageHero<Tab>
          icon={IconWallet}
          title={C.titleAll}
          subtitle={C.hintAll}
          tabs={tabs}
          value={filters.status}
          onChange={(k) => set("status", k)}
        />
      )}

      {error && (
        <Alert>
          <span>{C.loadFailed}</span>{" "}
          <button type="button" onClick={() => reload()} className="font-bold underline">
            {m.common.retry}
          </button>
        </Alert>
      )}

      {sum && (
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
          <Tile
            icon={IconHourglass}
            label={C.cardPending}
            value={fmtNum(sum.pending_count)}
            sub={C.cardPendingSub.replace("{amount}", fmtMoney(sum.pending_sum))}
            tone={sum.overdue_count > 0 ? "danger" : sum.pending_count > 0 ? "accent" : "muted"}
            onClick={() => set("status", "pending")}
          />
          <Tile
            icon={IconCheck}
            label={C.cardApproved}
            value={fmtMoney(sum.approved_month_sum)}
            sub={C.cardApprovedSub.replace("{n}", fmtNum(sum.approved_month_count))}
            tone="success"
            onClick={() => set("status", "approved")}
          />
          <Tile
            icon={IconBlock}
            label={C.cardRejected}
            value={fmtNum(sum.rejected_month_count)}
            tone="muted"
            onClick={() => set("status", "rejected")}
          />
          <Tile
            icon={IconWarning}
            label={C.cardOverdue.replace("{h}", fmtNum(sum.overdue_hours))}
            value={fmtNum(sum.overdue_count)}
            tone={sum.overdue_count > 0 ? "danger" : "muted"}
            onClick={() => {
              setFilters((f) => ({ ...f, status: "pending", overdue: "1" }));
              setPage(1);
            }}
          />
        </div>
      )}

      {filters.overdue === "1" && (
        <Alert tone="warning">
          <span>{C.overdueOnly}</span>{" "}
          <button type="button" onClick={() => set("overdue", "")} className="font-bold underline">
            {C.overdueClear}
          </button>
        </Alert>
      )}

      {/* ── الفلاترُ في كرتٍ واحد (فحصُ ٢٠٢٦-١٠-٢٧) — النوعُ و«مين الغلطان» أزرارٌ بدل القوائم،
          والتاريخان باسمهما فوقهما مع أزرار مدّةٍ جاهزة بيوم الحساب من الخادم. ── */}
      <section className="surface space-y-4 p-4">
        {/* **والحالُ في الكرت نفسِه داخل التبويب** — لا كرتان فوق بعض. */}
        {embedded && (
          <Chips
            wrap
            items={tabs.map((t) => ({ id: t.key, label: t.label, count: t.count }))}
            value={filters.status}
            onChange={(k) => set("status", k as Tab)}
          />
        )}
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <div className="space-y-1.5">
            <p className="text-xs font-bold text-ink-muted">{C.kindLabel}</p>
            <Chips
              wrap
              items={[{ id: "", label: C.filterKind }, ...Object.keys(KINDS).map((k) => ({ id: k, label: KINDS[k] }))]}
              value={filters.kind}
              onChange={(k) => set("kind", k)}
            />
          </div>
          <div className="space-y-1.5">
            <p className="text-xs font-bold text-ink-muted">{C.faultLabel}</p>
            <Chips
              wrap
              items={[
                { id: "", label: C.faultAll },
                ...Object.keys(FAULTS)
                  .filter((k) => !(filters.kind === "driver" && k === "driver"))
                  .map((k) => ({ id: k, label: FAULTS[k] })),
              ]}
              value={filters.fault}
              onChange={(k) => set("fault", k)}
            />
          </div>
        </div>
        <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
          <Input
            id="comp-person"
            label={C.personLabel}
            value={filters.person}
            onChange={(e) => set("person", e.target.value)}
            placeholder={C.filterPerson}
          />
          <div className="space-y-1.5">
            <p className="text-xs font-bold text-ink-muted">{C.dateLabel}</p>
            <PeriodPicker
              from={filters.from}
              to={filters.to}
              today={data?.today ?? ""}
              allLabel={C.anyDate}
              onChange={(f, t) => {
                setFilters((x) => ({ ...x, from: f, to: t }));
                setPage(1);
              }}
            />
          </div>
        </div>
        {filtered && (
          <Button variant="ghost" onClick={() => setFilters((f) => ({ ...EMPTY, status: f.status }))}>
            {C.clearFilters}
          </Button>
        )}
      </section>

      {data && rows.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span
            className={`inline-flex h-16 w-16 items-center justify-center rounded-full ${
              pendingTab ? "bg-success-solid text-on-solid" : "bg-field text-ink-muted"
            }`}
          >
            <IconCheck size={28} />
          </span>
          <h2 className="heading-section">{pendingTab && !filtered ? C.empty : C.emptyHistory}</h2>
          <p className="max-w-md text-sm text-ink-muted">{filtered ? C.emptyFilteredHint : C.emptyHint}</p>
        </section>
      ) : (
        rows.length > 0 && (
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2 2xl:grid-cols-3">
            {rows.map((c) => (
              <CompensationCard
                key={c.id}
                c={c}
                pendingTab={pendingTab}
                canAct={canDecide && c.status === "pending" && !!c.can_decide}
                openUser={open.user}
                openOrder={open.order}
                canTickets={canTickets}
                onDecide={(approve) => setDeciding({ c, approve })}
              />
            ))}
          </div>
        )
      )}

      {data && data.total > data.per_page && (
        <div className="flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}

      {/* **وطلباتُ حركة المحفظة اليدويّة هنا أيضاً** (قرارُ المالك ٢٠٢٦-١٠-٠٤). */}
      <section className="space-y-3">
        <div className="flex items-center gap-3">
          <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
            <IconWallet size={18} />
          </span>
          <div className="min-w-0">
            <h2 className="heading-section">{m.admin.acc.requestsTitle}</h2>
            <p className="text-xs text-ink-muted">{m.admin.acc.requestsHint}</p>
          </div>
        </div>
        <WalletRequestsList />
      </section>

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
    </Wrap>
  );
}

function Wrap({ embedded, children }: { embedded: boolean; children: React.ReactNode }) {
  // **وفراغٌ بين الأجزاء في التبويب أيضاً** (فحصُ ٢٠٢٦-١٠-٢٧) — كان `div` عارياً فالتصقت المربّعاتُ
  // والفلاترُ والبطاقاتُ ببعضها؛ والصفحةُ المستقلّةُ تأخذه من `PageContainer`.
  return embedded ? <div className="space-y-5">{children}</div> : <PageContainer width="full">{children}</PageContainer>;
}

/** **لونُ النوع في رأس البطاقة** — يُعرف النوعُ بنظرة. */
const KIND_TONE: Record<string, string> = {
  driver: "bg-accent text-on-bright",
  merchant_goods: "bg-warning-solid text-on-solid",
  complaint: "bg-danger-solid text-on-solid",
};

/** **كرتُ تعويض** — بشكل بطاقات الموافقات في الخزينة: رأسٌ ملوّن، المستفيدُ والمبلغُ كبيراً، السببُ مقتبساً، والقرار. */
function CompensationCard({
  c,
  pendingTab,
  canAct,
  openUser,
  openOrder,
  canTickets,
  onDecide,
}: {
  c: Compensation;
  pendingTab: boolean;
  canAct: boolean;
  openUser: boolean;
  openOrder: boolean;
  canTickets: boolean;
  onDecide: (approve: boolean) => void;
}) {
  const initial = (c.driver_name || "?").trim().charAt(0);
  return (
    <article className="surface flex flex-col overflow-hidden transition-shadow hover:elev-2">
      <header className="flex items-center gap-2.5 border-b border-line-soft bg-field px-4 py-2.5">
        <span
          className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${KIND_TONE[c.kind] ?? "bg-accent text-on-bright"}`}
        >
          <IconWallet size={15} />
        </span>
        <span className="min-w-0 flex-1 truncate text-sm font-bold">{KINDS[c.kind] ?? c.kind}</span>
        <span className="shrink-0 text-2xs text-ink-muted" dir="ltr">
          {fmtDateTime(c.created_at)}
        </span>
      </header>

      <div className="flex flex-1 flex-col gap-3 p-4">
        {/* **المستفيدُ والمبلغ** */}
        <div className="flex items-center gap-3">
          <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-field text-base font-bold text-ink-muted">
            {initial}
          </span>
          <div className="min-w-0 flex-1">
            <OpenLink allowed={openUser} href={`/dashboard/users/${c.driver_id}`} className="block truncate font-bold hover:underline">
              {c.driver_name}
            </OpenLink>
            <p className="truncate text-xs text-ink-muted">
              {ROLES[c.beneficiary_role] ? `${ROLES[c.beneficiary_role]} · ` : ""}
              {C.colProposedBy}: {c.proposed_by ? c.proposed_by_name || "—" : C.engine}
            </p>
          </div>
          {/* **والمرفوضُ لا يُكتب «المبلغ» كبيراً** (فحصُ ٢٠٢٦-١٠-٢٧) — لم يُصرف شيء، فيُقال «كان
              المقترح» بلونٍ باهت. */}
          {c.status === "rejected" ? (
            <div className="shrink-0 text-end">
              <span className="block text-2xs text-ink-muted">{C.suggestedRejected}</span>
              <span className="block text-sm font-bold text-ink-muted">
                <Money value={c.suggested_amount} />
              </span>
            </div>
          ) : (
            <div className="shrink-0 text-end">
              <span className="block text-2xs text-ink-muted">
                {c.status === "pending" || c.amount == null ? C.suggested : C.amount}
              </span>
              <span className="figure block leading-none">
                <Money value={c.amount ?? c.suggested_amount} />
              </span>
            </div>
          )}
        </div>

        {/* **السببُ والذنب** */}
        <div className="border-s-2 border-accent ps-3 text-sm leading-relaxed">
          <p>{c.reason_label}</p>
          <p className="text-xs text-ink-muted">
            {C.fault}: {c.fault ? FAULTS[c.fault] ?? c.fault : "—"}
          </p>
          {c.fault_mismatch && (
            <p className="flex items-center gap-1 text-xs font-bold text-warning">
              <IconWarning size={12} /> {FAULTS[c.expected_fault] ?? ""}
            </p>
          )}
        </div>

        {(c.order_number > 0 || c.ticket_number > 0) && (
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-sm">
            {c.order_number > 0 && (
              <span className="flex items-center gap-1">
                <IconOrder size={14} className="text-ink-muted" />
                <OpenLink
                  allowed={openOrder}
                  href={`/dashboard/orders?q=${c.order_number}`}
                  className="font-bold text-primary-dark hover:underline"
                  dir="ltr"
                >
                  #{fmtRef(c.order_number)}
                </OpenLink>
              </span>
            )}
            {c.ticket_number > 0 && (
              <span className="flex items-center gap-1">
                <IconSupport size={14} className="text-ink-muted" />
                <OpenLink
                  allowed={canTickets && !!c.ticket_id}
                  href={`/dashboard/tickets?tab=tickets&t=${c.ticket_id}`}
                  className="text-primary-dark hover:underline"
                >
                  {C.ticketRef.replace("{n}", fmtNum(c.ticket_number))}
                </OpenLink>
              </span>
            )}
          </div>
        )}

        {pendingTab ? (
          <p className={`flex items-center gap-1.5 text-xs ${c.overdue ? "font-bold text-danger" : "text-ink-muted"}`}>
            <IconDate size={12} />
            {C.colWaiting}: {fmtSpan((Date.now() - new Date(c.created_at).getTime()) / 1000)}
            {c.overdue && <Badge variant="danger">{C.overdueBadge}</Badge>}
          </p>
        ) : (
          <div className="space-y-1 rounded-control bg-field px-3 py-2 text-xs">
            <Badge variant={c.status === "approved" ? "success" : c.status === "rejected" ? "danger" : "warning"}>
              {STATUSES[c.status] ?? c.status}
            </Badge>
            {c.decided_by_name && (
              <p className="text-ink-muted">
                {C.decidedBy}: {c.decided_by_name}
                {c.decided_at && <span dir="ltr"> · {fmtDateTime(c.decided_at)}</span>}
              </p>
            )}
            {c.decision_note && <p className="text-ink">{c.decision_note}</p>}
            {c.self_approved && <p className="font-bold text-warning">{C.selfApproved}</p>}
          </div>
        )}
      </div>

      {/* **قرارٌ بضغطة** — موافقةٌ بالأخضر ورفضٌ بإطارٍ أحمر، عرضَ البطاقة. */}
      {canAct && (
        <footer className="grid grid-cols-2 gap-2 border-t border-line-soft p-3">
          <ApproveButton label={C.approve} onClick={() => onDecide(true)} />
          <RejectButton label={C.reject} onClick={() => onDecide(false)} />
        </footer>
      )}
    </article>
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
