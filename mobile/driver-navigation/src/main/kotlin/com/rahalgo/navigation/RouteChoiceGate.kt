package com.rahalgo.navigation

/**
 * ══════════════════════════════════════════════════════════════════
 * **بوّابةُ نشر البدائل — ولا تعود خطوطٌ لطلبٍ مضى**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ صحّة واجهة ٧، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ١ إلى ٤.)
 *
 * # **السباقُ الذي يُغلق هنا**
 *
 * **أمرُ المالك نصّاً**:
 *
 *	PICKUP request A starts
 *	↓ picked_up
 *	↓ clearChoices()
 *	↓ DROPOFF request B starts
 *	↓ A returns late
 *	فقد تعود خطوطٌ إلى PICKUP بعد أن انتقل الطلب إلى DROPOFF.
 *
 * **والفحصُ عند الاعتماد لا يكفي**: الخطُّ يُرسم على الخريطة **قبل
 * أن يضغط أحد**، **والسائقُ يرى طريقاً إلى متجرٍ فارقه.**
 *
 * # **فالحارسُ عند النشر لا عند الاستعمال**
 *
 * **كلُّ طلبٍ يلتقط سياقَه**، **ولا يُنشر ردٌّ إلّا إن كان سياقُه هو
 * الحاضر.** **ولا يظهر ولو للحظة.**
 *
 * # **والإلغاءُ وحدَه ليس حماية** (البند ٤)
 *
 * **`Job.cancel()` لا تصل الخادمَ**، **وقد يكون الردُّ في الطريق
 * أصلاً.** فالتسلسلُ يبقى ولو أُلغي.
 */
object RouteChoiceGate {

    /**
     * **سياقُ طلبٍ واحد** — يُلتقط عند الإطلاق ويُقارَن عند الردّ.
     */
    data class RequestContext(
        /** **الأحدثُ يفوز** — البند ٢. */
        val seq: Long,
        val orderId: String,
        val target: RouteTarget,
        val generation: Long,
        /**
         * **بصمةُ المسار الذي تعمل عليه الملاحةُ الآن.**
         *
         * **وبها يُفحص التوافق** (البند ٣): **فالبدائلُ بدائلُ مسارٍ
         * بعينه**، وردٌّ موصًى بهِ مسارٌ آخرُ **ليس بدائلَ لما نقود.**
         */
        val navRouteFingerprint: Long,
    )

    /** **لماذا رُفض الردّ** — يُسجَّل للتشخيص (البند ٣). */
    enum class Reject {
        NONE,
        STALE_SEQ,
        ORDER_CHANGED,
        TARGET_CHANGED,
        GENERATION_CHANGED,
        RECOMMENDED_MISMATCH,
        EMPTY,
    }

    /**
     * **أيُنشَر هذا الردّ؟**
     *
     * @param request **سياقُ الطلب** كما التُقط عند الإطلاق.
     * @param latestSeq **أحدثُ تسلسلٍ أُطلق** — فردٌّ أقدمُ يُطرح.
     * @param now **السياقُ الحاضر** عند وصول الردّ.
     * @param recommendedFingerprint **بصمةُ الموصى به في الردّ.**
     */
    fun verdict(
        request: RequestContext,
        latestSeq: Long,
        now: RequestContext,
        recommendedFingerprint: Long,
        alternativeCount: Int,
        compatibility: RouteCompatibility.Tuning = RouteCompatibility.Tuning(),
        recommendedGeometry: List<GeoPoint>? = null,
        currentGeometry: List<GeoPoint>? = null,
    ): Reject {
        // **الأحدثُ يفوز** — البند ٢: «B returns first and publishes ·
        // A returns later · A cannot overwrite B».
        if (request.seq != latestSeq) return Reject.STALE_SEQ
        if (request.orderId != now.orderId) return Reject.ORDER_CHANGED
        if (request.target != now.target) return Reject.TARGET_CHANGED
        if (request.generation != now.generation) return Reject.GENERATION_CHANGED
        if (alternativeCount <= 0) return Reject.EMPTY

        /**
         * **وتوافقُ الموصى به** — البندان ٣ و١٩.
         *
         * **البصمةُ أوّلاً** — فإن طابقت فلا حاجةَ إلى هندسة.
         * **وإلّا تُقاس المشابهة**: تقطيعٌ مختلفٌ أو نقطةُ التقاطٍ
         * تحرّكت متراً **لا يجوز أن تُسقط ردّاً سليماً.**
         */
        if (recommendedFingerprint != request.navRouteFingerprint) {
            val a = recommendedGeometry
            val b = currentGeometry
            if (a == null || b == null) return Reject.RECOMMENDED_MISMATCH
            if (!RouteCompatibility.sameRoute(a, b, compatibility)) {
                return Reject.RECOMMENDED_MISMATCH
            }
        }
        return Reject.NONE
    }

    fun accepted(reject: Reject): Boolean = reject == Reject.NONE

    /**
     * ══════════════════════════════════════════════════════════════
     * **أيُطلب جلبٌ الآن؟** — البندان ٦ و٨
     * ══════════════════════════════════════════════════════════════
     *
     * **هنا وحدَه يُتّخذ القرار** — تناديها الشاشةُ عند كلّ تبدّلِ
     * جيلٍ أو طورٍ أو حال، **ويَعُدّ عليها الاختبارُ الطلباتِ.**
     *
     * **ولولا أنّها واحدةٌ لكان الاختبارُ يقيس نسخةً منها** — وذاك
     * لا يُثبت شيئاً عن الشاشة.
     *
     * @param hasRoute **لا جلبَ قبل أن يُركَّب مسار.**
     * @param healthy `NavSituation.ON_ROUTE` وحدَها (البند ١٠).
     * @param hasOrigin **موضعُ السائق معلوم؟**
     * @param reason **واختيارُ السائق لا يطلب** — البند ٦.
     */
    fun shouldFetch(
        hasRoute: Boolean,
        healthy: Boolean,
        hasOrigin: Boolean,
        reason: RouteInstallReason,
    ): Boolean = hasRoute && healthy && hasOrigin && reason.fetchesAlternatives
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **توافقُ الموصى به — أهو المسارُ الذي نقوده؟**
 * ══════════════════════════════════════════════════════════════════
 *
 * (البندان ٣ و١٩.)
 *
 * **أمرُ المالك نصّاً**: «إذا identity الحالية شديدة الصرامة، استخدم
 * compatibility check مناسبًا حتى لا نرفض Response سليمة. لكن لا تخلط
 * هذا مع Diversity».
 *
 * # **وثلاثةُ مفاهيمَ لا يُخلط بينها**
 *
 * **الهويّة** (`RouteFingerprint`) — تطابقٌ أو لا. **صارمةٌ**: رأسٌ
 * واحدٌ يتزحزح متراً يُبدّلها.
 *
 * **والتنوّع** (`SharedRatio` في الخادم) — كم يشتركان؟ **مقياسُ
 * اختلافٍ بين بديلين.**
 *
 * **والتوافقُ هنا** — **أهما المسارُ نفسُه عمليّاً؟** سؤالٌ ثالثٌ:
 * **يقبل تقطيعاً مختلفاً ونقطةَ التقاطٍ تحرّكت، ويرفض طريقاً آخر.**
 *
 * **ولو استُعملت الهويّةُ لرُفض كلُّ ردٍّ تقريباً**: الطلبُ الثاني
 * أصلُه موضعُ السائق **وقد تحرّك أمتاراً بين النداءين.**
 */
object RouteCompatibility {

    data class Tuning(
        /**
         * **أدنى تشابهٍ ليُعدَّا مساراً واحداً.**
         *
         * **وعالٍ عمداً**: هذا سؤالُ «أهو هو؟» لا «كم يشتركان؟».
         * **وقِيس في الخادم أنّ بديلين حقيقيّين أقصى تشابهِهما ٨٢٫٦٪**
         * — فالتسعون تفصل «هو هو» عن «بديلٌ قريب».
         */
        val minShared: Double = 0.90,

        /** **وفرقُ الطول المسموح** — نقطةُ التقاطٍ تحرّكت لا أكثر. */
        val maxLengthDeltaRatio: Double = 0.05,
    )

    /**
     * **أهما المسارُ نفسُه عمليّاً؟**
     *
     * **بالإسقاط لا بمطابقة الرؤوس** — كما في الخادم: **يُعاد
     * التقطيعُ ويُسأل كم من عيّنات أحدهما تقع على الآخر.**
     */
    fun sameRoute(a: List<GeoPoint>, b: List<GeoPoint>, tuning: Tuning = Tuning()): Boolean {
        if (a.size < 2 || b.size < 2) return false

        val lenA = pathLength(a)
        val lenB = pathLength(b)
        if (lenA <= 0 || lenB <= 0) return false
        val longer = maxOf(lenA, lenB)
        if (kotlin.math.abs(lenA - lenB) / longer > tuning.maxLengthDeltaRatio) return false

        val shared = minOf(similarity(a, b), similarity(b, a))
        return shared >= tuning.minShared
    }

    /** **نسبةُ عيّنات `b` التي تقع على `a`.** */
    fun similarity(a: List<GeoPoint>, b: List<GeoPoint>): Double {
        if (a.size < 2 || b.size < 2) return 0.0
        val samples = resample(b, STEP_M)
        if (samples.isEmpty()) return 0.0
        var on = 0
        for (p in samples) {
            if (onPath(a, p)) on++
        }
        return on.toDouble() / samples.size
    }

    const val STEP_M = 25.0
    const val TOLERANCE_M = 30.0

    private fun pathLength(g: List<GeoPoint>): Double {
        var d = 0.0
        for (i in 1 until g.size) {
            d += GpsQuality.metersBetween(g[i - 1].lat, g[i - 1].lng, g[i].lat, g[i].lng)
        }
        return d
    }

    private fun resample(g: List<GeoPoint>, step: Double): List<GeoPoint> {
        if (g.size < 2) return g
        val out = mutableListOf(g[0])
        var carry = 0.0
        for (i in 1 until g.size) {
            val a = g[i - 1]
            val b = g[i]
            val d = GpsQuality.metersBetween(a.lat, a.lng, b.lat, b.lng)
            if (d <= 0) continue
            var t = step - carry
            while (t <= d) {
                val f = t / d
                out += GeoPoint(a.lat + (b.lat - a.lat) * f, a.lng + (b.lng - a.lng) * f)
                t += step
            }
            carry = (carry + d) % step
        }
        out += g[g.size - 1]
        return out
    }

    /**
     * **أتقع النقطةُ على المسار؟**
     *
     * **بالقطعة لا بالرأس** — فرؤوسُ الطرق السريعة متباعدة، **ونقطةٌ
     * في وسط قطعةٍ بعيدةٌ عن كلّ رأس** (العيبُ الذي أُصلح في الخادم).
     */
    private fun onPath(g: List<GeoPoint>, p: GeoPoint): Boolean {
        for (i in 0 until g.size - 1) {
            if (segmentDistanceM(p, g[i], g[i + 1]) <= TOLERANCE_M) return true
        }
        return false
    }

    private fun segmentDistanceM(p: GeoPoint, a: GeoPoint, b: GeoPoint): Double {
        val kx = kotlin.math.cos(Math.toRadians(p.lat)) * 111320.0
        val ky = 110540.0
        val px = p.lng * kx
        val py = p.lat * ky
        val ax = a.lng * kx
        val ay = a.lat * ky
        val bx = b.lng * kx
        val by = b.lat * ky
        val dx = bx - ax
        val dy = by - ay
        if (dx == 0.0 && dy == 0.0) return kotlin.math.hypot(px - ax, py - ay)
        var t = ((px - ax) * dx + (py - ay) * dy) / (dx * dx + dy * dy)
        t = t.coerceIn(0.0, 1.0)
        return kotlin.math.hypot(px - (ax + t * dx), py - (ay + t * dy))
    }
}

/**
 * ══════════════════════════════════════════════════════════════════
 * **لماذا رُكّب المسار — والاختيارُ لا يُطلق جلباً**
 * ══════════════════════════════════════════════════════════════════
 *
 * (البنود ٦ و٧ و٩.)
 *
 * **أمرُ المالك نصّاً**: «بعد أن اختار السائق Alternative: لا نريد أن
 * تعود لوحة البدائل مباشرةً بسبب generation التي سببها الاختيار
 * نفسه».
 *
 * # **والجيلُ وحدَه لا يقول السبب**
 *
 * **يزيد في ثلاث حالاتٍ**: مسارٌ أوّل، وإعادةُ حسابٍ نجحت، **واختيارُ
 * السائق.** **والأولى والثانيةُ تستحقّان جلباً، والثالثةُ لا** —
 * فالسائقُ اختار للتوّ.
 *
 * **ولا يُعدَّل `NavEngine`** (البند ٧) — السببُ يُمسك هنا، **حيث
 * يُعرف من ناداه.**
 */
enum class RouteInstallReason {
    /** **أوّلُ مسارٍ لهذه الساق** — يُطلب. */
    INITIAL,

    /** **تبدّلت الوجهة** — يُطلب بعد استقرار الموصى به. */
    TARGET_CHANGED,

    /** **إعادةُ حسابٍ نجحت** — يُطلب بعد عودة `ON_ROUTE` (البند ٩). */
    AUTOMATIC_REROUTE,

    /** **اختارَ السائق** — **ولا يُطلب** (البند ٦). */
    USER_SELECTION,
    ;

    /** **أيستحقّ جلبَ بدائل؟** */
    val fetchesAlternatives: Boolean get() = this != USER_SELECTION
}
