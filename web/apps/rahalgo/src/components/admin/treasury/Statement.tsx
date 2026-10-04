"use client";

/**
 * **كشفُ حساب الخزينة برصيدٍ جارٍ** (قرارُ المالك ٢٠٢٦-١٠-٠٤) — كلُّ قيدٍ في الخزينة
 * والرصيدُ بعده، بمدّةٍ ونوع، **وتصديرُه ملفّاً بيوم دمشق.** وسحبُ الأدمن يظهر
 * هنا باسمه («سحبُ الأدمن من رصيد الخزينة»).
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtDateTime, fmtMoney, errorText } from "@rahalgo/i18n";
import {
  Alert,
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
}

export function StatementTab() {
  const canCall = useCanCall();
  const canExport = canCall("GET", "/treasury/statement/export");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [kind, setKind] = useState("");
  const [page, setPage] = useState(1);
  const [fail, setFail] = useState("");

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
                  {data.lines.map((l) => (
                    <tr key={l.id} className="border-b border-line-soft">
                      <td className="whitespace-nowrap p-2" dir="ltr">
                        {fmtDateTime(l.created_at)}
                      </td>
                      <td className="p-2 font-medium">{l.kind_ar}</td>
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
                      <td className="max-w-xs truncate p-2 text-ink-muted">{l.note}</td>
                    </tr>
                  ))}
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

