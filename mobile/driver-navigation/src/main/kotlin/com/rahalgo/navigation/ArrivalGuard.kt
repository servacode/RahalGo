package com.rahalgo.navigation

import kotlin.math.max
import kotlin.math.min

/**
 * ══════════════════════════════════════════════════════════════════
 * **ثلاثُ حالاتٍ لا واحدة**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ دلالات الوصول التجاريّ، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * **وخلطُها هو العيبُ الذي وقع مرّتين**:
 *
 *     ROUTE_ENDPOINT_REACHED  ←  انتهت الهندسةُ التي يعرفها المحرّك
 *     NEAR_BUSINESS_TARGET    ←  قاربَ المتجرَ أو بابَ الزبون
 *     BUSINESS_TARGET_ARRIVED ←  **الدعوى** — وحدَها تملك عبارةَ الوصول
 *
 * **والأولى ليست الثالثة.** «انتهى الطريقُ المرسوم» شيء، **و«وصل
 * السائقُ إلى الزبون» شيءٌ آخر.**
 */
enum class ArrivalPhase {
    /** **ما زال في الطريق.** */
    EN_ROUTE,

    /**
     * **آخرُ ميل** — انتهى إرشادُ الطريق ولمّا يُبلَغ الهدف.
     *
     * **ولم تُسمّ «near»**: مئةٌ وثمانون متراً ليست قُرباً (البند ٣).
     *
     * **والملاحةُ انتهت ولا تخترع طريقاً** — الهدفُ يبقى ظاهراً على
     * الخريطة، **والسائقُ يقطع آخرَ الأمتار بنفسه.**
     */
    LAST_MILE_TO_TARGET,

    /** **وصل** — وهنا وحدَها تُقال العبارة. */
    BUSINESS_TARGET_ARRIVED,
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **حدُّ الوصول التجاريّ — ولماذا ليس مئة**
 * ══════════════════════════════════════════════════════════════════
 *
 * **الخطأُ الذي وقعتُ فيه**: أخذتُ **توزيعَ مسافة الالتقاط** (p99
 * الحضريّ ٩٣٫٥م) **وجعلتُه حدَّ وصول.** وهما كمّيّتان مختلفتان:
 *
 *     مسافةُ الالتقاط  ←  كم يبعد المتجرُ عن **أقرب طريقٍ سالكة**
 *     حدُّ الوصول      ←  كم يُقبل بُعدُ السائق عن **المتجر** ليُقال «وصل»
 *
 * **فأن ينتهي الخطُّ على بُعد ٩٣م لا يعني أنّ السائقَ وصل** — يعني
 * أنّ الطريقَ المرسومَ انتهى هناك.
 *
 * # ومن أين الرقم
 *
 * **من المنتج لا من الاستنتاج.** قرارُ المالك ٢٠٢٦-٠٨-١٢:
 *
 *   «ثمانون كثيرٌ جدّاً… خمسةَ عشرَ ممتازة، المسافاتُ في الرقّة
 *    قريبة، مدينةٌ صغيرة»
 *
 * **وهو الرقمُ الذي تعمل به `near()` في الشاشة منذ ذلك اليوم** —
 * فلا رقمان لحقيقةٍ واحدة (البند ١٤).
 *
 * # وما قِيس هنا
 *
 * **٣٠٠ متجرٍ حضريٍّ حقيقيٍّ من `syria.osm.pbf`** — كم مرّةً تتصادف
 * نهايةُ الخطّ مع حدّ الوصول:
 *
 *     ١٥م → ٥٥٫٧٪   ·   ٢٠م → ٧٠٫٠٪   ·   ٢٥م → ٨٢٫٠٪   ·   ٣٠م → ٨٨٫٧٪
 *
 * **وفي الباقي يمشي السائقُ ٢٣م وسيطاً** — وذلك سلوكٌ سليمٌ لا عيب:
 * **يركن ويمشي إلى الباب، فتُقال العبارةُ حين يصل حقّاً.**
 */
object BusinessArrival {

    /** **حدُّ الوصول** — قرارُ المالك ٢٠٢٦-٠٨-١٢، ومصدرُه واحد. */
    const val RADIUS_M = 15.0

    /**
     * **وما يُسمح به من شكّ الجهاز — مسقوفاً.**
     *
     * ══════════════════════════════════════════════════════════════
     * **ولماذا يُضاف هنا ويُطرح في `RouteProgress`**
     * ══════════════════════════════════════════════════════════════
     *
     * **لأنّ خمسةَ عشرَ هي دقّةُ الـGPS في أحسن حالاتها** — كما كُتب
     * يومَ اختيرت. **فاشتراطُ `مسافة + دقّة ≤ ١٥` لا يتحقّق أبداً**،
     * **وميزةٌ لا تعمل أسوأُ من ميزةٍ لا توجد** (نصُّ التعليق نفسِه).
     *
     * **فالسماحُ مسقوفٌ لا مفتوح**: الحدُّ الفعّالُ بين ١٥م و٣٠م
     * **ولا يتجاوزهما**، **ودقّةٌ أسوأُ من [MAX_ACCURACY_M] لا تؤكّد
     * شيئاً أصلاً.**
     */
    const val ACCURACY_ALLOWANCE_M = 15.0

    /**
     * **ودقّةٌ أوسعُ من هذه لا تُثبت وصولاً.**
     *
     * **أمرُ المالك نصّاً**: «GPS accuracy 50m لا تجعلنا نقول "وصلت"
     * من عشرات الأمتار». **فقراءةٌ بخمسين تُرفض قبل أن تُقاس.**
     */
    const val MAX_ACCURACY_M = 30.0

    /**
     * **الحدُّ الفعّالُ لهذه القراءة** — بين [RADIUS_M] و٣٠م.
     */
    fun effectiveRadiusM(accuracyM: Float): Double =
        RADIUS_M + min(max(0.0, accuracyM.toDouble()), ACCURACY_ALLOWANCE_M)
}

/**
 * **يحكم أيَّ الحالات الثلاث نحن فيها.**
 *
 * **ولا يخترع موضعاً**: يقرأ ما قالته [RouteProgress] عن الهندسة،
 * **وما تقوله القراءةُ عن الهدف الحقيقيّ.**
 */
object ArrivalGuard {

    /**
     * @param routeArrived ما قالته [RouteProgress] على الهندسة.
     * @param target **الهدفُ التجاريّ** — و`null` تعني: لم يُعطَ،
     *   **فيُعمل بالسلوك القديم** ولا تنكسر شاشةٌ لم تُحدَّث بعد.
     */
    fun phase(
        routeArrived: Boolean,
        target: GeoPoint?,
        fix: NavFix?,
        grade: FixGrade,
    ): ArrivalPhase {
        // ── لا هدفَ فلا حارس — البند ٤ من الإغلاق السابق ──────────
        if (target == null) {
            return if (routeArrived) ArrivalPhase.BUSINESS_TARGET_ARRIVED else ArrivalPhase.EN_ROUTE
        }

        val confirmed = confirms(target, fix, grade)
        if (confirmed) return ArrivalPhase.BUSINESS_TARGET_ARRIVED
        // **وانتهاءُ الخطّ حالةٌ قائمةٌ بذاتها** — لا وصولاً ولا سيراً.
        if (routeArrived) return ArrivalPhase.LAST_MILE_TO_TARGET
        return ArrivalPhase.EN_ROUTE
    }

    /**
     * **أتؤكّد هذه القراءةُ الوصولَ؟**
     *
     * ══════════════════════════════════════════════════════════════
     * **والدعوى تحتاج شاهداً مقبولاً — البند ٥**
     * ══════════════════════════════════════════════════════════════
     *
     *     REJECTED  →  لا شيء
     *     DEGRADED  →  **لا تؤكّد وحدَها** — موضعُها يُؤخذ ولا يُبنى
     *                  عليه إعلانٌ نهائيّ
     *     ACCEPTED  →  تؤكّد ضمنَ الحدّ الفعّال
     *
     * **ولا أنبوبَ ثانياً** — هذه درجاتُ المرحلة ١ نفسُها.
     */
    fun confirms(target: GeoPoint?, fix: NavFix?, grade: FixGrade): Boolean {
        if (target == null || fix == null) return false
        // **والمتدهورةُ لا تؤكّد** — البند ٥ صراحةً.
        if (grade != FixGrade.ACCEPTED) return false
        if (fix.accuracyM > BusinessArrival.MAX_ACCURACY_M) return false
        val d = distanceToTargetM(target, fix)
        return d in 0.0..BusinessArrival.effectiveRadiusM(fix.accuracyM)
    }

    /**
     * **أقاربَ الهدفَ؟** — حالُ العرض، لا الدعوى.
     *
     * **وهي ما تبنيه الشاشةُ عليه** — بالحدّ نفسِه ولا شهادةَ لازمة،
     * **لأنّها اقتراحٌ لا فعل** (نصُّ الشاشة: «واقتراحٌ لا فعل»).
     */
    fun near(target: GeoPoint?, lat: Double, lng: Double): Boolean {
        if (target == null) return false
        return distanceM(lat, lng, target.lat, target.lng) <= BusinessArrival.RADIUS_M
    }

    /**
     * **كم يبعد عن هدفه** — للتشخيص والاختبار.
     *
     * **وسالبٌ يعني: لا هدفَ أو لا قراءة.**
     */
    fun distanceToTargetM(target: GeoPoint?, fix: NavFix?): Double {
        if (target == null || fix == null) return -1.0
        return distanceM(fix.lat, fix.lng, target.lat, target.lng)
    }

    /**
     * **مسافةٌ على سطح الكرة** — بالصيغة نفسِها التي في
     * [RouteProjector]، فلا رقمان لمسافةٍ واحدة.
     */
    fun distanceM(aLat: Double, aLng: Double, bLat: Double, bLng: Double): Double {
        val r = 6371000.0
        val p1 = Math.toRadians(aLat)
        val p2 = Math.toRadians(bLat)
        val dp = Math.toRadians(bLat - aLat)
        val dl = Math.toRadians(bLng - aLng)
        val s = Math.sin(dp / 2) * Math.sin(dp / 2) +
            Math.cos(p1) * Math.cos(p2) * Math.sin(dl / 2) * Math.sin(dl / 2)
        return 2 * r * Math.asin(max(0.0, Math.sqrt(s)).coerceAtMost(1.0))
    }
}
