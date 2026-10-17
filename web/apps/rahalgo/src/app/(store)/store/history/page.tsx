"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/history — «سجل الطلبات»: ما انتهى أمره**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `HistoryScreen.kt`: رقمُ الطلب · المجموعُ وخصمُ المنصّة والمستحقّ ·
 *  الأصناف · الحالة · **و«إبلاغ» عن السائق لما يُبلَّغ عنه** — `reportable`.)
 *
 * **وزاد الويبُ صفحاتٍ وتصفيةً بالحالة** — المحرّكُ يردّهما أصلاً
 * (`closed_only` · `status` · `page` · `status_counts`)، **والتطبيقُ يقرأ
 * الصفحةَ الأولى وحدَها.** سجلٌّ يطول بلا صفحاتٍ لا يُقرأ آخرُه.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDateTime } from "@rahalgo/i18n";
import { Button, Chips, EmptyState, IconReceipt, IconWarning, LoadingState, ReloadState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { MerchantOrder, OrdersPage } from "@/lib/store/types";
import { MoneyLines, orderStatus, ReportDriver } from "@/components/store/HistoryParts";

const m = getMessages(defaultLocale);
const H = m.store.history;

const PER_PAGE = 20;

export default function HistoryPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const [status, setStatus] = useState("");
  const [rows, setRows] = useState<MerchantOrder[] | null>(null);
  const [total, setTotal] = useState(0);
  const [counts, setCounts] = useState<Record<string, number>>({});
  const [reportable, setReportable] = useState<Set<string>>(new Set());
  const [page, setPage] = useState(1);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const [reporting, setReporting] = useState<MerchantOrder | null>(null);

  const fetchPage = useCallback(
    async (p: number, reset: boolean) => {
      if (!id) return;
      setBusy(true);
      setFailed(false);
      try {
        const q = new URLSearchParams({ closed_only: "true", page: String(p), per_page: String(PER_PAGE) });
        if (status) q.set("status", status);
        const r = await api<OrdersPage>(`/api/v1/merchant/stores/${id}/orders?${q.toString()}`);
        setRows((old) => (reset ? r.orders ?? [] : [...(old ?? []), ...(r.orders ?? [])]));
        setTotal(r.total ?? 0);
        if (!status) setCounts(r.status_counts ?? {});
        setReportable((old) => new Set([...(reset ? [] : [...old]), ...(r.reportable ?? [])]));
        setPage(p);
      } catch {
        setFailed(true);
      } finally {
        setBusy(false);
      }
    },
    [id, status],
  );
  useEffect(() => {
    setRows(null);
    void fetchPage(1, true);
  }, [fetchPage]);

  const chips = [
    { id: "", label: H.all },
    ...Object.entries(counts)
      .filter(([, n]) => n > 0)
      .map(([k, n]) => ({ id: k, label: orderStatus(k), count: n })),
  ];

  return (
    <div className="flex flex-col gap-3">
      <header>
        <h2 className="heading-page">{H.title}</h2>
        <p className="text-sm text-ink-muted">{H.hint}</p>
      </header>
      {chips.length > 2 && <Chips items={chips} value={status} onChange={setStatus} />}
      {failed && !rows && <ReloadState onRetry={() => void fetchPage(1, true)} />}
      {rows == null && !failed && <LoadingState variant="list" rows={4} />}
      {rows != null && rows.length === 0 && <EmptyState icon={IconReceipt} title={H.empty} />}
      <ul className="flex flex-col gap-2">
        {rows?.map((o) => (
          <li key={o.id} className="surface flex flex-col gap-2 rounded-card p-3" data-testid="history-order">
            <Link href={`/store/history/${o.id}`} className="flex flex-col gap-2">
              <header className="flex items-center justify-between gap-2">
                <span className="font-bold">{`#${o.number}`}</span>
                <span className={`text-xs ${o.status === "delivered" ? "text-accent-text" : "text-ink-muted"}`}>
                  {orderStatus(o.status)}
                </span>
              </header>
              <MoneyLines o={o} />
              {o.items.length > 0 && (
                <p className="text-xs text-ink-muted">{o.items.map((l) => `${l.qty}× ${l.name}`).join(m.common.listSep)}</p>
              )}
              <p className="flex items-center justify-between text-2xs text-ink-muted">
                <span>{fmtDateTime(o.closed_at ?? o.created_at)}</span>
                <span className="text-accent-text">{H.open}</span>
              </p>
            </Link>
            {reportable.has(o.id) && (
              <Button variant="ghost" className="flex w-fit items-center gap-1 text-danger" onClick={() => setReporting(o)}>
                <IconWarning size={14} />
                {H.report}
              </Button>
            )}
          </li>
        ))}
      </ul>
      {rows != null && rows.length < total && (
        <Button variant="secondary" disabled={busy} onClick={() => void fetchPage(page + 1, false)} data-testid="history-more">
          {H.more}
        </Button>
      )}
      {failed && rows && <ReloadState onRetry={() => void fetchPage(page + 1, false)} />}
      {reporting && (
        <ReportDriver
          order={reporting}
          onClose={() => setReporting(null)}
          onSent={() => {
            const done = reporting.id;
            setReporting(null);
            setReportable((s) => {
              const n = new Set(s);
              n.delete(done);
              return n;
            });
          }}
        />
      )}
    </div>
  );
}
