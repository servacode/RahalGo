package com.rahalgo.driver.trip

import com.rahalgo.driver.R
import com.rahalgo.navigation.CueStage
import com.rahalgo.navigation.ManeuverKinds
import com.rahalgo.navigation.NavClips
import com.rahalgo.navigation.NavManeuver
import com.rahalgo.navigation.TripTarget
import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لكلّ ما يمكن أن يُقال مقطعٌ موجود — يُفحص لا يُفترض**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٤: «نعتمد الصوت».)
 *
 * # ولماذا حارسٌ لا مراجعة
 *
 * **[NavClips] تركّب الاسمَ من قِطَع**: أصلٌ + مسافةٌ أو «الآن».
 * **فمن أضاف عتبةَ مسافةٍ في `VoicePlanner` ونسي أن يولّد مقطعَها**
 * بنى تطبيقاً ناجحاً **يخرس عند تلك المسافة وحدَها** — ولا سجلَّ
 * يشتكي ولا شاشةَ تتغيّر. **ويُكتشف في الشارع.**
 *
 * **وهذا الاختبار يمشي الاحتمالاتِ كلَّها** — نوعاً ومعدِّلاً وطوراً
 * ومسافةً — **ويسأل جدولَ الموارد نفسَه.**
 */
class NavClipsCoverageTest {

    /** **أسماءُ ما في `res/raw` فعلاً** — تُقرأ من `R` لا من قائمةٍ نكتبها. */
    private val available: Set<String> = R.raw::class.java.fields.map { it.name }.toSet()

    private val modifiers = listOf(
        null, "right", "left", "slight right", "slight left",
        "sharp right", "sharp left", "straight", "uturn",
    )

    private val kinds = listOf(
        ManeuverKinds.DEPART, ManeuverKinds.ARRIVE, ManeuverKinds.STRAIGHT,
        ManeuverKinds.TURN_LEFT, ManeuverKinds.TURN_RIGHT,
        ManeuverKinds.SLIGHT_LEFT, ManeuverKinds.SLIGHT_RIGHT,
        ManeuverKinds.SHARP_LEFT, ManeuverKinds.SHARP_RIGHT,
        ManeuverKinds.U_TURN, ManeuverKinds.MERGE, ManeuverKinds.FORK,
        ManeuverKinds.OFF_RAMP, ManeuverKinds.ROUNDABOUT,
        ManeuverKinds.EXIT_ROUNDABOUT, ManeuverKinds.UNKNOWN,
        // **ونوعٌ لم يُخترع بعد** — الخادمُ قد يرسله غداً.
        "SOMETHING_NEW",
    )

    @Test
    fun `لكلّ مناورةٍ في كلّ طورٍ وكلّ مسافةٍ مقطعٌ موجود`() {
        val exits = listOf(null, 1, 2, 6, 12, 13, 0, 99)
        val distances = NavClips.DISTANCES.toList() + listOf(null, 137, 5000)
        val missing = sortedSetOf<String>()
        var checked = 0
        for (kind in kinds) {
            for (modifier in modifiers) {
                for (exit in exits) {
                    val m = NavManeuver(
                        kind = kind,
                        modifier = modifier,
                        atDistanceM = 100.0,
                        roundaboutExit = exit,
                    )
                    for (stage in CueStage.entries) {
                        for (d in distances) {
                            val clip = NavClips.maneuver(m, stage, d)
                            checked++
                            if (clip !in available) missing += clip
                        }
                    }
                }
            }
        }
        assertTrue("فُحص $checked احتمالاً — قليل", checked > 5000)
        assertEquals("مقاطعُ ناقصةٌ في res/raw", emptySet<String>(), missing.toSet())
    }

    @Test
    fun `ولجملِ الأحداث كلِّها مقاطع`() {
        val events = listOf(
            NavClips.FOLLOW_ROUTE,
            NavClips.REROUTING,
            NavClips.REROUTE_FAILED,
            NavClips.WRONG_WAY,
            NavClips.ROUTE_END,
            NavClips.arrival(TripTarget.PICKUP),
            NavClips.arrival(TripTarget.DROPOFF),
        )
        assertEquals("مقاطعُ أحداثٍ ناقصة", emptyList<String>(), events.filter { it !in available })
    }

    /**
     * **وعتباتُ `VoicePlanner` كلُّها مسجَّلة.**
     *
     * **وهذا هو الحارسُ الحقيقيّ**: من أضاف `250` إلى `roundMeters`
     * ولم يولّد مقاطعَها **أسقط هذا الاختبار قبل أن يُسقط الصوت.**
     */
    @Test
    fun `كلُّ مسافةٍ يقرّبها المخطّط لها مقاطع`() {
        val planner = com.rahalgo.navigation.VoicePlanner()
        val produced = (10..2500 step 10).map { planner.roundMeters(it.toDouble()) }.toSortedSet()
        val unrecorded = produced.filter { it !in NavClips.DISTANCES }
        assertEquals("عتبةٌ يقرّب إليها المخطّطُ ولا مقطعَ لها", emptyList<Int>(), unrecorded)
    }
}
