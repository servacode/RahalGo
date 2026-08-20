package com.rahalgo.map

import org.maplibre.android.camera.CameraPosition
import org.maplibre.android.camera.CameraUpdateFactory
import org.maplibre.android.geometry.LatLng
import org.maplibre.android.maps.MapLibreMap

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أوليّاتُ الكاميرا — تنفّذ ولا تقرّر**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠: «`map-core` ينفّذ: رسم marker · تحريك
 *  marker · camera primitives»، **و«قرار الملاحة يبقى داخل
 *  `driver-navigation`»**.)
 *
 * # وما لا تعرفه هذه الوحدة
 *
 * **لا طلبَ ولا سائقَ ولا جلسةَ ملاحة.** تتلقّى أرقاماً — موضعٌ ودورانٌ
 * وميلٌ وتقريبٌ ومدّة — وتنفّذها. **ومن أراد أن يعرف لماذا هذا الرقم
 * فليقرأ `NavCamera` في وحدة الملاحة.**
 *
 * **وحارسُ المرحلة ٠ يحرس هذا الحدّ** (`check-map-core-pure.mjs`).
 */
object CameraPrimitives {

    /**
     * **ينقل الكاميرا بانسيابٍ إلى حالٍ كاملة.**
     *
     * **و`easeCamera` لا `moveCamera`**: الثانيةُ تقفز، **والقفزُ في
     * الملاحة يجعل الشاشةَ ترتجّ مع كلّ قراءة.**
     *
     * **والمدّةُ تأتي من المنطق** لتوافق حركةَ الأيقونة: **كاميرا
     * تسبق أيقونتَها تجعلها تبدو منزلقةً إلى الخلف.**
     */
    fun ease(
        map: MapLibreMap,
        lat: Double,
        lng: Double,
        zoom: Double,
        bearingDeg: Float,
        tiltDeg: Float,
        durationMs: Long,
    ) {
        val target = CameraPosition.Builder()
            .target(LatLng(lat, lng))
            .zoom(zoom)
            .bearing(bearingDeg.toDouble())
            .tilt(tiltDeg.toDouble())
            .build()
        // **ومدّةٌ صفرٌ تُقرأ «افعلها الآن»** — و`easeCamera` ترفض
        // الصفرَ في بعض النسخ، فيُضمن حدٌّ أدنى.
        map.easeCamera(
            CameraUpdateFactory.newCameraPosition(target),
            durationMs.coerceAtLeast(MIN_MS).toInt(),
        )
    }

    /** **دورانُ الخريطة الحاليّ** — يقرؤه المنطقُ ليقرّر ألّا يُدير. */
    fun bearingOf(map: MapLibreMap): Float = map.cameraPosition.bearing.toFloat()

    private const val MIN_MS = 1L
}
