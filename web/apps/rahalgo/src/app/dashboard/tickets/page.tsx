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
 *
 * **وعلى كلّ تبويبٍ عددُ ما ينتظر** (فحصُ ٢٠٢٦-١٠-٢٧) — شكاوى غيرُ محلولة (والمتأخّرةُ بالأحمر)،
 * نزاعاتٌ مفتوحة، تعويضاتٌ تنتظر قراراً، بضاعةٌ بلا تعويض — فيُعرف أين الشغلُ بلا فتح كلّ تبويب.
 * **والتبويبُ يُقرأ من الرابط في أوّل رسم** — كان يُقرأ بعده فيومض تبويبُ الشكاوى ويطلب الخادم.
 */

import { useCallback, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  PageContainer,
  useLiveRefresh,
  IconSupport,
  IconStar,
  IconTrendDown,
  IconBalance,
  IconSwap,
  IconWallet,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
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
const TABS: readonly Tab[] = ["tickets", "ratings", "losses", "disputes", "goods", "compensations"];

/** **عددُ ما ينتظر على التبويب** — و`danger` يلوّنه بالأحمر (متأخّر). */
type Counts = Partial<Record<Tab, { n: number; danger?: boolean }>>;

export default function ProblemsPage() {
  useAuth();
  const canCall = useCanCall();
  const canTickets = canCall("GET", "/tickets");
  const canDisputes = canCall("GET", "/disputes");
  const canGoods = canCall("GET", "/losses/goods-compensations");
  const canComps = canCall("GET", "/compensations");
  // **تبويباتٌ بسطرٍ واحد** (قرارُ المالك ٢٠٢٦-١٠-٢٥: «خلّيها بسطر واحد بطريقة واضحة إنّهن تبويبات»).
  const tabs = [
    ...(canTickets ? [{ key: "tickets" as const, label: m.terms.complaints, icon: IconSupport }] : []),
    ...(canCall("GET", "/ratings") ? [{ key: "ratings" as const, label: m.admin.nav.ratings, icon: IconStar }] : []),
    ...(canCall("GET", "/reports/losses") ? [{ key: "losses" as const, label: m.admin.nav.losses, icon: IconTrendDown }] : []),
    ...(canDisputes ? [{ key: "disputes" as const, label: m.admin.nav.claims, icon: IconBalance }] : []),
    ...(canGoods ? [{ key: "goods" as const, label: m.admin.losses.goodsTab, icon: IconSwap }] : []),
    // **والتعويضاتُ هنا لا في الخزينة** (قرارُ المالك ٢٠٢٦-١٠-٢٧).
    ...(canComps ? [{ key: "compensations" as const, label: m.admin.treasury.links.compensations, icon: IconWallet }] : []),
  ];
  const params = useSearchParams();
  const fromUrl = params.get("tab") as Tab | null;
  const [tab, setTab] = useState<Tab | "">(fromUrl && TABS.includes(fromUrl) ? fromUrl : "");
  // **والصلاحيّةُ تصل بعد أوّل رسم** — فتبويبٌ غيرُ موجودٍ يُصحَّح إلى أوّل ما يملك.
  const active: Tab | "" = tabs.some((t) => t.key === tab) ? tab : (tabs[0]?.key ?? "");
  const pick = (k: Tab) => {
    setTab(k);
    window.history.replaceState(null, "", `?tab=${k}`);
  };

  // ── أعدادُ التبويبات — كلُّ نداءٍ بقدرة بابه، وتعذّرُ واحدٍ لا يُسقط غيرَه ──
  const [counts, setCounts] = useState<Counts>({});
  const loadCounts = useCallback(async () => {
    const [tk, dp, cp, gd] = await Promise.allSettled([
      canTickets
        ? api<{ total: number; late?: number }>("/api/v1/admin/tickets?status=unresolved&per_page=1")
        : Promise.reject(),
      canDisputes ? api<{ open_count: number }>("/api/v1/admin/disputes?party=&status=open&page=1") : Promise.reject(),
      canComps
        ? api<{ summary?: { pending_count: number; overdue_count: number } }>(
            "/api/v1/admin/compensations?status=pending&per_page=1",
          )
        : Promise.reject(),
      canGoods
        ? api<{ orders: { request_status: string }[] }>("/api/v1/admin/losses/goods-compensations")
        : Promise.reject(),
    ]);
    const next: Counts = {};
    if (tk.status === "fulfilled" && tk.value) next.tickets = { n: tk.value.total, danger: (tk.value.late ?? 0) > 0 };
    if (dp.status === "fulfilled" && dp.value) next.disputes = { n: dp.value.open_count };
    if (cp.status === "fulfilled" && cp.value?.summary)
      next.compensations = { n: cp.value.summary.pending_count, danger: cp.value.summary.overdue_count > 0 };
    if (gd.status === "fulfilled" && gd.value)
      next.goods = { n: (gd.value.orders ?? []).filter((o) => o.request_status !== "pending").length };
    setCounts(next);
  }, [canTickets, canDisputes, canComps, canGoods]);
  useEffect(() => {
    void loadCounts();
  }, [loadCounts]);
  useLiveRefresh(["ticket", "dispute", "order", "wallet"], loadCounts);

  return (
    <PageContainer>
      {/* **رأسٌ واحدٌ مرتّب** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «التصميم الاحترافي») — العنوانُ بأيقونته وتحته
          التبويباتُ بسطرٍ واحد، **ولا عنوانَ ثانٍ داخل التبويب.** */}
      <section className="surface overflow-hidden">
        <div className="flex items-center gap-3 p-4">
          <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
            <IconSupport size={22} />
          </span>
          <div className="min-w-0">
            <h1 className="heading-page">{m.admin.nav.problems}</h1>
            <p className="text-xs text-ink-muted">{m.admin.tickets.pageHint}</p>
          </div>
        </div>
        {tabs.length > 1 && active && (
          <nav className="flex gap-1 overflow-x-auto border-t border-line-soft px-2" role="tablist">
            {tabs.map(({ key, label, icon: Icon }) => {
              const on = active === key;
              const c = counts[key];
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
                  {c && c.n > 0 && (
                    <span
                      className={`rounded-full px-1.5 text-2xs tabular-nums ${
                        c.danger ? "bg-danger-solid text-on-solid" : "bg-warning-solid text-on-solid"
                      }`}
                    >
                      {fmtNum(c.n)}
                    </span>
                  )}
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
