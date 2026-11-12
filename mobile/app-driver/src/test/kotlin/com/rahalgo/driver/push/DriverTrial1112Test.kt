package com.rahalgo.driver.push

import org.junit.Assert.assertEquals
import org.junit.Assert.assertFalse
import org.junit.Assert.assertTrue
import org.junit.Test
import java.io.File

/** **ملاحظاتُ تجربة المالك ٢٠٢٦-١١-١٢ على تطبيق السائق.** */
class DriverTrial1112Test {

    private fun src(rel: String): String =
        (File(rel).takeIf { it.exists() } ?: File("app-driver/$rel")).readText()

    @Test
    fun `offer ring lasts until the offer expires`() {
        val now = java.time.Instant.parse("2026-11-12T10:00:00Z").toEpochMilli()
        assertEquals(45_000L, OfferAlert.ringMs("2026-11-12T10:00:45Z", now))
        // **عرضٌ مات لا يرنّ.**
        assertEquals(0L, OfferAlert.ringMs("2026-11-12T09:59:59Z", now))
        // **حمولةٌ قديمةٌ بلا أجلٍ تأخذ دقيقة، ولا يزيد على خمس.**
        assertEquals(60_000L, OfferAlert.ringMs(null, now))
        assertEquals(300_000L, OfferAlert.ringMs("2026-11-12T11:00:00Z", now))
        assertTrue(OfferAlert.isOffer(mapOf("kind" to "order_offer")))
        assertFalse(OfferAlert.isOffer(mapOf("kind" to "order")))
    }

    @Test
    fun `offer alert is wired in the push service and silenced on resume`() {
        val push = src("src/main/kotlin/com/rahalgo/driver/push/PushService.kt")
        assertTrue("خدمةُ الإشعار لا تنادي تنبيهَ العرض", push.contains("OfferAlert.show("))
        val main = src("src/main/kotlin/com/rahalgo/driver/MainActivity.kt")
        assertTrue("الرنّةُ لا تسكت حين يُفتح التطبيق", main.contains("OfferAlert.cancelAll(context)"))
        val manifest = src("src/main/AndroidManifest.xml")
        assertTrue(manifest.contains("android.permission.USE_FULL_SCREEN_INTENT"))
    }

    @Test
    fun `no permission prompt before login`() {
        val main = src("src/main/kotlin/com/rahalgo/driver/MainActivity.kt")
        assertTrue(
            "تطبيقُ السائق يسأل الأذونَ فوق شاشة الدخول — وشاشةُ «تجهيز حسابك» تسألها بعده",
            main.contains("AppFrame(permissions = emptyList())"),
        )
    }

    @Test
    fun `trip chat sheet takes about two thirds of the screen`() {
        assertTrue(com.rahalgo.driver.trip.CHAT_SHEET_SCREEN_SHARE in 0.6f..0.65f)
        val sheet = src("src/main/kotlin/com/rahalgo/driver/trip/ChatSheet.kt")
        assertFalse("نافذةُ الدردشة رجعت ٣٤٠", sheet.contains("heightIn(max = 340.dp)"))
        assertTrue(sheet.contains(".heightIn(max = tall)") && sheet.contains(".fillMaxHeight()"))
    }

    /** **دردشةُ الرحلة بالمكوّن المشترك نفسِه** — لا حقلٌ ولا قائمةٌ خاصّان بالسائق. */
    @Test
    fun `trip chat uses the shared list and composer`() {
        val sheet = src("src/main/kotlin/com/rahalgo/driver/trip/ChatSheet.kt")
        assertTrue(sheet.contains("com.rahalgo.ui.ChatMessageList("))
        assertTrue(sheet.contains("com.rahalgo.ui.ChatComposer("))
        assertFalse("حقلُ كتابةٍ خاصٌّ بالسائق رجع", sheet.contains("OutlinedTextField("))
        assertFalse("قائمةٌ خاصّةٌ بالسائق رجعت", sheet.contains("LazyColumn("))
    }

    @Test
    fun `chat button says darsha not hadith`() {
        val strings = src("src/main/res/values/strings.xml")
        assertTrue(strings.contains("<string name=\"trip_chat\">دردشة الزبون</string>"))
        assertFalse(strings.contains("حديث الزبون"))
    }
}
