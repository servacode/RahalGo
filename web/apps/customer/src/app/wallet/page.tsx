"use client";

/** محفظتي: الرصيد والسجل — يعرف الزبون أين صُرفت نقوده (دفع من المحفظة، تعويض،
 *  كوبون/خصم، أو تصحيح خطأ مالي). الزبون لا لوحة له فالسجل هنا ضروري. */

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ListRow,
  Card,
  Button,
  IconPrint,
  TabCards,
  type TabItem,
  StatementSheet,
  currentMonthRange,
  type StatementData,
  useLiveRefresh,
  IconWallet,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);
const W = m.shared.walletTabs;
const KIND_LABELS: Record<string, string> = m.shared.txKinds;
const KIND_HINTS: Record<string, string> = m.shared.walletKindHints;

/** ترتيب التبويبات من منظور الزبون: ماله أولاً، ثم ما صُرف منه. */
const KIND_ORDER = ["topup", "order_payment", "refund", "compensation", "adjustment"];
const ALL = "all";
const STATEMENT = "statement";

interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
}
interface Statement {
  balance: number;
  transactions: Tx[];
}

export default function WalletPage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [st, setSt] = useState<Statement | null>(null);
  const [tab, setTab] = useState(ALL);

  // كشف الحساب بمداه الخاص لا بلمحة الصفحة (آخر 50) — وإلا كان ناقصاً بصمت
  const [range, setRange] = useState(currentMonthRange());
  const [sheet, setSheet] = useState<StatementData | null>(null);
  const [sheetLoading, setSheetLoading] = useState(false);

  useEffect(() => {
    if (tab !== STATEMENT) return;
    setSheetLoading(true);
    api<StatementData>(`/api/v1/my/wallet?from=${range.from}&to=${range.to}`)
      .then(setSheet)
      .catch(() => setSheet(null))
      .finally(() => setSheetLoading(false));
  }, [tab, range]);

  const load = useCallback(() => {
    api<Statement>("/api/v1/my/wallet")
      .then(setSt)
      .catch(() => setSt({ balance: 0, transactions: [] }));
  }, []);

  useEffect(() => {
    if (loading) return;
    if (!isLoggedIn(user)) {
      router.replace("/login?next=/wallet");
      return;
    }
    load();
  }, [user, loading, router, load]);

  useLiveRefresh(["wallet"], load);

  const txs = useMemo(() => st?.transactions ?? [], [st]);

  // تبويبات الأنواع الموجودة فعلاً فقط: تبويبٌ فارغ يَعِد بشيء ثم يخذل.
  const tabs = useMemo<TabItem[]>(() => {
    // العدّ والمجموع معاً: البطاقة تعرض «كم مرة» و«كم مبلغاً» في نظرة واحدة
    const agg = new Map<string, { n: number; sum: number }>();
    for (const t of txs) {
      const a = agg.get(t.kind) ?? { n: 0, sum: 0 };
      agg.set(t.kind, { n: a.n + 1, sum: a.sum + t.amount });
    }
    const present = KIND_ORDER.filter((k) => agg.has(k));
    for (const k of agg.keys()) if (!present.includes(k)) present.push(k);
    return [
      { key: ALL, label: W.all, count: txs.length, value: txs.reduce((s, t) => s + t.amount, 0) },
      ...present.map((k) => ({
        key: k,
        label: KIND_LABELS[k] ?? k,
        count: agg.get(k)?.n,
        value: agg.get(k)?.sum,
      })),
    ];
  }, [txs]);

  if (!st) return <LoadingState />;

  const current = tab === STATEMENT || tabs.some((t) => t.key === tab) ? tab : ALL;
  const shown = current === ALL ? txs : txs.filter((t) => t.kind === current);

  return (
    <PageContainer width="medium">
      <PageHeader
        icon={IconWallet}
        title={m.terms.wallet}
        actions={
          <Button
            variant="secondary"
            onClick={() => setTab(tab === STATEMENT ? ALL : STATEMENT)}
            className="flex items-center gap-2"
          >
            <IconPrint size={16} />
            {m.shared.statement.open}
          </Button>
        }
      />

      <div className="mb-3 rounded-card bg-primary p-6 text-center text-white">
        <p className="text-sm opacity-80">{m.site.wallet.balance}</p>
        <p className="mt-1 text-3xl font-bold">
          {fmtNum(st.balance)} <span className="text-base font-normal">{m.common.currency}</span>
        </p>
      </div>
      <p className="mb-6 rounded-control bg-page px-3 py-2 text-xs leading-relaxed text-ink-muted">
        {m.site.wallet.hint}
      </p>

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
            holderName={user?.full_name}
            holderPhone={user?.phone}
          />
        ) : shown.length === 0 ? (
          <EmptyState icon={IconWallet} title={m.terms.noTransactions} />
        ) : (
          <>
            {/* لا شريط مجموع: البطاقة النشطة تعرضه فوق */}
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
    </PageContainer>
  );
}
