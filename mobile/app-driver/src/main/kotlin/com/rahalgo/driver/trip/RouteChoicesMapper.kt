package com.rahalgo.driver.trip

import com.rahalgo.navigation.GeoPoint
import com.rahalgo.navigation.NavManeuver
import com.rahalgo.navigation.NavRoute
import com.rahalgo.navigation.RouteChoices
import com.rahalgo.navigation.RouteOption
import com.rahalgo.navigation.RouteTarget
import com.rahalgo.shared.model.OrderRoute
import com.rahalgo.shared.model.RouteAlternative

/**
 * ══════════════════════════════════════════════════════════════════
 * **من عقد الخادم إلى خيارات الملاحة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٧، البند ٢٥ من التحليل.)
 *
 * **أمرُ المالك**: «Android لا يجب أن يعرف OSRM JSON. Backend يظل
 * يحول OSRM response إلى RahalGo contract».
 *
 * **ولا كلمةَ من OSRM هنا** — الحقولُ كلُّها مفاهيمُنا.
 *
 * # **والبديلُ الفاسدُ لا يُسقط المجموعة**
 *
 * (البند ٣٣ من التحليل: «ولا Alternative فاسدة تسقط RouteSet كلها إذا
 * Primary سليمة».)
 *
 * **فكلُّ بديلٍ يُحوَّل على حدة**، **وما لم يصلح يُترك** — والموصى به
 * يبقى.
 */
object RouteChoicesMapper {

    /**
     * **يبني الخيارات** — أو `null` إن لم يصلح الموصى به نفسُه.
     *
     * @param setId **معرِّفُ الجلبة** — يُولَّد عند الجلب.
     * @param generation **جيلُ الملاحة وقتَ الطلب** (البند ٢٠).
     */
    fun toChoices(
        route: OrderRoute?,
        setId: String,
        generation: Long,
        originLat: Double,
        originLng: Double,
    ): RouteChoices? {
        val nav = NavRouteMapper.toNavRoute(route) ?: return null
        val r = route ?: return null

        val recommended = RouteOption(
            // **ومعرِّفٌ من الخادم أو من البصمة** — فعقدٌ قديمٌ بلا
            // `route_id` **لا يُسقط شيئاً**، والهويّةُ تُشتقّ محلّيّاً.
            routeId = r.routeId.ifBlank { localId(nav) },
            route = nav,
            engineDurationS = if (r.engineDurationS >= 0) r.engineDurationS else r.durationS,
            distanceM = r.distanceM,
        )

        val alternatives = r.alternatives.mapNotNull { toOption(it) }

        return RouteChoices(
            setId = setId,
            generation = generation,
            target = RouteTarget.of(r.target),
            originLat = originLat,
            originLng = originLng,
            recommended = recommended,
            alternatives = alternatives,
        )
    }

    private fun toOption(a: RouteAlternative): RouteOption? {
        if (!a.hasNavigation) return null

        val points = a.points.mapNotNull {
            if (it.size >= 2) GeoPoint(it[0], it[1]) else null
        }
        // **والتراكميّةُ تُصدَّق إن طابقت الهندسة** — كما في المرحلة ٢.
        if (points.size < 2 || a.cumulativeM.size != points.size) return null

        val maneuvers = a.maneuvers.map {
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

        val nav = NavRoute(points, a.cumulativeM.toDoubleArray(), maneuvers)
        if (!nav.usable) return null

        return RouteOption(
            routeId = a.routeId.ifBlank { localId(nav) },
            route = nav,
            engineDurationS = a.engineDurationS,
            distanceM = a.distanceM,
            deltaDistanceM = a.deltaDistanceM,
            deltaDurationS = a.deltaDurationS,
            sharedRatio = a.sharedRatio,
            decisionDivergenceM = a.decisionDivergenceM,
            firstDivergenceM = a.firstDivergenceM,
        )
    }

    /**
     * **معرِّفٌ محلّيٌّ حين لا يرسله الخادم.**
     *
     * **ويُشتقّ من البصمة** — وهي هويّةٌ لا تشابه (البند ٩).
     */
    private fun localId(route: NavRoute): String =
        java.lang.Long.toHexString(com.rahalgo.navigation.RouteFingerprint.of(route))
}
