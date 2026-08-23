package com.rahalgo.navigation

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════
 * **الوصولُ التجاريّ — الرفائدُ أ إلى ك**
 * ══════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ دلالات الوصول، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 */
class BusinessArrivalTest {

    private val target = GeoPoint(35.9500, 39.0100)

    /** **درجةُ عرضٍ ≈ ١١١٣٢٠م.** */
    private fun north(m: Double) = GeoPoint(target.lat + m / 111320.0, target.lng)

    private fun fix(p: GeoPoint, accuracy: Float = 5f) =
        NavFix(p.lat, p.lng, accuracy, 0f, 0f, 1_000L)

    private fun phase(
        at: GeoPoint,
        accuracy: Float = 5f,
        grade: FixGrade = FixGrade.ACCEPTED,
        routeArrived: Boolean = true,
        tgt: GeoPoint? = target,
    ) = ArrivalGuard.phase(routeArrived, tgt, fix(at, accuracy), grade)

    // ══════════════════════════════════════════════════════════════
    // **أ–د · تدرّجُ المسافة**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `أ - خمسةُ أمتارٍ فوصولٌ تجاريّ`() {
        assertEquals(ArrivalPhase.BUSINESS_TARGET_ARRIVED, phase(north(5.0)))
    }

    @Test
    fun `ب - خمسةَ عشرَ عند الحدّ`() {
        // **بدقّةٍ ممتازة: الحدُّ الفعّالُ ٢٠م** (١٥ + ٥ سماحاً).
        assertEquals(ArrivalPhase.BUSINESS_TARGET_ARRIVED, phase(north(15.0)))
        // **وبدقّةٍ صفرٍ نظريّاً: الحدُّ ١٥م بالضبط.**
        assertEquals(
            ArrivalPhase.BUSINESS_TARGET_ARRIVED,
            phase(north(15.0), accuracy = 0f),
        )
        assertEquals(
            ArrivalPhase.LAST_MILE_TO_TARGET,
            phase(north(16.0), accuracy = 0f),
        )
    }

    @Test
    fun `ج - ثلاثون خارجَ الحدّ بدقّةٍ جيّدة`() {
        // **١٥ + ٥ = ٢٠** — والثلاثون فوقَها.
        assertEquals(ArrivalPhase.LAST_MILE_TO_TARGET, phase(north(30.0)))
    }

    @Test
    fun `د - مئةٌ فالخطُّ انتهى ولا وصول`() {
        assertEquals(ArrivalPhase.LAST_MILE_TO_TARGET, phase(north(100.0)))
        // **ومئةٌ كانت حدّي الخاطئ** — أُخذت من توزيع الالتقاط.
        assertFalse(
            "مئةٌ ليست حدَّ وصولٍ تجاريّ",
            phase(north(100.0)) == ArrivalPhase.BUSINESS_TARGET_ARRIVED,
        )
    }

    @Test
    fun `والحدُّ من المنتج لا من توزيع الالتقاط`() {
        // **قرارُ المالك ٢٠٢٦-٠٨-١٢** — وهو ما تعمل به الشاشة.
        assertEquals(15.0, BusinessArrival.RADIUS_M, 0.0)
        // **والحدُّ الفعّالُ لا يتجاوز ثلاثين أبداً.**
        assertEquals(15.0, BusinessArrival.effectiveRadiusM(0f), 0.0)
        assertEquals(25.0, BusinessArrival.effectiveRadiusM(10f), 0.0)
        assertEquals(30.0, BusinessArrival.effectiveRadiusM(15f), 0.0)
        assertEquals(30.0, BusinessArrival.effectiveRadiusM(200f), 0.0)
    }

    // ══════════════════════════════════════════════════════════════
    // **هـ · انتهى الخطُّ بعيداً ثمّ اقترب**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `هـ - مئةٌ وثمانون فحالةٌ ثالثةٌ لا وصولَ ولا سير`() {
        assertEquals(ArrivalPhase.LAST_MILE_TO_TARGET, phase(north(180.0)))
        // ── ثمّ يقترب ─────────────────────────────────────────────
        assertEquals(ArrivalPhase.LAST_MILE_TO_TARGET, phase(north(60.0)))
        assertEquals(ArrivalPhase.BUSINESS_TARGET_ARRIVED, phase(north(10.0)))
    }

    // ══════════════════════════════════════════════════════════════
    // **و–ح · جودةُ القراءة — البند ٥**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `و - قراءةٌ مرفوضةٌ داخلَ الحدّ لا تُعلن`() {
        assertEquals(
            ArrivalPhase.LAST_MILE_TO_TARGET,
            phase(north(3.0), grade = FixGrade.REJECTED),
        )
    }

    @Test
    fun `ز - والمتدهورةُ وحدَها لا تؤكّد`() {
        /**
         * **أمرُ المالك (البند ٥) نصّاً**: «DEGRADED → لا تؤكد
         * Business Arrival وحدها».
         *
         * **ولا يُبنى إعلانٌ نهائيٌّ على موضعٍ تقريبيّ.**
         */
        assertEquals(
            ArrivalPhase.LAST_MILE_TO_TARGET,
            phase(north(3.0), grade = FixGrade.DEGRADED),
        )
        assertFalse(ArrivalGuard.confirms(target, fix(north(3.0)), FixGrade.DEGRADED))
    }

    @Test
    fun `ح - والمقبولةُ تؤكّد`() {
        assertTrue(ArrivalGuard.confirms(target, fix(north(3.0)), FixGrade.ACCEPTED))
    }

    @Test
    fun `ودقّةُ خمسين لا تقول وصلت من عشرات الأمتار`() {
        /**
         * **أمرُ المالك نصّاً**: «GPS accuracy 50m لا تجعلنا نقول
         * "وصلت" من عشرات الأمتار».
         *
         * **فتُرفض قبل أن تُقاس** — ولو كانت المسافةُ صفراً.
         */
        assertEquals(
            ArrivalPhase.LAST_MILE_TO_TARGET,
            phase(north(0.0), accuracy = 50f),
        )
        assertEquals(30.0, BusinessArrival.MAX_ACCURACY_M, 0.0)
    }

    // ══════════════════════════════════════════════════════════════
    // **والسيرُ لم ينتهِ بعد**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `قربٌ بلا نهايةِ خطٍّ فوصولٌ أيضاً`() {
        /**
         * **ودخولُ الحدّ يكفي**: من بلغ بابَ الزبون فقد وصل، **ولو
         * بقي في الهندسة مترٌ لم يُمشَ.**
         *
         * **والعكسُ هو الخطر** — لا هذا.
         */
        assertEquals(
            ArrivalPhase.BUSINESS_TARGET_ARRIVED,
            phase(north(5.0), routeArrived = false),
        )
    }

    @Test
    fun `وبعيدٌ في الطريق فسيرٌ`() {
        assertEquals(
            ArrivalPhase.EN_ROUTE,
            phase(north(500.0), routeArrived = false),
        )
    }

    @Test
    fun `ولا هدفَ فالسلوكُ القديم`() {
        assertEquals(ArrivalPhase.BUSINESS_TARGET_ARRIVED, phase(north(900.0), tgt = null))
        assertEquals(
            ArrivalPhase.EN_ROUTE,
            phase(north(900.0), tgt = null, routeArrived = false),
        )
    }

    // ══════════════════════════════════════════════════════════════
    // **البند ١٤ — رقمٌ واحدٌ للشاشة والصوت**
    // ══════════════════════════════════════════════════════════════

    @Test
    fun `قربُ الشاشة والوصولُ من مصدرٍ واحد`() {
        assertTrue(ArrivalGuard.near(target, north(14.0).lat, north(14.0).lng))
        assertFalse(ArrivalGuard.near(target, north(16.0).lat, north(16.0).lng))
        // **ولا رقمَ سحريٌّ في التطبيق** — يُقرأ من هنا.
        //
        // **ويُفحص من المصدر لا بالاستيراد** — `app-driver` يرى
        // `driver-navigation` ولا عكس.
        val src = java.io.File(
            "../app-driver/src/main/kotlin/com/rahalgo/driver/orders/OrdersViewModel.kt",
        )
        assertTrue("لم يُوجد المصدر: " + src.absolutePath, src.isFile)
        val text = src.readText()
        assertTrue(
            "ARRIVAL_M لا تقرأ من BusinessArrival",
            text.contains("ARRIVAL_M = com.rahalgo.navigation.BusinessArrival.RADIUS_M"),
        )
        assertFalse(
            "رقمٌ مكتوبٌ بدل المصدر الواحد",
            text.contains("ARRIVAL_M = 15f"),
        )
    }
}
