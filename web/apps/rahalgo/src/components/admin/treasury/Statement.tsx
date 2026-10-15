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

import { Fragment, useState } from "react";
import { getMessages, defaultLocale, fmtDateTime, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  EmptyState,
  Input,
  LoadingState,
  Pagination,
  ReloadState,
  Select,
  StatCard,
  StatGrid,
  IconWallet,
  useLiveData,
} from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { T, Signed, Amount } from "./shared";

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

export function StatementTab() {
  const canCall = useCanCall();
  const canExport = canCall("GET", "/treasury/statement/export");
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

  const { data, error, reload } = useLiveData<{
    lines: Line[];
    total: number;
    per_page: number;
    opening: number;
    closing: number;
    in: number;
    out: number;
  }>(() => api(`/api/v1/admin/treasury/statement?${qs.toString()}&page=${page}`), ["wallet"], [from, to, kind, page]);

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

  const kinds = Object.entries(m.shared.txKinds as Record<string, string>);
  // **مدى الكشف بلا النوع** — للتفصيل، فمجموعُه هو مبلغُ السطر.
  const range = new URLSearchParams();
  if (from) range.set("from", from);
  if (to) range.set("to", to);

  return (
    <div className="space-y-4">
      {fail && <Alert onDismiss={() => setFail("")}>{fail}</Alert>}
      <div className="flex flex-wrap items-end gap-3">
        <div className="w-40">
          <Input id="ts-from" type="date" label={m.shared.statement.from} value={from}
            onChange={(e) => { setFrom(e.target.value); setPage(1); }} />
        </div>
        <div className="w-40">
          <Input id="ts-to" type="date" label={m.shared.statement.to} value={to}
            onChange={(e) => { setTo(e.target.value); setPage(1); }} />
        </div>
        <div className="w-48">
          <Select id="ts-kind" label={S.kind} value={kind} onChange={(e) => { setKind(e.target.value); setPage(1); }}>
            <option value="">{S.allKinds}</option>
            {kinds.map(([k, label]) => (
              <option key={k} value={k}>
                {label}
              </option>
            ))}
          </Select>
        </div>
        {canExport && (
          <Button variant="secondary" onClick={download}>
            {S.export}
          </Button>
        )}
      </div>
      <p className="text-xs text-ink-muted">{S.tz}</p>
      <p className="text-xs text-ink-muted">
        {S.personal}: {S.personalHint}
      </p>

      {error && !data ? (
        <ReloadState onRetry={reload} />
      ) : !data ? (
        <LoadingState />
      ) : (
        <>
          <StatGrid>
            <StatCard label={S.opening} value={fmtMoney(data.opening)} icon={IconWallet} />
            <StatCard label={S.in} value={fmtMoney(data.in)} tone="success" />
            <StatCard label={S.out} value={fmtMoney(data.out)} tone="danger" />
            <StatCard label={S.closing} value={fmtMoney(data.closing)} emphasis />
          </StatGrid>
          {data.lines.length === 0 ? (
            <EmptyState icon={IconWallet} title={S.empty} />
          ) : (
            <div className="surface overflow-x-auto">
              <table className="w-full text-sm">
                <thead className="text-2xs text-ink-muted">
                  <tr className="border-b border-line-soft">
                    <th className="p-2 text-start">{S.colDate}</th>
                    <th className="p-2 text-start">{S.colKind}</th>
                    <th className="p-2 text-start">{S.colAmount}</th>
                    <th className="p-2 text-start">{S.colBalance}</th>
                    <th className="p-2 text-start">{S.colOrder}</th>
                    <th className="p-2 text-start">{S.colBy}</th>
                    <th className="p-2 text-start">{S.colNote}</th>
                  </tr>
                </thead>
                <tbody>
                  {data.lines.map((l) => {
                    const key = `${l.id}-${l.order_id ?? ""}`;
                    const isOpen = open === key;
                    return (
                      <Fragment key={key}>
                        <tr className="border-b border-line-soft">
                          <td className="whitespace-nowrap p-2" dir="ltr">
                            {fmtDateTime(l.created_at)}
                          </td>
                          <td className="p-2 font-medium">
                            <span className="flex flex-wrap items-center gap-1.5">
                              {l.grouped && l.order_number != null
                                ? S.orderProfit.replace("{n}", String(l.order_number))
                                : l.kind_ar}
                              {l.personal && (
                                <Badge variant="neutral">
                                  {S.personal}
                                </Badge>
                              )}
                            </span>
                          </td>
                          <td className="p-2">
                            <Signed value={l.amount} />
                          </td>
                          <td className="p-2">
                            <Amount value={l.balance} />
                          </td>
                          <td className="p-2" dir="ltr">
                            {l.order_number ?? ""}
                          </td>
                          <td className="p-2">{l.by_name}</td>
                          <td className="max-w-xs p-2 text-ink-muted">
                            {l.grouped && l.order_id ? (
                              <span className="flex flex-wrap items-center gap-2">
                                {(l.entries ?? 0) > 1 && (
                                  <span className="text-xs">{S.entries.replace("{n}", String(l.entries))}</span>
                                )}
                                <Button variant="secondary" className="!px-2.5" onClick={() => setOpen(isOpen ? null : key)}>
                                  {isOpen ? S.hideBreakdown : S.showBreakdown}
                                </Button>
                              </span>
                            ) : (
                              <span className="block truncate">{l.note}</span>
                            )}
                          </td>
                        </tr>
                        {isOpen && l.order_id && (
                          <tr className="border-b border-line-soft">
                            <td colSpan={7}>
                              <OrderBreakdown orderID={l.order_id} range={range.toString()} />
                            </td>
                          </tr>
                        )}
                      </Fragment>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </>
      )}
    </div>
  );
}

