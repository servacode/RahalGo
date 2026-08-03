"use client";

/** طلبات سحب الرصيد — المالية تصرف أو ترفض، والقيد يُسجَّل في دفتر المحفظة. */

import { useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDateTime } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  useLiveData,
  IconWallet,
  IconUser,
  IconStatus,
  IconDate,
  IconWarning,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth, hasRole } from "@/lib/auth";

const m = getMessages(defaultLocale);
const P = m.shared.payout;

interface Payout {
  id: string;
  user_id: string;
  user_name: string;
  user_phone: string;
  amount: number;
  status: "pending" | "paid" | "rejected";
  note: string;
  decision: string;
  balance: number;
  created_at: string;
}

const VARIANT: Record<Payout["status"], "warning" | "success" | "danger"> = {
  pending: "warning",
  paid: "success",
  rejected: "danger",
};

function errText(err: unknown): string {
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key.split(".").pop() ?? "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export default function PayoutsPage() {
  const { user } = useAuth();
  const canDecide = hasRole(user, "admin", "finance");
  const [status, setStatus] = useState("");
  const [deciding, setDeciding] = useState<{ p: Payout; approve: boolean } | null>(null);
  const [view, setView] = useViewMode("payouts");

  const { data, loading, reload } = useLiveData<Payout[]>(
    () => api(`/api/v1/admin/payouts${status ? `?status=${status}` : ""}`),
    ["wallet"],
    // **والحالةُ تُعيد الجلب** — وكانت تُضيء ولا تُنادي الشبكة، **ولم يشتكِ
    // منه أحدٌ بعد**: من رأى القائمةَ لا تتغيّر ظنّ أن لا طلباتٍ في تلك الحالة.
    [status],
  );

  if (loading) return <LoadingState />;
  const rows = data ?? [];

  const columns: DataColumn<Payout>[] = [
    {
      id: "who",
      header: m.terms.name,
      icon: <IconUser />,
      primary: true,
      cell: (p) => (
        <span>
          {p.user_name}{" "}
          <span dir="ltr" className="text-xs text-ink-muted">
            {p.user_phone}
          </span>
        </span>
      ),
    },
    {
      id: "amount",
      header: P.amount,
      icon: <IconWallet />,
      primary: true,
      cell: (p) => (
        <span className="font-bold text-primary-dark" dir="ltr">
          {fmtNum(p.amount)}
        </span>
      ),
    },
    {
      id: "balance",
      header: m.terms.walletBalance,
      icon: <IconWallet />,
      cell: (p) => <span dir="ltr">{fmtNum(p.balance)}</span>,
    },
    {
      id: "status",
      header: m.terms.status,
      icon: <IconStatus />,
      cell: (p) => <Badge variant={VARIANT[p.status]}>{P.status[p.status]}</Badge>,
    },
    {
      id: "note",
      header: P.note,
      cell: (p) => <span className="text-sm text-ink-muted">{p.decision || p.note || "—"}</span>,
    },
    {
      id: "date",
      header: m.terms.date,
      icon: <IconDate />,
      cell: (p) => <span dir="ltr">{fmtDateTime(p.created_at)}</span>,
    },
  ];

  const filters = [
    { id: "", label: m.terms.allStatuses },
    { id: "pending", label: P.status.pending },
    { id: "paid", label: P.status.paid },
    { id: "rejected", label: P.status.rejected },
  ];

  return (
    <PageContainer width="full">
      <PageHeader
        icon={IconWallet}
        title={P.title}
        actions={
          <div className="flex flex-wrap items-center gap-2">
            <div className="flex rounded-control border border-line bg-page p-1">
              {filters.map((f) => (
                <button
                  key={f.id}
                  type="button"
                  onClick={() => setStatus(f.id)}
                  className={`rounded-[7px] px-3 py-1.5 text-sm transition-colors ${
                    status === f.id
                      ? "bg-surface font-medium text-primary-dark shadow-sm"
                      : "text-ink-muted hover:text-ink"
                  }`}
                >
                  {f.label}
                </button>
              ))}
            </div>
            <ViewToggle
              view={view}
              onChange={setView}
              tableLabel={m.common.viewTable}
              cardsLabel={m.common.viewCards}
            />
          </div>
        }
      />

      {rows.length === 0 ? (
        <EmptyState icon={IconWallet} title={P.empty} />
      ) : (
        <DataView
          items={rows}
          getKey={(p) => p.id}
          columns={columns}
          view={view}
          empty={P.empty}
          actions={
            canDecide
              ? (p) =>
                  p.status === "pending" ? (
                    <>
                      <Button onClick={() => setDeciding({ p, approve: true })}>
                        {P.status.paid}
                      </Button>
                      <Button variant="danger" onClick={() => setDeciding({ p, approve: false })}>
                        {P.status.rejected}
                      </Button>
                    </>
                  ) : null
              : undefined
          }
        />
      )}

      {deciding && (
        <DecideModal
          payout={deciding.p}
          approve={deciding.approve}
          onClose={() => setDeciding(null)}
          onDone={() => {
            setDeciding(null);
            reload();
          }}
        />
      )}
    </PageContainer>
  );
}

function DecideModal({
  payout,
  approve,
  onClose,
  onDone,
}: {
  payout: Payout;
  approve: boolean;
  onClose: () => void;
  onDone: () => void;
}) {
  const [decision, setDecision] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/payouts/${payout.id}/decide`, {
        method: "POST",
        body: JSON.stringify({ status: approve ? "paid" : "rejected", decision }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={approve ? P.status.paid : P.status.rejected}>
      <form onSubmit={submit} className="space-y-4">
        <div className="rounded-control border border-line bg-page px-3 py-2 text-sm">
          <p className="font-bold">{payout.user_name}</p>
          <p className="text-ink-muted" dir="ltr">
            {fmtNum(payout.amount)} {m.common.currency}
          </p>
        </div>
        <Input
          id="decision"
          label={P.note}
          required={!approve}
          value={decision}
          onChange={(e) => setDecision(e.target.value)}
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
          <Button type="submit" variant={approve ? "primary" : "danger"} disabled={busy}>
            {busy ? m.common.loading : m.common.confirm}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
