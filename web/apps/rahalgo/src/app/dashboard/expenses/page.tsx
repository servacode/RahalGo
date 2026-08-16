"use client";

/**
 * **مصروفاتُ التشغيل — ما ينفقه المكتبُ لا ما يخسره العمل.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦: «يوجد مكتبٌ للشركة وموظّفون وعمّال وما إلى ذلك من
 *  المصاريف — يجب أن تُوثَّق بشكلٍ صحيح».)
 *
 * # ولماذا ليست في «الخسائر»
 *
 * **الخسارةُ ما لم يكن يجب أن يقع**: طلبٌ فشل فعُوّض سائقُه. **وإيجارُ المكتب
 * كلفةُ تشغيلٍ مخطَّطة.**
 *
 * **ولو خُلطا لَتضخّم تقريرُ الخسائر بالإيجار** — فيبدو أداءُ المنصّة أسوأَ
 * ممّا هو، **ولا يُعرف كم كلّف الفشلُ فعلاً**: وهو الرقمُ الذي تُتَّخذ عليه
 * قراراتُ الحظر والتعويض.
 *
 * # والمالُ يخرج من الخزينة
 *
 * (بقرار المالك: «نعم يخرج من خزينة المنصّة لأنّه مصروفٌ تابعٌ للمنصّة».)
 *
 * **فكلُّ مصروفٍ قيدان**: صفٌّ يقول على أيّ بابٍ صُرف، **وقيدٌ في الخزينة يقول
 * كم نقص رصيدُها.**
 *
 * # والتوزيعُ على الأبواب هو السؤال
 *
 * **ومجموعٌ واحدٌ لا يقول أين ذهب المال** — «كم على الرواتب وكم على الإيجار»
 * هو ما يُبنى عليه قرار.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, errorText } from "@rahalgo/i18n";
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
  EmptyState,
  LoadingState,
  ReloadState,
  Pagination,
  StatGrid,
  StatCard,
  Checkbox,
  IconWallet,
  IconAdd,
  IconStatus,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const X = m.admin.expenses;

interface Row {
  id: string;
  category: string;
  amount: number;
  note: string;
  spent_at: string;
  by: string;
}
interface Cat {
  id: string;
  name: string;
  active: boolean;
  sort_order: number;
  /** **كم صُرف على هذا الباب** — **ومن أراد إطفاءه يعرف ماذا يخفي.** */
  used: number;
}
interface Data {
  expenses: Row[];
  /** **عددُ البنود** — للترقيم. */
  total: number;
  /** **ومجموعُ المبلغ باسمٍ آخر** — **واسمٌ واحدٌ لمعنيين يجعل الترقيمَ
   *  يقرأ المبلغَ عددَ صفوف**: مليونٌ ونصف يصير أربعين ألفَ صفحة. */
  total_amount: number;
  page: number;
  per_page: number;
  by_category: { name: string; count: number; total: number }[];
}

/** **أوّلُ الشهر الجاري وآخرُه** — «كم أنفق المكتبُ هذا الشهر» أوّلُ سؤال. */
function monthRange(): { from: string; to: string } {
  const d = new Date();
  const p = (n: number) => String(n).padStart(2, "0");
  const first = `${d.getFullYear()}-${p(d.getMonth() + 1)}-01`;
  const last = new Date(d.getFullYear(), d.getMonth() + 1, 0);
  return {
    from: first,
    to: `${last.getFullYear()}-${p(last.getMonth() + 1)}-${p(last.getDate())}`,
  };
}

export default function ExpensesPage() {
  const [range, setRange] = useState(monthRange());
  const [page, setPage] = useState(1);
  const [data, setData] = useState<Data | null | "failed">(null);
  const [cats, setCats] = useState<Cat[]>([]);
  const [adding, setAdding] = useState(false);
  const [catsOpen, setCatsOpen] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api<Data>(
      `/api/v1/admin/expenses?from=${range.from}&to=${range.to}&page=${page}`,
    )
      .then(setData)
      // **والفشلُ ليس فراغاً** — **و«لا مصروفات» على قراءةٍ فشلت تُقرأ
      // شهراً بلا نفقة.**
      .catch(() => setData("failed"));
  }, [range, page]);

  const loadCats = useCallback(() => {
    api<{ categories: Cat[] }>("/api/v1/admin/expenses/categories")
      .then((r) => setCats(r.categories ?? []))
      .catch((err) => setError(errorText(err)));
  }, []);

  useEffect(load, [load]);
  useEffect(loadCats, [loadCats]);

  if (data === "failed") return <ReloadState onRetry={load} />;
  if (!data) return <LoadingState />;

  return (
    <PageContainer>
      <PageHeader
        icon={IconWallet}
        title={X.title}
        subtitle={X.hint}
        actions={
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => setCatsOpen(true)}>
              {X.manageCats}
            </Button>
            <Button onClick={() => setAdding(true)} className="flex items-center gap-1.5">
              <IconAdd size={16} />
              {X.add}
            </Button>
          </div>
        }
      />

      {error && <Alert className="mb-3">{error}</Alert>}

      <div className="mb-4 flex flex-wrap items-end gap-3">
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
      </div>

      <StatGrid>
        <StatCard
          label={`${X.total} (${m.common.currency})`}
          value={fmtNum(data.total_amount)}
          icon={IconWallet}
          tone={data.total_amount > 0 ? "danger" : "default"}
        />

      </StatGrid>

      {/* **والتوزيعُ على الأبواب** — **ومجموعٌ واحدٌ لا يقول أين ذهب المال.** */}
      {data.by_category.length > 0 && (
        <ul className="mb-4 flex flex-wrap gap-2">
          {data.by_category.map((c) => (
            <li key={c.name} className="rounded-control border border-line px-3 py-1.5 text-sm">
              <span className="font-medium">{c.name}</span>{" "}
              <span dir="ltr" className="font-bold tabular-nums text-danger">
                {fmtNum(c.total)}
              </span>{" "}
              <span className="text-xs text-ink-muted">({fmtNum(c.count)})</span>
            </li>
          ))}
        </ul>
      )}

      {data.expenses.length === 0 ? (
        <EmptyState icon={IconWallet} title={X.empty} />
      ) : (
        <ul className="divide-y divide-line surface">
          {data.expenses.map((e) => (
            <li key={e.id} className="flex flex-wrap items-center gap-3 px-3 py-2.5">
              <Badge variant="neutral">{e.category}</Badge>
              <span dir="ltr" className="shrink-0 font-bold tabular-nums text-danger">
                {fmtNum(e.amount)}
              </span>
              <span className="min-w-0 flex-1 truncate text-sm text-ink-muted">{e.note}</span>
              <span className="shrink-0 text-xs text-ink-muted">{e.by}</span>
              <span dir="ltr" className="shrink-0 text-xs text-ink-muted">
                {fmtDate(e.spent_at)}
              </span>
              {/* **والخطأُ يُلغى ولا يُمحى** — **وحذفُ الصفّ يترك قيدَه في
                  الخزينة بلا صاحب**: مالٌ خرج ولا يُعرف لماذا. */}
              <Button
                variant="ghost"
                className="!px-2 text-xs"
                onClick={() => {
                  void (async () => {
                    try {
                      await api(`/api/v1/admin/expenses/${e.id}/void`, { method: "POST" });
                      setError("");
                    } catch (err) {
                      setError(errorText(err));
                    }
                    load();
                  })();
                }}
              >
                {X.void}
              </Button>
            </li>
          ))}
        </ul>
      )}

      <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />

      {adding && (
        <AddModal
          cats={cats.filter((c) => c.active)}
          onClose={() => setAdding(false)}
          onDone={() => {
            setAdding(false);
            load();
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
    </PageContainer>
  );
}

/** **تسجيلٌ يدويٌّ** — (قرارُ المالك: «أسجّل كلَّ شيءٍ بيدي»). */
function AddModal({
  cats,
  onClose,
  onDone,
}: {
  cats: Cat[];
  onClose: () => void;
  onDone: () => void;
}) {
  const [categoryID, setCategoryID] = useState(cats[0]?.id ?? "");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  /** **ويومُ الصرف غيرُ يومِ التسجيل** — يُسجَّل إيجارُ الشهر الماضي اليوم. */
  const [spentAt, setSpentAt] = useState(new Date().toISOString().slice(0, 10));
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit() {
    const n = Number(amount);
    if (!categoryID || !Number.isFinite(n) || n <= 0 || busy) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/expenses", {
        method: "POST",
        body: JSON.stringify({
          category_id: categoryID,
          amount: n,
          note: note.trim(),
          spent_at: spentAt,
        }),
      });
      onDone();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={X.add}>
      <div className="space-y-3">
        {/* **ويُقال إنّه يخرج من الخزينة** — **ومن لم يعلم ظنّه دفتراً
            جانبيّاً** فسجّل ما دفعه من جيبه. */}
        <p className="text-sm text-ink-muted">{X.addHint}</p>
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
          label={`${X.amount} (${m.common.currency})`}
          required
          inputMode="numeric"
          value={amount}
          onChange={(e) => setAmount(e.target.value)}
        />
        <Input
          id="exp-date"
          type="date"
          label={X.spentAt}
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
        {error && <Alert>{error}</Alert>}
        <div className="flex gap-2">
          <Button variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button disabled={busy || !amount.trim() || !categoryID} onClick={() => void submit()}>
            {busy ? m.common.loading : m.common.save}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

/**
 * **أبوابُ المصروف — تُدار من الشاشة.**
 *
 * (قرارُ المالك: «قائمةٌ ويمكنني الإضافة والحذف والتعديل — أفضلُ من قائمةٍ
 *  ثابتة».)
 *
 * **ولا يُحذف بابٌ صُرف عليه** — يُطفأ فلا يُختار جديداً، **وحذفُه يمحو تبويبَ
 * ما مضى** فيُقرأ تاريخُ الإنفاق ناقصاً.
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
      onChanged();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={X.manageCats}>
      <div className="space-y-3">
        <div className="flex gap-2">
          <Input
            id="cat-name"
            label={X.catName}
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
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
          {cats.map((c) => (
            <li key={c.id} className="flex items-center gap-3 py-2">
              <span className="min-w-0 flex-1 truncate">{c.name}</span>
              {c.used > 0 && (
                <span className="shrink-0 text-xs text-ink-muted">
                  {X.usedCount.replace("{n}", fmtNum(c.used))}
                </span>
              )}
              {/* **ويُطفأ ولا يُحذف** — **وبابٌ محذوفٌ يمحو تبويبَ ما صُرف
                  عليه**، فيُقرأ تاريخُ الإنفاق ناقصاً. */}
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
