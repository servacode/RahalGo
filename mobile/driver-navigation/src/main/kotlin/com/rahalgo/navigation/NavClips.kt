package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **من مناورةٍ إلى اسمِ مقطعٍ مسجَّل — ولا نصَّ في الطريق**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٤: «نعتمد الصوت» — صوتُ `rahalgo2` مسجَّلٌ
 *  مسبقاً بدل النطق الآليّ.)
 *
 * # ولماذا لا يُطابَق بالنصّ
 *
 * **النصُّ يتبدّل ولا يشعر أحد.** من كتب `"انعطف يميناً"` في مكانين
 * ثمّ عدّل أحدَهما **خرس الصوتُ عند ذلك المنعطف وحدَه** — ولا اختبارَ
 * يمسك ذلك، ولا سجلَّ يشتكي. **والمطابقةُ بالنوع والمعدِّل تسقط عند
 * الترجمة لا عند القيادة.**
 *
 * # وأسماءُ الشوارع لا تُقال
 *
 * **قرارُ المالك ٢٠٢٦-٠٨-٢٤**: تُحذف من الصوت وتبقى على الشاشة.
 * **ولا يمكن تسجيلُ اسمٍ لا نعرفه** — و«عند دوّار النعيم» تحتاج مقطعاً
 * لكلّ دوّارٍ في سوريا.
 *
 * **والقياسُ خفّف الثمن**: من ستّةَ عشرَ اسماً فريداً في مُدوَّنة الرقّة
 * **تسعةٌ لاتينيّةٌ يرفضها [StreetName] أصلاً**، وسبعٌ عربيّةٌ **خمسٌ
 * منها أسماءُ دوّارات** — والدوّارُ يُقال بمخرجه وهو الذي يهمّ السائق.
 *
 * # وما لا مقطعَ له
 *
 * **يُردّ [FOLLOW_ROUTE]** — «تابع المسار» صحيحةٌ دائماً. **ولا يُردّ
 * فارغٌ**: صمتٌ عند مناورةٍ أسوأُ من جملةٍ عامّة.
 */
object NavClips {

    /** **مَخرجُ كلّ ما لا نعرفه** — وله مقطعٌ في الحزمة. */
    const val FOLLOW_ROUTE = "follow_route"

    // ══════════════════════════════════════════════════════════════════
    // **جملُ الأحداث** — أسماؤها كما في `corpus.ts` حرفاً بحرف
    // ══════════════════════════════════════════════════════════════════

    const val REROUTING = "recalculating_route"
    const val REROUTE_FAILED = "reroute_failed"
    const val WRONG_WAY = "wrong_way"
    const val ROUTE_END = "route_end"
    const val ARRIVED_PICKUP = "arrived_pickup"
    const val ARRIVED_DROPOFF = "arrived_dropoff"

    fun arrival(target: TripTarget): String = when (target) {
        TripTarget.PICKUP -> ARRIVED_PICKUP
        TripTarget.DROPOFF -> ARRIVED_DROPOFF
    }

    /**
     * **مسافاتُ الإعلان المسجَّلة** — ولا مقطعَ لسواها.
     *
     * **وهي عينُ `ANNOUNCE_DISTANCES` في `corpus.ts`.** وحارسٌ
     * (`NavClipsTest`) يقارنها بما تردّه [VoicePlanner.roundMeters]،
     * **فمن أضاف عتبةً هناك ونسي هنا أسقط الاختبار قبل أن يُسقط الصوت.**
     */
    val DISTANCES: Set<Int> = setOf(50, 100, 150, 200, 250, 300, 400, 500, 700, 1000, 1500, 2000)

    /** **وأقصى مخرجٍ مسجَّل** — من الأوّل إلى الثاني عشر. */
    const val MAX_ROUNDABOUT_EXIT = 12

    /**
     * **ما لا تسبقه مسافةٌ أبداً.**
     *
     * **و«بعد ثلاثمئة متر، ابدأ السير» لا يقولها أحد**: الانطلاقُ عند
     * القدم لا بعد ثلاثمئة متر. **و«بعد ثلاثمئة متر، أنت تقترب من
     * وجهتك» مثلُها** — الاقترابُ وصفٌ لا موعد.
     *
     * (أمسكها `NavClipsCoverageTest` ٢٠٢٦-٠٨-٢٤ قبل أن تُبنى نسخة.)
     */
    private val NO_DISTANCE: Set<String> = setOf(FOLLOW_ROUTE, "depart", "approaching_destination")

    /**
     * **وما لا تلحقه «الآن».**
     *
     * **و«وجهتك على اليمين الآن» عربيّةٌ عرجاء** — والوجهةُ على اليمين
     * سواءٌ أكان الآن أم بعد قليل.
     */
    private val NO_NOW: Set<String> = NO_DISTANCE + setOf("destination_right", "destination_left")

    /**
     * ══════════════════════════════════════════════════════════════════
     * **أصلُ المقطع للمناورة — بلا مسافةٍ ولا «الآن»**
     * ══════════════════════════════════════════════════════════════════
     *
     * **والمعدِّلُ قد يغيب**: OSRM يعطي `fork` بلا يمينٍ ولا يسار.
     * **ومن اخترع له جهةً أضلَّ سائقاً** — فيُردّ الأصلُ العامّ.
     */
    fun stem(m: NavManeuver): String = when (m.kind) {
        ManeuverKinds.DEPART -> "depart"
        ManeuverKinds.STRAIGHT -> "continue_straight"
        ManeuverKinds.TURN_RIGHT -> "turn_right"
        ManeuverKinds.TURN_LEFT -> "turn_left"
        ManeuverKinds.SLIGHT_RIGHT -> "slight_right"
        ManeuverKinds.SLIGHT_LEFT -> "slight_left"
        ManeuverKinds.SHARP_RIGHT -> "sharp_right"
        ManeuverKinds.SHARP_LEFT -> "sharp_left"
        ManeuverKinds.U_TURN -> "uturn"
        ManeuverKinds.MERGE -> sided(m.modifier, "merge_right", "merge_left", "merge")
        ManeuverKinds.FORK -> sided(m.modifier, "fork_right", "fork_left", "fork")
        ManeuverKinds.OFF_RAMP -> sided(m.modifier, "ramp_right", "ramp_left", "ramp")
        ManeuverKinds.ROUNDABOUT -> roundabout(m)
        ManeuverKinds.EXIT_ROUNDABOUT -> "exit_roundabout"
        // **والوصولُ بجهةٍ يُقال بها** — «وجهتك على اليمين» أنفعُ من
        // «لقد وصلت»، **والسائقُ يبحث بعينه لا بعدّادِه.**
        ManeuverKinds.ARRIVE -> sided(m.modifier, "destination_right", "destination_left", "approaching_destination")
        else -> FOLLOW_ROUTE
    }

    private fun sided(modifier: String?, right: String, left: String, neither: String): String =
        when (modifier) {
            "right", "slight right", "sharp right" -> right
            "left", "slight left", "sharp left" -> left
            else -> neither
        }

    private fun roundabout(m: NavManeuver): String {
        val exit = m.roundaboutExit ?: return "roundabout_continue"
        if (exit < 1 || exit > MAX_ROUNDABOUT_EXIT) return "roundabout_continue"
        return "roundabout_exit_$exit"
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **المقطعُ بمسافته — أو بلاها إن لم تُسجَّل**
     * ══════════════════════════════════════════════════════════════════
     *
     * **ومسافةٌ غيرُ مسجَّلةٍ لا تُسقط التعليمة**: تُقال المناورةُ بلا
     * مسافة. **والسائقُ يسمع «انعطف يميناً» فينظر إلى الشاشة** —
     * **وذاك خيرٌ من صمت.**
     */
    fun withDistance(stem: String, meters: Int?): String {
        if (stem in NO_DISTANCE) return stem
        if (meters == null || meters !in DISTANCES) return stem
        return "${stem}_in_${meters}m"
    }

    /**
     * **صيغةُ التنفيذ** — «انعطف يميناً **الآن**».
     *
     * **ولا «الآن» للوصول**: «وجهتك على اليمين الآن» عربيّةٌ عرجاء،
     * ولا مقطعَ لها في الحزمة.
     */
    fun now(stem: String): String = if (stem in NO_NOW) stem else "${stem}_now"

    /**
     * ══════════════════════════════════════════════════════════════════
     * **المقطعُ الكاملُ لمناورةٍ في طور**
     * ══════════════════════════════════════════════════════════════════
     *
     *	PREPARE · APPROACH  →  `turn_right_in_300m`
     *	NOW                 →  `turn_right_now`
     */
    fun maneuver(m: NavManeuver, stage: CueStage, meters: Int?): String {
        val base = stem(m)
        if (base == FOLLOW_ROUTE) return FOLLOW_ROUTE
        return if (stage == CueStage.NOW) now(base) else withDistance(base, meters)
    }

    /**
     * ══════════════════════════════════════════════════════════════════
     * **لاحقةُ المناورة الثانية المتقاربة**
     * ══════════════════════════════════════════════════════════════════
     *
     * «انعطف يميناً» **ثمّ** «ثمّ انعطف يساراً مباشرةً».
     *
     * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤، وهو سلوكُ غوغل ماب نفسُه.)
     *
     * **ولا لاحقةَ لما لا نعرفه** — فيُردّ فارغٌ ولا يُقال شيءٌ ثانٍ،
     * **و«تابع المسار» بعد «انعطف يميناً» تشويشٌ لا إرشاد.**
     */
    fun then(m: NavManeuver): String? {
        val base = stem(m)
        if (base == FOLLOW_ROUTE || base in NO_DISTANCE) return null
        return "then_$base"
    }
}
