"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **زوّارُ الموقع والتحميلات** (طلبُ المالك ٢٠٢٦-١٠-٠٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * يقرأ `GET /api/v1/admin/site-stats` — **والمحرّكُ يعدّ كلَّ شخصٍ مرّةً في
 * اليوم** (`site_stats.go`). **ولا يظهر لمن لا يملك الباب** (`useCanCall`).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { StatCard, StatGrid, Card, Alert, IconUsers, IconUser, IconApp } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";

const S = getMessages(defaultLocale).admin.home.site;

interface Apps {
  customer: number;
  driver: number;
  merchant: number;
  rep: number;
}

interface SiteStats {
  today: { visits: number; downloads: Apps; new_accounts: number };
  totals: { visits_unique_ever: number; visits_days_sum: number; downloads: Apps; accounts: number };
  series: { day: string; visits: number; downloads: number; new_accounts: number }[];
}

const APPS: { key: keyof Apps; label: string }[] = [
  { key: "customer", label: S.appCustomer },
  { key: "driver", label: S.appDriver },
  { key: "merchant", label: S.appMerchant },
  { key: "rep", label: S.appRep },
];

const sum = (a: Apps) => a.customer + a.driver + a.merchant + a.rep;

function AppsBreakdown({ title, apps }: { title: string; apps: Apps }) {
  return (
    <Card>
      <p className="mb-2 flex items-baseline gap-2 text-sm font-bold">
        <span className="flex-1">{title}</span>
        <span>{fmtNum(sum(apps))}</span>
      </p>
      <div className="space-y-1 text-sm">
        {APPS.map((x) => (
          <div key={x.key} className="flex items-baseline gap-2">
            <span className="min-w-0 flex-1 text-ink-muted">{x.label}</span>
            <span className={apps[x.key] ? "font-bold" : "text-ink-muted"}>{fmtNum(apps[x.key])}</span>
          </div>
        ))}
      </div>
    </Card>
  );
}

function DailyBars({ series }: { series: SiteStats["series"] }) {
  const [hover, setHover] = useState<number | null>(null);
  const max = Math.max(1, ...series.map((d) => Math.max(d.visits, d.downloads)));
  const h = (v: number) => Math.max(v > 0 ? 6 : 2, (v / max) * 100);
  return (
    <section className="surface p-5">
      <div className="mb-4 flex flex-wrap items-center gap-3 text-sm">
        <h3 className="flex-1 font-bold">{S.chartTitle}</h3>
        <span className="flex items-center gap-1 text-xs text-ink-muted">
          <span aria-hidden className="h-2.5 w-2.5 rounded-badge bg-primary" />
          {S.chartVisits}
        </span>
        <span className="flex items-center gap-1 text-xs text-ink-muted">
          <span aria-hidden className="h-2.5 w-2.5 rounded-badge bg-accent" />
          {S.chartDownloads}
        </span>
      </div>
      <div className="flex h-40 items-end gap-0.5" dir="ltr">
        {series.map((d, i) => (
          <div
            key={d.day}
            className="relative flex h-full flex-1 items-end justify-center gap-px"
            onMouseEnter={() => setHover(i)}
            onMouseLeave={() => setHover(null)}
          >
            <div
              className={`w-1/2 max-w-3 rounded-t ${d.visits > 0 ? "bg-primary" : "bg-line"}`}
              style={{ height: `${h(d.visits)}%` }}
            />
            <div
              className={`w-1/2 max-w-3 rounded-t ${d.downloads > 0 ? "bg-accent" : "bg-line"}`}
              style={{ height: `${h(d.downloads)}%` }}
            />
            {hover === i && (
              <div className="pointer-events-none absolute bottom-full z-10 mb-1 whitespace-nowrap surface-inset px-2.5 py-1.5 text-xs elev-2" dir="rtl">
                <span className="text-ink-muted">{d.day.slice(5)} · </span>
                <span className="font-bold">
                  {S.chartVisits} {fmtNum(d.visits)} · {S.chartDownloads} {fmtNum(d.downloads)}
                </span>
              </div>
            )}
          </div>
        ))}
      </div>
      <div className="mt-1 flex gap-0.5 border-t border-line-soft pt-1" dir="ltr">
        {series.map((d) => (
          <div key={d.day} className="flex-1 text-center text-2xs text-ink-muted">
            {d.day.slice(8)}
          </div>
        ))}
      </div>
    </section>
  );
}

export default function SiteStatsSection() {
  const allowed = useCanCall()("GET", "/site-stats");
  const [data, setData] = useState<SiteStats | null>(null);
  const [failed, setFailed] = useState(false);

  const load = useCallback(() => {
    api<SiteStats>("/api/v1/admin/site-stats?days=30")
      .then((d) => {
        setData(d);
        setFailed(false);
      })
      .catch(() => setFailed(true));
  }, []);

  useEffect(() => {
    if (allowed) load();
  }, [allowed, load]);

  if (!allowed) return null;

  return (
    <section>
      <h2 className="mb-2 flex flex-wrap items-baseline gap-2 font-bold">
        {S.title}
        <span className="text-xs font-normal text-ink-muted">{S.note}</span>
      </h2>
      {failed && !data && <Alert tone="warning">{S.failed}</Alert>}
      {data && (
        <div className="space-y-3">
          <StatGrid>
            <StatCard icon={IconUsers} label={S.visitsToday} value={fmtNum(data.today.visits)} tone={data.today.visits ? "default" : "muted"} />
            <StatCard
              icon={IconUsers}
              label={S.visitsAll}
              value={fmtNum(data.totals.visits_unique_ever)}
              sub={S.visitsAllSub.replace("{n}", fmtNum(data.totals.visits_days_sum))}
            />
            <StatCard
              icon={IconApp}
              label={S.downloadsToday}
              value={fmtNum(sum(data.today.downloads))}
              tone={sum(data.today.downloads) ? "default" : "muted"}
            />
            <StatCard
              icon={IconUser}
              label={S.accountsToday}
              value={fmtNum(data.today.new_accounts)}
              sub={S.accountsAll.replace("{n}", fmtNum(data.totals.accounts))}
              tone={data.today.new_accounts ? "default" : "muted"}
            />
          </StatGrid>
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <AppsBreakdown title={S.downloadsToday} apps={data.today.downloads} />
            <AppsBreakdown title={S.downloadsAll} apps={data.totals.downloads} />
          </div>
          <DailyBars series={data.series} />
        </div>
      )}
    </section>
  );
}
