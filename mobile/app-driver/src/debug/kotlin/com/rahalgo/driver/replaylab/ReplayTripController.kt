package com.rahalgo.driver.replaylab

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.driver.trip.AndroidSpeaker
import com.rahalgo.driver.trip.TripState
import com.rahalgo.driver.trip.TripStep
import com.rahalgo.driver.trip.VoiceOrchestrator
import com.rahalgo.navigation.GeoPoint
import com.rahalgo.navigation.NavigationSession
import com.rahalgo.navigation.VoicePlanner
import com.rahalgo.shared.model.DriverOrder
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import org.maplibre.android.geometry.LatLng

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متحكّمُ الرحلة التجريبيّة المواجِهةِ للمالك** — يقود شاشةَ الرحلة الحقيقيّة
 * ══════════════════════════════════════════════════════════════════════
 *
 * **لا واجهةَ ملاحةٍ ثانية**: يبني `TripState` مُصطنَعاً (طلبٌ محلّيٌّ بلا خادم)
 * ويقود `NavigationSession` الحقيقيَّ بقراءاتٍ مصنوعةٍ على طرقاتٍ حقيقيّة، فتعرضه
 * `TripScreen` نفسُها التي يراها السائق: خريطةٌ، لوحاتٌ، RTL، أطوار، صوت،
 * مسافة/زمن، إعادةُ توجيه. **معزولٌ ماليّاً وشبكيّاً** — لا طلبَ ولا دفترَ ولا FCM.
 */
internal class ReplayTripController(context: Context) {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)

    private var currentTarget: GeoPoint? = null
    private var currentRerouteGeom: List<GeoPoint>? = null

    // **سلسلةُ الصوت كما في تطبيق السائق تماماً**: ناطقُ نظامٍ مُهيّأ (`init`)،
    // ثمّ `ClipSpeaker` (مقاطعُ الملاحة المسجّلة، والنطقُ الآليُّ احتياطاً)، ثمّ
    // المنسّق. **بلا `init` وبلا `ClipSpeaker` كان صامتاً.**
    private val speaker = AndroidSpeaker(context).also { it.init() }
    private val clipSpeaker = com.rahalgo.driver.trip.ClipSpeaker(context, speaker)
    val voice = VoiceOrchestrator(clipSpeaker)

    /** محرّكُ الملاحةِ الحقيقيّ + صوتٌ حقيقيّ + مصدرُ إعادةِ توجيهٍ مُصطنَعٍ محلّيّ. */
    val session = NavigationSession(
        context = context,
        source = SyntheticRouteSource(scope, { currentRerouteGeom }, { currentTarget }),
        voice = VoicePlanner(),
    ).also { s ->
        s.onNav = { st ->
            voice.offer(st.cues, st.progress?.progressM ?: 0.0, navigating = s.running)
        }
    }

    // **افتراضاً مشهدُ الخروج/الإعادة** — أغنى تجربةً (طرقٌ حقيقيّة + إعادةُ توجيه).
    var scenario by mutableStateOf(ReplayScenarios.reroute()); private set
    var status by mutableStateOf("assigned"); private set
    var speed by mutableStateOf(1); private set
    var paused by mutableStateOf(false); private set
    var running by mutableStateOf(false); private set

    // **طلبٌ مُصطنَعٌ محلّيّ** — عاديٌّ (لا مخصّص) فلا اتّفاقَ سعر، ولا صورةَ تسليمٍ
    // (`requirePhoto=false`)، ولا نقد. أرقامٌ للعرض فقط، بلا أثرٍ ماليّ.
    private val order = DriverOrder(
        id = "replay-demo",
        number = 9999L,
        status = "assigned",
        kind = "standard",
        merchantName = "متجرُ الاختبار (وضع تجريبيّ)",
        addressText = "الرقّة — عنوانُ زبونِ الاختبار",
        customerName = "زبونُ الاختبار",
        lat = 35.9450, lng = 39.0200, // الزبون (dropoff)
        navLat = 35.9520, navLng = 39.0095, // المتجر (pickup)
        cashDue = 0L,
        deliveryFee = 6500L,
        itemsCount = 2,
    )

    private fun legIndex(): Int = when (status) {
        "assigned", "at_pickup" -> 0
        else -> 1
    }

    /** `TripState` مُصطنَعٌ للطور الحاليّ — تقرؤه `TripScreen` كأيّ طلبٍ حقيقيّ. */
    fun tripState(): TripState {
        val leg = scenario.legs[legIndex().coerceIn(0, scenario.legs.size - 1)]
        return TripState(
            order = order.copy(status = status),
            step = TripStep.of(status),
            driver = LatLng(scenario.driverStart.lat, scenario.driverStart.lng),
            pickup = LatLng(scenario.merchant.lat, scenario.merchant.lng),
            dropoff = LatLng(scenario.customer.lat, scenario.customer.lng),
            routeLine = leg.route.geometry.map { LatLng(it.lat, it.lng) },
            navRoute = leg.route,
            // ══════════════════════════════════════════════════════════
            // **والزمنُ والمسافةُ من محرّك الملاحة نفسِه — لا من اختلاق**
            // ══════════════════════════════════════════════════════════
            //
            // (قِيس في الرحلة المحاكاة ٢٠٢٦-٠٩-٢٩: اللوحُ بقي على «جاري
            //  حساب الطريق…» الرحلةَ كلَّها.)
            //
            // **وكانت هذه الحقولُ تُترك فارغةً** — فتقرأ `TripPanel` أنّ
            // لا مسافةَ معروفةً فتُظهر حالةَ الحساب بحقّ: **فالمِرقابُ لم
            // يعطها، لا أنّ اللوحَ أخطأ.**
            //
            // **وطولُ الساق من المسار المركَّب**، **وما بقي من
            // `RouteProgress` الحقيقيّة** التي تحسبها `NavigationSession`
            // من القراءات المُغذّاة — **فالأرقامُ التي يراها المالكُ هي
            // أرقامُ المحرّك حرفاً بحرف، تنقص كما تنقص في الشارع.**
            routeM = leg.route.totalM,
            routeSec = session.nav?.remainingSec?.takeIf { it >= 0 } ?: -1.0,
            remainingM = session.nav?.remainingM ?: -1.0,
            requirePhoto = false,
        )
    }

    private var job: Job? = null

    fun start() {
        stop()
        running = true
        paused = false
        status = "assigned"
        session.replayBegin()
        job = scope.launch { drive() }
    }

    /** يُنادى من `TripActions.step` — تقدّمُ الطور (وصول/استلام/تسليم). */
    fun onStep(to: String) {
        status = to
        if (to == "delivered") {
            running = false
            paused = false
            job?.cancel()
            job = null
            session.replayEnd()
            voice.stop()
        }
    }

    fun pause() { if (running) paused = true }
    fun resume() { if (running) paused = false }
    fun changeSpeed(x: Int) { if (x in 1..3) speed = x }
    fun restart() = start()

    /** السرعةُ المعروضةُ كم/س — سرعةُ الإعادةِ الأساسُ (~٣٠) مضروبةً بالمضاعِف. */
    fun kmh(): Int = Math.round(8.3f * 3.6f * speed)

    fun stop() {
        job?.cancel()
        job = null
        session.replayEnd()
        session.stop()
        voice.stop()
        running = false
        paused = false
    }

    fun dispose() {
        stop()
        speaker.shutdown()
    }

    private suspend fun drive() {
        var lastLeg = -1
        var idx = 0
        var leg = scenario.legs[0]
        while (running && status != "delivered") {
            val li = legIndex()
            if (li != lastLeg) {
                lastLeg = li
                leg = scenario.legs[li.coerceIn(0, scenario.legs.size - 1)]
                currentTarget = leg.target
                currentRerouteGeom = leg.rerouteGeom
                idx = 0
                // **المسارُ والوجهةُ للساق الحاليّة** — و`TripScreen` تضبطهما أيضاً
                // من الحالة؛ نضبطهما هنا لضمانِ إسقاطِ القراءات على المسار الصحيح.
                session.setRoute(leg.route)
                session.setArrivalTarget(leg.target)
            }
            if (idx < leg.fixes.size) {
                while (paused && running) delay(120)
                if (!running) break
                session.replayFeed(leg.fixes[idx])
                idx++
                delay((1000L / speed).coerceAtLeast(80L)) // 1x=1000 · 2x=500 · 5x=200
            } else {
                // الساقُ انتهت — يقفُ عند الوجهة حتّى يتقدّم الطورُ (زرُّ السائق/كشفُ الوصول).
                delay(200)
            }
        }
    }
}
