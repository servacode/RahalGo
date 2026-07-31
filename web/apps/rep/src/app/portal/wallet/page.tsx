"use client";

/** المحفظة — حركات عمولات المندوب. */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ListRow,
  IconWallet,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");
const KIND_LABELS: Record<string, string> = m.shared.txKinds;

interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
}

export default function WalletPage() {
  const [txs, setTxs] = useState<Tx[] | null>(null);

  useEffect(() => {
    api<{ transactions: Tx[] }>("/api/v1/rep/wallet")
      .then((s) => setTxs(s.transactions))
      .catch(() => setTxs([]));
  }, []);

  if (!txs) return <LoadingState />;

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
                    {fmt.format(tx.amount)}
                  </span>
                  <span className="text-xs text-ink-muted" dir="ltr">
                    {new Date(tx.created_at).toLocaleDateString("ar-SY")}
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
