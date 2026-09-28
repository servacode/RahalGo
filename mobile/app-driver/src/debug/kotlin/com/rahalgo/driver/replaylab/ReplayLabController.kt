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

    /** محرّكُ الملاحةِ الحقيقيّ — مصدرُ التوجيهِ مُصطنَعٌ محلّيٌّ (لا شبكة). */
    val session = NavigationSession(
        context = context,
        source = SyntheticRouteSource(scope) { currentTarget },
        voice = null,
    )

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
        running = false
        paused = false
        phase = Phase.IDLE
        legLabel = ""
        currentTarget = null
    }

    /** يُنادى من `onCleared` — لا أثرَ يبقى. */
    fun dispose() {
        stop()
    }

    private suspend fun drive() {
        val legs = scenario.legs
        for ((i, leg) in legs.withIndex()) {
            phase = if (i == 0) Phase.TO_MERCHANT else Phase.TO_CUSTOMER
            legLabel = leg.label
            currentTarget = leg.target
            // **المسارُ يُركَّب أوّلاً ثمّ الوجهة** — والملاحةُ تُسقط عليه.
            session.setRoute(leg.route)
            session.setArrivalTarget(leg.target)
            var arrived = false
            for (fix in leg.fixes) {
                while (paused) delay(120)
                session.replayFeed(fix)
                if (session.nav?.arrivedAtTarget == true) {
                    arrived = true
                    break
                }
                // 1x=1000ms · 2x=500ms · 5x=200ms
                delay((1000L / speed).coerceAtLeast(80L))
            }
            // مهلةٌ قصيرةٌ ليستقرّ كشفُ الوصول قبل تبديل الساق
            if (arrived) delay(400)
        }
        phase = Phase.ARRIVED
        running = false
        paused = false
    }
}
