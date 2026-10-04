"use client";

/**
 * **مصروفاتُ التشغيل — ما ينفقه المكتبُ لا ما يخسره العمل.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦: «يوجد مكتبٌ للشركة وموظّفون وعمّال وما إلى ذلك من
 *  المصاريف — يجب أن تُوثَّق بشكلٍ صحيح».)
 *
 * **والإيجارُ ليس خسارة** — كلفةُ تشغيلٍ مخطَّطة، وتخرج من خزينة المنصّة.
 *
 * # قراراتُ المالك ٢٠٢٦-١٠-٠٤
 *
 * ١ · فوق سقف الموافقة (من الإعدادات) يصير المصروفُ اقتراحاً يوافق عليه موظّفٌ آخر.
 * ٢ · الشهرُ بتاريخ الصرف — هنا وفي صفحة الأرباح.
 * ٣ · الصفحةُ بمسارها، ويربطها قسمُ الخزينة تبويباً.
 * ٤ · صورةُ الإيصال اختياريّة، وإلزاميّةٌ فوق السقف.
 * ٥ · الإلغاءُ بسببٍ ومن موظّفٍ غيرِ من سجّل، والملغى يبقى ظاهراً مشطوباً.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtDate,
  errorText,
  westernDigits,
  damascusDay,
} from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Input,
  Select,
  Modal,
  Textarea,
  PageContainer,
  PageHeader,
  LoadingState,
  ReloadState,
  Pagination,
  StatGrid,
  StatCard,
  Checkbox,
  DataView,
  ViewToggle,
  useViewMode,
  type DataColumn,
  ImageUpload,
  IconWallet,
  IconAdd,
  IconDate,
  IconUser,
  IconNote,
  IconStatus,
  IconReceipt,
  IconSearch,
  IconTrendUp,
  IconTrendDown,
  IconTrendFlat,
  IconEdit,
  FormActions,
} from "@rahalgo/ui";
import { api, apiFile, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const X = m.admin.expenses;
const CUR = m.common.currency;

type Status = "posted" | "voided" | "pending" | "rejected";

interface Row {
  id: string;
  source: "expense" | "request";
  status: Status;
  category: string;
  amount: number;
  note: string;
  spent_at: string;
  by: string;
  by_id: string;
  decider: string;
  decided_at: string | null;
  reason: string;
  self_approved: boolean;
  receipt_url: string | null;
}
interface Cat {
  id: string;
  name: string;
  active: boolean;
  sort_order: number;
  used: number;
  spent: number;
}
interface Data {
  expenses: Row[];
  total: number;
  total_amount: number;
  pending_amount: number;
  pending_count: number;
  page: number;
  per_page: number;
  by_category: { id: string; name: string; count: number; total: number }[];
  summary: {
    month_total: number;
    prev_month_total: number;
    top_category: string;
    top_category_total: number;
  };
  threshold: number;
  treasury_balance: number;
  today: string;
}

/** **أوّلُ الشهر الجاري ويومُ اليوم بيوم دمشق** — لا بيوم المتصفّح ولا بغرينتش. */
function monthRange(): { from: string; to: string } {
  const today = damascusDay();
  const [y, mo] = today.split("-").map(Number);
  const last = new Date(Date.UTC(y!, mo!, 0)).getUTCDate();
  return { from: `${today.slice(0, 8)}01`, to: `${today.slice(0, 8)}${String(last).padStart(2, "0")}` };
}

/**
 * **قراءةُ المبلغ بالأرقام العربيّة والفواصل** — «١٥٠٬٠٠٠» و«150,000» و«150 000» سواء.
 * ويردّ `null` مع سببٍ يُقال — وكان الحفظُ يسكت فلا يحدث شيء.
 */
function parseAmount(raw: string): { value: number | null; error: string } {
  const s = westernDigits(raw).replace(/[\s,\u060C_]/g, "");
  if (s === "") return { value: null, error: X.amountZero };
  if (!/^\d+$/.test(s)) return { value: null, error: X.amountInvalid };
  const n = Number(s);
  if (!Number.isSafeInteger(n)) return { value: null, error: X.amountInvalid };
  if (n <= 0) return { value: null, error: X.amountZero };
  return { value: n, error: "" };
}

const statusVariant: Record<Status, "success" | "neutral" | "warning" | "danger"> = {
  posted: "success",
  voided: "neutral",
  pending: "warning",
  rejected: "danger",
};

export default function ExpensesPage() {
  const [range, setRange] = useState(monthRange());
  const [category, setCategory] = useState("");
  const [status, setStatus] = useState<"" | Status>("");
  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<Data | null | "failed">(null);
  const [cats, setCats] = useState<Cat[]>([]);
  const [adding, setAdding] = useState(false);
  const [catsOpen, setCatsOpen] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const { can } = useAuth();
  const canManage = can("finance.manage");
  const canExport = can("finance.export");
  const [voiding, setVoiding] = useState<Row | null>(null);
  const [why, setWhy] = useState("");
  const [view, setView] = useViewMode("expenses");

  // **ومدًى مقلوبٌ يُقال** — وكان يعطي قائمةً فارغةً بلا تنبيه.
  const inverted = range.from !== "" && range.to !== "" && range.from > range.to;

  const qs = useMemo(() => {
    const p = new URLSearchParams({ from: range.from, to: range.to, page: String(page) });
    if (category) p.set("category_id", category);
    if (status) p.set("status", status);
    if (q.trim()) p.set("q", q.trim());
    return p.toString();
  }, [range, page, category, status, q]);

  const load = useCallback(() => {
    if (inverted) return;
    api<Data>(`/api/v1/admin/expenses?${qs}`)
      .then((d) => {
        setWhy("");
        setData(d);
      })
      .catch((e) => {
        setWhy(errorText(e));
        setData("failed");
      });
  }, [qs, inverted]);

  const loadCats = useCallback(() => {
    api<{ categories: Cat[] }>("/api/v1/admin/expenses/categories")
      .then((r) => setCats(r.categories ?? []))
      .catch((err) => setError(errorText(err)));
  }, []);

  useEffect(load, [load]);
  useEffect(loadCats, [loadCats]);

  function exportCsv() {
    void (async () => {
      try {
        const p = new URLSearchParams(qs);
        p.delete("page");
        const res = await apiFile(`/api/v1/admin/expenses/export?${p.toString()}`);
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `expenses-${range.from}_${range.to}.csv`;
        a.click();
        URL.revokeObjectURL(url);
      } catch (err) {
        setError(errorText(err));
      }
    })();
  }

  async function decide(r: Row, approve: boolean) {
    try {
      await api(`/api/v1/admin/expense-requests/${r.id}/${approve ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
    load();
  }

  if (data === "failed") return <ReloadState onRetry={load} label={why || undefined} />;
  if (!data) return <LoadingState />;

  const s = data.summary;
  const diff = s.month_total - s.prev_month_total;

  const columns: DataColumn<Row>[] = [
    {
      id: "date",
      header: X.colDate,
      icon: <IconDate />,
      cell: (e) => (
        <span dir="ltr" className="whitespace-nowrap text-sm">
          {fmtDate(e.spent_at)}
        </span>
      ),
    },
    {
      id: "category",
      header: X.colCategory,
      primary: true,
      cell: (e) => <span className="font-medium">{e.category}</span>,
    },
    {
      id: "amount",
      header: `${X.colAmount} (${CUR})`,
      icon: <IconWallet />,
      cell: (e) => (
        <span
          dir="ltr"
          className={`whitespace-nowrap font-bold tabular-nums ${
            e.status === "voided" || e.status === "rejected" ? "text-ink-muted line-through" : "text-danger"
          }`}
        >
          {fmtNum(e.amount)} {CUR}
        </span>
      ),
    },
    {
      id: "note",
      header: X.colNote,
      icon: <IconNote />,
      cell: (e) => (
        <span className="flex flex-col text-sm">
          <span className={e.status === "voided" ? "line-through text-ink-muted" : ""}>{e.note}</span>
          {e.status === "voided" && e.reason && (
            <span className="text-xs text-ink-muted">
              {X.voidedBy.replace("{name}", e.decider).replace("{reason}", e.reason)}
            </span>
          )}
        </span>
      ),
    },
    {
      id: "by",
      header: X.colBy,
      icon: <IconUser />,
      cell: (e) => <span className="text-sm text-ink-muted">{e.by}</span>,
    },
    {
      id: "status",
      header: X.colStatus,
      icon: <IconStatus />,
      cell: (e) => <Badge variant={statusVariant[e.status]}>{X.status[e.status]}</Badge>,
    },
    {
      id: "receipt",
      header: X.colReceipt,
      icon: <IconReceipt />,
      hide: (e) => !e.receipt_url,
      cell: (e) =>
        e.receipt_url ? (
          <a
            href={mediaUrl(e.receipt_url) ?? "#"}
            target="_blank"
            rel="noreferrer"
            className="text-sm text-primary underline"
          >
            {X.viewReceipt}
          </a>
        ) : null,
    },
  ];

  return (
    <PageContainer>
      <PageHeader
        icon={IconWallet}
        title={X.title}
        subtitle={X.hint}
        actions={
          <div className="flex flex-wrap gap-2">
            {canExport && (
              <Button variant="secondary" onClick={exportCsv} disabled={inverted}>
                {X.export}
              </Button>
            )}
            {canManage && (
              <>
                <Button variant="secondary" onClick={() => setCatsOpen(true)}>
                  {X.manageCats}
                </Button>
                <Button onClick={() => setAdding(true)} className="flex items-center gap-1.5">
                  <IconAdd size={16} />
                  {X.add}
                </Button>
              </>
            )}
          </div>
        }
      />

      {error && <Alert className="mb-3">{error}</Alert>}
      {notice && (
        <Alert tone="success" className="mb-3">
          {notice}
        </Alert>
      )}

      <StatGrid>
        <StatCard
          label={`${X.monthTotal} (${CUR})`}
          value={fmtNum(s.month_total)}
          icon={IconWallet}
          tone={s.month_total > 0 ? "danger" : "muted"}
        />
        <StatCard
          label={`${X.vsLastMonth} (${CUR})`}
          value={`${diff > 0 ? "+" : ""}${fmtNum(diff)}`}
          sub={X.lastMonthWas.replace("{n}", fmtNum(s.prev_month_total))}
          icon={diff > 0 ? IconTrendUp : diff < 0 ? IconTrendDown : IconTrendFlat}
          tone={diff > 0 ? "danger" : diff < 0 ? "success" : "muted"}
        />
        <StatCard
          label={X.topCategory}
          value={s.top_category || X.none}
          sub={s.top_category ? `${fmtNum(s.top_category_total)} ${CUR}` : undefined}
          icon={IconStatus}
        />
      </StatGrid>

      {data.pending_count > 0 && (
        <Alert tone="warning" className="my-3">
          {X.pendingNote
            .replace("{n}", fmtNum(data.pending_count))
            .replace("{amount}", `${fmtNum(data.pending_amount)} ${CUR}`)}
        </Alert>
      )}

      {/* **شريطُ المدّة والمرشّحات** — يلتفّ على الجوّال. */}
      <div className="my-4 flex flex-wrap items-end gap-3">
        <div className="w-40">
          <Input
            id="exp-from"
            type="date"
            label={m.shared.statement.from}
            value={range.from}
            onChange={(e) => {
              setRange({ ...range, from: e.target.value });
              setPage(1);
            }}
          />
        </div>
        <div className="w-40">
          <Input
            id="exp-to"
            type="date"
            label={m.shared.statement.to}
            value={range.to}
            onChange={(e) => {
              setRange({ ...range, to: e.target.value });
              setPage(1);
            }}
          />
        </div>
        <div className="w-44">
          <Select
            id="exp-filter-cat"
            label={X.colCategory}
            value={category}
            onChange={(e) => {
              setCategory(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{X.allCategories}</option>
            {cats.map((c) => (
              <option key={c.id} value={c.id}>
                {c.name}
              </option>
            ))}
          </Select>
        </div>
        <div className="w-44">
          <Select
            id="exp-filter-status"
            label={X.colStatus}
            value={status}
            onChange={(e) => {
              setStatus(e.target.value as "" | Status);
              setPage(1);
            }}
          >
            <option value="">{X.allStatuses}</option>
            {(["posted", "pending", "voided", "rejected"] as Status[]).map((k) => (
              <option key={k} value={k}>
                {X.status[k]}
              </option>
            ))}
          </Select>
        </div>
        <div className="min-w-0 flex-1 basis-48">
          <Input
            id="exp-search"
            label={X.search}
            value={q}
            onChange={(e) => {
              setQ(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>

      {inverted && <Alert className="mb-3">{X.invertedRange}</Alert>}

      {/* **والتوزيعُ على الأبواب بالمبلغ** — شريطٌ بالنسبة، ومجموعٌ واحدٌ لا يقول أين ذهب المال. */}
      {data.by_category.length > 0 && data.total_amount > 0 && (
        <div className="mb-4 space-y-1.5">
          <div className="flex h-3 w-full overflow-hidden rounded-full bg-line">
            {data.by_category.map((c, i) => (
              <span
                key={c.id}
                title={`${c.name} · ${fmtNum(c.total)} ${CUR}`}
                className={i % 2 === 0 ? "bg-danger" : "bg-warning"}
                style={{ width: `${(c.total / data.total_amount) * 100}%` }}
              />
            ))}
          </div>
          <ul className="flex flex-wrap gap-2 text-sm">
            {data.by_category.map((c) => (
              <li key={c.id} className="rounded-control border border-line px-3 py-1">
                <span className="font-medium">{c.name}</span>{" "}
                <span dir="ltr" className="font-bold tabular-nums text-danger">
                  {fmtNum(c.total)} {CUR}
                </span>{" "}
                <span className="text-xs text-ink-muted">
                  ({X.usedCount.replace("{n}", fmtNum(c.count))} ·{" "}
                  {fmtNum(Math.round((c.total / data.total_amount) * 100))}%)
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}

      <DataView
        items={inverted ? [] : data.expenses}
        getKey={(e) => e.id}
        columns={columns}
        view={view}
        empty={X.empty}
        actions={
          canManage
            ? (e) =>
                e.status === "posted" ? (
                  <Button variant="ghost" className="!px-2 text-xs" onClick={() => setVoiding(e)}>
                    {X.void}
                  </Button>
                ) : e.status === "pending" ? (
                  <div className="flex gap-1">
                    <Button className="!px-2 text-xs" onClick={() => void decide(e, true)}>
                      {X.approve}
                    </Button>
                    <Button variant="ghost" className="!px-2 text-xs" onClick={() => void decide(e, false)}>
                      {X.reject}
                    </Button>
                  </div>
                ) : null
            : undefined
        }
      />

      <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />

      {adding && (
        <AddModal
          cats={cats.filter((c) => c.active)}
          threshold={data.threshold}
          balance={data.treasury_balance}
          today={data.today}
          onClose={() => setAdding(false)}
          onDone={(pending) => {
            setAdding(false);
            setNotice(pending ? X.sentForApproval : "");
            load();
            loadCats();
          }}
        />
      )}
      {catsOpen && (
        <CatsModal
          cats={cats}
          onClose={() => setCatsOpen(false)}
          onChanged={() => {
            loadCats();
            load();
          }}
        />
      )}
      {voiding && (
        <VoidModal
          row={voiding}
          onClose={() => setVoiding(null)}
          onDone={() => {
            setVoiding(null);
            load();
            loadCats();
          }}
        />
      )}
    </PageContainer>
  );
}

/** **الإلغاءُ بسببٍ إلزاميّ** — ومن موظّفٍ غيرِ من سجّل (يحكم الخادم ويقول السبب). */
function VoidModal({ row, onClose, onDone }: { row: Row; onClose: () => void; onDone: () => void }) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit() {
    if (!reason.trim() || busy) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/expenses/${row.id}/void`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim() }),
      });
      onDone();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal open onClose={onClose} title={X.voidConfirmTitle}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{X.voidConfirmBody}</p>
        <p className="text-sm">
          {row.category} ·{" "}
          <span dir="ltr" className="font-bold">
            {fmtNum(row.amount)} {CUR}
          </span>
        </p>
        <Textarea
          id="exp-void-reason"
          label={X.voidReason}
          required
          rows={2}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions
          onSave={() => void submit()}
          onCancel={onClose}
          busy={busy}
          saveLabel={X.void}
        />
      </div>
    </Modal>
  );
}

/** **تسجيلٌ يدويٌّ على خطوتين** — النموذجُ ثمّ مراجعةُ المبلغ ورصيدِ الخزينة بعده. */
function AddModal({
  cats,
  threshold,
  balance,
  today,
  onClose,
  onDone,
}: {
  cats: Cat[];
  threshold: number;
  balance: number;
  today: string;
  onClose: () => void;
  onDone: (pending: boolean) => void;
}) {
  const [categoryID, setCategoryID] = useState(cats[0]?.id ?? "");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  // **ويومُ الصرف الافتراضيُّ يومُ دمشق** — كان بغرينتش فيظهر أمسِ بعد منتصف الليل.
  const [spentAt, setSpentAt] = useState(today || damascusDay());
  const [receipt, setReceipt] = useState("");
  const [review, setReview] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const parsed = parseAmount(amount);
  const over = parsed.value !== null && parsed.value > threshold;

  function next() {
    // **وكلُّ خطأٍ يُقال** — كان الحفظُ يسكت مع «١٠٠٠» أو «1,000» أو بلا باب.
    if (!categoryID) return setError(X.noCategory);
    if (parsed.value === null) return setError(parsed.error);
    if (spentAt > (today || damascusDay())) return setError(X.futureDate);
    if (over && !receipt) return setError(m.errors.expense_receipt_required);
    setError("");
    setReview(true);
  }

  async function submit() {
    if (busy || parsed.value === null) return;
    setBusy(true);
    setError("");
    try {
      const r = await api<{ id: string; status: string }>("/api/v1/admin/expenses", {
        method: "POST",
        body: JSON.stringify({
          category_id: categoryID,
          amount: parsed.value,
          note: note.trim(),
          spent_at: spentAt,
          receipt_media_id: receipt || undefined,
        }),
      });
      onDone(r?.status === "pending");
    } catch (err) {
      setError(errorText(err));
      setReview(false);
    } finally {
      setBusy(false);
    }
  }

  if (review && parsed.value !== null) {
    const cat = cats.find((c) => c.id === categoryID);
    return (
      <Modal open onClose={onClose} title={X.reviewTitle}>
        <div className="space-y-3">
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between gap-3">
              <dt className="text-ink-muted">{X.category}</dt>
              <dd className="font-medium">{cat?.name}</dd>
            </div>
            <div className="flex justify-between gap-3">
              <dt className="text-ink-muted">{X.reviewAmount}</dt>
              <dd dir="ltr" className="figure text-danger">
                {fmtNum(parsed.value)} {CUR}
              </dd>
            </div>
            <div className="flex justify-between gap-3">
              <dt className="text-ink-muted">{X.spentAt}</dt>
              <dd dir="ltr">{fmtDate(spentAt)}</dd>
            </div>
            {!over && (
              <div className="flex justify-between gap-3">
                <dt className="text-ink-muted">{X.reviewBalanceAfter}</dt>
                <dd dir="ltr" className="font-bold">
                  {fmtNum(balance - parsed.value)} {CUR}
                </dd>
              </div>
            )}
          </dl>
          <Alert tone={over ? "warning" : "info"}>
            {over
              ? X.reviewNeedsApproval.replace("{cap}", `${fmtNum(threshold)} ${CUR}`)
              : X.reviewDirect}
          </Alert>
          {error && <Alert>{error}</Alert>}
          <FormActions
            onSave={() => void submit()}
            onCancel={() => setReview(false)}
            busy={busy}
            saveLabel={X.confirmSave}
            cancelLabel={X.back}
          />
        </div>
      </Modal>
    );
  }

  return (
    <Modal open onClose={onClose} title={X.add}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{X.addHint}</p>
        {cats.length === 0 && <Alert>{X.noCategory}</Alert>}
        <Select
          id="exp-cat"
          label={X.category}
          required
          value={categoryID}
          onChange={(e) => setCategoryID(e.target.value)}
        >
          {cats.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </Select>
        <Input
          id="exp-amount"
          label={`${X.amount} (${CUR})`}
          required
          inputMode="numeric"
          dir="ltr"
          value={amount}
          error={amount !== "" && parsed.value === null ? parsed.error : undefined}
          onChange={(e) => setAmount(e.target.value)}
        />
        <Input
          id="exp-date"
          type="date"
          label={X.spentAt}
          max={today || damascusDay()}
          value={spentAt}
          onChange={(e) => setSpentAt(e.target.value)}
        />
        <Textarea
          id="exp-note"
          label={X.note}
          rows={2}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        <ImageUpload
          kind="expense_receipt"
          label={over ? X.receiptRequired : X.receiptOptional}
          path="/api/v1/admin/expenses/receipt"
          onChange={setReceipt}
          api={api}
          mediaUrl={mediaUrl}
          errorText={errorText}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions onSave={next} onCancel={onClose} />
      </div>
    </Modal>
  );
}

/**
 * **أبوابُ المصروف — إضافةٌ وتعديلُ اسمٍ وترتيبٌ وتفعيلٌ وإطفاء، ومع كلّ بابٍ مبلغُه.**
 *
 * **ولا يُحذف بابٌ صُرف عليه** — يُطفأ، وحذفُه يمحو تبويبَ ما مضى.
 */
function CatsModal({
  cats,
  onClose,
  onChanged,
}: {
  cats: Cat[];
  onClose: () => void;
  onChanged: () => void;
}) {
  const [name, setName] = useState("");
  const [editing, setEditing] = useState<{ id: string; name: string } | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function save(body: Record<string, unknown>) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/expenses/categories", {
        method: "POST",
        body: JSON.stringify(body),
      });
      setName("");
      setEditing(null);
      onChanged();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  /** **الترتيبُ بتبديل مكانَي بابين متجاورين.** */
  async function move(i: number, dir: -1 | 1) {
    const a = cats[i];
    const b = cats[i + dir];
    if (!a || !b || busy) return;
    const sa = a.sort_order === b.sort_order ? a.sort_order + dir : b.sort_order;
    const sb = a.sort_order;
    await save({ id: a.id, sort_order: sa });
    await save({ id: b.id, sort_order: sb });
  }

  return (
    <Modal open onClose={onClose} title={X.manageCats}>
      <div className="space-y-3">
        <div className="flex gap-2">
          <Input id="cat-name" label={X.catName} value={name} onChange={(e) => setName(e.target.value)} />
          <Button
            className="self-end"
            disabled={busy || !name.trim()}
            onClick={() => void save({ name: name.trim() })}
          >
            {m.common.save}
          </Button>
        </div>
        {error && <Alert>{error}</Alert>}
        <ul className="divide-y divide-line">
          {cats.map((c, i) => (
            <li key={c.id} className="flex flex-wrap items-center gap-2 py-2">
              {editing?.id === c.id ? (
                <div className="flex min-w-0 flex-1 gap-2">
                  <Input
                    id={`cat-edit-${c.id}`}
                    label={X.catRename}
                    value={editing.name}
                    onChange={(e) => setEditing({ id: c.id, name: e.target.value })}
                  />
                  <Button
                    className="self-end"
                    disabled={busy || !editing.name.trim()}
                    onClick={() => void save({ id: c.id, name: editing.name.trim() })}
                  >
                    {m.common.save}
                  </Button>
                </div>
              ) : (
                <span className="flex min-w-0 flex-1 flex-col">
                  <span className="truncate font-medium">{c.name}</span>
                  {c.used > 0 && (
                    <span className="text-xs text-ink-muted">
                      {X.usedCount.replace("{n}", fmtNum(c.used))} ·{" "}
                      {X.catSpent.replace("{amount}", `${fmtNum(c.spent)} ${CUR}`)}
                    </span>
                  )}
                </span>
              )}
              <Button
                variant="ghost"
                className="!px-2 text-xs"
                aria-label={X.catRename}
                onClick={() => setEditing({ id: c.id, name: c.name })}
              >
                <IconEdit size={14} />
              </Button>
              <Button variant="ghost" className="!px-2 text-xs" disabled={i === 0} onClick={() => void move(i, -1)}>
                {X.catUp}
              </Button>
              <Button
                variant="ghost"
                className="!px-2 text-xs"
                disabled={i === cats.length - 1}
                onClick={() => void move(i, 1)}
              >
                {X.catDown}
              </Button>
              <Checkbox
                id={`cat-${c.id}`}
                label={X.catActive}
                checked={c.active}
                onChange={(e) => void save({ id: c.id, active: e.target.checked })}
              />
            </li>
          ))}
        </ul>
      </div>
    </Modal>
  );
}
