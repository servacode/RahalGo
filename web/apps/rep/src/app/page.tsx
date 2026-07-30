"use client";

/** لوحة المندوب — صفحة واحدة: كوده للدعوة، إحصاءاته، متاجره، كشف عمولاته. */

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  IconStore,
  IconWallet,
  IconOrder,
  IconSuccess,
  IconLogout,
  IconPromos,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isRep } from "@/lib/auth";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");
const KIND_LABELS: Record<string, string> = m.admin.users.txKinds;

interface Me {
  invite_code: string | null;
  full_name: string;
  merchants: number;
  delivered_orders: number;
  total_commissions: number;
  balance: number;
}
interface RepMerchant {
  id: string;
  name: string;
  category_icon: string;
  logo_thumb_url: string | null;
  status: string;
  joined_at: string;
  delivered_orders: number;
}
interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
}

export default function RepPage() {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const [me, setMe] = useState<Me | null>(null);
  const [merchants, setMerchants] = useState<RepMerchant[]>([]);
  const [txs, setTxs] = useState<Tx[]>([]);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    if (loading) return;
    if (!isRep(user)) {
      router.replace("/login");
      return;
    }
    api<Me>("/api/v1/rep/me").then(setMe).catch(() => undefined);
    api<RepMerchant[]>("/api/v1/rep/merchants").then(setMerchants).catch(() => undefined);
    api<{ transactions: Tx[] }>("/api/v1/rep/wallet")
      .then((s) => setTxs(s.transactions))
      .catch(() => undefined);
  }, [user, loading, router]);

  if (loading || !me) {
    return (
      <main className="flex min-h-screen items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  const code = me.invite_code ?? "—";
  const shareText = encodeURIComponent(m.rep.shareText.replace("{code}", code));

  const stats = [
    { label: m.rep.stats.merchants, value: fmt.format(me.merchants), icon: <IconStore /> },
    {
      label: m.rep.stats.delivered,
      value: fmt.format(me.delivered_orders),
      icon: <IconSuccess className="text-success" />,
    },
    {
      label: `${m.rep.stats.commissions} (${m.common.currency})`,
      value: fmt.format(me.total_commissions),
      icon: <IconOrder />,
    },
    {
      label: `${m.rep.stats.balance} (${m.common.currency})`,
      value: fmt.format(me.balance),
      icon: <IconWallet className="text-primary" />,
    },
  ];

  return (
    <div className="min-h-screen">
      <header className="border-b border-line bg-surface">
        <div className="mx-auto flex max-w-3xl items-center gap-3 px-4 py-3">
          <span className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-white">
            ر
          </span>
          <p className="font-bold">{m.rep.loginTitle}</p>
          <span className="text-sm text-ink-muted" dir="ltr">
            {user?.phone}
          </span>
          <button
            onClick={() => {
              logout();
              router.replace("/login");
            }}
            className="ms-auto flex items-center gap-1.5 rounded-control px-2 py-1.5 text-sm text-danger hover:bg-danger/10"
          >
            <IconLogout size={16} />
            {m.auth.logout}
          </button>
        </div>
      </header>

      <main className="mx-auto max-w-3xl space-y-6 p-4">
        {/* الكود — قلب اللوحة */}
        <section className="rounded-card bg-primary p-6 text-center text-white">
          <p className="mb-2 flex items-center justify-center gap-2 text-sm opacity-80">
            <IconPromos size={16} />
            {m.rep.codeTitle}
          </p>
          <p className="font-mono text-4xl font-bold tracking-widest" dir="ltr">
            {code}
          </p>
          <p className="mx-auto mt-3 max-w-md text-xs leading-relaxed opacity-80">
            {m.rep.codeHint}
          </p>
          <div className="mt-4 flex justify-center gap-2">
            <Button
              variant="secondary"
              onClick={() => {
                void navigator.clipboard.writeText(code);
                setCopied(true);
                setTimeout(() => setCopied(false), 1500);
              }}
            >
              {copied ? m.rep.copied : m.rep.copy}
            </Button>
            <a
              href={`https://wa.me/?text=${shareText}`}
              target="_blank"
              rel="noreferrer"
              className="rounded-control bg-white px-4 py-2 text-sm font-medium text-primary-dark"
            >
              {m.rep.share}
            </a>
          </div>
        </section>

        <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
          {stats.map((s) => (
            <div key={s.label} className="rounded-card border border-line bg-surface p-4">
              <div className="mb-1 text-ink-muted">{s.icon}</div>
              <p className="text-xl font-bold">{s.value}</p>
              <p className="text-xs text-ink-muted">{s.label}</p>
            </div>
          ))}
        </div>

        <section>
          <h2 className="mb-3 flex items-center gap-2 font-bold">
            <IconStore size={18} className="text-primary" />
            {m.rep.merchantsTitle}
          </h2>
          {merchants.length === 0 ? (
            <p className="rounded-card border border-line bg-surface p-6 text-center text-sm text-ink-muted">
              {m.rep.merchantsEmpty}
            </p>
          ) : (
            <ul className="space-y-2">
              {merchants.map((mr) => {
                const logo = mediaUrl(mr.logo_thumb_url);
                return (
                  <li
                    key={mr.id}
                    className="flex items-center gap-3 rounded-card border border-line bg-surface p-3"
                  >
                    {logo ? (
                      // eslint-disable-next-line @next/next/no-img-element
                      <img src={logo} alt="" className="h-11 w-11 rounded-control object-cover" />
                    ) : (
                      <span className="flex h-11 w-11 items-center justify-center rounded-control bg-primary-light">
                        {mr.category_icon}
                      </span>
                    )}
                    <div className="min-w-0 flex-1">
                      <p className="truncate font-medium">{mr.name}</p>
                      <p className="text-xs text-ink-muted">
                        {m.rep.joinedAt} <span dir="ltr">{mr.joined_at}</span> —{" "}
                        {m.rep.deliveredCount.replace("{n}", fmt.format(mr.delivered_orders))}
                      </p>
                    </div>
                    <Badge variant={mr.status === "active" ? "success" : "danger"}>
                      {mr.status === "active" ? m.admin.merchants.active : m.admin.merchants.inactive}
                    </Badge>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        <section>
          <h2 className="mb-3 flex items-center gap-2 font-bold">
            <IconWallet size={18} className="text-primary" />
            {m.rep.walletTitle}
          </h2>
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
        </section>
      </main>
    </div>
  );
}
