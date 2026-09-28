package com.rahalgo.driver.replaylab

import com.rahalgo.navigation.GeoPoint
import com.rahalgo.navigation.ManeuverKinds
import com.rahalgo.navigation.NavFix
import com.rahalgo.navigation.NavManeuver
import com.rahalgo.navigation.NavRoute
import com.rahalgo.navigation.ReplayDrive
import com.rahalgo.navigation.RouteReply
import com.rahalgo.navigation.RerouteFailure
import com.rahalgo.navigation.RouteSource
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مختبرُ الإعادة — سيناريوهاتُ الرقّة على طرقاتٍ حقيقيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **الهندسةُ من محرّك التوجيه الحقيقيّ** (OSRM على التجهيز، حُسبت مرّةً
 * وضُمّنت): السائقُ يسير على شوارعِ الرقّة الفعليّة (شارع عدنان المالكي،
 * طريق الكرنيش، شارع المنصور…) لا على خطوطٍ تعبر المباني. **بياناتٌ محلّيّةٌ
 * محضة** تُغذّى محرّكَ الملاحةِ الحقيقيّ — لا طلبَ ولا خادمَ ولا دفتر.
 */

internal data class ReplayLeg(
    val label: String,
    val route: NavRoute,
    val fixes: List<NavFix>,
    val target: GeoPoint,
    /** هندسةُ الالتفاف الحقيقيّة (للسيناريو ذي إعادةِ التوجيه) — أو null. */
    val rerouteGeom: List<GeoPoint>?,
)

internal data class ReplayScenario(
    val id: String,
    val title: String,
    val driverStart: GeoPoint,
    val merchant: GeoPoint,
    val customer: GeoPoint,
    val legs: List<ReplayLeg>,
    val offRoute: Boolean,
)

internal object ReplayScenarios {

    private val DRIVER = GeoPoint(35.9600, 39.0140)
    private val MERCHANT = GeoPoint(35.9520, 39.0095)
    private val CUSTOMER = GeoPoint(35.9450, 39.0200)

    // ── هندسةُ الطرقات الحقيقيّة (OSRM/التجهيز) ──────────────────────────
    // LEG1: السائق ← المتجر (1265م، عبر شارع عدنان المالكي).
    private val LEG1_GEOM = listOf(GeoPoint(35.959999,39.013979), GeoPoint(35.958945,39.013963), GeoPoint(35.958497,39.013772), GeoPoint(35.958308,39.013802), GeoPoint(35.95816,39.013957), GeoPoint(35.95705,39.012864), GeoPoint(35.956379,39.012477), GeoPoint(35.956237,39.012302), GeoPoint(35.95621,39.012096), GeoPoint(35.956102,39.012032), GeoPoint(35.955998,39.010774), GeoPoint(35.955116,39.009359), GeoPoint(35.954953,39.009244), GeoPoint(35.954791,39.009279), GeoPoint(35.954658,39.009033), GeoPoint(35.954426,39.008974), GeoPoint(35.954018,39.00841), GeoPoint(35.953271,39.008303), GeoPoint(35.952145,39.008407), GeoPoint(35.952208,39.009356), GeoPoint(35.951994,39.00937))
    private val LEG1_MAN = listOf(NavManeuver(kind=ManeuverKinds.DEPART, atDistanceM=0.0, streetName=""), NavManeuver(kind=ManeuverKinds.TURN_RIGHT, atDistanceM=212.5, streetName=""), NavManeuver(kind=ManeuverKinds.STRAIGHT, atDistanceM=475.0, streetName=""), NavManeuver(kind=ManeuverKinds.STRAIGHT, atDistanceM=508.4, streetName=""), NavManeuver(kind=ManeuverKinds.STRAIGHT, atDistanceM=823.2, streetName="شارع عدنان المالكي"), NavManeuver(kind=ManeuverKinds.STRAIGHT, atDistanceM=877.4, streetName="شارع عدنان المالكي"), NavManeuver(kind=ManeuverKinds.TURN_LEFT, atDistanceM=1155.2, streetName=""), NavManeuver(kind=ManeuverKinds.TURN_RIGHT, atDistanceM=1241.1, streetName=""), NavManeuver(kind=ManeuverKinds.ARRIVE, atDistanceM=1264.9, streetName=""))

    // LEG1DET: التفافٌ حقيقيٌّ (2988م، عبر طريق الكرنيش وشارع المنصور) — لسيناريو الخروج/الإعادة.
    private val LEG1DET_GEOM = listOf(GeoPoint(35.959999,39.013979), GeoPoint(35.958945,39.013963), GeoPoint(35.958497,39.013772), GeoPoint(35.958308,39.013802), GeoPoint(35.95816,39.013957), GeoPoint(35.95705,39.012864), GeoPoint(35.956379,39.012477), GeoPoint(35.956237,39.012302), GeoPoint(35.956183,39.012066), GeoPoint(35.956064,39.012035), GeoPoint(35.955948,39.012157), GeoPoint(35.955946,39.012285), GeoPoint(35.956045,39.012408), GeoPoint(35.956212,39.013971), GeoPoint(35.956825,39.017781), GeoPoint(35.956981,39.019151), GeoPoint(35.956924,39.019736), GeoPoint(35.957822,39.019991), GeoPoint(35.957711,39.020476), GeoPoint(35.957515,39.020421), GeoPoint(35.956898,39.020247), GeoPoint(35.957037,39.019669), GeoPoint(35.957049,39.018868), GeoPoint(35.956269,39.013855), GeoPoint(35.956193,39.012726), GeoPoint(35.956245,39.012176), GeoPoint(35.956102,39.012032), GeoPoint(35.955998,39.010774), GeoPoint(35.955116,39.009359), GeoPoint(35.954953,39.009244), GeoPoint(35.954791,39.009279), GeoPoint(35.954658,39.009033), GeoPoint(35.954426,39.008974), GeoPoint(35.954116,39.008498), GeoPoint(35.9539,39.008355), GeoPoint(35.953271,39.008303), GeoPoint(35.952145,39.008407), GeoPoint(35.952208,39.009356), GeoPoint(35.951994,39.00937))

    // LEG2: المتجر ← الزبون (2037م، عبر شارع عدنان المالكي وطريق حلب).
    private val LEG2_GEOM = listOf(GeoPoint(35.951994,39.00937), GeoPoint(35.951645,39.009394), GeoPoint(35.951624,39.00844), GeoPoint(35.94906,39.008596), GeoPoint(35.945619,39.008626), GeoPoint(35.945198,39.010595), GeoPoint(35.944871,39.010784), GeoPoint(35.944869,39.012628), GeoPoint(35.94533,39.012776), GeoPoint(35.945212,39.013656), GeoPoint(35.944826,39.015787), GeoPoint(35.943798,39.019186), GeoPoint(35.945223,39.019747), GeoPoint(35.945141,39.020057))
    private val LEG2_MAN = listOf(NavManeuver(kind=ManeuverKinds.DEPART, atDistanceM=0.0, streetName=""), NavManeuver(kind=ManeuverKinds.TURN_RIGHT, atDistanceM=38.8, streetName=""), NavManeuver(kind=ManeuverKinds.TURN_LEFT, atDistanceM=118.5, streetName="شارع عدنان المالكي"), NavManeuver(kind=ManeuverKinds.SLIGHT_LEFT, atDistanceM=791.7, streetName=""), NavManeuver(kind=ManeuverKinds.TURN_LEFT, atDistanceM=1002.8, streetName="طريق حلب"), NavManeuver(kind=ManeuverKinds.TURN_LEFT, atDistanceM=1169.0, streetName=""), NavManeuver(kind=ManeuverKinds.TURN_RIGHT, atDistanceM=1236.4, streetName=""), NavManeuver(kind=ManeuverKinds.TURN_LEFT, atDistanceM=1828.2, streetName=""), NavManeuver(kind=ManeuverKinds.TURN_RIGHT, atDistanceM=2007.3, streetName=""), NavManeuver(kind=ManeuverKinds.ARRIVE, atDistanceM=2036.7, streetName=""))

    private fun legRoute(geom: List<GeoPoint>, maneuvers: List<NavManeuver>) = NavRoute.of(geom, maneuvers)

    private fun leg(label: String, routeGeom: List<GeoPoint>, maneuvers: List<NavManeuver>, fixGeom: List<GeoPoint>, target: GeoPoint, startMs: Long, rerouteGeom: List<GeoPoint>?): ReplayLeg {
        val route = legRoute(routeGeom, maneuvers)
        val fixes = ReplayDrive.fixes(fixGeom, speedMps = 8.3f, stepSec = 1.0, startMs = startMs)
        return ReplayLeg(label, route, fixes, target, rerouteGeom)
    }

    fun normal(): ReplayScenario {
        val leg1 = leg("إلى المتجر", LEG1_GEOM, LEG1_MAN, LEG1_GEOM, MERCHANT, 0L, null)
        val startMs2 = (leg1.fixes.lastOrNull()?.atMs ?: 0L) + 1000L
        val leg2 = leg("إلى الزبون", LEG2_GEOM, LEG2_MAN, LEG2_GEOM, CUSTOMER, startMs2, null)
        return ReplayScenario("normal", "رحلة عاديّة", DRIVER, MERCHANT, CUSTOMER, listOf(leg1, leg2), offRoute = false)
    }

    fun reroute(): ReplayScenario {
        // المسارُ المخطَّطُ مباشرٌ (LEG1)، لكنّ القراءاتِ تسلك التفافاً حقيقيّاً
        // (LEG1DET) — فتخرج عن المسار فيُطلَق كشفُ الخروج وإعادةُ التوجيه الحقيقيّة،
        // ومصدرُ المسار المُصطنَع يعيد التوجيهَ على طرقاتٍ حقيقيّةٍ من LEG1DET.
        val leg1 = leg("إلى المتجر (بانحراف)", LEG1_GEOM, LEG1_MAN, LEG1DET_GEOM, MERCHANT, 0L, LEG1DET_GEOM)
        val startMs2 = (leg1.fixes.lastOrNull()?.atMs ?: 0L) + 1000L
        val leg2 = leg("إلى الزبون", LEG2_GEOM, LEG2_MAN, LEG2_GEOM, CUSTOMER, startMs2, null)
        return ReplayScenario("reroute", "خروجٌ وإعادةُ توجيه", DRIVER, MERCHANT, CUSTOMER, listOf(leg1, leg2), offRoute = true)
    }

    fun all(): List<ReplayScenario> = listOf(normal(), reroute())
}

/**
 * **مصدرُ مسارٍ مُصطنَعٌ محلّيّ** — يعيد التوجيهَ **على طرقاتٍ حقيقيّة**:
 * يقتطع بقيّةَ هندسةِ الالتفاف الحقيقيّة من أقربِ نقطةٍ لموضع السائق إلى الوجهة.
 * **لا خادم، لا شبكة.** ويبدأ من موضع السائق فيقبله حارسُ `installRerouted`.
 */
internal class SyntheticRouteSource(
    private val scope: CoroutineScope,
    private val geomProvider: () -> List<GeoPoint>?,
    private val targetProvider: () -> GeoPoint?,
) : RouteSource {

    private fun metersBetween(a: GeoPoint, b: GeoPoint): Double {
        val r = 6371000.0
        val dLat = Math.toRadians(b.lat - a.lat); val dLng = Math.toRadians(b.lng - a.lng)
        val h = Math.sin(dLat/2)*Math.sin(dLat/2) + Math.cos(Math.toRadians(a.lat))*Math.cos(Math.toRadians(b.lat))*Math.sin(dLng/2)*Math.sin(dLng/2)
        return 2*r*Math.asin(Math.min(1.0, Math.sqrt(h)))
    }

    override fun request(lat: Double, lng: Double, seq: Long, done: (RouteReply) -> Unit) {
        val here = GeoPoint(lat, lng)
        val target = targetProvider()
        val geom = geomProvider()
        scope.launch(Dispatchers.Default) {
            delay(400) // محاكاةُ زمنِ حساب
            val tail: List<GeoPoint> = if (geom != null && geom.size >= 2) {
                // أقربُ رأسٍ لموضع السائق، ثمّ ما بعده حتّى الوجهة (طرقاتٌ حقيقيّة).
                var idx = 0; var best = Double.MAX_VALUE
                for (i in geom.indices) { val d = metersBetween(here, geom[i]); if (d < best) { best = d; idx = i } }
                listOf(here) + geom.subList(idx, geom.size)
            } else if (target != null) {
                listOf(here, GeoPoint((lat + target.lat) / 2, (lng + target.lng) / 2), target)
            } else {
                done(RouteReply.Failed(RerouteFailure.NO_ROUTE)); return@launch
            }
            var total = 0.0
            for (i in 1 until tail.size) total += metersBetween(tail[i - 1], tail[i])
            val maneuvers = listOf(
                NavManeuver(kind = ManeuverKinds.DEPART, atDistanceM = 0.0, streetName = "مسارٌ مُعاد"),
                NavManeuver(kind = ManeuverKinds.ARRIVE, atDistanceM = total.coerceAtLeast(2.0), streetName = "مسارٌ مُعاد"),
            )
            val route = NavRoute.of(tail, maneuvers)
            done(if (route.usable) RouteReply.Ok(route) else RouteReply.Failed(RerouteFailure.INVALID_ROUTE))
        }
    }
}
