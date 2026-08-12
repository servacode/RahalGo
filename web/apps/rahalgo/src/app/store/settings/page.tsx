"use client";

/**
 * **إعداداتُ المتجر — ما يملكه صاحبُه بيده.**
 *
 * # ثلاثةٌ كانت مبنيّةً بلا زرّ
 *
 * `PUT /stores/{id}/hours` و`PATCH /stores/{id}/settings` **نقطتان تعملان منذ
 * البداية ولا يناديهما أحد.** فصاحبُ المطعم:
 *
 *   - **لا يستطيع أن يقول متى يفتح** — فيبقى متجرُه مفتوحاً أربعاً وعشرين
 *     ساعةً في النظام، **وتصله طلباتٌ في الثالثة فجراً فيُلغيها فتُحسب عليه
 *     مخالفة.**
 *   - **ولا أن يقول كم يستغرق تحضيرُه** — فالوقتُ المتوقَّعُ الذي يراه الزبون
 *     رقمٌ ضبطته المنصةُ لا مطبخُه.
 *   - **ولا أن يضع حدّاً أدنى للطلب** — فيقبل طلباً بعشرة آلافٍ يخسر فيه.
 *
 * **وكلُّها ضبطٌ يعرفه هو ولا تعرفه المنصة.** ومن اتّصل ليُضبَط له مرّةً لا
 * يتّصل في الثانية.
 *
 * # والإغلاقُ الطارئ معها
 *
 * كان في الشريط العلويّ وحدَه — **زرٌّ يُضغط في لحظة انقطاع كهرباء.** وموضعُه
 * هنا أيضاً: **من يفتح إعداداتِ دوامه هو من يفكّر في إغلاقه.**
 */

import dynamic from "next/dynamic";
import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  PageContainer,
  PageHeader,
  LoadingState,
  FormSection,
  StoreHours,
  Button,
  Input,
  IconSettings,
  IconDate,
  IconBalance,
  IconLocation,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store";

/**
 * **الخريطةُ لا تُرسم على الخادم** — `leaflet` يلمس `window` عند التحميل،
 * **ومن رسمها في التوليد المسبق أسقط الصفحة كلَّها.**
 */
const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), {
  ssr: false,
});

const m = getMessages(defaultLocale);
const S = m.merchant.storeSettings;

export default function MerchantSettingsPage() {
  const { store, refresh } = useStore();
  const [prep, setPrep] = useState("");
  const [address, setAddress] = useState("");
  const [coords, setCoords] = useState<{ lat: number; lng: number } | null>(null);
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!store) return;
    setPrep(String(store.default_prep_minutes ?? ""));
    setAddress(store.address_text ?? "");
    setCoords(
      store.lat != null && store.lng != null ? { lat: store.lat, lng: store.lng } : null,
    );
  }, [store]);

  if (!store) return <LoadingState />;

  async function saveSettings() {
    if (!store) return;
    setBusy(true);
    setError("");
    setSaved(false);
    try {
      await api(`/api/v1/merchant/stores/${store.id}/settings`, {
        method: "PATCH",
        body: JSON.stringify({
          default_prep_minutes: Number(prep) || 0,
          address_text: address,
          // **والنقطةُ تُرسَل كاملةً أو لا تُرسَل** — نصفُها ينقل المتجرَ
          // إلى خطِّ الاستواء، والمحرّكُ يرفض النصفَ صراحةً.
          ...(coords ? { lat: coords.lat, lng: coords.lng } : {}),
        }),
      });
      setSaved(true);
      await refresh();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <PageContainer>
      <PageHeader icon={IconSettings} title={S.title} subtitle={store.name} />

      <FormSection title={m.admin.hours.title} icon={<IconDate />}>
        <p className="mb-3 text-xs text-ink-muted">{S.hoursHint}</p>
        <StoreHours
          api={api}
          path={`/api/v1/merchant/stores/${store.id}/hours`}
          emergency={{
            value: store.emergency_closed ?? false,
            save: async (v) => {
              await api(`/api/v1/merchant/stores/${store.id}/emergency`, {
                method: "POST",
                body: JSON.stringify({ closed: v }),
              });
              await refresh();
            },
          }}
        />
      </FormSection>

      {/* ══════════════════════════════════════════════════════════════
          **موضعُ المتجر — وعليه يقوم كلُّ شيء**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-١٢: «يجب إضافة عنوان المتجر بالإعدادات لأنّه
           غير موجود بالويب»، و«نسوي دبّوس المتجر إلزامي».)

          **والدبّوسُ ليس زينةً على خريطة**: منه تُحسب المسافةُ التي يقرّر بها
          السائقُ أيَّ طلبٍ يأخذ، **وعليه يقوم توزيعُ «الأقرب»**، وبه يعرف أين
          يقف. **ومتجرٌ بلا دبّوس ظهر للسائق بلا مسافة** — ووقع فعلاً في طلبٍ
          حقيقيّ (٢٠٢٦-٠٨-١٢).

          **والعنوانُ نصّاً معه لا بدلاً منه**: الدبّوسُ يوصله إلى الشارع،
          **والنصُّ يقول له أيُّ بابٍ من أبوابه** — «جانب مغسلة أبو الهيف». */}
      <FormSection title={S.location} icon={<IconLocation />}>
        <div className="space-y-3">
          <Input
            id="address"
            label={S.addressText}
            value={address}
            onChange={(e) => {
              setAddress(e.target.value);
              setSaved(false);
            }}
          />
          <p className="text-xs text-ink-muted">{S.locationHint}</p>
          <PickMap
            lat={coords?.lat ?? null}
            lng={coords?.lng ?? null}
            onPick={(lat, lng) => {
              setCoords({ lat, lng });
              setSaved(false);
            }}
          />
          {!coords && <p className="text-sm text-danger">{S.locationMissing}</p>}
        </div>
      </FormSection>

      <FormSection title={S.operations} icon={<IconBalance />}>
        <div className="space-y-3">
          {/* **ورقمُ التحضير يراه الزبون** — لا يبقى في مطبخٍ لا نراه. */}
          <div>
            <Input
              id="prep"
              label={S.prepMinutes}
              type="number"
              value={prep}
              onChange={(e) => {
                setPrep(e.target.value);
                setSaved(false);
              }}
            />
            <p className="mt-1 text-xs text-ink-muted">{S.prepHint}</p>
          </div>
          {/* ══════════════════════════════════════════════════════════
              **ولا حدَّ أدنى للطلب في هذه المنصّة**
              ══════════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-٠١، وأعاده ٢٠٢٦-٠٨-١٠: «اتّفقنا سابقاً
               لا يوجد حدّ — لأنّ الطلبات مدفوعة ونحن نقبض ثمنَ التوصيل بغضّ
               النظر عن سعر الطلب».)

              **والمحرّكُ ينفّذه منذ ذلك اليوم** — `service.go` يقرأ الحدَّ
              ثمّ يُهمله صراحةً (`_ = minOrder`) بتعليقٍ يشرح لماذا.

              **وبقي الحقلُ في هذه الشاشة وحدَه**: يُكتب ويُحفظ **ولا يُقرأ
              عند الطلب أبداً.** **وحقلٌ يُملأ ولا يفعل شيئاً أسوأُ من حقلٍ
              غائب**: من وضع فيه ٢٠٬٠٠٠ ظنّ أنّه حمى نفسَه من الطلبات
              الصغيرة، **ثمّ يشكو أنّ المنصّة لا تحترم إعداداتِه** — وهو لم
              يكن يعمل يوماً.

              **والعمودُ يبقى في القاعدة** — لا يُحذف لأجل شاشة، **ويُعاد
              إحياؤه بسطرين إن قرّر المالكُ غيرَ ذلك.** */}
          {error && <p className="text-sm text-danger">{error}</p>}
          {saved && <p className="text-sm text-success">{S.saved}</p>}
          <div className="flex justify-end">
            <Button disabled={busy} onClick={() => void saveSettings()}>
              {m.common.save}
            </Button>
          </div>
        </div>
      </FormSection>
    </PageContainer>
  );
}
