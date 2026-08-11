"use client";

/**
 * خريطة مناطق التغطية — نموذج الدوائر: مركز + نصف قطر.
 * النقر يحدد/ينقل المركز أثناء التحرير، والدائرة تُعاين حياً مع تغيير نصف القطر.
 */

import { MapContainer, Circle, CircleMarker, useMapEvents } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import { FallbackTileLayer } from "@rahalgo/ui/map";
import { themeColor } from "@rahalgo/ui";

const RAQQA_CENTER: [number, number] = [35.9528, 39.0079];

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

export interface ZoneShape {
  id: string;
  name: string;
  lat: number;
  lng: number;
  radius_m: number;
  delivery_fee: number;
  active: boolean;
}

function ClickCapture({ onClick }: { onClick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(e) {
      onClick(e.latlng.lat, e.latlng.lng);
    },
  });
  return null;
}

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
  // **تُقرأ عند كلّ رسم** — والخريطةُ تُرسم في المتصفّح وحدَه.
  const c = C();
  return (
    <MapContainer
      center={RAQQA_CENTER}
      zoom={13}
      className="h-full w-full"
      style={{ cursor: editing ? "crosshair" : undefined }}
    >
      <FallbackTileLayer />
      {editing && <ClickCapture onClick={onMapClick} />}

      {zones.map((z) => (
        <Circle
          key={z.id}
          center={[z.lat, z.lng]}
          radius={z.radius_m}
          /* ══════════════════════════════════════════════════════════
             **والدائرةُ تُرى على خريطةٍ مزدحمة — لا تُخمَّن**
             ══════════════════════════════════════════════════════════

             (شهده المالك ٢٠٢٦-٠٨-١١: «الدائرةُ الخاصّةُ بالمناطق شفّافةٌ
              جدّاً، لا أستطيع رؤيتَها جيّداً».)

             **كانت تعبئتُها ٠٫١٥** — وخريطةُ الشوارع تحتها ملوّنةٌ مزدحمة:
             مبانٍ ونهرٌ وطرقٌ صفراء. **وخمسةَ عشرَ بالمئة فوق ذلك لا تُقرأ
             حدّاً**، فيُخمَّن مدى التغطية بدل أن يُرى.

             **وحدُّ الدائرة هو ما يقول أين تنتهي التغطية** — فغُلّظ،
             **والتعبئةُ تكفي لتُميَّز الداخلَ من الخارج بلا أن تُخفي الشارع
             الذي يُنظر إليه.** */
          pathOptions={{
            color: z.id === selectedID ? c.accent : z.active ? c.primary : c.muted,
            fillOpacity: z.id === selectedID ? 0.42 : 0.28,
            weight: z.id === selectedID ? 5 : 4,
          }}
          eventHandlers={{ click: () => onZoneClick(z.id) }}
        />
      ))}

      {draft && (
        <>
          <Circle
            center={[draft.lat, draft.lng]}
            radius={draft.radiusM}
            /* **والمسوّدةُ تُرى كأختِها** — وهي ما يُرسم الآن. */
            pathOptions={{ color: c.accentDark, fillColor: c.accent, fillOpacity: 0.35, weight: 4, dashArray: "8" }}
          />
          <CircleMarker
            center={[draft.lat, draft.lng]}
            radius={6}
            pathOptions={{ color: c.accentDark, fillColor: c.accent, fillOpacity: 1 }}
          />
        </>
      )}
    </MapContainer>
  );
}
