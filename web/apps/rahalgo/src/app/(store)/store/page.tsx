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
import { IconWarning, LoadingState, StatCard, StatGrid, useLiveData } from "@rahalgo/ui";
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
import SectionsEditor from "@/components/store/SectionsEditor";
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

  const missing: Spot[] = [];
  if (mine != null && mine.length === 0) missing.push("sections");
  if (!store.has_hours) missing.push("hours");
  if (store.lat == null || !store.address_text.trim()) missing.push("address");
  if (!store.settlement_confirmed) missing.push("settlement");
  if (!store.returns_confirmed) missing.push("returns");

  return (
    <div className="flex flex-col gap-3">
      {missing.length > 0 && !suspended && (
        <section className="flex flex-col gap-1 rounded-card border border-danger-edge bg-danger-tint p-3" data-testid="store-missing">
          <p className="flex items-center gap-2 font-bold text-danger">
            <IconWarning size={18} className="shrink-0" />
            {T.setupTitle}
          </p>
          {missing.map((s) => (
            <button key={s} type="button" onClick={() => jump(s)} className="py-1.5 text-start text-sm hover:underline">
              {`• ${T.setup[s]}  ←`}
            </button>
          ))}
        </section>
      )}

      <NameCard />
      <LogoCard />
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
      <div id="spot-address" className="scroll-mt-20">
        <AddressEditor />
      </div>
    </div>
  );
}
