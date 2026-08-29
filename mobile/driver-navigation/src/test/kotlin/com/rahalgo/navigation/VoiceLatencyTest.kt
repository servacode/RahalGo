package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متى تنتهي التعليمةُ — لا متى تبدأ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «الصوتُ غيرُ مطابقٍ للطريق، أحياناً يكون
 *  متأخّراً — خصوصاً إذا كان هناك أكثرُ من انعطافٍ قريباتٍ على بعض».)
 *
 * # ولماذا يُقاس على الحاسوب
 *
 * **الخروجُ إلى الشارع بعد كلّ تعديلٍ لا يُقاس** — والسائقُ لا يحمل
 * ساعةَ توقيت. **وهنا تُعرف الثواني بالضبط**، وتُعاد التجربةُ ألفَ مرّةٍ
 * في ثانية.
 *
 * # والمقياسُ واحد
 *
 * **كم متراً بقي بينك وبين المناورة حين انتهت الجملة؟**
 *
 *	موجبٌ  ⇒ سمعتَها قبل أن تصل  ✔
 *	سالبٌ  ⇒ سمعتَها بعد أن جاوزت  ✘
 */
class VoiceLatencyTest {

    /** **سرعةُ شارعٍ داخليّ** — خمسون كم/س. */
    private val speed = 13.9

    private fun turn(atM: Double, kind: String = ManeuverKinds.TURN_RIGHT) = NavManeuver(
        kind = kind,
        modifier = if (kind == ManeuverKinds.TURN_RIGHT) "right" else "left",
        atDistanceM = atM,
        stepDistanceM = 400.0,
        stepDurationS = 400.0 / 13.9,
    )

    /**
     * **كم مترا يبقى حين تنتهي الجملة.**
     *
     * **والعتبةُ موضعُ البدء** — فيُطرح منها ما يُقطع أثناء الكلام.
     */
    private fun marginM(planner: VoicePlanner, stage: CueStage, m: NavManeuver, thenM: NavManeuver? = null): Double {
        val startM = planner.triggerFor(stage, m, speed)
        val clip = NavClips.maneuver(m, stage, if (stage == CueStage.NOW) null else planner.roundMeters(startM))
        var sec = ClipDurations.seconds(clip)
        if (thenM != null) sec += ClipDurations.seconds(NavClips.then(thenM))
        return startM - sec * speed
    }

    // ══════════════════════════════════════════════════════════════════
    // **العطبُ القديم — يُثبت أوّلاً**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **أين يقع العطبُ بالضبط — تُمشى الشبكةُ كلُّها.**
     *
     * # وما ردّه القياس
     *
     * **صفرُ حالاتٍ متأخّرةٍ من مئةٍ وستٍّ وعشرين** — وأضيقُها بقي فيه
     * ٥٫٣ أمتارٍ من الفراغ (دوّارٌ عند أربعةِ أمتارٍ في الثانية).
     *
     * **فطولُ المقاطع لم يكن سببَ ما بلّغ عنه المالك.** وفرضيّتي كانت
     * خاطئةً وأثبت القياسُ خطأَها — **وحساب المدّة بقي لأنّه يزيد
     * الفراغَ في السرعات الدنيا حيث يقُصُّ الحدُّ الأدنى العتبةَ.**
     *
     * **وهذا الاختبارُ يحرس ألّا ينقلب الرقمُ سالباً** إن طالت جملةٌ
     * يوماً أو ضُيّقت عتبة.
     */
    @Test
    fun `لا جملةَ تنتهي بعد مناورتها حتّى بالسلوك القديم`() {
        val old = VoicePlanner(VoiceTuning(speechAware = false))
        val worst = mutableListOf<Triple<String, Double, Double>>()
        for (v in listOf(2.0, 4.0, 6.0, 8.0, 13.9, 20.0, 25.0)) {
            for (kind in listOf(ManeuverKinds.TURN_RIGHT, ManeuverKinds.ROUNDABOUT, ManeuverKinds.FORK)) {
                for (exit in listOf(null, 11)) {
                    val m = NavManeuver(
                        kind = kind, modifier = "right", atDistanceM = 900.0,
                        roundaboutExit = exit,
                        stepDistanceM = 500.0, stepDurationS = 500.0 / v,
                    )
                    for (stage in listOf(CueStage.PREPARE, CueStage.APPROACH, CueStage.NOW)) {
                        val startM = old.triggerFor(stage, m, v)
                        val clip = NavClips.maneuver(
                            m, stage, if (stage == CueStage.NOW) null else old.roundMeters(startM),
                        )
                        worst += Triple("$kind/$stage/سرعة=$v", startM - ClipDurations.seconds(clip) * v, v)
                    }
                }
            }
        }
        worst.sortBy { it.second }
        println("── أسوأُ عشرةٍ بالسلوك القديم ──")
        for ((name, margin, _) in worst.take(10)) println("  ${"%8.1f".format(margin)}م  $name")
        val negatives = worst.count { it.second < 0 }
        println("سالبٌ في $negatives من ${worst.size}")
        assertEquals("جملٌ تنتهي بعد مناورتها بالسلوك القديم", 0, negatives)
        // **وأضيقُ فراغٍ قِيس ٥٫٣ أمتار** — فإن نزل عن ثلاثةٍ يوماً
        // فقد ضاق شيءٌ، **ويُنظر فيه قبل أن يصير سالباً في الشارع.**
        assertTrue("أضيقُ فراغٍ ${worst.first().second} — ضاق", worst.first().second >= 3.0)
    }

    // ══════════════════════════════════════════════════════════════════
    // **وبعد الإصلاح — لا تعليمةَ تنتهي بعد مناورتها**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `كلُّ مناورةٍ في كلّ طورٍ تنتهي قبل موضعها`() {
        val planner = VoicePlanner()
        val late = mutableListOf<String>()
        val kinds = listOf(
            ManeuverKinds.TURN_RIGHT, ManeuverKinds.TURN_LEFT,
            ManeuverKinds.SLIGHT_RIGHT, ManeuverKinds.SHARP_LEFT,
            ManeuverKinds.U_TURN, ManeuverKinds.MERGE, ManeuverKinds.FORK,
            ManeuverKinds.OFF_RAMP, ManeuverKinds.ROUNDABOUT,
        )
        // **من مشي الرجل إلى الأوتوستراد** — والعتبةُ تتبدّل بالسرعة.
        for (v in listOf(4.0, 8.0, 13.9, 20.0, 25.0)) {
            for (kind in kinds) {
                for (exit in listOf(null, 2, 11)) {
                    val m = NavManeuver(
                        kind = kind,
                        modifier = "right",
                        atDistanceM = 900.0,
                        roundaboutExit = exit,
                        stepDistanceM = 500.0,
                        stepDurationS = 500.0 / v,
                    )
                    for (stage in listOf(CueStage.PREPARE, CueStage.APPROACH, CueStage.NOW)) {
                        val startM = planner.triggerFor(stage, m, v)
                        val clip = NavClips.maneuver(
                            m, stage,
                            if (stage == CueStage.NOW) null else planner.roundMeters(startM),
                        )
                        val margin = startM - ClipDurations.seconds(clip) * v
                        if (margin < 0.0) late += "$kind/$stage/${v}م.ث → ${"%.1f".format(margin)}م"
                    }
                }
            }
        }
        assertEquals("تعليماتٌ تنتهي بعد مناورتها", emptyList<String>(), late)
    }

    /**
     * **والمناورتان المتقاربتان** — وهو ما بلّغ عنه المالك بعينه.
     *
     * **وتُقاسان معاً**: الجملةُ الأولى واللاحقةُ في نفَسٍ واحد،
     * **فالفراغُ يُحسب لهما لا لواحدة.**
     */
    @Test
    fun `المتقاربتان تُقالان كلتاهما قبل الأولى`() {
        val planner = VoicePlanner()
        val first = turn(500.0, ManeuverKinds.TURN_RIGHT)
        val second = turn(560.0, ManeuverKinds.TURN_LEFT)
        for (stage in listOf(CueStage.APPROACH, CueStage.NOW)) {
            val margin = marginM(planner, stage, first, second)
            assertTrue("$stage: انتهت الجملتان بعد المنعطف بـ${-margin}م", margin >= 0.0)
        }
    }

    /**
     * **وللمناورة الثانية لاحقةٌ مسجَّلة** — وإلّا سُمع شطرُ الجملة.
     *
     * **وهذا هو ما كسرتُه يوم ٢٠٢٦-٠٨-٢٤** حين ربطتُ المقاطعَ ونسيتُ
     * الجملةَ المدموجة.
     */
    @Test
    fun `لكلّ مناورةٍ يُدمَج عليها لاحقةٌ معروفة`() {
        val missing = mutableListOf<String>()
        for (kind in ManeuverKinds.all()) {
            for (modifier in listOf(null, "right", "left")) {
                val m = NavManeuver(kind = kind, modifier = modifier, atDistanceM = 100.0, roundaboutExit = 2)
                val t = NavClips.then(m) ?: continue
                // **ولا تُطلب لاحقةٌ لا طولَ لها** — فالغائبُ يأخذ
                // الوسيط، **وهو ما يخفي النقص.**
                if (ClipDurations.ms(t) == ClipDurations.DEFAULT_MS) missing += "$kind/$modifier → $t"
            }
        }
        assertEquals("لواحقُ بلا مقاطع", emptyList<String>(), missing)
    }

    @Test
    fun `وجدولُ الأطوال كاملٌ ومعقول`() {
        assertTrue("جدولٌ صغير: ${ClipDurations.size}", ClipDurations.size >= 560)
        assertTrue("أطولُ مقطعٍ غيرُ معقول", ClipDurations.LONGEST_MS in 3_000..12_000)
        assertTrue("«انعطف يميناً الآن» يجب أن تكون قصيرة", ClipDurations.ms("turn_right_now") < 3_000)
    }
}
