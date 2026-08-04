"use client";

/**
 * **كشفُ صندوقي — مالٌ في ذمّتي أقرؤه.**
 *
 * كان السائقُ يرى المجموعَ وحدَه في رأس لوحته: `123,000 / 500,000`. **ولا يرى
 * تفصيلَه**: من أيّ طلبٍ قبض، ومتى سلّم، وكم بقي.
 *
 * والنقطةُ `GET /driver/cash` **مبنيّةٌ منذ البداية ولا يناديها أحد.**
 *
 * **ومالٌ في ذمّة إنسانٍ بلا كشفٍ يقرؤه خلافٌ ينتظر**: يقول «سلّمتُ» وتقول
 * المنصةُ «لم يصل»، **ولا ورقةَ بينهما.** ومن رأى حركتَه مسطورةً بالتاريخ
 * والرقم اطمأنّ — ومن لم يرَ شكّ.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  LoadingState,
  EmptyState,
  useLiveRefresh,
  IconBalance,
  IconOrder,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.driver.cashbox;

interface Entry {
  kind: string;
  amount: number;
  note: string;
  order_number: number | null;
  created_at: string;
}

interface Me {
  cash_held: number;
  cash_limit: number;
}

export default function DriverCashPage() {
  const [me, setMe] = useState<Me | null>(null);
  const [rows, setRows] = useState<Entry[] | null>(null);

  const load = useCallback(() => {
    api<Me>("/api/v1/driver/me").then(setMe).catch(() => undefined);
    api<Entry[] | { entries: Entry[] }>("/api/v1/driver/cash")
      .then((r) => setRows(Array.isArray(r) ? r : (r.entries ?? [])))
      .catch(() => setRows([]));
  }, []);

  useEffect(load, [load]);
  // **وتسويةُ المالية تصل بلا تحديثِ صفحة** — من سلّم صندوقَه يريد أن يراه صفراً.
  useLiveRefresh(["wallet", "order"], load);

  if (!rows || !me) return <LoadingState />;

  const ratio = me.cash_limit > 0 ? me.cash_held / me.cash_limit : 0;

  return (
    <PageContainer>
      <PageHeader icon={IconBalance} title={C.title} />

      {/* **ما في ذمّتي الآن — وكم بقي من سقفي.**

          والسقفُ ليس زينة: **من بلغه لا يُعرض عليه طلبٌ نقديٌّ جديد**، فيقف
          عمله ولا يعرف لماذا. */}
      <div className="mb-4 rounded-card border border-line bg-surface p-4">
        <div className="mb-2 flex items-end justify-between">
          <span className="text-sm text-ink-muted">{C.held}</span>
          <span dir="ltr" className="text-2xl font-bold tabular-nums text-warning">
            {fmtNum(me.cash_held)}{" "}
            <span className="text-xs font-normal text-ink-muted">{m.common.currency}</span>
          </span>
        </div>
        <div className="h-2 overflow-hidden rounded-badge bg-page">
          <div
            className={`h-full rounded-badge transition-[width] ${
              ratio > 0.8 ? "bg-danger" : ratio > 0.5 ? "bg-warning" : "bg-success"
            }`}
            style={{ width: `${Math.min(100, ratio * 100)}%` }}
          />
        </div>
        <p dir="ltr" className="mt-1 text-end text-xs text-ink-muted">
          {fmtNum(me.cash_held)} / {fmtNum(me.cash_limit)}
        </p>
      </div>

      {rows.length === 0 ? (
        <EmptyState icon={IconBalance} title={C.empty} />
      ) : (
        <ul className="divide-y divide-line rounded-card border border-line bg-surface">
          {rows.map((e, i) => (
            <li key={i} className="flex items-center gap-3 px-3 py-2.5 text-sm">
              <span className="min-w-0 flex-1">
                {e.order_number != null ? (
                  <span dir="ltr" className="flex items-center gap-1 font-bold tabular-nums">
                    <IconOrder size={14} />#{fmtRef(e.order_number)}
                  </span>
                ) : (
                  <span className="font-medium">
                    {(C.kinds as Record<string, string>)[e.kind] ?? e.kind}
                  </span>
                )}
                {e.note && <span className="block text-xs text-ink-muted">{e.note}</span>}
                <span dir="ltr" className="block text-xs text-ink-muted">
                  {fmtDateTime(e.created_at)}
                </span>
              </span>
              {/* **الموجبُ قبضٌ والسالبُ تسليم** — واللونُ يقولها قبل الرقم. */}
              <span
                dir="ltr"
                className={`shrink-0 text-base font-bold tabular-nums ${
                  e.amount < 0 ? "text-success" : "text-warning"
                }`}
              >
                {e.amount > 0 ? "+" : ""}
                {fmtNum(e.amount)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </PageContainer>
  );
}
