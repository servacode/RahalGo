"use client";

/**
 * **خريطةُ المكتب — تُرى ولا تُنقر.**
 *
 * `PickMap` هو ما في المشروع لعرض نقطةٍ على خريطة، **ونقرُه يختار موضعاً** —
 * وهو هنا بلا معنًى: الزبونُ لا يضبط موقعَ المكتب. **فتُبتلع النقرة.**
 *
 * **وزرُّ «تحديد موقعي» يُخفى** (قرارُ المالك ٢٠٢٦-٠٨-٠٩): **خريطةٌ تُقرأ لا
 * تُضبط**، وزرٌّ يطلب إذنَ الموقع فيها يسأل بلا سبب.
 *
 * **ولا تُبنى خريطةُ عرضٍ ثانية**: مكوّنان يرسمان الخريطةَ نفسَها **يفترقان
 * في طبقة البلاط أو في تعامُلهما مع انقطاعها** — وقد بُني `FallbackTileLayer`
 * لهذا بعينه.
 */

import dynamic from "next/dynamic";

const PickMap = dynamic(() => import("@rahalgo/ui/map").then((m) => m.PickMap), {
  ssr: false,
});

export default function ContactMap({ lat, lng }: { lat: number; lng: number }) {
  return (
    <div className="overflow-hidden rounded-control border border-line">
      <PickMap lat={lat} lng={lng} height="h-72" hideLocate onPick={() => undefined} />
    </div>
  );
}
