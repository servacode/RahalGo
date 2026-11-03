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
  Button,
  Input,
  Modal,
  Badge,
  AdminPage,
  AdminAsync,
  AdminEntityCard,
  AdminEntityGrid,
  ContactButtons,
  FilterBar,
  FilterSelect,
  StatusBadge,
  SummaryStrip,
  agoText,
  useRememberedFilters,
  useLiveData,
  IconWallet,
  IconUser,
  IconStatus,
  IconWarning,
  Money,
  FormActions,
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
  // **والفلترُ من الرابط يغلب المحفوظَ** — بطاقةُ «بانتظار قرارك» في الرئيسيّة تفتح فلتراً بعينه.
  const [fromUrl] = useState(() => {
    const u: Partial<Record<"status" | "role" | "month", string>> = {};
    if (readParam("status")) u.status = validStatus(readParam("status"));
    if (readParam("role")) u.role = validRole(readParam("role"));
    if (readParam("month") === "1") u.month = "1";
    return u;
  });
  const rem = useRememberedFilters("payouts", { status: "", role: "", month: "" }, fromUrl);
  const status = validStatus(rem.filters.status);
  const role = validRole(rem.filters.role);
  const month = rem.filters.month === "1";
  const [page, setPage] = useState(1);
  const [deciding, setDeciding] = useState<{ p: Payout; action: Action } | null>(null);

  /** **وتغييرُ الفلتر يرجع للصفحة الأولى** — كانت الصفحةُ الثالثةُ تبقى فيظهر «لا طلبات». */
  function setStatus(s: string, onlyMonth = false) {
    rem.set({ status: s, month: onlyMonth ? "1" : "" });
    setPage(1);
  }
  function clearFilters() {
    rem.reset();
    setPage(1);
  }
  function setRole(r: string) {
    rem.set({ role: r });
    setPage(1);
  }

  const qs = new URLSearchParams({ page: String(page) });
  if (status) qs.set("status", status);
  if (role) qs.set("role", role);
  if (month) qs.set("month", "1");

  const { data, loading, error, reload } = useLiveData<PayoutsData | null>(
    // **ولا جلبَ قبل قراءة المحفوظ** — وإلّا جلبنا بالافتراضيّ ثم أعدنا.
    () => (rem.ready ? api<PayoutsData>(`/api/v1/admin/payouts?${qs.toString()}`) : new Promise<null>(() => {})),
    ["wallet"],
    [status, role, month, page, rem.ready],
  );

  const rows = data?.payouts ?? [];
  const st = data?.stats;
  const filtered = status !== "" || role !== "" || month;
  // **رقمُ «بانتظار المالية» للمعلَّق وحدَه** — والخادمُ القديمُ بلا الحقل: الفرق.
  const pendingCount = st ? (st.pending_count ?? st.waiting_count - st.processing_count) : undefined;

  const count = (n: number) => A.cardCount.replace("{n}", fmtNum(n));

  // **الأرقامُ الأربعة** — و«بانتظار الصرف» يشمل «قيد الصرف»: مالٌ محجوزٌ لم يخرج،
  // ويفتح الحالتين معاً · وبطاقتا «هالشهر» تفتحان قرارات الشهر وحدَها.
  // **والألوانُ بمعنى واحد**: أصفرُ ينتظر · أخضرُ تمّ · أحمرُ مشكلة (المرتجعُ) · والباقي رمادي.
  return (
    <AdminPage<string>
      icon={IconWallet}
      title={P.adminTitle}
      tabs={[
        { key: "all", label: A.allStatuses },
        ...TAB_STATUSES.map((s) => ({
          key: s,
          label: A.status[s],
          count: s === "pending" ? pendingCount : s === "processing" ? st?.processing_count : undefined,
          attention: s === "pending" && (pendingCount ?? 0) > 0,
        })),
        { key: FAILED_OR_REVERSED, label: A.failedOrReversed },
        // **«بانتظار الصرف» تبويبٌ يظهر حين تفتحه بطاقتُه** — كي لا يبقى الرأسُ بلا تبويبٍ مختار.
        ...(status === WAITING ? [{ key: WAITING, label: A.waitingTab }] : []),
        // **وكذلك «فشل» أو «مرتجع» وحدَهما من رابطٍ قديم.**
        ...(status === "failed" || status === "reversed" ? [{ key: status, label: A.status[status] }] : []),
      ]}
      tab={status || "all"}
      onTab={(k) => setStatus(k === "all" ? "" : k)}
      notices={
        // **وفشلُ التحديث بعد تغيير الفلتر يُقال** — كانت الكروتُ القديمةُ تبقى بلا كلمة.
        error && data ? <Alert tone="warning">{A.staleData}</Alert> : undefined
      }
      toolbar={
        <FilterBar filtered={filtered} onClear={clearFilters} end={data ? fmtNum(data.total) : undefined}>
          <FilterSelect id="payouts-role" label={A.roleCol} value={role} onChange={setRole}>
            <option value="">{A.allRoles}</option>
            {ROLES.map((r) => (
              <option key={r} value={r}>
                {roleLabelByCode(r)}
              </option>
            ))}
          </FilterSelect>
          {/* **وحصرُ الشهر يُقال** — كي لا يظنّ القارئُ أنّ ما يراه كلُّ ما صُرف. */}
          {month && (
            <Button variant="secondary" onClick={() => setStatus(status, false)}>
              {A.monthOnly} · {A.allMonths}
            </Button>
          )}
        </FilterBar>
      }
      summary={
        <SummaryStrip
          items={[
            {
              key: "waiting",
              icon: IconWallet,
              label: A.cardWaiting,
              value: fmtMoney(st?.waiting_sum ?? 0),
              meta:
                (st?.processing_count ?? 0) > 0
                  ? A.waitingSplit
                      .replace("{a}", fmtNum(pendingCount ?? 0))
                      .replace("{b}", fmtNum(st?.processing_count ?? 0))
                  : count(st?.waiting_count ?? 0),
              tone: (st?.waiting_count ?? 0) > 0 ? "attention" : "muted",
              onOpen: () => setStatus(WAITING),
            },
            {
              key: "processing",
              icon: IconStatus,
              label: A.cardProcessing,
              value: fmtMoney(st?.processing_sum ?? 0),
              meta: count(st?.processing_count ?? 0),
              tone: (st?.processing_count ?? 0) > 0 ? "neutral" : "muted",
              onOpen: () => setStatus("processing"),
            },
            {
              key: "paid",
              icon: IconWallet,
              label: A.cardPaidMonth,
              value: fmtMoney(st?.paid_month_sum ?? 0),
              meta: count(st?.paid_month_count ?? 0),
              tone: (st?.paid_month_count ?? 0) > 0 ? "good" : "muted",
              onOpen: () => setStatus("paid", true),
            },
            {
              key: "returned",
              icon: IconWarning,
              label: A.cardReturnedMonth,
              value: fmtMoney(st?.returned_month_sum ?? 0),
              meta: count(st?.returned_month_count ?? 0),
              tone: (st?.returned_month_count ?? 0) > 0 ? "danger" : "muted",
              onOpen: () => setStatus(FAILED_OR_REVERSED, true),
            },
          ]}
        />
      }
      pagination={{
        page,
        total: data?.total ?? 0,
        perPage: data?.per_page ?? 1,
        onChange: setPage,
      }}
    >
      <AdminAsync
        loading={loading}
        error={error}
        onRetry={reload}
        hasData={!!data}
        empty={rows.length === 0}
        emptyTitle={emptyTitle(status, filtered)}
        emptyIcon={IconWallet}
        emptyAction={
          filtered ? (
            <Button variant="secondary" onClick={clearFilters}>
              {A.clearFilter}
            </Button>
          ) : undefined
        }
      >
        <AdminEntityGrid>
          {rows.map((p) => (
            <PayoutCard
              key={p.id}
              p={p}
              canDecide={canDecide}
              own={p.user_id === meId && !owner}
              onAct={(a) => setDeciding({ p, action: a })}
            />
          ))}
        </AdminEntityGrid>
      </AdminAsync>

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

interface PayoutsData {
  payouts: Payout[];
  total: number;
  per_page: number;
  pending_total: number;
  stats: Stats;
}

/**
 * **كرتُ طلب سحب** — مين طلب (اسمُه رابطٌ لحسابه)، قديش، شو متاح بمحفظته، ملاحظتُه وقرارُ الماليّة،
 * ومين قرّر — وأفعالُه: فعلان ظاهران على الأكثر، و«ارتدّ السحب» (الخطِر) تحت «⋯» بتأكيد.
 */
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
  const visible: Action[] = acts.filter((a) => a !== "reversed").slice(0, 2);
  const more = acts
    .filter((a) => !visible.includes(a))
    .map((a) => ({
      key: a,
      label: actionLabel(a, p.status),
      danger: NEEDS_REASON.includes(a),
      onSelect: () => onAct(a),
    }));
  const name = p.user_name || p.user_phone;
  const tone =
    short || p.status === "failed"
      ? "danger"
      : p.status === "pending" || p.status === "processing"
        ? "attention"
        : p.status === "paid"
          ? "good"
          : "neutral";
  return (
    <AdminEntityCard
      tone={tone}
      leading={<IconUser size={20} className="text-ink-muted" />}
      eyebrow={
        <>
          {p.user_role ? `${roleLabelByCode(p.user_role)} · ` : ""}
          <span title={fmtDateTime(p.created_at)}>{agoText(p.created_at)}</span>
        </>
      }
      title={name}
      href={`/dashboard/users/${p.user_id}`}
      status={<StatusBadge domain="payout" status={p.status} />}
      facts={[
        { key: "amount", label: P.amount, value: fmtMoney(p.amount) },
        { key: "date", label: m.terms.date, value: <span dir="ltr">{fmtDateTime(p.created_at)}</span> },
        ...(!closed
          ? [
              {
                key: "avail",
                label: m.terms.walletAvailable,
                value: <span className={short ? "text-danger" : ""}>{fmtMoney(p.available)}</span>,
                href: `/dashboard/users/${p.user_id}?tab=wallet`,
              },
              {
                key: "bal",
                label: `${m.terms.walletBalance} · ${m.terms.walletReserved}`,
                value: `${fmtMoney(p.balance)} · ${fmtMoney(p.reserved)}`,
              },
            ]
          : [
              ...(p.paid_via
                ? [{ key: "method", label: A.methodLabel, value: p.paid_via === "cash" ? A.paidViaCash : A.paidViaTransfer }]
                : []),
              ...(p.decided_at
                ? [
                    {
                      key: "decided",
                      label: A.decidedCol,
                      value: `${p.decided_by_name || "—"} · ${fmtDateTime(p.decided_at)}`,
                      wide: true,
                    },
                  ]
                : []),
            ]),
      ]}
      primary={
        visible.length > 0 ? (
          <>
            {visible.map((a) => (
              <ActButton key={a} a={a} from={p.status} onAct={onAct} />
            ))}
          </>
        ) : undefined
      }
      more={more}
    >
      {short && <Alert tone="error">{A.shortfall}</Alert>}
      {/* **ملاحظةُ الطالب وقرارُ المالية سطران** — كانت الأولى تختفي بعد القرار. */}
      {(p.note || p.decision) && (
        <div className="space-y-1 text-sm">
          {p.note && (
            <p className="border-s-2 border-line ps-2">
              <span className="text-xs text-ink-muted">{A.requesterNote}: </span>
              {p.note}
            </p>
          )}
          {p.decision && (
            <p className="border-s-2 border-line ps-2">
              <span className="text-xs text-ink-muted">{A.financeNote}: </span>
              {p.decision}
            </p>
          )}
        </div>
      )}
      <ContactButtons phone={p.user_phone} name={name} showNumber />
      {own && canDecide && ACTIONS[p.status].length > 0 && <p className="text-sm text-ink-muted">{A.ownRequest}</p>}
    </AdminEntityCard>
  );
}

function ActButton({ a, from, onAct }: { a: Action; from: Status; onAct: (a: Action) => void }) {
  return (
    <Button
      size="sm"
      variant={a === "paid" ? "primary" : NEEDS_REASON.includes(a) ? "danger" : "secondary"}
      onClick={() => onAct(a)}
    >
      {actionLabel(a, from)}
    </Button>
  );
}

/** **لونُ الأثر** — مالٌ يخرج أو يرتدّ أصفر، ومالٌ يرجع لصاحبه أو يبقى محجوزاً معلومة. */
const IMPACT_TONE: Record<Action, "warning" | "info"> = {
  paid: "warning",
  reversed: "warning",
  processing: "info",
  rejected: "info",
  failed: "info",
};

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
        {/* **والنافذةُ تقول كلَّ ما يلزم القرار** — الدورُ والرقمُ والمحفظةُ وملاحظةُ الطالب. */}
        <div className="space-y-1 rounded-control border border-line bg-field px-3 py-2 text-sm">
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="font-bold">{payout.user_name || payout.user_phone}</span>
            {payout.user_role && <Badge variant="neutral">{roleLabelByCode(payout.user_role)}</Badge>}
          </div>
          {payout.user_name && payout.user_name !== payout.user_phone && (
            <p className="text-2xs text-ink-muted" dir="ltr">
              {payout.user_phone}
            </p>
          )}
          <p className="figure">
            <Money value={payout.amount} />
          </p>
          {!CLOSED.includes(payout.status) && (
            <p className="text-2xs text-ink-muted">
              {A.walletNow}: {m.terms.walletBalance} {fmtMoney(payout.balance)} · {m.terms.walletReserved}{" "}
              {fmtMoney(payout.reserved)} · {m.terms.walletAvailable} {fmtMoney(payout.available)}
            </p>
          )}
          {payout.note && (
            <p className="border-s-2 border-line ps-2">
              <span className="text-2xs text-ink-muted">{A.requesterNote}: </span>
              {payout.note}
            </p>
          )}
        </div>
        {action === "paid" && (
          <fieldset className="space-y-2">
            <legend className="mb-1 text-sm font-medium">{A.methodLabel}</legend>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              {(["cash", "transfer"] as const).map((v) => (
                <button
                  key={v}
                  type="button"
                  aria-pressed={method === v}
                  onClick={() => {
                    setMethod(v);
                    setError("");
                  }}
                  className={`rounded-control border px-3 py-2.5 text-start transition-colors ${
                    method === v ? "border-primary bg-primary-tint" : "border-line bg-surface hover:bg-row-hover"
                  }`}
                >
                  <span className="block text-sm font-bold">{v === "cash" ? A.paidViaCash : A.paidViaTransfer}</span>
                  <span className="block text-2xs text-ink-muted">
                    {v === "cash" ? A.methodCashHint : A.methodTransferHint}
                  </span>
                </button>
              ))}
            </div>
          </fieldset>
        )}
        <Alert tone={IMPACT_TONE[action]}>
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
