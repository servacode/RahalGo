"use client";

/**
 * صفحة المحفظة — **نسخة واحدة مركزية** لكل من له رصيد.
 *
 * كانت مبنيّةً مرّتين: مرّة في بوابة المندوب ومرّة في موقع الزبون — 574 سطراً
 * لمنطقٍ واحد. وكل تحسين كان يُنفَّذ مرّتين أو يُنسى في إحداهما، وهذا كيف تنشأ
 * الفروق التي لا يقصدها أحد. (GROUND-RULES §1.2: يُبنى مرة هنا ويرثه الجميع.)
 *
 * ما يختلف بين الأدوار **مُعامِلات لا نسخ**: مسار النقطة، وترتيب التبويبات
 * (المندوب يبدأ بالعمولة، والزبون بالشحن)، ونصّ التنبيه، وهل يملك طلب سحب.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtTime } from "@rahalgo/i18n";
import { Badge, Button, Input, Modal } from "./components";
import { PageContainer, PageHeader, Card, EmptyState, LoadingState, ListRow, TabCards } from "./layout";
import type { TabItem } from "./layout";
import { StatementSheet, currentMonthRange, type StatementData } from "./Statement";
import { useLiveData } from "./Notifications";
import {
  IconArrowIn,
  IconArrowOut, IconWallet, IconWarning, IconPrint } from "./icons";

const m = getMessages(defaultLocale);
const P = m.shared.payout;
const W = m.shared.walletTabs;
const KIND_LABELS: Record<string, string> = m.shared.txKinds;
const KIND_HINTS: Record<string, string> = m.shared.walletKindHints;

const ALL = "all";
const REQUESTS = "requests";
const STATEMENT = "statement";

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

interface Tx {
  id: string | number;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
  /**
   * **رقمُ الطلب — يُرسله الخادمُ وكانت الواجهةُ ترميه.**
   *
   * فتعرض شظيّةَ المعرّف من نصّ الملاحظة: «دفع طلب ‎#1d448c90» — وهي حروفٌ
   * لا يعرفها صاحبُ المحفظة ولا يجدها في شيء. **ورقمٌ لا يُبحث به ليس
   * رقماً، هو ضجيج.**
   */
  order_number?: number | null;
  ticket_number?: number | null;
  /** منفّذُ الحركة — يُعرض للحركات اليدوية وحدها */
  by_name?: string | null;
}

interface Payout {
  id: string;
  amount: number;
  status: "pending" | "paid" | "rejected";
  note: string;
  decision: string;
  created_at: string;
}

const STATUS_VARIANT: Record<Payout["status"], "warning" | "success" | "danger"> = {
  pending: "warning",
  paid: "success",
  rejected: "danger",
};

function errText(err: unknown): string {
  const key =
    typeof err === "object" && err && "body" in err
      ? ((err as { body?: { message_key?: string } }).body?.message_key ?? "").split(".").pop() ?? ""
      : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

/**
 * بطاقةُ حركةٍ واحدة.
 *
 * كان السطرُ يقول نوعَ الحركة والمبلغَ والتاريخ — **ويسكت عن سببها**. فيرى
 * صاحبُ المحفظة «دفع طلب ‎−108,000» ولا يعرف أيَّ طلب، ولا متى بالساعة، ولا
 * أيّ حركةٍ هي إن سأل عنها.
 *
 * **وأربعةُ أشياء تجعل الرقمَ مفهوماً:**
 *
 *   - **ما هو**: نوعُ الحركة بلفظٍ عربيّ لا باسم قيدٍ محاسبيّ
 *   - **لماذا**: الطلبُ برقمه — لا بشظيّة معرّفه
 *   - **متى**: اليومَ والساعة. **والساعةُ ليست زينة**: من يرى حركتين في يومٍ
 *     واحد يفرّق بينهما بها وحدها
 *   - **أيُّها**: رقمُ الحركة — وهو ما يُقال للمالية حين يُسأل «أيّ حركة؟»
 *
 * والمبلغُ أكبرُ ما في البطاقة ولونُه يقول اتجاهه قبل أن تُقرأ إشارتُه.
 */
/**
 * **بطاقةُ حركةٍ — كلُّ عنصرٍ في حقلٍ مستقلّ.**
 *
 * كانت سطراً واحداً تتزاحم فيه أربعةُ أخبار: النوعُ والطلبُ والتاريخُ ورقمُ
 * الحركة — **مفصولةً بنقاطٍ صغيرة**. فتُقرأ كتلةً واحدة، **ومن بحث عن تاريخ
 * قرأ رقمَ حركة.**
 *
 * **والمالُ أوّلُ ما يُنظر إليه**: مبلغٌ كبيرٌ بإشارته ولونه، **وسهمٌ يقول
 * داخلٌ أم خارج** — فاللونُ وحدَه لا يكفي لمن لا يميّزه.
 */
function TxCard({ tx }: { tx: Tx }) {
  const positive = tx.amount >= 0;
  const T = m.shared.txCard;
  return (
    <li className="rounded-card border border-line bg-surface p-4 transition-colors hover:border-primary/40">
      {/* ── الترويسة: الاتجاهُ · النوعُ · المبلغ ────────────────────── */}
      <div className="flex items-center gap-3">
        <span
          className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-control ${
            positive ? "bg-success/10 text-success" : "bg-danger/10 text-danger"
          }`}
          aria-hidden
        >
          {positive ? <IconArrowIn size={18} /> : <IconArrowOut size={18} />}
        </span>
        <p className="min-w-0 flex-1 truncate font-bold">{KIND_LABELS[tx.kind] ?? tx.kind}</p>
        <span
          className={`shrink-0 text-lg font-bold tabular-nums ${
            positive ? "text-success" : "text-danger"
          }`}
          dir="ltr"
        >
          {positive ? "+" : "−"}
          {fmtNum(Math.abs(tx.amount))}{" "}
          <span className="text-xs font-normal opacity-70">{m.common.currency}</span>
        </span>
      </div>

      {/* ── حقولٌ مُعنوَنة: متى · على ماذا · رقمُ الحركة ──────────────── */}
      <div className="mt-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
        <div className="rounded-control bg-page/70 px-2.5 py-1.5">
          <p className="text-2xs text-ink-muted">{T.when}</p>
          <p className="mt-0.5 text-xs font-medium tabular-nums" dir="ltr">
            {fmtDate(tx.created_at)} · {fmtTime(tx.created_at)}
          </p>
        </div>
        <div className="rounded-control bg-page/70 px-2.5 py-1.5">
          <p className="text-2xs text-ink-muted">{T.about}</p>
          <p className="mt-0.5 text-xs font-medium tabular-nums" dir="ltr">
            {tx.order_number != null
              ? `${T.order} #${fmtNum(tx.order_number)}`
              : tx.ticket_number != null
                ? `${T.ticket} #${fmtNum(tx.ticket_number)}`
                : "—"}
          </p>
        </div>
        <div className="rounded-control bg-page/70 px-2.5 py-1.5">
          <p className="text-2xs text-ink-muted">{T.txNo}</p>
          <p className="mt-0.5 text-xs font-medium tabular-nums" dir="ltr">
            {fmtNum(Number(tx.id))}
          </p>
        </div>
      </div>

      {/* **الملاحظةُ حقلٌ قائمٌ بذاته لا ذيلٌ مقتطع**: هي غالباً سببُ حركةٍ
          يدوية — «تعويض عن طلبٍ فشل» — وقطعُها يُبقي السؤال. */}
      {tx.note && (
        <p className="mt-2 rounded-control border border-line px-3 py-2 text-xs leading-relaxed text-ink-muted">
          {tx.note}
          {tx.by_name && <span className="opacity-70"> — {tx.by_name}</span>}
        </p>
      )}
    </li>
  );
}

export function WalletPage({
  api,
  path,
  kindOrder,
  balanceLabel,
  hint,
  payouts = false,
  holderName,
  holderPhone,
  width = "full",
}: {
  api: ApiFn;
  /** نقطة كشف المحفظة — تختلف بالدور: `/api/v1/rep/wallet` أو `/api/v1/my/wallet` */
  path: string;
  /** ترتيب التبويبات بأهمّيتها لهذا الدور، لا بورودها في القاعدة */
  kindOrder: string[];
  balanceLabel: string;
  hint: string;
  /** هل يملك صاحب الحساب طلب سحب رصيده (المندوب نعم، الزبون لا) */
  payouts?: boolean;
  holderName?: string;
  holderPhone?: string;
  width?: "full" | "medium";
}) {
  const [tab, setTab] = useState(ALL);
  const [asking, setAsking] = useState(false);

  const { data: statement, loading, reload } = useLiveData<{ balance: number; transactions: Tx[] }>(
    () => api(path),
    ["wallet"],
  );
  const { data: requests, reload: reloadPayouts } = useLiveData<Payout[]>(
    () => (payouts ? api("/api/v1/me/payouts") : Promise.resolve([] as Payout[])),
    ["wallet"],
  );

  // كشف الحساب يُجلب بمداه الخاص — لا يشتقّ من لمحة الصفحة (آخر 50) وإلا كان
  // مستنداً مالياً ناقصاً بصمت.
  const [range, setRange] = useState(currentMonthRange());
  const [sheet, setSheet] = useState<StatementData | null>(null);
  const [sheetLoading, setSheetLoading] = useState(false);

  useEffect(() => {
    if (tab !== STATEMENT) return;
    setSheetLoading(true);
    api<StatementData>(`${path}?from=${range.from}&to=${range.to}`)
      .then(setSheet)
      .catch(() => setSheet(null))
      .finally(() => setSheetLoading(false));
  }, [api, path, tab, range]);

  const refreshAll = useCallback(() => {
    reload();
    reloadPayouts();
  }, [reload, reloadPayouts]);

  const txs = useMemo(() => statement?.transactions ?? [], [statement]);
  const reqs = useMemo(() => requests ?? [], [requests]);

  const tabs = useMemo<TabItem[]>(() => {
    // العدّ والمجموع معاً: البطاقة تعرض «كم مرة» و«كم مبلغاً» في نظرة واحدة
    const agg = new Map<string, { n: number; sum: number }>();
    for (const t of txs) {
      const a = agg.get(t.kind) ?? { n: 0, sum: 0 };
      agg.set(t.kind, { n: a.n + 1, sum: a.sum + t.amount });
    }
    // الأنواع الموجودة فعلاً فقط: تبويبٌ فارغ يَعِد بشيء ثم يخذل
    const present = kindOrder.filter((k) => agg.has(k));
    for (const k of agg.keys()) if (!present.includes(k)) present.push(k);
    return [
      { key: ALL, label: W.all, count: txs.length, value: txs.reduce((s, t) => s + t.amount, 0) },
      // طلبات السحب بلا مجموع عمداً: خلط المعلّق بالمدفوع بالمرفوض في رقم واحد
      // يعطي مبلغاً لا يعني شيئاً — العدد وحده هو الصادق هنا.
      ...(reqs.length ? [{ key: REQUESTS, label: W.requests, count: reqs.length }] : []),
      ...present.map((k) => ({
        key: k,
        label: KIND_LABELS[k] ?? k,
        count: agg.get(k)?.n,
        value: agg.get(k)?.sum,
      })),
    ];
  }, [txs, reqs, kindOrder]);

  if (loading) return <LoadingState />;

  const balance = statement?.balance ?? 0;
  const hasPending = reqs.some((p) => p.status === "pending");
  // التبويب المختار قد يختفي بعد تحديث حيّ (آخر حركة من نوعه أُلغيت) — نرتدّ للكل
  const current = tab === STATEMENT || tabs.some((t) => t.key === tab) ? tab : ALL;
  const shown = current === ALL ? txs : txs.filter((t) => t.kind === current);

  const statementBtn = (
    <Button
      variant="secondary"
      onClick={() => setTab(tab === STATEMENT ? ALL : STATEMENT)}
      className="flex items-center gap-2"
    >
      <IconPrint size={16} />
      {m.shared.statement.open}
    </Button>
  );

  return (
    <PageContainer width={width}>
      <PageHeader
        icon={IconWallet}
        title={m.terms.wallet}
        actions={
          <>
            {/* كشف الحساب إجراء لا تصنيف: لا رقم له فلا مكان له بين البطاقات */}
            {statementBtn}
            {payouts && (
              <Button onClick={() => setAsking(true)} disabled={balance <= 0 || hasPending}>
                {P.request}
              </Button>
            )}
          </>
        }
      />

      {/* الرصيد — الرقم الذي يهمّ صاحبه أولاً */}
      <div className="rounded-card bg-primary p-6 text-center text-white">
        <p className="text-sm opacity-80">{balanceLabel}</p>
        <p className="mt-1 text-3xl font-bold" dir="ltr">
          {fmtNum(balance)} <span className="text-base font-normal">{m.common.currency}</span>
        </p>
        <p className="mx-auto mt-2 max-w-lg text-xs leading-relaxed opacity-70">{hint}</p>
      </div>

      <Card title={m.terms.transactions} icon={IconWallet}>
        {current !== STATEMENT && (
          <TabCards items={tabs} active={current} onChange={setTab} className="mb-4" />
        )}

        {/* شرح النوع: أسماء القيود المحاسبية ليست بديهية لمن لم يكتبها */}
        {current !== STATEMENT && KIND_HINTS[current] && (
          <p className="mb-3 text-xs leading-relaxed text-ink-muted">{KIND_HINTS[current]}</p>
        )}

        {current === STATEMENT ? (
          <StatementSheet
            data={sheet}
            loading={sheetLoading}
            from={range.from}
            to={range.to}
            onFrom={(from) => setRange((r) => ({ ...r, from }))}
            onTo={(to) => setRange((r) => ({ ...r, to }))}
            onQuick={(from, to) => setRange({ from, to })}
            holderName={holderName}
            holderPhone={holderPhone}
          />
        ) : current === REQUESTS ? (
          <ul className="space-y-2">
            {reqs.map((p) => (
              <ListRow
                key={p.id}
                title={
                  <span dir="ltr">
                    {fmtNum(p.amount)} {m.common.currency}
                  </span>
                }
                subtitle={p.decision || p.note || undefined}
                trailing={
                  <>
                    <Badge variant={STATUS_VARIANT[p.status]}>{P.status[p.status]}</Badge>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {fmtDate(p.created_at)}
                    </span>
                  </>
                }
              />
            ))}
          </ul>
        ) : shown.length === 0 ? (
          <EmptyState icon={IconWallet} title={m.terms.noTransactions} />
        ) : (
          // لا شريط مجموع هنا: البطاقة النشطة تعرضه فوق — تكراره ضجيج
          <ul className="space-y-2.5">
            {shown.map((tx) => (
              <TxCard key={tx.id} tx={tx} />
            ))}
          </ul>
        )}
      </Card>

      {asking && (
        <PayoutModal
          api={api}
          balance={balance}
          onClose={() => setAsking(false)}
          onDone={() => {
            setAsking(false);
            refreshAll();
          }}
        />
      )}
    </PageContainer>
  );
}

function PayoutModal({
  api,
  balance,
  onClose,
  onDone,
}: {
  api: ApiFn;
  balance: number;
  onClose: () => void;
  onDone: () => void;
}) {
  const [amount, setAmount] = useState(String(balance));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    const value = Number(amount);
    if (!Number.isFinite(value) || value <= 0 || value > balance) {
      return setError(m.errors.insufficient_balance);
    }
    setBusy(true);
    try {
      await api("/api/v1/me/payouts", {
        method: "POST",
        body: JSON.stringify({ amount: value, note }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.title}>
      <form onSubmit={submit} className="space-y-4">
        <div>
          <Input
            id="payout-amount"
            label={P.amount}
            type="number"
            dir="ltr"
            min="1"
            max={balance}
            required
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="text-center text-lg font-bold"
          />
          <button
            type="button"
            onClick={() => setAmount(String(balance))}
            className="mt-1.5 text-sm font-medium text-primary hover:underline"
          >
            {P.all} ({fmtNum(balance)} {m.common.currency})
          </button>
        </div>

        <Input
          id="payout-note"
          label={P.note}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={P.notePlaceholder}
        />

        {error && (
          <p
            role="alert"
            className="flex items-start gap-2 rounded-control border border-danger/25 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            <IconWarning size={16} className="mt-0.5 shrink-0" />
            {error}
          </p>
        )}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {busy ? m.common.loading : P.submit}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
