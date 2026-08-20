package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertNotNull
import org.junit.Assert.assertNull
import org.junit.Assert.assertTrue
import org.junit.Test
import kotlin.math.abs

/**
 * ══════════════════════════════════════════════════════════════════════
 * **اختباراتُ سلسلة الملاحة — بلا جهاز**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-٢٠، البند ١١.)
 *
 * **وكلُّ ما هنا حسابٌ محض** — لا `android.location.Location` ولا
 * محاكٍ. **ومنطقٌ لا يُختبر إلّا على جهازٍ لا يُختبر.**
 *
 * **والنقاطُ من الرقّة بإحداثيّاتها الحقيقيّة** — لا أرقامٌ مخترعة:
 * `0.00001` درجةٍ ≈ متر واحد، **فتُقرأ المسافاتُ في الرأس.**
 */
class NavPipelineTest {

    private val baseLat = 35.9506
    private val baseLng = 39.0094

    /** **متر واحدٌ شمالاً** ≈ `0.000009` درجةِ عرض. */
    private fun north(meters: Double) = baseLat + meters * 0.000009

    private fun fix(
        lat: Double = baseLat,
        lng: Double = baseLng,
        acc: Float = 8f,
        speed: Float? = 10f,
        bearing: Float? = 90f,
        atMs: Long = 1_000L,
    ) = NavFix(lat, lng, acc, speed, bearing, atMs)

    // ══════════════════════════════════════════════════════════════════
    // **جودةُ القراءة**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `القراءةُ الدقيقةُ تُقبل`() {
        val (grade, reason) = GpsQuality.grade(fix(acc = 5f), null)
        assertEquals(FixGrade.ACCEPTED, grade)
        assertEquals(RejectReason.NONE, reason)
    }

    /** **دقّةُ أربعين متراً موضعٌ صالحٌ واتّجاهٌ كاذب.** */
    @Test
    fun `الدقّةُ المتوسّطةُ تُقبل متدهورةً لا مرفوضة`() {
        val (grade, _) = GpsQuality.grade(fix(acc = 40f), null)
        assertEquals(FixGrade.DEGRADED, grade)
    }

    @Test
    fun `الدقّةُ السيّئةُ تُرفض`() {
        val (grade, reason) = GpsQuality.grade(fix(acc = 90f), null)
        assertEquals(FixGrade.REJECTED, grade)
        assertEquals(RejectReason.ACCURACY, reason)
    }

    /**
     * **قفزةٌ لا يقطعها راكبُ درّاجة.**
     *
     * **خمسُ مئةِ مترٍ في ثانية** = ١٨٠٠ كم/س — خطأُ قمرٍ صناعيّ.
     */
    @Test
    fun `القفزةُ المستحيلةُ تُرفض`() {
        val first = fix(atMs = 1_000L)
        val jump = fix(lat = north(500.0), atMs = 2_000L)
        val (grade, reason) = GpsQuality.grade(jump, first)
        assertEquals(FixGrade.REJECTED, grade)
        assertEquals(RejectReason.TELEPORT, reason)
    }

    /** **وثلاثون متراً في ثانيةٍ سيرٌ سريعٌ لا قفزة** — لا تُرفض. */
    @Test
    fun `السرعةُ العاليةُ المشروعةُ لا تُرفض`() {
        val first = fix(atMs = 1_000L)
        val fast = fix(lat = north(30.0), atMs = 2_000L)
        val (grade, _) = GpsQuality.grade(fast, first)
        assertEquals(FixGrade.ACCEPTED, grade)
    }

    /** **وقراءةٌ أقدمُ ممّا عُرض تُرجع الدرّاجةَ إلى الوراء.** */
    @Test
    fun `القراءةُ القديمةُ تُرفض`() {
        val newer = fix(atMs = 5_000L)
        val older = fix(lat = north(3.0), atMs = 4_000L)
        val (grade, reason) = GpsQuality.grade(older, newer)
        assertEquals(FixGrade.REJECTED, grade)
        assertEquals(RejectReason.STALE, reason)
    }

    /**
     * **والشكُّ يُطرح قبل الحكم.**
     *
     * قراءةٌ دقّتُها ٥٠م بعدت ٦٠م في ثانية — **قد تكون واقفةً بحقّ**،
     * فلا تُرفض.
     */
    @Test
    fun `الشكُّ يُطرح فلا تُرفض قراءةٌ مشكوكةٌ قريبة`() {
        val first = fix(acc = 50f, atMs = 1_000L)
        val next = fix(lat = north(60.0), acc = 50f, atMs = 2_000L)
        val (grade, _) = GpsQuality.grade(next, first)
        assertTrue(grade != FixGrade.REJECTED)
    }

    // ══════════════════════════════════════════════════════════════════
    // **الاتّجاه**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `الزاويةُ تُردّ إلى المدى`() {
        assertEquals(350f, GpsQuality.normalize(-10f), 0.01f)
        assertEquals(10f, GpsQuality.normalize(370f), 0.01f)
    }

    /** **والفرقُ بين ٣٥٩ و١ درجتان لا ٣٥٨.** */
    @Test
    fun `فرقُ الزاويتين بأقصر الطريقين`() {
        assertEquals(2f, GpsQuality.angleDelta(359f, 1f), 0.01f)
        assertEquals(-2f, GpsQuality.angleDelta(1f, 359f), 0.01f)
    }

    @Test
    fun `أوّلُ اتّجاهٍ يُؤخذ كما هو`() {
        val t = BearingTracker()
        val out = t.update(fix(bearing = 90f), null, FixGrade.ACCEPTED)
        assertEquals(90f, out!!, 0.01f)
    }

    /**
     * **ولا يدور السهمُ والسائقُ واقف** — وهو ما نهى عنه المالك نصّاً.
     */
    @Test
    fun `الاتّجاهُ يثبت عند الوقوف مهما قال الجهاز`() {
        val t = BearingTracker()
        t.update(fix(bearing = 90f, speed = 10f, atMs = 1_000L), null, FixGrade.ACCEPTED)
        // **وقف** — والجهازُ يقول إنّه استدار مئةً وثمانين.
        val stopped = fix(bearing = 270f, speed = 0.2f, atMs = 2_000L)
        val out = t.update(stopped, null, FixGrade.ACCEPTED)
        assertEquals("الاتّجاهُ تبدّل والسائقُ واقف", 90f, out!!, 0.01f)
    }

    /** **والمتدهورةُ لا تُعطي اتّجاهاً أبدا.** */
    @Test
    fun `القراءةُ المتدهورةُ لا تحرّك الاتّجاه`() {
        val t = BearingTracker()
        t.update(fix(bearing = 90f, atMs = 1_000L), null, FixGrade.ACCEPTED)
        val out = t.update(fix(bearing = 200f, atMs = 2_000L), null, FixGrade.DEGRADED)
        assertEquals(90f, out!!, 0.01f)
    }

    /**
     * **والتنعيمُ يقترب ولا يقفز.**
     *
     * من ٩٠ إلى ١٨٠ في قراءةٍ واحدةٍ بفاصل ٣٠٠ ملّي: **يقطع نحوَ ٦٣٪**
     * (`1 - e^-1`) — أكثرَ من النصف وأقلَّ من الكلّ.
     */
    @Test
    fun `التنعيمُ يقترب تدريجيّاً لا يقفز`() {
        val t = BearingTracker()
        t.update(fix(bearing = 90f, atMs = 1_000L), null, FixGrade.ACCEPTED)
        val out = t.update(fix(bearing = 180f, atMs = 1_300L), null, FixGrade.ACCEPTED)!!
        assertTrue("لم يتحرّك: $out", out > 100f)
        assertTrue("قفز إلى الهدف: $out", out < 175f)
    }

    /** **ومعامِلُ التنعيم يتبع الزمن** — فاصلٌ أطولُ يقترب أكثر. */
    @Test
    fun `الفاصلُ الأطولُ يقترب أكثر`() {
        fun run(gapMs: Long): Float {
            val t = BearingTracker()
            t.update(fix(bearing = 0f, atMs = 1_000L), null, FixGrade.ACCEPTED)
            return t.update(fix(bearing = 90f, atMs = 1_000L + gapMs), null, FixGrade.ACCEPTED)!!
        }
        assertTrue(run(1_000L) > run(200L))
    }

    /** **وعبورُ الشمال لا يلفّ لفّةً كاملة.** */
    @Test
    fun `التنعيمُ يعبر الشمال بأقصر طريق`() {
        val t = BearingTracker()
        t.update(fix(bearing = 350f, atMs = 1_000L), null, FixGrade.ACCEPTED)
        val out = t.update(fix(bearing = 10f, atMs = 2_000L), null, FixGrade.ACCEPTED)!!
        // **يجب أن يقع بين ٣٥٠ و٣٧٠ (أي ١٠)** — لا أن يمرّ بـ١٨٠.
        val ok = out >= 350f || out <= 10f
        assertTrue("لفّ لفّةً كاملة: $out", ok)
    }

    // ══════════════════════════════════════════════════════════════════
    // **التنعيم**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `زمنُ الحركة يتبع الفاصل الفعليّ`() {
        assertEquals(1_000L, PathSmoother.animationMs(1_000L, 500L, FixGrade.ACCEPTED))
    }

    /** **وقراءةٌ تأخّرت خمسَ ثوانٍ لا تُمشي النقطةَ خمسَ ثوان.** */
    @Test
    fun `زمنُ الحركة مقيَّدٌ بسقف`() {
        assertEquals(
            PathSmoother.MAX_ANIM_MS,
            PathSmoother.animationMs(5_000L, 1_000L, FixGrade.ACCEPTED),
        )
    }

    /** **والشكُّ يُترجَم بطئاً لا رفضا.** */
    @Test
    fun `المتدهورةُ تُبطئ الحركة`() {
        val normal = PathSmoother.animationMs(500L, 1_000L, FixGrade.ACCEPTED)
        val degraded = PathSmoother.animationMs(500L, 1_000L, FixGrade.DEGRADED)
        assertTrue(degraded > normal)
    }

    @Test
    fun `الاستيفاءُ يبلغ الهدفَ عند الواحد`() {
        val (lat, lng) = PathSmoother.lerp(0.0, 0.0, 10.0, 20.0, 1.0)
        assertEquals(10.0, lat, 1e-9)
        assertEquals(20.0, lng, 1e-9)
    }

    @Test
    fun `استيفاءُ الزاويةِ بأقصر طريق`() {
        val mid = PathSmoother.lerpAngle(350f, 10f, 0.5)
        assertTrue("مرّ بالجنوب: $mid", mid >= 355f || mid <= 5f)
    }

    // ══════════════════════════════════════════════════════════════════
    // **الكاميرا**
    // ══════════════════════════════════════════════════════════════════

    @Test
    fun `الكاميرا تتبع الموضعَ وتميل وتدور`() {
        val cam = NavCamera.follow(baseLat, baseLng, 120f, 0f, 800L)
        assertEquals(baseLat, cam.lat, 1e-9)
        assertEquals(120f, cam.bearingDeg, 0.01f)
        assertEquals(NavCamera.NAV_TILT, cam.tiltDeg, 0.01f)
        assertEquals(NavCamera.NAV_ZOOM, cam.zoom, 1e-9)
        assertEquals(800L, cam.durationMs)
    }

    /** **ودورانٌ إلى الشمال عند البدء يقلب الشاشةَ بلا سبب.** */
    @Test
    fun `الكاميرا تُبقي دورانَها حين لا يُعرف اتّجاه`() {
        val cam = NavCamera.follow(baseLat, baseLng, null, 47f, 500L)
        assertEquals(47f, cam.bearingDeg, 0.01f)
    }

    /** **ومن أنهى رحلتَه يجد خريطتَه كما عرفها.** */
    @Test
    fun `الخروجُ من الملاحة يعيد الخريطةَ مسطّحةً شمالا`() {
        val cam = NavCamera.rest(baseLat, baseLng)
        assertEquals(0f, cam.bearingDeg, 0.01f)
        assertEquals(0f, cam.tiltDeg, 0.01f)
    }

    // ══════════════════════════════════════════════════════════════════
    // **السلسلةُ كاملةً**
    // ══════════════════════════════════════════════════════════════════

    /** **والمرفوضةُ لا تُحرّك شيئاً على الشاشة.** */
    @Test
    fun `القفزةُ لا تحرّك الأيقونة`() {
        val p = NavPipeline()
        p.onFix(fix(atMs = 1_000L))
        val step = p.onFix(fix(lat = north(800.0), atMs = 2_000L))
        assertEquals(FixGrade.REJECTED, step.grade)
        assertNull("تحرّكت على قفزة", step.targetLat)
        assertNull(p.cameraFor(step, 0f))
    }

    @Test
    fun `السلسلةُ تعدّ ما رأت وما رفضت`() {
        val p = NavPipeline()
        p.onFix(fix(atMs = 1_000L))
        p.onFix(fix(lat = north(900.0), atMs = 2_000L)) // قفزة
        p.onFix(fix(lat = north(10.0), acc = 45f, atMs = 3_000L)) // متدهورة
        assertEquals(3, p.seen)
        assertEquals(1, p.rejected)
        assertEquals(1, p.degraded)
    }

    /**
     * **رحلةٌ كاملةٌ تُعاد بلا جهاز.**
     *
     * مستقيمٌ ثمّ انعطافٌ يمينٌ ثمّ وقوفٌ ثمّ انطلاق — **وهي بذرةُ
     * `navigation-fixtures`.**
     */
    @Test
    fun `رحلةٌ مصطنعةٌ تنتهي باتّجاهٍ صحيحٍ ولا ترتجف عند الوقوف`() {
        val p = NavPipeline()
        var lat = baseLat
        var t = 1_000L
        // **مستقيمٌ شمالاً** — عشرُ قراءات.
        repeat(10) {
            lat = north(10.0 * (it + 1))
            p.onFix(NavFix(lat, baseLng, 6f, 11f, 0f, t))
            t += 1_000
        }
        val north = p.headingDeg
        assertNotNull(north)
        assertTrue("لم يستقرّ شمالاً: $north", GpsQuality.angleClose(north!!, 0f, 12f))

        // **انعطافٌ يمينٌ إلى الشرق** — ستُّ قراءات.
        repeat(6) {
            p.onFix(NavFix(lat, baseLng + 0.0001 * (it + 1), 6f, 11f, 90f, t))
            t += 1_000
        }
        val east = p.headingDeg!!
        assertTrue("لم يتّجه شرقاً: $east", GpsQuality.angleClose(east, 90f, 20f))

        // **وقوفٌ** — والجهازُ يهذي.
        repeat(5) {
            p.onFix(NavFix(lat, baseLng + 0.0006, 6f, 0.1f, 300f, t))
            t += 1_000
        }
        val stopped = p.headingDeg!!
        assertEquals("دار السهمُ والسائقُ واقف", east, stopped, 0.01f)
    }

    /** **ولا تُقاس زاويةٌ بالطرح المباشر** — حارسٌ على الأداة نفسها. */
    @Test
    fun `المسافةُ بين نقطتين معقولة`() {
        val d = GpsQuality.metersBetween(baseLat, baseLng, north(100.0), baseLng)
        assertTrue("المسافةُ $d", abs(d - 100.0) < 5.0)
    }
}
