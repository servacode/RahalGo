"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/settings/{page} — صفحةُ إعدادٍ واحدة من «متجري»**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StorePageScreen` في `StoreHome.kt`.) تُفتح من سطرها في «متجري»،
 * **وفيها المحرّرُ القائمُ نفسُه** بلا نداءٍ جديد، وسهمُ رجوعٍ إلى «متجري».
 */

import { useParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LoadingState } from "@rahalgo/ui";
import { useStore } from "@/lib/store/context";
import { useStoreSections } from "@/lib/store/setup";
import HoursEditor from "@/components/store/HoursEditor";
import SectionsEditor from "@/components/store/SectionsEditor";
import AddressEditor from "@/components/store/AddressEditor";
import { PrepCard, RateCard, ReturnsCard, SettlementCard } from "@/components/store/StoreSettings";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const HM = m.store.home;

const TITLES = {
  hours: HM.rowHours,
  sections: HM.rowSections,
  address: HM.rowAddress,
  settlement: HM.rowSettlement,
  returns: HM.rowReturns,
  rate: HM.rowRate,
  prep: HM.rowPrep,
} as const;
type Page = keyof typeof TITLES;

function Sections() {
  const { store } = useStore();
  const sec = useStoreSections(store?.id ?? "");
  if (!sec.known) return <LoadingState variant="list" rows={3} />;
  return <SectionsEditor all={sec.all} mine={sec.mine} save={sec.save} />;
}

export default function StoreSettingsPage() {
  const params = useParams<{ page: string }>();
  const page = (params?.page ?? "") as Page;
  const { store } = useStore();
  if (!store) return null;
  if (!(page in TITLES)) return null;
  return (
    <div className="flex flex-col gap-4">
      <PageHeader title={TITLES[page]} />
      {page === "hours" && <HoursEditor />}
      {page === "sections" && <Sections />}
      {page === "address" && <AddressEditor />}
      {page === "settlement" && <SettlementCard />}
      {page === "returns" && <ReturnsCard />}
      {page === "rate" && <RateCard />}
      {page === "prep" && <PrepCard />}
    </div>
  );
}
