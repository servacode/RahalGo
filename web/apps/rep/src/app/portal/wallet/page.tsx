"use client";

/** المحفظة — حركات عمولات المندوب. */

import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ListRow,
  useLiveData,
  IconWallet,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const KIND_LABELS: Record<string, string> = m.shared.txKinds;

interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
}

export default function WalletPage() {
  const { data, loading } = useLiveData<{ transactions: Tx[] }>(
    () => api("/api/v1/rep/wallet"),
    ["wallet"],
  );

  if (loading) return <LoadingState />;
  const txs = data?.transactions ?? [];

  return (
    <PageContainer>
      <PageHeader icon={IconWallet} title={m.terms.wallet} />

      {txs.length === 0 ? (
        <EmptyState icon={IconWallet} title={m.terms.noTransactions} />
      ) : (
        <ul className="space-y-2">
          {txs.map((tx) => (
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
      )}
    </PageContainer>
  );
}
