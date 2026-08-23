package com.rahalgo.driver.trip

import com.rahalgo.navigation.GeoPoint
import com.rahalgo.navigation.NavManeuver
import com.rahalgo.navigation.NavRoute
import com.rahalgo.shared.model.OrderRoute

/**
 * ══════════════════════════════════════════════════════════════════════
 * **من جواب المحرّك إلى مسارِ الملاحة — الوصلةُ الناقصة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٣أ، أمرُ المالك ٢٠٢٦-٠٨-٢٠: «الوصلةُ الناقصة… جزءٌ رسميٌّ
 *  من المرحلة ٣ ويجب بناؤها».)
 *
 * # وما كان قبلها
 *
 * **المرحلةُ الثانيةُ بنت المحرّكَ ولم تركّبه**: `cumulative_m`
 * و`maneuvers` تصل من الخادم منذ ٢٠٢٦-٠٨-٢٠، **ولا سطرَ في التطبيق
 * يقرؤهما.** و`NavRoute` و`RouteProgress` و`RouteProjector` مبنيّةٌ
 * ومختبَرةٌ **ولا شيءَ يبنيها خارجَ الاختبار.**
 *
 * **فكان تقدّمٌ لا يُحسب وانحرافٌ لا يُقاس** — لا لأنّ المنطقَ ناقص،
 * بل لأنّ السلكَ لم يُوصَل.
 *
 * # ولماذا هنا لا في `driver-navigation`
 *
 * **`driver-navigation` لا تعرف `shared` ولا الشبكة** (قِيس في
 * `build.gradle.kts`) — **وهذا عمداً**: وحدةُ الملاحة تعرف نقاطاً
 * وأزمنة، **ولو عرفت `OrderRoute` لعرفت الطلبَ والخادمَ والسائق.**
 *
 * **وتطبيقُ السائق وحدَه يعرف الاثنين** — فالتحويلُ موضعُه هنا.
 */
object NavRouteMapper {

    /**
     * **يحوّل جوابَ المحرّك إلى مسارِ ملاحة — أو فارغاً.**
     *
     * **وفارغٌ ليس عطباً**: خادمٌ لم يُحدَّث، أو مسارٌ من مخبأٍ قديم،
     * أو محرّكٌ ردّ بلا خطوات. **والخريطةُ ترسم خطَّها كما كانت
     * والملاحةُ تعمل بلا إرشاد.**
     */
    fun toNavRoute(route: OrderRoute?): NavRoute? {
        if (route == null || !route.available || !route.hasNavigation) return null

        val points = route.points.mapNotNull {
            if (it.size >= 2) GeoPoint(it[0], it[1]) else null
        }
        // **والتراكميّةُ تُصدَّق إن طابقت الهندسة** — `hasNavigation`
        // تفحص طولَ `points` الخامّ، **وقد يسقط منه رأسٌ مشوَّهٌ هنا.**
        if (points.size < 2 || route.cumulativeM.size != points.size) return null

        val maneuvers = route.maneuvers.map {
            NavManeuver(
                kind = it.kind,
                modifier = it.modifier,
                atDistanceM = it.atDistanceM,
                atIndex = it.atIndex,
                stepDistanceM = it.stepDistanceM,
                stepDurationS = it.stepDurationS,
                streetName = it.streetName,
                roundaboutExit = it.roundaboutExit,
                roundaboutName = it.roundaboutName,
            )
        }
        // **والتراكميّةُ تُؤخذ من المحرّك لا تُحسب هنا** — **وحسبةٌ
        // ثانيةٌ تفترق عن الأولى يوماً**، فتصير المناوراتُ عند
        // مسافاتٍ لا تطابق الهندسة.
        val nav = NavRoute(points, route.cumulativeM.toDoubleArray(), maneuvers)
        return nav.takeIf { it.usable }
    }
}
