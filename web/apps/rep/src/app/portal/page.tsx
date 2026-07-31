"use client";

/** نظرة عامة — كود المندوب وإحصاءاته السريعة. */

import { useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Button,
  IconStore,
  IconWallet,
  IconOrder,
  IconSuccess,
  IconPromos,
  IconLink,
  PageContainer,
  StatGrid,
  StatCard,
  LoadingState,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);

interface Me {
  invite_code: string | null;
  full_name: string;
  merchants: number;
  delivered_orders: number;
  total_commissions: number;
  balance: number;
}

export default function OverviewPage() {
  const [copied, setCopied] = useState(false);
  const { data: me } = useLiveData<Me>(() => api("/api/v1/rep/me"), [
    "lead",
    "order",
    "wallet",
  ]);

  if (!me) {
    return <LoadingState />;
  }

  const code = me.invite_code ?? "—";
  const shareText = encodeURIComponent(m.rep.shareText.replace("{code}", code));


  return (
    <PageContainer>
      {/* الكود — قلب اللوحة */}
      <section className="rounded-card bg-primary p-6 text-center text-white">
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
            className="rounded-control bg-white px-4 py-2 text-sm font-medium text-primary-dark"
          >
            {m.rep.share}
          </a>
          <Link
            href="/portal/link"
            className="flex items-center gap-1.5 rounded-control border border-white/40 px-4 py-2 text-sm font-medium text-white hover:bg-white/10"
          >
            <IconLink size={15} />
            {m.rep.nav.link}
          </Link>
        </div>
      </section>

      <StatGrid>
        <StatCard icon={IconStore} label={m.rep.stats.merchants} value={me.merchants} />
        <StatCard icon={IconSuccess} label={m.rep.stats.delivered} value={me.delivered_orders} tone="success" />
        <StatCard icon={IconOrder} label={`${m.rep.stats.commissions} (${m.common.currency})`} value={me.total_commissions} />
        <StatCard icon={IconWallet} label={`${m.terms.walletBalance} (${m.common.currency})`} value={me.balance} tone="accent" />
      </StatGrid>
    </PageContainer>
  );
}
