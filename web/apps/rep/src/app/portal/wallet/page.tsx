"use client";

/** كشف عمولات المندوب — حركات محفظته مع كل طلب مُسلَّم. */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconWallet } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");
const KIND_LABELS: Record<string, string> = m.admin.users.txKinds;

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

  if (!txs) {
    return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;
  }

  return (
    <div className="space-y-4">
      <h1 className="flex items-center gap-2 text-lg font-bold">
        <IconWallet size={20} className="text-primary" />
        {m.rep.walletTitle}
      </h1>

      {txs.length === 0 ? (
        <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-ink-muted">
          {m.rep.walletEmpty}
        </p>
      ) : (
        <ul className="space-y-2">
          {txs.map((tx) => (
            <li
              key={tx.id}
              className="flex items-center gap-3 rounded-card border border-line bg-surface p-3 text-sm"
            >
              <div className="min-w-0 flex-1">
                <p className="font-medium">{KIND_LABELS[tx.kind] ?? tx.kind}</p>
                {tx.note && <p className="truncate text-xs text-ink-muted">{tx.note}</p>}
              </div>
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
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
