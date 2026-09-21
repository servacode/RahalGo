package com.rahalgo.ui

import com.rahalgo.shared.model.ApiErrorBody
import com.rahalgo.shared.net.ApiClient
import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertNotEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Before
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **CAF-02 · 13-029 — مفتاحٌ لجسمين، وسلامةُ المحاولة غيرِ المحسومة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ما يحرسه الخادم
 *
 * **بصمةُ الجسم**: مفتاحٌ عاد بجسمٍ مطابقٍ ⇒ إعادةُ المثبَّت؛ **بجسمٍ
 * مختلفٍ ⇒ `409 idempotency_key_reused` وصفرُ تنفيذٍ ثانٍ.**
 *
 * # وما يحرسه العميل (تصحيحُ المالك)
 *
 * **محاولةٌ لم تُحسَم تبقى ممثَّلةً** — **لا تُطرَح بتعديلٍ صامتٍ يُدوّر
 * المفتاح.** فمن عدّل السلّةَ ومحاولتُه السابقةُ مجهولةُ المصير:
 *
 *   1. **يُرَدّ `idempotency_key_reused`** — وهو ليس حسماً (`isDecided`
 *      تُبقيه غيرَ محسوم)، **فيبقى المفتاحُ ويبقى «لا ندري».**
 *   2. **ولا يُنشأ طلبٌ ثانٍ صامتٌ** فوق سابقٍ قد نجح.
 *   3. **والخروجُ بإقرارٍ صريحٍ** (`acknowledgeUncertain`) بعد التحقّق
 *      من «طلباتي» — **فيُمحى المفتاحُ القديمُ عن قصدٍ، ويُولَّد طازج.**
 *
 * **ويُقاس بمحرّكٍ وهميٍّ يُطبّق `acquireClaim` بعد الهجرة**: بصمةٌ لكلّ
 * مفتاح، مطابقةٌ تُعيد ومختلفةٌ تَرفض.
 */
class Caf02ReuseTest {

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

    /** **«مفتاحٌ لجسمين» ليس حسماً — يبقى المفتاح.** */
    @Test
    fun keyReusedIsNotDecided() {
        assertFalse(
            "**idempotency_key_reused عُدّ حسماً ⇒ يُمحى المفتاح ⇒ طلبٌ صامتٌ ثانٍ**",
            isDecided(ApiClient.ApiException(409, ApiErrorBody(code = "idempotency_key_reused"))),
        )
    }

    // ── محرّكٌ وهميٌّ يُطبّق acquireClaim بعد CAF-02 ─────────────────────

    private class FpEngine {
        var orders = 0
        private val committedNo = HashMap<String, Int>()    // key -> رقم الطلب المثبَّت
        private val committedFp = HashMap<String, String>() // key -> بصمةُ المثبَّت
        val leaseHeld = HashMap<String, String>()          // key -> بصمةُ الأولى قيد المعالجة

        fun create(key: String, fp: String): Int {
            committedNo[key]?.let { no ->
                // **صفٌّ مثبَّت**: بصمةٌ مطابقةٌ ⇒ إعادة، مختلفةٌ ⇒ رفض.
                if (committedFp[key] == fp) return no
                throw ApiClient.ApiException(409, ApiErrorBody(code = "idempotency_key_reused"))
            }
            leaseHeld[key]?.let { heldFp ->
                // **أولى قيد المعالجة**: بصمتُها محفوظةٌ منذ الإدراج —
                // **جسمٌ مختلفٌ يُرَدّ reused، وإلّا in_progress.**
                if (heldFp != fp) {
                    throw ApiClient.ApiException(409, ApiErrorBody(code = "idempotency_key_reused"))
                }
                throw ApiClient.ApiException(409, ApiErrorBody(code = "in_progress"))
            }
            orders += 1
            committedNo[key] = orders
            committedFp[key] = fp
            return orders
        }

        /** الطلبُ الأوّلُ (البطيء) يُثبَّت. */
        fun originalCommits(key: String) {
            val fp = leaseHeld.remove(key) ?: return
            if (committedNo[key] == null) {
                orders += 1
                committedNo[key] = orders
                committedFp[key] = fp
            }
        }
    }

    /** يحاكي كتلةَ `CartScreen.send` — مع حالِ «لا ندري». */
    private class CartSim(val be: FpEngine) {
        var uncertain = false

        fun send(slot: String, fp: String): Int? {
            val key = Attempt.key(slot)
            return try {
                val n = be.create(key, fp)
                Attempt.clear(slot); uncertain = false
                n
            } catch (e: Exception) {
                if (isDecided(e)) { Attempt.clear(slot); uncertain = false } else uncertain = true
                null
            }
        }

        /** يحاكي `CartViewModel.acknowledgeUncertain` — إقرارٌ صريح. */
        fun acknowledgeUncertain(slot: String) {
            Attempt.clear(slot); uncertain = false
        }
    }

    /**
     * **غيرُ محسومةٍ ثمّ الجسمُ نفسُه ⇒ تعافٍ، طلبٌ واحد.**
     *
     * انقطعت الأولى (قيد المعالجة)، **فبقي المفتاحُ و«لا ندري»**؛ ثمّ
     * ثبتت الأولى، **فإعادةٌ بالجسم نفسِه تردّ المثبَّت** — لا ثانيَ.
     */
    @Test
    fun uncertainThenSameBodyRecoversOneOrder() {
        val be = FpEngine()
        val k = Attempt.key(Attempt.ORDER)
        be.leaseHeld[k] = "bodyA"
        val sim = CartSim(be)

        assertNull("**قُبل والأولى قيد المعالجة**", sim.send(Attempt.ORDER, "bodyA"))
        assertTrue("**«لا ندري» يجب أن يبقى**", sim.uncertain)
        assertTrue("**المفتاحُ يجب أن يبقى**", Attempt.pending(Attempt.ORDER))
        assertEquals("**لا طلبَ بعد** — الأولى لم تثبت", 0, be.orders)

        be.originalCommits(k) // الأولى تثبت #1

        val r = sim.send(Attempt.ORDER, "bodyA") // نفسُ الجسم ⇒ إعادةُ المثبَّت
        assertEquals("**لم يُستردّ المثبَّت**", 1, r)
        assertFalse("**بقي «لا ندري» بعد التعافي**", sim.uncertain)
        assertFalse("**بقي المفتاحُ بعد النجاح**", Attempt.pending(Attempt.ORDER))
        assertEquals("**طلبٌ واحدٌ بالضبط**", 1, be.orders)
    }

    /**
     * **الشاهدُ الأقوى — غيرُ محسومةٍ ثمّ جسمٌ مُعدَّلٌ لا يُنشئ ثانياً صامتاً.**
     *
     * (تصحيحُ المالك: **«uncertain old attempt → cart edit → silently drop
     * old key → submit new order» ممنوع.**)
     */
    @Test
    fun uncertainThenEditedBodyMakesNoSilentSecondOrder() {
        val be = FpEngine()
        val k = Attempt.key(Attempt.ORDER)
        be.leaseHeld[k] = "bodyA"
        val sim = CartSim(be)

        sim.send(Attempt.ORDER, "bodyA") // in_progress ⇒ uncertain، المفتاحُ باقٍ
        assertTrue(sim.uncertain)

        // **يعدّل السلّةَ (bodyB) ويعيد بالمفتاح نفسِه** ⇒ reused.
        val r = sim.send(Attempt.ORDER, "bodyB")
        assertNull("**عُدّل الجسمُ — يجب ألّا يُقبَل**", r)
        assertTrue("**«لا ندري» يجب أن يبقى بعد reused**", sim.uncertain)
        assertTrue("**المفتاحُ يجب ألّا يُطرَح صامتاً**", Attempt.pending(Attempt.ORDER))
        assertEquals("**نفسُ المفتاح — لم يُدوَّر صامتاً**", k, Attempt.key(Attempt.ORDER))
        assertEquals("**لا طلبَ ثانٍ صامت**", 0, be.orders)
    }

    /**
     * **والإقرارُ الصريحُ وحدَه يفتح محاولةً جديدة** — بمفتاحٍ طازج.
     */
    @Test
    fun explicitAcknowledgementMintsFreshKeyAndPlacesOne() {
        val be = FpEngine()
        val k = Attempt.key(Attempt.ORDER)
        be.leaseHeld[k] = "bodyA"
        val sim = CartSim(be)

        sim.send(Attempt.ORDER, "bodyA")  // uncertain
        sim.send(Attempt.ORDER, "bodyB")  // reused، يبقى «لا ندري»
        assertTrue(sim.uncertain)

        // **تحقّق من «طلباتي» فلم يجد، فأقرّ ببدء محاولةٍ جديدة.**
        sim.acknowledgeUncertain(Attempt.ORDER)
        assertFalse("**بقي «لا ندري» بعد الإقرار**", sim.uncertain)
        assertFalse("**بقي المفتاحُ القديمُ بعد الإقرار**", Attempt.pending(Attempt.ORDER))

        val fresh = Attempt.key(Attempt.ORDER)
        assertNotEquals("**لم يُطازَج المفتاحُ بعد الإقرار**", k, fresh)

        val r = sim.send(Attempt.ORDER, "bodyB") // مفتاحٌ طازجٌ ⇒ طلبٌ جديد
        assertEquals(1, r)
        assertEquals("**طلبٌ واحدٌ بالضبط بعد الإقرار**", 1, be.orders)
    }

    /**
     * **ولا إقرارَ صامت**: تعديلُ السلّةِ وحدَه — بلا `acknowledgeUncertain` —
     * **لا يمحو المفتاحَ ولا يُطازجه.** (يمنع «الطرحَ الصامت» صراحةً.)
     */
    @Test
    fun editingCartAloneNeverRetiresUncertainKey() {
        val be = FpEngine()
        val k = Attempt.key(Attempt.ORDER)
        be.leaseHeld[k] = "bodyA"
        val sim = CartSim(be)
        sim.send(Attempt.ORDER, "bodyA")

        // **يُعدّل ويعيد مراراً** — لا يُطازَج المفتاحُ أبداً بلا إقرار.
        repeat(3) { sim.send(Attempt.ORDER, "bodyB") }

        assertTrue("**لا يُطرَح المفتاحُ بتعديلٍ صامت**", Attempt.pending(Attempt.ORDER))
        assertEquals("**نفسُ المفتاح**", k, Attempt.key(Attempt.ORDER))
        assertEquals("**ولا طلبَ صامت**", 0, be.orders)
    }
}
