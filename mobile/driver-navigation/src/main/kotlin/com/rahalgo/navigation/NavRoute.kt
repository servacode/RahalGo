package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════════
 * **المسارُ كما تفهمه الملاحة — لا كما يرسمه الخطّ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٢، بأمر المالك ٢٠٢٦-٠٨-٢٠.)
 *
 * # ولا يعرف OSRM
 *
 * **`kind` مفاهيمُنا لا كلماتِ محرّك** — والتحويلُ وقع في الخادم.
 * **فيومَ يُبدَّل المحرّكُ لا يُمسّ سطرٌ هنا.**
 *
 * # والمرجعُ مسافةٌ لا فهرس
 *
 * (تصحيحُ المالك: «اعتمد `atDistanceM` كمرجعٍ أساسيّ… ولا يجب أن
 *  يعتمد منطقُ الملاحة على `atIndex` وحدَه».)
 *
 * **والفهرسُ يتبدّل إن بُسِّطت الهندسةُ يوماً** — **والمسافةُ على
 * الطريق لا تتبدّل.**
 */
data class NavRoute(
    val geometry: List<GeoPoint>,
    /** **مسافةُ كلّ رأسٍ من البداية** — بطول `geometry`. */
    val cumulativeM: DoubleArray,
    val maneuvers: List<NavManeuver>,
) {
    /** **طولُ المسار** — آخرُ تراكميّة. */
    val totalM: Double get() = if (cumulativeM.isEmpty()) 0.0 else cumulativeM[cumulativeM.size - 1]

    /** **أصالحٌ للملاحة؟** — رأسان فأكثرُ وتراكميّةٌ مطابقةٌ ومناورة. */
    val usable: Boolean
        get() = geometry.size >= 2 &&
            cumulativeM.size == geometry.size &&
            maneuvers.isNotEmpty()

    // **و`equals` مولَّدةٌ على مصفوفةٍ تقارن المرجعَ لا المحتوى** —
    // فتُكتب بيد. (وهو ما يحذّر منه المترجمُ في `data class`.)
    override fun equals(other: Any?): Boolean {
        if (this === other) return true
        if (other !is NavRoute) return false
        return geometry == other.geometry &&
            cumulativeM.contentEquals(other.cumulativeM) &&
            maneuvers == other.maneuvers
    }

    override fun hashCode(): Int {
        var h = geometry.hashCode()
        h = 31 * h + cumulativeM.contentHashCode()
        h = 31 * h + maneuvers.hashCode()
        return h
    }

    companion object {
        /**
         * **يبني المسارَ ويحسب التراكميّةَ إن غابت.**
         *
         * **والخادمُ يرسلها محسوبةً** — وهذا للرفائد والاختبار.
         */
        fun of(points: List<GeoPoint>, maneuvers: List<NavManeuver>): NavRoute {
            val cum = DoubleArray(points.size)
            for (i in 1 until points.size) {
                cum[i] = cum[i - 1] + GpsQuality.metersBetween(
                    points[i - 1].lat, points[i - 1].lng, points[i].lat, points[i].lng,
                )
            }
            return NavRoute(points, cum, maneuvers)
        }
    }
}

/** **نقطةٌ على المسار** — بلا وقتٍ ولا دقّة. */
data class GeoPoint(val lat: Double, val lng: Double)

/**
 * **مناورةٌ واحدة.**
 *
 * **و`kind` نصٌّ لا تعداد**: نوعٌ جديدٌ من الخادم **يُقرأ نصّاً ولا
 * يُسقط التطبيق.** والعرضُ يفهم ما يعرف ويقول «تابع المسار» لما لا
 * يعرف.
 */
data class NavManeuver(
    val kind: String,
    val modifier: String? = null,
    /** **المرجعُ الأساسيّ** — موضعُها على طول المسار. */
    val atDistanceM: Double,
    /** **تسريعٌ لا مرجع** — انظر أعلى الملفّ. */
    val atIndex: Int = 0,
    val stepDistanceM: Double = 0.0,
    val stepDurationS: Double = 0.0,
    val streetName: String? = null,
    val roundaboutExit: Int? = null,
    val roundaboutName: String? = null,
) {
    /** **أهي دوّار؟** — يُقرأ في العرض. */
    val isRoundabout: Boolean get() = kind == "ROUNDABOUT" || kind == "EXIT_ROUNDABOUT"
}

/** **أنواعُ المناورات التي نعرفها** — وما عداها يُعرض «تابع المسار». */
object ManeuverKinds {
    const val DEPART = "DEPART"
    const val ARRIVE = "ARRIVE"
    const val STRAIGHT = "STRAIGHT"
    const val TURN_LEFT = "TURN_LEFT"
    const val TURN_RIGHT = "TURN_RIGHT"
    const val SLIGHT_LEFT = "SLIGHT_LEFT"
    const val SLIGHT_RIGHT = "SLIGHT_RIGHT"
    const val SHARP_LEFT = "SHARP_LEFT"
    const val SHARP_RIGHT = "SHARP_RIGHT"
    const val U_TURN = "U_TURN"
    const val MERGE = "MERGE"
    const val FORK = "FORK"
    const val OFF_RAMP = "OFF_RAMP"
    const val ROUNDABOUT = "ROUNDABOUT"
    const val EXIT_ROUNDABOUT = "EXIT_ROUNDABOUT"
    const val UNKNOWN = "UNKNOWN"

    private val known = setOf(
        DEPART, ARRIVE, STRAIGHT, TURN_LEFT, TURN_RIGHT, SLIGHT_LEFT, SLIGHT_RIGHT,
        SHARP_LEFT, SHARP_RIGHT, U_TURN, MERGE, FORK, OFF_RAMP,
        ROUNDABOUT, EXIT_ROUNDABOUT,
    )

    /** **أنعرفه؟** — وما لا نعرفه لا يُسقط شيئاً. */
    fun isKnown(kind: String): Boolean = kind in known

    /** **ما نعرفه كلُّه** — تمشيه الحرّاسُ فلا يُنسى نوعٌ في جدول. */
    fun all(): Set<String> = known
}
