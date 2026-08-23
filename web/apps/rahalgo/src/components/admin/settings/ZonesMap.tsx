"use client";

/**
 * خريطة مناطق التغطية — نموذج الدوائر: مركز + نصف قطر.
 * النقر يحدد/ينقل المركز أثناء التحرير، والدائرة تُعاين حياً مع تغيير نصف القطر.
 */

import { ZonesMap as MapView } from "@rahalgo/ui/zonesmap";
import type { ZoneShape } from "@rahalgo/ui/zonesmap";
import { themeColor } from "@rahalgo/ui";
import { getMessages, defaultLocale } from "@rahalgo/i18n";

const msg = getMessages(defaultLocale);

/**
 * ألوانُ الخريطة من الرموز المركزية لا مكتوبةً بالحرف.
 *
 * **Leaflet يرسم على canvas** فلا تصله فئاتُ Tailwind — فتُمرَّر إليه قيمٌ
 * صريحة. وكانت مكتوبةً هنا بالحرف، **فبقيت على اللوحة القديمة حين تغيّرت
 * لوحةُ المشروع**: خريطةٌ بلونين وشاشةٌ بلونين آخرين في نافذةٍ واحدة.
 *
 * **وهذا هو الفرقُ بين ثيمٍ مركزيّ وثيمٍ يُقال إنه مركزيّ**: أن يُغيَّر رقمٌ
 * واحد فيتبعه كلُّ شيء — بما فيه ما لا يُرسم بـCSS.
 */
//
// **وقد كان يُقال إنّه مركزيّ وليس كذلك.**
//
// كانت تُقرأ من `tokens.ts` — **ولوحتُه شاخت** حين صار الثيمُ داكناً: بقي
// `primary` فيها أزرقَ بترولياً والشاشةُ صارت سماويّة، **فرسمت الخريطةُ
// مناطقَها بلونٍ لا وجودَ له في النافذة نفسِها** — وهو عينُ ما يحذّر منه
// التعليقُ أعلاه.
//
// **والآن تُقرأ من الثيم لحظةَ الرسم** — يُغيَّر رقمٌ واحدٌ فيتبعه كلُّ شيء.
//
// **ودالّةٌ لا ثابت**: التوكنُ يُقرأ من الوثيقة، **وثابتٌ في أعلى الوحدة
// يُحسَب مرّةً قبل أن يُحمَّل الثيم** فيعود الرمادَ الأخير.
const C = () => ({
  primary: themeColor("primary"),
  accent: themeColor("accent"),
  accentDark: themeColor("accent-dark"),
  /** منطقةٌ مُطفأة — محايدُ الحدود نفسه الذي في الرموز */
  muted: themeColor("ink-muted"),
});



export type { ZoneShape };

export default function ZonesMap({
  zones,
  editing,
  draft,
  selectedID,
  onMapClick,
  onZoneClick,
}: {
  zones: ZoneShape[];
  editing: boolean;
  draft: { lat: number; lng: number; radiusM: number } | null;
  selectedID: string | null;
  onMapClick: (lat: number, lng: number) => void;
  onZoneClick: (id: string) => void;
}) {
  const c = C();
  return (
    <MapView
      zones={zones}
      editing={editing}
      draft={draft}
      selectedID={selectedID}
      onMapClick={onMapClick}
      onZoneClick={onZoneClick}
      palette={{
        primary: c.primary,
        accent: c.accent,
        accentDark: c.accentDark,
        muted: c.muted,
      }}
      unavailableLabel={msg.map.unavailable}
    />
  );
}
