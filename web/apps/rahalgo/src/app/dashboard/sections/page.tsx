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
import {
  PageContainer,
  IconStore,
  IconGrid,
  IconList,
  IconOrder,
  IconSettings,
  IconShieldCheck,
} from "@rahalgo/ui";
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
  const [tab, setTab] = useState<TabKey>("sections");
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
      {/* **رأسٌ واحدٌ مرتّب** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «تصميم احترافي»): العنوانُ بأيقونته، وتحته
          التبويباتُ بسطرٍ واحد — أيقونةٌ واسمٌ وعدٌّ — **والناقصُ يُلوَّن تنبيهاً**. */}
      <section className="surface mb-4 overflow-hidden">
        <div className="flex items-center gap-3 p-4">
          <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
            <IconStore size={22} />
          </span>
          <div className="min-w-0">
            <h1 className="heading-page">{S.title}</h1>
            <p className="truncate text-xs text-ink-muted">{S.hint}</p>
          </div>
        </div>
        <nav className="flex gap-1 overflow-x-auto border-t border-line-soft px-2" role="tablist">
          {(
            [
              ["sections", T.sections, IconGrid],
              ["items", T.items, IconOrder],
              ["stores", T.stores, IconStore],
              ["categories", T.categories, IconList],
              ["setup", T.setup, IconSettings],
              ["quality", T.quality, IconShieldCheck],
            ] as const
          ).map(([key, label, Icon]) => {
            const on = tab === key;
            const n = counts[key];
            const warn = (key === "setup" || key === "quality") && (n ?? 0) > 0;
            return (
              <button
                key={key}
                type="button"
                role="tab"
                aria-selected={on}
                onClick={() => setTab(key)}
                className={`relative inline-flex shrink-0 items-center gap-2 px-3 py-3 text-sm font-bold transition-colors ${
                  on ? "text-ink" : "text-ink-muted hover:text-ink"
                }`}
              >
                <Icon size={17} />
                {label}
                <span
                  className={`min-w-[1.5rem] rounded-full px-1.5 py-0.5 text-center text-2xs tabular-nums ${
                    warn ? "bg-warning-solid text-on-solid" : on ? "bg-accent text-on-bright" : "bg-field text-ink-muted"
                  }`}
                >
                  {n ?? "…"}
                </span>
                {on && <span className="absolute inset-x-2 bottom-0 h-0.5 rounded-full bg-accent" />}
              </button>
            );
          })}
        </nav>
      </section>
      {tab === "sections" && <SectionsTab />}
      {tab === "items" && <ItemsTab />}
      {tab === "stores" && <StoresTab />}
      {tab === "categories" && <CategoriesTab />}
      {tab === "setup" && <SetupTab />}
      {tab === "quality" && <QualityTab />}
    </PageContainer>
  );
}
