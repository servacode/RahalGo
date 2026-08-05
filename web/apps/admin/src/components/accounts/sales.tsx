"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  IconCheck,
  IconCopy,
  useLiveRefresh,
  PageHeader,
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
  IconBlock,
  IconUnblock,
  Modal,
  Input,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import WalletModal from "@/components/WalletModal";

const m = getMessages(defaultLocale);

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
        {copied ? <IconCheck size={14} /> : <IconCopy size={14} />}
      </span>
    </button>
  );
}

export default function SalesTable() {
  const { user: me } = useAuth();
  const router = useRouter();
  const canWallet = !!me?.roles.some((r) => r === "admin" || r === "finance");

  const [reps, setReps] = useState<Rep[]>([]);
  const [error, setError] = useState("");
  const [walletFor, setWalletFor] = useState<Rep | null>(null);
  const [statusFor, setStatusFor] = useState<{ rep: Rep; status: string } | null>(null);
  const isAdmin = !!me?.roles.includes("admin");

  async function setStatus(rep: Rep, status: string, reason = "") {
    await api(`/api/v1/admin/users/${rep.id}`, {
      method: "PATCH",
      body: JSON.stringify({ status, status_reason: reason }),
    });
    setStatusFor(null);
    await load();
  }
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

  useLiveRefresh(["lead", "wallet", "account"], load);

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
      cell: (p) => <span className="font-bold">{fmtNum(p.merchants_count)}</span>,
    },
    {
      id: "commissions",
      header: m.admin.sales.totalCommissions,
      icon: <IconWallet />,
      cell: (p) => (
        <span className="font-bold text-success">
          {fmtNum(p.total_commissions)} {m.common.currency}
        </span>
      ),
    },
    {
      id: "balance",
      header: m.admin.sales.balance,
      cell: (p) => `${fmtNum(p.balance)} ${m.common.currency}`,
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
        <PageHeader icon={IconUser} title={m.admin.sales.title} />
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      <DataView
        items={reps}
        getKey={(p) => p.id}
        columns={columns}
        view={view}
        empty={m.admin.sales.empty}
        onRowClick={(p) => router.push(`/dashboard/users/${p.id}`)}
        actions={(p) => (
          <>
            <Button
              variant="secondary"
              onClick={() => setWalletFor(p)}
              className="flex items-center gap-1.5"
            >
              <IconWallet size={15} />
              {m.admin.users.wallet}
            </Button>
            {isAdmin &&
              (p.status === "active" ? (
                <>
                  <Button
                    variant="secondary"
                    onClick={() => setStatusFor({ rep: p, status: "suspended" })}
                    className="flex items-center gap-1.5 !text-warning"
                  >
                    <IconBlock size={15} />
                    {m.admin.users.suspend}
                  </Button>
                  <Button
                    variant="danger"
                    onClick={() => setStatusFor({ rep: p, status: "blocked" })}
                    className="flex items-center gap-1.5"
                  >
                    <IconBlock size={15} />
                    {m.admin.users.block}
                  </Button>
                </>
              ) : (
                <Button
                  variant="secondary"
                  onClick={() => void setStatus(p, "active")}
                  className="flex items-center gap-1.5"
                >
                  <IconUnblock size={15} />
                  {m.admin.users.activate}
                </Button>
              ))}
          </>
        )}
      />

      {walletFor && (
        <WalletModal user={walletFor} onClose={() => setWalletFor(null)} isAdmin={canWallet} />
      )}
      {statusFor && (
        <Modal
          open
          onClose={() => setStatusFor(null)}
          title={m.admin.users.statusReasonTitle.replace(
            "{action}",
            statusFor.status === "suspended" ? m.admin.users.suspend : m.admin.users.block
          )}
        >
          <StatusReasonForm
            status={statusFor.status}
            onSubmit={(reason) => setStatus(statusFor.rep, statusFor.status, reason)}
            onClose={() => setStatusFor(null)}
          />
        </Modal>
      )}
    </div>
  );
}

function StatusReasonForm({
  status,
  onSubmit,
  onClose,
}: {
  status: string;
  onSubmit: (reason: string) => void;
  onClose: () => void;
}) {
  const [reason, setReason] = useState("");
  const label = status === "suspended" ? m.admin.users.suspend : m.admin.users.block;
  return (
    <form
      onSubmit={(e) => {
        e.preventDefault();
        onSubmit(reason);
      }}
      className="space-y-4"
    >
      <Input
        id="rep-status-reason"
        label={m.admin.users.statusReasonLabel}
        required
        autoFocus
        value={reason}
        onChange={(e) => setReason(e.target.value)}
      />
      <div className="flex justify-end gap-2">
        <Button type="button" variant="secondary" onClick={onClose}>
          {m.common.cancel}
        </Button>
        <Button type="submit" variant={status === "blocked" ? "danger" : "primary"}>
          {label}
        </Button>
      </div>
    </form>
  );
}
