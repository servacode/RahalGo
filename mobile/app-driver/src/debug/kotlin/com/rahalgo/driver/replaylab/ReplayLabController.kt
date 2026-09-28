package com.rahalgo.driver.replaylab

import android.content.Context
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.setValue
import com.rahalgo.navigation.GeoPoint
import com.rahalgo.navigation.NavigationSession
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متحكّمُ مختبر الإعادة** — يقود محرّكَ الملاحة الحقيقيَّ بقراءاتٍ مصنوعة
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يعيش خارجَ الشاشة** (يُمسَك في `ViewModel`) فتنجو الرحلةُ من دورانِ
 * الشاشة والخلفيّة/المقدّمة. **العزلُ تامّ**: `NavigationSession` بمصدرِ
 * مسارٍ مُصطنَعٍ محلّيّ ولا صوتَ ولا مُسجِّل — لا طلبَ ولا خادمَ ولا دفتر.
 */
internal class ReplayLabController(context: Context) {

    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main.immediate)

    private var currentTarget: GeoPoint? = null
    private var currentRerouteGeom: List<GeoPoint>? = null

    // **الصوتُ الحقيقيّ** — كما في تطبيق السائق: مخطِّطٌ في الجلسة يولّد
    // التعليمات، ومنسّقٌ ينطقها عبر ناطقِ النظام.
    private val speaker = com.rahalgo.driver.trip.AndroidSpeaker(context)
    private val voice = com.rahalgo.driver.trip.VoiceOrchestrator(speaker)

    /** محرّكُ الملاحةِ الحقيقيّ — توجيهٌ مُصطنَعٌ محلّيٌّ (لا شبكة) + صوتٌ حقيقيّ. */
    val session = NavigationSession(
        context = context,
        source = SyntheticRouteSource(scope, { currentRerouteGeom }, { currentTarget }),
        voice = com.rahalgo.navigation.VoicePlanner(),
    ).also { s ->
        s.onNav = { state ->
            voice.offer(
                state.cues,
                state.progress?.progressM ?: 0.0,
                navigating = s.running,
            )
        }
    }

    var scenario by mutableStateOf(ReplayScenarios.normal()); private set
    var phase by mutableStateOf(Phase.IDLE); private set
    var legLabel by mutableStateOf(""); private set
    var running by mutableStateOf(false); private set
    var paused by mutableStateOf(false); private set
    var speed by mutableStateOf(1); private set

    enum class Phase { IDLE, TO_MERCHANT, TO_CUSTOMER, ARRIVED }

    private var job: Job? = null

    fun scenarios() = ReplayScenarios.all()

    fun selectScenario(s: ReplayScenario) {
        if (!running) scenario = s
    }

    fun changeSpeed(x: Int) {
        if (x == 1 || x == 2 || x == 5) speed = x
    }

    fun start() {
        stop()
        running = true
        paused = false
        session.replayBegin()
        job = scope.launch { drive() }
    }

    fun pause() {
        if (running) paused = true
    }

    fun resume() {
        if (running) paused = false
    }

    fun restart() = start()

    fun stop() {
        job?.cancel()
        job = null
        session.replayEnd()
        session.stop()
        voice.stop()
        running = false
        paused = false
        phase = Phase.IDLE
        legLabel = ""
        currentTarget = null
        currentRerouteGeom = null
    }

    /** يُنادى من `onCleared` — لا أثرَ يبقى (يُغلَق الناطقُ أيضاً). */
    fun dispose() {
        stop()
        speaker.shutdown()
    }

    private suspend fun drive() {
        val legs = scenario.legs
        android.util.Log.i(TAG, "START scenario=${scenario.id} legs=${legs.size} offRoute=${scenario.offRoute}")
        for ((i, leg) in legs.withIndex()) {
            phase = if (i == 0) Phase.TO_MERCHANT else Phase.TO_CUSTOMER
            legLabel = leg.label
            currentTarget = leg.target
            currentRerouteGeom = leg.rerouteGeom
            // **المسارُ يُركَّب أوّلاً ثمّ الوجهة** — والملاحةُ تُسقط عليه.
            session.setRoute(leg.route)
            session.setArrivalTarget(leg.target)
            android.util.Log.i(TAG, "LEG ${i + 1} '${leg.label}' target=${leg.target.lat},${leg.target.lng} fixes=${leg.fixes.size}")
            var arrived = false
            var lastReroute = session.rerouteStatus
            var lastOff = false
            var n = 0
            for (fix in leg.fixes) {
                while (paused) delay(120)
                session.replayFeed(fix)
                val nav = session.nav
                val off = nav?.isOffRoute == true
                val rr = session.rerouteStatus
                if (off != lastOff) {
                    android.util.Log.w(TAG, "OFF_ROUTE=${off} at ${fix.lat},${fix.lng}")
                    lastOff = off
                }
                if (rr != lastReroute) {
                    android.util.Log.w(TAG, "REROUTE_STATUS=$rr routeVertices=${session.route?.geometry?.size}")
                    lastReroute = rr
                }
                if (n % 3 == 0) {
                    android.util.Log.i(
                        TAG,
                        "tick leg=${i + 1} pos=${"%.5f".format(fix.lat)},${"%.5f".format(fix.lng)} " +
                            "remainM=${nav?.remainingM?.toInt()} etaS=${nav?.remainingSec?.toInt()} " +
                            "maneuver=${nav?.currentManeuver?.kind} off=$off reroute=$rr arrived=${nav?.arrivedAtTarget}",
                    )
                }
                n++
                if (nav?.arrivedAtTarget == true) {
                    arrived = true
                    android.util.Log.i(TAG, "ARRIVED leg ${i + 1} '${leg.label}'")
                    break
                }
                delay((1000L / speed).coerceAtLeast(80L)) // 1x=1000 · 2x=500 · 5x=200
            }
            if (arrived) delay(400) // ليستقرّ الوصول قبل تبديل الساق
        }
        phase = Phase.ARRIVED
        running = false
        paused = false
        android.util.Log.i(TAG, "DONE scenario=${scenario.id} phase=ARRIVED")
    }

    private companion object {
        const val TAG = "ReplayLab"
    }
}
