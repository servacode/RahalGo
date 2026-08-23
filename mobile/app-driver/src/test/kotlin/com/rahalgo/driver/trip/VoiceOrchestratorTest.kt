package com.rahalgo.driver.trip

import com.rahalgo.navigation.CueId
import com.rahalgo.navigation.CueKind
import com.rahalgo.navigation.CueStage
import com.rahalgo.navigation.Speaker
import com.rahalgo.navigation.VoiceCue
import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ المنسّق — بناطقٍ في اليد لا بسمّاعة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * المعرّفات: `VORCH-*`
 *
 * (أمرُ المالك، البند ٣١: «ولا تجعل Unit Tests تحتاج سمّاعةً
 *  حقيقيّة».)
 */
class VoiceOrchestratorTest {

    /** **ناطقٌ يسجّل ولا ينطق** — ويمسك الجوابَ إن أُريد. */
    class FakeSpeaker(override var available: Boolean = true) : Speaker {
        val said = ArrayList<String>()
        val flushes = ArrayList<Boolean>()
        var stops = 0
            private set
        private var pending: (() -> Unit)? = null

        /** **أينهي الجملةَ فوراً؟** — والافتراضُ نعم. */
        var instant = true

        override fun speak(id: String, text: String, flush: Boolean, done: (Boolean) -> Unit) {
            said += text
            flushes += flush
            if (instant) done(true) else pending = { done(true) }
        }

        override fun stop() {
            stops++
            pending = null
        }

        /** **ينهي الجملةَ المعلَّقة.** */
        fun finish() {
            val p = pending
            pending = null
            p?.invoke()
        }

        val busy: Boolean get() = pending != null
    }

    private fun cue(
        atM: Double,
        stage: CueStage,
        priority: Int,
        text: String,
        generation: Long = 1L,
        validUntil: Double = Double.MAX_VALUE,
    ) = VoiceCue(
        id = CueId(generation, atM, stage),
        kind = CueKind.MANEUVER,
        stage = stage,
        priority = priority,
        validUntilProgressM = validUntil,
        text = text,
    )

    // ══════════════════════════════════════════════════════════════════
    // **١ · لا تكرار**
    // ══════════════════════════════════════════════════════════════════

    /** **VORCH-001** — الهُويّةُ نفسُها لا تُقال مرّتين. */
    @Test
    fun `VORCH-001 لا تُقال الهُويّةُ مرّتين`() {
        val s = FakeSpeaker()
        val o = VoiceOrchestrator(s)
        val c = cue(300.0, CueStage.NOW, 3, "انعطف يميناً الآن")
        repeat(10) { o.offer(listOf(c), progressM = 280.0) }
        println("VORCH-001 · قيلت=${s.said.size} · مطروحة=${o.dropped}")
        assertEquals(1, s.said.size)
        assertEquals(9, o.dropped)
    }

    /** **VORCH-002** — وإعادةُ إنشاء الشاشة لا تُعيد ما قيل. */
    @Test
    fun `VORCH-002 إعادةُ إنشاء الشاشة لا تُعيد ما قيل`() {
        val s = FakeSpeaker()
        val o = VoiceOrchestrator(s)
        val c = cue(300.0, CueStage.APPROACH, 1, "بعد مئة وخمسين متراً، انعطف يميناً")
        o.offer(listOf(c), 150.0)
        // **والشاشةُ تُبنى من جديدٍ فيُعاد توليدُ التعليمة نفسِها.**
        o.offer(listOf(c), 155.0)
        println("VORCH-002 · قيلت=${s.said.size}")
        assertEquals("أُعيدت تعليمةٌ بعد دورانِ جهاز", 1, s.said.size)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · الصلاحيّة**
    // ══════════════════════════════════════════════════════════════════

    /** **VORCH-010** — المنتهيةُ صلاحيّتُها لا تُقال. */
    @Test
    fun `VORCH-010 تعليمةٌ مضى وقتُها تُطرح قبل النطق`() {
        val s = FakeSpeaker()
        val o = VoiceOrchestrator(s)
        val c = cue(300.0, CueStage.NOW, 3, "انعطف يميناً الآن", validUntil = 325.0)
        // **والسائقُ تجاوز المنعطفَ بالفعل.**
        o.offer(listOf(c), progressM = 400.0)
        println("VORCH-010 · قيلت=${s.said.size} · ساقطةٌ بالصلاحيّة=${o.staleDropped}")
        assertEquals("قيلت تعليمةٌ عن منعطفٍ خلفَه", 0, s.said.size)
        assertEquals(1, o.staleDropped)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٣ · الأولويّةُ والطابور**
    // ══════════════════════════════════════════════════════════════════

    /** **VORCH-020** — «الآن» تقطع تمهيداً جارياً. */
    @Test
    fun `VORCH-020 الآن تقطع التمهيد`() {
        val s = FakeSpeaker().also { it.instant = false }
        val o = VoiceOrchestrator(s)
        o.offer(listOf(cue(900.0, CueStage.PREPARE, 0, "بعد خمس مئة متر، انعطف يساراً")), 400.0)
        assertTrue("لم يبدأ التمهيد", s.busy)
        o.offer(listOf(cue(300.0, CueStage.NOW, 3, "انعطف يميناً الآن")), 280.0)
        println("VORCH-020 · قيلت=${s.said} · قطع=${s.flushes}")
        assertEquals(2, s.said.size)
        assertTrue("لم تُقطع الجملةُ الأدنى", s.flushes.last())
    }

    /** **VORCH-021** — والتمهيدُ لا يقطع «الآن». */
    @Test
    fun `VORCH-021 التمهيدُ لا يقطع الآن`() {
        val s = FakeSpeaker().also { it.instant = false }
        val o = VoiceOrchestrator(s)
        o.offer(listOf(cue(300.0, CueStage.NOW, 3, "انعطف يميناً الآن")), 280.0)
        o.offer(listOf(cue(900.0, CueStage.PREPARE, 0, "بعد خمس مئة متر، انعطف يساراً")), 285.0)
        println("VORCH-021 · قيلت=${s.said}")
        assertEquals("التمهيدُ قطع «الآن»", 1, s.said.size)
        assertEquals(1, o.pending)
    }

    /** **VORCH-022** — والطابورُ محكومٌ لا مفتوح. */
    @Test
    fun `VORCH-022 الطابورُ لا يتجاوز حدَّه`() {
        val s = FakeSpeaker().also { it.instant = false }
        val o = VoiceOrchestrator(s)
        o.offer(listOf(cue(100.0, CueStage.NOW, 3, "الأولى")), 0.0)
        val many = (1..10).map { cue(100.0 + it, CueStage.PREPARE, 0, "تمهيد $it") }
        o.offer(many, 0.0)
        println("VORCH-022 · ينتظر=${o.pending} · مطروحة=${o.dropped}")
        assertTrue("طابورٌ بلا حدّ: ${o.pending}", o.pending <= 3)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٤ · الكتم**
    // ══════════════════════════════════════════════════════════════════

    /** **VORCH-030** — المكتومُ لا ينطق ولا يستدرك. */
    @Test
    fun `VORCH-030 الكتمُ لا يستدرك ما فات`() {
        val s = FakeSpeaker()
        val o = VoiceOrchestrator(s)
        o.muted = true
        o.offer(listOf(cue(300.0, CueStage.PREPARE, 0, "تمهيد")), 100.0)
        o.offer(listOf(cue(300.0, CueStage.APPROACH, 1, "اقتراب")), 150.0)
        assertEquals("نُطق وهو مكتوم", 0, s.said.size)

        o.muted = false
        // **والقديمُ لا يُعاد** — وُسم «قيل» وهو مكتوم.
        o.offer(listOf(cue(300.0, CueStage.PREPARE, 0, "تمهيد")), 200.0)
        assertEquals("استدرك ما فات بعد رفع الكتم", 0, s.said.size)

        // **والطورُ الذي لم يحن يعمل طبيعيّاً.**
        o.offer(listOf(cue(300.0, CueStage.NOW, 3, "انعطف يميناً الآن")), 280.0)
        println("VORCH-030 · بعد رفع الكتم=${s.said}")
        assertEquals(1, s.said.size)
    }

    // ══════════════════════════════════════════════════════════════════
    // **٥ · غيابُ الصوت**
    // ══════════════════════════════════════════════════════════════════

    /** **VORCH-040** — بلا ناطقٍ صالحٍ لا شيءَ يسقط. */
    @Test
    fun `VORCH-040 بلا صوتٍ تعمل الملاحةُ صامتة`() {
        val s = FakeSpeaker(available = false)
        val o = VoiceOrchestrator(s)
        o.offer(listOf(cue(300.0, CueStage.NOW, 3, "انعطف يميناً الآن")), 280.0)
        println("VORCH-040 · قيلت=${s.said.size} · مطروحة=${o.dropped}")
        assertEquals(0, s.said.size)
        assertNull(o.current)
    }

    /** **VORCH-041** — وإيقافُ الملاحة يُسكت ويُفرغ. */
    @Test
    fun `VORCH-041 إيقافُ الملاحة يُسكت`() {
        val s = FakeSpeaker().also { it.instant = false }
        val o = VoiceOrchestrator(s)
        o.offer(listOf(cue(300.0, CueStage.NOW, 3, "انعطف يميناً الآن")), 280.0)
        o.stop()
        println("VORCH-041 · إسكات=${s.stops} · ينتظر=${o.pending}")
        assertEquals(1, s.stops)
        assertEquals(0, o.pending)
        assertNull(o.current)
    }

    /** **VORCH-042** — و«الملاحةُ متوقّفة» تمنع النطق. */
    @Test
    fun `VORCH-042 لا نطقَ والملاحةُ متوقّفة`() {
        val s = FakeSpeaker()
        val o = VoiceOrchestrator(s)
        o.offer(listOf(cue(300.0, CueStage.NOW, 3, "انعطف يميناً الآن")), 280.0, navigating = false)
        println("VORCH-042 · قيلت=${s.said.size}")
        assertEquals(0, s.said.size)
    }
}
