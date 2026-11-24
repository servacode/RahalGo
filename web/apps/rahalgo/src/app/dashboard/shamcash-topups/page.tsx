"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/dashboard/shamcash-topups — طلباتُ شحن المحفظة بشام كاش** (٢٠٢٦-١١-٢٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * الزبونُ يحوّل على حساب شام كاش ويبعث رقمَ العملية (وصورةَ الإيصال اختياراً). **الموظفُ يتأكّد أنّ
 * التحويلَ وصل الحسابَ** ثمّ «تم، اشحن» فتنضاف الحصّةُ لمحفظة الزبون، أو «مرفوض» بسببٍ يراه الزبون.
 * **والقرارُ يُكتب مرّةً** — وإعادتُه تردّ `already` ولا تُكرّر الشحن.
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDateTime, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Chips,
  Confirm,
  DataView,
  FormActions,
  IconCash,
  IconDate,
  IconUser,
  Input,
  LoadingState,
  Modal,
  PageContainer,
  Pagination,
  ReloadState,
  Textarea,
  ViewToggle,
  useLiveData,
  useViewMode,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PageHero } from "@/components/admin/PageHero";
import { Panel } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const T = m.admin.shamcashTopups;

type Status = "pending" | "approved" | "rejected";

interface TopupRow {
  id: string;
  user_id: string;
  user_name: string;
  user_phone: string;
  currency: string;
  amount: number;
  txn_ref: string;
  receipt_media_id: string | null;
  receipt_url: string | null;
  status: Status;
  reject_reason: string;
  verifier: string;
  decided_by_name: string | null;
  decided_at: string | null;
  movement_id: string | null;
  created_at: string;
  rejected_before: boolean;
}

interface TopupList {
  requests: TopupRow[];
  total: number;
  per_page: number;
  pending: number;
}

export default function ShamcashTopupsPage() {
  const { can } = useAuth();
  const canManage = can("finance.manage");
  const [view, setView, { narrow }] = useViewMode("admin-shamcash-topups", "table");
  const [status, setStatus] = useState<Status>("pending");
  const [q, setQ] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [page, setPage] = useState(1);
  const [notice, setNotice] = useState("");
  const [error, setError] = useState("");
  const [approving, setApproving] = useState<TopupRow | null>(null);
  const [rejecting, setRejecting] = useState<TopupRow | null>(null);
  const [busy, setBusy] = useState(false);

  const qs = new URLSearchParams();
  qs.set("status", status);
  if (q.trim()) qs.set("q", q.trim());
  if (from) qs.set("from", from);
  if (to) qs.set("to", to);
  qs.set("page", String(page));
  const list = useLiveData<TopupList>(() => api(`/api/v1/admin/shamcash-topups?${qs.toString()}`), ["wallet"], [qs.toString()]);

  function filter(fn: () => void) {
    fn();
    setPage(1);
  }

  async function approve(row: TopupRow) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      const r = await api<{ already: boolean }>(`/api/v1/admin/shamcash-topups/${row.id}/approve`, { method: "POST", body: "{}" });
      setApproving(null);
      setNotice(r.already ? T.already : T.approved);
      list.reload();
    } catch (e) {
      setApproving(null);
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const pending = list.data?.pending ?? 0;

  return (
    <PageContainer>
      <PageHero icon={IconCash} title={T.title} />
      <p className="mb-3 text-sm text-ink-muted">{T.hint}</p>
      {notice && (
        <Alert tone="success" className="mb-3">
          {notice}
        </Alert>
      )}
      {error && <Alert className="mb-3">{error}</Alert>}

      <Panel title={T.title} className="mb-3">
        <div className="space-y-3">
          <Chips
            value={status}
            onChange={(v) => filter(() => setStatus(v as Status))}
            items={[
              { id: "pending", label: pending > 0 ? `${T.tabPending} (${fmtNum(pending)})` : T.tabPending },
              { id: "approved", label: T.tabApproved },
              { id: "rejected", label: T.tabRejected },
            ]}
          />
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
            <Input id="sct-q" label={T.search} value={q} onChange={(e) => filter(() => setQ(e.target.value))} />
            <Input id="sct-from" type="date" label={T.from} value={from} onChange={(e) => filter(() => setFrom(e.target.value))} />
            <Input id="sct-to" type="date" label={T.to} value={to} onChange={(e) => filter(() => setTo(e.target.value))} />
          </div>
        </div>
      </Panel>

      {!narrow && (
        <div className="mb-2 flex justify-end">
          <ViewToggle view={view} onChange={setView} tableLabel={m.common.viewTable} cardsLabel={m.common.viewCards} />
        </div>
      )}
      {list.error && !list.data && <ReloadState onRetry={list.reload} />}
      {!list.data && !list.error && <LoadingState variant="list" rows={4} />}
      {list.data && (
        <>
          <DataView<TopupRow>
            view={narrow ? "cards" : view}
            items={list.data.requests}
            getKey={(r) => r.id}
            empty={T.empty}
            columns={[
              {
                id: "customer",
                header: T.colCustomer,
                primary: true,
                icon: <IconUser size={14} />,
                cell: (r) => (
                  <span className="flex flex-col">
                    <span className="font-bold">{r.user_name}</span>
                    <span dir="ltr" className="text-xs text-ink-muted">
                      {r.user_phone}
                    </span>
                  </span>
                ),
              },
              {
                id: "amount",
                header: T.colAmount,
                icon: <IconCash size={14} />,
                cell: (r) => (
                  <span dir="ltr" className="font-bold text-accent-text">
                    {fmtMoney(r.amount)}
                  </span>
                ),
              },
              {
                id: "txn",
                header: T.colTxn,
                cell: (r) => (
                  <span className="flex flex-col gap-1">
                    <span dir="ltr" className="font-bold">
                      {r.txn_ref}
                    </span>
                    {r.rejected_before && <Badge variant="danger">{T.rejectedBefore}</Badge>}
                  </span>
                ),
              },
              {
                id: "receipt",
                header: T.colReceipt,
                cell: (r) => {
                  const url = mediaUrl(r.receipt_url);
                  return url ? (
                    <a href={url} target="_blank" rel="noreferrer" className="block h-12 w-12 overflow-hidden rounded-control border border-line" title={T.receiptOpen}>
                      <img src={url} alt={T.receiptOpen} loading="lazy" className="h-full w-full object-cover" />
                    </a>
                  ) : (
                    <span className="text-xs text-ink-muted">{T.noReceipt}</span>
                  );
                },
              },
              {
                id: "date",
                header: T.colDate,
                icon: <IconDate size={14} />,
                cell: (r) => <span dir="ltr">{fmtDateTime(r.created_at)}</span>,
              },
              {
                id: "status",
                header: T.colStatus,
                cell: (r) => (
                  <span className="flex flex-col gap-1">
                    <Badge variant={r.status === "approved" ? "success" : r.status === "rejected" ? "danger" : "warning"}>
                      {r.status === "approved" ? T.statusApproved : r.status === "rejected" ? T.statusRejected : T.statusPending}
                    </Badge>
                    {r.status === "rejected" && r.reject_reason && (
                      <span className="text-xs text-ink-muted">{T.rejectedWhy.replace("{r}", r.reject_reason)}</span>
                    )}
                    {r.status !== "pending" && r.decided_by_name && r.decided_at && (
                      <span className="text-xs text-ink-muted">
                        {T.decidedBy.replace("{name}", r.decided_by_name).replace("{date}", fmtDateTime(r.decided_at))}
                      </span>
                    )}
                  </span>
                ),
              },
            ]}
            actions={
              canManage
                ? (r) =>
                    r.status === "pending" ? (
                      <>
                        <Button className="min-h-11" onClick={() => setApproving(r)}>
                          {T.approve}
                        </Button>
                        <Button variant="danger-outline" className="min-h-11" onClick={() => setRejecting(r)}>
                          {T.reject}
                        </Button>
                      </>
                    ) : null
                : undefined
            }
          />
          <Pagination page={page} total={list.data.total} perPage={list.data.per_page} onChange={setPage} />
        </>
      )}

      <Confirm
        open={approving !== null}
        tone="primary"
        title={T.approveTitle}
        body={approving ? T.approveBody.replace("{amount}", fmtMoney(approving.amount)).replace("{name}", approving.user_name) : undefined}
        confirmLabel={T.approveYes}
        busy={busy}
        onCancel={() => setApproving(null)}
        onConfirm={() => approving && void approve(approving)}
      />
      {rejecting && (
        <RejectModal
          row={rejecting}
          onClose={() => setRejecting(null)}
          onDone={(already) => {
            setRejecting(null);
            setNotice(already ? T.already : T.rejected);
            list.reload();
          }}
        />
      )}
    </PageContainer>
  );
}

function RejectModal({ row, onClose, onDone }: { row: TopupRow; onClose: () => void; onDone: (already: boolean) => void }) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit() {
    if (busy || reason.trim() === "") return;
    setBusy(true);
    setError("");
    try {
      const r = await api<{ already: boolean }>(`/api/v1/admin/shamcash-topups/${row.id}/reject`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim() }),
      });
      onDone(r.already);
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal open onClose={onClose} title={T.rejectTitle}>
      <div className="space-y-3">
        <p className="text-sm">{T.rejectBody}</p>
        <Textarea id="sct-reason" label={T.reasonLabel} required rows={2} value={reason} onChange={(e) => setReason(e.target.value)} />
        {error && <Alert>{error}</Alert>}
        <FormActions
          onSave={() => void submit()}
          onCancel={onClose}
          busy={busy || reason.trim() === ""}
          saveLabel={T.rejectYes}
          cancelLabel={T.back}
        />
      </div>
    </Modal>
  );
}
