package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ تنقية الأسماء — على مُدوَّنةٍ حقيقيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * المعرّفات: `LBL-*` · **تُغلق `TD-BIDI-NAMES`.**
 *
 * **والأسماءُ مقيسةٌ من محرّكنا** على مساراتٍ حيّةٍ في خمس مدنٍ سوريّة
 * (٢٠٢٦-٠٨-٢١) — **لا مخترعةٌ لتنجح.**
 */
class MapLabelTest {

    /** **المُدوَّنةُ كما جاءت من المحرّك** — بمحارفها كلِّها. */
    private val corpus = listOf(
        "الشارع الموسط",
        "المتحلق الجنوبي",
        "روضة الميدان‬‎",
        "شارع أبو بكر الصدّيق",
        "شارع الزاهرة",
        "شارع المجتهد",
        "Al Walid Street",
        "شارع الأشرفية",
        "شارع التوحيد",
        "شارع الحمدانية",
        "شارع الشيخ طه",
        "شارع الصنوبري",
        "Ar Raqqa - Homs Road",
        "فارس الخوري‬‎",
        "‫شارع البعث‬‎",
        "‫قاسم أمين‬‎",
        "‫ِشارع ديربعلبة‬‎",
        "الجسر العتيق الدير",
        "شارح حسن الطه",
        "شارع الكورنيش",
        "شارع سينما فؤاد",
        "Adnan Malki Street",
        "الجسر الجديد",
        "دوار النعيم",
        "دوار الصوامع",
    )

    /** **الخمسةُ التي كانت تُرفض** — قِيست قبل الإصلاح. */
    private val wereRejected = listOf(
        "روضة الميدان‬‎",
        "فارس الخوري‬‎",
        "‫شارع البعث‬‎",
        "‫قاسم أمين‬‎",
        "‫ِشارع ديربعلبة‬‎",
    )

    private fun hasBidi(s: String) = s.any {
        it in '‎'..'‏' || it in '‪'..'‮' || it in '⁦'..'⁩' || it == '؜'
    }

    // ══════════════════════════════════════════════════════════════════
    // **١ · التنقية**
    // ══════════════════════════════════════════════════════════════════

    /** **LBL-001** — المُدوَّنةُ قبل وبعد. */
    @Test
    fun `LBL-001 التنقيةُ تستردّ الأسماء المرفوضة`() {
        val withBidi = corpus.count { hasBidi(it) }
        val speakBefore = corpus.count { StreetName.speakableRaw(it) != null }
        val speakAfter = corpus.count { MapLabel.forVoice(it) != null }
        val mapAfter = corpus.count { MapLabel.forMap(it) != null }
        println(
            "LBL-001 · المجموع=${corpus.size} · فيها محارفُ تحكّم=$withBidi\n" +
                "LBL-001 · للنطق قبلَ التنقية=$speakBefore · وبعدها=$speakAfter · وللخريطة=$mapAfter",
        )
        assertEquals("عددُ ذوات المحارف تبدّل", 5, withBidi)
        assertEquals("التنقيةُ لم تستردّ الخمسة", speakBefore + 5, speakAfter)
        // **والخريطةُ تقبل الجميع** — بما فيها اللاتينيّ.
        assertEquals("الخريطةُ أسقطت اسماً", corpus.size, mapAfter)
    }

    /** **LBL-002** — وكلُّ اسمٍ كان يُرفض صار يُنطق. */
    @Test
    fun `LBL-002 الخمسةُ صارت نظيفة`() {
        for (n in wereRejected) {
            val clean = MapLabel.normalize(n)
            assertTrue("بقيت محارفُ تحكّمٍ في: ${clean}", clean != null && !hasBidi(clean))
            assertTrue("لم يُقبل للنطق: $clean", MapLabel.forVoice(n) != null)
            println("LBL-002 · ${n.length} محرفاً → «$clean»")
        }
    }

    /** **LBL-003** — ولا يُحذف يونيكودٌ عشوائيّاً. */
    @Test
    fun `LBL-003 التشكيلُ والهمزاتُ تبقى`() {
        assertEquals("شارع أبو بكر الصدّيق", MapLabel.normalize("شارع أبو بكر الصدّيق"))
        assertEquals("الرقة 12", MapLabel.normalize("الرقة 12"))
        assertEquals("طريق 4", MapLabel.normalize("طريق 4"))
        assertEquals("Ar Raqqa - Homs Road", MapLabel.normalize("Ar Raqqa - Homs Road"))
        // **والشدّةُ حرفٌ لا زينة** — من جرّدها غيّر الاسم.
        assertTrue(MapLabel.normalize("الصدّيق")!!.contains('ّ'))
    }

    /** **LBL-004** — والفراغاتُ تُطوى بعد الحذف. */
    @Test
    fun `LBL-004 الفراغاتُ المتكرّرةُ تُطوى`() {
        assertEquals("شارع البعث", MapLabel.normalize("شارع‬ ‎ البعث"))
        assertEquals("شارع البعث", MapLabel.normalize("  شارع البعث  "))
        assertNull(MapLabel.normalize("‫‬‎"))
        assertNull(MapLabel.normalize(null))
        assertNull(MapLabel.normalize(""))
    }

    // ══════════════════════════════════════════════════════════════════
    // **٢ · سياستان لا واحدة**
    // ══════════════════════════════════════════════════════════════════

    /**
     * **LBL-010** — الخريطةُ تقبل اللاتينيَّ والنطقُ يرفضه.
     *
     * (قرارُ المالك، البند ١٧: `Map policy != Voice policy`.)
     */
    @Test
    fun `LBL-010 الخريطةُ تقبل اللاتينيَّ والنطقُ يرفضه`() {
        val latin = listOf("Al Walid Street", "Adnan Malki Street", "Ar Raqqa - Homs Road")
        for (n in latin) {
            assertEquals("الخريطةُ أسقطت لاتينيّاً", n, MapLabel.forMap(n))
            assertNull("النطقُ قبل لاتينيّاً", MapLabel.forVoice(n))
        }
        println("LBL-010 · لاتينيّة=${latin.size} · للخريطة=كلُّها · للنطق=صفر")
    }

    /** **LBL-011** — ولا منطقَ تنقيةٍ ثانٍ في الجملة. */
    @Test
    fun `LBL-011 جملةُ النطق تستفيد من التنقية نفسِها`() {
        val m = NavManeuver(
            ManeuverKinds.TURN_LEFT, "left", 100.0, 0, 50.0, 5.0,
            streetName = "‫شارع البعث‬‎",
        )
        val text = VoicePhrases.maneuver(m, 200)
        println("LBL-011 · «$text»")
        assertTrue("الاسمُ النظيفُ لم يدخل الجملة", text.contains("شارع البعث"))
        assertTrue("تسرّبت محارفُ تحكّمٍ إلى النطق", !hasBidi(text))
    }
}
