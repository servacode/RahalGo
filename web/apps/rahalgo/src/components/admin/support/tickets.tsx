"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDate, fmtDateTime, errorText} from "@rahalgo/i18n";
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
  ComplaintCard,
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
  Money,
  FormActions,
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
  /** صاحبُ الشكوى — من يُعوَّض ومن يُخبَر (قرارُ المالك ٢٠٢٦-١٠-٠٤). */
  complainant_name?: string;
  complainant_phone?: string;
  complainant_kind?: "customer" | "driver" | "merchant";
  status: "open" | "in_progress" | "awaiting_finance" | "resolved";
  /** ما دُفع فعلاً — بعد موافقة الماليّة. */
  compensation: number;
  /** ما اقترحه الدعمُ على الماليّة، وحالُه. */
  compensation_proposed?: number;
  compensation_status?: "" | "pending" | "approved" | "rejected";
  /** تأخّر ردُّ المكتب عن المهلة (`support.late_reply_hours`). */
  late?: boolean;
  resolution: string;
  created_at: string;
  resolved_at: string | null;
  replies?: Reply[];
}

interface TicketPage {
  tickets: Ticket[];
  total: number;
  late?: number;
  late_hours?: number;
  page: number;
  per_page: number;
}

const STATUS_LABELS: Record<string, string> = m.admin.tickets.status;
/**
 * **أسبابُ الشكوى — من المعجمين معاً.**
 *
 * (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)
 *
 * **الزبونُ يشتكي بأسبابه والسائقُ يبلّغ بأسبابه** — **ومعجمٌ واحدٌ يترك
 * نصفَ الصفوف بلا شارة**: يُقرأ «بلاغُ سائق» ولا يُعرف **أعلى المتجر هو
 * أم على الزبون** حتّى يُفتح السطر. **وهي عينُ العلّة التي أُصلحت في
 * ملفّ الحساب ٢٠٢٦-٠٨-١٥** — وبقيت هنا.
 */
const REASONS: Record<string, string> = {
  ...m.site.complaint.reasons,
  ...m.driver.history.reportReasons,
};
const KIND_LABELS: Record<string, string> = m.admin.tickets.complainantKind;

/**
 * **التعويضُ كما هو الآن** — مدفوعٌ · بانتظار الماليّة · رفضته · أو بلا تعويض.
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «الدعم يحوّل التعويض للماليّة».)
 */
function CompensationCell({ t }: { t: Ticket }) {
  if (t.compensation > 0) {
    return (
      <Badge variant="primary">
        <Money value={t.compensation} />
      </Badge>
    );
  }
  if (t.compensation_status === "pending") {
    return (
      <Badge variant="warning">
        {m.admin.tickets.compPending} · <Money value={t.compensation_proposed ?? 0} />
      </Badge>
    );
  }
  if (t.compensation_status === "rejected") {
    return <Badge variant="neutral">{m.admin.tickets.compRejected}</Badge>;
  }
  return <span className="text-ink-muted">{m.admin.tickets.noCompensation}</span>;
}

const STATUS_VARIANT: Record<string, "warning" | "primary" | "success"> = {
  open: "warning",
  in_progress: "primary",
  awaiting_finance: "warning",
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

const textareaCls =
  "w-full surface-inset px-3 py-2 text-sm outline-none focus:border-primary";

export function TicketsView() {
  const [data, setData] = useState<TicketPage | null>(null);
  // **والحالُ من الرابط** — بطاقتا «شكاوى مفتوحة» و«متأخّرة» في رئيسيّة المدير
  // تفتحان هنا على العدد نفسِه (قرارُ المالك ٢٠٢٦-١٠-٠٤). و`late` حالٌ مُركَّبةٌ
  // في القائمة: غيرُ محلولةٍ لم يردّ عليها أحدٌ بعد مهلة الإعدادات.
  const [status, setStatus] = useState(() => {
    if (typeof window === "undefined") return "";
    const q = new URLSearchParams(window.location.search);
    return q.get("late") === "1" ? "late" : (q.get("status") ?? "");
  });
  const [page, setPage] = useState(1);
  const [error, setError] = useState("");
  const [creating, setCreating] = useState(false);
  const [detailID, setDetailID] = useState<string | null>(null);
  const [view, setView] = useViewMode("tickets");

  // ══════════════════════════════════════════════════════════════════
  // **وتُفتح تذكرةٌ بعينها من رابط**
  // ══════════════════════════════════════════════════════════════════
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-١٥.)
  //
  // **وكان ملفُّ الحساب يرمي إلى `/dashboard/tickets` بلا رقم** — فتقع
  // في القائمة كلِّها **وتبحث بيدك عمّا كنتَ تنظر إليه.**
  //
  // **وتُقرأ مرّةً عند الإقلاع لا في كلّ رسم**: من أغلق النافذة لا
  // تُعاد عليه، **وإلّا لَما استطاع إغلاقها.**
  const params = useSearchParams();
  useEffect(() => {
    const t = params.get("t");
    if (t) setDetailID(t);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const load = useCallback(async () => {
    try {
      const params = new URLSearchParams({
        status: status === "late" ? "" : status,
        late: status === "late" ? "1" : "",
        page: String(page),
        per_page: "12",
      });
      setData(await api<TicketPage>(`/api/v1/admin/tickets?${params}`));
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, [status, page]);

  useEffect(() => {
    load();
  }, [load]);

  useLiveRefresh(["ticket"], load);

  const columns: DataColumn<Ticket>[] = [
    {
      id: "number",
      header: m.admin.tickets.table.number,
      icon: <IconSupport />,
      primary: true,
      cell: (t) => <span className="font-bold">#{fmtRef(t.number)}</span>,
    },
    {
      // **صاحبُ الشكوى لا زبونُ الطلب** — بلاغُ السائق صاحبُه السائق.
      id: "customer",
      header: m.admin.tickets.complainant,
      icon: <IconUser />,
      primary: true,
      cell: (t) => (
        <span className="flex flex-wrap items-center gap-1.5">
          {t.complainant_kind && (
            <Badge variant="neutral">{KIND_LABELS[t.complainant_kind]}</Badge>
          )}
          {t.complainant_name || t.customer_name || "—"}{" "}
          <span dir="ltr" className="text-xs text-ink-muted">
            {t.complainant_phone || t.customer_phone}
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
          {/* ══════════════════════════════════════════════════════════
              **وبلاغُ السائق يُميَّز عن شكوى الزبون**
              ══════════════════════════════════════════════════════════

              (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦.)

              **وعمودُ «الزبون» يقول زبونَ الطلب لا كاتبَ الشكوى** —
              فبلاغُ السائق على الزبون **يُقرأ شكوى الزبون نفسِه**،
              والحكمان متناقضان.

              **والمحرّكُ يرسل `opened_by_customer` ولا تقرؤه الشاشة.** */}
          {!t.opened_by_customer && t.reason && (
            <Badge variant="primary">{m.admin.tickets.byDriver}</Badge>
          )}
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
      cell: (t) => <CompensationCell t={t} />,
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
      cell: (t) => (
        <span className="flex flex-wrap items-center gap-1.5">
          <Badge variant={STATUS_VARIANT[t.status]}>{STATUS_LABELS[t.status]}</Badge>
          {/* **متأخّرةٌ بالأحمر** — مهلةُ الردّ ساعتان (قرارُ المالك ٢٠٢٦-١٠-٠٤). */}
          {t.late && <Badge variant="danger">{m.admin.tickets.lateBadge}</Badge>}
        </span>
      ),
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

      {/* **عددُ المتأخّرة كلِّها** — يُنقر فيصفّي عليها. */}
      {data && (data.late ?? 0) > 0 && (
        <Alert className="mb-4">
          <button
            type="button"
            onClick={() => {
              setStatus("late");
              setPage(1);
            }}
            className="font-bold hover:underline"
          >
            {m.admin.tickets.lateCount.replace("{n}", fmtNum(data.late ?? 0))}
          </button>{" "}
          <span className="text-xs">
            {m.admin.tickets.lateRule.replace("{h}", fmtNum(data.late_hours ?? 2))}
          </span>
        </Alert>
      )}

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
          <option value="unresolved">{m.admin.tickets.filterUnresolved}</option>
          <option value="late">{m.admin.tickets.filterLate}</option>
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

      {/* **والكرتُ هو كرتُ البلاغ نفسُه** الذي تعرضه لوحاتُ المتجر والسائق
          والمندوب والزبون — **لا كرتٌ سادسٌ تبنيه الإدارةُ من أعمدتها.**
          (طلبُ المالك ٢٠٢٦-٠٨-٠٨: «كروت وجداول مثل باقي المشروع».)

          **والجدولُ يبقى من الأعمدة**: الإدارةُ تمسح مئةَ بلاغٍ بالعين،
          والكرتُ لمن يقرأ واحداً. */}
      <DataView
        items={data?.tickets ?? []}
        loading={data === null && !error}
        getKey={(t) => t.id}
        columns={columns}
        view={view}
        empty={m.admin.tickets.empty}
        card={(t) => (
          <ComplaintCard
            ticket={t}
            note={[t.customer_name, t.customer_phone].filter(Boolean).join(" · ")}
            action={
              <Button
                variant="secondary"
                onClick={() => setDetailID(t.id)}
                className="flex w-full items-center justify-center gap-1.5"
              >
                <IconReply size={15} />
                {m.admin.ordersPage.details}
              </Button>
            }
          />
        )}
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

      {/* **ولا سطرَ للإجمالي** — (قرارُ المالك ٢٠٢٦-٠٨-١٦).

          **والترقيمُ يقوله بموضعه**: «٢ / ٧» تقول العددَ والموضعَ معاً،
          **وسطرٌ ثانٍ يقول نصفَ ما تقوله** — و«الإجمالي: ٠» تحت قائمةٍ
          فارغةٍ تقول ما تقوله القائمةُ نفسُها. */}
      {data && (
        <div className="mt-4">
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
      setError(errorText(err));
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
        <FormActions submit onCancel={onClose} busy={busy} saveLabel={m.admin.tickets.form.create} />
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
  const { user: me, can } = useAuth();
  const router = useRouter();
  // ════════════════════════════════════════════════════════════════
  // **وإغلاقُ التذكرة قدرةُ الدعم لا قدرةُ المال** (٢٠٢٦-٠٩-١٣)
  // ════════════════════════════════════════════════════════════════
  //
  // **ودورةُ ٢٥ نقلت `POST /tickets/{id}/resolve` إلى
  // `support.manage`** — **ولم يُصحَّح الزرّ.** **فصار يُعرَض
  // للماليّة وهي لا تملكها فيُردّ ٤٠٣**، **ويُخفى عن موظّف الدعم
  // وهو صاحبُه.** (قِيس ٢٠٢٦-٠٩-١٣.)
  const canResolve = can("support.manage");

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
      setError(errorText(err));
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
      setError(errorText(err));
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
      setError(errorText(err));
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
        <FormSection title={m.admin.tickets.complainant} icon={<IconUser />}>
          <div className="flex flex-wrap items-center gap-x-5 gap-y-1 text-sm">
            {ticket.complainant_kind && (
              <Badge variant="neutral">{KIND_LABELS[ticket.complainant_kind]}</Badge>
            )}
            <span className="font-medium">
              {ticket.complainant_name || ticket.customer_name || "—"}
            </span>
            <span dir="ltr" className="text-ink-muted">
              {ticket.complainant_phone || ticket.customer_phone}
            </span>
            <Badge variant={STATUS_VARIANT[ticket.status]}>{STATUS_LABELS[ticket.status]}</Badge>
            {ticket.late && <Badge variant="danger">{m.admin.tickets.lateBadge}</Badge>}
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

        {ticket.status === "resolved" || ticket.status === "awaiting_finance" ? (
          <FormSection title={m.admin.tickets.resolveTitle} icon={<IconCheck />}>
            <div className="space-y-2 text-sm">
              {ticket.resolution && <p className="whitespace-pre-wrap">{ticket.resolution}</p>}
              <div className="flex flex-wrap items-center gap-x-5 gap-y-1">
                <span>
                  {m.admin.tickets.table.compensation}: <CompensationCell t={ticket} />
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
                  label={m.admin.tickets.compProposedAmount}
                  icon={<IconWallet />}
                  dir="ltr"
                  inputMode="numeric"
                  value={compensation}
                  onChange={(e) => setCompensation(e.target.value.replace(/\D/g, ""))}
                />
                <p className="text-xs text-ink-muted">{m.admin.tickets.compProposedHint}</p>
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
