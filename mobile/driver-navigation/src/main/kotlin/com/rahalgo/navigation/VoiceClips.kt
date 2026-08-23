package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **من التعليمة إلى مقاطعِ صوتٍ مسجَّلة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٣: «أريد صوتاً عربيّاً حقيقيّاً، لا صوتي ولا
 *  صوتَ روبوت».)
 *
 * # ولماذا تُشتقّ من الحال لا من النصّ
 *
 * **الطريقُ المعكوسُ ممكن**: يُؤخذ النصُّ المنطوقُ ويُقطَّع ويُبحث عن
 * كلّ قطعةٍ في جدول. **وهو هشٌّ لأنّ النصَّ يُبدَّل لأسبابٍ لا علاقةَ
 * لها بالصوت** — فاصلةٌ تُضاف أو كلمةٌ تُحسَّن، **فينكسر النطقُ كلُّه
 * بصمت** ويسقط إلى المحرّك الآليّ ولا أحدَ يعلم.
 *
 * **وهنا يُقرأ ما تقرؤه `VoicePhrases` نفسُها**: نوعُ التعليمة، ونوعُ
 * المناورة، والمسافةُ المقرَّبة. **فالمصدرُ واحدٌ للاثنين.**
 *
 * # وأسماءُ الشوارع لا تُنطق
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٣.) **وهي الجزءُ الوحيدُ غيرُ المحدود** —
 * وبحذفها تصير القائمةُ خمسةً وأربعين مقطعاً مغلقاً **يصلح لكلّ سوريا
 * لا للرقّة وحدها.**
 *
 * **والاسمُ يبقى مكتوباً على الشاشة** — من أراد أن يتأكّد نظر.
 *
 * # وفارغٌ يعني «قُلها بالمحرّك»
 *
 * **ولا يصمت التطبيقُ أبداً**: ما لا مقطعَ له يُنطق بالمحرّك الآليّ
 * كما كان. **وتعليمةٌ بصوتٍ رديءٍ خيرٌ من صمتٍ في منعطف.**
 */
object VoiceClips {

    /** **مقاطعُ هذه التعليمة بالترتيب** — وفارغةٌ تعني الارتدادَ للمحرّك. */
    fun of(cue: VoiceCue): List<String> = when (cue.kind) {
        CueKind.REROUTE -> listOf("rerouting")
        CueKind.WRONG_WAY -> listOf("wrong_way")
        CueKind.ROUTE_END -> listOf("route_end")
        CueKind.ARRIVE -> listOf("act_arrive")
        CueKind.MANEUVER -> maneuver(cue)
    }

    private fun maneuver(cue: VoiceCue): List<String> {
        val m = cue.maneuver ?: return emptyList()
        val act = action(m) ?: return emptyList()
        val d = cue.roundedDistanceM
        return when {
            // **و«الآن» تلحق الفعلَ ولا تسبقه** — كما في `VoicePhrases`.
            d == null && m.kind != ManeuverKinds.ARRIVE -> listOf(act, "now")
            d == null -> listOf(act)
            else -> {
                // **ومسافةٌ خارجَ الجدول تُسقط التعليمةَ إلى المحرّك
                // كاملةً** — **ولا تُنطق بلا مسافتها**: «انعطف يميناً»
                // بلا «بعد مئتي متر» تُقرأ أمراً فوريّاً، **فينعطف
                // قبل أوانه.**
                val dist = distance(d) ?: return emptyList()
                listOf(dist, act)
            }
        }
    }

    private fun distance(m: Int): String? = when (m) {
        50, 100, 150, 200, 300, 400, 500, 700, 1000 -> "d_$m"
        else -> null
    }

    private fun action(m: NavManeuver): String? = when (m.kind) {
        ManeuverKinds.DEPART -> "act_depart"
        ManeuverKinds.STRAIGHT -> "act_straight"
        ManeuverKinds.TURN_RIGHT -> "act_right"
        ManeuverKinds.TURN_LEFT -> "act_left"
        ManeuverKinds.SLIGHT_RIGHT -> "act_slight_right"
        ManeuverKinds.SLIGHT_LEFT -> "act_slight_left"
        ManeuverKinds.SHARP_RIGHT -> "act_sharp_right"
        ManeuverKinds.SHARP_LEFT -> "act_sharp_left"
        ManeuverKinds.U_TURN -> "act_uturn"
        ManeuverKinds.MERGE -> "act_merge"
        ManeuverKinds.OFF_RAMP -> "act_off_ramp"
        ManeuverKinds.EXIT_ROUNDABOUT -> "act_exit_roundabout"
        ManeuverKinds.ARRIVE -> "act_arrive"
        ManeuverKinds.UNKNOWN -> "act_follow"
        ManeuverKinds.FORK -> fork(m)
        ManeuverKinds.ROUNDABOUT -> roundabout(m)
        else -> "act_follow"
    }

    private fun fork(m: NavManeuver): String = when (m.modifier) {
        "right", "slight right" -> "fork_right"
        "left", "slight left" -> "fork_left"
        else -> "fork_none"
    }

    /**
     * **ومخارجُ الدوّار ستّةٌ لا أكثر.**
     *
     * **ودوّارٌ بسبعة مخارجَ يسقط إلى المحرّك** — ولا يُقال «اخرج من
     * الدوار» مجرَّداً: **من فقد رقمَ مخرجه خرج من الأوّل.**
     */
    private fun roundabout(m: NavManeuver): String? {
        val exit = m.roundaboutExit ?: return "rb_none"
        return if (exit in 1..6) "rb_$exit" else null
    }
}
