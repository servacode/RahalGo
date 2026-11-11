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
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDamascusTime, fmtMoney, fmtNum, fmtRelDay } from "@rahalgo/i18n";
import { Badge, Button, Chip, DataView, EmptyState, IconReceipt, IconWarning, InvoiceLines, LoadingState, ReloadState, useViewMode } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { noPayout, type MerchantOrder, type OrdersPage } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { MoneyLines, orderStatus, ReportDriver } from "@/components/store/HistoryParts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const S = m.store;
const H = S.history;
const K = S.console;

const PER_PAGE = 20;

export default function HistoryPage() {
  const { store } = useStore();
  const router = useRouter();
  /** **جدولٌ فوق ٧٦٨ بكسل وقائمةٌ تحتها** — نفسُ مفتاح الشاشة المعتمد في اللوحة. */
  const [, , { narrow }] = useViewMode("store-history", "table");
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
        const q = new URLSearchParams({ with_active: "true", page: String(p), per_page: String(PER_PAGE) });
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
    { id: "", label: H.all, count: undefined as number | undefined },
    ...Object.entries(counts)
      .filter(([, n]) => n > 0)
      .map(([k, n]) => ({ id: k, label: orderStatus(k), count: n as number | undefined })),
  ];

  /** **شارةُ الحالة** — لونٌ ونصٌّ معاً: تسليمٌ أخضر، وما انتهى بلا بيعٍ أحمر، والباقي رماديّ. */
  const statusBadge = (st: string) => (
    <Badge variant={st === "delivered" ? "success" : noPayout(st) ? "danger" : "neutral"}>{orderStatus(st)}</Badge>
  );
  const itemsText = (o: MerchantOrder) => o.items.map((i) => `${i.name} × ${fmtNum(i.qty)}`).join(m.common.listSep);

  return (
    <div className="console-page" data-testid="store-history">
      <PageHeader title={H.title} subtitle={total > 0 ? fill(K.histCount, { n: fmtNum(total) }) : H.hint} />
      {chips.length > 2 && (
        <div className="rail-x -mx-rg-md px-rg-md lg:mx-0 lg:px-0" aria-label={H.all}>
          {chips.map((c) => (
            <Chip key={c.id} selected={status === c.id} count={c.count} onClick={() => setStatus(c.id)}>
              {c.label}
            </Chip>
          ))}
        </div>
      )}
      {failed && !rows && <ReloadState onRetry={() => void fetchPage(1, true)} />}
      {rows == null && !failed && <LoadingState variant="list" rows={4} />}
      {rows != null && rows.length === 0 && <EmptyState icon={IconReceipt} title={H.empty} />}

      {/* ── الحاسوب: جدولٌ كثيفٌ، والصفُّ يفتح التفاصيل ── */}
      {rows != null && rows.length > 0 && !narrow && (
        <DataView<MerchantOrder>
          dense
          view="table"
          items={rows}
          getKey={(o) => o.id}
          empty={H.empty}
          onRowClick={(o) => router.push(`/store/history/${o.id}`)}
          columns={[
            {
              id: "number",
              header: K.colOrder,
              widthClass: "w-28",
              cell: (o) => (
                <Link
                  href={`/store/history/${o.id}`}
                  onClick={(e) => e.stopPropagation()}
                  className="font-bold text-ink hover:underline"
                  data-testid="history-order"
                >
                  <bdi>{`#${o.number}`}</bdi>
                </Link>
              ),
            },
            {
              id: "when",
              header: K.colWhen,
              widthClass: "w-44",
              cell: (o) => (
                <span className="flex flex-col">
                  <span>{fmtRelDay(o.created_at)}</span>
                  <span className="text-caption text-ink-muted" dir="ltr">
                    {fmtDamascusTime(o.created_at)}
                  </span>
                </span>
              ),
            },
            { id: "status", header: K.colStatus, widthClass: "w-36", cell: (o) => statusBadge(o.status) },
            {
              id: "items",
              header: K.colItems,
              cell: (o) => <span className="line-clamp-2 text-ink-muted">{itemsText(o) || "—"}</span>,
            },
            {
              id: "sum",
              header: S.orders.sum,
              align: "end",
              widthClass: "w-32",
              cell: (o) => (noPayout(o.status) ? <span className="text-ink-muted">—</span> : fmtMoney(o.subtotal)),
            },
            {
              id: "cut",
              header: K.colCommission,
              align: "end",
              widthClass: "w-32",
              cell: (o) => (noPayout(o.status) ? <span className="text-ink-muted">—</span> : <span className="text-ink-muted">{`− ${fmtMoney(o.platform_commission)}`}</span>),
            },
            {
              id: "due",
              header: S.orders.due,
              align: "end",
              widthClass: "w-36",
              cell: (o) => (noPayout(o.status) ? <span className="text-ink-muted">—</span> : <span className="font-bold text-accent-text">{fmtMoney(o.merchant_net)}</span>),
            },
          ]}
          actions={(o) =>
            reportable.has(o.id) ? (
              <Button variant="ghost" className="flex min-h-9 items-center gap-rg-xs text-danger" onClick={() => setReporting(o)}>
                <IconWarning size={14} />
                {H.report}
              </Button>
            ) : null
          }
        />
      )}

      {/* ── الجوّال: بعناوين أيّامٍ وبطاقةٍ لكلّ طلب (مرآةُ `HistoryScreen.kt`، تدقيقُ ٢٠٢٦-١٠-٢٧) —
          «طلبُ مبارح العصر» يُوجد تحت يومه، وبرأسه ساعتُه. ── */}
      {rows != null && rows.length > 0 && narrow && (
        <ul className="flex flex-col gap-rg-sm md:grid md:grid-cols-2 md:items-start">
          {rows.map((o, i, all) => {
            const day = fmtRelDay(o.created_at);
            const prev = all[i - 1];
            const head = !prev || fmtRelDay(prev.created_at) !== day;
            return (
              <li key={o.id} className="flex flex-col gap-rg-sm">
                {head && <h3 className="console-h2 px-rg-xs pt-rg-sm text-ink-muted">{day}</h3>}
                <div className="panel-lite flex flex-col gap-rg-sm p-rg-md" data-testid="history-order">
                  <Link href={`/store/history/${o.id}`} className="flex flex-col gap-rg-sm">
                    <header className="flex items-center justify-between gap-rg-sm">
                      <span className="flex items-baseline gap-rg-sm">
                        <span className="font-bold">{`#${o.number}`}</span>
                        <span className="text-caption text-ink-muted" dir="ltr">
                          {fmtDamascusTime(o.created_at)}
                        </span>
                      </span>
                      {statusBadge(o.status)}
                    </header>
                    {/* **سطرٌ لكلّ عنصر ثمّ الحسبة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — «بندورة × 1 كيلو ...... 600 ل.س»،
                        **وبما اختاره الزبون** (تجربةُ المالك ٢٠٢٦-١٠-٢٩) — «بندورة» بلا «كيلو» لا تُجهَّز. */}
                    {o.items.length > 0 && <InvoiceLines items={o.items} className="border-b border-line-soft pb-rg-sm" />}
                    <MoneyLines o={o} />
                  </Link>
                  {reportable.has(o.id) && (
                    <Button variant="ghost" className="flex min-h-11 w-fit items-center gap-rg-xs text-danger" onClick={() => setReporting(o)}>
                      <IconWarning size={14} />
                      {H.report}
                    </Button>
                  )}
                </div>
              </li>
            );
          })}
        </ul>
      )}
      {rows != null && rows.length < total && (
        <Button variant="secondary" className="min-h-11 self-center" disabled={busy} onClick={() => void fetchPage(page + 1, false)} data-testid="history-more">
          {H.more}
        </Button>
      )}
      {rows != null && rows.length > 0 && (
        <p className="console-meta text-center">{fill(K.shownOf, { a: fmtNum(rows.length), b: fmtNum(total) })}</p>
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
