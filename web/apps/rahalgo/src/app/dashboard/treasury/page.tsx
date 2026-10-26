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
import { useRouter, useSearchParams } from "next/navigation";
import {
  PageContainer,
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
import { TopupTab } from "@/components/admin/treasury/Topup";
import { useCanCall } from "@/lib/policy";
import { PageHero, type HeroTab } from "@/components/admin/PageHero";

type Tab = "overview" | "statement" | "cashbox" | "approvals" | "topup" | "health";
const TABS: Tab[] = ["overview", "statement", "cashbox", "approvals", "topup", "health"];

/** **صفحاتُ المال التي لها بابُها** — تُفتح من هنا. */
const LINKS: { href: string; label: string }[] = [
  { href: "/dashboard/cash", label: T.links.cash },
  { href: "/dashboard/payouts", label: T.links.payouts },
  { href: "/dashboard/obligations", label: T.links.obligations },
  { href: "/dashboard/profits", label: T.links.profits },
  { href: "/dashboard/expenses", label: T.links.expenses },
  { href: "/dashboard/losses", label: T.links.losses },
  { href: "/dashboard/compensations", label: T.links.compensations },
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
  // **«شحن محفظة» لمن يقترح الحركة** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — بقدرة بابه.
  const canTopup = useCanCall()("GET", "/treasury/wallet-lookup");
  const fromUrl = params.get("tab") as Tab | null;
  const [tab, setTab] = useState<Tab>(fromUrl && TABS.includes(fromUrl) ? fromUrl : "overview");

  useEffect(() => {
    if (fromUrl && TABS.includes(fromUrl) && fromUrl !== tab) setTab(fromUrl);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fromUrl]);

  // **سطرٌ واحد** (طلبُ المالك ٢٠٢٦-١٠-٢٣): تبويباتُ الخزينة وروابطُ صفحات المال
  // في صفٍّ واحد — والرابطُ مفتاحُه مسارُه، فيُفتح بابُه بدل أن يُبدَّل التبويب.
  function go(k: Tab | string) {
    if (k.startsWith("/")) {
      router.push(k);
      return;
    }
    if (!TABS.includes(k as Tab)) return;
    setTab(k as Tab);
    router.replace(`/dashboard/treasury?tab=${k}`);
  }

  const tabs: HeroTab<string>[] = [
    { key: "overview", label: T.tabs.overview, icon: IconWallet },
    { key: "statement", label: T.tabs.statement, icon: IconStatus },
    { key: "cashbox", label: T.tabs.cashbox, icon: IconBalance },
    { key: "approvals", label: T.tabs.approvals, icon: IconCheck },
    ...(canTopup ? [{ key: "topup", label: T.topup.tab, icon: IconWallet }] : []),
    { key: "health", label: T.tabs.health, icon: IconShieldCheck },
  ];

  return (
    <PageContainer>
      {/* **رأسٌ موحّد** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «الخزينة بالتصميم الجديد») — تبويباتُ الخزينة بسطرٍ واحد،
          **وصفحاتُ المال التي لها بابُها روابطُ صغيرةٌ فوقها** (كانت مخلوطةً بالتبويبات، ٢٠٢٦-١٠-٢٣). */}
      <PageHero<string> icon={IconWallet} title={T.title} subtitle={T.subtitle} tabs={tabs} value={tab} onChange={go}>
        <div className="flex flex-wrap gap-1.5">
          {LINKS.map((l) => (
            <button
              key={l.href}
              type="button"
              onClick={() => go(l.href)}
              className="rounded-full border border-line-soft px-3 py-1 text-xs font-bold text-ink-muted transition-colors hover:bg-row-hover hover:text-ink"
            >
              {l.label} ↗
            </button>
          ))}
        </div>
      </PageHero>
      {tab === "overview" && <OverviewTab onTab={go} />}
      {tab === "statement" && <StatementTab />}
      {tab === "cashbox" && <CashboxTab />}
      {tab === "approvals" && <ApprovalsTab />}
      {tab === "topup" && canTopup && <TopupTab />}
      {tab === "health" && <HealthTab />}
    </PageContainer>
  );
}
