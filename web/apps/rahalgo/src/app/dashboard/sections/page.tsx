"use client";

/**
 * **«السوق» — أصنافُ كلّ المتاجر.** (قراراتُ المالك ٢٠٢٦-١٠-٠٤)
 *
 * «السوق عندي يحوي الأصناف الخاصّة بكلّ المتاجر، لأنّ المتاجرَ أصلاً لا تُعرض
 * على الزبون». **والمتاجرُ تُدار هنا ولا تُعرض على الزبون** — واسمُها «المتاجر».
 *
 * # أربعةُ تبويبات
 *
 *	الأقسام        ←  بطاقاتٌ تُرتَّب بالسحب (هو ترتيبُها عند الزبون) · وحذفُ القسم
 *	الأصناف        ←  الأحدثُ أوّلاً بعلامة «جديد» · بحثٌ وفلاترُ وإجراءاتٌ جماعيّة
 *	المتاجر        ←  صفحةٌ موحّدة
 *	جودةُ البيانات ←  بلا صورة · بلا دوام · التجريبيُّ وحذفُه بتأكيد
 *
 * **وفتحُ السوق يُصفّر عدّادَ «المضافُ حديثاً»** في القائمة الجانبيّة — لهذا
 * الموظّف وحدَه.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { PageContainer, PageHeader, StatCard, IconStore } from "@rahalgo/ui";
import { api } from "@/lib/api";
import SectionsTab from "@/components/admin/market/SectionsTab";
import ItemsTab from "@/components/admin/market/ItemsTab";
import StoresTab from "@/components/admin/market/StoresTab";
import QualityTab from "@/components/admin/market/QualityTab";
import CategoriesTab from "@/components/admin/market/CategoriesTab";
import SetupTab from "@/components/admin/market/SetupTab";

const m = getMessages(defaultLocale);
const S = m.admin.sections;
const T = m.admin.market.tabs;

type TabKey = "sections" | "items" | "stores" | "categories" | "setup" | "quality";

export default function MarketPage() {
  // **والتبويبُ من الرابط** (`?tab=stores`) — حذفُ متجرٍ أو رجوعٌ من صفحته يعود إلى قائمة المتاجر
  // (تصميمُ اللوحة ٢٠٢٦-١٠-٢٥؛ كان الحذفُ يرمي إلى `/dashboard/merchants` وهي غيرُ موجودة).
  const [tab, setTab] = useState<TabKey>(() => {
    if (typeof window === "undefined") return "sections";
    const t = new URLSearchParams(window.location.search).get("tab") as TabKey | null;
    return t && ["sections", "items", "stores", "categories", "setup", "quality"].includes(t) ? t : "sections";
  });
  // **مربّعاتُ العدد بدل التبويبات** (طلبُ المالك ٢٠٢٦-١٠-٢٢) — العددُ وحدَه، والكبسُ يفتح قسمَه.
  const [counts, setCounts] = useState<Partial<Record<TabKey, number>>>({});

  useEffect(() => {
    const total = (r: unknown) => (r as { total?: number })?.total ?? 0;
    const len = (r: unknown) =>
      Array.isArray(r) ? r.length : ((r as { sections?: unknown[] })?.sections?.length ?? 0);
    void Promise.allSettled([
      api("/api/v1/admin/sections"),
      api("/api/v1/admin/market/items?per_page=1"),
      api("/api/v1/admin/market/stores?per_page=1"),
      api("/api/v1/admin/categories"),
      api<{ stores: { missing: string[] }[] }>("/api/v1/admin/merchants-setup"),
      api<{ items_without_image_count?: number; stores_without_hours?: unknown[]; test_stores?: unknown[]; test_items_count?: number }>(
        "/api/v1/admin/market/quality",
      ),
    ]).then(([sec, items, stores, cats, setup, quality]) => {
      const v = <T,>(r: PromiseSettledResult<T>) => (r.status === "fulfilled" ? r.value : undefined);
      setCounts({
        sections: len(v(sec)),
        items: total(v(items)),
        stores: total(v(stores)),
        categories: len(v(cats)),
        setup: (v(setup)?.stores ?? []).filter((x) => x.missing.length > 0).length,
        quality: (() => {
          const q = v(quality);
          return q
            ? (q.items_without_image_count ?? 0) + (q.stores_without_hours?.length ?? 0) +
                (q.test_stores?.length ?? 0) + (q.test_items_count ?? 0)
            : undefined;
        })(),
      });
    });
  }, []);

  // **فُتح السوق** — فيُصفَّر عدّادُ الجديد لهذا الموظّف. **وتعثّرُه لا يمنع
  // الصفحة**: العدّادُ يبقى كما هو، وهو أهونُ من شاشةٍ لا تُفتح.
  useEffect(() => {
    api("/api/v1/admin/market/seen", { method: "POST" }).catch(() => undefined);
  }, []);

  return (
    <PageContainer>
      <PageHeader icon={IconStore} title={S.title} subtitle={S.hint} />
      {/* **سطرٌ واحدٌ صغير** (طلبُ المالك ٢٠٢٦-١٠-٢٢) — ستّةٌ على الشاشة العريضة، وثلاثةٌ على الجوّال. */}
      <div className="mb-4 grid grid-cols-3 gap-2 lg:grid-cols-6">
          {(
            [
              ["sections", T.sections],
              ["items", T.items],
              ["stores", T.stores],
              ["categories", T.categories],
              ["setup", T.setup],
              ["quality", T.quality],
            ] as const
          ).map(([key, label]) => (
            <StatCard
              key={key}
              label={label}
              value={counts[key] ?? "…"}
              tone={(key === "setup" || key === "quality") && (counts[key] ?? 0) > 0 ? "warning" : "default"}
              selected={tab === key}
              onClick={() => setTab(key)}
            />
          ))}
      </div>
      {tab === "sections" && <SectionsTab />}
      {tab === "items" && <ItemsTab />}
      {tab === "stores" && <StoresTab />}
      {tab === "categories" && <CategoriesTab />}
      {tab === "setup" && <SetupTab />}
      {tab === "quality" && <QualityTab />}
    </PageContainer>
  );
}
