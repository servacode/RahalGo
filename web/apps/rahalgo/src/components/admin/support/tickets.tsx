"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  fmtRef,
  fmtDateTime,
  fmtSpan,
  errorText,
} from "@rahalgo/i18n";
import {
  Pagination,
  Alert,
  useLiveRefresh,
  Button,
  Input,
  Select,
  Badge,
  Modal,
  FormSection,
  LoadingState,
  IconSupport,
  IconAdd,
  IconUser,
  IconPhone,
  IconOrder,
  IconWallet,
  IconWarning,
  IconReply,
  IconCheck,
  Money,
  FormActions,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { Tile } from "@/components/admin/DashParts";

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
  /** **سقفُ اقتراح التعويض** — سقفُ موافقة الشكوى نفسُه، وصفرُه «بلا سقف». */
  compensation_cap?: number;
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

  const T = m.admin.tickets;
  const tickets = data?.tickets ?? [];
  const filtered = status !== "";
  const lateN = data?.late ?? 0;
  const lateH = data?.late_hours ?? 2;
  /** **وتغييرُ الفلتر يرجع للصفحة الأولى.** */
  const pickStatus = (s: string) => {
    setStatus(s);
    setPage(1);
  };

  // **تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦)** — رقمان يُضغطان، والشرحُ وزرُّ الشكوى الجديدة
  // والفلترُ في بطاقةٍ واحدة، وكرتٌ لكلّ شكوى بأفعالها. **ولا جدول.**
  return (
    <div>
      <div className="mb-4 grid grid-cols-2 gap-2">
        <Tile
          icon={IconSupport}
          label={T.shownCount}
          value={data ? fmtNum(data.total) : "—"}
          sub={filtered ? undefined : T.allStatuses}
          tone={(data?.total ?? 0) > 0 ? "accent" : "muted"}
          onClick={() => pickStatus("")}
        />
        {/* **عددُ المتأخّرة كلِّها** — يُضغط فيصفّي عليها. */}
        <Tile
          icon={IconWarning}
          label={T.lateTile}
          value={fmtNum(lateN)}
          sub={T.lateTileSub.replace("{h}", fmtNum(lateH))}
          tone={lateN > 0 ? "danger" : "muted"}
          onClick={() => pickStatus("late")}
        />
      </div>

      <section className="surface mb-4 space-y-3 p-4">
        <div className="flex flex-wrap items-center justify-between gap-3">
          <p className="min-w-0 flex-1 text-sm text-ink-muted">{T.intro}</p>
          <Button
            onClick={() => setCreating(true)}
            className="flex w-full items-center justify-center gap-1.5 sm:w-auto"
          >
            <IconAdd size={16} />
            {T.newTicket}
          </Button>
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <Select
            id="status-filter"
            label={T.statusLabel}
            value={status}
            onChange={(e) => pickStatus(e.target.value)}
          >
            <option value="">{T.allStatuses}</option>
            <option value="unresolved">{T.filterUnresolved}</option>
            <option value="late">{T.filterLate}</option>
            {Object.entries(STATUS_LABELS).map(([k, v]) => (
              <option key={k} value={k}>
                {v}
              </option>
            ))}
          </Select>
        </div>
      </section>

      {error && <Alert className="mb-4">{error}</Alert>}

      {data === null && !error ? (
        <LoadingState variant="text" />
      ) : tickets.length === 0 ? (
        !error && (
          <section className="surface flex flex-col items-center gap-3 p-8 text-center">
            <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
              <IconSupport size={28} />
            </span>
            <h2 className="heading-section">{filtered ? T.empty : T.emptyAll}</h2>
            <p className="max-w-md text-sm text-ink-muted">{filtered ? T.emptyFilteredHint : T.emptyHint}</p>
          </section>
        )
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
          {tickets.map((t) => (
            <TicketCard key={t.id} t={t} onOpen={() => setDetailID(t.id)} />
          ))}
        </ul>
      )}

      {/* **ولا سطرَ للإجمالي** (قرارُ المالك ٢٠٢٦-٠٨-١٦) — والترقيمُ لا يظهر لصفحةٍ واحدة. */}
      {data && data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
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

/** لونُ دائرة الكرت بحال الشكوى. */
const STATUS_TINT: Record<string, string> = {
  open: "bg-warning-solid text-on-solid",
  in_progress: "bg-accent text-on-bright",
  awaiting_finance: "bg-warning-solid text-on-solid",
  resolved: "bg-success-solid text-on-solid",
};

/** **عمرُ الشكوى** — «من ٣ يوم» أو «من ٢ س ١٠ د». */
function ageOf(iso: string): string {
  const T = m.admin.tickets;
  const sec = (Date.now() - new Date(iso).getTime()) / 1000;
  const days = Math.floor(sec / 86400);
  if (days >= 1) return T.ageDays.replace("{n}", fmtNum(days));
  if (sec < 60) return T.ageNow;
  return T.ageSpan.replace("{t}", fmtSpan(sec));
}

/**
 * **كرتُ شكوى** (تصميمٌ احترافيّ — المالك ٢٠٢٦-١٠-٢٦) — صاحبُها وزبونُ الطلب، الموضوعُ وسببُه،
 * الحالُ بلونها، الطلبُ والتعويضُ والعمر، والحلُّ إن وُجد، وزرُّ الفتح.
 */
function TicketCard({ t, onOpen }: { t: Ticket; onOpen: () => void }) {
  const T = m.admin.tickets;
  const who = t.complainant_name || t.customer_name || "—";
  const phone = t.complainant_phone || t.customer_phone;
  // **وزبونُ الطلب حين لا يكون هو صاحبَ الشكوى** — بلاغُ السائق أو المتجر يكون عليه.
  const orderCustomer =
    t.customer_name && t.customer_name !== who
      ? [t.customer_name, t.customer_phone].filter(Boolean).join(" · ")
      : "";
  return (
    <li className={`surface flex flex-col gap-3 p-4 ${t.late ? "border-danger" : ""}`}>
      <div className="flex items-start gap-3">
        <span
          className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${STATUS_TINT[t.status] ?? "bg-field text-ink"}`}
        >
          <IconUser size={20} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="flex flex-wrap items-center gap-1.5">
            <span className="truncate font-bold">{who}</span>
            {t.complainant_kind && <Badge variant="neutral">{KIND_LABELS[t.complainant_kind]}</Badge>}
          </span>
          {phone && (
            <span className="block text-2xs text-ink-muted" dir="ltr">
              {phone}
            </span>
          )}
        </span>
        <span className="flex shrink-0 flex-col items-end gap-1">
          <span dir="ltr" className="text-sm font-bold tabular-nums text-primary">
            #{fmtRef(t.number)}
          </span>
          <Badge variant={STATUS_VARIANT[t.status]}>{STATUS_LABELS[t.status]}</Badge>
          {/* **متأخّرةٌ بالأحمر** — مهلةُ الردّ ساعتان (قرارُ المالك ٢٠٢٦-١٠-٠٤). */}
          {t.late && <Badge variant="danger">{T.lateBadge}</Badge>}
        </span>
      </div>

      <div className="space-y-1.5">
        <p className="font-medium leading-snug">{t.subject}</p>
        {/* **بلاغُ السائق يُميَّز عن شكوى الزبون، والسببُ المصنَّف بجانبه** (فحصُ المالك ٢٠٢٦-٠٨-١٦). */}
        {t.reason && (
          <div className="flex flex-wrap items-center gap-1.5">
            {!t.opened_by_customer && <Badge variant="primary">{T.byDriver}</Badge>}
            {REASONS[t.reason] && <Badge variant="warning">{REASONS[t.reason]}</Badge>}
          </div>
        )}
        {orderCustomer && (
          <p className="text-xs text-ink-muted">
            {T.orderCustomer}: <span className="text-ink">{orderCustomer}</span>
          </p>
        )}
      </div>

      <div className="grid grid-cols-2 gap-2">
        <div className="rounded-control bg-field px-3 py-2">
          <p className="flex items-center gap-1 text-2xs text-ink-muted">
            <IconOrder size={12} />
            {T.table.order}
          </p>
          {t.order_number ? (
            <Link
              href={`/dashboard/orders?q=${t.order_number}`}
              dir="ltr"
              className="mt-0.5 block text-sm font-bold tabular-nums text-primary hover:underline"
            >
              #{fmtRef(t.order_number)}
            </Link>
          ) : (
            <p className="mt-0.5 text-sm text-ink-muted">—</p>
          )}
        </div>
        <div className="rounded-control bg-field px-3 py-2">
          <p className="flex items-center gap-1 text-2xs text-ink-muted">
            <IconWallet size={12} />
            {T.table.compensation}
          </p>
          <div className="mt-0.5 text-sm">
            <CompensationCell t={t} />
          </div>
        </div>
      </div>

      <div className={`border-s-2 ps-3 ${t.resolution ? "border-success" : "border-line"}`}>
        <p className="mb-0.5 flex items-center gap-1 text-2xs font-bold text-ink-muted">
          <IconReply size={12} />
          {T.lastReply}
        </p>
        <p className={`line-clamp-3 text-sm ${t.resolution ? "" : "text-ink-muted"}`}>
          {t.resolution || T.noReplyYet}
        </p>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 text-2xs text-ink-muted">
        <span className={t.late ? "font-bold text-danger" : ""}>{ageOf(t.created_at)}</span>
        <span dir="ltr">{fmtDateTime(t.created_at)}</span>
      </div>

      <div className="mt-auto border-t border-line-soft pt-3">
        <Button variant="secondary" onClick={onOpen} className="flex w-full items-center justify-center gap-1.5">
          <IconReply size={15} />
          {T.openTicket}
        </Button>
      </div>
    </li>
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
                {(ticket.compensation_cap ?? 0) > 0 && (
                  <p className="text-xs text-ink-muted">
                    {m.admin.tickets.compCap.replace("{cap}", fmtMoney(ticket.compensation_cap ?? 0))}
                  </p>
                )}
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
