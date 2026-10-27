"use client";

/**
 * **كشفُ حساب الخزينة برصيدٍ جارٍ** (قرارُ المالك ٢٠٢٦-١٠-٠٤) — كلُّ قيدٍ في الخزينة
 * والرصيدُ بعده، بمدّةٍ ونوع، **وتصديرُه ملفّاً بيوم دمشق.** وسحبُ الأدمن يظهر
 * هنا باسمه («سحبُ الأدمن من رصيد الخزينة»).
 *
 * # قرارُ المالك ٢٠٢٦-١٠-١٥ — عرضٌ والدفترُ لا يتغيّر
 *
 * **قيودُ ربح الطلب الواحد سطرٌ واحد** «ربح طلب #N: X» — ينفتح على تفصيله من أعمدة الطلب
 * ودفتره (`/treasury/statement/orders/{id}`): هامشٌ وعمولةٌ وتوصيلٌ ناقصاً السائقَ والمندوبَ
 * والخصم. **والسطورُ تُجمع إلى مبلغ السطر** — وما لا تشرحه سطرُ «فرق» ظاهر.
 * **وحركاتُ الأدمن الشخصيّة** (طلباته وشحنُه ومستردُّه) **تُوسَم «حركة شخصية»** — ليست ربحاً.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtDateTime, errorText, damascusDay } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Chips,
  Input,
  LoadingState,
  Pagination,
  ReloadState,
  Select,
  IconReceipt,
  IconArrowIn,
  IconArrowOut,
  useLiveData,
} from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { T, Signed, Amount, Equation, EmptyCard, dayStartsLine, shiftDay } from "./shared";

const m = getMessages(defaultLocale);
const S = T.statement;

interface Line {
  id: number;
  created_at: string;
  kind: string;
  kind_ar: string;
  amount: number;
  balance: number;
  note: string;
  ref: string;
  order_number: number | null;
  by_name: string;
  /** **«حركة شخصية»** — من حساب الأدمن نفسِه، ليست ربحاً. */
  personal?: boolean;
  /** **«ربح طلب #N»** — قيودُ الطلب الواحد مجمَّعة (`entries` عددُها). */
  grouped?: boolean;
  entries?: number;
  order_id?: string;
}

interface Breakdown {
  total: number;
  lines: { key: string; amount: number }[];
  entries: { id: number; created_at: string; amount: number; note: string }[];
}

/** **تفصيلُ «ربح طلب #N»** — يُجلب حين يُفتح، بمدى الكشف نفسِه فيُجمع إلى مبلغ السطر. */
function OrderBreakdown({ orderID, range }: { orderID: string; range: string }) {
  const { data, error, reload } = useLiveData<Breakdown>(
    () => api(`/api/v1/admin/treasury/statement/orders/${orderID}?${range}`),
    ["wallet"],
    [orderID, range],
  );
  if (error && !data) return <ReloadState onRetry={reload} />;
  if (!data) return <LoadingState variant="text" />;
  const label = S.breakdownLines as Record<string, string>;
  return (
    <div className="space-y-2 p-2">
      <ul className="space-y-1 text-sm">
        {data.lines.map((l) => (
          <li key={l.key} className={`flex justify-between gap-3 ${l.key === "diff" ? "font-bold text-warning" : ""}`}>
            <span>{label[l.key] ?? l.key}</span>
            <Signed value={l.amount} />
          </li>
        ))}
        <li className="flex justify-between gap-3 border-t border-line-soft pt-1 font-bold">
          <span>{S.colAmount}</span>
          <Signed value={data.total} />
        </li>
      </ul>
      {data.entries.length > 0 && (
        <details className="text-xs text-ink-muted">
          <summary className="cursor-pointer">{S.breakdownLedger}</summary>
          <ul className="mt-1 space-y-0.5">
            {data.entries.map((e) => (
              <li key={e.id} className="flex flex-wrap justify-between gap-2">
                <span dir="ltr">{fmtDateTime(e.created_at)}</span>
                <span className="min-w-0 flex-1 truncate">{e.note}</span>
                <Signed value={e.amount} />
              </li>
            ))}
          </ul>
        </details>
      )}
    </div>
  );
}

type Preset = "all" | "today" | "week" | "month" | "custom";
const PRESETS: Preset[] = ["all", "today", "week", "month", "custom"];

interface StatementData {
  lines: Line[];
  total: number;
  per_page: number;
  opening: number;
  closing: number;
  in: number;
  out: number;
  /** **الأنواعُ التي مرّت بالخزينة** — لفلتر النوع. */
  kinds?: string[];
  /** **يومُ الحساب اليوم** — لأزرار المدّة. */
  today?: string;
  day_start_hour?: number;
}

export function StatementTab() {
  const canCall = useCanCall();
  const canExport = canCall("GET", "/treasury/statement/export");
  const [preset, setPreset] = useState<Preset>("all");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [kind, setKind] = useState("");
  const [page, setPage] = useState(1);
  const [fail, setFail] = useState("");
  const [open, setOpen] = useState<string | null>(null);

  const qs = new URLSearchParams();
  if (from) qs.set("from", from);
  if (to) qs.set("to", to);
  if (kind) qs.set("kind", kind);

  const { data, error, reload } = useLiveData<StatementData>(
    () => api(`/api/v1/admin/treasury/statement?${qs.toString()}&page=${page}`),
    ["wallet"],
    [from, to, kind, page],
  );

  /**
   * **أزرارُ المدّة بيوم المنصّة** (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) — «اليوم» يومُ الحساب الذي يردّه المحرّك
   * (يبدأ بساعة الفتح)، لا يومُ المتصفّح؛ والافتراضُ «من أول يوم» فيصل اليومُ مع أوّل ردّ.
   */
  function pickPreset(p: Preset) {
    setPreset(p);
    setPage(1);
    if (p === "custom") return;
    const today = data?.today ?? damascusDay();
    if (p === "all") {
      setFrom("");
      setTo("");
    } else if (p === "today") {
      setFrom(today);
      setTo(today);
    } else if (p === "week") {
      setFrom(shiftDay(today, -6));
      setTo(today);
    } else {
      setFrom(`${today.slice(0, 8)}01`);
      setTo(today);
    }
  }

  function download() {
    void (async () => {
      try {
        const res = await apiFile(`/api/v1/admin/treasury/statement/export?${qs.toString()}`);
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `treasury-${from || "start"}_${to || "today"}.csv`;
        a.click();
        URL.revokeObjectURL(url);
      } catch (err) {
        setFail(errorText(err));
      }
    })();
  }

  const allKinds = m.shared.txKinds as Record<string, string>;
  // **الأنواعُ المستعملةُ وحدَها** — والمختارُ يبقى في القائمة ولو خلت مدّتُه.
  const used = data?.kinds ?? Object.keys(allKinds);
  const kindKeys = kind && !used.includes(kind) ? [kind, ...used] : used;
  const kindName = (k: string) => allKinds[k] ?? k;
  // **مدى الكشف بلا النوع** — للتفصيل، فمجموعُه هو مبلغُ السطر.
  const range = new URLSearchParams();
  if (from) range.set("from", from);
  if (to) range.set("to", to);
  const Q = S.quick as Record<Preset, string>;

  return (
    <div className="space-y-4">
      {fail && <Alert onDismiss={() => setFail("")}>{fail}</Alert>}

      {/* **الفلاترُ ببطاقةٍ واحدة** — حبّاتُ المدّة أوّلاً، ثمّ التاريخان لمدّةٍ محدّدة، ثمّ النوع والتنزيل. */}
      <section className="surface space-y-3 p-4">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm text-ink-muted">{S.period}:</span>
          <Chips wrap items={PRESETS.map((p) => ({ id: p, label: Q[p] }))} value={preset} onChange={pickPreset} />
        </div>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4 lg:items-end">
          {preset === "custom" && (
            <>
              <Input
                id="ts-from"
                type="date"
                label={m.shared.statement.from}
                value={from}
                onChange={(e) => {
                  setFrom(e.target.value);
                  setPage(1);
                }}
              />
              <Input
                id="ts-to"
                type="date"
                label={m.shared.statement.to}
                value={to}
                onChange={(e) => {
                  setTo(e.target.value);
                  setPage(1);
                }}
              />
            </>
          )}
          <Select
            id="ts-kind"
            label={S.kind}
            value={kind}
            onChange={(e) => {
              setKind(e.target.value);
              setPage(1);
            }}
          >
            <option value="">{S.allKinds}</option>
            {kindKeys.map((k) => (
              <option key={k} value={k}>
                {kindName(k)}
              </option>
            ))}
          </Select>
          {canExport && (
            <Button variant="secondary" onClick={download} className="w-full">
              {S.export}
            </Button>
          )}
        </div>
        <p className="text-2xs text-ink-muted">
          {dayStartsLine(data?.day_start_hour)} · {S.tz}
        </p>
      </section>

      {error && !data ? (
        <ReloadState onRetry={reload} />
      ) : !data ? (
        <LoadingState />
      ) : (
        <>
          <section className="surface space-y-3 p-4">
            {kind ? (
              <>
                {/* **نوعٌ مختار ⇒ مجموعُه وحدَه** (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) — رصيدا البداية والنهاية للمحفظة
                    كلِّها لا للنوع، فكانت المعادلةُ تُقرأ «٠ + ٠ − ٠ = −١٢٧». */}
                <h3 className="heading-card">{S.kindTotal.replace("{kind}", kindName(kind))}</h3>
                <Equation
                  terms={[
                    { label: S.in, value: data.in, tone: "success" },
                    { label: S.out, value: data.out, tone: "danger", op: "−" },
                    {
                      label: S.kindNet,
                      value: data.in - data.out,
                      op: "=",
                      result: true,
                      signed: true,
                      tone: data.in - data.out < 0 ? "danger" : data.in - data.out > 0 ? "success" : undefined,
                    },
                  ]}
                />
                <p className="text-xs text-ink-muted">{S.kindHint}</p>
              </>
            ) : (
              <>
                {/* **المعادلة** — البدايةُ + الداخل − الطالع = النهاية، **والداخلُ والطالعُ مجموعُ الأسطر تحتها.** */}
                <Equation
                  terms={[
                    { label: S.opening, value: data.opening },
                    { label: S.in, value: data.in, tone: "success", op: "+", hint: S.inHint },
                    { label: S.out, value: data.out, tone: "danger", op: "−", hint: S.outHint },
                    { label: S.closing, value: data.closing, op: "=", result: true, tone: data.closing < 0 ? "danger" : "success" },
                  ]}
                />
                <p className="text-xs text-ink-muted">{S.howTo}</p>
              </>
            )}
            <p className="text-2xs text-ink-muted">
              {S.personal}: {S.personalHint}
            </p>
          </section>

          {data.lines.length === 0 ? (
            <EmptyCard icon={IconReceipt} title={S.empty} hint={S.emptyHint} tone="muted" />
          ) : (
            <ul className="space-y-2">
              {data.lines.map((l) => {
                const key = `${l.id}-${l.order_id ?? ""}`;
                const isOpen = open === key;
                const canOpen = !!(l.grouped && l.order_id);
                const up = l.amount >= 0;
                return (
                  <li key={key} className="surface overflow-hidden">
                    <button
                      type="button"
                      disabled={!canOpen}
                      onClick={() => setOpen(isOpen ? null : key)}
                      className="flex w-full items-center gap-3 p-3 text-start transition-colors enabled:hover:bg-row-hover"
                    >
                      <span
                        className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-on-solid ${
                          up ? "bg-success-solid" : "bg-danger-solid"
                        }`}
                      >
                        {up ? <IconArrowIn size={18} /> : <IconArrowOut size={18} />}
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="flex flex-wrap items-center gap-1.5 font-bold">
                          {l.grouped && l.order_number != null
                            ? S.orderProfit.replace("{n}", String(l.order_number))
                            : l.kind_ar}
                          {l.personal && <Badge variant="neutral">{S.personal}</Badge>}
                        </span>
                        <span className="block truncate text-xs text-ink-muted">
                          <span dir="ltr">{fmtDateTime(l.created_at)}</span>
                          {!l.grouped && l.by_name && ` · ${S.colBy} ${l.by_name}`}
                          {!l.grouped && l.note && ` · ${l.note}`}
                        </span>
                      </span>
                      <span className="shrink-0 text-end">
                        <span className={`figure block leading-none ${up ? "text-success" : "text-danger"}`}>
                          <Signed value={l.amount} />
                        </span>
                        <span className="block text-2xs text-ink-muted">
                          {S.colBalance}: <Amount value={l.balance} />
                        </span>
                      </span>
                      {canOpen && <span className="shrink-0 text-xs text-ink-muted">{isOpen ? "▴" : "▾"}</span>}
                    </button>
                    {isOpen && l.order_id && (
                      <div className="border-t border-line-soft bg-field">
                        <OrderBreakdown orderID={l.order_id} range={range.toString()} />
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
          {data.total > data.per_page && (
            <div className="flex justify-center">
              <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
            </div>
          )}
        </>
      )}
    </div>
  );
}
