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
  Chips,
  Input,
  Radio,
  Modal,
  PageContainer,
  LoadingState,
  ReloadState,
  Pagination,
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
import { roleLabelByCode } from "@/lib/rolemeta";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const P = m.shared.payout;
const A = m.shared.payoutAdmin;

type Status = "pending" | "processing" | "paid" | "rejected" | "failed" | "reversed";
/** **«مرتجع أو فاشل» فلترٌ واحد** — يفهمه الخادمُ (`payoutFailedOrReversed`). */
const FAILED_OR_REVERSED = "failed_or_reversed";
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
  // **وقرارُ السحب قدرةٌ بذاتها** — `payouts.decide`.
  const canDecide = can("payouts.decide");
  // **والفلترُ من الرابط عند الفتح** — بطاقةُ «بانتظار قرارك» في الرئيسيّة.
  const [status, setStatusRaw] = useState(() => readParam("status"));
  const [role, setRoleRaw] = useState(() => readParam("role"));
  const [page, setPage] = useState(1);
  const [deciding, setDeciding] = useState<{ p: Payout; action: Action } | null>(null);

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

  const roleChips = [
    { id: "", label: A.allRoles },
    ...ROLES.map((r) => ({ id: r, label: roleLabelByCode(r) })),
  ];

  const count = (n: number) => A.cardCount.replace("{n}", fmtNum(n));

  // **تصميمٌ احترافيّ بلا جداول** (المالك ٢٠٢٦-١٠-٢٦: «امسح العرض بالجداول وتصميم احترافي») —
  // رأسٌ موحّدٌ بتبويبات الحالات، أربعةُ أرقامٍ تُضغط، وكرتٌ لكلّ طلبِ سحبٍ بأفعاله.
  return (
    <PageContainer>
      <PageHero<string>
        icon={IconWallet}
        title={P.adminTitle}
        subtitle={A.pageHint}
        tabs={[
          { key: "all", label: A.allStatuses },
          ...STATUSES.map((s) => ({
            key: s,
            label: A.status[s],
            count: s === "pending" ? st?.waiting_count : s === "processing" ? st?.processing_count : undefined,
            warn: s === "pending" && (st?.waiting_count ?? 0) > 0,
          })),
          { key: FAILED_OR_REVERSED, label: A.failedOrReversed },
        ]}
        value={status || "all"}
        onChange={(k) => setStatus(k === "all" ? "" : k)}
      />

      {/* **الأرقامُ الأربعة** — و«بانتظار الصرف» يشمل «قيد الصرف»: مالٌ محجوزٌ لم يخرج. */}
      <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-4">
        <Tile
          icon={IconWallet}
          label={A.cardWaiting}
          value={fmtMoney(st?.waiting_sum ?? 0)}
          sub={count(st?.waiting_count ?? 0)}
          tone={(st?.waiting_count ?? 0) > 0 ? "warning" : "muted"}
          onClick={() => setStatus("pending")}
        />
        <Tile
          icon={IconStatus}
          label={A.cardProcessing}
          value={fmtMoney(st?.processing_sum ?? 0)}
          sub={count(st?.processing_count ?? 0)}
          tone={(st?.processing_count ?? 0) > 0 ? "accent" : "muted"}
          onClick={() => setStatus("processing")}
        />
        <Tile
          icon={IconWallet}
          label={A.cardPaidMonth}
          value={fmtMoney(st?.paid_month_sum ?? 0)}
          sub={count(st?.paid_month_count ?? 0)}
          tone={(st?.paid_month_count ?? 0) > 0 ? "success" : "muted"}
          onClick={() => setStatus("paid")}
        />
        <Tile
          icon={IconWarning}
          label={A.cardReturnedMonth}
          value={fmtMoney(st?.returned_month_sum ?? 0)}
          sub={count(st?.returned_month_count ?? 0)}
          tone={(st?.returned_month_count ?? 0) > 0 ? "danger" : "muted"}
          onClick={() => setStatus(FAILED_OR_REVERSED)}
        />
      </div>

      <section className="surface mb-4 flex flex-wrap items-center gap-2 p-3">
        <span className="text-sm text-ink-muted">{A.roleCol}:</span>
        <Chips items={roleChips} value={role} onChange={setRole} wrap />
      </section>

      {error && !data ? (
        <ReloadState onRetry={reload} label={A.loadError} />
      ) : rows.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
            <IconWallet size={28} />
          </span>
          <h2 className="heading-section">{filtered ? A.emptyFiltered : A.emptyAll}</h2>
          <p className="max-w-md text-sm text-ink-muted">{A.emptyHint}</p>
        </section>
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {rows.map((p) => (
            <PayoutCard key={p.id} p={p} canDecide={canDecide} onAct={(a) => setDeciding({ p, action: a })} />
          ))}
        </ul>
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

const STATUS_TINT: Record<Status, string> = {
  pending: "bg-warning-solid text-on-solid",
  processing: "bg-accent text-on-bright",
  paid: "bg-success-solid text-on-solid",
  rejected: "bg-field text-ink-muted",
  failed: "bg-danger-solid text-on-solid",
  reversed: "bg-field text-ink",
};

/** **كرتُ طلب سحب** — مين طلب، قديش، شو متاح بمحفظته، ملاحظتُه وقرارُ الماليّة، ومين قرّر — وأفعالُه. */
function PayoutCard({ p, canDecide, onAct }: { p: Payout; canDecide: boolean; onAct: (a: Action) => void }) {
  const acts = canDecide ? ACTIONS[p.status] : [];
  return (
    <li className="surface flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <span className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${STATUS_TINT[p.status]}`}>
          <IconUser size={20} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold">{p.user_name || p.user_phone}</span>
          <span className="block text-2xs text-ink-muted" dir="ltr">
            {p.user_phone}
          </span>
        </span>
        <span className="flex shrink-0 flex-col items-end gap-1">
          <Badge variant={VARIANT[p.status]}>{A.status[p.status]}</Badge>
          {p.user_role && <Badge variant="neutral">{roleLabelByCode(p.user_role)}</Badge>}
        </span>
      </div>

      <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
        <span>
          <span className="block text-2xs text-ink-muted">{P.amount}</span>
          <span className="figure block">{fmtMoney(p.amount)}</span>
          {p.paid_via && (
            <span className="block text-2xs text-ink-muted">{p.paid_via === "cash" ? A.paidViaCash : A.paidViaTransfer}</span>
          )}
        </span>
        {/* **والمحجوزُ والمتاحُ يُعرَضان** — `XG-12`: الماليّةُ تقرّر على ما يجوز صرفُه. */}
        <span className="text-end text-sm">
          <span className="block text-2xs text-ink-muted">{m.terms.walletAvailable}</span>
          <b className={`tabular-nums ${p.available < p.amount && p.status === "pending" ? "text-danger" : ""}`}>
            {fmtMoney(p.available)}
          </b>
          <span className="block text-2xs text-ink-muted">
            {m.terms.walletBalance} {fmtNum(p.balance)} · {m.terms.walletReserved} {fmtNum(p.reserved)}
          </span>
        </span>
      </div>

      {/* **ملاحظةُ الطالب وقرارُ المالية سطران** — كانت الأولى تختفي بعد القرار. */}
      {(p.note || p.decision) && (
        <div className="space-y-1 text-sm">
          {p.note && (
            <p className="border-s-2 border-line ps-2">
              <span className="text-2xs text-ink-muted">{A.requesterNote}: </span>
              {p.note}
            </p>
          )}
          {p.decision && (
            <p className="border-s-2 border-line ps-2">
              <span className="text-2xs text-ink-muted">{A.financeNote}: </span>
              {p.decision}
            </p>
          )}
        </div>
      )}

      <div className="flex flex-wrap items-center justify-between gap-2 text-2xs text-ink-muted">
        <span>
          {m.terms.date}: <span dir="ltr">{fmtDateTime(p.created_at)}</span>
        </span>
        {p.decided_at && (
          <span>
            {A.decidedCol}: {p.decided_by_name || "—"} · <span dir="ltr">{fmtDateTime(p.decided_at)}</span>
          </span>
        )}
      </div>

      {acts.length > 0 && (
        <div className="mt-auto grid grid-cols-1 gap-2 border-t border-line-soft pt-3 sm:grid-cols-3">
          {acts.map((a) => (
            <Button
              key={a}
              className="w-full"
              variant={a === "paid" ? "primary" : NEEDS_REASON.includes(a) ? "danger" : "secondary"}
              onClick={() => onAct(a)}
            >
              {actionLabel(a, p.status)}
            </Button>
          ))}
        </div>
      )}
    </li>
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
