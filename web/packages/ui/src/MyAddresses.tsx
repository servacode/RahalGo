"use client";

/**
 * **«عناويني» — قسمٌ مركزيٌّ واحدٌ لكلّ التطبيقات.**
 *
 * # لماذا هنا لا في كلّ تطبيق
 *
 * كان مكتوباً في صفحة حساب الزبون وحدَها: **لاقطُ الخريطة، وحقلُ النصّ،
 * والعنوانُ المعنوَن، وتحميلُ الخريطة الديناميكيّ** — أربعون سطراً. **ونسخُها
 * إلى لوحة السائق يعني نسختين تفترقان يوماً**: يُصلَح مركزُ الخريطة في إحداهما
 * ويبقى الآخر على إحداثيةٍ قديمة، **ولا يلاحظ أحد.**
 *
 * وهي عائلةُ الخلل التي تكرّرت في هذه الجولة أربع مرّات: أنواعُ الوسائط · سقفُ
 * النقد · شرطُ الساعات · وحسابُ الوقت المتوقَّع. **قاعدةٌ مكتوبةٌ مرّتين تفترق
 * بلا صوت.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «بند عناويني لازم يطبَّق بشكل مركزي».)
 *
 * # والخريطةُ تُحمَّل عند الحاجة
 *
 * مكتبتُها ثقيلة، **وصفحةُ حسابٍ تُفتح لتغيير كلمةِ مرورٍ لا يجب أن تجرّها.**
 * فتُحمَّل ديناميكياً هنا مرّةً واحدة، **بدل أن يتذكّر كلُّ تطبيقٍ أن يفعل.**
 *
 * # ولماذا يحتاجها السائقُ والمندوب
 *
 * **كلُّ من في الميدان زبونٌ أيضاً**: يطلب لبيته ولأهله. **وعنوانٌ يُكتب مرّةً
 * ويُستعمل دائماً** — ومن لا يجد دفترَه في لوحته يكتبه من جديد في كلّ طلب.
 */

import { lazy, Suspense } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { AddressBook } from "./AddressBook";
import { FormSection, Input } from "./components";
import { IconLocation } from "./icons";

const m = getMessages(defaultLocale);

/** نوعُ دالّة النداء — **مكرّرٌ هنا** لأنّ `AddressBook` لا تُصدّره. */
type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;

/**
 * الخريطةُ من مدخلٍ فرعيّ — **كي لا تُجرّ مكتبتُها لكلّ صفحة.**
 *
 * **و`lazy` من React لا `next/dynamic`**: هذه الحزمةُ لا تعتمد على إطارٍ
 * بعينه، **وارتباطُها بـ`next` يمنع استعمالَها في أيّ شيءٍ آخر يوماً.**
 */
const PickMap = lazy(() =>
  import("@rahalgo/ui/map").then((mod) => ({ default: mod.PickMap })),
);

/**
 * لاقطُ العنوان: **خريطةٌ وحقلُ نصٍّ يتبادلان الخدمة.**
 *
 * من يعرف عنوانه لا يجيد الخريطة، ومن يعرف مكانه لا يجيد وصفه. فتحريكُ الدبّوس
 * يملأ النصَّ بالترميز العكسيّ، **والنصُّ يبقى قابلاً للتحرير** لأنّ العنوانَ
 * الرسميّ لا يصف بيتاً في الرقّة: **«خلف الجامع، الطابق الثاني» أدلُّ من أيّ
 * إحداثية.**
 */
function AddressPicker({
  api,
  value,
  onChange,
}: {
  api: ApiFn;
  value: { lat: number; lng: number; address: string } | null;
  onChange: (v: { lat: number; lng: number; address: string }) => void;
}) {
  // **مركزُ الرقّة** — نقطةُ البدء حين لا عنوانَ بعد.
  const lat = value?.lat ?? 35.9528;
  const lng = value?.lng ?? 39.0079;
  return (
    <div className="space-y-2">
      <Suspense
        fallback={<div className="h-64 w-full animate-pulse rounded-card bg-page" />}
      >
        <PickMap
          lat={value ? lat : null}
          lng={value ? lng : null}
          onPick={(la: number, ln: number) => {
            onChange({ lat: la, lng: ln, address: value?.address ?? "" });
            api<{ address: string }>(`/api/v1/geo/reverse?lat=${la}&lng=${ln}`)
              .then((r: { address: string }) =>
                r.address ? onChange({ lat: la, lng: ln, address: r.address }) : undefined,
              )
              .catch(() => undefined);
          }}
        />
      </Suspense>
      <Input
        id="addr-text"
        label={m.site.addresses.addressText}
        required
        value={value?.address ?? ""}
        onChange={(e) => onChange({ lat, lng, address: e.target.value })}
      />
    </div>
  );
}

/** قسمُ «عناويني» جاهزاً — عنوانُه وشرحُه ودفترُه ولاقطُه. */
export function MyAddresses({ api }: { api: ApiFn }) {
  return (
    <FormSection title={m.site.addresses.title} icon={<IconLocation />}>
      <p className="mb-3 text-xs text-ink-muted">{m.site.addresses.hint}</p>
      <AddressBook
        api={api}
        picker={(value, onChange) => (
          <AddressPicker api={api} value={value} onChange={onChange} />
        )}
      />
    </FormSection>
  );
}
