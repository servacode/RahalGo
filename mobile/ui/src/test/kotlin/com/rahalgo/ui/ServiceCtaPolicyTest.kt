package com.rahalgo.ui

import org.junit.Assert.assertEquals
import org.junit.Test

/**
 * ══════════════════════════════════════════════════════════════════════
 * **متى يُعرَض زرُّ التوسّع — وأيُّ زرّ** (`CR`، ٢٠٢٦-٠٩-١٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولو قرّرت كلُّ شاشةٍ ذلك بنفسها لَعرضته إحداها لسببٍ زمنيّ** —
 * **فيُدعى من المتجرُ مغلقٌ عنده إلى «طلب منطقته»، فيظنّ أنّنا لا
 * نصله.**
 *
 * **والخادمُ يردّ مثلَ هذه النيّة** (`reason_mismatch`) — **وزرٌّ يُعرَض
 * ليُردَّ زرٌّ كاذب.**
 */
class ServiceCtaPolicyTest {

    /** **ولكلّ نيّةٍ سببُها** — ولا تتبادلان. */
    @Test
    fun `لكلِّ سببٍ جغرافيٍّ نيّتُه`() {
        assertEquals(
            "**عنوانٌ خارجَ الشكل يطلب توسيعَ النطاق**",
            ServiceReason.KIND_COVERAGE,
            ServiceReason.ctaKind(ServiceReason.ADDRESS_OUTSIDE_COVERAGE),
        )
        for (r in listOf(
            ServiceReason.CITY_NOT_SUPPORTED,
            ServiceReason.PROVINCE_NOT_SUPPORTED,
            ServiceReason.AREA_NOT_SUPPORTED,
        )) {
            assertEquals(
                "**من لم نصله بعدُ يُخبَر لا يطلب توسيعَ نطاق**: $r",
                ServiceReason.KIND_INTEREST,
                ServiceReason.ctaKind(r),
            )
        }
    }

    /**
     * **والأسبابُ الزمنيّةُ لا زرَّ لها.**
     *
     * **ومنصّةٌ خارجَ دوامها ومنطقةٌ خارجَ وقتها ومتجرٌ مغلقٌ تعود بعد
     * ساعات** — **ولا علاقةَ لها بتوسّعٍ جغرافيّ.**
     *
     * **و`coverage_unavailable` عطبُ إعدادٍ في اللوحة** — **ولا يُدعى
     * الناسُ إلى طلب مناطقهم بسبب عطبٍ عندنا** (البند ٢٥).
     */
    @Test
    fun `الأسبابُ الزمنيّةُ وعطبُ الإعداد بلا زرّ`() {
        for (r in listOf(
            ServiceReason.AVAILABLE,
            ServiceReason.LAUNCH_CLOSED,
            ServiceReason.TEMPORARILY_UNAVAILABLE,
            ServiceReason.PLATFORM_CLOSED_NOW,
            ServiceReason.ZONE_CLOSED_NOW,
            ServiceReason.MERCHANT_CLOSED_NOW,
            ServiceReason.COVERAGE_UNAVAILABLE,
            ServiceReason.INVALID_LOCATION,
            "",
            "شيءٌ لا نعرفه",
        )) {
            assertEquals(
                "**زرُّ توسّعٍ عُرض لسببٍ لا يخصّه**: $r",
                "",
                ServiceReason.ctaKind(r),
            )
        }
    }
}
