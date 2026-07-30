"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  Badge,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconUser,
  IconPhone,
  IconStore,
  IconWallet,
  IconStatus,
  IconPromos,
  IconView,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/WalletModal";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

interface Rep {
  id: string;
  phone: string;
  full_name: string;
  status: string;
  invite_code: string | null;
  balance: number;
  merchants_count: number;
  total_commissions: number;
}

function errText(err: unknown): string {
  return err instanceof ApiError ? m.errors.internal : m.errors.internal;
}

function CopyCode({ code }: { code: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <button
      onClick={() => {
        void navigator.clipboard.writeText(code);
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      }}
      className="inline-flex items-center gap-1.5 rounded-badge border border-accent bg-accent/10 px-2.5 py-1 font-mono text-sm font-bold text-accent-dark transition-colors hover:bg-accent/20"
      dir="ltr"
      title={m.admin.sales.copyCode}
    >
      {code}
      <span className="text-xs font-normal">
        {copied ? m.admin.sales.copied : `📋`}
      </span>
    </button>
  );
}

export default function SalesPage() {
  const { user: me } = useAuth();
  const router = useRouter();
  const canWallet = !!me?.roles.some((r) => r === "admin" || r === "finance");

  const [reps, setReps] = useState<Rep[]>([]);
  const [error, setError] = useState("");
  const [walletFor, setWalletFor] = useState<Rep | null>(null);
  const [view, setView] = useViewMode("sales", "cards");

  const load = useCallback(async () => {
    try {
      setReps(await api<Rep[]>("/api/v1/admin/salesreps"));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const columns: DataColumn<Rep>[] = [
    {
      id: "name",
      header: m.admin.users.table.name,
      icon: <IconUser />,
      primary: true,
      cell: (p) => (
        <span className="flex w-full items-center justify-between gap-2">
          <span className="truncate">{p.full_name || "—"}</span>
          <button
            type="button"
            title={m.admin.users.viewProfile}
            onClick={(e) => {
              e.stopPropagation();
              router.push(`/dashboard/users/${p.id}`);
            }}
            className="shrink-0 rounded-control p-1 text-ink-muted transition-colors hover:bg-primary-light hover:text-primary"
          >
            <IconView size={17} />
          </button>
        </span>
      ),
    },
    {
      id: "phone",
      header: m.admin.users.table.phone,
      icon: <IconPhone />,
      primary: true,
      cell: (p) => (
        <span dir="ltr" className="font-medium">
          {p.phone}
        </span>
      ),
    },
    {
      id: "code",
      header: m.admin.sales.inviteCode,
      icon: <IconPromos />,
      cell: (p) => (p.invite_code ? <CopyCode code={p.invite_code} /> : "—"),
    },
    {
      id: "merchants",
      header: m.admin.sales.merchantsCount,
      icon: <IconStore />,
      cell: (p) => <span className="font-bold">{fmt.format(p.merchants_count)}</span>,
    },
    {
      id: "commissions",
      header: m.admin.sales.totalCommissions,
      icon: <IconWallet />,
      cell: (p) => (
        <span className="font-bold text-success">
          {fmt.format(p.total_commissions)} {m.common.currency}
        </span>
      ),
    },
    {
      id: "balance",
      header: m.admin.sales.balance,
      cell: (p) => `${fmt.format(p.balance)} ${m.common.currency}`,
    },
    {
      id: "status",
      header: m.admin.users.table.status,
      icon: <IconStatus />,
      cell: (p) => (
        <Badge
          variant={p.status === "active" ? "success" : p.status === "suspended" ? "warning" : "danger"}
        >
          {p.status === "active"
            ? m.admin.users.active
            : p.status === "suspended"
              ? m.admin.users.suspended
              : m.admin.users.blocked}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <IconUser className="text-primary" />
          {m.admin.sales.title}
        </h1>
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <DataView
        items={reps}
        getKey={(p) => p.id}
        columns={columns}
        view={view}
        empty={m.admin.sales.empty}
        onRowClick={(p) => router.push(`/dashboard/users/${p.id}`)}
        actions={(p) => (
          <Button
            variant="secondary"
            onClick={() => setWalletFor(p)}
            className="flex items-center gap-1.5"
          >
            <IconWallet size={15} />
            {m.admin.users.wallet}
          </Button>
        )}
      />

      {walletFor && (
        <WalletModal user={walletFor} onClose={() => setWalletFor(null)} isAdmin={canWallet} />
      )}
    </div>
  );
}
