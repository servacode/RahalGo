"use client";

/**
 * طلبات سحب الرصيد — المالية تصرف أو ترفض، والقيد يُسجَّل في دفتر المحفظة.
 *
 * **قراراتُ المالك ٢٠٢٦-١٠-٠٤ (قسمُ طلبات السحب):**
 *  - الحالاتُ الستُّ كلُّها بأسمائها وألوانها وفلاترها وأفعالها:
 *    معلَّق ← «اصرف» / «ارفض» / «بدأ الصرف» · قيد الصرف ← «تم الصرف» / «فشل» ·
 *    مصروف ← «ارتدّ السحب» بسببٍ إجباريّ.
 *  - كلُّ فعلٍ يقول أثرَه بالأرقام قبل التأكيد («رح ينخصم ١٠٠٬٠٠٠ ل.س من رصيد أحمد»).
 *  - الصرفُ نقداً من المكتب يخرج من صندوق المكتب (يختاره القرِّر).
 *  - «إضافة رصيد» خرجت من هنا: الشحنُ طلبٌ من صفحة الحساب يوافق عليه موظفٌ آخر.
 *  - أربعُ بطاقات · الدورُ عمودٌ وفلتر · ملاحظةُ الطالب وقرارُ المالية منفصلتان ·
 *    مين قرّر ومتى · وتغييرُ الفلتر يرجع للصفحة الأولى · والخطأُ لا يُعرض فراغاً.
 */

import { useState } from "react";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  fmtDateTime,
  errorText,
} from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  Chips,
  Input,
  Radio,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ReloadState,
  DataView,
  Pagination,
  StatGrid,
  StatCard,
  ViewToggle,
  useViewMode,
  type DataColumn,
  useLiveData,
  IconWallet,
  IconUser,
  IconStatus,
  IconDate,
  IconAdd,
  Money,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanOpen } from "@/lib/policy";
import { roleLabelByCode } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const P = m.shared.payout;
const A = m.shared.payoutAdmin;

type Status = "pending" | "processing" | "paid" | "rejected" | "failed" | "reversed";
type Role = "driver" | "merchant" | "sales";

interface Payout {
  id: string;
  user_id: string;
  user_name: string;
  user_phone: string;
  user_role: Role | "";
  amount: number;
  status: Status;
  note: string;
  decision: string;
  balance: number;
  reserved: number;
  available: number;
  created_at: string;
  decided_at: string | null;
  decided_by_name: string;
  paid_via: "" | "cash" | "transfer";
}

interface Stats {
  waiting_count: number;
  waiting_sum: number;
  processing_count: number;
  processing_sum: number;
  paid_month_count: number;
  paid_month_sum: number;
  returned_month_count: number;
  returned_month_sum: number;
}

const STATUSES: Status[] = ["pending", "processing", "paid", "rejected", "failed", "reversed"];
const ROLES: Role[] = ["driver", "merchant", "sales"];

const VARIANT: Record<Status, "warning" | "info" | "success" | "danger" | "neutral" | "violet"> = {
  pending: "warning",
  processing: "info",
  paid: "success",
  rejected: "neutral",
  failed: "danger",
  reversed: "violet",
};

/** **الأفعالُ حسب الحال** — آلةُ الخادم نفسُها (`payout_handlers.go`). */
type Action = "paid" | "rejected" | "processing" | "failed" | "reversed";
const ACTIONS: Record<Status, Action[]> = {
  pending: ["paid", "processing", "rejected"],
  processing: ["paid", "failed"],
  paid: ["reversed"],
  rejected: [],
  failed: [],
  reversed: [],
};

function actionLabel(a: Action, from: Status): string {
  switch (a) {
    case "paid":
      return from === "processing" ? A.actDone : A.actPay;
    case "rejected":
      return A.actReject;
    case "processing":
      return A.actStart;
    case "failed":
      return A.actFail;
    case "reversed":
      return A.actReverse;
  }
}

const NEEDS_REASON: Action[] = ["rejected", "failed", "reversed"];

function readParam(key: string): string {
  if (typeof window === "undefined") return "";
  return new URLSearchParams(window.location.search).get(key) ?? "";
}

export default function PayoutsPage() {
  const { can } = useAuth();
  const canUsers = useCanOpen().user;
  // **وقرارُ السحب قدرةٌ بذاتها** — `payouts.decide`.
  const canDecide = can("payouts.decide");
  // **والفلترُ من الرابط عند الفتح** — بطاقةُ «بانتظار قرارك» في الرئيسيّة.
  const [status, setStatusRaw] = useState(() => readParam("status"));
  const [role, setRoleRaw] = useState(() => readParam("role"));
  const [page, setPage] = useState(1);
  const [deciding, setDeciding] = useState<{ p: Payout; action: Action } | null>(null);
  const [view, setView] = useViewMode("payouts");

  /** **وتغييرُ الفلتر يرجع للصفحة الأولى** — كانت الصفحةُ الثالثةُ تبقى فيظهر «لا طلبات». */
  function setStatus(s: string) {
    setStatusRaw(s);
    setPage(1);
  }
  function setRole(r: string) {
    setRoleRaw(r);
    setPage(1);
  }

  const qs = new URLSearchParams({ page: String(page) });
  if (status) qs.set("status", status);
  if (role) qs.set("role", role);

  const { data, loading, error, reload } = useLiveData<{
    payouts: Payout[];
    total: number;
    per_page: number;
    pending_total: number;
    stats: Stats;
  }>(() => api(`/api/v1/admin/payouts?${qs.toString()}`), ["wallet"], [status, role, page]);

  if (loading) return <LoadingState />;
  const rows = data?.payouts ?? [];
  const st = data?.stats;
  const filtered = status !== "" || role !== "";

  const columns: DataColumn<Payout>[] = [
    {
      id: "who",
      header: m.terms.name,
      icon: <IconUser />,
      primary: true,
      cell: (p) => (
        <span>
          {p.user_name}{" "}
          <span dir="ltr" className="text-xs text-ink-muted">
            {p.user_phone}
          </span>
        </span>
      ),
    },
    {
      id: "role",
      header: A.roleCol,
      cell: (p) => (p.user_role ? roleLabelByCode(p.user_role) : "—"),
    },
    {
      id: "amount",
      header: P.amount,
      icon: <IconWallet />,
      primary: true,
      cell: (p) => (
        <span className="font-bold text-primary-dark">
          <Money value={p.amount} />
        </span>
      ),
    },
    {
      /* **والمحجوزُ والمتاحُ يُعرَضان** — `XG-12`: الماليّةُ تقرّر على ما يجوز صرفُه. */
      id: "available",
      header: m.terms.walletAvailable,
      icon: <IconWallet />,
      cell: (p) => (
        <span className="text-sm">
          <Money value={p.available} small />
          <span className="block text-xs text-ink-muted">
            {m.terms.walletBalance}: {fmtNum(p.balance)} · {m.terms.walletReserved}: {fmtNum(p.reserved)}
          </span>
        </span>
      ),
    },
    {
      id: "status",
      header: m.terms.status,
      icon: <IconStatus />,
      cell: (p) => (
        <span>
          <Badge variant={VARIANT[p.status]}>{A.status[p.status]}</Badge>
          {p.paid_via && (
            <span className="block text-xs text-ink-muted">
              {p.paid_via === "cash" ? A.paidViaCash : A.paidViaTransfer}
            </span>
          )}
        </span>
      ),
    },
    {
      /* **ملاحظةُ الطالب وقرارُ المالية سطران** — كانت الأولى تختفي بعد القرار. */
      id: "note",
      header: A.requesterNote,
      cell: (p) => (
        <span className="text-sm">
          <span className="block">{p.note || "—"}</span>
          {p.decision && (
            <span className="block text-ink-muted">
              {A.financeNote}: {p.decision}
            </span>
          )}
        </span>
      ),
    },
    {
      id: "decided",
      header: A.decidedCol,
      cell: (p) =>
        p.decided_at ? (
          <span className="text-sm">
            {p.decided_by_name || "—"}
            <span dir="ltr" className="block text-xs text-ink-muted">
              {fmtDateTime(p.decided_at)}
            </span>
          </span>
        ) : (
          "—"
        ),
    },
    {
      id: "date",
      header: m.terms.date,
      icon: <IconDate />,
      cell: (p) => <span dir="ltr">{fmtDateTime(p.created_at)}</span>,
    },
  ];

  const statusChips = [
    { id: "", label: A.allStatuses },
    ...STATUSES.map((s) => ({ id: s, label: A.status[s] })),
  ];
  const roleChips = [
    { id: "", label: A.allRoles },
    ...ROLES.map((r) => ({ id: r, label: roleLabelByCode(r) })),
  ];

  const count = (n: number) => A.cardCount.replace("{n}", fmtNum(n));

  return (
    <PageContainer width="full">
      <PageHeader
        icon={IconWallet}
        title={P.adminTitle}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            {/* **«إضافة رصيد» خرجت من هنا** (قرارُ المالك ٢٠٢٦-١٠-٠٤، البند ٣):
                الشحنُ طلبٌ من صفحة الحساب، يوافق عليه موظفٌ آخر، ويدخل صندوقَ المكتب. */}
            {/* **ولمن يفتح الحسابات وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٥): الماليّةُ لا تراها. */}
            {canUsers && (
              <ButtonLink href="/dashboard/users" variant="secondary" title={A.topupHint}>
                <IconAdd size={16} />
                {A.topupLink}
              </ButtonLink>
            )}
            <ViewToggle
              view={view}
              onChange={setView}
              tableLabel={m.common.viewTable}
              cardsLabel={m.common.viewCards}
            />
          </div>
        }
      />

      {/* **البطاقاتُ الأربع** — و«بانتظار الصرف» يشمل «قيد الصرف»: مالٌ محجوزٌ لم يخرج. */}
      <StatGrid>
        <StatCard
          label={A.cardWaiting}
          value={fmtMoney(st?.waiting_sum ?? 0)}
          sub={count(st?.waiting_count ?? 0)}
          icon={IconWallet}
          tone={(st?.waiting_count ?? 0) > 0 ? "accent" : "muted"}
          onClick={() => setStatus("pending")}
          selected={status === "pending"}
        />
        <StatCard
          label={A.cardProcessing}
          value={fmtMoney(st?.processing_sum ?? 0)}
          sub={count(st?.processing_count ?? 0)}
          icon={IconWallet}
          tone={(st?.processing_count ?? 0) > 0 ? "warning" : "muted"}
          onClick={() => setStatus("processing")}
          selected={status === "processing"}
        />
        <StatCard
          label={A.cardPaidMonth}
          value={fmtMoney(st?.paid_month_sum ?? 0)}
          sub={count(st?.paid_month_count ?? 0)}
          icon={IconWallet}
          tone="default"
          onClick={() => setStatus("paid")}
          selected={status === "paid"}
        />
        <StatCard
          label={A.cardReturnedMonth}
          value={fmtMoney(st?.returned_month_sum ?? 0)}
          sub={count(st?.returned_month_count ?? 0)}
          icon={IconWallet}
          tone={(st?.returned_month_count ?? 0) > 0 ? "danger" : "muted"}
        />
      </StatGrid>

      <div className="my-4 space-y-2">
        <Chips items={statusChips} value={status} onChange={setStatus} wrap />
        <Chips items={roleChips} value={role} onChange={setRole} wrap />
      </div>

      {error && !data ? (
        <ReloadState onRetry={reload} label={A.loadError} />
      ) : rows.length === 0 ? (
        <EmptyState icon={IconWallet} title={filtered ? A.emptyFiltered : A.emptyAll} />
      ) : (
        <DataView
          items={rows}
          getKey={(p) => p.id}
          columns={columns}
          view={view}
          empty={filtered ? A.emptyFiltered : A.emptyAll}
          actions={
            canDecide
              ? (p) =>
                  ACTIONS[p.status].length > 0 ? (
                    <>
                      {ACTIONS[p.status].map((a) => (
                        <Button
                          key={a}
                          variant={
                            a === "paid" ? "primary" : NEEDS_REASON.includes(a) ? "danger" : "secondary"
                          }
                          onClick={() => setDeciding({ p, action: a })}
                        >
                          {actionLabel(a, p.status)}
                        </Button>
                      ))}
                    </>
                  ) : null
              : undefined
          }
        />
      )}

      {/* **ولا يظهر لصفحةٍ واحدة** — عنصرٌ لا يفعل شيئاً يزاحم ما يفعل. */}
      {data && data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}

      {deciding && (
        <DecideModal
          payout={deciding.p}
          action={deciding.action}
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

/** **جملةُ الأثر** — ماذا سيحدث للمال، بالأرقام، قبل التأكيد. */
function impactOf(p: Payout, action: Action, method: "cash" | "transfer"): string[] {
  const vars = (s: string) => s.replace("{amount}", fmtMoney(p.amount)).replace("{name}", p.user_name);
  switch (action) {
    case "paid":
      return [vars(A.impactPay), ...(method === "cash" ? [A.impactCashOut] : [])];
    case "rejected":
      return [vars(A.impactReject)];
    case "processing":
      return [vars(A.impactStart)];
    case "failed":
      return [vars(A.impactFail)];
    case "reversed":
      return [vars(A.impactReverse), ...(p.paid_via === "cash" ? [A.impactCashIn] : [])];
  }
}

function DecideModal({
  payout,
  action,
  onClose,
  onDone,
}: {
  payout: Payout;
  action: Action;
  onClose: () => void;
  onDone: () => void;
}) {
  const [decision, setDecision] = useState("");
  const [method, setMethod] = useState<"cash" | "transfer">("cash");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const reasonRequired = NEEDS_REASON.includes(action);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      // **والمبلغُ والطريقةُ في الجسم** — يدخلان بصمةَ تأكيد كلمة السرّ،
      // والخادمُ يردّ مبلغاً تغيّر منذ فُتحت النافذة.
      await api(`/api/v1/admin/payouts/${payout.id}/decide`, {
        method: "POST",
        body: JSON.stringify({
          status: action,
          decision: decision.trim(),
          amount: payout.amount,
          ...(action === "paid" ? { method } : {}),
        }),
      });
      onDone();
    } catch (err) {
      setError(errorText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={actionLabel(action, payout.status)}>
      <form onSubmit={submit} className="space-y-4">
        <div className="rounded-control border border-line bg-field px-3 py-2 text-sm">
          <p className="font-bold">{payout.user_name}</p>
          <p className="text-ink-muted">
            <Money value={payout.amount} />
          </p>
        </div>
        {action === "paid" && (
          <fieldset className="space-y-2">
            <legend className="mb-1 text-sm font-medium">{A.methodLabel}</legend>
            <Radio
              name="method"
              label={A.paidViaCash}
              checked={method === "cash"}
              onChange={() => setMethod("cash")}
            />
            <Radio
              name="method"
              label={A.paidViaTransfer}
              checked={method === "transfer"}
              onChange={() => setMethod("transfer")}
            />
          </fieldset>
        )}
        <Alert tone="warning">
          {impactOf(payout, action, method).map((line) => (
            <span key={line} className="block">
              {line}
            </span>
          ))}
        </Alert>
        <Input
          id="decision"
          label={reasonRequired ? A.reasonRequired : A.reasonOptional}
          required={reasonRequired}
          value={decision}
          onChange={(e) => setDecision(e.target.value)}
          placeholder={A.reasonPlaceholder}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions
          submit
          onCancel={onClose}
          busy={busy}
          saveLabel={busy ? m.common.loading : actionLabel(action, payout.status)}
        />
      </form>
    </Modal>
  );
}
