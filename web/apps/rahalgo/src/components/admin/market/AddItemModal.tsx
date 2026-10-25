"use client";

/**
 * **إضافةُ صنفٍ من السوق** — نُقلت من صفحة القسم إلى تبويب الأصناف (قرارُ المالك ٢٠٢٦-١٠-٢٥:
 * «بالأصناف نحطّ خيار أضف صنفاً ونحذفه من القسم»). **وقسمُ السوق يُختار هنا** بعد أن كان
 * مثبَّتاً بالصفحة التي فُتحت منها.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, storeLabel } from "@rahalgo/i18n";
import { Input, Select, Modal, LoadingState, FormActions } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import ImageUpload from "@/components/admin/ImageUpload";

const MARKET_MEDIA = "/api/v1/admin/market/media";
const m = getMessages(defaultLocale);
const S = m.admin.sections;
const I = m.admin.market.items;

interface MerchantRow {
  id: string;
  name: string;
  /** **رقمُه** — متجران باسمٍ واحدٍ في القائمة يُفرَّقان به. */
  number: number;
}
interface MenuSectionRow {
  id: string;
  name: string;
}

/**
 * **إضافةُ صنفٍ من بابِ السوق — والمتجرُ يُسأل عنه أوّلاً.**
 *
 * # ولماذا قسمُ المتجر أيضاً
 *
 * الصنفُ يعيش في قائمة متجره (`section_id`) **ويُعرض في قسم السوق**
 * (`platform_section_id`) — **موضعان لا واحد**: الأوّلُ ترتيبُ المطبخ يراه
 * صاحبُه، والثاني بابُ الزبون. **ولا يُنشأ صنفٌ بلا موضعٍ في قائمة صاحبه**:
 * يصير يتيماً لا يجده من يملكه.
 *
 * **وقسمُ السوق مثبَّتٌ هنا** — هو الصفحةُ التي فُتح منها، فلا يُسأل عنه.
 */
export default function AddItemModal({
  sections,
  onClose,
  onSaved,
}: {
  sections: { id: string; name: string }[];
  onClose: () => void;
  onSaved: () => void;
}) {
  const [merchants, setMerchants] = useState<MerchantRow[]>([]);
  const [sectionID, setSectionID] = useState("");
  const [merchantID, setMerchantID] = useState("");
  const [menuSections, setMenuSections] = useState<MenuSectionRow[] | null>(null);
  const [menuSectionID, setMenuSectionID] = useState("");
  const [name, setName] = useState("");
  const [price, setPrice] = useState("");
  const [imageID, setImageID] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    // **وقائمةُ الاختيار كاملةٌ** — (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦).
    //
    // **كانت تُنادى بلا حدٍّ فيردّ المحرّكُ عشرين** — **وهي قائمةُ اختيار**:
    // فمن عنده خمسةٌ وعشرون متجراً **لا يستطيع أن يضيف صنفاً لخمسةٍ منها**،
    // **ولا شيءَ يقول إنّها قُصّت** — إنّما لا يجد المتجرَ فيظنّه غيرَ مسجَّل.
    api<{ merchants: MerchantRow[] }>("/api/v1/admin/merchants?per_page=100")
      .then((r) => setMerchants(r.merchants ?? []))
      .catch((err) => setError(errorText(err)));
  }, []);

  // **وقائمةُ المتجر تُقرأ حين يُختار** — لا قبله: أقسامُ متجرٍ لم يُختر
  // بعدُ لا معنى لها، **وقراءةُ قوائم كلّ المتاجر سلفاً حملٌ بلا سبب.**
  useEffect(() => {
    if (!merchantID) {
      setMenuSections(null);
      setMenuSectionID("");
      return;
    }
    setMenuSections(null);
    api<MenuSectionRow[]>(`/api/v1/admin/merchants/${merchantID}/menu`)
      .then((r) => {
        const list = r ?? [];
        setMenuSections(list);
        setMenuSectionID(list[0]?.id ?? "");
      })
      .catch((err) => setError(errorText(err)));
  }, [merchantID]);

  async function submit() {
    if (!sectionID) return setError(I.pickSection);
    if (!merchantID) return setError(S.pickMerchant);
    if (!menuSectionID) return setError(S.pickMenuSection);
    if (!name.trim()) return setError(S.itemNameRequired);
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/market/stores/${merchantID}/items`, {
        method: "POST",
        body: JSON.stringify({
          section_id: menuSectionID,
          name: name.trim(),
          price: Number(price) || 0,
          platform_section_id: sectionID,
          ...(imageID !== null ? { image_media_id: imageID } : {}),
        }),
      });
      onSaved();
    } catch (err) {
      const key = err instanceof ApiError ? (err.body.message_key.split(".").pop() ?? "") : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    <Modal open title={I.addItem} onClose={onClose}>
      <div className="space-y-3">
        <Select label={I.marketSection} value={sectionID} onChange={(e) => setSectionID(e.target.value)}>
          <option value="">{I.pickSection}</option>
          {sections.map((x) => (
            <option key={x.id} value={x.id}>
              {x.name}
            </option>
          ))}
        </Select>
        <Select
          label={S.merchant}
          value={merchantID}
          onChange={(e) => setMerchantID(e.target.value)}
        >
          <option value="">{S.pickMerchant}</option>
          {merchants.map((x) => (
            <option key={x.id} value={x.id}>
              {storeLabel(x.name, x.number)}
            </option>
          ))}
        </Select>

        {/* **وقسمُ المطبخ لا يُسأل عنه قبل المتجر** — حقلٌ فارغٌ معطَّلٌ يُقرأ
            عطباً، **وغيابُه يقول «اختر المتجر أوّلاً» بلا كلمة.** */}
        {merchantID &&
          (menuSections === null ? (
            <LoadingState variant="inline" />
          ) : menuSections.length === 0 ? (
            <p className="text-sm text-danger">{S.merchantHasNoSections}</p>
          ) : (
            <Select
              label={S.menuSection}
              value={menuSectionID}
              onChange={(e) => setMenuSectionID(e.target.value)}
            >
              {menuSections.map((x) => (
                <option key={x.id} value={x.id}>
                  {x.name}
                </option>
              ))}
            </Select>
          ))}

        <ImageUpload path={MARKET_MEDIA} kind="menu_item" label={S.itemImage} onChange={setImageID} />
        <Input label={S.itemName} value={name} onChange={(e) => setName(e.target.value)} />
        {/* **سعرُ الشراء لا سعرُ البيع** — ما يقبضه المتجر، **والمنصةُ تحسب
            الهامشَ فوقه.** واللفظُ يقول أيَّهما، فـ«السعر» وحدَها تحتمل الاثنين. */}
        <Input
          label={S.itemPrice}
          type="number"
          value={price}
          onChange={(e) => setPrice(e.target.value)}
        />
        {error && <p className="text-sm text-danger">{error}</p>}
        <FormActions onSave={submit} onCancel={onClose} busy={busy} />
      </div>
    </Modal>
  );
}
