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
 *
 * **وقراراتُ ٢٠٢٦-١٠-١٥:** اسمُ الصفحة «السحوبات» (هو نفسُه في شريط الخزينة) · زرُّ
 * «شحن رصيد» خرج من الرأس · البطاقةُ الرابعةُ تفتح «مرتجع أو فاشل» (`failed_or_reversed`) ·
 * والدورُ من الطلب نفسِه (التطبيقُ الذي طلب منه) لا من أدوار الحساب.
 *  - أربعُ بطاقات · الدورُ عمودٌ وفلتر · ملاحظةُ الطالب وقرارُ المالية منفصلتان ·
 *    مين قرّر ومتى · وتغييرُ الفلتر يرجع للصفحة الأولى · والخطأُ لا يُعرض فراغاً.
 *
 * **وتدقيقُ ٢٠٢٦-١٠-٢٧:** المتاحُ بعد الحجز لا يُقارَن بالمبلغ (الحجزُ منه) ·
 * تبويبُ «بانتظار المالية» بعدِّ المعلَّق وحدَه (`pending_count`) · بطاقةُ «بانتظار الصرف»
 * تفتح الحالتين (`waiting`) · وبطاقتا «هالشهر» تحصران القائمةَ في الشهر (`month=1`) ·
 * ستُّ تبويبات · وأزرارُ القرار تختفي عن طلب الموظّف نفسِه (إلّا المالك).
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
  Input,
  Modal,
  useLiveData,
  IconWallet,
  IconStatus,
  IconWarning,
  IconHourglass,
  Money,
  FormActions,
  AdminPage,
  AdminCard,
  AdminEmpty,
  AdminLoading,
  AdminReload,
  Fact,
  FilterBar,
  FilterSelect,
  MoreMenu,
  StatusBadge,
  SummaryStrip,
  type AdminTab,
  type SummaryItem,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { isOwner, roleLabelByCode } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const P = m.shared.payout;
const A = m.shared.payoutAdmin;

type Status = "pending" | "processing" | "paid" | "rejected" | "failed" | "reversed";
/** **«مرتجع أو فاشل» فلترٌ واحد** — يفهمه الخادمُ (`payoutFailedOrReversed`). */
const FAILED_OR_REVERSED = "failed_or_reversed";
/** **«بانتظار الصرف» فلترٌ واحد** — معلَّقٌ وقيدُ الصرف، كبطاقته (`payoutWaiting`). */
const WAITING = "waiting";
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
  /** المعلَّقُ وحدَه — رقمُ التبويب. */
  pending_count?: number;
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
/** **ستُّ تبويبات** — «فشل» و«مرتجع» داخلَ «مرتجع أو فاشل» فلا يتكرّران. */
const TAB_STATUSES: Status[] = ["pending", "processing", "paid", "rejected"];
/** **الحالاتُ المسكّرة** — لا قرارَ بعدها، فلا تُعرض محفظتُه الحاليّة. */
const CLOSED: Status[] = ["paid", "rejected", "failed", "reversed"];
const ROLES: Role[] = ["driver", "merchant", "sales"];

/* **ولونُ الحالة من القاموس الواحد** (`StatusBadge` بـ`domain="payout"`) — لا خريطةَ ألوانٍ هنا. */

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

/** **والفلترُ من الرابط يُفحص** — قيمةٌ غلطٌ كانت تعلّق الصفحةَ على خطأ ٤٠٠. */
function validStatus(s: string): string {
  return s === FAILED_OR_REVERSED || s === WAITING || (STATUSES as string[]).includes(s) ? s : "";
}
function validRole(r: string): string {
  return (ROLES as string[]).includes(r) ? r : "";
}

/** **جملةُ الفراغ حسب الفلتر** — «ما في شي مرتجع — منيح» أوضحُ من جملةٍ عامّة. */
function emptyTitle(status: string, filtered: boolean): string {
  const E = A.emptyByStatus;
  if (status === WAITING) return E.waiting;
  if (status === FAILED_OR_REVERSED || status === "failed" || status === "reversed") return E.returned;
  if (status === "pending" || status === "processing" || status === "paid" || status === "rejected") return E[status];
  return filtered ? A.emptyFiltered : A.emptyAll;
}

export default function PayoutsPage() {
  const { can, user } = useAuth();
  // **وقرارُ السحب قدرةٌ بذاتها** — `payouts.decide`.
  const canDecide = can("payouts.decide");
  // **ولا أحدَ يقرّر في سحبه هو** — الخادمُ يرفضه (`approval.Check` بـ`OwnFunds`) إلّا المالك.
  const meId = user?.id ?? "";
  const owner = isOwner(user?.roles ?? []);
  // **والفلترُ من الرابط عند الفتح** — بطاقةُ «بانتظار قرارك» في الرئيسيّة.
  const [status, setStatusRaw] = useState(() => validStatus(readParam("status")));
  const [role, setRoleRaw] = useState(() => validRole(readParam("role")));
  const [month, setMonthRaw] = useState(() => readParam("month") === "1");
  const [page, setPage] = useState(1);
  const [deciding, setDeciding] = useState<{ p: Payout; action: Action } | null>(null);

  /** **وتغييرُ الفلتر يرجع للصفحة الأولى** — كانت الصفحةُ الثالثةُ تبقى فيظهر «لا طلبات». */
  function setStatus(s: string) {
    setStatusRaw(s);
    setPage(1);
  }
  function clearFilters() {
    setStatusRaw("");
    setRoleRaw("");
    setMonthRaw(false);
    setPage(1);
  }
  function setRole(r: string) {
    setRoleRaw(r);
    setPage(1);
  }
  function setMonth(on: boolean) {
    setMonthRaw(on);
    setPage(1);
  }

  const qs = new URLSearchParams({ page: String(page) });
  if (status) qs.set("status", status);
  if (role) qs.set("role", role);
  if (month) qs.set("month", "1");

  const { data, loading, error, reload } = useLiveData<{
    payouts: Payout[];
    total: number;
    per_page: number;
    pending_total: number;
    stats: Stats;
  }>(() => api(`/api/v1/admin/payouts?${qs.toString()}`), ["wallet"], [status, role, month, page]);

  const rows = data?.payouts ?? [];
  const st = data?.stats;
  const filtered = status !== "" || role !== "" || month;
  // **رقمُ «بانتظار المالية» للمعلَّق وحدَه** — والخادمُ القديمُ بلا الحقل: الفرق.
  const pendingCount = st ? (st.pending_count ?? st.waiting_count - st.processing_count) : undefined;
  const count = (n: number) => A.cardCount.replace("{n}", fmtNum(n));

  // ══════════════════════════════════════════════════════════════════
  // **دفعةُ التصميم ١ (٢٠٢٦-١٠-٣١):** قالبُ اللوحة الواحد — رأسٌ بسطرٍ وتبويباتٌ خطّاً، شريطُ فلاتر
  // (الدور + هالشهر)، أربعةُ أرقامٍ **لا تُفلتر** (التبويباتُ هي الفلتر)، وبطاقةٌ لكلّ طلبٍ بلا صناديقَ
  // داخلها. **والرفضُ والفشلُ والارتدادُ خلف «⋯»** — لا أحمرَ بعرض البطاقة بجانب «اصرف».
  // ══════════════════════════════════════════════════════════════════
  const tabs: AdminTab[] = [
    { key: "all", label: A.allStatuses },
    ...TAB_STATUSES.map((s) => ({
      key: s,
      label: A.status[s],
      count: s === "pending" ? pendingCount : s === "processing" ? st?.processing_count : undefined,
      attention: s === "pending",
    })),
    { key: FAILED_OR_REVERSED, label: A.failedOrReversed },
    // **«بانتظار الصرف» تبويبٌ يظهر حين يفتحه رابطُه** — كي لا يبقى الرأسُ بلا تبويبٍ مختار.
    ...(status === WAITING ? [{ key: WAITING, label: A.waitingTab }] : []),
    // **وكذلك «فشل» أو «مرتجع» وحدَهما من رابطٍ قديم.**
    ...(status === "failed" || status === "reversed" ? [{ key: status, label: A.status[status] }] : []),
  ];

  const waiting = st?.waiting_count ?? 0;
  const summary: SummaryItem[] = [
    {
      key: "waiting",
      icon: IconHourglass,
      label: A.cardWaiting,
      value: fmtMoney(st?.waiting_sum ?? 0),
      meta:
        (st?.processing_count ?? 0) > 0
          ? A.waitingSplit.replace("{a}", fmtNum(pendingCount ?? 0)).replace("{b}", fmtNum(st?.processing_count ?? 0))
          : count(waiting),
      tone: waiting > 0 ? "attention" : "muted",
    },
    {
      key: "processing",
      icon: IconStatus,
      label: A.cardProcessing,
      value: fmtMoney(st?.processing_sum ?? 0),
      meta: count(st?.processing_count ?? 0),
      tone: (st?.processing_count ?? 0) > 0 ? "neutral" : "muted",
    },
    {
      key: "paid",
      icon: IconWallet,
      label: A.cardPaidMonth,
      value: fmtMoney(st?.paid_month_sum ?? 0),
      meta: count(st?.paid_month_count ?? 0),
      tone: (st?.paid_month_count ?? 0) > 0 ? "good" : "muted",
    },
    {
      key: "returned",
      icon: IconWarning,
      label: A.cardReturnedMonth,
      value: fmtMoney(st?.returned_month_sum ?? 0),
      meta: count(st?.returned_month_count ?? 0),
      tone: (st?.returned_month_count ?? 0) > 0 ? "danger" : "muted",
    },
  ];

  return (
    <AdminPage
      icon={IconWallet}
      title={P.adminTitle}
      count={data ? data.total : null}
      tabs={tabs}
      tab={status || "all"}
      onTab={(k) => setStatus(k === "all" ? "" : k)}
      notices={
        /* **وفشلُ التحديث بعد تغيير الفلتر يُقال** — كانت الكروتُ القديمةُ تبقى بلا كلمة. */
        error && data ? <Alert tone="warning">{A.staleData}</Alert> : undefined
      }
      toolbar={
        <FilterBar
          filtered={role !== "" || month}
          onClear={() => {
            setRole("");
            setMonth(false);
          }}
        >
          <FilterSelect id="payout-role" label={A.roleCol} value={role} onChange={setRole}>
            <option value="">{A.allRoles}</option>
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {roleLabelByCode(r)}
              </option>
            ))}
          </FilterSelect>
          {/* **وحصرُ الشهر قائمةٌ ظاهرة** — كان يُفتح من البطاقتين وحدَهما فيُنسى مفعّلاً. */}
          <FilterSelect
            id="payout-month"
            label={m.admin.kit.period.label}
            value={month ? "month" : ""}
            onChange={(v) => setMonth(v === "month")}
          >
            <option value="">{A.scopeAll}</option>
            <option value="month">{A.scopeMonth}</option>
          </FilterSelect>
        </FilterBar>
      }
      summary={<SummaryStrip items={summary} />}
      pagination={data ? { page, total: data.total, perPage: data.per_page, onChange: setPage } : undefined}
    >
      {loading && !data ? (
        <AdminLoading />
      ) : error && !data ? (
        <AdminReload onRetry={reload} label={A.loadError} />
      ) : rows.length === 0 ? (
        <AdminEmpty
          icon={IconWallet}
          title={emptyTitle(status, filtered)}
          action={
            filtered ? (
              <Button variant="secondary" size="sm" onClick={clearFilters}>
                {A.clearFilter}
              </Button>
            ) : undefined
          }
        />
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {rows.map((p) => (
            <PayoutCard
              key={p.id}
              p={p}
              canDecide={canDecide}
              own={p.user_id === meId && !owner}
              onAct={(a) => setDeciding({ p, action: a })}
            />
          ))}
        </ul>
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
    </AdminPage>
  );
}

/** **كرتُ طلب سحب** — مين طلب، قديش، شو متاح بمحفظته، ملاحظتُه وقرارُ الماليّة، ومين قرّر — وأفعالُه. */
function PayoutCard({
  p,
  canDecide,
  own,
  onAct,
}: {
  p: Payout;
  canDecide: boolean;
  /** **طلبُ الموظّف نفسِه** — لا أزرار، والخادمُ يرفض قراره فيه. */
  own: boolean;
  onAct: (a: Action) => void;
}) {
  const acts = canDecide && !own ? ACTIONS[p.status] : [];
  const closed = CLOSED.includes(p.status);
  // **المتاحُ بعد حجز هذا الطلب** — فالنقصُ الحقيقيُّ متاحٌ سالب (الرصيدُ نقص بعد الحجز).
  const short = !closed && p.available < 0;
  // **الأفعالُ العاديّةُ أزرار، والخطِرةُ (رفض · فشل · ارتداد) خلف «⋯»** — زرٌّ أساسيٌّ واحد.
  const plain = acts.filter((a) => !NEEDS_REASON.includes(a));
  const risky = acts.filter((a) => NEEDS_REASON.includes(a));
  return (
    <AdminCard as="li" className="flex flex-col gap-4">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className="truncate font-bold">{p.user_name || p.user_phone}</p>
          {p.user_name && p.user_name !== p.user_phone && (
            <p className="text-sm text-ink-muted" dir="ltr">
              {p.user_phone}
            </p>
          )}
        </div>
        <div className="flex shrink-0 flex-wrap items-center justify-end gap-1.5">
          {p.user_role && <Badge variant="neutral">{roleLabelByCode(p.user_role)}</Badge>}
          <StatusBadge domain="payout" status={p.status} />
        </div>
      </div>

      <div className="flex items-end justify-between gap-3">
        <div>
          <p className="text-xs text-ink-muted">{P.amount}</p>
          <p className="admin-figure">{fmtMoney(p.amount)}</p>
        </div>
        {p.paid_via && (
          <Badge variant="neutral">{p.paid_via === "cash" ? A.paidViaCash : A.paidViaTransfer}</Badge>
        )}
      </div>

      {/* **والمحجوزُ والمتاحُ يُعرَضان** — `XG-12`: الماليّةُ تقرّر على ما يجوز صرفُه.
          وللطلبات المفتوحة وحدَها: المسكّرُ لا قرارَ فيه، ومحفظةُ اليوم لا تخصّه. */}
      {!closed && (
        <dl className="grid grid-cols-2 gap-3 border-t border-line-soft pt-3 sm:grid-cols-3">
          <Fact label={m.terms.walletAvailable}>
            <b className={`tabular-nums ${short ? "text-danger" : ""}`}>{fmtMoney(p.available)}</b>
          </Fact>
          <Fact label={m.terms.walletBalance}>
            <span className="tabular-nums">{fmtMoney(p.balance)}</span>
          </Fact>
          <Fact label={m.terms.walletReserved}>
            <span className="tabular-nums">{fmtMoney(p.reserved)}</span>
          </Fact>
        </dl>
      )}
      {short && <Alert tone="error">{A.shortfall}</Alert>}

      {/* **ملاحظةُ الطالب وقرارُ المالية سطران** — كانت الأولى تختفي بعد القرار. */}
      {(p.note || p.decision) && (
        <dl className="space-y-2 border-t border-line-soft pt-3">
          {p.note && <Fact label={A.requesterNote}>{p.note}</Fact>}
          {p.decision && <Fact label={A.financeNote}>{p.decision}</Fact>}
        </dl>
      )}

      <div className="flex flex-wrap items-center justify-between gap-x-3 gap-y-1 text-xs text-ink-muted">
        <span>
          {m.terms.date}: <span dir="ltr">{fmtDateTime(p.created_at)}</span>
        </span>
        {p.decided_at && (
          <span>
            {A.decidedCol}: {p.decided_by_name || "—"} · <span dir="ltr">{fmtDateTime(p.decided_at)}</span>
          </span>
        )}
      </div>

      {own && canDecide && ACTIONS[p.status].length > 0 && (
        <p className="mt-auto border-t border-line-soft pt-3 text-sm text-ink-muted">{A.ownRequest}</p>
      )}

      {acts.length > 0 && (
        <div className="mt-auto flex items-center gap-2 border-t border-line-soft pt-3">
          {plain.map((a) => (
            <Button key={a} size="sm" variant={a === "paid" ? "primary" : "secondary"} onClick={() => onAct(a)}>
              {actionLabel(a, p.status)}
            </Button>
          ))}
          {risky.length > 0 && (
            <div className="ms-auto">
              <MoreMenu
                items={risky.map((a) => ({
                  key: a,
                  label: actionLabel(a, p.status),
                  danger: true,
                  onSelect: () => onAct(a),
                }))}
              />
            </div>
          )}
        </div>
      )}
    </AdminCard>
  );
}

/** **لونُ الأثر** — مالٌ يخرج أو يرتدّ أصفر، ومالٌ يرجع لصاحبه أو يبقى محجوزاً معلومة. */
const impactTone = (a: Action): "warning" | "info" => (a === "paid" || a === "reversed" ? "warning" : "info");

/** **جملةُ الأثر** — ماذا سيحدث للمال، بالأرقام، قبل التأكيد. */
function impactOf(p: Payout, action: Action, method: "cash" | "transfer" | ""): string[] {
  const vars = (s: string) => s.replace("{amount}", fmtMoney(p.amount)).replace("{name}", p.user_name || p.user_phone);
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
  // **ولا طريقةَ جاهزة** — «نقداً» افتراضاً كان يُخرج المبلغَ من صندوق المكتب لمن لم ينتبه.
  const [method, setMethod] = useState<"cash" | "transfer" | "">("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const reasonRequired = NEEDS_REASON.includes(action);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (action === "paid" && !method) {
      setError(A.methodPick);
      return;
    }
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
        {/* **والنافذةُ تقول كلَّ ما يلزم القرار** — الدورُ والرقمُ والمحفظةُ وملاحظةُ الطالب.
            (دفعةُ التصميم ١: بلا صندوقٍ داخل النافذة — خطٌّ يفصله عمّا تحته.) */}
        <div className="space-y-3 border-b border-line-soft pb-4">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <div className="min-w-0">
              <p className="font-bold">{payout.user_name || payout.user_phone}</p>
              {payout.user_name && payout.user_name !== payout.user_phone && (
                <p className="text-sm text-ink-muted" dir="ltr">
                  {payout.user_phone}
                </p>
              )}
            </div>
            {payout.user_role && <Badge variant="neutral">{roleLabelByCode(payout.user_role)}</Badge>}
          </div>
          <p className="admin-figure">
            <Money value={payout.amount} />
          </p>
          {!CLOSED.includes(payout.status) && (
            <dl className="grid grid-cols-2 gap-3 sm:grid-cols-3">
              <Fact label={m.terms.walletAvailable}>
                <span className="tabular-nums">{fmtMoney(payout.available)}</span>
              </Fact>
              <Fact label={m.terms.walletBalance}>
                <span className="tabular-nums">{fmtMoney(payout.balance)}</span>
              </Fact>
              <Fact label={m.terms.walletReserved}>
                <span className="tabular-nums">{fmtMoney(payout.reserved)}</span>
              </Fact>
            </dl>
          )}
          {payout.note && (
            <dl>
              <Fact label={A.requesterNote}>{payout.note}</Fact>
            </dl>
          )}
        </div>
        {action === "paid" && (
          <fieldset className="space-y-2">
            <legend className="mb-1 text-sm font-medium">{A.methodLabel}</legend>
            {/* **زرّان من العُدّة لا أزرارٌ مبنيّةٌ باليد** — وأثرُ النقد يقوله سطرُ الأثر تحت. */}
            <div className="grid grid-cols-2 gap-2">
              {(["cash", "transfer"] as const).map((v) => (
                <Button
                  key={v}
                  type="button"
                  aria-pressed={method === v}
                  variant={method === v ? "primary" : "secondary"}
                  onClick={() => {
                    setMethod(v);
                    setError("");
                  }}
                >
                  {v === "cash" ? A.paidViaCash : A.paidViaTransfer}
                </Button>
              ))}
            </div>
          </fieldset>
        )}
        <Alert tone={impactTone(action)}>
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
