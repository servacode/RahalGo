"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/settings/{page} — صفحةُ إعدادٍ واحدة من «متجري»**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `StorePageScreen` في `StoreHome.kt`.) تُفتح من سطرها في «متجري»،
 * **وفيها المحرّرُ القائمُ نفسُه** بلا نداءٍ جديد، وسهمُ رجوعٍ إلى «متجري».
 */

import Link from "next/link";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Badge, LoadingState } from "@rahalgo/ui";
import { useStore } from "@/lib/store/context";
import { useStoreSections } from "@/lib/store/setup";
import HoursEditor from "@/components/store/HoursEditor";
import SectionsEditor from "@/components/store/SectionsEditor";
import AddressEditor from "@/components/store/AddressEditor";
import { AmeenCard, PrepCard, RateCard, ReturnsCard, SettlementCard } from "@/components/store/StoreSettings";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const HM = m.store.home;
const K = m.store.console;

const TITLES = {
  hours: HM.rowHours,
  sections: HM.rowSections,
  address: HM.rowAddress,
  settlement: HM.rowSettlement,
  returns: HM.rowReturns,
  rate: HM.rowRate,
  prep: HM.rowPrep,
  ameen: HM.rowAmeen,
} as const;
type Page = keyof typeof TITLES;

function Sections() {
  const { store } = useStore();
  const sec = useStoreSections(store?.id ?? "");
  if (!sec.known) return <LoadingState variant="list" rows={3} />;
  return <SectionsEditor all={sec.all} mine={sec.mine} save={sec.save} />;
}

/** **قائمةُ الإعدادات الجانبيّة (حاسوبٌ فقط)** — تنقّلٌ بين الصفحات الثماني، والناقصُ يُنبَّه إليه. */
function SettingsNav({ current }: { current: Page }) {
  const { store } = useStore();
  if (!store) return null;
  const todo: Partial<Record<Page, boolean>> = {
    hours: !store.has_hours,
    settlement: !store.settlement_confirmed,
    returns: !store.returns_confirmed,
  };
  return (
    <nav className="panel hidden flex-col p-2 lg:flex" aria-label={K.settingsNav}>
      <h2 className="console-h2 px-3 py-2">{K.settingsNav}</h2>
      {(Object.keys(TITLES) as Page[])
        .filter((k) => k !== "ameen" || store.uses_ameen === true || current === "ameen")
        .map((k) => (
          <Link
            key={k}
            href={`/store/settings/${k}`}
            aria-current={k === current ? "page" : undefined}
            className={`flex min-h-11 items-center justify-between gap-2 rounded-control px-3 text-sm transition-colors ${
              k === current ? "bg-row-hover font-bold text-ink" : "text-ink-muted hover:bg-row-hover hover:text-ink"
            }`}
          >
            <span className="truncate">{TITLES[k]}</span>
            {todo[k] && <Badge variant="danger">{HM.notSet}</Badge>}
          </Link>
        ))}
    </nav>
  );
}

export default function StoreSettingsPage() {
  const params = useParams<{ page: string }>();
  const page = (params?.page ?? "") as Page;
  const { store } = useStore();
  if (!store) return null;
  if (!(page in TITLES)) return null;
  return (
    <div className="console-page" data-testid="store-settings">
      <PageHeader title={TITLES[page]} />
      <div className="grid grid-cols-1 gap-4 md:grid-cols-[minmax(0,1fr)_16rem] md:items-start md:gap-5">
        <div className="flex min-w-0 flex-col gap-4">
          {page === "hours" && <HoursEditor />}
          {page === "sections" && <Sections />}
          {page === "address" && <AddressEditor />}
          {page === "settlement" && <SettlementCard />}
          {page === "returns" && <ReturnsCard />}
          {page === "rate" && <RateCard />}
          {page === "prep" && <PrepCard />}
          {page === "ameen" && <AmeenCard />}
        </div>
        <SettingsNav current={page} />
      </div>
    </div>
  );
}
