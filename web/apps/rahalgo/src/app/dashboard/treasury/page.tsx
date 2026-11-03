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
  AdminPage,
  type AdminTab,
  IconWallet,
} from "@rahalgo/ui";
import { T } from "@/components/admin/treasury/shared";
import { OverviewTab } from "@/components/admin/treasury/Overview";
import { StatementTab } from "@/components/admin/treasury/Statement";
import { CashboxTab } from "@/components/admin/treasury/Cashbox";
import { ApprovalsTab } from "@/components/admin/treasury/Approvals";
import { HealthTab } from "@/components/admin/treasury/Health";
import { TopupTab } from "@/components/admin/treasury/Topup";
import { useCanCall } from "@/lib/policy";

type Tab = "overview" | "statement" | "cashbox" | "approvals" | "topup" | "health";
const TABS: Tab[] = ["overview", "statement", "cashbox", "approvals", "topup", "health"];

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

  // **تبويباتُ الخزينة بسطرٍ واحد** — وصفحاتُ المال الأخرى في القائمة الجانبيّة.
  function go(k: Tab | string) {
    if (!TABS.includes(k as Tab)) return;
    setTab(k as Tab);
    router.replace(`/dashboard/treasury?tab=${k}`);
  }

  const tabs: AdminTab<string>[] = [
    { key: "overview", label: T.tabs.overview },
    { key: "statement", label: T.tabs.statement },
    { key: "cashbox", label: T.tabs.cashbox },
    { key: "approvals", label: T.tabs.approvals },
    ...(canTopup ? [{ key: "topup", label: T.topup.tab }] : []),
    { key: "health", label: T.tabs.health },
  ];

  return (
    <AdminPage<string> icon={IconWallet} title={T.title} tabs={tabs} tab={tab} onTab={go}>
      {tab === "overview" && <OverviewTab onTab={go} />}
      {tab === "statement" && <StatementTab />}
      {tab === "cashbox" && <CashboxTab />}
      {tab === "approvals" && <ApprovalsTab />}
      {tab === "topup" && canTopup && <TopupTab />}
      {tab === "health" && <HealthTab />}
    </AdminPage>
  );
}
