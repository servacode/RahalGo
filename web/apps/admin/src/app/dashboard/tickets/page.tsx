"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDate, fmtDateTime } from "@rahalgo/i18n";
import {
  Pagination,
  Alert,
  useLiveRefresh,
  PageHeader,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  FormSection,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  IconSupport,
  IconAdd,
  IconUser,
  IconPhone,
  IconOrder,
  IconWallet,
  IconStatus,
  IconDate,
  IconReply,
  IconCheck,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface Reply {
  id: number;
  author_id: string | null;
  body: string;
  created_at: string;
}

interface Ticket {
  id: string;
  number: number;
  customer_phone: string;
  customer_name: string;
  order_id: string | null;
  order_number: number | null;
  subject: string;
  /** رمزُ سببٍ مصنَّف — فارغٌ في تذكرةٍ فتحها موظّف. */
  reason?: string;
  opened_by_customer?: boolean;
  status: "open" | "in_progress" | "resolved";
  compensation: number;
  resolution: string;
  created_at: string;
  resolved_at: string | null;
  replies?: Reply[];
}

interface TicketPage {
  tickets: Ticket[];
  total: number;
  page: number;
  per_page: number;
}

const STATUS_LABELS: Record<string, string> = m.admin.tickets.status;
const REASONS: Record<string, string> = m.site.complaint.reasons;
const STATUS_VARIANT: Record<string, "warning" | "primary" | "success"> = {
  open: "warning",
  in_progress: "primary",
  resolved: "success",
};

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}
function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

const textareaCls =
  "w-full surface-inset px-3 py-2 text-sm outline-none focus:border-primary";

export default function TicketsPage() {
  const [data, setData] = useState<TicketPage | null>(null);
  const [status, setStatus] = useState("");
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [detailID, setDetailID] = useState<string | null>(null);
  const [view, setView] = useViewMode("tickets");

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({ status, page: String(page), per_page: "12" });
      setData(await api<TicketPage>(`/api/v1/admin/tickets?${params}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [status, page]);

  useEffect(() => {
    load();
  }, [load]);

  useLiveRefresh(["ticket"], load);

  const totalPages = data ? Math.max(1, Math.ceil(data.total / data.per_page)) : 1;

  const columns: DataColumn<Ticket>[] = [
    {
      id: "number",
      header: m.admin.tickets.table.number,
      icon: <IconSupport />,
      primary: true,
      cell: (t) => <span className="font-bold">#{fmtRef(t.number)}</span>,
    },
    {
      id: "customer",
      header: m.admin.tickets.table.customer,
      icon: <IconUser />,
      primary: true,
      cell: (t) => (
        <span>
          {t.customer_name || "—"}{" "}
          <span dir="ltr" className="text-xs text-ink-muted">
            {t.customer_phone}
          </span>
        </span>
      ),
    },
    {
      id: "subject",
      header: m.admin.tickets.table.subject,
      icon: <IconReply />,
      cell: (t) => (
        <span className="flex flex-wrap items-center gap-1.5">
          <span className="line-clamp-1">{t.subject}</span>
          {/* **السببُ المصنَّف بجانب النصّ.**

              الموضوعُ يقول «شكوى على الطلب #١٢» ولا يقول **ما الشكوى** —
              فيُفتح كلُّ سطرٍ ليُعرف. **والرمزُ يُقرأ من الصفّ**: من يبحث عن
              «لم يصلني طلبي» يجدها بالنظر لا بالفتح. */}
          {t.reason && REASONS[t.reason] && (
            <Badge variant="warning">{REASONS[t.reason]}</Badge>
          )}
        </span>
      ),
    },
    {
      id: "order",
      header: m.admin.tickets.table.order,
      icon: <IconOrder />,
      cell: (t) =>
        t.order_number ? (
          <span className="font-medium">#{fmtRef(t.order_number)}</span>
        ) : (
          <span className="text-ink-muted">—</span>
        ),
    },
    {
      id: "compensation",
      header: m.admin.tickets.table.compensation,
      icon: <IconWallet />,
      cell: (t) =>
        t.compensation > 0 ? (
          <Badge variant="primary">
            {fmtNum(t.compensation)} {m.common.currency}
          </Badge>
        ) : (
          <span className="text-ink-muted">{m.admin.tickets.noCompensation}</span>
        ),
    },
    {
      id: "created",
      header: m.admin.tickets.table.created,
      icon: <IconDate />,
      cell: (t) => fmtDate(t.created_at),
    },
    {
      id: "status",
      header: m.admin.ordersPage.statusCol,
      icon: <IconStatus />,
      cell: (t) => <Badge variant={STATUS_VARIANT[t.status]}>{STATUS_LABELS[t.status]}</Badge>,
    },
  ];

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconSupport} title={m.admin.tickets.title} />
        <div className="flex items-center gap-2">
          <ViewToggle
            view={view}
            onChange={setView}
            tableLabel={m.common.viewTable}
            cardsLabel={m.common.viewCards}
          />
          <Button onClick={() => setCreating(true)} className="flex items-center gap-1.5">
            <IconAdd size={16} />
            {m.admin.tickets.newTicket}
          </Button>
        </div>
      </div>

      <div className="mb-4 w-52">
        <Select
          id="status-filter"
          value={status}
          onChange={(e) => {
            setStatus(e.target.value);
            setPage(1);
          }}
        >
          <option value="">{m.admin.tickets.allStatuses}</option>
          {Object.entries(STATUS_LABELS).map(([k, v]) => (
            <option key={k} value={k}>
              {v}
            </option>
          ))}
        </Select>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      <DataView
        items={data?.tickets ?? []}
        getKey={(t) => t.id}
        columns={columns}
        view={view}
        empty={m.admin.tickets.empty}
        actions={(t) => (
          <Button
            variant="secondary"
            onClick={() => setDetailID(t.id)}
            className="flex items-center gap-1.5"
          >
            <IconReply size={15} />
            {m.admin.ordersPage.details}
          </Button>
        )}
      />

      {data && (
        <div className="mt-4 flex flex-wrap items-center justify-between gap-2 text-sm text-ink-muted">
          <span>{m.admin.users.totalCount.replace("{count}", fmtNum(data.total))}</span>
          {/* **والترقيمُ من المكوّن المشترك** — وأرقامُه كانت لاتينيّةً في
              واجهةٍ عربية لأنّها لم تمرّ بـ`fmtNum`. */}
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}

      {creating && (
        <CreateTicketModal
          onClose={() => setCreating(false)}
          onCreated={(id) => {
            setCreating(false);
            setDetailID(id);
            load();
          }}
        />
      )}
      {detailID && (
        <TicketDetailModal ticketID={detailID} onClose={() => setDetailID(null)} onChanged={load} />
      )}
    </div>
  );
}

// ---------- فتح تذكرة ----------

function CreateTicketModal({
  onClose,
  onCreated,
}: {
  onClose: () => void;
  onCreated: (id: string) => void;
}) {
  const [phone, setPhone] = useState("");
  const [orderNumber, setOrderNumber] = useState("");
  const [subject, setSubject] = useState("");
  const [body, setBody] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const t = await api<Ticket>("/api/v1/admin/tickets", {
        method: "POST",
        body: JSON.stringify({
          customer_phone: phone,
          order_number: orderNumber ? Number(orderNumber) : null,
          subject,
          body,
        }),
      });
      onCreated(t.id);
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={m.admin.tickets.form.title}>
      <form onSubmit={submit} className="space-y-4">
        <Input
          id="t-phone"
          label={m.admin.tickets.form.customerPhone}
          icon={<IconPhone />}
          dir="ltr"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          required
        />
        <Input
          id="t-order"
          label={m.admin.tickets.form.orderNumber}
          icon={<IconOrder />}
          dir="ltr"
          inputMode="numeric"
          value={orderNumber}
          onChange={(e) => setOrderNumber(e.target.value.replace(/\D/g, ""))}
        />
        <Input
          id="t-subject"
          label={m.admin.tickets.form.subject}
          icon={<IconReply />}
          value={subject}
          onChange={(e) => setSubject(e.target.value)}
          required
        />
        <div>
          <label htmlFor="t-body" className="mb-1.5 block text-sm font-medium">
            {m.admin.tickets.form.body}
          </label>
          <textarea
            id="t-body"
            value={body}
            onChange={(e) => setBody(e.target.value)}
            rows={4}
            className={textareaCls}
          />
        </div>
        {error && (
          <Alert>{error}</Alert>
        )}
        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {m.admin.tickets.form.create}
          </Button>
        </div>
      </form>
    </Modal>
  );
}

// ---------- تفاصيل التذكرة ----------

function TicketDetailModal({
  ticketID,
  onClose,
  onChanged,
}: {
  ticketID: string;
  onClose: () => void;
  onChanged: () => void;
}) {
  const { user: me } = useAuth();
  const router = useRouter();
  const canResolve = !!me?.roles.some((r) => r === "admin" || r === "finance");

  const [ticket, setTicket] = useState<Ticket | null>(null);
  const [reply, setReply] = useState("");
  const [resolution, setResolution] = useState("");
  const [compensation, setCompensation] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setTicket(await api<Ticket>(`/api/v1/admin/tickets/${ticketID}`));
    } catch (err) {
      setError(errText(err));
    }
  }, [ticketID]);

  useEffect(() => {
    load();
  }, [load]);

  useLiveRefresh(["ticket"], load);

  async function sendReply(e: React.FormEvent) {
    e.preventDefault();
    if (!reply.trim()) return;
    setBusy(true);
    setError("");
    try {
      setTicket(
        await api<Ticket>(`/api/v1/admin/tickets/${ticketID}/replies`, {
          method: "POST",
          body: JSON.stringify({ body: reply }),
        })
      );
      setReply("");
      onChanged();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  async function resolve(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      setTicket(
        await api<Ticket>(`/api/v1/admin/tickets/${ticketID}/resolve`, {
          method: "POST",
          body: JSON.stringify({
            resolution,
            compensation: compensation ? Number(compensation) : 0,
          }),
        })
      );
      onChanged();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  if (!ticket) {
    return (
      <Modal open onClose={onClose} title={m.admin.tickets.detail}>
        <p className="py-8 text-center text-ink-muted">{error || m.common.loading}</p>
      </Modal>
    );
  }

  return (
    <Modal
      open
      onClose={onClose}
      size="lg"
      title={`${m.admin.tickets.detail} #${fmtRef(ticket.number)}`}
    >
      <div className="space-y-5">
        <FormSection title={m.admin.tickets.table.customer} icon={<IconUser />}>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-1 text-sm">
            <span className="font-medium">{ticket.customer_name || "—"}</span>
            <span dir="ltr" className="text-ink-muted">
              {ticket.customer_phone}
            </span>
            <Badge variant={STATUS_VARIANT[ticket.status]}>{STATUS_LABELS[ticket.status]}</Badge>
            {ticket.order_number && (
              <button
                type="button"
                onClick={() => router.push(`/dashboard/orders?q=${ticket.order_number}`)}
                className="font-medium text-primary hover:underline"
              >
                {m.admin.tickets.viewOrder} #{fmtRef(ticket.order_number)}
              </button>
            )}
          </div>
          <p className="mt-2 font-medium">{ticket.subject}</p>
        </FormSection>

        <FormSection title={m.admin.tickets.thread} icon={<IconReply />}>
          {ticket.replies && ticket.replies.length > 0 ? (
            <ol className="space-y-2">
              {ticket.replies.map((r) => (
                <li key={r.id} className="rounded-control bg-field px-3 py-2 text-sm">
                  <p className="whitespace-pre-wrap">{r.body}</p>
                  <p className="mt-1 text-xs text-ink-muted">
                    {fmtDateTime(r.created_at)}
                  </p>
                </li>
              ))}
            </ol>
          ) : (
            <p className="text-sm text-ink-muted">{m.admin.tickets.noReplies}</p>
          )}

          {ticket.status !== "resolved" && (
            <form onSubmit={sendReply} className="mt-3 flex items-start gap-2">
              <textarea
                value={reply}
                onChange={(e) => setReply(e.target.value)}
                rows={2}
                placeholder={m.admin.tickets.replyPlaceholder}
                className={textareaCls}
              />
              <Button type="submit" disabled={busy || !reply.trim()}>
                {m.admin.tickets.sendReply}
              </Button>
            </form>
          )}
        </FormSection>

        {ticket.status === "resolved" ? (
          <FormSection title={m.admin.tickets.resolveTitle} icon={<IconCheck />}>
            <div className="space-y-2 text-sm">
              {ticket.resolution && <p className="whitespace-pre-wrap">{ticket.resolution}</p>}
              <div className="flex flex-wrap items-center gap-x-5 gap-y-1">
                <span>
                  {m.admin.tickets.table.compensation}:{" "}
                  {ticket.compensation > 0 ? (
                    <Badge variant="primary">
                      {fmtNum(ticket.compensation)} {m.common.currency}
                    </Badge>
                  ) : (
                    <span className="text-ink-muted">{m.admin.tickets.noCompensation}</span>
                  )}
                </span>
                {ticket.resolved_at && (
                  <span className="text-xs text-ink-muted">
                    {m.admin.tickets.resolvedAt}{" "}
                    {fmtDateTime(ticket.resolved_at)}
                  </span>
                )}
              </div>
            </div>
          </FormSection>
        ) : (
          canResolve && (
            <FormSection title={m.admin.tickets.resolveTitle} icon={<IconCheck />}>
              <form onSubmit={resolve} className="space-y-3">
                <div>
                  <label htmlFor="t-resolution" className="mb-1.5 block text-sm font-medium">
                    {m.admin.tickets.resolution}
                  </label>
                  <textarea
                    id="t-resolution"
                    value={resolution}
                    onChange={(e) => setResolution(e.target.value)}
                    rows={2}
                    className={textareaCls}
                  />
                </div>
                <Input
                  id="t-compensation"
                  label={m.admin.tickets.compensationAmount}
                  icon={<IconWallet />}
                  dir="ltr"
                  inputMode="numeric"
                  value={compensation}
                  onChange={(e) => setCompensation(e.target.value.replace(/\D/g, ""))}
                />
                <p className="text-xs text-ink-muted">{m.admin.tickets.compensationHint}</p>
                <div className="flex justify-end">
                  <Button type="submit" disabled={busy || !resolution.trim()}>
                    {m.admin.tickets.resolveButton}
                  </Button>
                </div>
              </form>
            </FormSection>
          )
        )}

        {error && (
          <Alert>{error}</Alert>
        )}
      </div>
    </Modal>
  );
}
