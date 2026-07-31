"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  useLiveRefresh,
  PageHeader,
  Button,
  Input,
  Badge,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconUser,
  IconPhone,
  IconSearch,
  IconOrder,
  IconWallet,
  IconStatus,
  IconDate,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/WalletModal";

const m = getMessages(defaultLocale);

interface Customer {
  id: string;
  phone: string;
  full_name: string;
  status: string;
  balance: number;
  orders_count: number;
  delivered_count: number;
  total_spent: number;
  last_order_at: string | null;
}

interface CustomerPage {
  customers: Customer[];
  total: number;
  page: number;
  per_page: number;
}

function errText(err: unknown): string {
  return err instanceof ApiError ? m.errors.internal : m.errors.internal;
}

export default function CustomersPage() {
  const { user: me } = useAuth();
  const router = useRouter();
  const canWallet = !!me?.roles.some((r) => r === "admin" || r === "finance");

  const [data, setData] = useState<CustomerPage | null>(null);
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [walletFor, setWalletFor] = useState<Customer | null>(null);
  const [view, setView] = useViewMode("customers");

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({ query, page: String(page), per_page: "12" });
      setData(await api<CustomerPage>(`/api/v1/admin/customers?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [query, page]);

  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  useLiveRefresh(["account", "order"], load);

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  const columns: DataColumn<Customer>[] = [
    {
      id: "name",
      header: m.admin.users.table.name,
      icon: <IconUser />,
      primary: true,
      cell: (c) => c.full_name || "—",
    },
    {
      id: "phone",
      header: m.admin.users.table.phone,
      icon: <IconPhone />,
      primary: true,
      cell: (c) => (
        <span dir="ltr" className="font-medium">
          {c.phone}
        </span>
      ),
    },
    {
      id: "orders",
      header: m.admin.customers.ordersCount,
      icon: <IconOrder />,
      cell: (c) => (
        <span>
          <span className="font-bold">{fmtNum(c.orders_count)}</span>{" "}
          <span className="text-xs text-success">
            ({fmtNum(c.delivered_count)} {m.admin.customers.deliveredCount})
          </span>
        </span>
      ),
    },
    {
      id: "spent",
      header: m.admin.customers.totalSpent,
      icon: <IconWallet />,
      cell: (c) => (
        <span className="font-bold text-primary-dark">
          {fmtNum(c.total_spent)} {m.common.currency}
        </span>
      ),
    },
    {
      id: "balance",
      header: m.admin.customers.balance,
      cell: (c) =>
        c.balance > 0 ? (
          <Badge variant="primary">{fmtNum(c.balance)}</Badge>
        ) : (
          <span className="text-ink-muted">{m.common.zero}</span>
        ),
    },
    {
      id: "last",
      header: m.admin.customers.lastOrder,
      icon: <IconDate />,
      cell: (c) =>
        c.last_order_at ? (
          fmtDate(c.last_order_at)
        ) : (
          <span className="text-ink-muted">{m.admin.customers.never}</span>
        ),
    },
    {
      id: "status",
      header: m.admin.users.table.status,
      icon: <IconStatus />,
      cell: (c) => (
        <Badge variant={c.status === "active" ? "success" : "danger"}>
          {c.status === "active" ? m.admin.users.active : m.admin.users.blocked}
        </Badge>
      ),
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconUser} title={m.admin.customers.title} />
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>

      <div className="mb-4 w-64">
        <Input
          icon={<IconSearch />}
          placeholder={m.admin.customers.searchPlaceholder}
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            setPage(1);
          }}
        />
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <DataView
        items={data?.customers ?? []}
        getKey={(c) => c.id}
        columns={columns}
        view={view}
        empty={m.admin.customers.empty}
        actions={(c) => (
          <>
            <Button
              variant="secondary"
              onClick={() => router.push(`/dashboard/orders?q=${encodeURIComponent(c.phone)}`)}
              className="flex items-center gap-1.5"
            >
              <IconOrder size={15} />
              {m.admin.customers.viewOrders}
            </Button>
            <Button
              variant="ghost"
              onClick={() => setWalletFor(c)}
              className="flex items-center gap-1.5"
            >
              <IconWallet size={15} />
              {m.admin.users.wallet}
            </Button>
          </>
        )}
      />

      {data && (
        <div className="mt-4 flex items-center justify-between text-sm text-ink-muted">
          <span>{m.admin.users.totalCount.replace("{count}", String(data.total))}</span>
          <div className="flex items-center gap-2">
            <Button variant="secondary" disabled={page <= 1} onClick={() => setPage(page - 1)}>
              {m.admin.users.prev}
            </Button>
            <span>
              {page} / {totalPages}
            </span>
            <Button
              variant="secondary"
              disabled={page >= totalPages}
              onClick={() => setPage(page + 1)}
            >
              {m.admin.users.next}
            </Button>
          </div>
        </div>
      )}

      {walletFor && (
        <WalletModal user={walletFor} onClose={() => setWalletFor(null)} isAdmin={canWallet} />
      )}
    </div>
  );
}
