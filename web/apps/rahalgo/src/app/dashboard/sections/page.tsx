"use client";

/**
 * **«السوق» — أصنافُ كلّ المتاجر.** (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
 *
 * «السوق عندي يحوي الأصناف الخاصّة بكلّ المتاجر، لأنّ المتاجرَ أصلاً لا تُعرض
 * على الزبون». **والمتاجرُ تُدار هنا ولا تُعرض على الزبون** — واسمُها «المتاجر».
 *
 * # ستّةُ تبويبات
 *
 *	الأقسام        ←  بطاقاتٌ تُرتَّب بالسحب أو بالأسهم (هو ترتيبُها عند الزبون) · وحذفُ القسم
 *	الأصناف        ←  الأحدثُ أوّلاً بعلامة «جديد» · بحثٌ وفلاترُ وإجراءاتٌ جماعيّة
 *	المتاجر        ←  صفحةٌ موحّدة بفلاتر الحال
 *	التصنيفات      ←  مطاعم · بقالة… وترتيبُها
 *	إعداد المتاجر  ←  مين ضبط ومين لا
 *	جودةُ البيانات ←  بلا صورة · بلا دوام · التجريبيُّ وحذفُه بتأكيد
 *
 * **والتبويبُ في الرابط** (`?tab=items`، تدقيقُ السوق ٢٠٢٦-١٠-٢٧) — كان في الذاكرة وحدَها، فزرُّ «صفحة
 * المتجر» في خريطة العمليات (`?merchant=ID`) يفتح الأقسام، والتحديثُ والرجوعُ يضيّعانه.
 * **و`?merchant=ID` يفتح الأصنافَ مرشَّحةً بمتجره.**
 *
 * **وفتحُ السوق يُصفّر عدّادَ «المضافُ حديثاً»** في القائمة الجانبيّة — لهذا الموظّف وحدَه.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  PageContainer,
  useLiveData,
  IconStore,
  IconGrid,
  IconList,
  IconOrder,
  IconSettings,
  IconShieldCheck,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { PageHero, type HeroTab } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";
import SectionsTab from "@/components/admin/market/SectionsTab";
import ItemsTab from "@/components/admin/market/ItemsTab";
import StoresTab from "@/components/admin/market/StoresTab";
import QualityTab from "@/components/admin/market/QualityTab";
import CategoriesTab from "@/components/admin/market/CategoriesTab";
import SetupTab from "@/components/admin/market/SetupTab";

const m = getMessages(defaultLocale);
const S = m.admin.sections;
const T = m.admin.market.tabs;
const U = m.admin.market.ui;

type TabKey = "sections" | "items" | "stores" | "categories" | "setup" | "quality";
const TAB_KEYS: TabKey[] = ["sections", "items", "stores", "categories", "setup", "quality"];

function readParam(key: string): string {
  if (typeof window === "undefined") return "";
  return new URLSearchParams(window.location.search).get(key) ?? "";
}

interface Counts {
  sections?: number;
  items?: number;
  stores?: number;
  categories?: number;
  setup?: number;
  noImage?: number;
  noHours?: number;
  test?: number;
}

/** **رقمٌ تعذّر جلبُه لا يُقال صفراً** — `undefined` يبقى «…». */
const opt = <T,>(p: Promise<T>) => p.catch(() => undefined);

export default function MarketPage() {
  const canCall = useCanCall();
  const canSetup = canCall("GET", "/merchants-setup");
  // **التبويبُ من الرابط** — و`?merchant=` وحدَه يعني «أصنافُ هذا المتجر».
  const [tab, setTabRaw] = useState<TabKey>(() => {
    const t = readParam("tab") as TabKey;
    if (TAB_KEYS.includes(t)) return t;
    return readParam("merchant") ? "items" : "sections";
  });
  const [merchant] = useState(() => readParam("merchant"));

  function setTab(k: TabKey) {
    setTabRaw(k);
    try {
      const url = new URL(window.location.href);
      url.searchParams.set("tab", k);
      if (k !== "items") url.searchParams.delete("merchant");
      window.history.replaceState(window.history.state, "", url.toString());
    } catch {
      // @empty-ok — رابطٌ لم يتحدّث لا يمنع فتحَ التبويب.
    }
  }

  // **الأرقامُ تتحدّث وحدَها** (تدقيقُ السوق ٢٠٢٦-١٠-٢٧) — كانت تُجلب مرّةً عند الفتح فتبقى قديمة.
  const { data: counts } = useLiveData<Counts>(
    async () => {
      const [sec, items, stores, cats, setup, quality] = await Promise.all([
        opt(api<{ sections?: unknown[] }>("/api/v1/admin/sections")),
        opt(api<{ total?: number }>("/api/v1/admin/market/items?per_page=1")),
        opt(api<{ total?: number }>("/api/v1/admin/market/stores?per_page=1")),
        opt(api<unknown[]>("/api/v1/admin/categories")),
        canSetup ? opt(api<{ stores: { missing: string[] }[] }>("/api/v1/admin/merchants-setup")) : Promise.resolve(undefined),
        opt(
          api<{
            items_without_image_count?: number;
            stores_without_hours?: unknown[];
            test_stores?: unknown[];
            test_items_count?: number;
          }>("/api/v1/admin/market/quality"),
        ),
      ]);
      return {
        sections: sec?.sections?.length,
        items: items?.total,
        stores: stores?.total,
        categories: Array.isArray(cats) ? cats.length : undefined,
        setup: setup ? setup.stores.filter((x) => x.missing.length > 0).length : undefined,
        noImage: quality?.items_without_image_count,
        noHours: quality?.stores_without_hours?.length,
        test: quality ? (quality.test_stores?.length ?? 0) + (quality.test_items_count ?? 0) : undefined,
      };
    },
    ["catalog", "menu", "merchant"],
    [canSetup],
  );

  // **فُتح السوق** — فيُصفَّر عدّادُ الجديد لهذا الموظّف. **وتعثّرُه لا يمنع
  // الصفحة**: العدّادُ يبقى كما هو، وهو أهونُ من شاشةٍ لا تُفتح.
  useEffect(() => {
    api("/api/v1/admin/market/seen", { method: "POST" }).catch(() => undefined);
  }, []);

  // **ومن لا يملك إعدادَ المتاجر وفتح رابطَه يرى الأقسام** — لا صفحةً فارغة.
  const shown: TabKey = tab === "setup" && !canSetup ? "sections" : tab;
  const c = counts ?? {};
  const quality =
    c.noImage === undefined || c.noHours === undefined || c.test === undefined ? undefined : c.noImage + c.noHours + c.test;
  const num = (n: number | undefined) => (n === undefined ? "…" : fmtNum(n));

  const tabs: HeroTab<TabKey>[] = [
    { key: "sections", label: T.sections, icon: IconGrid, count: c.sections },
    { key: "items", label: T.items, icon: IconOrder, count: c.items },
    { key: "stores", label: T.stores, icon: IconStore, count: c.stores },
    { key: "categories", label: T.categories, icon: IconList, count: c.categories },
    ...(canSetup ? [{ key: "setup" as const, label: T.setup, icon: IconSettings, count: c.setup, warn: true }] : []),
    { key: "quality", label: T.quality, icon: IconShieldCheck, count: quality, warn: true },
  ];

  return (
    <PageContainer>
      <PageHero<TabKey>
        icon={IconStore}
        title={S.title}
        subtitle={U.heroHint}
        tabs={tabs}
        value={shown}
        onChange={setTab}
      >
        {/* **الأرقامُ تُكبس فتفتح تبويبَها** — والبرتقاليُّ فيه شغل. */}
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-5">
          <Tile icon={IconGrid} label={U.tileSections} value={num(c.sections)} tone="accent" onClick={() => setTab("sections")} />
          <Tile icon={IconOrder} label={U.tileItems} value={num(c.items)} tone="accent" onClick={() => setTab("items")} />
          <Tile icon={IconStore} label={U.tileStores} value={num(c.stores)} tone="accent" onClick={() => setTab("stores")} />
          {canSetup && (
            <Tile
              icon={IconSettings}
              label={U.tileSetup}
              value={num(c.setup)}
              tone={(c.setup ?? 0) > 0 ? "warning" : "success"}
              onClick={() => setTab("setup")}
            />
          )}
          <Tile
            icon={IconShieldCheck}
            label={U.tileQuality}
            value={num(quality)}
            sub={
              quality === undefined
                ? undefined
                : U.qualitySub
                    .replace("{a}", fmtNum(c.noImage ?? 0))
                    .replace("{b}", fmtNum(c.noHours ?? 0))
                    .replace("{c}", fmtNum(c.test ?? 0))
            }
            tone={(quality ?? 0) > 0 ? "warning" : "success"}
            onClick={() => setTab("quality")}
          />
        </div>
      </PageHero>
      {shown === "sections" && <SectionsTab />}
      {shown === "items" && <ItemsTab initialMerchant={merchant} />}
      {shown === "stores" && <StoresTab />}
      {shown === "categories" && <CategoriesTab />}
      {shown === "setup" && <SetupTab />}
      {shown === "quality" && <QualityTab />}
    </PageContainer>
  );
}
