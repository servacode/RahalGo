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

import { useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
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
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store";

const m = getMessages(defaultLocale);
const S = m.merchant.storeSettings;

export default function MerchantSettingsPage() {
  const { store, refresh } = useStore();
  const [prep, setPrep] = useState("");
  const [minOrder, setMinOrder] = useState("");
  const [busy, setBusy] = useState(false);
  const [saved, setSaved] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!store) return;
    setPrep(String(store.default_prep_minutes ?? ""));
    setMinOrder(String(store.min_order ?? 0));
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
          min_order: Number(minOrder) || 0,
        }),
      });
      setSaved(true);
      await refresh();
    } catch {
      setError(m.errors.internal);
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
          <div>
            <Input
              id="min-order"
              label={`${S.minOrder} (${m.common.currency})`}
              type="number"
              value={minOrder}
              onChange={(e) => {
                setMinOrder(e.target.value);
                setSaved(false);
              }}
            />
            <p className="mt-1 text-xs text-ink-muted">{S.minOrderHint}</p>
          </div>
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
