"use client";

/** محفظتي: الرصيد والسجل — يعرف الزبون أين صُرفت نقوده (دفع من المحفظة، تعويض،
 *  كوبون/خصم، أو تصحيح خطأ مالي). الزبون لا لوحة له فالسجل هنا ضروري. */

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconWallet } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

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
interface Statement {
  balance: number;
  transactions: Tx[];
}

export default function WalletPage() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const [st, setSt] = useState<Statement | null>(null);

  useEffect(() => {
    if (loading) return;
    if (!isLoggedIn(user)) {
      router.replace("/login?next=/wallet");
      return;
    }
    api<Statement>("/api/v1/my/wallet")
      .then(setSt)
      .catch(() => setSt({ balance: 0, transactions: [] }));
  }, [user, loading, router]);

  if (!st) return <p className="py-10 text-center text-ink-muted">{m.common.loading}</p>;

  return (
    <div className="mx-auto max-w-xl">
      <h1 className="mb-5 flex items-center gap-2 text-xl font-bold">
        <IconWallet className="text-primary" />
        {m.site.wallet.title}
      </h1>

      <div className="mb-3 rounded-card bg-primary p-6 text-center text-white">
        <p className="text-sm opacity-80">{m.site.wallet.balance}</p>
        <p className="mt-1 text-3xl font-bold">
          {fmt.format(st.balance)} <span className="text-base font-normal">{m.common.currency}</span>
        </p>
      </div>
      <p className="mb-6 rounded-control bg-page px-3 py-2 text-xs leading-relaxed text-ink-muted">
        {m.site.wallet.hint}
      </p>

      <h2 className="mb-3 font-bold">{m.site.wallet.statement}</h2>
      {st.transactions.length === 0 ? (
        <p className="py-6 text-center text-sm text-ink-muted">{m.site.wallet.empty}</p>
      ) : (
        <ul className="space-y-2">
          {st.transactions.map((tx) => (
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
