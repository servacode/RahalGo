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
import { PageContainer, PageHeader, Tabs, IconStore } from "@rahalgo/ui";
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

  // **فُتح السوق** — فيُصفَّر عدّادُ الجديد لهذا الموظّف. **وتعثّرُه لا يمنع
  // الصفحة**: العدّادُ يبقى كما هو، وهو أهونُ من شاشةٍ لا تُفتح.
  useEffect(() => {
    api("/api/v1/admin/market/seen", { method: "POST" }).catch(() => undefined);
  }, []);

  return (
    <PageContainer>
      <PageHeader icon={IconStore} title={S.title} subtitle={S.hint} />
      <Tabs<TabKey>
        className="mb-4"
        value={tab}
        onChange={setTab}
        items={[
          { key: "sections", label: T.sections },
          { key: "items", label: T.items },
          { key: "stores", label: T.stores },
          { key: "categories", label: T.categories },
          { key: "setup", label: T.setup },
          { key: "quality", label: T.quality },
        ]}
      />
      {tab === "sections" && <SectionsTab />}
      {tab === "items" && <ItemsTab />}
      {tab === "stores" && <StoresTab />}
      {tab === "categories" && <CategoriesTab />}
      {tab === "setup" && <SetupTab />}
      {tab === "quality" && <QualityTab />}
    </PageContainer>
  );
}
