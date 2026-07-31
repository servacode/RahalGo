"use client";

/** الأصناف — المحرّر المركزي نفسه بمسارات بوابة المتجر. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { MenuManager, PageContainer, PageHeader, LoadingState, IconStore, type MenuPaths } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store";

const m = getMessages(defaultLocale);

/** مسارات بوابة المتجر — الحارس في الخادم يتحقق من الملكية. */
const PATHS: MenuPaths = {
  menu: (id) => `/api/v1/merchant/stores/${id}/menu`,
  sections: (id) => `/api/v1/merchant/stores/${id}/menu/sections`,
  section: (id) => `/api/v1/merchant/menu/sections/${id}`,
  items: (id) => `/api/v1/merchant/stores/${id}/menu/items`,
  item: (id) => `/api/v1/merchant/menu/items/${id}`,
};

export default function MerchantMenuPage() {
  const { store } = useStore();
  if (!store) return <LoadingState />;
  return (
    <PageContainer>
      <PageHeader icon={IconStore} title={m.terms.menu} subtitle={store.name} />
      <MenuManager api={api} paths={PATHS} merchantID={store.id} />
    </PageContainer>
  );
}
