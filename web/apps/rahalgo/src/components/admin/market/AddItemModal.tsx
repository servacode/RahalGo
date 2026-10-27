"use client";

/**
 * **إضافةُ صنفٍ من السوق** — نُقلت من صفحة القسم إلى تبويب الأصناف (قرارُ المالك ٢٠٢٦-١٠-٢٥:
 * «بالأصناف نحطّ خيار أضف صنفاً ونحذفه من القسم»).
 *
 * # خطوتان: المتجرُ بالبحث، ثمّ نافذةُ الصنف نفسُها
 *
 * (تدقيقُ السوق ٢٠٢٦-١٠-٢٧.) **كانت النافذةُ تسأل عن «قسمه في قائمة المتجر»** من `GET /menu` — وهو
 * لا يردّ إلّا أقسامَ السوق التي للمتجر فيها صنفٌ من قبل، **فمتجرٌ جديدٌ بلا أصنافٍ يقرأ «لا أقسام —
 * أنشئ قسماً أوّلاً» ولا مكانَ لإنشائه**: لا يُضاف له صنفٌ أبداً. **والحقلُ يُرسَل `section_id` ولا
 * يكتبه الخادمُ أصلاً**، والسعرُ الفارغ يُحفظ صفراً بلا تنبيه.
 *
 * **فصار**: يُختار المتجرُ بالبحث (لا قائمةٌ تنقص بعد مئة)، **ثمّ تُفتح نافذةُ تعديل الصنف نفسُها**
 * (`MenuItemEditor`) — بالليرة أو الدولار والوصف والإضافات، **كتطبيق المتجر بالضبط.** والحفظُ من بابِ
 * السوق (`POST /market/stores/{id}/items`) بقدرة السوق، والصورةُ من باب صوره.
 */

import { useState } from "react";
import { getMessages, defaultLocale, storeLabel } from "@rahalgo/i18n";
import { MenuItemEditor, type MenuPaths } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import ImageUpload from "@/components/admin/ImageUpload";
import { StorePickerModal, type PickedStore } from "./MarketParts";

export const MARKET_MEDIA = "/api/v1/admin/market/media";
const m = getMessages(defaultLocale);
const U = m.admin.market.ui;

/**
 * **مساراتُ السوق لنافذة الصنف** — الإضافةُ والتعديلُ من باب السوق (`market.manage`) لا من باب إدارة
 * المتاجر، **فموظّفُ السوق لا يُردّ ٤٠٣ على صنفٍ هو صاحبُ تعديله.** وسعرُ صرف المتجر لمن يقرأ المتاجر.
 */
export function marketMenuPaths(canReadStore: boolean): MenuPaths {
  return {
    menu: (id) => `/api/v1/admin/merchants/${id}/menu`,
    items: (id) => `/api/v1/admin/market/stores/${id}/items`,
    item: (id) => `/api/v1/admin/market/items/${id}`,
    platformSections: () => `/api/v1/admin/sections`,
    ...(canReadStore ? { merchant: (id: string) => `/api/v1/admin/merchants/${id}` } : {}),
  };
}

/** **صورةُ الصنف في النافذة** — من باب صور السوق لا من باب المحتوى. */
export function marketItemImage(initialUrl: string | null | undefined, onChange: (id: string | null) => void) {
  return <ImageUpload path={MARKET_MEDIA} kind="menu_item" label={m.shared.menuEditor.itemImage} initialUrl={initialUrl} onChange={onChange} />;
}

export default function AddItemModal({
  sections,
  onClose,
  onSaved,
}: {
  /** **أقسامُ السوق الفعّالةُ وحدَها** — والمطفأُ يردّه المحرّك «غير موجود». */
  sections: { id: string; name: string; active?: boolean }[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const canReadStore = useCanCall()("GET", "/merchants/{id}");
  const [store, setStore] = useState<PickedStore | null>(null);

  if (!store) {
    return <StorePickerModal title={U.addPickStoreTitle} hint={U.addPickStoreHint} onPick={setStore} onClose={onClose} />;
  }
  return (
    <MenuItemEditor
      api={api}
      paths={marketMenuPaths(canReadStore)}
      merchantID={store.id}
      platformSections={sections.filter((s) => s.active !== false)}
      editing={{ item: null, sectionId: "" }}
      imageUpload={marketItemImage}
      requirePositivePrice
      note={
        <span className="flex flex-wrap items-center justify-between gap-2">
          <span>
            <span className="text-ink-muted">{U.addForStore} </span>
            <b>{storeLabel(store.name, store.number)}</b>
          </span>
          <button type="button" className="text-xs font-bold text-accent hover:underline" onClick={() => setStore(null)}>
            {U.changeStore}
          </button>
        </span>
      }
      onClose={onClose}
      onSaved={onSaved}
    />
  );
}
