"use client";

/** نظرة عامة — كود المندوب وإحصاءاته السريعة. */

import { useEffect, useState } from "react";
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
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

interface Me {
  invite_code: string | null;
  full_name: string;
  merchants: number;
  delivered_orders: number;
  total_commissions: number;
  balance: number;
}

export default function OverviewPage() {
  const [me, setMe] = useState<Me | null>(null);
  const [copied, setCopied] = useState(false);

  useEffect(() => {
    api<Me>("/api/v1/rep/me").then(setMe).catch(() => undefined);
  }, []);

  if (!me) {
    return <p className="py-12 text-center text-ink-muted">{m.common.loading}</p>;
  }

  const code = me.invite_code ?? "—";
  const shareText = encodeURIComponent(m.rep.shareText.replace("{code}", code));

  const stats = [
    { label: m.rep.stats.merchants, value: fmt.format(me.merchants), icon: <IconStore /> },
    {
      label: m.rep.stats.delivered,
      value: fmt.format(me.delivered_orders),
      icon: <IconSuccess className="text-success" />,
    },
    {
      label: `${m.rep.stats.commissions} (${m.common.currency})`,
      value: fmt.format(me.total_commissions),
      icon: <IconOrder />,
    },
    {
      label: `${m.rep.stats.balance} (${m.common.currency})`,
      value: fmt.format(me.balance),
      icon: <IconWallet className="text-primary" />,
    },
  ];

  return (
    <div className="space-y-6">
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

      <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
        {stats.map((s) => (
          <div key={s.label} className="rounded-card border border-line bg-surface p-4">
            <div className="mb-1 text-ink-muted">{s.icon}</div>
            <p className="text-xl font-bold">{s.value}</p>
            <p className="text-xs text-ink-muted">{s.label}</p>
          </div>
        ))}
      </div>
    </div>
  );
}
