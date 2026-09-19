package com.rahalgo.ui

import com.rahalgo.shared.model.ApiErrorBody
import com.rahalgo.shared.net.ApiClient
import java.io.IOException
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CUST-DEF-002 · CAF-02 — «قيد المعالجة» لا يُنشئ طلباً ثانياً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # العطب
 *
 * **`isDecided` كانت تعدّ كلَّ ردِّ خادمٍ حسماً** — **فردُّ `409 in_progress`
 * (المحرّكُ ما زال يعالج الطلبَ الأوّل) كان يمحو مفتاحَ المحاولة.** ثمّ
 * يُثبَّت الطلبُ الأوّل، **فإعادةُ الضغط تحمل مفتاحاً جديداً فيُنشأ طلبٌ ثانٍ.**
 *
 * # العقدُ بعد الإصلاح
 *
 * **`409 in_progress` و`idempotency_reclaimed` ليسا حسماً** — يُحمَل المفتاحُ
 * نفسُه، وإعادةُ النداء تردّ الطلبَ المثبَّت (`writeReplay`). **محاولةٌ منطقيّةٌ
 * واحدةٌ ⇒ طلبٌ واحدٌ بالضبط.** وما عداهما من ردود الخادم حسمٌ كما كان.
 *
 * **ويُقاس بمحرّكٍ وهميٍّ يُطبّق تكرارَ `idempotency.go`**: مفتاحٌ محجوزٌ يردّ
 * `in_progress`، ومفتاحٌ مثبَّتٌ يُعيد طلبَه، **ومفتاحٌ جديدٌ يُنشئ آخر** — فلو
 * مُحي المفتاحُ على `in_progress` لَصار طلبان.
 */
class CustDef002Test {

    private class Mem : Attempt.Store {
        val rows = mutableMapOf<String, String>()
        override fun get(key: String): String? = rows[key]
        override fun put(key: String, value: String) { rows[key] = value }
        override fun remove(key: String) { rows.remove(key) }
    }

    private lateinit var disk: Mem

    @Before
    fun setUp() {
        disk = Mem()
        Attempt.resetForTest(disk)
    }

    private fun inProgress() =
        ApiClient.ApiException(409, ApiErrorBody(code = "in_progress"))

    // ── تصنيفُ الرد ───────────────────────────────────────────────────

    /** **«قيد المعالجة» بنوعيه ليس حسماً — يبقى المفتاح.** */
    @Test
    fun stillProcessingIsNotDecided() {
        assertFalse(
            "**409 in_progress عُدّ حسماً ⇒ يُمحى المفتاح ⇒ طلبٌ ثانٍ**",
            isDecided(inProgress()),
        )
        assertFalse(
            "**409 idempotency_reclaimed عُدّ حسماً**",
            isDecided(ApiClient.ApiException(409, ApiErrorBody(code = "idempotency_reclaimed"))),
        )
    }

    /** **وما عداهما حسمٌ — يُمحى المفتاحُ كما كان.** */
    @Test
    fun genuineServerAnswersRemainDecided() {
        assertTrue(isDecided(ApiClient.ApiException(400, ApiErrorBody(code = "validation"))))
        assertTrue(
            "**409 آخر (متجرٌ مغلق) حسمٌ حقيقيّ**",
            isDecided(ApiClient.ApiException(409, ApiErrorBody(code = "merchant_closed"))),
        )
        assertTrue(isDecided(ApiClient.ApiException(409, ApiErrorBody(code = "cash_blocked"))))
        assertFalse("**انقطاعُ الشبكة ليس حسماً**", isDecided(IOException("لا شبكة")))
    }

    // ── محرّكٌ وهميٌّ يُطبّق التكرار كما في idempotency.go ────────────────

    private class FakeEngine {
        var orders = 0
        private val committed = HashMap<String, Int>() // key -> order number
        val leaseHeld = HashSet<String>()              // key -> original request still processing

        /** يُطابق `POST /orders`: إعادةُ المثبَّت، أو in_progress، أو إنشاء. */
        fun create(key: String): Int {
            committed[key]?.let { return it }          // writeReplay
            if (key in leaseHeld) throw ApiClient.ApiException(409, ApiErrorBody(code = "in_progress"))
            orders += 1
            committed[key] = orders
            return orders
        }

        /** الطلبُ الأوّلُ (الذي تجاوز مهلةَ التطبيق) يُثبَّت في الخادم. */
        fun originalCommits(key: String) {
            leaseHeld.remove(key)
            if (committed[key] == null) {
                orders += 1
                committed[key] = orders
            }
        }
    }

    /** يحاكي كتلةَ الإرسال في `CartScreen`/`CustomScreen` حرفاً. */
    private fun submit(be: FakeEngine, slot: String): Int? {
        val key = Attempt.key(slot)
        return try {
            val n = be.create(key)
            Attempt.clear(slot) // نجاحٌ (أو إعادةُ المثبَّت)
            n
        } catch (e: Exception) {
            if (isDecided(e)) Attempt.clear(slot)
            null // قيد المعالجة / غيرُ محسوم
        }
    }

    /**
     * **الشاهدُ الأقوى — طلبٌ واحدٌ بالضبط** (`CAF-02`).
     *
     * محاولةٌ منطقيّةٌ واحدةٌ: نداءٌ يرى `in_progress`، فالأوّلُ يُثبَّت،
     * فإعادةٌ بالمفتاح نفسِه تردّ المثبَّت. **`orders == 1`.**
     */
    @Test
    fun oneSubmissionYieldsExactlyOneOrder() {
        val be = FakeEngine()
        val k = Attempt.key(Attempt.ORDER) // المفتاحُ المولَّدُ المحفوظ
        be.leaseHeld.add(k)                // الخادم: الطلبُ الأوّلُ يحجز المفتاحَ ويُعالَج

        val r1 = submit(be, Attempt.ORDER) // يرى in_progress
        assertEquals("**قُبل الطلبُ والمفتاحُ محجوز**", null, r1)
        assertTrue("**مُحي المفتاحُ على «قيد المعالجة»**", Attempt.pending(Attempt.ORDER))
        assertEquals("**نفسُ المفتاح**", k, Attempt.key(Attempt.ORDER))

        be.originalCommits(k)              // الطلبُ الأوّلُ (البطيء) يُثبَّت #1

        val r2 = submit(be, Attempt.ORDER) // إعادةٌ بالمفتاح نفسِه ⇒ إعادةُ المثبَّت
        assertEquals("**لم يُستردّ الطلبُ المثبَّت**", 1, r2)
        assertFalse("**بقي المفتاحُ بعد النجاح**", Attempt.pending(Attempt.ORDER))
        assertEquals("**أُنشئ طلبٌ ثانٍ لمحاولةٍ واحدة**", 1, be.orders)
    }

    /**
     * **وموتُ العمليّة بين «قيد المعالجة» والإعادة لا يُنشئ ثانياً** (`SR`+`CAF-02`).
     */
    @Test
    fun processDeathBetweenRetriesStillOneOrder() {
        val be = FakeEngine()
        val k = Attempt.key(Attempt.ORDER)
        be.leaseHeld.add(k)
        submit(be, Attempt.ORDER)          // in_progress، يبقى المفتاح

        // **تموت العمليّةُ** — يُعاد بناءُ المخزن فوق القرص عينِه.
        Attempt.resetForTest(null)
        Attempt.resetForTest(disk)

        be.originalCommits(k)              // #1 يُثبَّت
        val r = submit(be, Attempt.ORDER) // المفتاحُ من القرص = k ⇒ إعادةُ المثبَّت
        assertEquals(1, r)
        assertEquals("**نسيَ المفتاحَ بعد القتل ⇒ طلبان**", 1, be.orders)
    }

    /** **والرفضُ الصريحُ يُنهي المحاولةَ — مفتاحٌ جديدٌ لتصحيحٍ جديد.** */
    @Test
    fun terminalRejectionRetiresKey() {
        val be = FakeEngine()
        val first = Attempt.key(Attempt.ORDER)
        // خادمٌ يردّ حسماً (تحقّقٌ فاشل) — يُطبَّق يدويّاً كالكتلة الحقيقيّة
        if (isDecided(ApiClient.ApiException(400, ApiErrorBody(code = "validation")))) {
            Attempt.clear(Attempt.ORDER)
        }
        assertFalse("**بقي المفتاحُ بعد رفضٍ صريح**", Attempt.pending(Attempt.ORDER))
        val second = Attempt.key(Attempt.ORDER)
        assertTrue("**حمل التصحيحُ مفتاحَ الرفض**", first != second)
        assertEquals("**لم يُنشأ أيُّ طلب**", 0, be.orders)
    }

    /** **والنجاحُ من أوّل نداءٍ يبقى كما كان — طلبٌ واحدٌ ومفتاحٌ يُمحى.** */
    @Test
    fun firstAttemptSuccessUnchanged() {
        val be = FakeEngine()
        val r = submit(be, Attempt.ORDER)
        assertEquals(1, r)
        assertEquals(1, be.orders)
        assertFalse(Attempt.pending(Attempt.ORDER))
    }
}
