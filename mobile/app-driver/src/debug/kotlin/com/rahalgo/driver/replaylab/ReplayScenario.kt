package com.rahalgo.driver.replaylab

import com.rahalgo.navigation.GeoPoint
import com.rahalgo.navigation.ManeuverKinds
import com.rahalgo.navigation.NavFix
import com.rahalgo.navigation.NavManeuver
import com.rahalgo.navigation.NavRoute
import com.rahalgo.navigation.ReplayDrive
import com.rahalgo.navigation.RouteReply
import com.rahalgo.navigation.RouteSource
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مختبرُ الإعادة — سيناريوهاتُ الرقّة** (بناءُ التطوير وحدَه)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **بياناتٌ محلّيّةٌ محضة**: نقاطُ الرقّة، وساقان (سائق←متجر←زبون)، وقراءاتٌ
 * مصنوعةٌ تُغذّى محرّكَ الملاحة الحقيقيَّ عبر `NavigationSession.replayFeed`.
 * **لا طلبَ ولا خادمَ ولا دفتر ولا دفعة** — عزلٌ تامّ.
 */

/** ساقٌ واحدة: مسارُها الحقيقيّ (للإسقاط وكشف الخروج)، وقراءاتُها، ووجهتُها. */
internal data class ReplayLeg(
    val label: String,
    val route: NavRoute,
    val fixes: List<NavFix>,
    val target: GeoPoint,
)

internal data class ReplayScenario(
    val id: String,
    val title: String,
    val driverStart: GeoPoint,
    val merchant: GeoPoint,
    val customer: GeoPoint,
    val legs: List<ReplayLeg>,
    /** أثناءَ الساقِ الأولى نخرج عن المسار قصيراً لنُطلق إعادةَ التوجيه الحقيقيّة. */
    val offRoute: Boolean,
)

internal object ReplayScenarios {

    // ── نقاطُ الرقّة (وسط المدينة) ───────────────────────────────────────
    private val DRIVER = GeoPoint(35.9600, 39.0140)
    private val MERCHANT = GeoPoint(35.9520, 39.0095) // مطعم بيت الرقّة (واقعيّ)
    private val CUSTOMER = GeoPoint(35.9450, 39.0200)

    // مسارُ السائق ← المتجر (خطٌّ متعدّدُ النقاط بمنعطف).
    private val LEG1_GEOM = listOf(
        DRIVER,
        GeoPoint(35.9578, 39.0132),
        GeoPoint(35.9556, 39.0118),
        GeoPoint(35.9538, 39.0104),
        MERCHANT,
    )

    // مسارُ المتجر ← الزبون.
    private val LEG2_GEOM = listOf(
        MERCHANT,
        GeoPoint(35.9505, 39.0128),
        GeoPoint(35.9484, 39.0158),
        GeoPoint(35.9466, 39.0182),
        CUSTOMER,
    )

    // **خروجٌ عن المسار في الساق الأولى**: نتبع نصفَها ثمّ ننحرف غرباً قصيراً
    // (خارجَ ممرّ المسار) فيُطلق `OffRouteDetector`، ثمّ نعود مستقيمين إلى المتجر
    // (وإعادةُ التوجيه الحقيقيّة تُركّب مساراً جديداً من نقطة الانحراف).
    private val LEG1_DEVIATED_GEOM = listOf(
        DRIVER,
        GeoPoint(35.9578, 39.0132),
        GeoPoint(35.9556, 39.0118),
        GeoPoint(35.9553, 39.0090), // انحرافٌ غرباً — خارج المسار
        GeoPoint(35.9548, 39.0088),
        GeoPoint(35.9538, 39.0092),
        MERCHANT,
    )

    // مسافةٌ بالأمتار بين نقطتين (هافرساين مبسّط) — لا نعتمد على رؤيةِ وحدةٍ أخرى.
    private fun metersBetween(a: GeoPoint, b: GeoPoint): Double {
        val r = 6371000.0
        val dLat = Math.toRadians(b.lat - a.lat)
        val dLng = Math.toRadians(b.lng - a.lng)
        val la1 = Math.toRadians(a.lat); val la2 = Math.toRadians(b.lat)
        val h = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
            Math.cos(la1) * Math.cos(la2) * Math.sin(dLng / 2) * Math.sin(dLng / 2)
        return 2 * r * Math.asin(Math.min(1.0, Math.sqrt(h)))
    }

    private fun polylineLength(geom: List<GeoPoint>): Double {
        var s = 0.0
        for (i in 1 until geom.size) s += metersBetween(geom[i - 1], geom[i])
        return s
    }

    private fun legRoute(geom: List<GeoPoint>, street: String): NavRoute {
        val total = polylineLength(geom).coerceAtLeast(2.0)
        // مناوراتٌ ضمن [0, total] — لا قيمَ لانهائيّة (تُفسد حسابَ التقدّم).
        val maneuvers = listOf(
            NavManeuver(kind = ManeuverKinds.DEPART, atDistanceM = 0.0, streetName = street),
            NavManeuver(kind = ManeuverKinds.TURN_RIGHT, atDistanceM = total * 0.45, streetName = street),
            NavManeuver(kind = ManeuverKinds.ARRIVE, atDistanceM = total, streetName = street),
        )
        return NavRoute.of(geom, maneuvers)
    }

    private fun leg(label: String, cleanGeom: List<GeoPoint>, fixGeom: List<GeoPoint>, target: GeoPoint, startMs: Long, street: String): ReplayLeg {
        // **المسارُ نظيفٌ دائماً** (للإسقاط وكشف الخروج)، **والقراءاتُ قد تنحرف**.
        val route = legRoute(cleanGeom, street)
        val fixes = ReplayDrive.fixes(fixGeom, speedMps = 8.3f, stepSec = 1.0, startMs = startMs)
        return ReplayLeg(label = label, route = route, fixes = fixes, target = target)
    }

    fun normal(): ReplayScenario {
        val leg1 = leg("إلى المتجر", LEG1_GEOM, LEG1_GEOM, MERCHANT, 0L, "شارع المتجر")
        val startMs2 = (leg1.fixes.lastOrNull()?.atMs ?: 0L) + 1000L
        val leg2 = leg("إلى الزبون", LEG2_GEOM, LEG2_GEOM, CUSTOMER, startMs2, "شارع الزبون")
        return ReplayScenario("normal", "رحلة عاديّة", DRIVER, MERCHANT, CUSTOMER, listOf(leg1, leg2), offRoute = false)
    }

    fun reroute(): ReplayScenario {
        val leg1 = leg("إلى المتجر (بانحراف)", LEG1_GEOM, LEG1_DEVIATED_GEOM, MERCHANT, 0L, "شارع المتجر")
        val startMs2 = (leg1.fixes.lastOrNull()?.atMs ?: 0L) + 1000L
        val leg2 = leg("إلى الزبون", LEG2_GEOM, LEG2_GEOM, CUSTOMER, startMs2, "شارع الزبون")
        return ReplayScenario("reroute", "خروجٌ وإعادةُ توجيه", DRIVER, MERCHANT, CUSTOMER, listOf(leg1, leg2), offRoute = true)
    }

    fun all(): List<ReplayScenario> = listOf(normal(), reroute())
}

/**
 * **مصدرُ مسارٍ مُصطنَعٌ محلّيّ** — يُركّب مساراً مستقيماً من موضع السائق الحاليّ
 * إلى الوجهة الحاليّة حين يطلب `RerouteEngine` إعادةَ توجيه. **لا خادم، لا شبكة.**
 *
 * **يبدأ من `(lat,lng)` بالضبط** — فحارسُ `installRerouted` (يرفض بُعدَ البداية
 * فوقَ ٦٠م) يقبله.
 */
internal class SyntheticRouteSource(
    private val scope: CoroutineScope,
    private val targetProvider: () -> GeoPoint?,
) : RouteSource {
    override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
        val target = targetProvider() ?: return done(RouteReply.Failed(com.rahalgo.navigation.RerouteFailure.NO_ROUTE))
        scope.launch(Dispatchers.Default) {
            delay(400) // محاكاةُ زمنِ حساب
            val mid = GeoPoint((lat + target.lat) / 2, (lng + target.lng) / 2)
            val geom = listOf(GeoPoint(lat, lng), mid, target)
            val maneuvers = listOf(
                NavManeuver(kind = ManeuverKinds.DEPART, atDistanceM = 0.0, streetName = "مسارٌ مُعاد"),
                NavManeuver(kind = ManeuverKinds.ARRIVE, atDistanceM = Double.MAX_VALUE, streetName = "مسارٌ مُعاد"),
            )
            val route = NavRoute.of(geom, maneuvers)
            done(if (route.usable) RouteReply.Ok(route) else RouteReply.Failed(com.rahalgo.navigation.RerouteFailure.INVALID_ROUTE))
        }
    }
}
