package com.rahalgo.map.data

/**
 * ══════════════════════════════════════════════════════════════════
 * **سياسةُ التبديل — والشبكةُ المتذبذبةُ لا تُعيد تحميلَ النمط**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ١٨.)
 *
 * **أمرُ المالك نصّاً**: «لا أريد Style reload كل ثانية مع تغير
 * Connectivity».
 *
 * # **ولماذا يكلّف التبديلُ أكثرَ ممّا يبدو**
 *
 * **تبديلُ المصدر يعني إعادةَ تحميل النمط.** وMapLibre عند تحميل
 * نمطٍ **تُسقط كلَّ ما فوقه**: خطَّ المسار، ودبّوسَ السائق، ونقطتَي
 * الاستلام والتسليم (البند ١٩). **فيُعاد تركيبُها كلَّها.**
 *
 * **وفي طريقٍ بين مدينتين تسقط الشبكةُ وتعود عشراتِ المرّات.** فلو
 * بُدِّل مع كلّ تغيُّرٍ **لومض المسارُ في وجه السائق وهو يقود.**
 *
 * # **فالتأخيرُ غيرُ متماثل — وذلك مقصود**
 *
 * **الذهابُ دونَ اتّصالٍ سريع**: الخريطةُ سقطت فعلاً، **والانتظارُ
 * يعني فراغاً في الشاشة.**
 *
 * **والعودةُ إلى الاتّصال بطيئة**: الحزمةُ المحلّيّةُ تعمل، **فلا
 * ثمنَ للانتظار ولا فائدةَ من العجلة.** بل أمرُ المالك: «Network
 * يعود → لا ضرورة لتبديل فوري أثناء Navigation».
 */
class MapSwitchPolicy(private val tuning: Tuning = Tuning()) {

    data class Tuning(
        /** **قبل الذهاب دونَ اتّصال** — انقطاعٌ مؤكَّدٌ لا ومضة. */
        val toOfflineDebounceMs: Long = 6_000,

        /** **وقبل العودة** — أطولُ بكثير، فلا داعيَ للعجلة. */
        val toOnlineDebounceMs: Long = 45_000,

        /**
         * **وفي الملاحة لا يُعاد الاتّصالُ إلّا عند نقطةٍ آمنة.**
         *
         * **أمرُ المالك**: «يمكن العودة Online عند نقطة آمنة/جلسة
         * جديدة». **فأثناء ملاحةٍ عاملةٍ دونَ اتّصالٍ لا تبديل.**
         */
        val holdOfflineDuringNavigation: Boolean = true,
    )

    enum class State { ONLINE_ACTIVE, OFFLINE_ACTIVE, SWITCHING }

    /** **ما يُطلب** — والقرارُ يُقاس بالزمن لا باللحظة. */
    data class Input(
        val desired: Desired,
        val navigating: Boolean,
        val nowMs: Long,
    )

    enum class Desired { ONLINE, OFFLINE, UNAVAILABLE }

    data class Output(val state: State, val active: Desired, val why: String)

    private var state: State = State.ONLINE_ACTIVE
    private var active: Desired = Desired.ONLINE
    private var pending: Desired? = null
    private var pendingSinceMs: Long = 0L

    fun state(): State = state
    fun active(): Desired = active

    /**
     * **يبتدئ من حالٍ معلوم** — عند إنشاء الخريطة.
     *
     * **ولا يُعدُّ تبديلاً** — فالتبديلُ ما يقع بعد استقرار.
     */
    fun start(desired: Desired) {
        state = when (desired) {
            Desired.ONLINE -> State.ONLINE_ACTIVE
            Desired.OFFLINE -> State.OFFLINE_ACTIVE
            Desired.UNAVAILABLE -> State.OFFLINE_ACTIVE
        }
        active = desired
        pending = null
        pendingSinceMs = 0L
    }

    fun update(input: Input): Output {
        val want = input.desired

        // ── لا تغيير ───────────────────────────────────────────────
        if (want == active) {
            pending = null
            state = stateOf(active)
            return Output(state, active, "مستقرّ")
        }

        /**
         * **وانقطاعُ البيانات كلِّيّاً يُبلَّغ فوراً** — البند ٣٧.
         *
         * **لا انتظارَ ولا تمهيد**: لا مصدرَ يعمل أصلاً، **والتأخيرُ
         * لا يحفظ شيئاً.**
         */
        if (want == Desired.UNAVAILABLE) {
            active = want
            state = State.OFFLINE_ACTIVE
            pending = null
            return Output(state, active, "لا مصدرَ للبلاطات")
        }

        // ── تعليقُ العودة أثناء ملاحةٍ عاملة ───────────────────────
        if (want == Desired.ONLINE &&
            active == Desired.OFFLINE &&
            input.navigating &&
            tuning.holdOfflineDuringNavigation
        ) {
            pending = null
            state = State.OFFLINE_ACTIVE
            return Output(state, active, "الاتّصالُ عاد — والملاحةُ تعمل دونَه فلا تبديل")
        }

        // ── مهلةٌ قبل التبديل ──────────────────────────────────────
        if (pending != want) {
            pending = want
            pendingSinceMs = input.nowMs
            state = State.SWITCHING
            return Output(state, active, "طُلب $want — تُنتظَر المهلة")
        }

        val waited = input.nowMs - pendingSinceMs
        val need = if (want == Desired.OFFLINE) {
            tuning.toOfflineDebounceMs
        } else {
            tuning.toOnlineDebounceMs
        }

        if (waited < need) {
            state = State.SWITCHING
            return Output(state, active, "بقي ${need - waited} م.ث")
        }

        active = want
        state = stateOf(active)
        pending = null
        return Output(state, active, "بُدِّل بعد $waited م.ث")
    }

    private fun stateOf(d: Desired): State = when (d) {
        Desired.ONLINE -> State.ONLINE_ACTIVE
        else -> State.OFFLINE_ACTIVE
    }
}
