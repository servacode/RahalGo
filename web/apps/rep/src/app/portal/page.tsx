"use client";

/** نظرة عامة — كود المندوب، هدف الشهر، وإحصاءاته. */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Button,
  Card,
  IconStore,
  IconWallet,
  IconOrder,
  IconSuccess,
  IconPromos,
  IconLink,
  IconLock,
  IconTrendUp,
  PageContainer,
  StatGrid,
  StatCard,
  LoadingState,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const G = m.rep.target;

interface Me {
  invite_code: string | null;
  full_name: string;
  merchants: number;
  delivered_orders: number;
  total_commissions: number;
  balance: number;
  month_merchants: number;
  month_delivered: number;
  month_commissions: number;
  monthly_target: number;
  pending_leads: number;
  whatsapp_verified: boolean;
}

/** كم يوماً بقي في الشهر — الهدف بلا مهلة ظاهرة لا يحرّك أحداً. */
function daysLeftInMonth(): number {
  const now = new Date();
  const end = new Date(now.getFullYear(), now.getMonth() + 1, 0);
  return Math.max(0, end.getDate() - now.getDate());
}

export default function OverviewPage() {
  const [copied, setCopied] = useState(false);
  const { data: me } = useLiveData<Me>(() => api("/api/v1/rep/me"), ["lead", "order", "wallet"]);

  if (!me) return <LoadingState />;

  const code = me.invite_code ?? "—";
  const shareText = encodeURIComponent(m.rep.shareText.replace("{code}", code));

  const target = Math.max(1, me.monthly_target);
  const done = me.month_merchants;
  const pct = Math.min(100, Math.round((done / target) * 100));
  const reached = done >= target;
  const days = daysLeftInMonth();

  return (
    <PageContainer>
      {/* الكود — قلب اللوحة، ومقفل حتى يوثّق المندوب قناة تواصله */}
      <section className="rounded-card bg-primary p-6 text-center text-on-bright">
        {!me.whatsapp_verified ? (
          <>
            <span className="mx-auto mb-3 flex h-12 w-12 items-center justify-center rounded-badge bg-on-solid-tint">
              <IconLock size={22} />
            </span>
            <p className="text-lg font-bold">{m.rep.lockedTitle}</p>
            <p className="mx-auto mt-2 max-w-md text-xs leading-relaxed opacity-80">{m.rep.lockedHint}</p>
            <Link
              href="/portal/account"
              className="mt-4 inline-block rounded-control bg-on-solid px-5 py-2.5 text-sm font-medium text-primary-dark"
            >
              {m.rep.lockedCta}
            </Link>
          </>
        ) : (
          <>
            <p className="mb-2 flex items-center justify-center gap-2 text-sm opacity-80">
              <IconPromos size={16} />
              {m.rep.codeTitle}
            </p>
            <p className="font-mono text-4xl font-bold tracking-widest" dir="ltr">
              {code}
            </p>
            <p className="mx-auto mt-3 max-w-md text-xs leading-relaxed opacity-80">{m.rep.codeHint}</p>
            <div className="mt-4 flex flex-wrap justify-center gap-2">
              <Button
                variant="secondary"
                onClick={() => {
                  void navigator.clipboard.writeText(code);
                  setCopied(true);
                  setTimeout(() => setCopied(false), 1500);
                }}
              >
                {copied ? m.rep.copied : m.rep.copy}
              </Button>
              <a
                href={`https://wa.me/?text=${shareText}`}
                target="_blank"
                rel="noreferrer"
                className="rounded-control bg-on-solid px-4 py-2 text-sm font-medium text-primary-dark"
              >
                {m.rep.share}
              </a>
              <Link
                href="/portal/link"
                className="flex items-center gap-1.5 rounded-control border border-on-solid-edge px-4 py-2 text-sm font-medium text-on-solid hover:bg-on-solid-tint"
              >
                <IconLink size={15} />
                {m.rep.nav.link}
              </Link>
            </div>
          </>
        )}
      </section>

      {/* هدف الشهر — الرقم الذي يقيس عمل المندوب فعلاً (PLAN §7) */}
      <Card title={G.title} icon={IconTrendUp}>
        <div className="flex flex-wrap items-end justify-between gap-2">
          <p className="text-2xl font-bold">
            <span className={reached ? "text-success" : "text-primary-dark"}>{fmtNum(done)}</span>
            <span className="text-base font-normal text-ink-muted">
              {" / "}
              {fmtNum(target)}
            </span>
          </p>
          <p className="text-sm text-ink-muted">
            {days === 0 ? G.lastDay : G.daysLeft.replace("{n}", fmtNum(days))}
          </p>
        </div>

        <div className="mt-2 h-2 overflow-hidden rounded-badge bg-page">
          <div
            className={`h-full rounded-badge transition-all ${reached ? "bg-success" : "bg-primary"}`}
            style={{ width: `${pct}%` }}
          />
        </div>

        <p className={`mt-2 text-sm ${reached ? "font-medium text-success" : "text-ink-muted"}`}>
          {reached ? G.done : G.remaining.replace("{n}", fmtNum(target - done))}
        </p>

        <div className="mt-4 grid gap-3 border-t border-line pt-4 sm:grid-cols-3">
          <div>
            <p className="text-lg font-bold">{fmtNum(me.month_delivered)}</p>
            <p className="text-xs text-ink-muted">{G.monthDelivered}</p>
          </div>
          <div>
            <p className="text-lg font-bold" dir="ltr">
              {fmtNum(me.month_commissions)}
            </p>
            <p className="text-xs text-ink-muted">
              {G.monthCommissions} ({m.common.currency})
            </p>
          </div>
          <div>
            <p className={`text-lg font-bold ${me.pending_leads ? "text-warning" : ""}`}>
              {fmtNum(me.pending_leads)}
            </p>
            <p className="text-xs text-ink-muted">{G.pendingLeads}</p>
          </div>
        </div>
      </Card>

      {/* الإجمالي التراكمي */}
      <StatGrid>
        <StatCard icon={IconStore} label={m.rep.stats.merchants} value={me.merchants} />
        <StatCard
          icon={IconSuccess}
          label={m.rep.stats.delivered}
          value={me.delivered_orders}
          tone="success"
        />
        <StatCard
          icon={IconOrder}
          label={`${m.rep.stats.commissions} (${m.common.currency})`}
          value={me.total_commissions}
        />
        <StatCard
          icon={IconWallet}
          label={`${m.terms.walletBalance} (${m.common.currency})`}
          value={me.balance}
          tone="accent"
        />
      </StatGrid>
    </PageContainer>
  );
}
