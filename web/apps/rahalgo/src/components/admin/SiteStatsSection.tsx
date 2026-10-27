"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **زوّارُ الموقع والتحميلات** (طلبُ المالك ٢٠٢٦-١٠-٠٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * يقرأ `GET /api/v1/admin/site-stats` — **والمحرّكُ يعدّ كلَّ شخصٍ مرّةً في
 * اليوم** (`site_stats.go`). **ولا يظهر لمن لا يملك الباب** (`useCanCall`).
 *
 * **واليومُ يومُ المنصّة** — يبدأ بساعة فتح الدوام كبقيّة الرئيسيّة (تدقيقُ ٢٠٢٦-١٠-٢٧)،
 * **وبالشكل نفسِه**: لوحةٌ وبطاقاتٌ (`DashParts`)، والرسمُ يُضغط على الجوّال لا يُوقَف عليه فقط.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconUsers, IconUser, IconApp, IconWarning } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { Panel, Tile } from "@/components/admin/DashParts";

const S = getMessages(defaultLocale).admin.home.site;

interface Apps {
  customer: number;
  driver: number;
  merchant: number;
  rep: number;
}

interface SiteStats {
  day?: string;
  day_start_hour?: number;
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

/** **تحميلاتٌ بالتطبيق** — بطاقةٌ واحدة: المجموعُ كبيرٌ وتحته سطرٌ لكلّ تطبيق. */
function AppsBreakdown({ title, apps }: { title: string; apps: Apps }) {
  return (
    <div className="rounded-card border border-line-soft p-3">
      <div className="mb-2 flex items-baseline gap-2">
        <span className="min-w-0 flex-1 text-sm text-ink-muted">{title}</span>
        <span className="figure leading-none">{fmtNum(sum(apps))}</span>
      </div>
      <div className="space-y-1 text-sm">
        {APPS.map((x) => (
          <div key={x.key} className="flex items-baseline gap-2">
            <span className="min-w-0 flex-1 text-ink-muted">{x.label}</span>
            <span className={apps[x.key] ? "font-bold" : "text-ink-muted"}>{fmtNum(apps[x.key])}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

/** **أعمدةُ آخر ٣٠ يوماً** — الوقوفُ أو الضغطُ على يومٍ يكتب أرقامَه تحت الرسم (يعمل على الجوّال). */
function DailyBars({ series }: { series: SiteStats["series"] }) {
  const [picked, setPicked] = useState<number | null>(null);
  const max = Math.max(1, ...series.map((d) => Math.max(d.visits, d.downloads)));
  const h = (v: number) => Math.max(v > 0 ? 6 : 2, (v / max) * 100);
  const sel = picked !== null ? series[picked] : series[series.length - 1];
  return (
    <div className="rounded-card border border-line-soft p-3">
      <div className="mb-3 flex flex-wrap items-center gap-3 text-sm">
        <span className="min-w-0 flex-1 font-bold">{S.chartTitle}</span>
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
          <button
            key={d.day}
            type="button"
            aria-label={d.day}
            aria-pressed={picked === i}
            className={`relative flex h-full flex-1 items-end justify-center gap-px rounded-t ${
              picked === i ? "bg-row-hover" : ""
            }`}
            onMouseEnter={() => setPicked(i)}
            onFocus={() => setPicked(i)}
            onClick={() => setPicked(i)}
          >
            <span
              className={`block w-1/2 max-w-3 rounded-t ${d.visits > 0 ? "bg-primary" : "bg-line"}`}
              style={{ height: `${h(d.visits)}%` }}
            />
            <span
              className={`block w-1/2 max-w-3 rounded-t ${d.downloads > 0 ? "bg-accent" : "bg-line"}`}
              style={{ height: `${h(d.downloads)}%` }}
            />
          </button>
        ))}
      </div>
      <div className="mt-1 flex gap-0.5 border-t border-line-soft pt-1" dir="ltr">
        {series.map((d) => (
          <div key={d.day} className="flex-1 text-center text-2xs text-ink-muted">
            {d.day.slice(8)}
          </div>
        ))}
      </div>
      {sel && (
        <p className="mt-2 rounded-control bg-field px-2.5 py-1.5 text-xs" aria-live="polite">
          <span className="font-bold">
            {S.dayDetail
              .replace("{day}", sel.day.slice(5))
              .replace("{v}", fmtNum(sel.visits))
              .replace("{d}", fmtNum(sel.downloads))
              .replace("{a}", fmtNum(sel.new_accounts))}
          </span>
        </p>
      )}
      <p className="mt-1 text-2xs text-ink-muted">{S.tapHint}</p>
    </div>
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
      // @empty-ok — **القسمُ يقول إنّه لم يصل** بدائرةٍ وجملة، والرئيسيّةُ تبقى.
      .catch(() => setFailed(true));
  }, []);

  useEffect(() => {
    if (allowed) load();
  }, [allowed, load]);

  if (!allowed) return null;

  const note = S.note.replace("{h}", fmtNum(data?.day_start_hour ?? 0));

  return (
    <Panel title={S.title} note={data ? note : undefined}>
      {failed && !data ? (
        <div className="flex flex-col items-center gap-2 py-6 text-center">
          <span className="inline-flex h-12 w-12 items-center justify-center rounded-full bg-warning-solid text-on-solid">
            <IconWarning size={22} />
          </span>
          <p className="text-sm font-bold">{S.failed}</p>
          <p className="max-w-md text-xs text-ink-muted">{S.failedHint}</p>
        </div>
      ) : !data ? (
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
          {[0, 1, 2, 3].map((i) => (
            <span key={i} className="block h-16 animate-pulse rounded-card bg-field" />
          ))}
        </div>
      ) : (
        <div className="space-y-3">
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
            <Tile icon={IconUsers} label={S.visitsToday} value={fmtNum(data.today.visits)} tone={data.today.visits ? "accent" : "muted"} />
            <Tile
              icon={IconUsers}
              label={S.visitsAll}
              value={fmtNum(data.totals.visits_unique_ever)}
              sub={S.visitsAllSub.replace("{n}", fmtNum(data.totals.visits_days_sum))}
            />
            <Tile
              icon={IconApp}
              label={S.downloadsToday}
              value={fmtNum(sum(data.today.downloads))}
              tone={sum(data.today.downloads) ? "accent" : "muted"}
            />
            <Tile
              icon={IconUser}
              label={S.accountsToday}
              value={fmtNum(data.today.new_accounts)}
              sub={S.accountsAll.replace("{n}", fmtNum(data.totals.accounts))}
              tone={data.today.new_accounts ? "success" : "muted"}
            />
          </div>
          <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
            <AppsBreakdown title={S.downloadsToday} apps={data.today.downloads} />
            <AppsBreakdown title={S.downloadsAll} apps={data.totals.downloads} />
          </div>
          <DailyBars series={data.series} />
        </div>
      )}
    </Panel>
  );
}
