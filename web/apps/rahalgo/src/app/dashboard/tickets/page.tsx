"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«المشاكل والتعويضات» — قسمٌ واحد** (قرارُ المالك ٢٠٢٦-١٠-٢٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «الشكاوى والتقييمات، والخسائر والنزاعات وتعويض البضاعة — كلّهن بقسمٍ واحد». **خمسةُ أوجهٍ
 * لسؤالٍ واحد: شو صار غلط وكيف انحلّ** — الشكوى تبدأه، والتقييمُ الضعيفُ يكشفه، والخسارةُ
 * والنزاعُ والتعويضُ آخرُه بالمال.
 *
 * **وكلُّ تبويبٍ بقدرة بابه** كما كان في قسمَيه: الدعمُ يرى الشكاوى والتقييمات، والماليّةُ الخسائرَ
 * والنزاعاتِ والتعويض، والمالكُ الكلّ. **والتبويبُ في الرابط** (`?tab=`) — ومنه يفتح
 * `/dashboard/losses` القديمُ تبويبَه.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { PageContainer, PageHeader, TabCards, IconSupport } from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";
import { TicketsView } from "@/components/admin/support/tickets";
import { RatingsView } from "@/components/admin/support/ratings";
import { LossesView } from "@/components/admin/money/losses";
import { DisputesView } from "@/components/admin/money/disputes";
import { GoodsCompensationView } from "@/components/admin/money/goodsCompensation";

const m = getMessages(defaultLocale);

type Tab = "tickets" | "ratings" | "losses" | "disputes" | "goods";

export default function ProblemsPage() {
  useAuth();
  const canCall = useCanCall();
  const tabs: { key: Tab; label: string }[] = [
    ...(canCall("GET", "/tickets") ? [{ key: "tickets" as const, label: m.terms.complaints }] : []),
    ...(canCall("GET", "/ratings") ? [{ key: "ratings" as const, label: m.admin.nav.ratings }] : []),
    ...(canCall("GET", "/reports/losses") ? [{ key: "losses" as const, label: m.admin.nav.losses }] : []),
    ...(canCall("GET", "/disputes") ? [{ key: "disputes" as const, label: m.admin.nav.claims }] : []),
    ...(canCall("GET", "/losses/goods-compensations")
      ? [{ key: "goods" as const, label: m.admin.losses.goodsTab }]
      : []),
  ];
  const [tab, setTab] = useState<Tab | "">("");
  useEffect(() => {
    const t = new URLSearchParams(window.location.search).get("tab") as Tab | null;
    if (t) setTab(t);
  }, []);
  // **والصلاحيّةُ تصل بعد أوّل رسم** — فتبويبٌ غيرُ موجودٍ يُصحَّح إلى أوّل ما يملك.
  const active = tabs.some((t) => t.key === tab) ? tab : (tabs[0]?.key ?? "");
  const pick = (k: string) => {
    setTab(k as Tab);
    window.history.replaceState(null, "", `?tab=${k}`);
  };

  return (
    <PageContainer>
      <PageHeader icon={IconSupport} title={m.admin.nav.problems} subtitle={m.admin.nav.problemsHint} />
      <div className="space-y-5">
        {tabs.length > 1 && <TabCards items={tabs} active={active} onChange={pick} />}
        {active === "tickets" && <TicketsView />}
        {active === "ratings" && <RatingsView />}
        {active === "losses" && <LossesView />}
        {active === "disputes" && <DisputesView />}
        {active === "goods" && <GoodsCompensationView />}
      </div>
    </PageContainer>
  );
}
