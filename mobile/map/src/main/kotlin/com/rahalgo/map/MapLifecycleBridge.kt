package com.rahalgo.map

/**
 * ══════════════════════════════════════════════════════════════════
 * **جسرُ دورة الحياة — واحدٌ لكلّ الخرائط**
 * ══════════════════════════════════════════════════════════════════
 *
 * (المرحلة ٦ب، قرارُ المالك ٢٠٢٦-٠٨-٢١، البنود ٢٨ و٢٩ و٣٠.)
 *
 * # **العيبُ الذي أُغلق هنا**
 *
 * **`TripMap` لم تكن تستدعي `onCreate`** (دَينُ `TD-MAP-ONCREATE`).
 * و`MapCanvas` كانت تستدعيها في `remember` **وتترك `onSaveInstance
 * State` بلا أحد.** فكانت لكلّ شاشةٍ دورةُ حياةٍ مختلفة.
 *
 * **و`onLowMemory` لم تكن تصل أيَّ خريطة** (`TD-MAP-LOWMEM`) —
 * فالنظامُ يطلب من التطبيق أن يفرّغ ذاكرةً **والخريطةُ لا تسمع**،
 * فتُقتل العمليّةُ بدل أن تتقلّص.
 *
 * # **ولماذا سجلٌّ لا مرجعٌ عامّ**
 *
 * **البند ٢٩**: «لا تنشئ global leaked references إلى Views».
 *
 * `MapView` تحمل `Context` النشاط. **ومرجعٌ ساكنٌ إليها يُبقي النشاطَ
 * كلَّه في الذاكرة بعد إغلاقه** — وهو التسريبُ الكلاسيكيّ.
 *
 * **فالسجلُّ يُسجَّل فيه عند الإنشاء ويُنزع عند الهدم**، **والنزعُ
 * مضمونٌ لأنّه في `onDestroy` نفسِها** لا في مسارٍ منفصلٍ قد لا يُسلك.
 */
object MapLifecycleBridge {

    /**
     * **ما يُدار** — مُجرَّدٌ عن `MapView` ليُختبر بلا أندرويد.
     *
     * **البند ٤٩ يطلب اختبار الجسر**، و`MapView` تحتاج جهازاً أو
     * Robolectric. **فالجسرُ يدير واجهةً، والوصلةُ إلى `MapView`
     * سطرٌ واحدٌ لا يحتاج اختباراً.**
     */
    interface Host {
        fun onCreate()
        fun onStart()
        fun onResume()
        fun onPause()
        fun onStop()
        fun onDestroy()
        fun onLowMemory()
        fun onSaveInstanceState()
    }

    enum class Phase { NEW, CREATED, STARTED, RESUMED, PAUSED, STOPPED, DESTROYED }

    /**
     * **مقبضٌ لخريطةٍ حيّة.**
     *
     * **وكلُّ نداءٍ عابرُ حالة**: `onStart` مرّتين لا تُنادي المضيفَ
     * مرّتين، **و`onDestroy` بعد `onDestroy` لا تفعل شيئاً** (البند
     * ٣٠). **فإعادةُ التركيب في Compose تقع كثيراً**، ولو مرَّ كلُّ
     * نداءٍ **لتضاعفت الأحداثُ بلا سبب.**
     */
    class Handle internal constructor(private val host: Host) {
        var phase: Phase = Phase.NEW
            private set

        internal val alive: Boolean get() = phase != Phase.DESTROYED

        fun create() {
            if (phase != Phase.NEW) return
            phase = Phase.CREATED
            host.onCreate()
        }

        fun start() {
            if (phase == Phase.DESTROYED) return
            if (phase == Phase.NEW) create()
            if (phase == Phase.STARTED || phase == Phase.RESUMED) return
            phase = Phase.STARTED
            host.onStart()
        }

        fun resume() {
            if (phase == Phase.DESTROYED) return
            if (phase != Phase.STARTED) start()
            if (phase == Phase.RESUMED) return
            phase = Phase.RESUMED
            host.onResume()
        }

        fun pause() {
            if (phase != Phase.RESUMED) return
            phase = Phase.PAUSED
            host.onPause()
        }

        fun stop() {
            if (phase == Phase.DESTROYED || phase == Phase.STOPPED || phase == Phase.NEW) return
            if (phase == Phase.RESUMED) pause()
            phase = Phase.STOPPED
            host.onStop()
        }

        fun saveInstanceState() {
            if (phase == Phase.DESTROYED || phase == Phase.NEW) return
            host.onSaveInstanceState()
        }

        fun destroy() {
            if (phase == Phase.DESTROYED) return
            if (phase == Phase.RESUMED) pause()
            if (phase == Phase.STARTED || phase == Phase.PAUSED) stop()
            phase = Phase.DESTROYED
            host.onDestroy()
            unregister(this)
        }

        internal fun lowMemory() {
            if (phase == Phase.DESTROYED) return
            host.onLowMemory()
        }
    }

    private val live = mutableListOf<Handle>()

    fun register(host: Host): Handle {
        val handle = Handle(host)
        synchronized(live) { live += handle }
        return handle
    }

    private fun unregister(handle: Handle) {
        synchronized(live) { live.remove(handle) }
    }

    /** **عددُ الخرائط الحيّة** — يُقرأ في الاختبار لكشف التسريب. */
    fun liveCount(): Int = synchronized(live) { live.count { it.alive } }

    /**
     * **يوصل نداءَ النظام إلى كلّ خريطةٍ حيّة** — البند ٢٩.
     *
     * **والمهدومةُ لا تستقبل** — تُنزع من السجلّ في `destroy`.
     */
    fun dispatchLowMemory() {
        val snapshot = synchronized(live) { live.toList() }
        for (h in snapshot) h.lowMemory()
    }

    /** **للاختبار وحدَه** — فالسجلُّ ساكنٌ ويعبر بين الاختبارات. */
    fun resetForTest() {
        synchronized(live) { live.clear() }
    }
}
