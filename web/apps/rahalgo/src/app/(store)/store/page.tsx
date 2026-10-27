"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store — «متجري»: الشاشةُ نفسُها التي في التطبيق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StoreScreen.kt` — المرحلةُ الثانية، قرارُ المالك ٢٠٢٦-١٠-١٧.)
 *
 * **بالترتيب نفسِه**: ما ينقص المتجر (وكلُّ سطرٍ يقود إلى بطاقته) · الاسمُ
 * وقلمُه · «إغلاق طارئ» · أرقامُ اليوم · طريقةُ استلام المستحقّات ·
 * الاسترداد · سعرُ الصرف وعملةُ العرض · مدّةُ التحضير · ساعاتُ العمل ·
 * أقسامُ السوق · العنوانُ ونقطتُه.
 *
 * **والحالُ ومفتاحُ «مفتوح / مغلق» في الشريط فوق** (`StoreBar`) — كما في
 * كلّ شاشة.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, damascusDay, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { fill } from "@/lib/customer/texts";
import { IconCheck, LoadingState, StatCard, StatGrid, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { SectionRef, StoreReport } from "@/lib/store/types";
import {
  EmergencyButton,
  LogoCard,
  NameCard,
  PrepCard,
  RateCard,
  ReturnsCard,
  SettlementCard,
} from "@/components/store/StoreSettings";
import HoursEditor from "@/components/store/HoursEditor";
import { OrderPhonesCard, StoreStaffCard } from "@/components/admin/StoreStaffAdmin";
import SectionsEditor from "@/components/store/SectionsEditor";
import OptionPresetsCard from "@/components/store/OptionPresetsCard";
import AddressEditor from "@/components/store/AddressEditor";

const m = getMessages(defaultLocale);
const S = m.store;
const T = S.settings;

type Spot = "sections" | "hours" | "address" | "settlement" | "returns";

function jump(spot: Spot) {
  document.getElementById(`spot-${spot}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
}

export default function StoreHome() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const load = useCallback(() => {
    const today = damascusDay();
    return api<StoreReport>(`/api/v1/merchant/stores/${id}/reports?from=${today}&to=${today}`);
  }, [id]);
  const { data: report, loading } = useLiveData(load, ["order"], [id]);

  const [all, setAll] = useState<SectionRef[]>([]);
  const [mine, setMine] = useState<SectionRef[] | null>(null);
  useEffect(() => {
    if (!id) return;
    api<{ sections: SectionRef[] }>("/api/v1/merchant/platform-sections")
      .then((r) => setAll(r.sections ?? []))
      .catch(() => undefined);
    api<{ sections: SectionRef[] }>(`/api/v1/merchant/stores/${id}/sections`)
      .then((r) => setMine(r.sections ?? []))
      // **وتعذّرُها لا يُقال «لا أقسام»** — تبقى مجهولةً فلا يُعرض نقصٌ ولا محرّر.
      .catch(() => undefined);
  }, [id]);

  if (!store) return null;
  const suspended = store.status === "suspended";

  // ══════════════════════════════════════════════════════════════════
  // **خطواتُ التجهيز بتقدّمها — لا كرتٌ أحمر** (مرآةُ `StoreScreen.kt`، تدقيقُ ٢٠٢٦-١٠-٢٧)
  // ══════════════════════════════════════════════════════════════════
  //
  // **كان كرتاً بلون الإيقاف** — فيظنّ المتجرُ الجديدُ أنّ عليه مشكلة، **وهي
  // خطواتُ تجهيزٍ عاديّة.** فصار شريطَ تقدّمٍ وكلُّ خطوةٍ بدائرتها (✓ لِما تمّ).
  // **وأقسامٌ لم تُقرأ بعدُ لا تُعدّ ناقصة.**
  const steps: { spot: Spot; done: boolean }[] = [
    { spot: "sections", done: mine == null || mine.length > 0 },
    { spot: "hours", done: store.has_hours },
    { spot: "address", done: store.lat != null && !!store.address_text.trim() },
    { spot: "settlement", done: store.settlement_confirmed },
    { spot: "returns", done: store.returns_confirmed },
  ];
  const doneCount = steps.filter((x) => x.done).length;

  return (
    <div className="flex flex-col gap-3">
      {doneCount < steps.length && !suspended && (
        <section className="flex flex-col gap-2 rounded-card border border-accent-edge bg-accent-tint p-3" data-testid="store-missing">
          <p className="font-bold text-accent-text">{fill(T.setupProgress, { done: fmtNum(doneCount), total: fmtNum(steps.length) })}</p>
          <div className="h-2 overflow-hidden rounded-badge bg-field" role="presentation">
            <div className="h-full rounded-badge bg-accent transition-all" style={{ width: `${(doneCount / steps.length) * 100}%` }} />
          </div>
          <ul className="flex flex-col">
            {steps.map(({ spot, done }) => (
              <li key={spot}>
                <button
                  type="button"
                  disabled={done}
                  onClick={() => jump(spot)}
                  className="flex min-h-12 w-full items-center gap-3 text-start text-sm disabled:cursor-default"
                >
                  <span
                    className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-badge border-2 ${
                      done ? "border-success bg-success text-on-bright" : "border-accent"
                    }`}
                    aria-hidden
                  >
                    {done && <IconCheck size={14} />}
                  </span>
                  <span className={`min-w-0 flex-1 ${done ? "text-ink-muted" : ""}`}>{T.setup[spot]}</span>
                  {!done && (
                    <span className="font-bold text-accent-text" aria-hidden>
                      ←
                    </span>
                  )}
                </button>
              </li>
            ))}
          </ul>
        </section>
      )}

      <NameCard />
      <LogoCard />
      {/* **موظّفوه ورقمُ طلباته بحسب الدوام** (قرارُ المالك ٢٠٢٦-١٠-٢٠) — لصاحب المتجر. */}
      <OrderPhonesCard url={`/api/v1/merchant/stores/${id}/order-phones`} canManage={!suspended} />
      <StoreStaffCard base="/api/v1/merchant/staff" canManage={!suspended} />
      <EmergencyButton />

      {loading && !report ? (
        <LoadingState variant="stats" rows={4} />
      ) : (
        report && (
          <StatGrid>
            <StatCard label={S.today.orders} value={fmtNum(report.summary.orders)} />
            <StatCard label={S.today.delivered} value={fmtNum(report.summary.delivered)} tone="success" />
            <StatCard label={S.today.cancelled} value={fmtNum(report.summary.cancelled)} tone="muted" />
            <StatCard
              label={S.today.due}
              value={fmtMoney(report.summary.sales - report.summary.platform_commission)}
              tone="accent"
              emphasis
            />
          </StatGrid>
        )
      )}

      {/* **وعنوانٌ يفصل اليوميَّ عن إعداداتٍ تُضبط مرّة** (تدقيقُ ٢٠٢٦-١٠-٢٧). */}
      <h2 className="heading-section pt-3">{T.title}</h2>
      <div id="spot-settlement" className="scroll-mt-20">
        <SettlementCard />
      </div>
      <div id="spot-returns" className="scroll-mt-20">
        <ReturnsCard />
      </div>
      <RateCard />
      <PrepCard />
      <div id="spot-hours" className="scroll-mt-20">
        <HoursEditor />
      </div>
      <div id="spot-sections" className="scroll-mt-20">
        {mine != null && <SectionsEditor all={all} mine={mine} onChange={setMine} />}
      </div>
      {/* **«خياراتي»** — مجموعاتُ الخيارات في محرّر الصنف (٢٠٢٦-١٠-٢٧). */}
      <OptionPresetsCard />
      <div id="spot-address" className="scroll-mt-20">
        <AddressEditor />
      </div>
    </div>
  );
}
