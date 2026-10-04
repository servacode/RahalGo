"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قسمُ الخزينة الموحّد** — قرارُ المالك ٢٠٢٦-١٠-٠٤
 * ══════════════════════════════════════════════════════════════════════
 *
 * «نظرةٌ عامّة · كشفُ حسابٍ برصيدٍ جارٍ وتصدير · أرباحٌ وخسائر · النقدُ والصندوق ·
 * الموافقات · السحوباتُ والشحن · الديون · صحّةُ الدفتر».
 *
 * **والخزينةُ تبقى محفظةَ الأدمن.** وكلُّ قسمٍ ماليٍّ يبقى في صفحته (المصروفات،
 * الأرباح، النقد، السحوبات، الديون…) **وهذه تربطها**: تبويباتٌ لما يخصّ الخزينةَ
 * نفسَها، وروابطُ لما له صفحتُه.
 */

import { Suspense, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import {
  PageContainer,
  PageHeader,
  Tabs,
  type TabDef,
  IconWallet,
  IconStatus,
  IconBalance,
  IconCheck,
  IconShieldCheck,
} from "@rahalgo/ui";
import { T } from "@/components/admin/treasury/shared";
import { OverviewTab } from "@/components/admin/treasury/Overview";
import { StatementTab } from "@/components/admin/treasury/Statement";
import { CashboxTab } from "@/components/admin/treasury/Cashbox";
import { ApprovalsTab } from "@/components/admin/treasury/Approvals";
import { HealthTab } from "@/components/admin/treasury/Health";

type Tab = "overview" | "statement" | "cashbox" | "approvals" | "health";
const TABS: Tab[] = ["overview", "statement", "cashbox", "approvals", "health"];

/** **صفحاتُ المال التي لها بابُها** — تُفتح من هنا. */
const LINKS: { href: string; label: string }[] = [
  { href: "/dashboard/cash", label: T.links.cash },
  { href: "/dashboard/payouts", label: T.links.payouts },
  { href: "/dashboard/obligations", label: T.links.obligations },
  { href: "/dashboard/profits", label: T.links.profits },
  { href: "/dashboard/expenses", label: T.links.expenses },
  { href: "/dashboard/losses", label: T.links.losses },
  { href: "/dashboard/compensations", label: T.links.compensations },
  { href: "/dashboard/wallet", label: T.links.myWallet },
];

export default function TreasuryPage() {
  return (
    <Suspense>
      <TreasuryHub />
    </Suspense>
  );
}

function TreasuryHub() {
  const router = useRouter();
  const params = useSearchParams();
  const fromUrl = params.get("tab") as Tab | null;
  const [tab, setTab] = useState<Tab>(fromUrl && TABS.includes(fromUrl) ? fromUrl : "overview");

  useEffect(() => {
    if (fromUrl && TABS.includes(fromUrl) && fromUrl !== tab) setTab(fromUrl);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fromUrl]);

  function go(k: Tab) {
    setTab(k);
    router.replace(`/dashboard/treasury?tab=${k}`);
  }

  const items: TabDef<Tab>[] = [
    { key: "overview", label: T.tabs.overview, icon: IconWallet },
    { key: "statement", label: T.tabs.statement, icon: IconStatus },
    { key: "cashbox", label: T.tabs.cashbox, icon: IconBalance },
    { key: "approvals", label: T.tabs.approvals, icon: IconCheck },
    { key: "health", label: T.tabs.health, icon: IconShieldCheck },
  ];

  return (
    <PageContainer>
      <PageHeader icon={IconWallet} title={T.title} subtitle={T.subtitle} />
      <nav aria-label={T.links.title} className="mb-3 flex flex-wrap gap-x-4 gap-y-1 text-sm">
        {LINKS.map((l) => (
          <Link key={l.href} href={l.href} className="text-primary underline">
            {l.label}
          </Link>
        ))}
      </nav>
      <Tabs items={items} value={tab} onChange={go} className="mb-4" />
      {tab === "overview" && <OverviewTab onTab={go} />}
      {tab === "statement" && <StatementTab />}
      {tab === "cashbox" && <CashboxTab />}
      {tab === "approvals" && <ApprovalsTab />}
      {tab === "health" && <HealthTab />}
    </PageContainer>
  );
}
