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
import { PageContainer, IconSupport, IconStar, IconTrendDown, IconBalance, IconSwap, IconWallet } from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";
import { TicketsView } from "@/components/admin/support/tickets";
import { RatingsView } from "@/components/admin/support/ratings";
import { LossesView } from "@/components/admin/money/losses";
import { DisputesView } from "@/components/admin/money/disputes";
import { GoodsCompensationView } from "@/components/admin/money/goodsCompensation";
import { CompensationsView } from "@/components/admin/money/CompensationsView";

const m = getMessages(defaultLocale);

type Tab = "tickets" | "ratings" | "losses" | "disputes" | "goods" | "compensations";

export default function ProblemsPage() {
  useAuth();
  const canCall = useCanCall();
  // **تبويباتٌ بسطرٍ واحد** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «خلّيها بسطر واحد بطريقة واضحة إنّهن تبويبات»).
  const tabs = [
    ...(canCall("GET", "/tickets") ? [{ key: "tickets" as const, label: m.terms.complaints, icon: IconSupport }] : []),
    ...(canCall("GET", "/ratings") ? [{ key: "ratings" as const, label: m.admin.nav.ratings, icon: IconStar }] : []),
    ...(canCall("GET", "/reports/losses") ? [{ key: "losses" as const, label: m.admin.nav.losses, icon: IconTrendDown }] : []),
    ...(canCall("GET", "/disputes") ? [{ key: "disputes" as const, label: m.admin.nav.claims, icon: IconBalance }] : []),
    ...(canCall("GET", "/losses/goods-compensations")
      ? [{ key: "goods" as const, label: m.admin.losses.goodsTab, icon: IconSwap }]
      : []),
    // **والتعويضاتُ هنا لا في الخزينة** (قرارُ المالك ٢٠٢٦-١٠-٢٧).
    ...(canCall("GET", "/compensations")
      ? [{ key: "compensations" as const, label: m.admin.treasury.links.compensations, icon: IconWallet }]
      : []),
  ];
  const [tab, setTab] = useState<Tab | "">("");
  useEffect(() => {
    const t = new URLSearchParams(window.location.search).get("tab") as Tab | null;
    if (t) setTab(t);
  }, []);
  // **والصلاحيّةُ تصل بعد أوّل رسم** — فتبويبٌ غيرُ موجودٍ يُصحَّح إلى أوّل ما يملك.
  const active: Tab | "" = tabs.some((t) => t.key === tab) ? tab : (tabs[0]?.key ?? "");
  const pick = (k: Tab) => {
    setTab(k as Tab);
    window.history.replaceState(null, "", `?tab=${k}`);
  };

  return (
    <PageContainer>
      {/* **رأسٌ واحدٌ مرتّب** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «التصميم الاحترافي») — العنوانُ بأيقونته وتحته
          التبويباتُ بسطرٍ واحد، **ولا عنوانَ ثانٍ داخل التبويب.** */}
      <section className="surface mb-4 overflow-hidden">
        <div className="flex items-center gap-3 p-4">
          <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
            <IconSupport size={22} />
          </span>
          <div className="min-w-0">
            <h1 className="heading-page">{m.admin.nav.problems}</h1>
            <p className="text-xs text-ink-muted">{m.admin.nav.problemsHint}</p>
          </div>
        </div>
        {tabs.length > 1 && active && (
          <nav className="flex gap-1 overflow-x-auto border-t border-line-soft px-2" role="tablist">
            {tabs.map(({ key, label, icon: Icon }) => {
              const on = active === key;
              return (
                <button
                  key={key}
                  type="button"
                  role="tab"
                  aria-selected={on}
                  onClick={() => pick(key)}
                  className={`relative inline-flex shrink-0 items-center gap-2 px-3 py-3 text-sm font-bold transition-colors ${
                    on ? "text-ink" : "text-ink-muted hover:text-ink"
                  }`}
                >
                  <Icon size={16} />
                  {label}
                  {on && <span className="absolute inset-x-2 bottom-0 h-0.5 rounded-full bg-accent" />}
                </button>
              );
            })}
          </nav>
        )}
      </section>
      <div>
        {active === "tickets" && <TicketsView />}
        {active === "ratings" && <RatingsView />}
        {active === "losses" && <LossesView />}
        {active === "disputes" && <DisputesView />}
        {active === "goods" && <GoodsCompensationView />}
        {active === "compensations" && <CompensationsView embedded />}
      </div>
    </PageContainer>
  );
}
