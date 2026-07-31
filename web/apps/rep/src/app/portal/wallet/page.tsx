"use client";

/**
 * المحفظة — الرصيد وحركاته **مبوّبةً بنوعها**، وطلب السحب.
 *
 * كانت الحركات قائمة واحدة يختلط فيها كل شيء: عمولةٌ فوق تسويةٍ فوق سحبٍ فوق
 * تعويض. والمندوب لا يسأل «ماذا جرى في محفظتي؟» بل يسأل سؤالاً محدّداً: كم
 * عمولةً قبضت؟ أين ذهب المبلغ الذي سحبته؟ ما هذه التسوية؟ فصار لكل سؤالٍ تبويبه،
 * وفوقه مجموعُه — الرقم الذي يبحث عنه أصلاً.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  StatementSheet,
  currentMonthRange,
  type StatementData,
  Input,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ListRow,
  Card,
  Tabs,
  type TabItem,
  useLiveData,
  IconWallet,
  IconWarning,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const P = m.shared.payout;
const W = m.shared.walletTabs;
const KIND_LABELS: Record<string, string> = m.shared.txKinds;
const KIND_HINTS: Record<string, string> = m.shared.walletKindHints;

/** ترتيب التبويبات: الأهمّ للمندوب أولاً، لا ترتيب ورودها في القاعدة. */
const KIND_ORDER = [
  "commission",
  "payout",
  "compensation",
  "adjustment",
  "refund",
  "topup",
  "order_payment",
];

const ALL = "all";
const REQUESTS = "requests";
const STATEMENT = "statement";

interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
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
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key.split(".").pop() ?? "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export default function WalletPage() {
  const { user } = useAuth();
  const [asking, setAsking] = useState(false);
  const [tab, setTab] = useState(ALL);

  // كشف الحساب يُجلب بمداه الخاص — لا يشتقّ من لمحة اللوحة (آخر 50) وإلا
  // كان مستنداً مالياً ناقصاً بصمت.
  const [range, setRange] = useState(currentMonthRange());
  const [sheet, setSheet] = useState<StatementData | null>(null);
  const [sheetLoading, setSheetLoading] = useState(false);

  useEffect(() => {
    if (tab !== STATEMENT) return;
    setSheetLoading(true);
    api<StatementData>(`/api/v1/rep/wallet?from=${range.from}&to=${range.to}`)
      .then(setSheet)
      .catch(() => setSheet(null))
      .finally(() => setSheetLoading(false));
  }, [tab, range]);

  const {
    data: statement,
    loading,
    reload,
  } = useLiveData<{ balance: number; transactions: Tx[] }>(() => api("/api/v1/rep/wallet"), [
    "wallet",
  ]);
  const { data: payouts, reload: reloadPayouts } = useLiveData<Payout[]>(
    () => api("/api/v1/me/payouts"),
    ["wallet"],
  );

  const refreshAll = useCallback(() => {
    reload();
    reloadPayouts();
  }, [reload, reloadPayouts]);

  const txs = useMemo(() => statement?.transactions ?? [], [statement]);
  const requests = useMemo(() => payouts ?? [], [payouts]);

  // تبويبات الأنواع الموجودة فعلاً فقط: تبويبٌ فارغ يَعِد بشيء ثم يخذل.
  const tabs = useMemo<TabItem[]>(() => {
    const counts = new Map<string, number>();
    for (const t of txs) counts.set(t.kind, (counts.get(t.kind) ?? 0) + 1);
    const present = KIND_ORDER.filter((k) => counts.has(k));
    for (const k of counts.keys()) if (!present.includes(k)) present.push(k);
    return [
      { key: ALL, label: W.all, count: txs.length },
      ...(requests.length ? [{ key: REQUESTS, label: W.requests, count: requests.length }] : []),
      ...present.map((k) => ({ key: k, label: KIND_LABELS[k] ?? k, count: counts.get(k) })),
      { key: STATEMENT, label: m.shared.statement.open },
    ];
  }, [txs, requests]);

  if (loading) return <LoadingState />;

  const balance = statement?.balance ?? 0;
  const hasPending = requests.some((p) => p.status === "pending");

  // التبويب المختار قد يختفي بعد تحديث حيّ (آخر حركة من نوعه أُلغيت) — نرتدّ للكل
  const current = tabs.some((t) => t.key === tab) ? tab : ALL;
  const shown = current === ALL ? txs : txs.filter((t) => t.kind === current);
  const total = shown.reduce((s, t) => s + t.amount, 0);

  return (
    <PageContainer>
      <PageHeader
        icon={IconWallet}
        title={m.terms.wallet}
        actions={
          <Button onClick={() => setAsking(true)} disabled={balance <= 0 || hasPending}>
            {P.request}
          </Button>
        }
      />

      {/* الرصيد — الرقم الذي يهمّ المندوب أولاً */}
      <div className="rounded-card bg-primary p-6 text-center text-white">
        <p className="text-sm opacity-80">{P.available}</p>
        <p className="mt-1 text-3xl font-bold" dir="ltr">
          {fmtNum(balance)} <span className="text-base font-normal">{m.common.currency}</span>
        </p>
        <p className="mt-2 text-xs opacity-70">{P.hint}</p>
      </div>

      <Card title={m.terms.transactions} icon={IconWallet}>
        <Tabs items={tabs} active={current} onChange={setTab} className="mb-3" />

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
            holderName={user?.full_name}
            holderPhone={user?.phone}
          />
        ) : current === REQUESTS ? (
          <ul className="space-y-2">
            {requests.map((p) => (
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
          <>
            {/* مجموع التبويب — سؤال المندوب الحقيقي: «كم قبضت؟» لا «كم حركة؟» */}
            <div className="mb-3 flex items-center justify-between gap-3 rounded-control bg-page px-3 py-2 text-sm">
              <span className="text-ink-muted">
                {current === ALL ? W.netAll : W.netOf.replace("{kind}", KIND_LABELS[current] ?? current)}
              </span>
              <span
                className={`font-bold ${total >= 0 ? "text-success" : "text-danger"}`}
                dir="ltr"
              >
                {total >= 0 ? "+" : ""}
                {fmtNum(total)} {m.common.currency}
              </span>
            </div>

            <ul className="space-y-2">
              {shown.map((tx) => (
                <ListRow
                  key={tx.id}
                  title={KIND_LABELS[tx.kind] ?? tx.kind}
                  subtitle={tx.note || undefined}
                  trailing={
                    <>
                      <span
                        className={`font-bold ${tx.amount >= 0 ? "text-success" : "text-danger"}`}
                        dir="ltr"
                      >
                        {tx.amount >= 0 ? "+" : ""}
                        {fmtNum(tx.amount)}
                      </span>
                      <span className="text-xs text-ink-muted" dir="ltr">
                        {fmtDate(tx.created_at)}
                      </span>
                    </>
                  }
                />
              ))}
            </ul>
          </>
        )}
      </Card>

      {asking && (
        <PayoutModal
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
  balance,
  onClose,
  onDone,
}: {
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
